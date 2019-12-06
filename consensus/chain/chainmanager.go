package chain

import (
	"bytes"
	"encoding/gob"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"sync"
	"time"
)

/*
	How to load Chain Manager when bootup
	Logic to store chainview
	How to manage Chain View
*/

type ChainManager struct {
	view *ViewGraph
	name string
	lock *sync.RWMutex
}

//to get byte array of this chain data, for loading later
func (s *ChainManager) GetChainData() (res []byte) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(s)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

//create new chain with root view
func InitNewChain(name string, rootView blockchain.ChainViewInterface) *ChainManager {
	cm := &ChainManager{
		name: name,
		lock: new(sync.RWMutex),
	}
	cm.view = NewViewGraph(name, rootView, cm.lock)
	return cm
}

func LoadChain(data []byte) (res *ChainManager) {
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(res)
	if err != nil {
		panic(err)
	}
	return res
}

func (s *ChainManager) AddChainView() {
	return
}

func (ChainManager) GetChainName() string {
	panic("implement me")
}

func (ChainManager) IsReady() bool {
	panic("implement me")
}

func (ChainManager) GetActiveShardNumber() int {
	panic("implement me")
}

func (ChainManager) GetPubkeyRole(pubkey string, round int) (string, byte) {
	panic("implement me")
}

func (ChainManager) GetConsensusType() string {
	panic("implement me")
}

func (ChainManager) GetLastBlockTimeStamp() int64 {
	panic("implement me")
}

func (ChainManager) GetMinBlkInterval() time.Duration {
	panic("implement me")
}

func (ChainManager) GetMaxBlkCreateTime() time.Duration {
	panic("implement me")
}

func (ChainManager) CurrentHeight() uint64 {
	panic("implement me")
}

func (ChainManager) GetCommitteeSize() int {
	panic("implement me")
}

func (ChainManager) GetCommittee() []incognitokey.CommitteePublicKey {
	panic("implement me")
}

func (ChainManager) GetPubKeyCommitteeIndex(string) int {
	panic("implement me")
}

func (ChainManager) GetLastProposerIndex() int {
	panic("implement me")
}

func (ChainManager) UnmarshalBlock(blockString []byte) (common.BlockInterface, error) {
	panic("implement me")
}

func (ChainManager) ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error {
	panic("implement me")
}

func (ChainManager) GetShardID() int {
	panic("implement me")
}

func (ChainManager) GetBestView() blockchain.ChainViewInterface {
	panic("implement me")
}

func (ChainManager) GetFinalView() blockchain.ChainViewInterface {
	panic("implement me")
}

func (ChainManager) GetAllViews() map[string]blockchain.ChainViewInterface {
	panic("implement me")
}

func (ChainManager) GetViewByHash(*common.Hash) (blockchain.ChainViewInterface, error) {
	panic("implement me")
}

func (ChainManager) GetGenesisTime() int64 {
	panic("implement me")
}
