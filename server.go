package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	peer2 "github.com/libp2p/go-libp2p-core/peer"

	"github.com/incognitochain/incognito-chain/metrics"
	"github.com/incognitochain/incognito-chain/peerv2"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/incognitochain/incognito-chain/blockchain/btc"
	"github.com/incognitochain/incognito-chain/consensus"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/memcache"
	"github.com/incognitochain/incognito-chain/pubsub"

	"github.com/incognitochain/incognito-chain/addrmanager"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/connmanager"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/transaction"

	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/databasemp"
	"github.com/incognitochain/incognito-chain/mempool"
	"github.com/incognitochain/incognito-chain/netsync"
	"github.com/incognitochain/incognito-chain/peer"
	"github.com/incognitochain/incognito-chain/rpcserver"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/incognitochain/incognito-chain/wire"
	libp2p "github.com/libp2p/go-libp2p-peer"

	p2ppubsub "github.com/libp2p/go-libp2p-pubsub"

	pb "github.com/libp2p/go-libp2p-pubsub/pb"
)

type Server struct {
	started     int32
	startupTime int64

	protocolVersion   string
	isEnableMining    bool
	chainParams       *blockchain.Params
	connManager       *connmanager.ConnManager
	blockChain        *blockchain.BlockChain
	dataBase          database.DatabaseInterface
	memCache          *memcache.MemoryCache
	rpcServer         *rpcserver.RpcServer
	memPool           *mempool.TxPool
	tempMemPool       *mempool.TxPool
	beaconPool        *mempool.BeaconPool
	shardPool         map[byte]blockchain.ShardPool
	shardToBeaconPool *mempool.ShardToBeaconPool
	crossShardPool    map[byte]blockchain.CrossShardPool
	waitGroup         sync.WaitGroup
	netSync           *netsync.NetSync
	addrManager       *addrmanager.AddrManager
	// userKeySet        *incognitokey.KeySet
	miningKeys      string
	privateKey      string
	wallet          *wallet.Wallet
	consensusEngine *consensus.Engine
	blockgen        *blockchain.BlockGenerator
	pusubManager    *pubsub.PubSubManager
	// The fee estimator keeps track of how long transactions are left in
	// the mempool before they are mined into blocks.
	feeEstimator map[byte]*mempool.FeeEstimator
	highway      *peerv2.ConnManager

	cQuit     chan struct{}
	cNewPeers chan *peer.Peer
}

// setupRPCListeners returns a slice of listeners that are configured for use
// with the RPC server depending on the configuration settings for listen
// addresses and TLS.
func (serverObj *Server) setupRPCListeners() ([]net.Listener, error) {
	// Setup TLS if not disabled.
	listenFunc := net.Listen
	if !cfg.DisableTLS {
		Logger.log.Debug("Disable TLS for RPC is false")
		// Generate the TLS cert and key file if both don't already
		// exist.
		if !fileExists(cfg.RPCKey) && !fileExists(cfg.RPCCert) {
			err := rpcserver.GenCertPair(cfg.RPCCert, cfg.RPCKey)
			if err != nil {
				return nil, err
			}
		}
		keyPair, err := tls.LoadX509KeyPair(cfg.RPCCert, cfg.RPCKey)
		if err != nil {
			return nil, err
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{keyPair},
			MinVersion:   tls.VersionTLS12,
		}

		// Change the standard net.Listen function to the tls one.
		listenFunc = func(net string, laddr string) (net.Listener, error) {
			return tls.Listen(net, laddr, &tlsConfig)
		}
	} else {
		Logger.log.Debug("Disable TLS for RPC is true")
	}

	netAddrs, err := common.ParseListeners(cfg.RPCListeners, "tcp")
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			log.Printf("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}
	return listeners, nil
}
func (serverObj *Server) setupRPCWsListeners() ([]net.Listener, error) {
	// Setup TLS if not disabled.
	listenFunc := net.Listen
	if !cfg.DisableTLS {
		Logger.log.Debug("Disable TLS for RPC is false")
		// Generate the TLS cert and key file if both don't already
		// exist.
		if !fileExists(cfg.RPCKey) && !fileExists(cfg.RPCCert) {
			err := rpcserver.GenCertPair(cfg.RPCCert, cfg.RPCKey)
			if err != nil {
				return nil, err
			}
		}
		keyPair, err := tls.LoadX509KeyPair(cfg.RPCCert, cfg.RPCKey)
		if err != nil {
			return nil, err
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{keyPair},
			MinVersion:   tls.VersionTLS12,
		}

		// Change the standard net.Listen function to the tls one.
		listenFunc = func(net string, laddr string) (net.Listener, error) {
			return tls.Listen(net, laddr, &tlsConfig)
		}
	} else {
		Logger.log.Debug("Disable TLS for RPC is true")
	}

	netAddrs, err := common.ParseListeners(cfg.RPCWSListeners, "tcp")
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			log.Printf("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}
	return listeners, nil
}

/*
NewServer - create server object which control all process of node
*/
func (serverObj *Server) NewServer(listenAddrs string, db database.DatabaseInterface, dbmp databasemp.DatabaseInterface, chainParams *blockchain.Params, protocolVer string, interrupt <-chan struct{}) error {
	// Init data for Server
	serverObj.protocolVersion = protocolVer
	serverObj.chainParams = chainParams
	serverObj.cQuit = make(chan struct{})
	serverObj.cNewPeers = make(chan *peer.Peer)
	serverObj.dataBase = db
	serverObj.memCache = memcache.New()
	serverObj.consensusEngine = consensus.New()

	//Init channel
	cPendingTxs := make(chan metadata.Transaction, 500)
	cRemovedTxs := make(chan metadata.Transaction, 500)

	var err error
	// init an pubsub manager
	var pubsubManager = pubsub.NewPubSubManager()

	serverObj.miningKeys = cfg.MiningKeys
	serverObj.privateKey = cfg.PrivateKey
	if serverObj.miningKeys == "" && serverObj.privateKey == "" {
		if cfg.NodeMode == common.NodeModeAuto || cfg.NodeMode == common.NodeModeBeacon || cfg.NodeMode == common.NodeModeShard {
			panic("miningkeys can't be empty in this node mode")
		}
	}
	//pusub???
	serverObj.pusubManager = pubsubManager
	serverObj.beaconPool = mempool.GetBeaconPool()
	serverObj.shardToBeaconPool = mempool.GetShardToBeaconPool()
	serverObj.crossShardPool = make(map[byte]blockchain.CrossShardPool)
	serverObj.shardPool = make(map[byte]blockchain.ShardPool)
	serverObj.blockChain = &blockchain.BlockChain{}
	serverObj.isEnableMining = cfg.EnableMining
	// create mempool tx
	serverObj.memPool = &mempool.TxPool{}

	relayShards := []byte{}
	if cfg.RelayShards == "all" {
		for index := 0; index < common.MaxShardNumber; index++ {
			relayShards = append(relayShards, byte(index))
		}
	} else {
		var validPath = regexp.MustCompile(`(?s)[[:digit:]]+`)
		relayShardsStr := validPath.FindAllString(cfg.RelayShards, -1)
		for index := 0; index < len(relayShardsStr); index++ {
			s, err := strconv.Atoi(relayShardsStr[index])
			if err == nil {
				relayShards = append(relayShards, byte(s))
			}
		}
	}
	var randomClient btc.RandomClient
	if cfg.BtcClient == 0 {
		randomClient = &btc.BlockCypherClient{}
		Logger.log.Info("Init 3-rd Party Random Client")

	} else {
		if cfg.BtcClientIP == common.EmptyString || cfg.BtcClientUsername == common.EmptyString || cfg.BtcClientPassword == common.EmptyString {
			Logger.log.Error("Please input Bitcoin Client Ip, Username, password. Otherwise, set btcclient is 0 or leave it to default value")
			os.Exit(2)
		}
		randomClient = btc.NewBTCClient(cfg.BtcClientUsername, cfg.BtcClientPassword, cfg.BtcClientIP, cfg.BtcClientPort)
		Logger.log.Infof("Init Bitcoin Core Client with IP %+v, Port %+v, Username %+v, Password %+v", cfg.BtcClientIP, cfg.BtcClientPort, cfg.BtcClientUsername, cfg.BtcClientPassword)
	}
	// Init block template generator
	serverObj.blockgen, err = blockchain.NewBlockGenerator(serverObj.memPool, serverObj.blockChain, serverObj.shardToBeaconPool, serverObj.crossShardPool, cPendingTxs, cRemovedTxs)
	if err != nil {
		return err
	}
	// Init consensus engine
	err = serverObj.consensusEngine.Init(&consensus.EngineConfig{
		Blockchain:    serverObj.blockChain,
		Node:          serverObj,
		BlockGen:      serverObj.blockgen,
		PubSubManager: serverObj.pusubManager,
	})
	if err != nil {
		return err
	}
	// TODO hy
	// Connect to highway
	Logger.log.Debug("Listenner: ", cfg.Listener)
	Logger.log.Debug("Bootnode: ", cfg.DiscoverPeersAddress)
	Logger.log.Debug("PrivateKey: ", cfg.PrivateKey)

	ip, port := peerv2.ParseListenner(cfg.Listener, "127.0.0.1", 9433)
	host := peerv2.NewHost(version(), ip, port, cfg.Libp2pPrivateKey)

	miningKeys := serverObj.consensusEngine.GetMiningPublicKeys()
	pubkey := miningKeys[common.BlsConsensus]
	dispatcher := &peerv2.Dispatcher{
		MessageListeners: &peerv2.MessageListeners{
			OnBlockShard:       serverObj.OnBlockShard,
			OnBlockBeacon:      serverObj.OnBlockBeacon,
			OnCrossShard:       serverObj.OnCrossShard,
			OnShardToBeacon:    serverObj.OnShardToBeacon,
			OnTx:               serverObj.OnTx,
			OnTxToken:          serverObj.OnTxToken,
			OnTxPrivacyToken:   serverObj.OnTxPrivacyToken,
			OnVersion:          serverObj.OnVersion,
			OnGetBlockBeacon:   serverObj.OnGetBlockBeacon,
			OnGetBlockShard:    serverObj.OnGetBlockShard,
			OnGetCrossShard:    serverObj.OnGetCrossShard,
			OnGetShardToBeacon: serverObj.OnGetShardToBeacon,
			OnVerAck:           serverObj.OnVerAck,
			OnGetAddr:          serverObj.OnGetAddr,
			OnAddr:             serverObj.OnAddr,

			//mubft
			OnBFTMsg:    serverObj.OnBFTMsg,
			OnPeerState: serverObj.OnPeerState,
		},
	}
	serverObj.highway = peerv2.NewConnManager(
		host,
		cfg.DiscoverPeersAddress,
		&pubkey,
		serverObj.consensusEngine,
		dispatcher,
		cfg.NodeMode,
		relayShards,
	)

	err = serverObj.blockChain.Init(&blockchain.Config{
		ChainParams: serverObj.chainParams,
		DataBase:    serverObj.dataBase,
		MemCache:    serverObj.memCache,
		//MemCache:          nil,
		BlockGen:          serverObj.blockgen,
		Interrupt:         interrupt,
		RelayShards:       relayShards,
		BeaconPool:        serverObj.beaconPool,
		ShardPool:         serverObj.shardPool,
		ShardToBeaconPool: serverObj.shardToBeaconPool,
		CrossShardPool:    serverObj.crossShardPool,
		Server:            serverObj,
		// UserKeySet:        serverObj.userKeySet,
		NodeMode:        cfg.NodeMode,
		FeeEstimator:    make(map[byte]blockchain.FeeEstimator),
		PubSubManager:   pubsubManager,
		RandomClient:    randomClient,
		ConsensusEngine: serverObj.consensusEngine,
		Highway:         serverObj.highway,
	})
	if err != nil {
		return err
	}
	serverObj.blockChain.InitChannelBlockchain(cRemovedTxs)
	if err != nil {
		return err
	}
	//init beacon pol
	mempool.InitBeaconPool(serverObj.pusubManager)
	//init shard pool
	mempool.InitShardPool(serverObj.shardPool, serverObj.pusubManager)
	//init cross shard pool
	mempool.InitCrossShardPool(serverObj.crossShardPool, db)

	//init shard to beacon bool
	mempool.InitShardToBeaconPool()
	// or if it cannot be loaded, create a new one.
	if cfg.FastStartup {
		Logger.log.Debug("Load chain dependencies from DB")
		serverObj.feeEstimator = make(map[byte]*mempool.FeeEstimator)
		for shardID, bestState := range serverObj.blockChain.BestState.Shard {
			_ = bestState
			feeEstimatorData, err := serverObj.dataBase.GetFeeEstimator(shardID)
			if err == nil && len(feeEstimatorData) > 0 {
				feeEstimator, err := mempool.RestoreFeeEstimator(feeEstimatorData)
				if err != nil {
					Logger.log.Debugf("Failed to restore fee estimator %v", err)
					Logger.log.Debug("Init NewFeeEstimator")
					serverObj.feeEstimator[shardID] = mempool.NewFeeEstimator(
						mempool.DefaultEstimateFeeMaxRollback,
						mempool.DefaultEstimateFeeMinRegisteredBlocks,
						cfg.LimitFee)
				} else {
					serverObj.feeEstimator[shardID] = feeEstimator
				}
			} else {
				Logger.log.Debugf("Failed to get fee estimator from DB %v", err)
				Logger.log.Debug("Init NewFeeEstimator")
				serverObj.feeEstimator[shardID] = mempool.NewFeeEstimator(
					mempool.DefaultEstimateFeeMaxRollback,
					mempool.DefaultEstimateFeeMinRegisteredBlocks,
					cfg.LimitFee)
			}
		}
	} else {
		err := serverObj.dataBase.CleanCommitments()
		if err != nil {
			Logger.log.Error(err)
			return err
		}
		err = serverObj.dataBase.CleanSerialNumbers()
		if err != nil {
			Logger.log.Error(err)
			return err
		}
		err = serverObj.dataBase.CleanFeeEstimator()
		if err != nil {
			Logger.log.Error(err)
			return err
		}

		serverObj.feeEstimator = make(map[byte]*mempool.FeeEstimator)
	}
	for shardID, feeEstimator := range serverObj.feeEstimator {
		serverObj.blockChain.SetFeeEstimator(feeEstimator, shardID)
	}

	serverObj.memPool.Init(&mempool.Config{
		BlockChain:        serverObj.blockChain,
		DataBase:          serverObj.dataBase,
		ChainParams:       chainParams,
		FeeEstimator:      serverObj.feeEstimator,
		TxLifeTime:        cfg.TxPoolTTL,
		MaxTx:             cfg.TxPoolMaxTx,
		DataBaseMempool:   dbmp,
		IsLoadFromMempool: cfg.LoadMempool,
		PersistMempool:    cfg.PersistMempool,
		RelayShards:       relayShards,
		// UserKeyset:        serverObj.userKeySet,
		PubSubManager: serverObj.pusubManager,
	})
	serverObj.memPool.AnnouncePersisDatabaseMempool()
	//add tx pool
	serverObj.blockChain.AddTxPool(serverObj.memPool)
	serverObj.memPool.InitChannelMempool(cPendingTxs, cRemovedTxs)
	//==============Temp mem pool only used for validation
	serverObj.tempMemPool = &mempool.TxPool{}
	serverObj.tempMemPool.Init(&mempool.Config{
		BlockChain:    serverObj.blockChain,
		DataBase:      serverObj.dataBase,
		ChainParams:   chainParams,
		FeeEstimator:  serverObj.feeEstimator,
		MaxTx:         cfg.TxPoolMaxTx,
		PubSubManager: pubsubManager,
	})
	go serverObj.tempMemPool.Start(serverObj.cQuit)
	serverObj.blockChain.AddTempTxPool(serverObj.tempMemPool)
	//===============

	serverObj.addrManager = addrmanager.NewAddrManager(cfg.DataDir, common.HashH(common.Uint32ToBytes(activeNetParams.Params.Net))) // use network param Net as key for storage

	// Init Net Sync manager to process messages
	serverObj.netSync = &netsync.NetSync{}
	serverObj.netSync.Init(&netsync.NetSyncConfig{
		BlockChain:        serverObj.blockChain,
		ChainParam:        chainParams,
		TxMemPool:         serverObj.memPool,
		Server:            serverObj,
		Consensus:         serverObj.consensusEngine, // for onBFTMsg
		ShardToBeaconPool: serverObj.shardToBeaconPool,
		CrossShardPool:    serverObj.crossShardPool,
		PubSubManager:     serverObj.pusubManager,
		RelayShard:        relayShards,
		RoleInCommittees:  -1,
	})
	// Create a connection manager.
	var listenPeer *peer.Peer
	if !cfg.DisableListen {
		var err error

		// this is initializing our listening peer
		listenPeer, err = serverObj.InitListenerPeer(serverObj.addrManager, listenAddrs)
		if err != nil {
			Logger.log.Error(err)
			return err
		}
	}
	isRelayNodeForConsensus := cfg.Accelerator
	if isRelayNodeForConsensus {
		cfg.MaxPeersSameShard = 9999
		cfg.MaxPeersOtherShard = 9999
		cfg.MaxPeersOther = 9999
		cfg.MaxPeersNoShard = 0
		cfg.MaxPeersBeacon = 9999
	}
	connManager := connmanager.New(&connmanager.Config{
		OnInboundAccept:      serverObj.InboundPeerConnected,
		OnOutboundConnection: serverObj.OutboundPeerConnected,
		ListenerPeer:         listenPeer,
		DiscoverPeers:        cfg.DiscoverPeers,
		DiscoverPeersAddress: cfg.DiscoverPeersAddress,
		ExternalAddress:      cfg.ExternalAddress,
		// config for connection of shard
		MaxPeersSameShard:  cfg.MaxPeersSameShard,
		MaxPeersOtherShard: cfg.MaxPeersOtherShard,
		MaxPeersOther:      cfg.MaxPeersOther,
		MaxPeersNoShard:    cfg.MaxPeersNoShard,
		MaxPeersBeacon:     cfg.MaxPeersBeacon,
	})
	serverObj.connManager = connManager

	// Start up persistent peers.
	permanentPeers := cfg.ConnectPeers
	if len(permanentPeers) == 0 {
		permanentPeers = cfg.AddPeers
	}

	for _, addr := range permanentPeers {
		go serverObj.connManager.Connect(addr, "", "", nil)
	}

	if !cfg.DisableRPC {
		// Setup listeners for the configured RPC listen addresses and
		// TLS settings.
		fmt.Println("settingup RPCListeners")
		httpListeners, err := serverObj.setupRPCListeners()
		wsListeners, err := serverObj.setupRPCWsListeners()
		if err != nil {
			return err
		}
		if len(httpListeners) == 0 && len(wsListeners) == 0 {
			return errors.New("RPCS: No valid listen address")
		}

		rpcConfig := rpcserver.RpcServerConfig{
			HttpListenters:  httpListeners,
			WsListenters:    wsListeners,
			RPCQuirks:       cfg.RPCQuirks,
			RPCMaxClients:   cfg.RPCMaxClients,
			RPCMaxWSClients: cfg.RPCMaxWSClients,
			ChainParams:     chainParams,
			BlockChain:      serverObj.blockChain,
			Blockgen:        serverObj.blockgen,
			TxMemPool:       serverObj.memPool,
			Server:          serverObj,
			Wallet:          serverObj.wallet,
			ConnMgr:         serverObj.connManager,
			AddrMgr:         serverObj.addrManager,
			RPCUser:         cfg.RPCUser,
			RPCPass:         cfg.RPCPass,
			RPCLimitUser:    cfg.RPCLimitUser,
			RPCLimitPass:    cfg.RPCLimitPass,
			DisableAuth:     cfg.RPCDisableAuth,
			NodeMode:        cfg.NodeMode,
			FeeEstimator:    serverObj.feeEstimator,
			ProtocolVersion: serverObj.protocolVersion,
			Database:        &serverObj.dataBase,
			MiningKeys:      cfg.MiningKeys,
			NetSync:         serverObj.netSync,
			PubSubManager:   pubsubManager,
			ConsensusEngine: serverObj.consensusEngine,
			MemCache:        serverObj.memCache,
		}
		serverObj.rpcServer = &rpcserver.RpcServer{}
		serverObj.rpcServer.Init(&rpcConfig)

		// init rpc client instance and stick to Blockchain object
		// in order to communicate to external services (ex. eth light node)
		//serverObj.blockChain.SetRPCClientChain(rpccaller.NewRPCClient())

		// Signal process shutdown when the RPC server requests it.
		go func() {
			<-serverObj.rpcServer.RequestedProcessShutdown()
			shutdownRequestChannel <- struct{}{}
		}()
	}

	//Init Metric Tool
	//if cfg.MetricUrl != "" {
	//	grafana := metrics.NewGrafana(cfg.MetricUrl, cfg.ExternalAddress)
	//	metrics.InitMetricTool(&grafana)
	//}
	return nil
}

/*
// InboundPeerConnected is invoked by the connection manager when a new
// inbound connection is established.
*/
func (serverObj *Server) InboundPeerConnected(peerConn *peer.PeerConn) {
	Logger.log.Debug("inbound connected")
}

/*
// outboundPeerConnected is invoked by the connection manager when a new
// outbound connection is established.  It initializes a new outbound server
// peer instance, associates it with the relevant state such as the connection
// request instance and the connection itserverObj, and finally notifies the address
// manager of the attempt.
*/
func (serverObj *Server) OutboundPeerConnected(peerConn *peer.PeerConn) {
	Logger.log.Debug("Outbound PEER connected with PEER Id - " + peerConn.GetRemotePeerID().Pretty())
	err := serverObj.PushVersionMessage(peerConn)
	if err != nil {
		Logger.log.Error(err)
	}
}

/*
// WaitForShutdown blocks until the main listener and peer handlers are stopped.
*/
func (serverObj *Server) WaitForShutdown() {
	serverObj.waitGroup.Wait()
}

/*
// Stop gracefully shuts down the connection manager.
*/
func (serverObj *Server) Stop() error {
	// stop connManager
	errStopConnManager := serverObj.connManager.Stop()
	if errStopConnManager != nil {
		Logger.log.Error(errStopConnManager)
	}

	// Shutdown the RPC server if it's not disabled.
	if !cfg.DisableRPC && serverObj.rpcServer != nil {
		serverObj.rpcServer.Stop()
	}

	// Save fee estimator in the db
	for shardID, feeEstimator := range serverObj.feeEstimator {
		Logger.log.Infof("Fee estimator data when saving #%d", feeEstimator)
		feeEstimatorData := feeEstimator.Save()
		if len(feeEstimatorData) > 0 {
			err := serverObj.dataBase.StoreFeeEstimator(feeEstimatorData, shardID)
			if err != nil {
				Logger.log.Errorf("Can't save fee estimator data on chain #%d: %v", shardID, err)
			} else {
				Logger.log.Infof("Save fee estimator data on chain #%d", shardID)
			}
		}
	}

	err := serverObj.consensusEngine.Stop("all")
	if err != nil {
		Logger.log.Error(err)
	}
	// Signal the remaining goroutines to cQuit.
	close(serverObj.cQuit)
	return nil
}

/*
// peerHandler is used to handle peer operations such as adding and removing
// peers to and from the server, banning peers, and broadcasting messages to
// peers.  It must be run in a goroutine.
*/
func (serverObj *Server) peerHandler() {
	// Start the address manager and sync manager, both of which are needed
	// by peers.  This is done here since their lifecycle is closely tied
	// to this handler and rather than adding more channels to sychronize
	// things, it's easier and slightly faster to simply start and stop them
	// in this handler.
	serverObj.addrManager.Start()
	serverObj.netSync.Start()

	Logger.log.Debug("Start peer handler")

	if len(cfg.ConnectPeers) == 0 {
		for _, addr := range serverObj.addrManager.AddressCache() {
			pk, pkT := addr.GetPublicKey()
			go serverObj.connManager.Connect(addr.GetRawAddress(), pk, pkT, nil)
		}
	}

	go serverObj.connManager.Start(cfg.DiscoverPeersAddress)

out:
	for {
		select {
		case p := <-serverObj.cNewPeers:
			serverObj.handleAddPeerMsg(p)
		case <-serverObj.cQuit:
			{
				break out
			}
		}
	}
	serverObj.netSync.Stop()
	errStopAddrManager := serverObj.addrManager.Stop()
	if errStopAddrManager != nil {
		Logger.log.Error(errStopAddrManager)
	}
	errStopConnManager := serverObj.connManager.Stop()
	if errStopAddrManager != nil {
		Logger.log.Error(errStopConnManager)
	}
}

/*
// Start begins accepting connections from peers.
*/
func (serverObj Server) Start() {
	// Already started?
	if atomic.AddInt32(&serverObj.started, 1) != 1 {
		return
	}
	Logger.log.Debug("Starting server")
	// --- Checkforce update code ---
	if serverObj.chainParams.CheckForce {
		serverObj.CheckForceUpdateSourceCode()
	}
	if cfg.IsTestnet() {
		Logger.log.Critical("************************" +
			"* Testnet is active *" +
			"************************")
	}
	// Server startup time. Used for the uptime command for uptime calculation.
	serverObj.startupTime = time.Now().Unix()

	// Start the peer handler which in turn starts the address and block
	// managers.
	serverObj.waitGroup.Add(1)

	serverObj.netSync.Start()

	go serverObj.highway.Start(serverObj.netSync)

	if !cfg.DisableRPC && serverObj.rpcServer != nil {
		serverObj.waitGroup.Add(1)

		// Start the rebroadcastHandler, which ensures user tx received by
		// the RPC server are rebroadcast until being included in a block.
		//go serverObj.rebroadcastHandler()

		serverObj.rpcServer.Start()
	}
	err := serverObj.consensusEngine.Start()
	if err != nil {
		Logger.log.Error(err)
		go serverObj.Stop()
		return
	}
	if cfg.NodeMode != common.NodeModeRelay {
		serverObj.memPool.IsBlockGenStarted = true
		serverObj.blockChain.SetIsBlockGenStarted(true)
		for _, shardPool := range serverObj.shardPool {
			go shardPool.Start(serverObj.cQuit)
		}
		go serverObj.beaconPool.Start(serverObj.cQuit)
	}

	go serverObj.blockChain.Synker.Start()
	if serverObj.memPool != nil {
		err := serverObj.memPool.LoadOrResetDatabaseMempool()
		if err != nil {
			Logger.log.Error(err)
		}
		go serverObj.TransactionPoolBroadcastLoop()
		go serverObj.memPool.Start(serverObj.cQuit)
		go serverObj.memPool.MonitorPool()
	}
	go serverObj.pusubManager.Start()
	// go metrics.StartSystemMetrics()
}

func (serverObj *Server) GetActiveShardNumber() int {
	return serverObj.blockChain.BestState.Beacon.ActiveShards
}

// func (serverObj *Server) GetNodePubKey() string {
// 	return serverObj.userKeySet.GetPublicKeyInBase58CheckEncode()
// }

// func (serverObj *Server) GetUserKeySet() *incognitokey.KeySet {
// 	return serverObj.userKeySet
// }

func (serverObj *Server) TransactionPoolBroadcastLoop() {
	ticker := time.NewTicker(serverObj.memPool.ScanTime)
	defer ticker.Stop()
	for _ = range ticker.C {
		txDescs := serverObj.memPool.GetPool()
		for _, txDesc := range txDescs {
			<-time.Tick(50 * time.Millisecond)
			if !txDesc.IsFowardMessage {
				tx := txDesc.Desc.Tx
				switch tx.GetType() {
				case common.TxNormalType:
					{
						txMsg, err := wire.MakeEmptyMessage(wire.CmdTx)
						if err != nil {
							continue
						}
						normalTx := tx.(*transaction.Tx)
						txMsg.(*wire.MessageTx).Transaction = normalTx
						err = serverObj.PushMessageToAll(txMsg)
						if err == nil {
							serverObj.memPool.MarkForwardedTransaction(*tx.Hash())
						}
					}
				case common.TxCustomTokenType:
					{
						txMsg, err := wire.MakeEmptyMessage(wire.CmdCustomToken)
						if err != nil {
							continue
						}
						customTokenTx := tx.(*transaction.TxNormalToken)
						txMsg.(*wire.MessageTxToken).Transaction = customTokenTx
						err = serverObj.PushMessageToAll(txMsg)
						if err == nil {
							serverObj.memPool.MarkForwardedTransaction(*tx.Hash())
						}
					}
				case common.TxCustomTokenPrivacyType:
					{
						txMsg, err := wire.MakeEmptyMessage(wire.CmdPrivacyCustomToken)
						if err != nil {
							continue
						}
						customPrivacyTokenTx := tx.(*transaction.TxCustomTokenPrivacy)
						txMsg.(*wire.MessageTxPrivacyToken).Transaction = customPrivacyTokenTx
						err = serverObj.PushMessageToAll(txMsg)
						if err == nil {
							serverObj.memPool.MarkForwardedTransaction(*tx.Hash())
						}
					}
				}
			}
		}
	}
}

// CheckForceUpdateSourceCode - loop to check current version with update version is equal
// Force source code to be updated and remove data
func (serverObject Server) CheckForceUpdateSourceCode() {
	go func() {
		ctx := context.Background()
		myClient, err := storage.NewClient(ctx, option.WithoutAuthentication())
		if err != nil {
			Logger.log.Error(err)
		}
		for {
			reader, err := myClient.Bucket("incognito").Object(serverObject.chainParams.ChainVersion).NewReader(ctx)
			if err != nil {
				Logger.log.Error(err)
				time.Sleep(10 * time.Second)
				continue
			}
			defer reader.Close()

			type VersionChain struct {
				Version    string `json:"Version"`
				Note       string `json:"Note"`
				RemoveData bool   `json:"RemoveData"`
			}
			versionChain := VersionChain{}
			currentVersion := version()
			body, err := ioutil.ReadAll(reader)
			if err != nil {
				Logger.log.Error(err)
				time.Sleep(10 * time.Second)
				continue
			}
			err = json.Unmarshal(body, &versionChain)
			if err != nil {
				Logger.log.Error(err)
				time.Sleep(10 * time.Second)
				continue
			}
			force := currentVersion != versionChain.Version
			if force {
				Logger.log.Error("\n*********************************************************************************\n" +
					versionChain.Note +
					"\n*********************************************************************************\n")
				Logger.log.Error("\n*********************************************************************************\n You're running version: " +
					currentVersion +
					"\n*********************************************************************************\n")
				Logger.log.Error("\n*********************************************************************************\n" +
					versionChain.Note +
					"\n*********************************************************************************\n")

				Logger.log.Error("\n*********************************************************************************\n New version: " +
					versionChain.Version +
					"\n*********************************************************************************\n")

				Logger.log.Error("\n*********************************************************************************\n" +
					"We're exited because having a force update on this souce code." +
					"\nPlease Update source code at https://github.com/incognitochain/incognito-chain" +
					"\n*********************************************************************************\n")
				if versionChain.RemoveData {
					serverObject.Stop()
					errRemove := os.RemoveAll(cfg.DataDir)
					if errRemove != nil {
						Logger.log.Error("We NEEDD to REMOVE database directory but can not process by error", errRemove)
					}
					time.Sleep(60 * time.Second)
				}
				os.Exit(common.ExitCodeForceUpdate)
			}
			time.Sleep(10 * time.Second)
		}
	}()
}

/*
// initListeners initializes the configured net listeners and adds any bound
// addresses to the address manager. Returns the listeners and a NAT interface,
// which is non-nil if UPnP is in use.
*/
func (serverObj *Server) InitListenerPeer(amgr *addrmanager.AddrManager, listenAddrs string) (*peer.Peer, error) {
	netAddr, err := common.ParseListener(listenAddrs, "ip")
	if err != nil {
		return nil, err
	}

	// use keycache to save listener peer into file, this will make peer id of listener not change after turn off node
	kc := KeyCache{}
	kc.Load(filepath.Join(cfg.DataDir, "listenerpeer.json"))

	// load seed of libp2p from keycache file, if not exist -> save a new data into keycache file
	seed := int64(0)
	seedC, _ := strconv.ParseInt(os.Getenv("LISTENER_PEER_SEED"), 10, 64)
	if seedC == 0 {
		key := "LISTENER_PEER_SEED"
		seedT := kc.Get(key)
		if seedT == nil {
			seed = common.RandInt64()
			kc.Set(key, seed)
		} else {
			seed = int64(seedT.(float64))
		}
	} else {
		seed = seedC
	}

	peerObj := peer.Peer{}
	peerObj.SetSeed(seed)
	peerObj.SetListeningAddress(*netAddr)
	peerObj.SetPeerConns(nil)
	peerObj.SetPendingPeers(nil)
	peerObj.SetConfig(*serverObj.NewPeerConfig())
	err = peerObj.Init(peer.PrefixProtocolID + version()) // it should be /incognito/x.yy.zz-beta
	if err != nil {
		return nil, err
	}

	kc.Save()
	return &peerObj, nil
}

/*
// newPeerConfig returns the configuration for the listening RemotePeer.
*/
func (serverObj *Server) NewPeerConfig() *peer.Config {
	// KeySetUser := serverObj.userKeySet
	config := &peer.Config{
		MessageListeners: peer.MessageListeners{
			OnBlockShard:       serverObj.OnBlockShard,
			OnBlockBeacon:      serverObj.OnBlockBeacon,
			OnCrossShard:       serverObj.OnCrossShard,
			OnShardToBeacon:    serverObj.OnShardToBeacon,
			OnTx:               serverObj.OnTx,
			OnTxToken:          serverObj.OnTxToken,
			OnTxPrivacyToken:   serverObj.OnTxPrivacyToken,
			OnVersion:          serverObj.OnVersion,
			OnGetBlockBeacon:   serverObj.OnGetBlockBeacon,
			OnGetBlockShard:    serverObj.OnGetBlockShard,
			OnGetCrossShard:    serverObj.OnGetCrossShard,
			OnGetShardToBeacon: serverObj.OnGetShardToBeacon,
			OnVerAck:           serverObj.OnVerAck,
			OnGetAddr:          serverObj.OnGetAddr,
			OnAddr:             serverObj.OnAddr,

			//mubft
			OnBFTMsg: serverObj.OnBFTMsg,
			// OnInvalidBlock:  serverObj.OnInvalidBlock,
			OnPeerState: serverObj.OnPeerState,
			//
			PushRawBytesToShard:  serverObj.PushRawBytesToShard,
			PushRawBytesToBeacon: serverObj.PushRawBytesToBeacon,
			GetCurrentRoleShard:  serverObj.GetCurrentRoleShard,
		},
		MaxInPeers:      cfg.MaxInPeers,
		MaxPeers:        cfg.MaxPeers,
		MaxOutPeers:     cfg.MaxOutPeers,
		ConsensusEngine: serverObj.consensusEngine,
	}
	// if KeySetUser != nil && len(KeySetUser.PrivateKey) != 0 {
	// 	config.UserKeySet = KeySetUser
	// }
	return config
}

// OnBlock is invoked when a peer receives a block message.  It
// blocks until the coin block has been fully processed.
func (serverObj *Server) OnBlockShard(p *peer.PeerConn,
	msg *wire.MessageBlockShard) {
	Logger.log.Debug("Receive a new blockshard START")

	var txProcessed chan struct{}
	serverObj.netSync.QueueBlock(nil, msg, txProcessed)
	//<-txProcessed

	Logger.log.Debug("Receive a new blockshard END")
}

func (serverObj *Server) OnBlockBeacon(p *peer.PeerConn,
	msg *wire.MessageBlockBeacon) {
	Logger.log.Debug("Receive a new blockbeacon START")

	var txProcessed chan struct{}
	serverObj.netSync.QueueBlock(nil, msg, txProcessed)
	//<-txProcessed

	Logger.log.Debug("Receive a new blockbeacon END")
}

func (serverObj *Server) OnCrossShard(p *peer.PeerConn,
	msg *wire.MessageCrossShard) {
	Logger.log.Debug("Receive a new crossshard START")

	var txProcessed chan struct{}
	serverObj.netSync.QueueBlock(nil, msg, txProcessed)
	//<-txProcessed

	Logger.log.Debug("Receive a new crossshard END")
}

func (serverObj *Server) OnShardToBeacon(p *peer.PeerConn,
	msg *wire.MessageShardToBeacon) {
	Logger.log.Debug("Receive a new shardToBeacon START")

	var txProcessed chan struct{}
	serverObj.netSync.QueueBlock(nil, msg, txProcessed)
	//<-txProcessed

	Logger.log.Debug("Receive a new shardToBeacon END")
}

func (serverObj *Server) OnGetBlockBeacon(_ *peer.PeerConn, msg *wire.MessageGetBlockBeacon) {
	Logger.log.Debug("Receive a " + msg.MessageType() + " message START")
	var txProcessed chan struct{}
	serverObj.netSync.QueueGetBlockBeacon(nil, msg, txProcessed)
	//<-txProcessed

	Logger.log.Debug("Receive a " + msg.MessageType() + " message END")
}
func (serverObj *Server) OnGetBlockShard(_ *peer.PeerConn, msg *wire.MessageGetBlockShard) {
	Logger.log.Debug("Receive a " + msg.MessageType() + " message START")
	var txProcessed chan struct{}
	serverObj.netSync.QueueGetBlockShard(nil, msg, txProcessed)
	//<-txProcessed

	Logger.log.Debug("Receive a " + msg.MessageType() + " message END")
}

func (serverObj *Server) OnGetCrossShard(_ *peer.PeerConn, msg *wire.MessageGetCrossShard) {
	Logger.log.Debug("Receive a getcrossshard START")
	var txProcessed chan struct{}
	serverObj.netSync.QueueMessage(nil, msg, txProcessed)
	Logger.log.Debug("Receive a getcrossshard END")
}

func (serverObj *Server) OnGetShardToBeacon(_ *peer.PeerConn, msg *wire.MessageGetShardToBeacon) {
	Logger.log.Debug("Receive a getshardtobeacon START")
	var txProcessed chan struct{}
	serverObj.netSync.QueueMessage(nil, msg, txProcessed)
	Logger.log.Debug("Receive a getshardtobeacon END")
}

// OnTx is invoked when a peer receives a tx message.  It blocks
// until the transaction has been fully processed.  Unlock the block
// handler this does not serialize all transactions through a single thread
// transactions don't rely on the previous one in a linear fashion like blocks.
func (serverObj *Server) OnTx(peer *peer.PeerConn, msg *wire.MessageTx) {
	Logger.log.Debug("Receive a new transaction START")
	var txProcessed chan struct{}
	serverObj.netSync.QueueTx(nil, msg, txProcessed)
	//<-txProcessed

	Logger.log.Debug("Receive a new transaction END")
}

func (serverObj *Server) OnTxToken(peer *peer.PeerConn, msg *wire.MessageTxToken) {
	Logger.log.Debug("Receive a new transaction(normal token) START")
	var txProcessed chan struct{}
	serverObj.netSync.QueueTxToken(nil, msg, txProcessed)
	//<-txProcessed

	Logger.log.Debug("Receive a new transaction(normal token) END")
}

func (serverObj *Server) OnTxPrivacyToken(peer *peer.PeerConn, msg *wire.MessageTxPrivacyToken) {
	Logger.log.Debug("Receive a new transaction(privacy token) START")
	var txProcessed chan struct{}
	serverObj.netSync.QueueTxPrivacyToken(nil, msg, txProcessed)
	//<-txProcessed

	Logger.log.Debug("Receive a new transaction(privacy token) END")
}

/*
// OnVersion is invoked when a peer receives a version message
// and is used to negotiate the protocol version details as well as kick start
// the communications.
*/
func (serverObj *Server) OnVersion(peerConn *peer.PeerConn, msg *wire.MessageVersion) {
	Logger.log.Debug("Receive version message START")

	pbk := ""
	pbkType := ""
	if msg.PublicKey != "" {
		//TODO hy set publickey here
		//fmt.Printf("Message %v %v %v\n", msg.SignDataB58, msg.PublicKey, msg.PublicKeyType)
		err := serverObj.consensusEngine.VerifyData([]byte(peerConn.GetListenerPeer().GetPeerID().Pretty()), msg.SignDataB58, msg.PublicKey, msg.PublicKeyType)
		//fmt.Println(err)

		if err == nil {
			pbk = msg.PublicKey
			pbkType = msg.PublicKeyType
		} else {
			peerConn.ForceClose()
			return
		}
	}

	peerConn.GetRemotePeer().SetPublicKey(pbk, pbkType)

	remotePeer := &peer.Peer{}
	remotePeer.SetListeningAddress(msg.LocalAddress)
	remotePeer.SetPeerID(msg.LocalPeerId)
	remotePeer.SetRawAddress(msg.RawLocalAddress)
	remotePeer.SetPublicKey(pbk, pbkType)
	serverObj.cNewPeers <- remotePeer

	if msg.ProtocolVersion != serverObj.protocolVersion {
		Logger.log.Error(errors.New("Not correct version "))
		peerConn.ForceClose()
		return
	}

	// check for accept connection
	if accepted, e := serverObj.connManager.CheckForAcceptConn(peerConn); !accepted {
		// not accept connection -> force close
		Logger.log.Error(e)
		peerConn.ForceClose()
		return
	}

	msgV, err := wire.MakeEmptyMessage(wire.CmdVerack)
	if err != nil {
		return
	}

	msgV.(*wire.MessageVerAck).Valid = true
	msgV.(*wire.MessageVerAck).Timestamp = time.Now()

	peerConn.QueueMessageWithEncoding(msgV, nil, peer.MessageToPeer, nil)

	//	push version message again
	if !peerConn.VerAckReceived() {
		err := serverObj.PushVersionMessage(peerConn)
		if err != nil {
			Logger.log.Error(err)
		}
	}

	Logger.log.Debug("Receive version message END")
}

/*
OnVerAck is invoked when a peer receives a version acknowlege message
*/
func (serverObj *Server) OnVerAck(peerConn *peer.PeerConn, msg *wire.MessageVerAck) {
	Logger.log.Debug("Receive verack message START")

	if msg.Valid {
		peerConn.SetVerValid(true)

		if peerConn.GetIsOutbound() {
			serverObj.addrManager.Good(peerConn.GetRemotePeer())
		}

		// send message for get addr
		//msgSG, err := wire.MakeEmptyMessage(wire.CmdGetAddr)
		//if err != nil {
		//	return
		//}
		//var dc chan<- struct{}
		//peerConn.QueueMessageWithEncoding(msgSG, dc, peer.MessageToPeer, nil)

		//	broadcast addr to all peer
		//listen := serverObj.connManager.GetListeningPeer()
		//msgSA, err := wire.MakeEmptyMessage(wire.CmdAddr)
		//if err != nil {
		//	return
		//}
		//
		//rawPeers := []wire.RawPeer{}
		//peers := serverObj.addrManager.AddressCache()
		//for _, peer := range peers {
		//	getPeerId, _ := serverObj.connManager.GetPeerId(peer.GetRawAddress())
		//	if peerConn.GetRemotePeerID().Pretty() != getPeerId {
		//		pk, pkT := peer.GetPublicKey()
		//		rawPeers = append(rawPeers, wire.RawPeer{peer.GetRawAddress(), pkT, pk})
		//	}
		//}
		//msgSA.(*wire.MessageAddr).RawPeers = rawPeers
		//var doneChan chan<- struct{}
		//listen.GetPeerConnsMtx().Lock()
		//for _, peerConn := range listen.GetPeerConns() {
		//	Logger.log.Debug("QueueMessageWithEncoding", peerConn)
		//	peerConn.QueueMessageWithEncoding(msgSA, doneChan, peer.MessageToPeer, nil)
		//}
		//listen.GetPeerConnsMtx().Unlock()
	} else {
		peerConn.SetVerValid(true)
	}

	Logger.log.Debug("Receive verack message END")
}

func (serverObj *Server) OnGetAddr(peerConn *peer.PeerConn, msg *wire.MessageGetAddr) {
	Logger.log.Debug("Receive getaddr message START")

	// send message for addr
	msgS, err := wire.MakeEmptyMessage(wire.CmdAddr)
	if err != nil {
		return
	}

	peers := serverObj.addrManager.AddressCache()
	rawPeers := []wire.RawPeer{}
	for _, peer := range peers {
		getPeerId, _ := serverObj.connManager.GetPeerId(peer.GetRawAddress())
		if peerConn.GetRemotePeerID().Pretty() != getPeerId {
			pk, pkT := peer.GetPublicKey()
			rawPeers = append(rawPeers, wire.RawPeer{peer.GetRawAddress(), pkT, pk})
		}
	}

	msgS.(*wire.MessageAddr).RawPeers = rawPeers
	var dc chan<- struct{}
	peerConn.QueueMessageWithEncoding(msgS, dc, peer.MessageToPeer, nil)

	Logger.log.Debug("Receive getaddr message END")
}

func (serverObj *Server) OnAddr(peerConn *peer.PeerConn, msg *wire.MessageAddr) {
	Logger.log.Debugf("Receive addr message %v", msg.RawPeers)
}

func (serverObj *Server) OnBFTMsg(p *peer.PeerConn, msg wire.Message) {
	Logger.log.Debug("Receive a BFTMsg START")
	var txProcessed chan struct{}
	isRelayNodeForConsensus := cfg.Accelerator
	if isRelayNodeForConsensus {
		senderPublicKey, _ := p.GetRemotePeer().GetPublicKey()
		// panic(senderPublicKey)
		// fmt.Println("eiiiiiiiiiiiii")
		// os.Exit(0)
		//TODO hy check here
		bestState := blockchain.GetBeaconBestState()
		beaconCommitteeList, err := incognitokey.CommitteeKeyListToString(bestState.BeaconCommittee)
		if err != nil {
			panic(err)
		}
		isInBeaconCommittee := common.IndexOfStr(senderPublicKey, beaconCommitteeList) != -1
		if isInBeaconCommittee {
			serverObj.PushMessageToBeacon(msg, map[libp2p.ID]bool{p.GetRemotePeerID(): true})
		}
		shardCommitteeList := make(map[byte][]string)
		for shardID, committee := range bestState.GetShardCommittee() {
			shardCommitteeList[shardID], err = incognitokey.CommitteeKeyListToString(committee)
			if err != nil {
				panic(err)
			}
		}
		for shardID, committees := range shardCommitteeList {
			isInShardCommitee := common.IndexOfStr(senderPublicKey, committees) != -1
			if isInShardCommitee {
				serverObj.PushMessageToShard(msg, shardID, map[libp2p.ID]bool{p.GetRemotePeerID(): true})
				break
			}
		}
	}
	serverObj.netSync.QueueMessage(nil, msg, txProcessed)
	Logger.log.Debug("Receive a BFTMsg END")
}

func (serverObj *Server) OnPeerState(_ *peer.PeerConn, msg *wire.MessagePeerState) {
	Logger.log.Debug("Receive a peerstate START")
	var txProcessed chan struct{}
	serverObj.netSync.QueueMessage(nil, msg, txProcessed)
	Logger.log.Debug("Receive a peerstate END")
}

func (serverObj *Server) GetPeerIDsFromPublicKey(pubKey string) []libp2p.ID {
	result := []libp2p.ID{}
	// panic(pubKey)
	// os.Exit(0)
	listener := serverObj.connManager.GetConfig().ListenerPeer
	for _, peerConn := range listener.GetPeerConns() {
		// Logger.log.Debug("Test PeerConn", peerConn.RemotePeer.PaymentAddress)
		pk, _ := peerConn.GetRemotePeer().GetPublicKey()
		if pk == pubKey {
			exist := false
			for _, item := range result {
				if item.Pretty() == peerConn.GetRemotePeer().GetPeerID().Pretty() {
					exist = true
				}
			}

			if !exist {
				result = append(result, peerConn.GetRemotePeer().GetPeerID())
			}
		}
	}

	return result
}

func (serverObj *Server) GetNodeRole() string {
	if serverObj.miningKeys == "" && serverObj.privateKey == "" {
		return ""
	}
	if cfg.NodeMode == "relay" {
		return "RELAY"
	}
	role, shardID := serverObj.consensusEngine.GetUserLayer()

	switch shardID {
	case -2:
		return ""
	case -1:
		return "BEACON_" + role
	default:
		return "SHARD_" + role
	}
	// pubkey := serverObj.userKeySet.GetPublicKeyInBase58CheckEncode()
	// if common.IndexOfStr(pubkey, blockchain.GetBestStateBeacon().BeaconCommittee) > -1 {
	// 	return "BEACON_VALIDATOR"
	// }
	// if common.IndexOfStr(pubkey, blockchain.GetBestStateBeacon().BeaconPendingValidator) > -1 {
	// 	return "BEACON_WAITING"
	// }
	// shardCommittee := blockchain.GetBestStateBeacon().GetShardCommittee()
	// for _, s := range shardCommittee {
	// 	if common.IndexOfStr(pubkey, s) > -1 {
	// 		return "SHARD_VALIDATOR"
	// 	}
	// }
	// shardPendingCommittee := blockchain.GetBestStateBeacon().GetShardPendingValidator()
	// for _, s := range shardPendingCommittee {
	// 	if common.IndexOfStr(pubkey, s) > -1 {
	// 		return "SHARD_VALIDATOR"
	// 	}
	// }
}

/*
PushMessageToAll broadcast msg
*/
func (serverObj *Server) PushMessageToAll(msg wire.Message) error {
	Logger.log.Debug("Push msg to all peers")

	// Publish message to highway
	if err := serverObj.highway.PublishMessage(msg); err != nil {
		return err
	}

	return nil
}

/*
PushMessageToPeer push msg to peer
*/
func (serverObj *Server) PushMessageToPeer(msg wire.Message, peerId libp2p.ID) error {
	Logger.log.Debugf("Push msg to peer %s", peerId.Pretty())
	var dc chan<- struct{}
	peerConn := serverObj.connManager.GetConfig().ListenerPeer.GetPeerConnByPeerID(peerId.Pretty())
	if peerConn != nil {
		msg.SetSenderID(serverObj.connManager.GetConfig().ListenerPeer.GetPeerID())
		peerConn.QueueMessageWithEncoding(msg, dc, peer.MessageToPeer, nil)
		Logger.log.Debugf("Pushed peer %s", peerId.Pretty())
		return nil
	} else {
		Logger.log.Error("RemotePeer not exist!")
	}
	return errors.New("RemotePeer not found")
}

/*
PushMessageToPeer push msg to pbk
*/
func (serverObj *Server) PushMessageToPbk(msg wire.Message, pbk string) error {
	Logger.log.Debugf("Push msg to pbk %s", pbk)
	peerConns := serverObj.connManager.GetPeerConnOfPublicKey(pbk)
	if len(peerConns) > 0 {
		for _, peerConn := range peerConns {
			msg.SetSenderID(peerConn.GetListenerPeer().GetPeerID())
			peerConn.QueueMessageWithEncoding(msg, nil, peer.MessageToPeer, nil)
		}
		Logger.log.Debugf("Pushed pbk %s", pbk)
		return nil
	} else {
		Logger.log.Error("RemotePeer not exist!")
	}
	return errors.New("RemotePeer not found")
}

/*
PushMessageToPeer push msg to pbk
*/
func (serverObj *Server) PushMessageToShard(msg wire.Message, shard byte, exclusivePeerIDs map[libp2p.ID]bool) error {
	Logger.log.Debugf("Push msg to shard %d", shard)

	// Publish message to highway
	if err := serverObj.highway.PublishMessageToShard(msg, shard); err != nil {
		return err
	}

	return nil
}

func (serverObj *Server) PushRawBytesToShard(p *peer.PeerConn, msgBytes *[]byte, shard byte) error {
	Logger.log.Debugf("Push raw bytes to shard %d", shard)
	peerConns := serverObj.connManager.GetPeerConnOfShard(shard)
	if len(peerConns) > 0 {
		for _, peerConn := range peerConns {
			if p == nil || peerConn != p {
				peerConn.QueueMessageWithBytes(msgBytes, nil)
			}
		}
		Logger.log.Debugf("Pushed shard %d", shard)
	} else {
		Logger.log.Error("RemotePeer of shard not exist!")
		peerConns := serverObj.connManager.GetPeerConnOfAll()
		for _, peerConn := range peerConns {
			if p == nil || peerConn != p {
				peerConn.QueueMessageWithBytes(msgBytes, nil)
			}
		}
	}
	return nil
}

/*
PushMessageToPeer push msg to beacon node
*/
func (serverObj *Server) PushMessageToBeacon(msg wire.Message, exclusivePeerIDs map[libp2p.ID]bool) error {
	// Publish message to highway
	if err := serverObj.highway.PublishMessage(msg); err != nil {
		return err
	}

	return nil
}

func (serverObj *Server) PushRawBytesToBeacon(p *peer.PeerConn, msgBytes *[]byte) error {
	Logger.log.Debugf("Push raw bytes to beacon")
	peerConns := serverObj.connManager.GetPeerConnOfBeacon()
	if len(peerConns) > 0 {
		for _, peerConn := range peerConns {
			if p == nil || peerConn != p {
				peerConn.QueueMessageWithBytes(msgBytes, nil)
			}
		}
		Logger.log.Debugf("Pushed raw bytes beacon done")
	} else {
		Logger.log.Error("RemotePeer of beacon raw bytes not exist!")
		peerConns := serverObj.connManager.GetPeerConnOfAll()
		for _, peerConn := range peerConns {
			if p == nil || peerConn != p {
				peerConn.QueueMessageWithBytes(msgBytes, nil)
			}
		}
	}
	return nil
}

// handleAddPeerMsg deals with adding new peers.  It is invoked from the
// peerHandler goroutine.
func (serverObj *Server) handleAddPeerMsg(peer *peer.Peer) bool {
	if peer == nil {
		return false
	}
	Logger.log.Debug("Zero peer have just sent a message version")
	//Logger.log.Debug(peer)
	return true
}

func (serverObj *Server) PushVersionMessage(peerConn *peer.PeerConn) error {
	// push message version
	msg, err := wire.MakeEmptyMessage(wire.CmdVersion)
	msg.(*wire.MessageVersion).Timestamp = time.Now().UnixNano()
	msg.(*wire.MessageVersion).LocalAddress = peerConn.GetListenerPeer().GetListeningAddress()
	msg.(*wire.MessageVersion).RawLocalAddress = peerConn.GetListenerPeer().GetRawAddress()
	msg.(*wire.MessageVersion).LocalPeerId = peerConn.GetListenerPeer().GetPeerID()
	msg.(*wire.MessageVersion).RemoteAddress = peerConn.GetListenerPeer().GetListeningAddress()
	msg.(*wire.MessageVersion).RawRemoteAddress = peerConn.GetListenerPeer().GetRawAddress()
	msg.(*wire.MessageVersion).RemotePeerId = peerConn.GetListenerPeer().GetPeerID()
	msg.(*wire.MessageVersion).ProtocolVersion = serverObj.protocolVersion

	// ValidateTransaction Public Key from ProducerPrvKey
	// publicKeyInBase58CheckEncode, publicKeyType := peerConn.GetListenerPeer().GetConfig().ConsensusEngine.GetCurrentMiningPublicKey()
	signDataInBase58CheckEncode := common.EmptyString
	// if publicKeyInBase58CheckEncode != "" {
	// msg.(*wire.MessageVersion).PublicKey = publicKeyInBase58CheckEncode
	// msg.(*wire.MessageVersion).PublicKeyType = publicKeyType
	// Logger.log.Info("Start Process Discover Peers", publicKeyInBase58CheckEncode)
	// sign data
	msg.(*wire.MessageVersion).PublicKey, msg.(*wire.MessageVersion).PublicKeyType, signDataInBase58CheckEncode, err = peerConn.GetListenerPeer().GetConfig().ConsensusEngine.SignDataWithCurrentMiningKey([]byte(peerConn.GetRemotePeer().GetPeerID().Pretty()))
	if err == nil {
		msg.(*wire.MessageVersion).SignDataB58 = signDataInBase58CheckEncode
	}
	// }
	// if peerConn.GetListenerPeer().GetConfig().UserKeySet != nil {
	// 	msg.(*wire.MessageVersion).PublicKey = peerConn.GetListenerPeer().GetConfig().UserKeySet.GetPublicKeyInBase58CheckEncode()
	// 	signDataB58, err := peerConn.GetListenerPeer().GetConfig().UserKeySet.SignDataInBase58CheckEncode()
	// 	if err == nil {
	// 		msg.(*wire.MessageVersion).SignDataB58 = signDataB58
	// 	}
	// }
	if err != nil {
		return err
	}
	peerConn.QueueMessageWithEncoding(msg, nil, peer.MessageToPeer, nil)
	return nil
}

func (serverObj *Server) GetCurrentRoleShard() (string, *byte) {
	return serverObj.connManager.GetCurrentRoleShard()
}

func (serverObj *Server) UpdateConsensusState(role string, userPbk string, currentShard *byte, beaconCommittee []string, shardCommittee map[byte][]string) {
	changed := serverObj.connManager.UpdateConsensusState(role, userPbk, currentShard, beaconCommittee, shardCommittee)
	if changed {
		Logger.log.Debug("UpdateConsensusState is true")
	} else {
		Logger.log.Debug("UpdateConsensusState is false")
	}
}

func (serverObj *Server) putResponseMsgs(msgs [][]byte) {
	for _, msg := range msgs {
		// Create dummy msg wrapping grpc response
		psMsg := &p2ppubsub.Message{
			Message: &pb.Message{
				Data: msg,
			},
		}
		serverObj.highway.PutMessage(psMsg)
	}
}

func (serverObj *Server) PushMessageGetBlockBeaconByHeight(from uint64, to uint64) error {
	msgs, err := serverObj.highway.Requester.GetBlockBeaconByHeight(from, to)
	if err != nil {
		Logger.log.Error(err)
		return err
	}
	// TODO(@0xbunyip): instead of putting response to queue, use it immediately in synker
	serverObj.putResponseMsgs(msgs)
	return nil
}

func (serverObj *Server) PushMessageGetBlockBeaconBySpecificHeight(heights []uint64, getFromPool bool) error {
	os.Exit(9)
	msg, err := wire.MakeEmptyMessage(wire.CmdGetBlockBeacon)
	if err != nil {
		return err
	}
	// msg.(*wire.MessageGetBlockBeacon).BlkHeights = heights
	// msg.(*wire.MessageGetBlockBeacon).BySpecificHeight = true
	// msg.(*wire.MessageGetBlockBeacon).FromPool = getFromPool
	// if peerID != "" {
	// 	return serverObj.PushMessageToPeer(msg, peerID)
	// }
	return serverObj.PushMessageToAll(msg)
}

func (serverObj *Server) PushMessageGetBlockBeaconByHash(blkHashes []common.Hash, getFromPool bool, peerID libp2p.ID) error {
	msg, err := wire.MakeEmptyMessage(wire.CmdGetBlockBeacon)
	if err != nil {
		return err
	}
	msg.(*wire.MessageGetBlockBeacon).ByHash = true
	msg.(*wire.MessageGetBlockBeacon).FromPool = getFromPool
	msg.(*wire.MessageGetBlockBeacon).BlkHashes = blkHashes
	if peerID != "" {
		return serverObj.PushMessageToPeer(msg, peerID)
	}
	return serverObj.PushMessageToBeacon(msg, map[peer2.ID]bool{})
}

func (serverObj *Server) PushMessageGetBlockShardByHeight(shardID byte, from uint64, to uint64) error {
	msgs, err := serverObj.highway.Requester.GetBlockShardByHeight(int32(shardID), from, to)
	if err != nil {
		Logger.log.Error(err)
		return err
	}

	serverObj.putResponseMsgs(msgs)
	return nil
}

func (serverObj *Server) PushMessageGetBlockShardBySpecificHeight(shardID byte, heights []uint64, getFromPool bool) error {

	msgs, err := serverObj.highway.Requester.GetBlockShardToBeaconByHeight(int32(shardID), 0, 0)
	if err != nil {
		Logger.log.Error(err)
		return err
	}

	serverObj.putResponseMsgs(msgs)
	return nil
	// msg, err := wire.MakeEmptyMessage(wire.CmdGetBlockShard)
	// if err != nil {
	// 	return err
	// }
	// msg.(*wire.MessageGetBlockShard).BlkHeights = heights
	// msg.(*wire.MessageGetBlockShard).BySpecificHeight = true
	// msg.(*wire.MessageGetBlockShard).ShardID = shardID
	// msg.(*wire.MessageGetBlockShard).FromPool = getFromPool
	// if peerID == "" {
	// 	return serverObj.PushMessageToShard(msg, shardID, map[libp2p.ID]bool{})
	// }
	// return serverObj.PushMessageToAll(msg)

}

func (serverObj *Server) PushMessageGetBlockShardByHash(shardID byte, blksHash []common.Hash, getFromPool bool, peerID libp2p.ID) error {
	msg, err := wire.MakeEmptyMessage(wire.CmdGetBlockShard)
	if err != nil {
		return err
	}
	msg.(*wire.MessageGetBlockShard).ByHash = true
	msg.(*wire.MessageGetBlockShard).FromPool = getFromPool
	msg.(*wire.MessageGetBlockShard).BlkHashes = blksHash
	msg.(*wire.MessageGetBlockShard).ShardID = shardID
	Logger.log.Critical("SEND REQUEST FOR BLOCK HASH peerID=empty shard=%+v blksHash[0]", peerID, shardID, blksHash[0].String())
	if peerID == "" {
		exclusivePeerIDs := map[libp2p.ID]bool{}
		return serverObj.PushMessageToShard(msg, shardID, exclusivePeerIDs)
	}
	return serverObj.PushMessageToPeer(msg, peerID)

}

func (serverObj *Server) PushMessageGetBlockShardToBeaconByHeight(shardID byte, from uint64, to uint64) error {
	msgs, err := serverObj.highway.Requester.GetBlockShardToBeaconByHeight(int32(shardID), from, to)
	if err != nil {
		Logger.log.Error(err)
		return err
	}

	serverObj.putResponseMsgs(msgs)
	return nil
}

func (serverObj *Server) PushMessageGetBlockShardToBeaconByHash(shardID byte, blkHashes []common.Hash, getFromPool bool, peerID libp2p.ID) error {
	Logger.log.Debugf("Send a GetShardToBeacon")
	listener := serverObj.connManager.GetConfig().ListenerPeer
	msg, err := wire.MakeEmptyMessage(wire.CmdGetShardToBeacon)
	if err != nil {
		return err
	}
	msg.(*wire.MessageGetShardToBeacon).ByHash = true
	msg.(*wire.MessageGetShardToBeacon).FromPool = getFromPool
	msg.(*wire.MessageGetShardToBeacon).ShardID = shardID
	msg.(*wire.MessageGetShardToBeacon).BlkHashes = blkHashes
	msg.(*wire.MessageGetShardToBeacon).Timestamp = time.Now().Unix()
	msg.SetSenderID(listener.GetPeerID())
	Logger.log.Debugf("Send a GetCrossShard from %s", listener.GetRawAddress())
	if peerID == "" {
		return serverObj.PushMessageToShard(msg, shardID, map[libp2p.ID]bool{})
	}
	return serverObj.PushMessageToPeer(msg, peerID)
}

func (serverObj *Server) PushMessageGetBlockShardToBeaconBySpecificHeight(shardID byte, blkHeights []uint64, getFromPool bool, peerID libp2p.ID) error {
	Logger.log.Debugf("Send a GetShardToBeacon")
	listener := serverObj.connManager.GetConfig().ListenerPeer
	msg, err := wire.MakeEmptyMessage(wire.CmdGetShardToBeacon)
	if err != nil {
		return err
	}
	msg.(*wire.MessageGetShardToBeacon).BySpecificHeight = true
	msg.(*wire.MessageGetShardToBeacon).FromPool = getFromPool
	msg.(*wire.MessageGetShardToBeacon).ShardID = shardID
	msg.(*wire.MessageGetShardToBeacon).BlkHeights = blkHeights
	msg.(*wire.MessageGetShardToBeacon).Timestamp = time.Now().Unix()
	msg.SetSenderID(listener.GetPeerID())
	Logger.log.Debugf("Send a GetShardToBeacon from %s", listener.GetRawAddress())
	if peerID == "" {
		return serverObj.PushMessageToShard(msg, shardID, map[libp2p.ID]bool{})
	}
	return serverObj.PushMessageToPeer(msg, peerID)
}

func (serverObj *Server) PushMessageGetBlockCrossShardByHash(fromShard byte, toShard byte, blkHashes []common.Hash, getFromPool bool, peerID libp2p.ID) error {
	Logger.log.Debugf("Send a GetCrossShard")
	listener := serverObj.connManager.GetConfig().ListenerPeer
	msg, err := wire.MakeEmptyMessage(wire.CmdGetCrossShard)
	if err != nil {
		return err
	}
	msg.(*wire.MessageGetCrossShard).ByHash = true
	msg.(*wire.MessageGetCrossShard).FromPool = getFromPool
	msg.(*wire.MessageGetCrossShard).FromShardID = fromShard
	msg.(*wire.MessageGetCrossShard).ToShardID = toShard
	msg.(*wire.MessageGetCrossShard).BlkHashes = blkHashes
	msg.(*wire.MessageGetCrossShard).Timestamp = time.Now().Unix()
	msg.SetSenderID(listener.GetPeerID())
	Logger.log.Debugf("Send a GetCrossShard from %s", listener.GetRawAddress())
	if peerID == "" {
		return serverObj.PushMessageToShard(msg, fromShard, map[libp2p.ID]bool{})
	}
	return serverObj.PushMessageToPeer(msg, peerID)

}

func (serverObj *Server) PushMessageGetBlockCrossShardBySpecificHeight(fromShard byte, toShard byte, blkHeights []uint64, getFromPool bool, peerID libp2p.ID) error {
	msgs, err := serverObj.highway.Requester.GetBlockCrossShardByHeight(
		int32(fromShard),
		int32(toShard),
		blkHeights,
		getFromPool,
	)
	if err != nil {
		Logger.log.Error(err)
		return err
	}

	serverObj.putResponseMsgs(msgs)
	return nil
}

func (serverObj *Server) PublishNodeState() error {
	listener := serverObj.connManager.GetConfig().ListenerPeer
	userKey, _ := serverObj.consensusEngine.GetCurrentMiningPublicKey()
	var userRole string
	var shardID byte
	if userKey != "" {
		userRole, shardID = serverObj.blockChain.BestState.Beacon.GetPubkeyRole(userKey, serverObj.blockChain.BestState.Beacon.BestBlock.Header.Round)
	} else {
		return errors.New("Can not load current mining key")
	}
	msg, err := wire.MakeEmptyMessage(wire.CmdPeerState)
	if err != nil {
		return err
	}
	msg.(*wire.MessagePeerState).Beacon = blockchain.ChainState{
		serverObj.blockChain.BestState.Beacon.BestBlock.Header.Timestamp,
		serverObj.blockChain.BestState.Beacon.BeaconHeight,
		serverObj.blockChain.BestState.Beacon.BestBlockHash,
		serverObj.blockChain.BestState.Beacon.Hash(),
	}
	if userRole != common.BeaconRole {
		msg.(*wire.MessagePeerState).Shards[shardID] = blockchain.ChainState{
			serverObj.blockChain.BestState.Shard[shardID].BestBlock.Header.Timestamp,
			serverObj.blockChain.BestState.Shard[shardID].ShardHeight,
			serverObj.blockChain.BestState.Shard[shardID].BestBlockHash,
			serverObj.blockChain.BestState.Shard[shardID].Hash(),
		}
	} else {
		msg.(*wire.MessagePeerState).ShardToBeaconPool = serverObj.shardToBeaconPool.GetValidBlockHeight()
	}
	publicKeyInBase58CheckEncode, _ := serverObj.consensusEngine.GetCurrentMiningPublicKey()
	if publicKeyInBase58CheckEncode != "" {
		_, _, shardID := serverObj.consensusEngine.GetUserRole()
		if (cfg.NodeMode == common.NodeModeAuto || cfg.NodeMode == common.NodeModeShard) && shardID >= 0 {
			msg.(*wire.MessagePeerState).CrossShardPool[byte(shardID)] = serverObj.crossShardPool[byte(shardID)].GetValidBlockHeight()
		}
	}
	//
	currentMiningKey := serverObj.consensusEngine.GetMiningPublicKeys()[serverObj.blockChain.BestState.Beacon.ConsensusAlgorithm]
	msg.(*wire.MessagePeerState).SenderMiningPublicKey, err = currentMiningKey.ToBase58()
	if err != nil {
		return err
	}
	msg.SetSenderID(listener.GetPeerID())
	fmt.Printf("PeerID send to Proxy when publish node state %v \n", listener.GetPeerID())
	if err != nil {
		return err
	}
	Logger.log.Debugf("Publish peerstate from %s", listener.GetRawAddress())
	serverObj.PushMessageToAll(msg)
	return nil
}

func (serverObj *Server) BoardcastNodeState() error {
	listener := serverObj.connManager.GetConfig().ListenerPeer
	msg, err := wire.MakeEmptyMessage(wire.CmdPeerState)
	if err != nil {
		return err
	}
	msg.(*wire.MessagePeerState).Beacon = blockchain.ChainState{
		serverObj.blockChain.BestState.Beacon.BestBlock.Header.Timestamp,
		serverObj.blockChain.BestState.Beacon.BeaconHeight,
		serverObj.blockChain.BestState.Beacon.BestBlockHash,
		serverObj.blockChain.BestState.Beacon.Hash(),
	}
	for _, shardID := range serverObj.blockChain.Synker.GetCurrentSyncShards() {
		msg.(*wire.MessagePeerState).Shards[shardID] = blockchain.ChainState{
			serverObj.blockChain.BestState.Shard[shardID].BestBlock.Header.Timestamp,
			serverObj.blockChain.BestState.Shard[shardID].ShardHeight,
			serverObj.blockChain.BestState.Shard[shardID].BestBlockHash,
			serverObj.blockChain.BestState.Shard[shardID].Hash(),
		}
	}
	msg.(*wire.MessagePeerState).ShardToBeaconPool = serverObj.shardToBeaconPool.GetValidBlockHeight()

	publicKeyInBase58CheckEncode, _ := serverObj.consensusEngine.GetCurrentMiningPublicKey()
	// signDataInBase58CheckEncode := common.EmptyString
	if publicKeyInBase58CheckEncode != "" {
		_, shardID := serverObj.consensusEngine.GetUserLayer()
		if (cfg.NodeMode == common.NodeModeAuto || cfg.NodeMode == common.NodeModeShard) && shardID >= 0 {
			msg.(*wire.MessagePeerState).CrossShardPool[byte(shardID)] = serverObj.crossShardPool[byte(shardID)].GetValidBlockHeight()
		}
	}
	userKey, _ := serverObj.consensusEngine.GetCurrentMiningPublicKey()
	if userKey != "" {
		metrics.SetGlobalParam("MINING_PUBKEY", userKey)
		userRole, shardID := serverObj.blockChain.BestState.Beacon.GetPubkeyRole(userKey, serverObj.blockChain.BestState.Beacon.BestBlock.Header.Round)
		if (cfg.NodeMode == common.NodeModeAuto || cfg.NodeMode == common.NodeModeShard) && userRole == common.NodeModeShard {
			userRole = serverObj.blockChain.BestState.Shard[shardID].GetPubkeyRole(userKey, serverObj.blockChain.BestState.Shard[shardID].BestBlock.Header.Round)
			if userRole == "shard-proposer" || userRole == "shard-validator" {
				msg.(*wire.MessagePeerState).CrossShardPool[shardID] = serverObj.crossShardPool[shardID].GetValidBlockHeight()
			}
		}
	}
	msg.SetSenderID(listener.GetPeerID())
	Logger.log.Debugf("Broadcast peerstate from %s", listener.GetRawAddress())
	serverObj.PushMessageToAll(msg)
	return nil
}

func (serverObj *Server) EnableMining(enable bool) error {
	serverObj.isEnableMining = enable
	return nil
}

func (serverObj *Server) IsEnableMining() bool {
	return serverObj.isEnableMining
}

func (serverObj *Server) GetChainMiningStatus(chain int) string {
	const (
		notmining = "notmining"
		syncing   = "syncing"
		ready     = "ready"
		mining    = "mining"
		pending   = "pending"
		waiting   = "waiting"
	)
	if chain >= common.MaxShardNumber || chain < -1 {
		return notmining
	}
	if cfg.MiningKeys != "" || cfg.PrivateKey != "" {
		//Beacon: chain = -1
		layer, role, shardID := serverObj.consensusEngine.GetUserRole()

		if shardID == -2 {
			return notmining
		}
		if chain != -1 && layer == common.BeaconRole {
			return notmining
		}

		switch layer {
		case common.BeaconRole:
			switch role {
			case common.CommitteeRole:
				if serverObj.blockChain.Synker.IsLatest(false, 0) {
					return mining
				}
				return syncing
			case common.PendingRole:
				return pending
			case common.WaitingRole:
				return waiting
			}
		case common.ShardRole:
			if chain != shardID {
				return notmining
			}
			switch role {
			case common.CommitteeRole:
				if serverObj.blockChain.Synker.IsLatest(true, byte(chain)) {
					return mining
				}
				return syncing
			case common.PendingRole:
				return pending
			case common.WaitingRole:
				return waiting
			}
		default:
			return notmining
		}

	}
	return notmining
}

func (serverObj *Server) GetMiningKeys() string {
	return serverObj.miningKeys
}

func (serverObj *Server) GetPrivateKey() string {
	return serverObj.privateKey
}

func (serverObj *Server) PushMessageToChain(msg wire.Message, chain blockchain.ChainInterface) error {
	chainID := chain.GetShardID()
	if chainID == -1 {
		serverObj.PushMessageToBeacon(msg, map[libp2p.ID]bool{})
	} else {
		serverObj.PushMessageToShard(msg, byte(chainID), map[libp2p.ID]bool{})
	}
	return nil
}

func (serverObj *Server) DropAllConnections() {
	serverObj.connManager.DropAllConnections()
}

func (serverObj *Server) PushBlockToAll(block common.BlockInterface, isBeacon bool) error {
	if isBeacon {
		msg, err := wire.MakeEmptyMessage(wire.CmdBlockBeacon)
		if err != nil {
			Logger.log.Error(err)
			return err
		}
		msg.(*wire.MessageBlockBeacon).Block = block.(*blockchain.BeaconBlock)
		serverObj.PushMessageToAll(msg)
		return nil
	} else {
		shardBlock := block.(*blockchain.ShardBlock)
		msgShard, err := wire.MakeEmptyMessage(wire.CmdBlockShard)
		if err != nil {
			Logger.log.Error(err)
			return err
		}
		msgShard.(*wire.MessageBlockShard).Block = shardBlock
		serverObj.PushMessageToShard(msgShard, shardBlock.Header.ShardID, map[libp2p.ID]bool{})

		shardToBeaconBlk := shardBlock.CreateShardToBeaconBlock(serverObj.blockChain)
		msgShardToBeacon, err := wire.MakeEmptyMessage(wire.CmdBlkShardToBeacon)
		if err != nil {
			Logger.log.Error(err)
			return err
		}
		msgShardToBeacon.(*wire.MessageShardToBeacon).Block = shardToBeaconBlk
		serverObj.PushMessageToBeacon(msgShardToBeacon, map[libp2p.ID]bool{})
		//TODO hy check
		crossShardBlks := shardBlock.CreateAllCrossShardBlock(serverObj.blockChain.BestState.Beacon.ActiveShards)
		for shardID, crossShardBlk := range crossShardBlks {
			msgCrossShardShard, err := wire.MakeEmptyMessage(wire.CmdCrossShard)
			if err != nil {
				Logger.log.Error(err)
				return err
			}
			msgCrossShardShard.(*wire.MessageCrossShard).Block = crossShardBlk
			serverObj.PushMessageToShard(msgCrossShardShard, shardID, map[libp2p.ID]bool{})
		}
	}
	return nil
}

func (serverObj *Server) GetPublicKeyRole(publicKey string, keyType string) (int, int) {
	var beaconBestState blockchain.BeaconBestState
	err := beaconBestState.CloneBeaconBestStateFrom(serverObj.blockChain.BestState.Beacon)
	if err != nil {
		return -2, -1
	}
	for shardID, pubkeyArr := range beaconBestState.ShardPendingValidator {
		keyList, _ := incognitokey.ExtractPublickeysFromCommitteeKeyList(pubkeyArr, keyType)
		found := common.IndexOfStr(publicKey, keyList)
		if found > -1 {
			return 0, int(shardID)
		}
	}
	for shardID, pubkeyArr := range beaconBestState.ShardCommittee {
		keyList, _ := incognitokey.ExtractPublickeysFromCommitteeKeyList(pubkeyArr, keyType)
		found := common.IndexOfStr(publicKey, keyList)
		if found > -1 {
			return 1, int(shardID)
		}
	}

	keyList, _ := incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.BeaconCommittee, keyType)
	found := common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return 1, -1
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.BeaconPendingValidator, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return 0, -1
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.CandidateBeaconWaitingForCurrentRandom, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return 0, -1
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.CandidateBeaconWaitingForNextRandom, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return 0, -1
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.CandidateShardWaitingForCurrentRandom, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return 0, -1
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.CandidateShardWaitingForNextRandom, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return 0, -1
	}

	return -1, -1
}

func (serverObj *Server) GetIncognitoPublicKeyRole(publicKey string) (int, bool, int) {
	var beaconBestState blockchain.BeaconBestState
	err := beaconBestState.CloneBeaconBestStateFrom(serverObj.blockChain.BestState.Beacon)
	if err != nil {
		return -2, false, -1
	}

	for shardID, pubkeyArr := range beaconBestState.ShardPendingValidator {
		for _, key := range pubkeyArr {
			if key.GetIncKeyBase58() == publicKey {
				return 1, false, int(shardID)
			}
		}
	}
	for shardID, pubkeyArr := range beaconBestState.ShardCommittee {
		for _, key := range pubkeyArr {
			if key.GetIncKeyBase58() == publicKey {
				return 2, false, int(shardID)
			}
		}
	}

	for _, key := range beaconBestState.BeaconCommittee {
		if key.GetIncKeyBase58() == publicKey {
			return 2, true, -1
		}
	}

	for _, key := range beaconBestState.BeaconPendingValidator {
		if key.GetIncKeyBase58() == publicKey {
			return 1, true, -1
		}
	}

	for _, key := range beaconBestState.CandidateBeaconWaitingForCurrentRandom {
		if key.GetIncKeyBase58() == publicKey {
			return 0, true, -1
		}
	}

	for _, key := range beaconBestState.CandidateBeaconWaitingForNextRandom {
		if key.GetIncKeyBase58() == publicKey {
			return 0, true, -1
		}
	}

	for _, key := range beaconBestState.CandidateShardWaitingForCurrentRandom {
		if key.GetIncKeyBase58() == publicKey {
			return 0, false, -1
		}
	}
	for _, key := range beaconBestState.CandidateShardWaitingForNextRandom {
		if key.GetIncKeyBase58() == publicKey {
			return 0, false, -1
		}
	}

	return -1, false, -1
}

func (serverObj *Server) GetMinerIncognitoPublickey(publicKey string, keyType string) []byte {
	var beaconBestState blockchain.BeaconBestState
	err := beaconBestState.CloneBeaconBestStateFrom(serverObj.blockChain.BestState.Beacon)
	if err != nil {
		return nil
	}
	for _, pubkeyArr := range beaconBestState.ShardPendingValidator {
		keyList, _ := incognitokey.ExtractPublickeysFromCommitteeKeyList(pubkeyArr, keyType)
		found := common.IndexOfStr(publicKey, keyList)
		if found > -1 {
			return pubkeyArr[found].GetNormalKey()
		}
	}
	for _, pubkeyArr := range beaconBestState.ShardCommittee {
		keyList, _ := incognitokey.ExtractPublickeysFromCommitteeKeyList(pubkeyArr, keyType)
		found := common.IndexOfStr(publicKey, keyList)
		if found > -1 {
			return pubkeyArr[found].GetNormalKey()
		}
	}

	keyList, _ := incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.BeaconCommittee, keyType)
	found := common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return beaconBestState.BeaconCommittee[found].GetNormalKey()
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.BeaconPendingValidator, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return beaconBestState.BeaconPendingValidator[found].GetNormalKey()
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.CandidateBeaconWaitingForCurrentRandom, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return beaconBestState.CandidateBeaconWaitingForCurrentRandom[found].GetNormalKey()
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.CandidateBeaconWaitingForNextRandom, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return beaconBestState.CandidateBeaconWaitingForNextRandom[found].GetNormalKey()
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.CandidateShardWaitingForCurrentRandom, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return beaconBestState.CandidateShardWaitingForCurrentRandom[found].GetNormalKey()
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(beaconBestState.CandidateShardWaitingForNextRandom, keyType)
	found = common.IndexOfStr(publicKey, keyList)
	if found > -1 {
		return beaconBestState.CandidateShardWaitingForNextRandom[found].GetNormalKey()
	}

	return nil
}
