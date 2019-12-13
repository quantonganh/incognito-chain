package blockchain

import (
	"encoding/binary"
	"errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction"
)

type CrossOutputCoin struct {
	BlockHeight         uint64
	BlockHash           common.Hash
	OutputCoinWithIndex []CrossOutputCoinWithIndex
}
type CrossTxTokenData struct {
	BlockHeight uint64
	BlockHash   common.Hash
	TxTokenData []transaction.TxNormalTokenData
}
type CrossTokenPrivacyData struct {
	BlockHeight      uint64
	BlockHash        common.Hash
	TokenPrivacyData []ContentCrossShardTokenPrivacyData
}
type CrossTransaction struct {
	BlockHeight         uint64
	BlockHash           common.Hash
	TokenPrivacyData    []ContentCrossShardTokenPrivacyData
	OutputCoinWithIndex []CrossOutputCoinWithIndex
}
type ContentCrossShardTokenPrivacyData struct {
	OutputCoinWithIndex []CrossOutputCoinWithIndex
	PropertyID          common.Hash // = hash of TxCustomTokenprivacy data
	PropertyName        string
	PropertySymbol      string
	Type                int    // action type
	Mintable            bool   // default false
	Amount              uint64 // init amount
}
type CrossShardTokenPrivacyMetaData struct {
	TokenID        common.Hash
	PropertyName   string
	PropertySymbol string
	Type           int    // action type
	Mintable       bool   // default false
	Amount         uint64 // init amount
}

type CrossOutputCoinWithIndex struct {
	OutputCoin      privacy.OutputCoin
	IndexInTx       byte
	EphemeralPubKey []byte
}

func (contentCrossShardTokenPrivacyData ContentCrossShardTokenPrivacyData) Bytes() []byte {
	res := []byte{}
	for _, item := range contentCrossShardTokenPrivacyData.OutputCoinWithIndex {
		res = append(res, item.Bytes()...)
	}
	res = append(res, contentCrossShardTokenPrivacyData.PropertyID.GetBytes()...)
	res = append(res, []byte(contentCrossShardTokenPrivacyData.PropertyName)...)
	res = append(res, []byte(contentCrossShardTokenPrivacyData.PropertySymbol)...)
	typeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(typeBytes, uint32(contentCrossShardTokenPrivacyData.Type))
	res = append(res, typeBytes...)
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint32(amountBytes, uint32(contentCrossShardTokenPrivacyData.Amount))
	res = append(res, amountBytes...)
	if contentCrossShardTokenPrivacyData.Mintable {
		res = append(res, []byte("true")...)
	} else {
		res = append(res, []byte("false")...)
	}
	return res
}
func (contentCrossShardTokenPrivacyData ContentCrossShardTokenPrivacyData) Hash() common.Hash {
	return common.HashH(contentCrossShardTokenPrivacyData.Bytes())
}
func (crossOutputCoin CrossOutputCoin) Hash() common.Hash {
	res := []byte{}
	res = append(res, crossOutputCoin.BlockHash.GetBytes()...)
	for _, coins := range crossOutputCoin.OutputCoinWithIndex {
		res = append(res, coins.Bytes()...)
	}
	return common.HashH(res)
}
func (crossTransaction CrossTransaction) Bytes() []byte {
	res := []byte{}
	res = append(res, crossTransaction.BlockHash.GetBytes()...)
	for _, coins := range crossTransaction.OutputCoinWithIndex {
		res = append(res, coins.Bytes()...)
	}
	for _, coins := range crossTransaction.TokenPrivacyData {
		res = append(res, coins.Bytes()...)
	}
	return res
}
func (crossTransaction CrossTransaction) Hash() common.Hash {
	return common.HashH(crossTransaction.Bytes())
}

/*
	Verify CrossShard Block
	- Agg Signature
	- MerklePath
*/
func (crossShardBlock *CrossShardBlock) VerifyCrossShardBlock(blockchain *BlockChain, committees []incognitokey.CommitteePublicKey) error {
	if err := blockchain.config.ConsensusEngine.ValidateBlockCommitteSig(crossShardBlock, committees, crossShardBlock.Header.ConsensusType); err != nil {
		return NewBlockChainError(SignatureError, err)
	}
	if ok := VerifyCrossShardBlockUTXO(crossShardBlock, crossShardBlock.MerklePathShard); !ok {
		return NewBlockChainError(HashError, errors.New("Fail to verify Merkle Path Shard"))
	}
	return nil
}

func (outCoinWithIndex CrossOutputCoinWithIndex) Bytes() []byte {
	res := []byte{}
	res = append(outCoinWithIndex.OutputCoin.Bytes())

	if len(outCoinWithIndex.EphemeralPubKey) > 0 {
		res = append(res, outCoinWithIndex.EphemeralPubKey...)
		res = append(res, outCoinWithIndex.IndexInTx)
	}

	return res
}
