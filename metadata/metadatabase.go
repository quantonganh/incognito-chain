package metadata

import (
	"fmt"
	"math"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database"
)

type MetadataBase struct {
	Type int
}

func NewMetadataBase(thisType int) *MetadataBase {
	return &MetadataBase{Type: thisType}
}

func (mb MetadataBase) IsMinerCreatedMetaType() bool {
	metaType := mb.GetType()
	for _, mType := range minerCreatedMetaTypes {
		if metaType == mType {
			return true
		}
	}
	return false
}

func (mb *MetadataBase) CalculateSize() uint64 {
	return 0
}

func (mb *MetadataBase) Validate() error {
	return nil
}

func (mb *MetadataBase) Process() error {
	return nil
}

func (mb MetadataBase) GetType() int {
	return mb.Type
}

func (mb MetadataBase) Hash() *common.Hash {
	record := strconv.Itoa(mb.Type)
	hash := common.HashH([]byte(record))
	return &hash
}

func (mb MetadataBase) CheckTransactionFee(
	tx Transaction,
	minFeePerKbTx uint64,
	beaconHeight int64,
	db database.DatabaseInterface,
) bool {
	if tx.GetType() == common.TxCustomTokenPrivacyType {
		feeNativeToken := tx.GetTxFee()
		feePToken := tx.GetTxFeeToken()
		if feePToken > 0 {
			tokenID := tx.GetTokenID()
			feePTokenToNativeTokenTmp, err := ConvertPrivacyTokenToNativeToken(feePToken, tx.GetTokenID(), beaconHeight, db)
			if err != nil {
				fmt.Printf("transaction %+v: %+v %v can not convert to native token",
					tx.Hash().String(), feePToken, tokenID)
				return false
			}
			feePTokenToNativeToken := uint64(math.Ceil(feePTokenToNativeTokenTmp))
			feeNativeToken += feePTokenToNativeToken
		}
		// get limit fee in native token
		actualTxSize := tx.GetTxActualSize()
		// check fee in native token
		minFee := actualTxSize * minFeePerKbTx
		if feeNativeToken < minFee {
			fmt.Printf("transaction %+v has %d fees PRV which is under the required amount of %d, tx size %d",
				tx.Hash().String(), feeNativeToken, minFee, actualTxSize)
			return false
		}
		return true
	}
	// normal privacy tx
	txFee := tx.GetTxFee()
	fullFee := minFeePerKbTx * tx.GetTxActualSize()
	return !(txFee < fullFee)
}

func (mb *MetadataBase) BuildReqActions(tx Transaction, bcr BlockchainRetriever, shardID byte) ([][]string, error) {
	return [][]string{}, nil
}

func (mb MetadataBase) VerifyMinerCreatedTxBeforeGettingInBlock(
	txsInBlock []Transaction,
	txsUsed []int,
	insts [][]string,
	instsUsed []int,
	shardID byte,
	txr Transaction,
	bcr BlockchainRetriever,
	accumulatedValues *AccumulatedValues,
) (bool, error) {
	return true, nil
}
