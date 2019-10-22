package blockchain

import (
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
)

type ShardToBeaconPool interface {
	RemoveBlock(map[byte]uint64)
	//GetFinalBlock() map[byte][]ShardToBeaconBlock
	AddShardToBeaconBlock(*ShardToBeaconBlock) (uint64, uint64, error)
	//ValidateShardToBeaconBlock(ShardToBeaconBlock) error
	GetValidBlockHash() map[byte][]common.Hash
	GetValidBlock(map[byte]uint64) map[byte][]*ShardToBeaconBlock
	GetValidBlockHeight() map[byte][]uint64
	GetLatestValidPendingBlockHeight() map[byte]uint64
	GetBlockByHeight(shardID byte, height uint64) *ShardToBeaconBlock
	SetShardState(map[byte]uint64)
	GetAllBlockHeight() map[byte][]uint64
	RevertShardToBeaconPool(s byte, height uint64)
}

type CrossShardPool interface {
	AddCrossShardBlock(*CrossShardBlock) (map[byte]uint64, byte, error)
	GetValidBlock(map[byte]uint64) map[byte][]*CrossShardBlock
	GetLatestValidBlockHeight() map[byte]uint64
	GetValidBlockHeight() map[byte][]uint64
	GetBlockByHeight(_shardID byte, height uint64) *CrossShardBlock
	RemoveBlockByHeight(map[byte]uint64)
	UpdatePool() map[byte]uint64
	GetAllBlockHeight() map[byte][]uint64
	RevertCrossShardPool(uint64)
}

type ShardPool interface {
	RemoveBlock(height uint64)
	AddShardBlock(block *ShardBlock) error
	GetValidBlockHash() []common.Hash
	GetValidBlock() []*ShardBlock
	GetValidBlockHeight() []uint64
	GetLatestValidBlockHeight() uint64
	SetShardState(height uint64)
	RevertShardPool(uint64)
	GetAllBlockHeight() []uint64
	Start(chan struct{})
}

type BeaconPool interface {
	RemoveBlock(height uint64)
	AddBeaconBlock(block *BeaconBlock) error
	GetValidBlock() []*BeaconBlock
	GetValidBlockHeight() []uint64
	SetBeaconState(height uint64)
	RevertBeconPool(height uint64)
	GetAllBlockHeight() []uint64
	Start(chan struct{})
}
type TxPool interface {
	// LastUpdated returns the last time a transaction was added to or
	// removed from the source pool.
	LastUpdated() time.Time

	// MiningDescs returns a slice of mining descriptors for all the
	// transactions in the source pool.
	MiningDescs() []*metadata.TxDesc

	// HaveTransaction returns whether or not the passed transaction hash
	// exists in the source pool.
	HaveTransaction(hash *common.Hash) bool

	// RemoveTx remove tx from tx resource
	RemoveTx(txs []metadata.Transaction, isInBlock bool)

	RemoveCandidateList([]string)

	RemoveTokenIDList([]string)

	EmptyPool() bool

	MaybeAcceptTransactionForBlockProducing(metadata.Transaction) (*metadata.TxDesc, error)
	ValidateTxList(txs []metadata.Transaction) error
	//CheckTransactionFee
	// CheckTransactionFee(tx metadata.Transaction) (uint64, error)

	// Check tx validate by it self
	// ValidateTxByItSelf(tx metadata.Transaction) bool
}

type FeeEstimator interface {
	RegisterBlock(block *ShardBlock) error
}

type ChainInterface interface {
	// GetChainName - get nam of this chain
	GetChainName() string
	// GetConsensusType - get what type of consensus this chain run
	GetConsensusType() string
	// GetLastBlockTimeStamp - get last block timestamp
	GetLastBlockTimeStamp() int64
	// GetMinBlockInterval - get minimum time interval between block
	GetMinBlockInterval() time.Duration
	// GetMaxBlockCreateTime - get maximum time to create block
	GetMaxBlockCreateTime() time.Duration
	// IsReady - check whether this chain is synced
	IsReady() bool
	// GetPubkeyRole - get public key role in this chain
	GetPubkeyRole(pubkey string, round int) (string, byte)
	// CurrentHeight - get the current height of this chain
	CurrentHeight() uint64
	// GetCommitteeSize - get committee size of this chain
	GetCommitteeSize() int
	// GetCommittee -  get committee list of this chain
	GetCommittee() []incognitokey.CommitteePublicKey
	// GetPubKeyCommitteeIndex - get publickey index in committee list of this chain
	GetPubKeyCommitteeIndex(string) int
	// GetLastProposerIndex - get last proposer index
	GetLastProposerIndex() int
	// UnmarshalBlock - unmarshal block belong to this type chain
	UnmarshalBlock(blockString []byte) (common.BlockInterface, error)
	// CreateNewBlock - create new block belong to this type of chain
	CreateNewBlock(round int) (common.BlockInterface, error)
	// InsertBlock - insert block but skip validation
	InsertBlock(block common.BlockInterface) error
	// InsertAndBroadcastBlock - insert block to chain then broadcast block (used by consensus)
	InsertAndBroadcastBlock(block common.BlockInterface) error
	// ValidateBlockSignatures - validate block signatures base-on a committee pubkey list
	ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error
	// ValidatePreSignBlock - validate block data that not signed by committee yet
	ValidatePreSignBlock(block common.BlockInterface) error
	// GetShardID - get shardID of this chain
	GetShardID() int
}
