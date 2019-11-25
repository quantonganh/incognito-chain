package main

import (
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"math"
	"time"
)

const TIMESLOT = 2 //s

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewEmptyBlock() common.BlockInterface {
	return &blockchain.ShardBlock{}
}

func NewBlock(height uint64, time int64, prev common.Hash) common.BlockInterface {
	return &blockchain.ShardBlock{
		Header: blockchain.ShardHeader{
			Height:            height,
			Timestamp:         time,
			PreviousBlockHash: prev,
		},
		Body: blockchain.ShardBody{},
	}
}

func GetTimeSlot(t int64) int {
	return int(math.Floor(float64(time.Now().Unix()-t) / 10))
}
func NextTimeSlot(t int64) int64 {
	return t + TIMESLOT
}
