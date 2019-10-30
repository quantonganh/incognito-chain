package transaction

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"sort"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge"
	"github.com/incognitochain/incognito-chain/wallet"
)

// TxCustomTokenPrivacy is class tx which is inherited from P tx(supporting privacy) for fee
// and contain data(with supporting privacy format) to support issuing and transfer a custom token(token from end-user, look like erc-20)
// Dev or end-user can use this class tx to create an token type which use personal purpose
// TxCustomTokenPrivacy is an advance format of TxNormalToken
// so that user need to spend a lot fee to create this class tx
type TxCustomTokenPrivacy struct {
	Tx                                    // inherit from normal tx of P(supporting privacy) with a high fee to ensure that tx could contain a big data of privacy for token
	TxPrivacyTokenData TxPrivacyTokenData `json:"TxTokenPrivacyData"` // supporting privacy format

	// private field, not use for json parser, only use as temp variable
	cachedHash *common.Hash // cached hash data of tx
}

func (txCustomTokenPrivacy *TxCustomTokenPrivacy) UnmarshalJSON(data []byte) error {
	tx := Tx{}
	err := json.Unmarshal(data, &tx)
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(PrivacyTokenPRVJsonError, err)
	}
	temp := &struct {
		TxTokenPrivacyData interface{}
	}{}
	err = json.Unmarshal(data, &temp)
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(PrivacyTokenJsonError, err)
	}
	TxTokenPrivacyDataJson, err := json.MarshalIndent(temp.TxTokenPrivacyData, "", "\t")
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(UnexpectedError, err)
	}
	err = json.Unmarshal(TxTokenPrivacyDataJson, &txCustomTokenPrivacy.TxPrivacyTokenData)
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(PrivacyTokenJsonError, err)
	}
	txCustomTokenPrivacy.Tx = tx
	return nil
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) String() string {
	// get hash of tx
	record := txCustomTokenPrivacy.Tx.Hash().String()

	// add more hash of tx custom token data privacy
	tokenPrivacyDataHash, _ := txCustomTokenPrivacy.TxPrivacyTokenData.Hash()
	record += tokenPrivacyDataHash.String()
	if txCustomTokenPrivacy.Metadata != nil {
		record += string(txCustomTokenPrivacy.Metadata.Hash()[:])
	}
	return record
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) JSONString() string {
	data, err := json.MarshalIndent(txCustomTokenPrivacy, "", "\t")
	if err != nil {
		Logger.log.Error(err)
		return ""
	}
	return string(data)
}

// Hash returns the hash of all fields of the transaction
func (txCustomTokenPrivacy *TxCustomTokenPrivacy) Hash() *common.Hash {
	if txCustomTokenPrivacy.cachedHash != nil {
		return txCustomTokenPrivacy.cachedHash
	}
	// final hash
	hash := common.HashH([]byte(txCustomTokenPrivacy.String()))
	return &hash
}

// GetTxActualSize computes the virtual size of a given transaction
// size of this tx = (normal TxNormal size) + (custom token data size)
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTxActualSize() uint64 {
	normalTxSize := txCustomTokenPrivacy.Tx.GetTxActualSize()

	tokenDataSize := uint64(0)
	tokenDataSize += txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.GetTxActualSize()
	tokenDataSize += uint64(len(txCustomTokenPrivacy.TxPrivacyTokenData.PropertyName))
	tokenDataSize += uint64(len(txCustomTokenPrivacy.TxPrivacyTokenData.PropertySymbol))
	tokenDataSize += uint64(len(txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID))
	tokenDataSize += 4 // for TxPrivacyTokenData.Type
	tokenDataSize += 8 // for TxPrivacyTokenData.Amount

	meta := txCustomTokenPrivacy.Metadata
	if meta != nil {
		tokenDataSize += meta.CalculateSize()
	}

	return normalTxSize + uint64(math.Ceil(float64(tokenDataSize)/1024))
}

func (tx TxCustomTokenPrivacy) GetTxPrivacyTokenActualSize() uint64 {
	tokenDataSize := uint64(0)
	tokenDataSize += tx.TxPrivacyTokenData.TxNormal.GetTxActualSize()
	tokenDataSize += uint64(len(tx.TxPrivacyTokenData.PropertyName))
	tokenDataSize += uint64(len(tx.TxPrivacyTokenData.PropertySymbol))
	tokenDataSize += uint64(len(tx.TxPrivacyTokenData.PropertyID))
	tokenDataSize += 4 // for TxPrivacyTokenData.Type
	tokenDataSize += 8 // for TxPrivacyTokenData.Amount

	meta := tx.TxPrivacyTokenData.TxNormal.Metadata
	if meta != nil {
		tokenDataSize += meta.CalculateSize()
	}

	return uint64(math.Ceil(float64(tokenDataSize) / 1024))
}

// CheckTransactionFee - check fee for all tx by use PRV as fee
func (tx TxCustomTokenPrivacy) CheckTransactionFee(minFeePerKbTxInNativeToken uint64) bool {
	if tx.IsSalaryTx() {
		return true
	}

	fullFee := minFeePerKbTxInNativeToken * tx.GetTxActualSize()
	return tx.GetTxFee() >= fullFee
}

//// CheckTransactionFeeByFeeToken - check fee for all tx by use token as fee
//func (tx TxCustomTokenPrivacy) CheckTransactionFeeByFeeToken(minFeePerKbTx uint64) bool {
//	if tx.IsSalaryTx() {
//		return true
//	}
//	fullFee := minFeePerKbTx * tx.GetTxActualSize()
//	return tx.GetTxFeeToken() >= fullFee
//}
//
//// CheckTransactionFeeByFeeTokenForTokenData - check fee for token data info in tx by use token as fee
//func (tx TxCustomTokenPrivacy) CheckTransactionFeeByFeeTokenForTokenData(minFeePerKbTx uint64) bool {
//	if tx.IsSalaryTx() {
//		return true
//	}
//	fullFee := minFeePerKbTx * tx.GetTxPrivacyTokenActualSize()
//	return tx.GetTxFeeToken() >= fullFee
//}

type TxPrivacyTokenInitParams struct {
	senderKey       *privacy.PrivateKey
	paymentInfo     []*privacy.PaymentInfo
	inputCoin       []*privacy.InputCoin
	feeNativeCoin   uint64
	tokenParams     *CustomTokenPrivacyParamTx
	db              database.DatabaseInterface
	metaData        metadata.Metadata
	hasPrivacyCoin  bool
	hasPrivacyToken bool
	shardID         byte
	info            []byte
}

func NewTxPrivacyTokenInitParams(senderKey *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoin []*privacy.InputCoin,
	feeNativeCoin uint64,
	tokenParams *CustomTokenPrivacyParamTx,
	db database.DatabaseInterface,
	metaData metadata.Metadata,
	hasPrivacyCoin bool,
	hasPrivacyToken bool,
	shardID byte,
	info []byte) *TxPrivacyTokenInitParams {
	params := &TxPrivacyTokenInitParams{
		shardID:         shardID,
		paymentInfo:     paymentInfo,
		metaData:        metaData,
		db:              db,
		feeNativeCoin:   feeNativeCoin,
		hasPrivacyCoin:  hasPrivacyCoin,
		hasPrivacyToken: hasPrivacyToken,
		inputCoin:       inputCoin,
		senderKey:       senderKey,
		tokenParams:     tokenParams,
		info:            info,
	}
	return params
}

// Init -  build normal tx component and privacy custom token data
func (txCustomTokenPrivacy *TxCustomTokenPrivacy) Init(params *TxPrivacyTokenInitParams) error {
	var err error
	// init data for tx PRV for fee
	normalTx := Tx{}
	err = normalTx.Init(NewTxPrivacyInitParams(
		params.senderKey,
		params.paymentInfo,
		params.inputCoin,
		params.feeNativeCoin,
		params.hasPrivacyCoin,
		params.db,
		nil,
		params.metaData,
		params.info))
	if err != nil {
		return NewTransactionErr(PrivacyTokenInitPRVError, err)
	}
	// override TxCustomTokenPrivacyType type
	normalTx.Type = common.TxCustomTokenPrivacyType
	txCustomTokenPrivacy.Tx = normalTx

	// check action type and create privacy custom toke data
	var handled = false
	// Add token data component
	switch params.tokenParams.TokenTxType {
	case CustomTokenInit:
		// case init a new privacy custom token
		{
			handled = true
			txCustomTokenPrivacy.TxPrivacyTokenData = TxPrivacyTokenData{
				Type:           params.tokenParams.TokenTxType,
				PropertyName:   params.tokenParams.PropertyName,
				PropertySymbol: params.tokenParams.PropertySymbol,
				Amount:         params.tokenParams.Amount,
			}

			// issue token with data of privacy
			temp := Tx{}
			temp.Type = common.TxNormalType
			temp.Proof = new(zkp.PaymentProof)
			tempOutputCoin := make([]*privacy.OutputCoin, 1)
			tempOutputCoin[0] = new(privacy.OutputCoin)
			tempOutputCoin[0].CoinDetails = new(privacy.Coin)
			tempOutputCoin[0].CoinDetails.SetValue(params.tokenParams.Amount)
			PK, err := new(privacy.Point).FromBytesS(params.tokenParams.Receiver[0].PaymentAddress.Pk)
			if err != nil {
				return NewTransactionErr(DecompressPaymentAddressError, err)
			}
			tempOutputCoin[0].CoinDetails.SetPublicKey(PK)
			tempOutputCoin[0].CoinDetails.SetRandomness(privacy.RandomScalar())

			sndOut := privacy.RandomScalar()
			tempOutputCoin[0].CoinDetails.SetSNDerivator(sndOut)
			temp.Proof.SetOutputCoins(tempOutputCoin)

			// create coin commitment
			err = temp.Proof.GetOutputCoins()[0].CoinDetails.CommitAll()
			if err != nil {
				return NewTransactionErr(CommitOutputCoinError, err)
			}
			// get last byte
			temp.PubKeyLastByteSender = params.tokenParams.Receiver[0].PaymentAddress.Pk[len(params.tokenParams.Receiver[0].PaymentAddress.Pk)-1]

			// sign Tx
			temp.SigPubKey = params.tokenParams.Receiver[0].PaymentAddress.Pk
			temp.sigPrivKey = *params.senderKey
			err = temp.signTx()
			if err != nil {
				Logger.log.Error(errors.New("can't sign this tx"))
				return NewTransactionErr(SignTxError, err)
			}

			txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal = temp
			hashInitToken, err := txCustomTokenPrivacy.TxPrivacyTokenData.Hash()
			if err != nil {
				Logger.log.Error(errors.New("can't hash this token data"))
				return NewTransactionErr(UnexpectedError, err)
			}

			if params.tokenParams.Mintable {
				propertyID, err := common.Hash{}.NewHashFromStr(params.tokenParams.PropertyID)
				if err != nil {
					return NewTransactionErr(TokenIDInvalidError, err)
				}
				txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID = *propertyID
				txCustomTokenPrivacy.TxPrivacyTokenData.Mintable = true
			} else {
				//NOTICE: @merman update PropertyID calculated from hash of tokendata and shardID
				newHashInitToken := common.HashH(append(hashInitToken.GetBytes(), params.shardID))
				Logger.log.Debug("New Privacy Token %+v ", newHashInitToken)
				existed := params.db.PrivacyTokenIDExisted(newHashInitToken)
				if existed {
					Logger.log.Error("INIT Tx Custom Token Privacy is Existed", newHashInitToken)
					return NewTransactionErr(TokenIDExistedError, errors.New("this token is existed in network"))
				}
				existed = params.db.PrivacyTokenIDCrossShardExisted(newHashInitToken)
				if existed {
					Logger.log.Error("INIT Tx Custom Token Privacy is Existed(crossshard)", newHashInitToken)
					return NewTransactionErr(TokenIDExistedByCrossShardError, errors.New("this token is existed in network via cross shard"))
				}
				txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID = newHashInitToken
				Logger.log.Debugf("A new token privacy wil be issued with ID: %+v", txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID.String())
			}
		}
	case CustomTokenTransfer:
		{
			handled = true
			// make a transfering for privacy custom token
			// fee always 0 and reuse function of normal tx for custom token ID
			temp := Tx{}
			propertyID, _ := common.Hash{}.NewHashFromStr(params.tokenParams.PropertyID)
			existed := params.db.PrivacyTokenIDExisted(*propertyID)
			existedCross := params.db.PrivacyTokenIDCrossShardExisted(*propertyID)
			if !existed && !existedCross {
				return NewTransactionErr(TokenIDExistedError, errors.New("invalid Token ID"))
			}
			Logger.log.Debugf("Token %+v wil be transfered with", propertyID)
			txCustomTokenPrivacy.TxPrivacyTokenData = TxPrivacyTokenData{
				Type:           params.tokenParams.TokenTxType,
				PropertyName:   params.tokenParams.PropertyName,
				PropertySymbol: params.tokenParams.PropertySymbol,
				PropertyID:     *propertyID,
				Mintable:       params.tokenParams.Mintable,
			}
			err := temp.Init(NewTxPrivacyInitParams(params.senderKey,
				params.tokenParams.Receiver,
				params.tokenParams.TokenInput,
				params.tokenParams.Fee,
				params.hasPrivacyToken,
				params.db,
				propertyID,
				nil,
				params.info))
			if err != nil {
				return NewTransactionErr(PrivacyTokenInitTokenDataError, err)
			}
			txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal = temp
		}
	}

	if !handled {
		return NewTransactionErr(PrivacyTokenTxTypeNotHandleError, errors.New("can't handle this TokenTxType"))
	}
	return nil
}

// ValidateType - check type of tx
func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateType() bool {
	return txCustomTokenPrivacy.Type == common.TxCustomTokenPrivacyType
}

// ValidateTxWithCurrentMempool - validate for serrial number use in tx is double with other tx in mempool
func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateTxWithCurrentMempool(mr metadata.MempoolRetriever) error {
	poolSerialNumbersHashH := mr.GetSerialNumbersHashH()
	err := txCustomTokenPrivacy.validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH)
	if err != nil {
		Logger.log.Error(err)
		return NewTransactionErr(DoubleSpendError, err)
	}
	return nil
}

// validateDoubleSpendTxWithCurrentMempool - from proof of PRV and pToken,
// check serrial numbers is valid,
// not double spend with any tx in mempool
// this a private func -> call by ValidateTxWithCurrentMempool
func (txCustomTokenPrivacy TxCustomTokenPrivacy) validateDoubleSpendTxWithCurrentMempool(poolSerialNumbersHashH map[common.Hash][]common.Hash) error {
	// check proof of PRV and pToken
	if txCustomTokenPrivacy.Proof == nil && txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof == nil {
		return errors.New("empty tx")
	}

	// collect serial number for PRV
	temp := make(map[common.Hash]interface{})
	if txCustomTokenPrivacy.Proof != nil {
		for _, desc := range txCustomTokenPrivacy.Proof.GetInputCoins() {
			hash := common.HashH(desc.CoinDetails.GetSerialNumber().ToBytesS())
			temp[hash] = nil
		}
	}
	// collect serial number for pToken
	if txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof != nil {
		for _, desc := range txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof.GetInputCoins() {
			hash := common.HashH(desc.CoinDetails.GetSerialNumber().ToBytesS())
			temp[hash] = nil
		}
	}

	// check with pool serial number in mempool
	for _, listSerialNumbers := range poolSerialNumbersHashH {
		for _, serialNumberHash := range listSerialNumbers {
			if _, ok := temp[serialNumberHash]; ok {
				return errors.New("double spend")
			}
		}
	}
	return nil
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateTxWithBlockChain(
	bcr metadata.BlockchainRetriever,
	shardID byte,
	db database.DatabaseInterface,
) error {
	err := txCustomTokenPrivacy.ValidateDoubleSpendWithBlockchain(bcr, shardID, db, nil)
	if err != nil {
		return NewTransactionErr(InvalidDoubleSpendPRVError, err)
	}
	err = txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.ValidateDoubleSpendWithBlockchain(bcr, shardID, db, txCustomTokenPrivacy.GetTokenID())
	if err != nil {
		return NewTransactionErr(InvalidDoubleSpendPrivacyTokenError, err)
	}
	return nil
}

// ValidateSanityData - validate sanity data of PRV and pToken
func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateSanityData(bcr metadata.BlockchainRetriever) (bool, error) {
	meta := txCustomTokenPrivacy.Tx.Metadata
	if meta != nil {
		isContinued, ok, err := meta.ValidateSanityData(bcr, &txCustomTokenPrivacy)
		if err != nil || !ok || !isContinued {
			return ok, err
		}
	}

	// validate sanity data for PRV
	//result, err := txCustomTokenPrivacy.Tx.validateNormalTxSanityData()
	result, err := txCustomTokenPrivacy.Tx.ValidateSanityData(bcr)
	if err != nil {
		return result, NewTransactionErr(InvalidSanityDataPRVError, err)
	}
	// validate sanity for pToken

	//result, err = txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.validateNormalTxSanityData()
	result, err = txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.ValidateSanityData(bcr)
	if err != nil {
		return result, NewTransactionErr(InvalidSanityDataPrivacyTokenError, err)
	}
	return result, nil
}

// ValidateTxByItself - validate tx by itself, check signature, proof,... and metadata
func (txCustomTokenPrivacy TxCustomTokenPrivacy) ValidateTxByItself(
	hasPrivacyCoin bool,
	db database.DatabaseInterface,
	bcr metadata.BlockchainRetriever,
	shardID byte,
) (bool, error) {
	// no need to check for tx init token
	if txCustomTokenPrivacy.TxPrivacyTokenData.Type == CustomTokenInit {
		return txCustomTokenPrivacy.Tx.ValidateTransaction(hasPrivacyCoin, db, shardID, nil)
	}
	// check for proof, signature ...
	if ok, err := txCustomTokenPrivacy.ValidateTransaction(hasPrivacyCoin, db, shardID, nil); !ok {
		return false, err
	}

	// check for metadata
	if txCustomTokenPrivacy.Metadata != nil {
		validateMetadata := txCustomTokenPrivacy.Metadata.ValidateMetadataByItself()
		if !validateMetadata {
			return validateMetadata, NewTransactionErr(UnexpectedError, errors.New("Metadata is invalid"))
		}
		return validateMetadata, nil
	}
	return true, nil
}

// ValidateTransaction - verify proof, signature, ... of PRV and pToken
func (txCustomTokenPrivacy *TxCustomTokenPrivacy) ValidateTransaction(hasPrivacyCoin bool, db database.DatabaseInterface, shardID byte, tokenID *common.Hash) (bool, error) {
	// validate for PRV
	ok, err := txCustomTokenPrivacy.Tx.ValidateTransaction(hasPrivacyCoin, db, shardID, nil)
	if ok {
		// validate for pToken
		tokenID := txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID
		if txCustomTokenPrivacy.TxPrivacyTokenData.Type == CustomTokenInit {
			return true, nil
		} else {
			return txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.ValidateTransaction(txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.IsPrivacy(), db, shardID, &tokenID)
		}
	}
	return false, err
}

// GetProof - return proof PRV of tx
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetProof() *zkp.PaymentProof {
	return txCustomTokenPrivacy.Proof
}

// VerifyMinerCreatedTxBeforeGettingInBlock
func (txCustomTokenPrivacy TxCustomTokenPrivacy) VerifyMinerCreatedTxBeforeGettingInBlock(
	txsInBlock []metadata.Transaction,
	txsUsed []int,
	insts [][]string,
	instsUsed []int,
	shardID byte,
	bcr metadata.BlockchainRetriever,
	accumulatedValues *metadata.AccumulatedValues,
) (bool, error) {
	if !txCustomTokenPrivacy.TxPrivacyTokenData.Mintable {
		return true, nil
	}
	meta := txCustomTokenPrivacy.Metadata
	if meta == nil {
		Logger.log.Error("Mintable custom token must contain metadata")
		return false, nil
	}
	if !meta.IsMinerCreatedMetaType() {
		return false, nil
	}
	return meta.VerifyMinerCreatedTxBeforeGettingInBlock(txsInBlock, txsUsed, insts, instsUsed, shardID, &txCustomTokenPrivacy, bcr, accumulatedValues)
}

// GetTokenReceivers - return receivers in tx, who receive token
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTokenReceivers() ([][]byte, []uint64) {
	pubkeys := [][]byte{}
	amounts := []uint64{}
	// get proof pToken
	proof := txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof
	if proof == nil {
		return pubkeys, amounts
	}
	// fetch payment info
	for _, coin := range proof.GetOutputCoins() {
		coinPubKey := coin.CoinDetails.GetPublicKey().ToBytesS()
		added := false
		// coinPubKey := vout.PaymentAddress.Pk
		for i, key := range pubkeys {
			if bytes.Equal(coinPubKey, key) {
				added = true
				amounts[i] += coin.CoinDetails.GetValue()
				break
			}
		}
		if !added {
			pubkeys = append(pubkeys, coinPubKey)
			amounts = append(amounts, coin.CoinDetails.GetValue())
		}
	}
	return pubkeys, amounts
}

// GetTokenUniqueReceiver
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTokenUniqueReceiver() (bool, []byte, uint64) {
	sender := []byte{}
	proof := txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof
	if proof == nil {
		return false, []byte{}, 0
	}
	if len(proof.GetInputCoins()) > 0 && proof.GetInputCoins()[0].CoinDetails != nil {
		sender = proof.GetInputCoins()[0].CoinDetails.GetPublicKey().ToBytesS()
	}
	pubkeys, amounts := txCustomTokenPrivacy.GetTokenReceivers()
	pubkey := []byte{}
	amount := uint64(0)
	count := 0
	for i, pk := range pubkeys {
		if !bytes.Equal(pk, sender) {
			pubkey = pk
			amount = amounts[i]
			count += 1
		}
	}
	return count == 1, pubkey, amount
}

// GetTransferData
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTransferData() (bool, []byte, uint64, *common.Hash) {
	unique, pk, amount := txCustomTokenPrivacy.GetTokenUniqueReceiver()
	return unique, pk, amount, &txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID
}

// IsCoinsBurning - checking this is a burning pToken
func (txCustomTokenPrivacy TxCustomTokenPrivacy) IsCoinsBurning() bool {
	// get proof of pToken
	proof := txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof
	if proof == nil || len(proof.GetOutputCoins()) == 0 {
		return false
	}
	//  validate receiver with burning address
	senderPKBytes := []byte{}
	if len(proof.GetInputCoins()) > 0 {
		senderPKBytes = proof.GetInputCoins()[0].CoinDetails.GetPublicKey().ToBytesS()
	}
	keyWalletBurningAccount, err := wallet.Base58CheckDeserialize(common.BurningAddress)
	if err != nil {
		Logger.log.Errorf("Can not deserialize burn address: %v\n", common.BurningAddress)
		return false
	}

	keysetBurningAccount := keyWalletBurningAccount.KeySet
	paymentAddressBurningAccount := keysetBurningAccount.PaymentAddress
	for _, outCoin := range proof.GetOutputCoins() {
		outPKBytes := outCoin.CoinDetails.GetPublicKey().ToBytesS()
		if !bytes.Equal(senderPKBytes, outPKBytes) && !bytes.Equal(outPKBytes, paymentAddressBurningAccount.Pk[:]) {
			return false
		}
	}
	return true
}

// CalculateTxValue - get tx value for pToken
func (txCustomTokenPrivacy TxCustomTokenPrivacy) CalculateTxValue() uint64 {
	proof := txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.Proof
	if proof == nil {
		return 0
	}
	if proof.GetOutputCoins() == nil || len(proof.GetOutputCoins()) == 0 {
		return 0
	}
	if proof.GetInputCoins() == nil || len(proof.GetInputCoins()) == 0 { // coinbase tx
		txValue := uint64(0)
		for _, outCoin := range proof.GetOutputCoins() {
			txValue += outCoin.CoinDetails.GetValue()
		}
		return txValue
	}

	if txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.IsPrivacy() {
		return 0
	}

	senderPKBytes := proof.GetInputCoins()[0].CoinDetails.GetPublicKey().ToBytesS()
	txValue := uint64(0)
	for _, outCoin := range proof.GetOutputCoins() {
		outPKBytes := outCoin.CoinDetails.GetPublicKey().ToBytesS()
		if bytes.Equal(senderPKBytes, outPKBytes) {
			continue
		}
		txValue += outCoin.CoinDetails.GetValue()
	}
	return txValue
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) ListSerialNumbersHashH() []common.Hash {
	tx := txCustomTokenPrivacy.Tx
	result := []common.Hash{}
	if tx.Proof != nil {
		for _, d := range tx.Proof.GetInputCoins() {
			hash := common.HashH(d.CoinDetails.GetSerialNumber().ToBytesS())
			result = append(result, hash)
		}
	}
	customTokenPrivacy := txCustomTokenPrivacy.TxPrivacyTokenData
	if customTokenPrivacy.TxNormal.Proof != nil {
		for _, d := range customTokenPrivacy.TxNormal.Proof.GetInputCoins() {
			hash := common.HashH(d.CoinDetails.GetSerialNumber().ToBytesS())
			result = append(result, hash)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

// GetSigPubKey - return sig pubkey for pToken
func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetSigPubKey() []byte {
	return txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal.SigPubKey
}

// GetTxFeeToken - return Token Fee use to pay for privacy token Tx
func (tx TxCustomTokenPrivacy) GetTxFeeToken() uint64 {
	return tx.TxPrivacyTokenData.TxNormal.Fee
}

func (txCustomTokenPrivacy TxCustomTokenPrivacy) GetTokenID() *common.Hash {
	return &txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID
}

// GetTxFee - return fee PRV of Tx which contain privacy token Tx
func (tx TxCustomTokenPrivacy) GetTxFee() uint64 {
	return tx.Tx.GetTxFee()
}

type TxPrivacyTokenInitParamsForASM struct {
	//senderKey       *privacy.PrivateKey
	//paymentInfo     []*privacy.PaymentInfo
	//inputCoin       []*privacy.InputCoin
	//feeNativeCoin   uint64
	//tokenParams     *CustomTokenPrivacyParamTx
	//db              database.DatabaseInterface
	//metaData        metadata.Metadata
	//hasPrivacyCoin  bool
	//hasPrivacyToken bool
	//shardID         byte
	//info            []byte

	txParam                           TxPrivacyTokenInitParams
	commitmentIndicesForNativeToken   []uint64
	commitmentBytesForNativeToken     [][]byte
	myCommitmentIndicesForNativeToken []uint64
	sndOutputsForNativeToken          []*privacy.Scalar

	commitmentIndicesForPToken   []uint64
	commitmentBytesForPToken     [][]byte
	myCommitmentIndicesForPToken []uint64
	sndOutputsForPToken          []*privacy.Scalar
}

func (param *TxPrivacyTokenInitParamsForASM) SetMetaData(meta metadata.Metadata) {
	param.txParam.metaData = meta
}

func NewTxPrivacyTokenInitParamsForASM(
	senderKey *privacy.PrivateKey,
	paymentInfo []*privacy.PaymentInfo,
	inputCoin []*privacy.InputCoin,
	feeNativeCoin uint64,
	tokenParams *CustomTokenPrivacyParamTx,
	metaData metadata.Metadata,
	hasPrivacyCoin bool,
	hasPrivacyToken bool,
	shardID byte,
	info []byte,
	commitmentIndicesForNativeToken []uint64,
	commitmentBytesForNativeToken [][]byte,
	myCommitmentIndicesForNativeToken []uint64,
	sndOutputsForNativeToken []*privacy.Scalar,

	commitmentIndicesForPToken []uint64,
	commitmentBytesForPToken [][]byte,
	myCommitmentIndicesForPToken []uint64,
	sndOutputsForPToken []*privacy.Scalar) *TxPrivacyTokenInitParamsForASM {

	txParam := NewTxPrivacyTokenInitParams(senderKey, paymentInfo, inputCoin, feeNativeCoin, tokenParams, nil, metaData, hasPrivacyCoin, hasPrivacyToken, shardID, info)
	params := &TxPrivacyTokenInitParamsForASM{
		txParam:                           *txParam,
		commitmentIndicesForNativeToken:   commitmentIndicesForNativeToken,
		commitmentBytesForNativeToken:     commitmentBytesForNativeToken,
		myCommitmentIndicesForNativeToken: myCommitmentIndicesForNativeToken,
		sndOutputsForNativeToken:          sndOutputsForNativeToken,

		commitmentIndicesForPToken:   commitmentIndicesForPToken,
		commitmentBytesForPToken:     commitmentBytesForPToken,
		myCommitmentIndicesForPToken: myCommitmentIndicesForPToken,
		sndOutputsForPToken:          sndOutputsForPToken,
	}
	return params
}

// Init -  build normal tx component and privacy custom token data
func (txCustomTokenPrivacy *TxCustomTokenPrivacy) InitForASM(params *TxPrivacyTokenInitParamsForASM) error {
	var err error
	// init data for tx PRV for fee
	normalTx := Tx{}
	err = normalTx.InitForASM(NewTxPrivacyInitParamsForASM(
		params.txParam.senderKey,
		params.txParam.paymentInfo,
		params.txParam.inputCoin,
		params.txParam.feeNativeCoin,
		params.txParam.hasPrivacyCoin,
		nil,
		params.txParam.metaData,
		params.txParam.info,
		params.commitmentIndicesForNativeToken,
		params.commitmentBytesForNativeToken,
		params.myCommitmentIndicesForNativeToken,
		params.sndOutputsForNativeToken,
	))
	if err != nil {
		return NewTransactionErr(PrivacyTokenInitPRVError, err)
	}

	// override TxCustomTokenPrivacyType type
	normalTx.Type = common.TxCustomTokenPrivacyType
	txCustomTokenPrivacy.Tx = normalTx

	// check action type and create privacy custom toke data
	var handled = false
	// Add token data component
	switch params.txParam.tokenParams.TokenTxType {
	case CustomTokenInit:
		// case init a new privacy custom token
		{
			handled = true
			txCustomTokenPrivacy.TxPrivacyTokenData = TxPrivacyTokenData{
				Type:           params.txParam.tokenParams.TokenTxType,
				PropertyName:   params.txParam.tokenParams.PropertyName,
				PropertySymbol: params.txParam.tokenParams.PropertySymbol,
				Amount:         params.txParam.tokenParams.Amount,
			}

			// issue token with data of privacy
			temp := Tx{}
			temp.Type = common.TxNormalType
			temp.Proof = new(zkp.PaymentProof)
			tempOutputCoin := make([]*privacy.OutputCoin, 1)
			tempOutputCoin[0] = new(privacy.OutputCoin)
			tempOutputCoin[0].CoinDetails = new(privacy.Coin)
			tempOutputCoin[0].CoinDetails.SetValue(params.txParam.tokenParams.Amount)
			PK, err := new(privacy.Point).FromBytesS(params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk)
			if err != nil {
				return NewTransactionErr(DecompressPaymentAddressError, err)
			}
			tempOutputCoin[0].CoinDetails.SetPublicKey(PK)
			tempOutputCoin[0].CoinDetails.SetRandomness(privacy.RandomScalar())

			sndOut := privacy.RandomScalar()
			tempOutputCoin[0].CoinDetails.SetSNDerivator(sndOut)
			temp.Proof.SetOutputCoins(tempOutputCoin)

			// create coin commitment
			err = temp.Proof.GetOutputCoins()[0].CoinDetails.CommitAll()
			if err != nil {
				return NewTransactionErr(CommitOutputCoinError, err)
			}
			// get last byte
			temp.PubKeyLastByteSender = params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk[len(params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk)-1]

			// sign Tx
			temp.SigPubKey = params.txParam.tokenParams.Receiver[0].PaymentAddress.Pk
			temp.sigPrivKey = *params.txParam.senderKey
			err = temp.signTx()
			if err != nil {
				Logger.log.Error(errors.New("can't sign this tx"))
				return NewTransactionErr(SignTxError, err)
			}

			txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal = temp
			hashInitToken, err := txCustomTokenPrivacy.TxPrivacyTokenData.Hash()
			if err != nil {
				Logger.log.Error(errors.New("can't hash this token data"))
				return NewTransactionErr(UnexpectedError, err)
			}

			if params.txParam.tokenParams.Mintable {
				propertyID, err := common.Hash{}.NewHashFromStr(params.txParam.tokenParams.PropertyID)
				if err != nil {
					return NewTransactionErr(TokenIDInvalidError, err)
				}
				txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID = *propertyID
				txCustomTokenPrivacy.TxPrivacyTokenData.Mintable = true
			} else {
				//NOTICE: @merman update PropertyID calculated from hash of tokendata and shardID
				newHashInitToken := common.HashH(append(hashInitToken.GetBytes(), params.txParam.shardID))
				txCustomTokenPrivacy.TxPrivacyTokenData.PropertyID = newHashInitToken
			}
		}
	case CustomTokenTransfer:
		{
			handled = true
			// make a transfering for privacy custom token
			// fee always 0 and reuse function of normal tx for custom token ID
			temp := Tx{}
			propertyID, _ := common.Hash{}.NewHashFromStr(params.txParam.tokenParams.PropertyID)
			txCustomTokenPrivacy.TxPrivacyTokenData = TxPrivacyTokenData{
				Type:           params.txParam.tokenParams.TokenTxType,
				PropertyName:   params.txParam.tokenParams.PropertyName,
				PropertySymbol: params.txParam.tokenParams.PropertySymbol,
				PropertyID:     *propertyID,
				Mintable:       params.txParam.tokenParams.Mintable,
			}
			err := temp.InitForASM(NewTxPrivacyInitParamsForASM(
				params.txParam.senderKey,
				params.txParam.tokenParams.Receiver,
				params.txParam.tokenParams.TokenInput,
				params.txParam.tokenParams.Fee,
				params.txParam.hasPrivacyToken,
				propertyID,
				nil,
				params.txParam.info,
				params.commitmentIndicesForPToken,
				params.commitmentBytesForPToken,
				params.myCommitmentIndicesForPToken,
				params.sndOutputsForPToken,
			))
			if err != nil {
				return NewTransactionErr(PrivacyTokenInitTokenDataError, err)
			}
			txCustomTokenPrivacy.TxPrivacyTokenData.TxNormal = temp
		}
	}

	if !handled {
		return NewTransactionErr(PrivacyTokenTxTypeNotHandleError, errors.New("can't handle this TokenTxType"))
	}
	return nil
}
