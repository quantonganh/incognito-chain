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

type ChainViewManager struct {
	manager *ViewGraph
	name    string
	lock    *sync.RWMutex
}

//to get byte array of this chain data, for loading later
func (s *ChainViewManager) GetChainData() (res []byte) {
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

//create new chain with root manager
func InitNewChain(name string, rootView blockchain.ChainViewInterface) *ChainViewManager {
	cm := &ChainViewManager{
		name: name,
		lock: new(sync.RWMutex),
	}
	cm.manager = NewViewGraph(name, rootView, cm.lock)
	return cm
}

func LoadChain(data []byte) (res *ChainViewManager) {
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(res)
	if err != nil {
		panic(err)
	}
	return res
}

func (s *ChainViewManager) AddChainView() {
	return
}

func (ChainViewManager) GetChainName() string {
	panic("implement me")
}

func (ChainViewManager) IsReady() bool {
	panic("implement me")
}

func (ChainViewManager) GetActiveShardNumber() int {
	panic("implement me")
}

func (ChainViewManager) GetPubkeyRole(pubkey string, round int) (string, byte) {
	panic("implement me")
}

func (ChainViewManager) GetConsensusType() string {
	panic("implement me")
}

func (ChainViewManager) GetLastBlockTimeStamp() int64 {
	panic("implement me")
}

func (ChainViewManager) GetMinBlkInterval() time.Duration {
	panic("implement me")
}

func (ChainViewManager) GetMaxBlkCreateTime() time.Duration {
	panic("implement me")
}

func (ChainViewManager) CurrentHeight() uint64 {
	panic("implement me")
}

func (ChainViewManager) GetCommitteeSize() int {
	panic("implement me")
}

func (ChainViewManager) GetCommittee() []incognitokey.CommitteePublicKey {
	panic("implement me")
}

func (ChainViewManager) GetPubKeyCommitteeIndex(string) int {
	panic("implement me")
}

func (ChainViewManager) GetLastProposerIndex() int {
	panic("implement me")
}

func (ChainViewManager) UnmarshalBlock(blockString []byte) (common.BlockInterface, error) {
	panic("implement me")
}

func (ChainViewManager) ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error {
	panic("implement me")
}

func (ChainViewManager) GetShardID() int {
	panic("implement me")
}

func (s *ChainViewManager) GetBestView() blockchain.ChainViewInterface {
	return s.manager.GetBestView()
}

func (s *ChainViewManager) GetFinalView() blockchain.ChainViewInterface {
	return s.manager.GetBestView()
}

func (ChainViewManager) GetAllViews() map[string]blockchain.ChainViewInterface {
	panic("implement me")
}

func (s *ChainViewManager) GetViewByHash(h *common.Hash) blockchain.ChainViewInterface {
	viewnode, ok := s.manager.node[*h]
	if !ok {
		return nil
	} else {
		return viewnode.view
	}
}

func (ChainViewManager) GetGenesisTime() int64 {
	panic("implement me")
}
