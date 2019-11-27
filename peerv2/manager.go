package peerv2

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wire"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

var MasterNodeID = "QmVsCnV9kRZ182MX11CpcHMyFAReyXV49a599AbqmwtNrV"
var HighwayBeaconID = byte(255)

func NewManager(
	host *Host,
	dpa string,
	ikey *incognitokey.CommitteePublicKey,
	cd ConsensusData,
	dispatcher *Dispatcher,
	nodeMode string,
	relayShard []byte,
) *Manager {
	master := peer.IDB58Encode(host.Host.ID()) == MasterNodeID
	log.Println("IsMasterNode:", master)
	return &Manager{
		LocalHost:            host,
		DiscoverPeersAddress: dpa,
		IdentityKey:          ikey,
		cd:                   cd,
		disp:                 dispatcher,
		IsMasterNode:         master,
		registerRequests:     make(chan int, 100),
		relayShard:           relayShard,
		nodeMode:             nodeMode,
	}
}

func (manager *Manager) PublishMessage(msg wire.Message) error {
	var topic string
	publishable := []string{wire.CmdBlockShard, wire.CmdBFT, wire.CmdBlockBeacon, wire.CmdTx, wire.CmdCustomToken, wire.CmdPeerState, wire.CmdBlkShardToBeacon}

	// msgCrossShard := msg.(wire.MessageCrossShard)
	msgType := msg.MessageType()
	for _, p := range publishable {
		topic = ""
		if msgType == p {
			for _, availableTopic := range manager.subs[msgType] {
				// fmt.Println("[hy]", availableTopic)
				if (availableTopic.Act == MessageTopicPair_PUB) || (availableTopic.Act == MessageTopicPair_PUBSUB) {
					topic = availableTopic.Name
					// if p == wire.CmdTx {
					// 	fmt.Printf("[hy] broadcast tx to topic %v\n", topic)
					// }
					err := broadcastMessage(msg, topic, manager.ps)
					if err != nil {
						fmt.Printf("Broadcast to topic %v error %v\n", topic, err)
						return err
					}
				}

			}
			if topic == "" {
				return errors.New("Can not find topic of this message type " + msgType + "for publish")
			}

			// return broadcastMessage(msg, topic, manager.ps)
		}
	}

	log.Println("Cannot publish message", msgType)
	return nil
}

func (manager *Manager) PublishMessageToShard(msg wire.Message, shardID byte) error {
	publishable := []string{wire.CmdCrossShard, wire.CmdBFT}
	msgType := msg.MessageType()
	for _, p := range publishable {
		if msgType == p {
			// Get topic for mess
			//TODO hy add more logic
			if msgType == wire.CmdCrossShard {
				// TODO(@0xakk0r0kamui): implicit order of subscriptions?
				return broadcastMessage(msg, manager.subs[msgType][shardID].Name, manager.ps)
			} else {
				for _, availableTopic := range manager.subs[msgType] {
					fmt.Println(availableTopic)
					if (availableTopic.Act == MessageTopicPair_PUB) || (availableTopic.Act == MessageTopicPair_PUBSUB) {
						return broadcastMessage(msg, availableTopic.Name, manager.ps)
					}
				}
			}
		}
	}

	log.Println("Cannot publish message", msgType)
	return nil
}

func (manager *Manager) Start(ns NetSync) {
	// connect to proxy node
	addr, err := multiaddr.NewMultiaddr(manager.DiscoverPeersAddress)
	if err != nil {
		panic(err)
	}

	addrInfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		panic(err)
	}

	// Pubsub
	// TODO(@0xbunyip): handle error
	manager.ps, _ = pubsub.NewFloodSub(context.Background(), manager.LocalHost.Host)
	manager.subs = m2t{}
	manager.messages = make(chan *pubsub.Message, 1000)

	// Wait until connection to highway is established to make sure gRPC won't fail
	// NOTE: must Connect after creating FloodSub
	connected := make(chan error)
	go manager.keepHighwayConnection(connected)
	<-connected

	req, err := NewRequester(manager.LocalHost.GRPC, addrInfo.ID)
	if err != nil {
		panic(err)
	}
	manager.Requester = req

	manager.Provider = NewBlockProvider(manager.LocalHost.GRPC, ns)

	go manager.manageRoleSubscription()

	manager.process()
}

// BroadcastCommittee floods message to topic `chain_committee` for highways
// Only masternode actually does the broadcast, other's messages will be ignored by highway
func (manager *Manager) BroadcastCommittee(
	epoch uint64,
	newBeaconCommittee []incognitokey.CommitteePublicKey,
	newAllShardCommittee map[byte][]incognitokey.CommitteePublicKey,
	newAllShardPending map[byte][]incognitokey.CommitteePublicKey,
) {
	if !manager.IsMasterNode {
		return
	}

	log.Println("Broadcasting committee to highways!!!")
	cc := &incognitokey.ChainCommittee{
		Epoch:             epoch,
		BeaconCommittee:   newBeaconCommittee,
		AllShardCommittee: newAllShardCommittee,
		AllShardPending:   newAllShardPending,
	}
	data, err := cc.ToByte()
	if err != nil {
		log.Println(err)
		return
	}

	topic := "chain_committee"
	err = manager.ps.Publish(topic, data)
	if err != nil {
		log.Println(err)
	}
}

type ConsensusData interface {
	GetUserRole() (string, string, int)
}

type Topic struct {
	Name string
	Sub  *pubsub.Subscription
	Act  MessageTopicPair_Action
}

type Manager struct {
	LocalHost            *Host
	DiscoverPeersAddress string
	IdentityKey          *incognitokey.CommitteePublicKey
	IsMasterNode         bool

	ps               *pubsub.PubSub
	subs             m2t                  // mapping from message to topic's subscription
	messages         chan *pubsub.Message // queue messages from all topics
	registerRequests chan int

	nodeMode   string
	relayShard []byte

	cd        ConsensusData
	disp      *Dispatcher
	Requester *BlockRequester
	Provider  *BlockProvider
}

func (manager *Manager) PutMessage(msg *pubsub.Message) {
	manager.messages <- msg
}

func (manager *Manager) process() {
	for {
		select {
		case msg := <-manager.messages:
			// fmt.Println("[db] go manager.disp.processInMessageString(string(msg.Data))")
			// go manager.disp.processInMessageString(string(msg.Data))
			err := manager.disp.processInMessageString(string(msg.Data))
			if err != nil {
				log.Println(err)
			}
		}
	}
}

// keepHighwayConnection periodically checks liveliness of connection to highway
// and try to connect if it's not available.
// The method push data to the given channel to signal that the first attempt had finished.
// Constructor can use this info to initialize other objects.
func (manager *Manager) keepHighwayConnection(connectedOnce chan error) {
	addr, err := multiaddr.NewMultiaddr(manager.DiscoverPeersAddress)
	if err != nil {
		panic(err)
	}

	hwPeerInfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		panic(err)
	}
	hwPID := hwPeerInfo.ID

	first := true
	net := manager.LocalHost.Host.Network()
	disconnected := true
	for ; true; <-time.Tick(10 * time.Second) {
		// Reconnect if not connected
		var err error
		if net.Connectedness(hwPID) != network.Connected {
			disconnected = true
			log.Println("Not connected to highway, connecting")
			ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
			if err = manager.LocalHost.Host.Connect(ctx, *hwPeerInfo); err != nil {
				log.Println("Could not connect to highway:", err, hwPeerInfo)
			}
		}

		if disconnected && net.Connectedness(hwPID) == network.Connected {
			// Register again since this might be a new highway
			log.Println("Connected to highway, sending register request")
			manager.registerRequests <- 1
			disconnected = false
		}

		// Notify that first attempt had finished
		if first {
			connectedOnce <- err
			first = false
		}
	}
}

func encodeMessage(msg wire.Message) (string, error) {
	// NOTE: copy from peerConn.outMessageHandler
	// Create messageHex
	messageBytes, err := msg.JsonSerialize()
	if err != nil {
		fmt.Println("Can not serialize json format for messageHex:" + msg.MessageType())
		fmt.Println(err)
		return "", err
	}

	// Add 24 bytes headerBytes into messageHex
	headerBytes := make([]byte, wire.MessageHeaderSize)
	// add command type of message
	managerdType, messageErr := wire.GetCmdType(reflect.TypeOf(msg))
	if messageErr != nil {
		fmt.Println("Can not get managerd type for " + msg.MessageType())
		fmt.Println(messageErr)
		return "", err
	}
	copy(headerBytes[:], []byte(managerdType))
	// add forward type of message at 13st byte
	forwardType := byte('s')
	forwardValue := byte(0)
	copy(headerBytes[wire.MessageCmdTypeSize:], []byte{forwardType})
	copy(headerBytes[wire.MessageCmdTypeSize+1:], []byte{forwardValue})
	messageBytes = append(messageBytes, headerBytes...)
	log.Printf("Encoded message TYPE %s CONTENT %s", managerdType, string(messageBytes))

	// zip data before send
	messageBytes, err = common.GZipFromBytes(messageBytes)
	if err != nil {
		fmt.Println("Can not gzip for messageHex:" + msg.MessageType())
		fmt.Println(err)
		return "", err
	}
	messageHex := hex.EncodeToString(messageBytes)
	//log.Debugf("Content in hex encode: %s", string(messageHex))
	// add end character to messageHex (delim '\n')
	// messageHex += "\n"
	return messageHex, nil
}

func broadcastMessage(msg wire.Message, topic string, ps *pubsub.PubSub) error {
	// Encode message to string first
	messageHex, err := encodeMessage(msg)
	if err != nil {
		return err
	}

	// Broadcast
	fmt.Printf("Publishing to topic %s\n", topic)
	return ps.Publish(topic, []byte(messageHex))
}

// manageRoleSubscription: polling current role every minute and subscribe to relevant topics
func (manager *Manager) manageRoleSubscription() {
	role := newUserRole("dummyLayer", "dummyRole", -1000)
	topics := m2t{}
	forced := false // only subscribe when role changed or last forced subscribe failed
	var err error
	for {
		select {
		case <-time.Tick(10 * time.Second):
			role, topics, err = manager.subscribe(role, topics, forced)
			if err != nil {
				log.Printf("subscribe failed: %v %+v", forced, err)
			} else {
				forced = false
			}

		case <-manager.registerRequests:
			log.Println("Received request to register")
			forced = true // register no matter if role changed or not
		}
	}
}

func (manager *Manager) subscribe(role userRole, topics m2t, forced bool) (userRole, m2t, error) {
	newRole := newUserRole(manager.cd.GetUserRole())
	if newRole == role && !forced { // Not forced => no need to subscribe when role stays the same
		return newRole, topics, nil
	}
	log.Printf("Role changed: %v -> %v", role, newRole)

	if newRole.role == common.WaitingRole && !forced { // Not forced => no need to subscribe when role is Waiting
		return newRole, topics, nil
	}

	// Registering
	pubkey, _ := manager.IdentityKey.ToBase58()
	roleSID := newRole.shardID
	if roleSID == -2 { // normal node
		roleSID = -1
	}
	shardIDs := []byte{byte(roleSID)}
	if manager.nodeMode == common.NodeModeRelay {
		shardIDs = append(manager.relayShard, HighwayBeaconID)
	}
	newTopics, roleOfTopics, err := manager.registerToProxy(pubkey, newRole.layer, shardIDs)
	if err != nil {
		return role, topics, err
	}

	if newRole != roleOfTopics {
		return role, topics, errors.Errorf("lole not matching with highway, local = %+v, highway = %+v", newRole, roleOfTopics)
	}

	log.Printf("Received topics = %+v, oldTopics = %+v", newTopics, topics)

	// Subscribing
	if err := manager.subscribeNewTopics(newTopics, topics); err != nil {
		return role, topics, err
	}

	return newRole, newTopics, nil
}

type userRole struct {
	layer   string
	role    string
	shardID int
}

func newUserRole(layer, role string, shardID int) userRole {
	return userRole{
		layer:   layer,
		role:    role,
		shardID: shardID,
	}
}

// subscribeNewTopics subscribes to new topics and unsubcribes any topics that aren't needed anymore
func (manager *Manager) subscribeNewTopics(newTopics, subscribed m2t) error {
	found := func(tName string, tmap m2t) bool {
		for _, topicList := range tmap {
			for _, t := range topicList {
				if tName == t.Name {
					return true
				}
			}
		}
		return false
	}

	// Subscribe to new topics
	for m, topicList := range newTopics {
		fmt.Printf("Process message %v and topic %v\n", m, topicList)
		for _, t := range topicList {

			if found(t.Name, subscribed) {
				fmt.Printf("Countinue 1 %v %v\n", t.Name, subscribed)
				continue
			}

			// TODO(@0xakk0r0kamui): check here
			if t.Act == MessageTopicPair_PUB {
				manager.subs[m] = append(manager.subs[m], Topic{Name: t.Name, Sub: nil, Act: t.Act})
				fmt.Printf("Countinue 2 %v %v\n", t.Name, subscribed)
				continue
			}

			fmt.Println("[db] subscribing", m, t.Name)

			s, err := manager.ps.Subscribe(t.Name)
			if err != nil {
				return errors.WithStack(err)
			}
			manager.subs[m] = append(manager.subs[m], Topic{Name: t.Name, Sub: s, Act: t.Act})
			go processSubscriptionMessage(manager.messages, s)
		}
	}

	// Unsubscribe to old ones
	for m, topicList := range subscribed {
		for _, t := range topicList {
			if found(t.Name, newTopics) {
				continue
			}

			// TODO(@0xakk0r0kamui): check here
			if t.Act == MessageTopicPair_PUB {
				continue
			}

			fmt.Println("[db] unsubscribing", m, t.Name)
			for _, s := range manager.subs[m] {
				if s.Name == t.Name {
					s.Sub.Cancel() // TODO(@0xbunyip): lock
				}
			}
			delete(manager.subs, m)
		}
	}
	return nil
}

// processSubscriptionMessage listens to a topic and pushes all messages to a queue to be processed later
func processSubscriptionMessage(inbox chan *pubsub.Message, sub *pubsub.Subscription) {
	ctx := context.Background()
	for {
		// TODO(@0xbunyip): check if topic is unsubbed then return, otherwise just continue
		msg, err := sub.Next(ctx)
		if err != nil { // Subscription might have been cancelled
			log.Println(err)
			return
		}

		inbox <- msg
	}
}

type m2t map[string][]Topic // Message to topics

func (manager *Manager) registerToProxy(
	pubkey string,
	layer string,
	shardID []byte,
) (m2t, userRole, error) {
	messagesWanted := getMessagesForLayer(manager.nodeMode, layer, shardID)
	fmt.Printf("-%v-;;;-%v-;;;-%v-;;;\n", messagesWanted, manager.nodeMode, shardID)
	// os.Exit(9)
	pairs, role, err := manager.Requester.Register(
		context.Background(),
		pubkey,
		messagesWanted,
		shardID,
		manager.LocalHost.Host.ID(),
	)
	if err != nil {
		return nil, userRole{}, err
	}

	// Mapping from message to list of topics
	topics := m2t{}
	for _, p := range pairs {
		for i, t := range p.Topic {
			topics[p.Message] = append(topics[p.Message], Topic{
				Name: t,
				Act:  p.Act[i],
			})
		}
	}
	r := userRole{
		layer:   role.Layer,
		role:    role.Role,
		shardID: int(role.Shard),
	}
	return topics, r, nil
}

func getMessagesForLayer(mode, layer string, shardID []byte) []string {
	switch mode {
	case common.NodeModeAuto:
		if layer == common.ShardRole {
			return []string{
				wire.CmdBlockShard,
				wire.CmdBlockBeacon,
				wire.CmdBFT,
				wire.CmdPeerState,
				wire.CmdCrossShard,
				wire.CmdBlkShardToBeacon,
				wire.CmdTx,
				wire.CmdPrivacyCustomToken,
				wire.CmdCustomToken,
			}
		} else if layer == common.BeaconRole {
			return []string{
				wire.CmdBlockBeacon,
				wire.CmdBFT,
				wire.CmdPeerState,
				wire.CmdBlkShardToBeacon,
			}
		} else {
			return []string{
				wire.CmdBlockBeacon,
				wire.CmdPeerState,
			}
		}
	case common.NodeModeRelay:
		return []string{
			wire.CmdTx,
			wire.CmdBlockShard,
			wire.CmdBlockBeacon,
			wire.CmdPeerState,
			wire.CmdPrivacyCustomToken,
			wire.CmdCustomToken,
		}
	}
	return []string{}
}

//go run *.go --listen "127.0.0.1:9433" --externaladdress "127.0.0.1:9433" --datadir "/data/fullnode" --discoverpeersaddress "127.0.0.1:9330" --loglevel debug
