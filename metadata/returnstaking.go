package metadata

import (
	"reflect"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/pkg/errors"
)

type ReturnStakingMetadata struct {
	MetadataBase
	TxID          string
	StakerAddress privacy.PaymentAddress
}

func NewReturnStaking(
	txID string,
	producerAddress privacy.PaymentAddress,
	metaType int,
) *ReturnStakingMetadata {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &ReturnStakingMetadata{
		TxID:          txID,
		StakerAddress: producerAddress,
		MetadataBase:  metadataBase,
	}
}

func (sbsRes ReturnStakingMetadata) CheckTransactionFee(tr Transaction, minFee uint64, beaconHeight int64, db database.DatabaseInterface) bool {
	// no need to have fee for this tx
	return true
}

func (sbsRes ReturnStakingMetadata) ValidateTxWithBlockChain(txr Transaction, bcr BlockchainRetriever, shardID byte, db database.DatabaseInterface) (bool, error) {
	stakingTx := bcr.GetStakingTx(shardID)
	for key, value := range stakingTx {
		committeePublicKey := incognitokey.CommitteePublicKey{}
		err := committeePublicKey.FromString(key)
		if err != nil {
			return false, err
		}
		if reflect.DeepEqual(sbsRes.StakerAddress.Pk, committeePublicKey.IncPubKey) && (sbsRes.TxID == value) {
			autoStakingList := bcr.GetAutoStakingList()
			if autoStakingList[key] {
				return false, errors.New("Can not return staking amount for candidate, who want to restaking.")
			}
			return true, nil
		}
	}
	return false, errors.New("Can not find any staking information of this publickey")
}

func (sbsRes ReturnStakingMetadata) ValidateSanityData(bcr BlockchainRetriever, txr Transaction) (bool, bool, error) {
	if len(sbsRes.StakerAddress.Pk) == 0 {
		return false, false, errors.New("Wrong request info's producer address")
	}
	if len(sbsRes.StakerAddress.Tk) == 0 {
		return false, false, errors.New("Wrong request info's producer address")
	}
	if sbsRes.TxID == "" {
		return false, false, errors.New("Wrong request info's Tx staking")
	}
	return false, true, nil
}

func (sbsRes ReturnStakingMetadata) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return true
}

func (sbsRes ReturnStakingMetadata) Hash() *common.Hash {
	record := sbsRes.StakerAddress.String()
	record += sbsRes.TxID

	// final hash
	record += sbsRes.MetadataBase.Hash().String()
	hash := common.HashH([]byte(record))
	return &hash
}

//validate in shard block
// func (sbsRes ReturnStakingMetadata) VerifyMinerCreatedTxBeforeGettingInBlock(
// 	txsInBlock []Transaction,
// 	txsUsed []int,
// 	insts [][]string,
// 	instUsed []int,
// 	shardID byte,
// 	tx Transaction,
// 	bcr BlockchainRetriever,
// 	accumulatedValues *AccumulatedValues,
// ) (bool, error) {

// 	if len(insts) == 0 {
// 		return false, errors.Errorf("no instruction found for BeaconBlockSalaryResponse tx %s", tx.Hash().String())
// 	}

// 	// check if tx staking is existed
// 	txValue, err := bcr.GetTxValue(sbsRes.TxID)
// 	if err != nil {
// 		return false, err // not exist
// 	}
// 	if txValue != tx.CalculateTxValue() {
// 		return false, errors.New("Not return correct amount")
// 	}

// 	// check if swaper is in this shard
// 	sa := sbsRes.StakerAddress
// 	spa := base58.Base58Check{}.Encode(sa.Pk[:], 0x00)

// 	txShardID, err := bcr.GetShardIDFromTx(sbsRes.TxID)
// 	if err != nil {
// 		return false, err // not exist
// 	}

// 	if common.GetShardIDFromLastByte(sa.Pk[len(sa.Pk)-1]) != txShardID {
// 		return false, errors.New(fmt.Sprint("SA: Not for this shard ", txShardID, common.GetShardIDFromLastByte(sa.Pk[len(sa.Pk)-1])))
// 	}

// 	// check if return public address is swaper
// 	inSwapper := false
// 	for i, inst := range insts {
// 		if instUsed[i] == 0 { // not used before
// 			if inst[0] == "swap" { // is swap action
// 				if strings.Contains(inst[2], spa) { // in swaper list
// 					inSwapper = true
// 					instUsed[i] += 1
// 					break
// 				}
// 			}
// 		}
// 	}

// 	if !inSwapper {
// 		Logger.log.Debug("SA: swaper public address", spa, insts[2])
// 		return false, errors.Errorf("Public address is not in swap instruction")
// 	}

// 	return true, nil
// }
