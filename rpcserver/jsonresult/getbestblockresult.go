package jsonresult

import "github.com/incognitochain/incognito-chain/blockchain"

type GetBestBlockResult struct {
	BestBlocks map[int]GetBestBlockItem `json:"BestBlocks"`
}

type GetBestBlockItem struct {
	Height         uint64 `json:"Height"`
	Hash           string `json:"Hash"`
	TotalTxs       uint64 `json:"TotalTxs"`
	BlockProducer  string `json:"BlockProducer"`
	ValidationData string `json:"ValidationData"`
	Epoch          uint64 `json:"Epoch"`
	Time           int64  `json:"Time"`
}

func NewGetBestBlockItemFromShard(bestView *blockchain.ShardView) *GetBestBlockItem {
	result := &GetBestBlockItem{
		Height:         bestView.BestBlock.Header.Height,
		Hash:           bestView.BestBlockHash.String(),
		TotalTxs:       bestView.TotalTxns,
		BlockProducer:  bestView.BestBlock.Header.Producer,
		ValidationData: bestView.BestBlock.GetValidationField(),
		Time:           bestView.BestBlock.Header.Timestamp,
	}
	return result
}

func NewGetBestBlockItemFromBeacon(bestView *blockchain.BeaconView) *GetBestBlockItem {
	result := &GetBestBlockItem{
		Height:         bestView.BestBlock.Header.Height,
		Hash:           bestView.BestBlock.Hash().String(),
		BlockProducer:  bestView.BestBlock.Header.Producer,
		ValidationData: bestView.BestBlock.GetValidationField(),
		Epoch:          bestView.Epoch,
		Time:           bestView.BestBlock.Header.Timestamp,
	}
	return result
}

type GetBestBlockHashResult struct {
	BestBlockHashes map[int]string `json:"BestBlockHashes"`
}
