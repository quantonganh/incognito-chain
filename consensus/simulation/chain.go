package main

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"time"
)

type Chain struct {
	BlockGraph      *BlockGraph
	UserPubKey      incognitokey.CommitteePublicKey
	CommitteePubkey []incognitokey.CommitteePublicKey
}

func NewChain(name string, committeePkStruct []incognitokey.CommitteePublicKey) *Chain {
	rootBlock := NewBlock(1, START_TIME, "", common.Hash{})
	bg := NewBlockGraph(name, rootBlock)
	bg.GetBestViewBlock()

	return &Chain{
		BlockGraph:      bg,
		CommitteePubkey: committeePkStruct,
	}
}

func (Chain) GetChainName() string {
	return "shard0"
}

func (Chain) GetConsensusType() string {
	return "bls"
}

func (s *Chain) GetLastBlockTimeStamp() int64 {
	b := s.BlockGraph.bestView.block
	return b.GetTimeStamp()
}

func (Chain) GetMinBlkInterval() time.Duration {
	return time.Second * TIMESLOT
}

func (Chain) GetMaxBlkCreateTime() time.Duration {
	return time.Second * TIMESLOT
}

func (Chain) IsReady() bool {
	return true
}

func (Chain) GetActiveShardNumber() int {
	return 8
}

func (s *Chain) CurrentHeight() uint64 {
	b := s.BlockGraph.bestView.block
	return b.GetHeight()
}

func (s *Chain) GetCommitteeSize() int {
	return len(s.CommitteePubkey)
}

func (s *Chain) GetPubKeyCommitteeIndex(pk string) int {
	for index, key := range s.CommitteePubkey {
		if key.GetMiningKeyBase58("bls") == pk {
			return index
		}
	}
	return -1
}

func (s *Chain) GetLastProposerIndex() int {
	b := s.BlockGraph.bestView.block
	strArr, _ := incognitokey.CommitteeKeyListToString(s.CommitteePubkey)
	return common.IndexOfStr(b.GetProducer(), strArr)
}

func (Chain) UnmarshalBlock(blockString []byte) (common.BlockInterface, error) {
	b := NewEmptyBlock()
	e := json.Unmarshal(blockString, &b)
	return b, e
}

func (s *Chain) CreateNewBlock(round int) (common.BlockInterface, error) {
	b := s.BlockGraph.bestView.block
	str, _ := s.UserPubKey.ToBase58()
	nb := NewBlock(b.GetHeight()+1, time.Now().Unix(), str, *b.Hash())
	return nb, nil
}

func (s *Chain) InsertAndBroadcastBlock(block common.BlockInterface) error {
	s.BlockGraph.AddBlock(block)
	s.BlockGraph.GetBestViewBlock()
	return nil
}

func (Chain) ValidatePreSignBlock(block common.BlockInterface) error {
	return nil
}

func (Chain) GetShardID() int {
	return 0
}

func (s *Chain) GetCommittee() []incognitokey.CommitteePublicKey {
	return s.CommitteePubkey
}

func (Chain) GetPubkeyRole(pubkey string, round int) (string, byte) {
	//not use in bft
	panic("implement me")
}

func (Chain) InsertBlk(block common.BlockInterface) error {
	//not use in bft
	panic("implement me")
}

func (Chain) ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error {
	//not use in bft
	panic("implement me")
}
