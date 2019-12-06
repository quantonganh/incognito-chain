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

type ViewManager struct {
	manager *ViewGraph
	name    string
	lock    *sync.RWMutex
}

//to get byte array of this chain data, for loading later
func (s *ViewManager) GetChainData() (res []byte) {
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
func InitNewChain(name string, rootView blockchain.ChainViewInterface) *ViewManager {
	cm := &ViewManager{
		name: name,
		lock: new(sync.RWMutex),
	}
	cm.manager = NewViewGraph(name, rootView, cm.lock)
	return cm
}

func LoadChain(data []byte) (res *ViewManager) {
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(res)
	if err != nil {
		panic(err)
	}
	return res
}

func (s *ViewManager) AddChainView(view blockchain.ChainViewInterface) {
	s.manager.AddView(view)
}

func (ViewManager) GetChainName() string {
	panic("implement me")
}

func (ViewManager) IsReady() bool {
	panic("implement me")
}

func (ViewManager) GetActiveShardNumber() int {
	panic("implement me")
}

func (ViewManager) GetPubkeyRole(pubkey string, round int) (string, byte) {
	panic("implement me")
}

func (ViewManager) GetConsensusType() string {
	panic("implement me")
}

func (ViewManager) GetLastBlockTimeStamp() int64 {
	panic("implement me")
}

func (ViewManager) GetMinBlkInterval() time.Duration {
	panic("implement me")
}

func (ViewManager) GetMaxBlkCreateTime() time.Duration {
	panic("implement me")
}

func (ViewManager) CurrentHeight() uint64 {
	panic("implement me")
}

func (ViewManager) GetCommitteeSize() int {
	panic("implement me")
}

func (ViewManager) GetCommittee() []incognitokey.CommitteePublicKey {
	panic("implement me")
}

func (ViewManager) GetPubKeyCommitteeIndex(string) int {
	panic("implement me")
}

func (ViewManager) GetLastProposerIndex() int {
	panic("implement me")
}

func (ViewManager) UnmarshalBlock(blockString []byte) (common.BlockInterface, error) {
	panic("implement me")
}

func (ViewManager) ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error {
	panic("implement me")
}

func (s *ViewManager) GetShardID() int {
	panic("implement me")
}

func (s *ViewManager) GetBestView() blockchain.ChainViewInterface {
	return s.manager.GetBestView()
}

func (s *ViewManager) GetFinalView() blockchain.ChainViewInterface {
	return s.manager.GetBestView()
}

func (ViewManager) GetAllViews() map[string]blockchain.ChainViewInterface {
	panic("implement me")
}

func (s *ViewManager) GetViewByHash(h *common.Hash) blockchain.ChainViewInterface {
	viewnode, ok := s.manager.node[*h]
	if !ok {
		return nil
	} else {
		return viewnode.view
	}
}

func (ViewManager) GetGenesisTime() int64 {
	panic("implement me")
}
