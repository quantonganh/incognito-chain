package blockchain

import (
	"errors"
	"sort"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge"
	"github.com/incognitochain/incognito-chain/transaction"
)

// TxViewPoint is used to contain data which is fetched from tx of every block
type TxViewPoint struct {
	tokenID           *common.Hash
	shardID           byte
	listSerialNumbers [][]byte // array serialNumbers

	// FOR PRV
	// use to fetch snDerivator
	mapSnD map[string][][]byte
	// use to fetch commitment
	mapCommitments map[string][][]byte //map[base58check.encode{pubkey}]([]([]byte-commitment))
	// use to fetch output coin
	mapOutputCoins map[string][]privacy.OutputCoin

	// data of NORMAL custom token
	customTokenTxs map[int32]*transaction.TxNormalToken

	// data of PRIVACY custom token
	privacyCustomTokenViewPoint map[int32]*TxViewPoint // sub tx viewpoint for token
	privacyCustomTokenTxs       map[int32]*transaction.TxCustomTokenPrivacy
	privacyCustomTokenMetadata  *CrossShardTokenPrivacyMetaData

	//cross shard tx token
	crossTxTokenData map[int32]*transaction.TxNormalTokenData

	// use to fetch tx - pubkey
	txByPubKey map[string]interface{} // map[base58check.encode{pubkey}+"_"+base58check.encode{txid})

	blockHeight uint64
	txVersion int8
	indexOutCoinInTx map[string][]byte
	ephemeralPubKey map[string][]byte

}

/*
ListSerialNumbers returns list serialNumber which is contained in TxViewPoint
*/
// #1: joinSplitDescType is "Coin" Or "Bond" or other token
func (view *TxViewPoint) ListSerialNumbers() [][]byte {
	return view.listSerialNumbers
}

// func (view *TxViewPoint) ListSnDerivators() []big.Int {
// 	return view.listSnD
// }
func (view *TxViewPoint) MapSnDerivators() map[string][][]byte {
	return view.mapSnD
}

func (view *TxViewPoint) ListSerialNumnbersEclipsePoint() []*privacy.Point {
	result := []*privacy.Point{}
	for _, commitment := range view.listSerialNumbers {
		point := &privacy.Point{}
		point.FromBytesS(commitment)
		result = append(result, point)
	}
	return result
}

// fetch from desc of tx to get serialNumber and commitments
// (note: still storage full data of commitments, serialnumbers, snderivator to check double spend)
func (view *TxViewPoint) processFetchTxViewPoint(
	shardID byte,
	db database.DatabaseInterface,
	proof *zkp.PaymentProof,
	tokenID *common.Hash,
) ([][]byte, map[string][][]byte, map[string][]privacy.OutputCoin, map[string][]privacy.Scalar, map[string][]byte, map[string][]byte, error) {
	acceptedSerialNumbers := make([][]byte, 0)
	acceptedCommitments := make(map[string][][]byte)
	acceptedOutputcoins := make(map[string][]privacy.OutputCoin)
	acceptedSnD := make(map[string][]privacy.Scalar)
	indexOutputInTx := make(map[string][]byte)
	acceptedEphemeralPubKey := make(map[string][]byte)

	if proof == nil {
		return acceptedSerialNumbers, acceptedCommitments, acceptedOutputcoins, acceptedSnD, indexOutputInTx, acceptedEphemeralPubKey, nil
	}
	// Get data for serialnumbers
	// Process input of transaction
	// Get Serial numbers of input
	// Append into accepttedSerialNumbers if this serial number haven't exist yet
	for _, item := range proof.GetInputCoins() {
		serialNum := item.CoinDetails.GetSerialNumber().ToBytesS()
		ok, err := db.HasSerialNumber(*tokenID, serialNum, shardID)
		if err != nil {
			return acceptedSerialNumbers, acceptedCommitments, acceptedOutputcoins, acceptedSnD, indexOutputInTx, acceptedEphemeralPubKey, err
		}
		if !ok {
			acceptedSerialNumbers = append(acceptedSerialNumbers, serialNum)
		}
	}

	// get data for ephemeral public key
	// ephemeralPubKey must not exist before in db
	ephemeralPubKey := []byte{}
	if proof.GetEphemeralPubKey() != nil && !proof.GetEphemeralPubKey().IsIdentity() {
		ephemeralPubKeyTmp :=  proof.GetEphemeralPubKey().ToBytesS()
		ok, err := db.HasEphemeralPubKey(*tokenID, ephemeralPubKeyTmp)
		if err != nil {
			return acceptedSerialNumbers, acceptedCommitments, acceptedOutputcoins, acceptedSnD, indexOutputInTx, acceptedEphemeralPubKey, err
		}
		if !ok {
			ephemeralPubKey = ephemeralPubKeyTmp
		}
	}

	// Process Output Coins (just created UTXO of this transaction)
	// Proccessed variable: commitment, snd, outputcoins
	// Commitment and SND must not exist before in db
	// Outputcoins will be stored as new utxo for next transaction

	if len(ephemeralPubKey) > 0 {
		// tx version 2 has privacy
		// Proccessed variable: commitment, outputcoins, indexOutInTx, ephemeralPubKey
		for i, item := range proof.GetOutputCoins() {
			commitment := item.CoinDetails.GetCoinCommitment().ToBytesS()
			pubkey := item.CoinDetails.GetPublicKey().ToBytesS()
			pubkeyStr := base58.Base58Check{}.Encode(pubkey, common.ZeroByte)
			ok, err := db.HasCommitment(*tokenID, commitment, shardID)
			if err != nil {
				return acceptedSerialNumbers, acceptedCommitments, acceptedOutputcoins, acceptedSnD, indexOutputInTx, acceptedEphemeralPubKey, err
			}
			if !ok {
				if acceptedCommitments[pubkeyStr] == nil {
					acceptedCommitments[pubkeyStr] = make([][]byte, 0)
				}
				// get data for commitments
				acceptedCommitments[pubkeyStr] = append(acceptedCommitments[pubkeyStr], item.CoinDetails.GetCoinCommitment().ToBytesS())

				// get data for output coin
				if acceptedOutputcoins[pubkeyStr] == nil {
					acceptedOutputcoins[pubkeyStr] = make([]privacy.OutputCoin, 0)
				}
				acceptedOutputcoins[pubkeyStr] = append(acceptedOutputcoins[pubkeyStr], *item)

				// get index output coin in tx
				if indexOutputInTx[pubkeyStr] == nil {
					indexOutputInTx[pubkeyStr] = make([]byte, 0)
				}
				indexOutputInTx[pubkeyStr] = append(indexOutputInTx[pubkeyStr], byte(i))
			}

			// get ephemeral public key
			acceptedEphemeralPubKey[pubkeyStr] = ephemeralPubKey
		}
	} else {
		// tx version 1 and tx version 2 no privacy
		// Proccessed variable: commitment, outputcoins, snds
		for _, item := range proof.GetOutputCoins() {
			commitment := item.CoinDetails.GetCoinCommitment().ToBytesS()
			pubkey := item.CoinDetails.GetPublicKey().ToBytesS()
			pubkeyStr := base58.Base58Check{}.Encode(pubkey, common.ZeroByte)
			ok, err := db.HasCommitment(*tokenID, commitment, shardID)
			if err != nil {
				return acceptedSerialNumbers, acceptedCommitments, acceptedOutputcoins, acceptedSnD, indexOutputInTx, acceptedEphemeralPubKey, err
			}
			if !ok {
				if acceptedCommitments[pubkeyStr] == nil {
					acceptedCommitments[pubkeyStr] = make([][]byte, 0)
				}
				// get data for commitments
				acceptedCommitments[pubkeyStr] = append(acceptedCommitments[pubkeyStr], item.CoinDetails.GetCoinCommitment().ToBytesS())

				// get data for output coin
				if acceptedOutputcoins[pubkeyStr] == nil {
					acceptedOutputcoins[pubkeyStr] = make([]privacy.OutputCoin, 0)
				}
				acceptedOutputcoins[pubkeyStr] = append(acceptedOutputcoins[pubkeyStr], *item)
			}

			// get data for Snderivators
			snD := item.CoinDetails.GetSNDerivatorRandom()
			if snD != nil && !snD.IsZero() {
				ok, err = db.HasSNDerivator(*tokenID, snD.ToBytesS())
				if err != nil {
					return acceptedSerialNumbers, acceptedCommitments, acceptedOutputcoins, acceptedSnD, indexOutputInTx, acceptedEphemeralPubKey, err
				}
				if !ok {
					acceptedSnD[pubkeyStr] = append(acceptedSnD[pubkeyStr], *snD)
				}
			} else {
				return acceptedSerialNumbers, acceptedCommitments, acceptedOutputcoins, acceptedSnD, indexOutputInTx, acceptedEphemeralPubKey,
					errors.New("snd in input coins is invalid")
			}
		}
	}





	return acceptedSerialNumbers, acceptedCommitments, acceptedOutputcoins, acceptedSnD, indexOutputInTx, acceptedEphemeralPubKey, nil
}

/*
fetchTxViewPointFromBlock get list serialnumber and commitments, output coins from txs in block and check if they are not in Main chain db
return a tx view point which contains list new serialNumbers and new commitments from block
// (note: still storage full data of commitments, serialnumbers, snderivator to check double spend)
*/

func (view *TxViewPoint) fetchTxViewPointFromBlock(db database.DatabaseInterface, block *ShardBlock) error {
	transactions := block.Body.Transactions
	// Loop through all of the transaction descs (except for the salary tx)
	acceptedSerialNumbers := make([][]byte, 0)
	acceptedCommitments := make(map[string][][]byte)
	acceptedOutputcoins := make(map[string][]privacy.OutputCoin)
	acceptedSnD := make(map[string][][]byte)
	acceptedIndexOutInTx := make(map[string][]byte)
	acceptedEphemeralPubKey := make(map[string][]byte)

	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])

	view.blockHeight = block.GetHeight()

	for indexTx, tx := range transactions {
		switch tx.GetType() {
		case common.TxNormalType, common.TxRewardType, common.TxReturnStakingType:
			{
				normalTx := tx.(*transaction.Tx)
				serialNumbers, commitments, outCoins, snDs, indexOutInTx, ephemeralPubKey, err := view.processFetchTxViewPoint(block.Header.ShardID, db, normalTx.Proof, prvCoinID)
				if err != nil {
					return NewBlockChainError(UnExpectedError, err)
				}
				acceptedSerialNumbers = append(acceptedSerialNumbers, serialNumbers...)
				for pubkey, data := range commitments {
					if acceptedCommitments[pubkey] == nil {
						acceptedCommitments[pubkey] = make([][]byte, 0)
					}
					acceptedCommitments[pubkey] = append(acceptedCommitments[pubkey], data...)
					view.txByPubKey[pubkey+"_"+base58.Base58Check{}.Encode(tx.Hash().GetBytes(), 0x0)+"_"+strconv.Itoa(int(block.Header.ShardID))] = true
				}
				for pubkey, data := range outCoins {
					if acceptedOutputcoins[pubkey] == nil {
						acceptedOutputcoins[pubkey] = make([]privacy.OutputCoin, 0)
					}
					acceptedOutputcoins[pubkey] = append(acceptedOutputcoins[pubkey], data...)
				}
				for pubkey, data := range snDs {
					if acceptedSnD[pubkey] == nil {
						acceptedSnD[pubkey] = make([][]byte, 0)
					}
					for _, snd := range data {
						acceptedSnD[pubkey] = append(acceptedSnD[pubkey], snd.ToBytesS())
					}
				}
				for pubkey, data := range indexOutInTx {
					if acceptedIndexOutInTx[pubkey] == nil {
						acceptedIndexOutInTx[pubkey] = make([]byte, 0)
					}
					for _, index := range data {
						acceptedIndexOutInTx[pubkey] = append(acceptedIndexOutInTx[pubkey], index)
					}
				}
				for pubkey, data := range ephemeralPubKey {
					acceptedEphemeralPubKey[pubkey] = data
				}
			}
		case common.TxCustomTokenType:
			//{
			//	tx := tx.(*transaction.TxNormalToken)
			//	serialNumbers, commitments, outCoins, snDs, err := view.processFetchTxViewPoint(block.Header.ShardID, db, tx.Proof, prvCoinID)
			//	if err != nil {
			//		return NewBlockChainError(UnExpectedError, err)
			//	}
			//	acceptedSerialNumbers = append(acceptedSerialNumbers, serialNumbers...)
			//	for pubkey, data := range commitments {
			//		if acceptedCommitments[pubkey] == nil {
			//			acceptedCommitments[pubkey] = make([][]byte, 0)
			//		}
			//		acceptedCommitments[pubkey] = append(acceptedCommitments[pubkey], data...)
			//		view.txByPubKey[pubkey+"_"+base58.Base58Check{}.Encode(tx.Hash().GetBytes(), 0x0)+"_"+strconv.Itoa(int(block.Header.ShardID))] = true
			//	}
			//	for pubkey, data := range outCoins {
			//		if acceptedOutputcoins[pubkey] == nil {
			//			acceptedOutputcoins[pubkey] = make([]privacy.OutputCoin, 0)
			//		}
			//		acceptedOutputcoins[pubkey] = append(acceptedOutputcoins[pubkey], data...)
			//	}
			//	for pubkey, data := range snDs {
			//		if snDs[pubkey] == nil {
			//			snDs[pubkey] = make([]privacy.Scalar, 0)
			//		}
			//		snDs[pubkey] = append(snDs[pubkey], data...)
			//	}
			//	// acceptedSnD = append(acceptedSnD, snDs...)
			//
			//	// indexTx is index of transaction in block
			//	view.customTokenTxs[int32(indexTx)] = tx
			//}
		case common.TxCustomTokenPrivacyType:
			{
				tx := tx.(*transaction.TxCustomTokenPrivacy)
				serialNumbers, commitments, outCoins, snDs, indexOutInTx, ephemeralPubKey, err := view.processFetchTxViewPoint(block.Header.ShardID, db, tx.Proof, prvCoinID)
				if err != nil {
					return NewBlockChainError(UnExpectedError, err)
				}
				acceptedSerialNumbers = append(acceptedSerialNumbers, serialNumbers...)
				for pubkey, data := range commitments {
					if acceptedCommitments[pubkey] == nil {
						acceptedCommitments[pubkey] = make([][]byte, 0)
					}
					acceptedCommitments[pubkey] = append(acceptedCommitments[pubkey], data...)
					view.txByPubKey[pubkey+"_"+base58.Base58Check{}.Encode(tx.Hash().GetBytes(), 0x0)+"_"+strconv.Itoa(int(block.Header.ShardID))] = true
				}
				for pubkey, data := range outCoins {
					if acceptedOutputcoins[pubkey] == nil {
						acceptedOutputcoins[pubkey] = make([]privacy.OutputCoin, 0)
					}
					acceptedOutputcoins[pubkey] = append(acceptedOutputcoins[pubkey], data...)
				}
				for pubkey, data := range snDs {
					if acceptedSnD[pubkey] == nil {
						acceptedSnD[pubkey] = make([][]byte, 0)
					}
					for _, snd := range data {
						acceptedSnD[pubkey] = append(acceptedSnD[pubkey], snd.ToBytesS())
					}
				}

				for pubkey, data := range indexOutInTx {
					if acceptedIndexOutInTx[pubkey] == nil {
						acceptedIndexOutInTx[pubkey] = make([]byte, 0)
					}
					for _, index := range data {
						acceptedIndexOutInTx[pubkey] = append(acceptedIndexOutInTx[pubkey], index)
					}
				}

				for pubkey, data := range ephemeralPubKey {
					acceptedEphemeralPubKey[pubkey] = data
				}

				// sub view for privacy custom token
				subView := NewTxViewPoint(block.Header.ShardID)
				subView.blockHeight = view.blockHeight
				subView.tokenID = &tx.TxPrivacyTokenData.PropertyID
				serialNumbersP, commitmentsP, outCoinsP, snDsP, indexOutInTxP, ephemeralPubKeyP, errP := subView.processFetchTxViewPoint(subView.shardID, db, tx.TxPrivacyTokenData.TxNormal.Proof, subView.tokenID)
				if errP != nil {
					return NewBlockChainError(UnExpectedError, errP)
				}
				subView.listSerialNumbers = serialNumbersP
				for pubkey, data := range commitmentsP {
					if subView.mapCommitments[pubkey] == nil {
						subView.mapCommitments[pubkey] = make([][]byte, 0)
					}
					subView.mapCommitments[pubkey] = append(subView.mapCommitments[pubkey], data...)
					view.txByPubKey[pubkey+"_"+base58.Base58Check{}.Encode(tx.Hash().GetBytes(), 0x0)+"_"+strconv.Itoa(int(block.Header.ShardID))] = true
				}
				for pubkey, data := range outCoinsP {
					if subView.mapOutputCoins[pubkey] == nil {
						subView.mapOutputCoins[pubkey] = make([]privacy.OutputCoin, 0)
					}
					subView.mapOutputCoins[pubkey] = append(subView.mapOutputCoins[pubkey], data...)
				}
				for pubkey, data := range snDsP {
					if subView.mapSnD[pubkey] == nil {
						subView.mapSnD[pubkey] = make([][]byte, 0)
					}
					for _, b := range data {
						temp := b.ToBytesS()
						subView.mapSnD[pubkey] = append(subView.mapSnD[pubkey], temp)
					}
				}

				for pubkey, data := range indexOutInTxP {
					if subView.indexOutCoinInTx[pubkey] == nil {
						subView.indexOutCoinInTx[pubkey] = make([]byte, 0)
					}
					subView.indexOutCoinInTx[pubkey] = append(subView.indexOutCoinInTx[pubkey], data...)
				}

				for pubkey, data := range ephemeralPubKeyP {
					subView.ephemeralPubKey[pubkey] = data
				}

				view.privacyCustomTokenViewPoint[int32(indexTx)] = subView
				view.privacyCustomTokenTxs[int32(indexTx)] = tx
			}
		default:
			{
				return NewBlockChainError(UnExpectedError, errors.New("TxNormal type is invalid"))
			}
		}
	}

	if len(acceptedSerialNumbers) > 0 {
		view.listSerialNumbers = acceptedSerialNumbers
	}
	if len(acceptedCommitments) > 0 {
		view.mapCommitments = acceptedCommitments
	}
	if len(acceptedOutputcoins) > 0 {
		view.mapOutputCoins = acceptedOutputcoins
	}
	if len(acceptedSnD) > 0 {
		view.mapSnD = acceptedSnD
	}
	if len(acceptedIndexOutInTx) > 0 {
		view.indexOutCoinInTx = acceptedIndexOutInTx
	}
	if len(acceptedEphemeralPubKey) > 0 {
		view.ephemeralPubKey = acceptedEphemeralPubKey
	}
	return nil
}

/*
Create a TxNormal view point, which contains data about serialNumbers and commitments
*/
func NewTxViewPoint(shardID byte) *TxViewPoint {
	result := &TxViewPoint{
		shardID:                     shardID,
		listSerialNumbers:           make([][]byte, 0),
		mapCommitments:              make(map[string][][]byte),
		mapOutputCoins:              make(map[string][]privacy.OutputCoin),
		mapSnD:                      make(map[string][][]byte),
		customTokenTxs:              make(map[int32]*transaction.TxNormalToken),
		tokenID:                     &common.Hash{},
		privacyCustomTokenViewPoint: make(map[int32]*TxViewPoint),
		privacyCustomTokenTxs:       make(map[int32]*transaction.TxCustomTokenPrivacy),
		privacyCustomTokenMetadata:  &CrossShardTokenPrivacyMetaData{},
		crossTxTokenData:            make(map[int32]*transaction.TxNormalTokenData),

		txByPubKey: make(map[string]interface{}),

		indexOutCoinInTx:  make(map[string][]byte),
		ephemeralPubKey:  make(map[string][]byte),
		blockHeight: uint64(0),
	}
	result.tokenID.SetBytes(common.PRVCoinID[:])
	return result
}

/*
	fetch information from cross output coin
	- UTXO: outcoin
	- Commitment
	- snd
*/
func (view *TxViewPoint) processFetchCrossOutputViewPoint(
	shardID byte,
	db database.DatabaseInterface,
	outputCoins []privacy.OutputCoin,
	tokenID *common.Hash,
) (map[string][][]byte, map[string][]privacy.OutputCoin, map[string][]privacy.Scalar, error) {
	acceptedCommitments := make(map[string][][]byte)
	acceptedOutputcoins := make(map[string][]privacy.OutputCoin)
	acceptedSnD := make(map[string][]privacy.Scalar)
	if len(outputCoins) == 0 {
		return acceptedCommitments, acceptedOutputcoins, acceptedSnD, nil
	}

	// Process Output Coins (just created UTXO of this transaction)
	// Proccessed variable: commitment, snd, outputcoins
	// Commitment and SND must not exist before in db
	// Outputcoins will be stored as new utxo for next transaction
	for _, outputCoin := range outputCoins {
		item := &outputCoin
		commitment := item.CoinDetails.GetCoinCommitment().ToBytesS()
		pubkey := item.CoinDetails.GetPublicKey().ToBytesS()
		pubkeyStr := base58.Base58Check{}.Encode(pubkey, common.ZeroByte)
		ok, err := db.HasCommitment(*tokenID, commitment, shardID)
		if err != nil {
			return acceptedCommitments, acceptedOutputcoins, acceptedSnD, err
		}
		if !ok {
			pubkeyStr := base58.Base58Check{}.Encode(pubkey, common.ZeroByte)
			if acceptedCommitments[pubkeyStr] == nil {
				acceptedCommitments[pubkeyStr] = make([][]byte, 0)
			}
			// get data for commitments
			acceptedCommitments[pubkeyStr] = append(acceptedCommitments[pubkeyStr], item.CoinDetails.GetCoinCommitment().ToBytesS())

			// get data for output coin
			if acceptedOutputcoins[pubkeyStr] == nil {
				acceptedOutputcoins[pubkeyStr] = make([]privacy.OutputCoin, 0)
			}
			acceptedOutputcoins[pubkeyStr] = append(acceptedOutputcoins[pubkeyStr], *item)
		}

		// get data for Snderivators
		snD := item.CoinDetails.GetSNDerivatorRandom()
		ok, err = db.HasSNDerivator(*tokenID, snD.ToBytesS())
		if !ok && err == nil {
			acceptedSnD[pubkeyStr] = append(acceptedSnD[pubkeyStr], *snD)
		}
	}
	return acceptedCommitments, acceptedOutputcoins, acceptedSnD, nil
}


func (view *TxViewPoint) fetchCrossTransactionViewPointFromBlock(db database.DatabaseInterface, block *ShardBlock) error {
	allShardCrossTransactions := block.Body.CrossTransactions
	// Loop through all of the transaction descs (except for the salary tx)
	acceptedOutputcoins := make(map[string][]privacy.OutputCoin)
	acceptedCommitments := make(map[string][][]byte)
	acceptedSnD := make(map[string][][]byte)
	prvCoinID := &common.Hash{}
	prvCoinID.SetBytes(common.PRVCoinID[:])
	//@NOTICE: this function just work for Normal Transaction

	// sort by shard ID
	shardIDs := []int{}
	for k := range allShardCrossTransactions {
		shardIDs = append(shardIDs, int(k))
	}
	sort.Ints(shardIDs)

	indexOut := 0
	for _, shardID := range shardIDs {
		crossTransactions := allShardCrossTransactions[byte(shardID)]
		for _, crossTransaction := range crossTransactions {
			commitments, outCoins, snDs, err := view.processFetchCrossOutputViewPoint(block.Header.ShardID, db, crossTransaction.OutputCoin,  prvCoinID)
			if err != nil {
				return NewBlockChainError(UnExpectedError, err)
			}
			for pubkey, data := range commitments {
				if acceptedCommitments[pubkey] == nil {
					acceptedCommitments[pubkey] = make([][]byte, 0)
				}
				acceptedCommitments[pubkey] = append(acceptedCommitments[pubkey], data...)
			}
			for pubkey, data := range outCoins {
				if acceptedOutputcoins[pubkey] == nil {
					acceptedOutputcoins[pubkey] = make([]privacy.OutputCoin, 0)
				}
				acceptedOutputcoins[pubkey] = append(acceptedOutputcoins[pubkey], data...)
			}
			for pubkey, data := range snDs {
				if acceptedSnD[pubkey] == nil {
					acceptedSnD[pubkey] = make([][]byte, 0)
				}
				for _, snd := range data {
					acceptedSnD[pubkey] = append(acceptedSnD[pubkey], snd.ToBytesS())
				}
			}
			if crossTransaction.TokenPrivacyData != nil && len(crossTransaction.TokenPrivacyData) > 0 {
				for _, tokenPrivacyData := range crossTransaction.TokenPrivacyData {
					subView := NewTxViewPoint(block.Header.ShardID)
					temp, err := common.Hash{}.NewHash(tokenPrivacyData.PropertyID.GetBytes())
					if err != nil {
						return err
					}
					subView.tokenID = temp
					subView.privacyCustomTokenMetadata.TokenID = *temp
					subView.privacyCustomTokenMetadata.PropertyName = tokenPrivacyData.PropertyName
					subView.privacyCustomTokenMetadata.PropertySymbol = tokenPrivacyData.PropertySymbol
					subView.privacyCustomTokenMetadata.Amount = tokenPrivacyData.Amount
					subView.privacyCustomTokenMetadata.Mintable = tokenPrivacyData.Mintable
					commitmentsP, outCoinsP, snDsP, err := subView.processFetchCrossOutputViewPoint(block.Header.ShardID, db, tokenPrivacyData.OutputCoin, subView.tokenID)
					if err != nil {
						return NewBlockChainError(UnExpectedError, err)
					}
					for pubkey, data := range commitmentsP {
						if subView.mapCommitments[pubkey] == nil {
							subView.mapCommitments[pubkey] = make([][]byte, 0)
						}
						subView.mapCommitments[pubkey] = append(subView.mapCommitments[pubkey], data...)
					}
					for pubkey, data := range outCoinsP {
						if subView.mapOutputCoins[pubkey] == nil {
							subView.mapOutputCoins[pubkey] = make([]privacy.OutputCoin, 0)
						}
						subView.mapOutputCoins[pubkey] = append(subView.mapOutputCoins[pubkey], data...)
					}
					for pubkey, data := range snDsP {
						if subView.mapSnD[pubkey] == nil {
							subView.mapSnD[pubkey] = make([][]byte, 0)
						}
						for _, t := range data {
							temp := t.ToBytesS()
							subView.mapSnD[pubkey] = append(subView.mapSnD[pubkey], temp)
						}
					}
					view.privacyCustomTokenViewPoint[int32(indexOut)] = subView
					indexOut++
				}
			}
		}
	}

	if len(acceptedCommitments) > 0 {
		view.mapCommitments = acceptedCommitments
	}
	if len(acceptedOutputcoins) > 0 {
		view.mapOutputCoins = acceptedOutputcoins
	}
	if len(acceptedSnD) > 0 {
		view.mapSnD = acceptedSnD
	}
	return nil
}
