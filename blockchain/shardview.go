package blockchain

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
)

// BestState houses information about the current best block and other info
// related to the state of the main chain as it exists from the point of view of
// the current best block.
//
// The BestSnapshot method can be used to obtain access to this information
// in a concurrent safe manner and the data will not be changed out from under
// the caller when chain state changes occur as the function name implies.
// However, the returned snapshot must be treated as immutable since it is
// shared by all callers.

type ShardView struct {
	BestBlockHash          common.Hash                       `json:"BestBlockHash"` // hash of block.
	BestBlock              *ShardBlock                       `json:"BestBlock"`     // block data
	BestBeaconHash         common.Hash                       `json:"BestBeaconHash"`
	BeaconHeight           uint64                            `json:"BeaconHeight"`
	ShardID                byte                              `json:"ShardID"`
	Epoch                  uint64                            `json:"Epoch"`
	ShardHeight            uint64                            `json:"ShardHeight"`
	MaxShardCommitteeSize  int                               `json:"MaxShardCommitteeSize"`
	MinShardCommitteeSize  int                               `json:"MinShardCommitteeSize"`
	ShardProposerIdx       int                               `json:"ShardProposerIdx"`
	ShardCommitteeHash     string                            `json:"CommitteeHash"`
	ShardCommittee         []incognitokey.CommitteePublicKey `json:"ShardCommittee"`
	ShardPendingValidator  []incognitokey.CommitteePublicKey `json:"ShardPendingValidator"`
	BestCrossShard         map[byte]uint64                   `json:"BestCrossShard"` // Best cross shard block by heigh
	StakingTx              map[string]string                 `json:"StakingTx"`
	NumTxns                uint64                            `json:"NumTxns"`                // The number of txns in the block.
	TotalTxns              uint64                            `json:"TotalTxns"`              // The total number of txns in the chain.
	TotalTxnsExcludeSalary uint64                            `json:"TotalTxnsExcludeSalary"` // for testing and benchmark
	ActiveShards           int                               `json:"ActiveShards"`
	ConsensusAlgorithm     string                            `json:"ConsensusAlgorithm"`
	ConsensusConfig        string                            `json:"ConsensusConfig"`

	// Number of blocks produced by producers in epoch
	NumOfBlocksByProducers map[string]uint64 `json:"NumOfBlocksByProducers"`

	BlockInterval      time.Duration
	BlockMaxCreateTime time.Duration

	GenesisTime int64 //use for consensus to get timeslot
	IsBest      bool
	// MetricBlockHeight uint64
	lock sync.RWMutex
}

func NewShardView() *ShardView {
	var view ShardView
	return &view
}
func NewShardViewWithConfig(shardID byte, netparam *Params) *ShardView {
	var view ShardView
	err := view.BestBlockHash.SetBytes(make([]byte, 32))
	if err != nil {
		panic(err)
	}
	err = view.BestBeaconHash.SetBytes(make([]byte, 32))
	if err != nil {
		panic(err)
	}
	view.BestBlock = nil
	view.ShardCommittee = []incognitokey.CommitteePublicKey{}
	view.MaxShardCommitteeSize = netparam.MaxShardCommitteeSize
	view.MinShardCommitteeSize = netparam.MinShardCommitteeSize
	view.ShardPendingValidator = []incognitokey.CommitteePublicKey{}
	view.ActiveShards = netparam.ActiveShards
	view.BestCrossShard = make(map[byte]uint64)
	view.StakingTx = make(map[string]string)
	view.ShardHeight = 1
	view.BeaconHeight = 1
	view.BlockInterval = netparam.MinShardBlockInterval
	view.BlockMaxCreateTime = netparam.MaxShardBlockCreation
	return &view
}

// Get role of a public key base on best state shard
func (view *ShardView) GetBytes() []byte {
	res := []byte{}
	res = append(res, view.BestBlockHash.GetBytes()...)
	res = append(res, view.BestBlock.Hash().GetBytes()...)
	res = append(res, view.BestBeaconHash.GetBytes()...)
	beaconHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(beaconHeightBytes, view.BeaconHeight)
	res = append(res, beaconHeightBytes...)
	res = append(res, view.ShardID)
	epochBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBytes, view.Epoch)
	res = append(res, epochBytes...)
	shardHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(shardHeightBytes, view.ShardHeight)
	res = append(res, shardHeightBytes...)
	shardCommitteeSizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(shardCommitteeSizeBytes, uint32(view.MaxShardCommitteeSize))
	res = append(res, shardCommitteeSizeBytes...)
	minShardCommitteeSizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(minShardCommitteeSizeBytes, uint32(view.MinShardCommitteeSize))
	res = append(res, minShardCommitteeSizeBytes...)
	proposerIdxBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(proposerIdxBytes, uint32(view.ShardProposerIdx))
	res = append(res, proposerIdxBytes...)
	for _, value := range view.ShardCommittee {
		valueBytes, err := value.Bytes()
		if err != nil {
			return nil
		}
		res = append(res, valueBytes...)
	}
	for _, value := range view.ShardPendingValidator {
		valueBytes, err := value.Bytes()
		if err != nil {
			return nil
		}
		res = append(res, valueBytes...)
	}
	keys := []int{}
	for k := range view.BestCrossShard {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, shardID := range keys {
		value := view.BestCrossShard[byte(shardID)]
		valueBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(valueBytes, value)
		res = append(res, valueBytes...)
	}
	keystr := []string{}
	for _, k := range view.StakingTx {
		keystr = append(keystr, k)
	}
	sort.Strings(keystr)
	for _, key := range keystr {
		value := view.StakingTx[key]
		res = append(res, []byte(key)...)
		res = append(res, []byte(value)...)
	}
	numTxnsBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(numTxnsBytes, view.NumTxns)
	res = append(res, numTxnsBytes...)
	totalTxnsBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(totalTxnsBytes, view.TotalTxns)
	res = append(res, totalTxnsBytes...)
	activeShardsBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(activeShardsBytes, uint32(view.ActiveShards))
	res = append(res, activeShardsBytes...)
	return res
}

func (view *ShardView) Hash() common.Hash {
	view.lock.RLock()
	defer view.lock.RUnlock()
	return common.HashH(view.GetBytes())
}

func (view *ShardView) SetMaxShardCommitteeSize(maxShardCommitteeSize int) bool {
	view.lock.Lock()
	defer view.lock.Unlock()
	// check input params, below MinCommitteeSize failed to acheive consensus
	if maxShardCommitteeSize < MinCommitteeSize {
		return false
	}
	// max committee size can't be lower than current min committee size
	if maxShardCommitteeSize >= view.MinShardCommitteeSize {
		view.MaxShardCommitteeSize = maxShardCommitteeSize
		return true
	}
	return false
}

func (view *ShardView) SetMinShardCommitteeSize(minShardCommitteeSize int) bool {
	view.lock.Lock()
	defer view.lock.Unlock()
	// check input params, below MinCommitteeSize failed to acheive consensus
	if minShardCommitteeSize < MinCommitteeSize {
		return false
	}
	// min committee size can't be greater than current min committee size
	if minShardCommitteeSize <= view.MaxShardCommitteeSize {
		view.MinShardCommitteeSize = minShardCommitteeSize
		return true
	}
	return false
}

func (view *ShardView) GetPubkeyRole(pubkey string, round int) string {
	keyList, _ := incognitokey.ExtractPublickeysFromCommitteeKeyList(view.ShardCommittee, view.ConsensusAlgorithm)
	// fmt.Printf("pubkey %v key list %v\n\n\n\n", pubkey, keyList)
	found := common.IndexOfStr(pubkey, keyList)
	if found > -1 {
		tmpID := (view.ShardProposerIdx + round) % len(keyList)
		if found == tmpID {
			return common.ProposerRole
		} else {
			return common.ValidatorRole
		}
	}

	keyList, _ = incognitokey.ExtractPublickeysFromCommitteeKeyList(view.ShardPendingValidator, view.ConsensusAlgorithm)
	found = common.IndexOfStr(pubkey, keyList)
	if found > -1 {
		return common.PendingRole
	}
	return common.EmptyString
}

func (view *ShardView) MarshalJSON() ([]byte, error) {
	view.lock.RLock()
	defer view.lock.RUnlock()
	type Alias ShardView
	b, err := json.Marshal(&struct {
		*Alias
	}{
		(*Alias)(view),
	})
	if err != nil {
		Logger.log.Error(err)
	}
	return b, err
}

func (view ShardView) GetShardHeight() uint64 {
	return view.ShardHeight
}

func (view ShardView) GetBeaconHeight() uint64 {
	return view.BeaconHeight
}

func (view *ShardView) cloneShardViewFrom(target ChainViewInterface) error {
	tempMarshal, err := json.Marshal(target)
	if err != nil {
		return NewBlockChainError(MashallJsonShardBestStateError, fmt.Errorf("Shard Best State %+v get %+v", target.CurrentHeight(), err))
	}
	err = json.Unmarshal(tempMarshal, view)
	if err != nil {
		return NewBlockChainError(UnmashallJsonShardBestStateError, fmt.Errorf("Clone Shard Best State %+v get %+v", target.CurrentHeight(), err))
	}
	if reflect.DeepEqual(*view, ShardView{}) {
		return NewBlockChainError(CloneShardBestStateError, fmt.Errorf("Shard Best State %+v clone failed", target.CurrentHeight()))
	}
	return nil
}
func (view *ShardView) GetStakingTx() map[string]string {
	view.lock.RLock()
	defer view.lock.RUnlock()
	m := make(map[string]string)
	for k, v := range view.StakingTx {
		m[k] = v
	}
	return m
}

func (view *ShardView) GetConsensusType() string {
	return ""
}
func (view *ShardView) GetPubKeyCommitteeIndex(string) int {
	return 0
}
func (view *ShardView) GetLastProposerIndex() int {
	return 0
}
func (view *ShardView) CreateNewBlock(timeslot uint64) (common.BlockInterface, error) {
	return nil, nil
}
func (view *ShardView) InsertBlk(block common.BlockInterface) error {
	return nil
}
func (view *ShardView) InsertAndBroadcastBlock(block common.BlockInterface) error {
	return nil
}
func (view *ShardView) ValidatePreSignBlock(block common.BlockInterface) error {
	return nil
}

func (view *ShardView) DeleteView() error {
	return nil
}

func (view *ShardView) GetConsensusConfig() string {
	return view.ConsensusConfig
}

func (view *ShardView) CurrentHeight() uint64 {
	return 0
}

func (view *ShardView) GetBlkMaxCreateTime() time.Duration {
	return 0
}

func (view *ShardView) GetBlkMinInterval() time.Duration {
	return 0
}

func (view *ShardView) GetLastBlockTimeStamp() int64 {
	return 0
}
func (view *ShardView) GetCommittee() []incognitokey.CommitteePublicKey {
	result := []incognitokey.CommitteePublicKey{}
	result = append([]incognitokey.CommitteePublicKey{}, view.ShardCommittee...)
	return result
}
func (view *ShardView) GetLastProposerIdx() int { return 0 }

func (view *ShardView) GetTimeslot() uint64 { return 0 }

func (view *ShardView) GetEpoch() uint64 {
	return 0
}

func (view ShardView) GetTxsInBestBlock() []metadata.Transaction {
	view.lock.RLock()
	defer view.lock.RUnlock()
	var result []metadata.Transaction
	copy(result, view.BestBlock.Body.Transactions)
	return result
}

func (view *ShardView) IsBestView() bool {
	return view.IsBest
}

func (view *ShardView) SetViewIsBest(isBest bool) {
	view.lock.Lock()
	defer view.lock.Unlock()
	view.IsBest = isBest
}

func (view *ShardView) GetTipBlock() common.BlockInterface {
	return view.BestBlock
}

func (view *ShardView) GetCommitteeHash() *common.Hash {
	view.lock.RLock()
	defer view.lock.RUnlock()
	result, err := common.Hash{}.NewHashFromStr(view.ShardCommitteeHash)
	if err != nil {
		panic(err)
	}
	return result
}
