package main

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"time"
)

type Chain struct {
	Blocks          []Block
	CommitteePubkey []incognitokey.CommitteePublicKey
}

func NewChain(committeePkStruct []incognitokey.CommitteePublicKey) *Chain {
	return &Chain{
		Blocks: []Block{Block{
			Height:      1,
			Timestamp:   time.Date(2019, 01, 01, 00, 00, 00, 00, time.Local).Unix(),
			ProposerIdx: 3,
		}},
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
	return s.Blocks[len(s.Blocks)-1].Timestamp
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

func (s *Chain) CurrentHeight() uint64 {
	return s.Blocks[len(s.Blocks)-1].Height
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
	return s.Blocks[len(s.Blocks)-1].ProposerIdx
}

func (Chain) UnmarshalBlock(blockString []byte) (common.BlockInterface, error) {
	b := &Block{}
	e := json.Unmarshal(blockString, &b)
	return b, e
}

func (Chain) CreateNewBlock(round int) (common.BlockInterface, error) {
	panic("implement me")

}

func (Chain) InsertAndBroadcastBlock(block common.BlockInterface) error {
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
