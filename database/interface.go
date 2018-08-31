package database

import (
	"github.com/ninjadotorg/cash-prototype/common"
	"github.com/ninjadotorg/cash-prototype/transaction"
)

// DB provides the interface that is used to store blocks.
type DB interface {
	StoreBlock(v interface{}) error
	FetchBlock(*common.Hash) ([]byte, error)
	HasBlock(*common.Hash) (bool, error)
	FetchAllBlocks() ([]*common.Hash, error)

	StoreBestBlock(v interface{}) error
	FetchBestState() ([]byte, error)

	StoreTx([]byte) error

	StoreBlockIndex(*common.Hash, int32) error
	GetIndexOfBlock(*common.Hash) (int32, error)
	GetBlockByIndex(int32) (*common.Hash, error)

	StoreEntry(*transaction.OutPoint, interface{}) error
	FetchEntry(*transaction.OutPoint) ([]byte, error)

	Close() error
}
