package lvdb

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// StoreSerialNumbers - store list serialNumbers by shardID
func (db *db) StoreSerialNumbers(tokenID common.Hash, serialNumbers [][]byte, shardID byte) error {
	key := addPrefixToKeyHash(string(serialNumbersPrefix), tokenID)
	key = append(key, shardID)

	var lenData int64
	lenSerialNumber, err := db.GetSerialNumbersLength(tokenID, shardID)
	if err != nil && lenSerialNumber == nil {
		return database.NewDatabaseError(database.StoreSerialNumbersError, err)
	}
	if lenSerialNumber == nil {
		lenData = 0
	} else {
		lenData = lenSerialNumber.Int64()
	}
	for _, s := range serialNumbers {
		newIndex := big.NewInt(lenData).Bytes()
		if lenData == 0 {
			newIndex = []byte{0}
		}
		// keySpec1 store serialNumber and index
		keySpec1 := append(key, s...)
		if err := db.Put(keySpec1, newIndex); err != nil {
			return database.NewDatabaseError(database.StoreSerialNumbersError, err)
		}
		// keyStoreLen store last index of array serialNumber
		keyStoreLen := append(key, []byte("len")...)
		if err := db.Put(keyStoreLen, newIndex); err != nil {
			return database.NewDatabaseError(database.StoreSerialNumbersError, err)
		}
		lenData++
	}
	return nil
}

// HasSerialNumber - Check serialNumber in list SerialNumbers by shardID
func (db *db) HasSerialNumber(tokenID common.Hash, serialNumber []byte, shardID byte) (bool, error) {
	key := addPrefixToKeyHash(string(serialNumbersPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, serialNumber...)
	hasValue, err := db.HasValue(keySpec)
	if err != nil {
		return false, database.NewDatabaseError(database.HasSerialNumberError, err, serialNumber, shardID, tokenID)
	} else {
		return hasValue, nil
	}
}

// ListSerialNumber -  return all serial number and its index
func (db *db) ListSerialNumber(tokenID common.Hash, shardID byte) (map[string]uint64, error) {
	result := make(map[string]uint64)
	key := addPrefixToKeyHash(string(serialNumbersPrefix), tokenID)
	key = append(key, shardID)

	iterator := db.lvdb.NewIterator(util.BytesPrefix(key), nil)
	for iterator.Next() {
		key1 := make([]byte, len(iterator.Key()))
		copy(key1, iterator.Key())
		if string(key1[len(key1)-3:]) == "len" {
			continue
		}
		serialNumberInByte := key1[len(key1)-privacy.Ed25519KeySize:]
		value := make([]byte, len(iterator.Value()))
		copy(value, iterator.Value())
		index := big.Int{}
		index.SetBytes(value)
		serialNumber := base58.Base58Check{}.Encode(serialNumberInByte, 0x0)
		result[serialNumber] = index.Uint64()
	}
	return result, nil
}

// GetCommitmentIndex - return index of commitment in db list
func (db *db) GetSerialNumbersLength(tokenID common.Hash, shardID byte) (*big.Int, error) {
	key := addPrefixToKeyHash(string(serialNumbersPrefix), tokenID)
	key = append(key, shardID)
	keyStoreLen := append(key, []byte("len")...)
	hasValue, err := db.HasValue(keyStoreLen)
	if err != nil {
		return nil, database.NewDatabaseError(database.GetSerialNumbersLengthError, err)
	} else {
		if !hasValue {
			return nil, nil
		} else {
			data, err := db.Get(keyStoreLen)
			if err != nil {
				return new(big.Int).SetInt64(0), nil
			} else {
				lenArray := new(big.Int).SetBytes(data)
				lenArray = lenArray.Add(lenArray, new(big.Int).SetInt64(1))
				return lenArray, nil
			}
		}
	}
}

// CleanSerialNumbers - clear all list serialNumber in DB
func (db *db) CleanSerialNumbers() error {
	iter := db.lvdb.NewIterator(util.BytesPrefix(serialNumbersPrefix), nil)
	for iter.Next() {
		err := db.Delete(iter.Key())
		if err != nil {
			return database.NewDatabaseError(database.CleanSerialNumbersError, err)
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return database.NewDatabaseError(database.CleanSerialNumbersError, err)
	}
	return nil
}

//StoreOutputCoins - store all output coin of pubkey
// key: [outcoinsPrefix][tokenID][shardID][hash(output)]
// value: output in bytes
func (db *db) StoreOutputCoins(tokenID common.Hash, publicKey []byte, outputCoinArr [][]byte, shardID byte) error {
	key := addPrefixToKeyHash(string(outcoinsPrefix), tokenID)
	key = append(key, shardID)

	key = append(key, publicKey...)
	batchData := []database.BatchData{}
	for _, outputCoin := range outputCoinArr {
		keyTemp := make([]byte, len(key))
		copy(keyTemp, key)
		keyTemp = append(keyTemp, common.HashB(outputCoin)...)
		// Put to batch
		batchData = append(batchData, database.BatchData{
			Key:   keyTemp,
			Value: outputCoin,
		})
	}
	if len(batchData) > 0 {
		err := db.PutBatch(batchData)
		if err != nil {
			return database.NewDatabaseError(database.StoreOutputCoinsError, err)
		}
	}

	return nil
}

// StoreCommitments - store list commitments by shardID
func (db *db) StoreCommitments(tokenID common.Hash, pubkey []byte, commitments [][]byte, shardID byte) error {
	key := addPrefixToKeyHash(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)

	// keySpec3 store last index of array commitment
	keySpec3 := make([]byte, len(key)+len("len"))
	temp := append(key, []byte("len")...)
	copy(keySpec3, temp)

	var lenData uint64
	lenCommitment, err := db.GetCommitmentLength(tokenID, shardID)
	if err != nil && lenCommitment == nil {
		return database.NewDatabaseError(database.StoreCommitmentsError, err)
	}
	if lenCommitment == nil {
		lenData = 0
	} else {
		lenData = lenCommitment.Uint64()
	}
	for _, c := range commitments {

		newIndex := new(big.Int).SetUint64(lenData).Bytes()
		if lenData == 0 {
			newIndex = []byte{0}
		}
		// keySpec1 use for create proof random
		keySpec1 := append(key, newIndex...)
		if err := db.Put(keySpec1, c); err != nil {
			return database.NewDatabaseError(database.StoreCommitmentsError, err)
		}
		// keySpec2 use for validate
		keySpec2 := append(key, c...)
		if err := db.Put(keySpec2, newIndex); err != nil {
			return database.NewDatabaseError(database.StoreCommitmentsError, err)
		}

		// len of commitment array
		if err := db.Put(keySpec3, newIndex); err != nil {
			return database.NewDatabaseError(database.StoreCommitmentsError, err)
		}
		lenData++
	}

	return nil
}

// HasCommitment - Check commitment in list commitments by shardID
func (db *db) HasCommitment(tokenID common.Hash, commitment []byte, shardID byte) (bool, error) {
	key := addPrefixToKeyHash(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, commitment...)
	hasValue, err := db.HasValue(keySpec)
	if err != nil {
		return false, database.NewDatabaseError(database.HasCommitmentError, err, commitment, shardID, tokenID.String())
	} else {
		return hasValue, nil
	}
}

// ListCommitment -  return all commitment and its index
func (db *db) ListCommitment(tokenID common.Hash, shardID byte) (map[string]uint64, error) {
	result := make(map[string]uint64)
	key := addPrefixToKeyHash(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)

	iterator := db.lvdb.NewIterator(util.BytesPrefix(key), nil)
	for iterator.Next() {
		key1 := make([]byte, len(iterator.Key()))
		copy(key1, iterator.Key())
		if string(key1[len(key1)-3:]) == "len" {
			continue
		}
		if len(key1) < len(key)+privacy.Ed25519KeySize {
			continue
		}
		commitmentInByte := key1[len(key1)-privacy.Ed25519KeySize:]
		value := make([]byte, len(iterator.Value()))
		copy(value, iterator.Value())
		index := big.Int{}
		index.SetBytes(value)
		commitment := base58.Base58Check{}.Encode(commitmentInByte, 0x0)
		result[commitment] = index.Uint64()
	}
	return result, nil
}

// ListCommitmentIndices -  return all commitment index and its value
func (db *db) ListCommitmentIndices(tokenID common.Hash, shardID byte) (map[uint64]string, error) {
	result := make(map[uint64]string)
	key := addPrefixToKeyHash(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)

	iterator := db.lvdb.NewIterator(util.BytesPrefix(key), nil)
	for iterator.Next() {
		key1 := make([]byte, len(iterator.Key()))
		copy(key1, iterator.Key())
		if string(key1[len(key1)-3:]) == "len" {
			continue
		}

		commitmentInByte := make([]byte, len(iterator.Value()))
		copy(commitmentInByte, iterator.Value())
		if len(commitmentInByte) != privacy.Ed25519KeySize {
			continue
		}
		indexInByte := key1[45:]
		index := big.Int{}
		index.SetBytes(indexInByte)
		commitment := base58.Base58Check{}.Encode(commitmentInByte, 0x0)
		result[index.Uint64()] = commitment
	}
	return result, nil
}

func (db *db) HasCommitmentIndex(tokenID common.Hash, commitmentIndex uint64, shardID byte) (bool, error) {
	key := addPrefixToKeyHash(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	var keySpec []byte
	if commitmentIndex == 0 {
		keySpec = append(key, byte(0))
	} else {
		keySpec = append(key, new(big.Int).SetUint64(commitmentIndex).Bytes()...)
	}
	_, err := db.Get(keySpec)
	if err != nil {
		return false, database.NewDatabaseError(database.HasCommitmentInexError, err, commitmentIndex, shardID, tokenID)
	} else {
		return true, nil
	}
}

func (db *db) GetCommitmentByIndex(tokenID common.Hash, commitmentIndex uint64, shardID byte) ([]byte, error) {
	key := addPrefixToKeyHash(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	var keySpec []byte
	if commitmentIndex == 0 {
		keySpec = append(key, byte(0))
	} else {
		keySpec = append(key, new(big.Int).SetUint64(commitmentIndex).Bytes()...)
	}
	data, err := db.Get(keySpec)
	if err != nil {
		return data, database.NewDatabaseError(database.GetCommitmentByIndexError, err, commitmentIndex, shardID, tokenID)
	} else {
		return data, nil
	}
}

// GetCommitmentIndex - return index of commitment in db list
func (db *db) GetCommitmentIndex(tokenID common.Hash, commitment []byte, shardID byte) (*big.Int, error) {
	key := addPrefixToKeyHash(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, commitment...)
	data, err := db.Get(keySpec)
	if err != nil {
		return nil, database.NewDatabaseError(database.GetCommitmentIndexError, err, commitment, shardID, tokenID)
	} else {
		return new(big.Int).SetBytes(data), nil
	}
}

// GetCommitmentIndex - return index of commitment in db list
func (db *db) GetCommitmentLength(tokenID common.Hash, shardID byte) (*big.Int, error) {
	key := addPrefixToKeyHash(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, []byte("len")...)
	hasValue, err := db.HasValue(keySpec)
	if err != nil {
		return nil, database.NewDatabaseError(database.GetCommitmentLengthError, err)
	} else {
		if !hasValue {
			return nil, nil
		} else {
			data, err := db.Get(keySpec)
			if err != nil {
				return nil, database.NewDatabaseError(database.GetCommitmentLengthError, err)
			} else {
				lenArray := new(big.Int).SetBytes(data)
				lenArray = lenArray.Add(lenArray, new(big.Int).SetInt64(1))
				return lenArray, nil
			}
		}
	}
}

//GetOutcoinsByPubkey - get all output coin of pubkey
// key: [outcoinsPrefix][tokenID][shardID][hash(output)]
// value: output in bytes
func (db *db) GetOutcoinsByPubkey(tokenID common.Hash, pubkey []byte, shardID byte) ([][]byte, error) {
	key := addPrefixToKeyHash(string(outcoinsPrefix), tokenID)
	key = append(key, shardID)

	key = append(key, pubkey...)
	arrDatabyPubkey := make([][]byte, 0)
	iter := db.lvdb.NewIterator(util.BytesPrefix(key), nil)
	if iter.Error() != nil {
		return nil, database.NewDatabaseError(database.GetOutputCoinByPublicKeyError, errors.Wrap(iter.Error(), "db.lvdb.NewIterator"))
	}
	for iter.Next() {
		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())
		arrDatabyPubkey = append(arrDatabyPubkey, value)
	}
	iter.Release()
	return arrDatabyPubkey, nil
}

// CleanCommitments - clear all list commitments in DB
func (db *db) CleanCommitments() error {
	iter := db.lvdb.NewIterator(util.BytesPrefix(commitmentsPrefix), nil)
	for iter.Next() {
		err := db.Delete(iter.Key())
		if err != nil {
			return database.NewDatabaseError(database.CleanCommitmentError, err)
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return database.NewDatabaseError(database.CleanCommitmentError, err)
	}
	return nil
}

// StoreSNDerivators - store list serialNumbers by shardID
func (db *db) StoreSNDerivators(tokenID common.Hash, sndArray [][]byte) error {
	key := addPrefixToKeyHash(string(snderivatorsPrefix), tokenID)

	// "snderivator-data:nil"
	batchData := []database.BatchData{}
	for _, snd := range sndArray {
		keySpec := make([]byte, len(key))
		copy(keySpec, key)
		keySpec = append(keySpec, snd...)
		batchData = append(batchData, database.BatchData{
			Key:   keySpec,
			Value: []byte{},
		})
	}
	if len(batchData) > 0 {
		err := db.PutBatch(batchData)
		if err != nil {
			return database.NewDatabaseError(database.StoreSNDerivatorsError, err)
		}
	}
	return nil
}

// HasSNDerivator - Check SnDerivator in list SnDerivators by shardID
func (db *db) HasSNDerivator(tokenID common.Hash, data []byte) (bool, error) {
	key := addPrefixToKeyHash(string(snderivatorsPrefix), tokenID)
	keySpec := append(key, data...)
	hasValue, err := db.HasValue(keySpec)
	if err != nil {
		return false, database.NewDatabaseError(database.HasSNDerivatorError, err, data, -1, tokenID)
	} else {
		return hasValue, nil
	}
}

func (db *db) ListSNDerivator(tokenID common.Hash) ([][]byte, error) {
	result := make([][]byte, 0)
	key := addPrefixToKeyHash(string(snderivatorsPrefix), tokenID)

	iterator := db.lvdb.NewIterator(util.BytesPrefix(key), nil)
	for iterator.Next() {
		key1 := make([]byte, len(iterator.Key()))
		copy(key1, iterator.Key())

		sndInByte := key1[len(key)-1:]
		result = append(result, sndInByte)
	}
	return result, nil
}

// CleanCommitments - clear all list commitments in DB
func (db *db) CleanSNDerivator() error {
	iter := db.lvdb.NewIterator(util.BytesPrefix(snderivatorsPrefix), nil)
	for iter.Next() {
		err := db.Delete(iter.Key())
		if err != nil {
			return database.NewDatabaseError(database.CleanSNDerivatorError, err)
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return database.NewDatabaseError(database.CleanSNDerivatorError, err)
	}
	return nil
}

// StoreFeeEstimator - Store data for FeeEstimator object
func (db *db) StoreFeeEstimator(val []byte, shardID byte) error {
	if err := db.Put(append(feeEstimatorPrefix, shardID), val); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "StoreFeeEstimator"))
	}
	return nil
}

// GetFeeEstimator - Get data for FeeEstimator object as a json in byte format
func (db *db) GetFeeEstimator(shardID byte) ([]byte, error) {
	b, err := db.Get(append(feeEstimatorPrefix, shardID))
	if err != nil {
		return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "GetFeeEstimator"))
	}
	return b, err
}

// CleanFeeEstimator - Clear FeeEstimator
func (db *db) CleanFeeEstimator() error {
	iter := db.lvdb.NewIterator(util.BytesPrefix(feeEstimatorPrefix), nil)
	for iter.Next() {
		err := db.Delete(iter.Key())
		if err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "CleanFeeEstimator"))
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "CleanFeeEstimator"))
	}
	return nil
}

/*
	StoreTransactionIndex
	Store tx detail location
  Key: prefixTx-txHash
	H: blockHash-blockIndex
*/
func (db *db) StoreTransactionIndex(txId common.Hash, blockHash common.Hash, index int, bd *[]database.BatchData) error {
	key := string(transactionKeyPrefix) + txId.String()
	value := blockHash.String() + string(Splitter) + strconv.Itoa(index)

	if bd != nil {
		*bd = append(*bd, database.BatchData{[]byte(key), []byte(value)})
		return nil
	}
	if err := db.Put([]byte(key), []byte(value)); err != nil {
		return database.NewDatabaseError(database.StoreTransactionIndexError, err, txId.String(), blockHash.String(), index)
	}

	return nil
}

/*
  Get Transaction by ID
*/

func (db *db) GetTransactionIndexById(txId common.Hash) (common.Hash, int, error) {
	key := string(transactionKeyPrefix) + txId.String()
	_, err := db.HasValue([]byte(key))
	if err != nil {
		return common.Hash{}, -1, database.NewDatabaseError(database.GetTransactionIndexByIdError, err, txId.String())
	}

	res, err := db.Get([]byte(key))
	if err != nil {
		return common.Hash{}, -1, database.NewDatabaseError(database.GetTransactionIndexByIdError, err, txId.String())
	}
	reses := strings.Split(string(res), (string(Splitter)))
	hash, err := common.Hash{}.NewHashFromStr(reses[0])
	if err != nil {
		return common.Hash{}, -1, database.NewDatabaseError(database.GetTransactionIndexByIdError, err, txId.String())
	}
	index, err := strconv.Atoi(reses[1])
	if err != nil {
		return common.Hash{}, -1, database.NewDatabaseError(database.GetTransactionIndexByIdError, err, txId.String())
	}
	return *hash, index, nil
}

// StoreTxByPublicKey - store txID by public key of receiver,
// use this data to get tx which send to receiver
func (db *db) StoreTxByPublicKey(publicKey []byte, txID common.Hash, shardID byte) error {
	key := make([]byte, 0)
	key = append(key, publicKey...)       // 1st 33b bytes for pubkey
	key = append(key, txID.GetBytes()...) // 2nd 32 bytes fir txID which receiver get from
	key = append(key, shardID)            // 3nd 1 byte for shardID where sender send to receiver

	if err := db.Put(key, []byte{}); err != nil {
		database.Logger.Log.Debug("StoreTxByPublicKey", err)
		return database.NewDatabaseError(database.StoreTxByPublicKeyError, err, txID.String(), publicKey, shardID)
	}

	return nil
}

// GetTxByPublicKey -  from public key, use this function to get list all txID which someone send use by txID from any shardID
func (db *db) GetTxByPublicKey(publicKey []byte) (map[byte][]common.Hash, error) {
	itertor := db.lvdb.NewIterator(util.BytesPrefix(publicKey), nil)
	result := make(map[byte][]common.Hash)
	for itertor.Next() {
		iKey := itertor.Key()
		key := make([]byte, len(iKey))
		copy(key, iKey)
		shardID := key[len(key)-1]
		if result[shardID] == nil {
			result[shardID] = make([]common.Hash, 0)
		}
		txID := common.Hash{}
		err := txID.SetBytes(key[common.PublicKeySize : common.PublicKeySize+common.HashSize])
		if err != nil {
			database.Logger.Log.Debugf("Err at GetTxByPublicKey", err)
			return nil, database.NewDatabaseError(database.GetTxByPublicKeyError, err, publicKey)
		}
		result[shardID] = append(result[shardID], txID)
	}
	return result, nil
}

/*********************************** TRANSACTION VERSION 2 ***********************************/

//StoreOutputCoins - store output coin bytes of publicKey in indexOutputInTx at blockHeight
// key: [outcoinsPrefixV2][tokenID][shardID][blockHeight][publicKey][indexOutputInTx][ephemeralPubKey][hash(output)]
// value: output in bytes
func (db *db) StoreOutputCoinsV2(tokenID common.Hash, shardID byte, blockHeight uint64, publicKey []byte, outputCoinBytes [][]byte, indexOutputInTx []byte, ephemeralPubKey []byte) error {
	keyTmp := addPrefixToKeyHash(string(outcoinsPrefixV2), tokenID)
	keyTmp = append(keyTmp, shardID)
	keyTmp = append(keyTmp, common.AddPaddingBigInt(new(big.Int).SetUint64(blockHeight), common.Uint64Size)...)
	keyTmp = append(keyTmp, publicKey...)

	batchData := []database.BatchData{}
	for i, item := range outputCoinBytes {
		key := []byte{}
		// has privacy
		if ephemeralPubKey != nil {
			key = append(keyTmp, indexOutputInTx[i])
			key = append(key, ephemeralPubKey...)
		}

		key = append(key, common.HashB(item)...)
		// Put to batch
		batchData = append(batchData, database.BatchData{
			Key:   key,
			Value: item,
		})
	}

	if len(batchData) > 0 {
		err := db.PutBatch(batchData)
		if err != nil {
			return database.NewDatabaseError(database.StoreOutputCoinsError, err)
		}
	}

	return nil
}

//GetOutcoinsByPubkeyV2 - get all output coin of pubkey at blockheight
// key: [outcoinsPrefixV2][tokenID][shardID][blockHeight][pubKey]
// value: output in bytes
//func (db *db) GetOutcoinsByPubKeyV2(tokenID common.Hash, shardID byte, blockHeight uint64, pubKey []byte) ([][]byte, error) {
//	key := addPrefixToKeyHash(string(outcoinsPrefixV2), tokenID)
//	key = append(key, shardID)
//	key = append(key, new(big.Int).SetUint64(blockHeight).Bytes()...)
//	key = append(key, pubKey...)
//
//	arrDatabyPubkey := make([][]byte, 0)
//	iter := db.lvdb.NewIterator(util.BytesPrefix(key), nil)
//	if iter.Error() != nil {
//		return nil, database.NewDatabaseError(database.GetOutputCoinByPublicKeyError, errors.Wrap(iter.Error(), "db.lvdb.NewIterator"))
//	}
//	for iter.Next() {
//		value := make([]byte, len(iter.Value()))
//		copy(value, iter.Value())
//		arrDatabyPubkey = append(arrDatabyPubkey, value)
//	}
//	iter.Release()
//	return arrDatabyPubkey, nil
//}

//GetOutcoinsByViewKeyV2 - get list of output coins corresponding to viewKey at blockHeight
// both UTXO with fixec public key and one time public key
// key: [outcoinsPrefixV2][tokenID][shardID][blockHeight]
// value: output in bytes
func (db *db) GetOutcoinsByViewKeyV2(tokenID common.Hash, shardID byte, blockHeight uint64, viewKey privacy.ViewingKey) ([][]byte, error) {
	// get output coins with one time public key corresponding to viewKey
	outCoinsFromViewKeyBytes := make([][]byte, 0)
	keyPrefix := addPrefixToKeyHash(string(outcoinsPrefixV2), tokenID)
	keyPrefix = append(keyPrefix, shardID)
	keyPrefix = append(keyPrefix, common.AddPaddingBigInt(new(big.Int).SetUint64(blockHeight), common.Uint64Size)...)
	lenKeyPrefix := len(keyPrefix)

	iter := db.lvdb.NewIterator(util.BytesPrefix(keyPrefix), nil)
	if iter.Error() != nil {
		return nil, database.NewDatabaseError(database.GetOutputCoinByPublicKeyError, errors.Wrap(iter.Error(), "db.lvdb.NewIterator"))
	}

	for iter.Next() {
		// get public one-time address, index of output in tx from key
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())

		// 97 = len(PK) + len(index) + len(EphemeralPubKey) + len(hash(outCoins))
		if len(key) == lenKeyPrefix+97 {
			offset := lenKeyPrefix
			pubOneTimeAddr := key[offset : offset+privacy.Ed25519KeySize]
			pubOneTimeAddrPoint, err := new(privacy.Point).FromBytesS(pubOneTimeAddr)
			if err != nil {
				database.Logger.Log.Errorf("Invalid one time address from db")
				return nil, err
			}
			offset += privacy.Ed25519KeySize

			indexOutInTx := key[offset]
			offset += 1

			ephemeralPubKey := key[offset : offset+privacy.Ed25519KeySize]
			ephemeralPubKeyPoint, err := new(privacy.Point).FromBytesS(ephemeralPubKey)
			if err != nil {
				database.Logger.Log.Errorf("Invalid ephemeralPubKey from db")
				return nil, err
			}

			isPair, randOTA, err := privacy.IsPairOneTimeAddr(pubOneTimeAddrPoint, ephemeralPubKeyPoint, viewKey, int(indexOutInTx))
			if isPair && err == nil {
				outCoinBytes := make([]byte, len(iter.Value()))
				copy(outCoinBytes, iter.Value())

				tmpCoin := new(privacy.OutputCoin)
				err := tmpCoin.SetBytes(outCoinBytes)
				if err != nil {
					database.Logger.Log.Errorf("Can not setbytes output coins from db")
				}
				tmpCoin.CoinDetails.SetPrivRandOTA(randOTA)

				outCoinsFromViewKeyBytes = append(outCoinsFromViewKeyBytes, tmpCoin.Bytes())
			}
		}
	}

	iter.Release()
	return outCoinsFromViewKeyBytes, nil
}

//func (db *db) GetOutcoinsByPubKeyV2InBlocks(tokenID common.Hash, shardID byte, fromBlock uint64, toBlock uint64, pubKey []byte) ([][]byte, error) {
//	outCoins := make([][]byte, 0)
//
//	for i := fromBlock; i <= toBlock; i++ {
//		outCoinsTmp, err := db.GetOutcoinsByPubKeyV2(tokenID, shardID, i, pubKey)
//		if err != nil {
//			return nil, err
//		}
//		outCoins = append(outCoins, outCoinsTmp...)
//	}
//
//	return outCoins, nil
//}

// GetOutcoinsByViewKeyV2InBlocks returns all output coins with fixed publicKey in all blocks
// and output coins with one-time address corresponding to viewing Key in block [fromBlock, toBlock]
func (db *db) GetOutcoinsByViewKeyV2InBlocks(tokenID common.Hash, shardID byte, fromBlock uint64, toBlock uint64, viewKey privacy.ViewingKey) ([][]byte, error) {
	// get all output coins with fixed public key in all blocks
	outCoins, err := db.GetOutcoinsByPubkey(tokenID, viewKey.Pk, shardID)
	if err != nil {
		return nil, err
	}

	// get output coins with one-time address corresponding to viewing Key in block [fromBlock, toBlock]
	for i := fromBlock; i <= toBlock; i++ {
		outCoinsTmp, err := db.GetOutcoinsByViewKeyV2(tokenID, shardID, i, viewKey)
		if err != nil {
			return nil, err
		}
		outCoins = append(outCoins, outCoinsTmp...)
	}

	return outCoins, nil
}

// StoreEphemeralPubKey - store ephemeralPubKey by shardID and tokenID
func (db *db) StoreEphemeralPubKey(tokenID common.Hash, ephemeralPubKey []byte) error {
	key := addPrefixToKeyHash(string(ephemeralPubKeyPrefix), tokenID)
	key = append(key, ephemeralPubKey...)

	if err := db.Put(key, nil); err != nil {
		return database.NewDatabaseError(database.StoreEphemeralPubKeyError, err)
	}

	return nil
}

// HasEphemeralPubKey - Check ephemeralPubKey in list ephemeralPubKey by shardID and tokenID
func (db *db) HasEphemeralPubKey(tokenID common.Hash, ephemeralPubKey []byte) (bool, error) {
	key := addPrefixToKeyHash(string(ephemeralPubKeyPrefix), tokenID)
	keySpec := append(key, ephemeralPubKey...)

	hasValue, err := db.HasValue(keySpec)
	if err != nil {
		return false, database.NewDatabaseError(database.HasEphemeralPubKeyError, err, ephemeralPubKey, tokenID.String())
	} else {
		return hasValue, nil
	}
}


// StoreTxByPubKeyV2 - store txID by public key of receiver,
// use this data to get tx which send to receiver
// It is used to store tx v2 (has one time address)
func (db *db) StoreTxByPubKeyV2(blockHeight uint64, shardIDReceiver byte, shardIDSender byte, publicKey []byte, txID common.Hash, indexOutputInTx byte, ephemeralPubKey []byte, isPToken bool) error {
	key := make([]byte, 0)
	key = append(key, common.AddPaddingBigInt(new(big.Int).SetUint64(blockHeight), common.Uint64Size)...)       // 8 bytes for block height
	key = append(key, shardIDReceiver)			// 1 byte for shardID receiver
	key = append(key, shardIDSender)			// 1 byte for shardID sender
	key = append(key, publicKey...)				// 32 bytes for public key
	key = append(key, indexOutputInTx)			// 1 byte for index of output in tx
	key = append(key, ephemeralPubKey...)		// 32 bytes for ephemeral public key
	key = append(key, txID.GetBytes()...) 		// 32 bytes for txID

	if isPToken {
		tmp := []byte(OutputPtokenTag)
		key = append(key, tmp...)
	}

	if err := db.Put(key, []byte{}); err != nil {
		database.Logger.Log.Debug("StoreTxByPublicKey", err)
		return database.NewDatabaseError(database.StoreTxByPublicKeyError, err, txID.String(), publicKey, blockHeight, shardIDSender)
	}

	return nil
}

// GetTxByPublicKey -  from public key, use this function to get list all txID which someone send use by txID from any shardID
// and index output of public key in tx normal, and in tx ptoken
func (db *db) GetTxByViewKeyV2(viewKey privacy.ViewingKey, blockHeight uint64) (map[byte][]common.Hash, map[common.Hash][]byte, map[common.Hash][]byte, error) {
	// get shardID of reciever
	shardIDReceiver := common.GetShardIDFromLastByte(viewKey.Pk[len(viewKey.Pk) - 1])

	// get key prefix
	keyPrefix := make([]byte, 0)
	keyPrefix = append(keyPrefix, common.AddPaddingBigInt(new(big.Int).SetUint64(blockHeight), common.Uint64Size)...)
	keyPrefix = append(keyPrefix, shardIDReceiver)
	lenKeyPrefix := len(keyPrefix)

	// get db with key prefix
	itertor := db.lvdb.NewIterator(util.BytesPrefix(keyPrefix), nil)

	listTxHash := make(map[byte][]common.Hash)
	indexOutputInTxNormal := make(map[common.Hash][]byte)
	indexOutputInTxPToken := make(map[common.Hash][]byte)

	for itertor.Next() {
		offset := lenKeyPrefix
		iKey := itertor.Key()
		key := make([]byte, len(iKey))
		copy(key, iKey)

		// get shardIDSender
		shardIDSender := key[lenKeyPrefix]
		offset++

		// get publicKey one time address, index output in tx, ephemeralPubkey
		pubKeyOTA := key[offset: offset + privacy.Ed25519KeySize]
		offset = offset + privacy.Ed25519KeySize
		pubKeyOTAPoint, err := new(privacy.Point).FromBytesS(pubKeyOTA)
		if err != nil{
			return nil, nil, nil, database.NewDatabaseError(database.GetTxByPublicKeyError, err)
		}
		indexOutputInTx := key[offset]
		offset = offset + 1
		ephemeralPubKey := key[offset: offset + privacy.Ed25519KeySize]
		offset = offset + privacy.Ed25519KeySize
		ephemeralPubKeyPoint, err := new(privacy.Point).FromBytesS(ephemeralPubKey)
		if err != nil{
			return nil, nil, nil, database.NewDatabaseError(database.GetTxByPublicKeyError, err)
		}

		// check whether pubKeyOTA corresponding to viewKey or not
		isPair, _, err := privacy.IsPairOneTimeAddr(pubKeyOTAPoint, ephemeralPubKeyPoint, viewKey, int(indexOutputInTx))
		if isPair && err == nil {
			if listTxHash[shardIDSender] == nil {
				listTxHash[shardIDSender] = make([]common.Hash, 0)
			}

			txID := common.Hash{}
			txIDBytes := key[offset: offset + common.HashSize]
			offset = offset + common.HashSize
			err := txID.SetBytes(txIDBytes)
			if err != nil {
				database.Logger.Log.Debugf("Err at GetTxByViewKeyV2", err)
				return nil, nil, nil, database.NewDatabaseError(database.GetTxByPublicKeyError, err)
			}

			isExisted, _ := common.SliceExists(listTxHash[shardIDSender], txID)

			if !isExisted {
				listTxHash[shardIDSender] = append(listTxHash[shardIDSender], txID)
			}

			isPToken := false
			if offset < len(key) {
				tmp := string(key[offset])
				if tmp == OutputPtokenTag {
					isPToken = true
				}
				offset++
			}

			if isPToken {
				if indexOutputInTxPToken[txID] == nil {
					indexOutputInTxPToken[txID] = make([]byte, 0)
				}
				indexOutputInTxPToken[txID] = append(indexOutputInTxPToken[txID], indexOutputInTx)
			} else{
				if indexOutputInTxNormal[txID] == nil {
					indexOutputInTxNormal[txID] = make([]byte, 0)
				}
				indexOutputInTxNormal[txID] = append(indexOutputInTxNormal[txID], indexOutputInTx)
			}

		}
	}
	return listTxHash, indexOutputInTxNormal, indexOutputInTxPToken, nil
}

// GetTxByPublicKey -  from public key, use this function to get list all txID which someone send use by txID from any shardID
func (db *db) GetTxByViewKeyV2InBlocks(viewKey privacy.ViewingKey, fromBlockHeight uint64, toBlockHeight uint64)(
	map[byte][]common.Hash, map[byte][]common.Hash, map[common.Hash][]byte, map[common.Hash][]byte, error) {

	txHashListFromFixedPK := make(map[byte][]common.Hash)
	txHashListFromPKOTA := make(map[byte][]common.Hash)
	indexOutputTxNormal :=  make(map[common.Hash][]byte)
	indexOutputTxPToken :=  make(map[common.Hash][]byte)

	// get tx with fixed public key
	txHashListFromFixedPK, err := db.GetTxByPublicKey( viewKey.Pk)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// get tx v2 with one time address
	for i := fromBlockHeight; i<= toBlockHeight; i++{
		txHashListFromPKOTATmp, indexOutputTxNormalTmp, indexOutputTxPTokenTmp, err := db.GetTxByViewKeyV2(viewKey, i)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		for shardID, txHashs := range txHashListFromPKOTATmp {
			if txHashListFromPKOTA[shardID] == nil {
				txHashListFromPKOTA[shardID] = make([]common.Hash, 0)
			}

			txHashListFromPKOTA[shardID] = append(txHashListFromPKOTA[shardID], txHashs...)
		}

		for txHash, indices := range indexOutputTxNormalTmp {
			indexOutputTxNormal[txHash] = indices
		}

		for txHash, indices := range indexOutputTxPTokenTmp {
			indexOutputTxPToken[txHash] = indices
		}
	}

	return txHashListFromFixedPK, txHashListFromPKOTA, indexOutputTxNormal, indexOutputTxPToken, nil
}

