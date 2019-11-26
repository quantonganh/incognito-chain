package jsonresult

import (
	"github.com/incognitochain/incognito-chain/common"
)

type ReceivedTransaction struct {
	TransactionDetail
	ReceivedAmounts map[common.Hash]ReceivedInfo `json:"ReceivedAmounts"`
	FromShardID     byte                         `json:"FromShardID"`
}

type ReceivedInfo struct {
	CoinDetails          ReceivedCoin `json:"CoinDetails"`
	CoinDetailsEncrypted string       `json:"CoinDetailsEncrypted"`
}

type ReceivedCoin struct {
	PublicKey string `json:"PublicKey"`
	Info      string `json:"Info"`
	Value     uint64 `json:"Value"`
}

type ListReceivedTransaction struct {
	ReceivedTransactions []ReceivedTransaction `json:"ReceivedTransactions"`
}
