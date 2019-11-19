package blsbftv2

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus/blsbft"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wire"
	"time"
)

type Node struct {
	consensusEngine *blsbft.BLSBFT
	chain           *Chain
}

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewNode(committee []string, index int) *Node {
	node := Node{}
	node.chain = NewChain()
	node.consensusEngine = &blsbft.BLSBFT{
		Chain:    node.chain,
		Node:     node,
		ChainKey: "shard",
		PeerID:   fmt.Sprintf("node-%d", index),
	}
	prvSeed, err := node.consensusEngine.LoadUserKeyFromIncPrivateKey(committee[index])
	failOnError(err)
	failOnError(node.consensusEngine.LoadUserKey(prvSeed))
	return &node
}

func (s *Node) Start() {
	s.consensusEngine.Start()
}
func (Node) PushMessageToChain(msg wire.Message, chain blockchain.ChainInterface) error {
	panic("implement me")
}

func (Node) UpdateConsensusState(role string, userPbk string, currentShard *byte, beaconCommittee []string, shardCommittee map[byte][]string) {
	return
}

func (Node) IsEnableMining() bool {
	return true
}

func (Node) GetMiningKeys() string {
	panic("implement me")
}

func (Node) GetPrivateKey() string {
	panic("implement me")
}

func (Node) DropAllConnections() {
	return
}

type Block struct {
	height      uint64
	timestamp   int64
	proposerIdx int
}

type Chain struct {
	Blocks    []Block
	Committee []string
}

func NewChain() *Chain {
	return &Chain{
		Blocks: []Block{Block{
			height:      1,
			timestamp:   time.Date(2019, 01, 01, 00, 00, 00, 00, time.Local).Unix(),
			proposerIdx: 3,
		}},
	}
}

func (Chain) GetChainName() string {
	return "shard0"
}

func (Chain) GetConsensusType() string {
	return "bls"
}

func (s *Chain) GetLastBlockTimeStamp() int64 {
	return s.Blocks[len(s.Blocks)-1].timestamp
}

func (Chain) GetMinBlkInterval() time.Duration {
	return time.Second * 1
}

func (Chain) GetMaxBlkCreateTime() time.Duration {
	return time.Second * 1
}

func (Chain) IsReady() bool {
	return true
}

func (Chain) GetActiveShardNumber() int {
	return 8
}

func (Chain) GetPubkeyRole(pubkey string, round int) (string, byte) {
	return "committee", 0
}

func (s *Chain) CurrentHeight() uint64 {
	return s.Blocks[len(s.Blocks)-1].height
}

func (s *Chain) GetCommitteeSize() int {
	return len(s.Committee)
}

func (s *Chain) GetCommittee() []incognitokey.CommitteePublicKey {
	panic("implement me")
}

func (s *Chain) GetPubKeyCommitteeIndex(string) int {
	panic("implement me")
}

func (s *Chain) GetLastProposerIndex() int {
	return s.Blocks[len(s.Blocks)-1].proposerIdx
}

func (Chain) UnmarshalBlock(blockString []byte) (common.BlockInterface, error) {
	panic("implement me")
}

func (Chain) CreateNewBlock(round int) (common.BlockInterface, error) {
	panic("implement me")
}

func (Chain) InsertBlk(block common.BlockInterface) error {
	return nil
}

func (Chain) InsertAndBroadcastBlock(block common.BlockInterface) error {
	return nil
}

func (Chain) ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error {
	return nil
}

func (Chain) ValidatePreSignBlock(block common.BlockInterface) error {
	return nil
}

func (Chain) GetShardID() int {
	return 0
}
