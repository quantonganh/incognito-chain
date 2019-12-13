package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/transaction"
)

const (
	RewardBase = 1666
	Duration   = 1000000
)

/*
	Two kind of Instruction:
	- Instruction created directly from proposer
		+ Swap instruction
		+ Brigde instruction
		+ Committees Pubkey instruction
		+
*/
type ShardBody struct {
	Instructions      [][]string
	CrossTransactions map[byte][]CrossTransaction //CrossOutputCoin from all other shard
	Transactions      []metadata.Transaction
}

/*
Customize UnmarshalJSON to parse list TxNormal
because we have many types of block, so we can need to customize data from marshal from json string to build a block
*/
func (shardBody *ShardBody) UnmarshalJSON(data []byte) error {
	type Alias ShardBody
	temp := &struct {
		Transactions []map[string]interface{}
		*Alias
	}{
		Alias: (*Alias)(shardBody),
	}

	err := json.Unmarshal(data, &temp)
	if err != nil {
		return NewBlockChainError(UnmashallJsonShardBlockError, err)
	}

	// process tx from tx interface of temp
	for _, txTemp := range temp.Transactions {
		txTempJson, _ := json.MarshalIndent(txTemp, "", "\t")
		//Logger.log.Debugf("Tx json data: ", string(txTempJson))

		var tx metadata.Transaction
		var parseErr error
		switch txTemp["Type"].(string) {
		case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType:
			{
				tx = &transaction.Tx{}
				parseErr = json.Unmarshal(txTempJson, &tx)
			}
		case common.TxCustomTokenType:
			{
				tx = &transaction.TxNormalToken{}
				parseErr = json.Unmarshal(txTempJson, &tx)
			}
		case common.TxCustomTokenPrivacyType:
			{
				tx = &transaction.TxCustomTokenPrivacy{}
				parseErr = json.Unmarshal(txTempJson, &tx)
			}
		default:
			{
				return NewBlockChainError(UnmashallJsonShardBlockError, errors.New("can not parse a wrong tx"))
			}
		}
		if parseErr != nil {
			return NewBlockChainError(UnmashallJsonShardBlockError, parseErr)
		}
		shardBody.Transactions = append(shardBody.Transactions, tx)
	}
	return nil
}
func (shardBody ShardBody) Hash() common.Hash {
	res := []byte{}

	for _, item := range shardBody.Instructions {
		for _, l := range item {
			res = append(res, []byte(l)...)
		}
	}
	keys := []int{}
	for k := range shardBody.CrossTransactions {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, shardID := range keys {
		for _, value := range shardBody.CrossTransactions[byte(shardID)] {
			res = append(res, []byte(fmt.Sprintf("%v", value.BlockHeight))...)
			res = append(res, value.BlockHash.GetBytes()...)
			for _, coins := range value.OutputCoinWithIndex {
				res = append(res, coins.Bytes()...)
			}
			for _, coins := range value.TokenPrivacyData {
				res = append(res, coins.Bytes()...)
			}
		}
	}
	for _, tx := range shardBody.Transactions {
		res = append(res, tx.Hash().GetBytes()...)
	}
	return common.HashH(res)
}

/*
- Concatenate all transaction in one shard as a string
- Then each shard producer a string value include all transactions within this block
- For each string value: Convert string value to hash value
- So if we have 256 shard, we will have 256 leaf value for merkle tree
- Make merkle root from these value
*/

func (shardBody ShardBody) CalcMerkleRootTx() *common.Hash {
	merkleRoots := Merkle{}.BuildMerkleTreeStore(shardBody.Transactions)
	merkleRoot := merkleRoots[len(merkleRoots)-1]
	return merkleRoot
}

func (shardBody ShardBody) ExtractIncomingCrossShardMap() (map[byte][]common.Hash, error) {
	crossShardMap := make(map[byte][]common.Hash)
	for shardID, crossblocks := range shardBody.CrossTransactions {
		for _, crossblock := range crossblocks {
			crossShardMap[shardID] = append(crossShardMap[shardID], crossblock.BlockHash)
		}
	}
	return crossShardMap, nil
}

func (shardBody ShardBody) ExtractOutgoingCrossShardMap() (map[byte][]common.Hash, error) {
	crossShardMap := make(map[byte][]common.Hash)
	return crossShardMap, nil
}
