package transaction

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitTxPrivacyToken(t *testing.T) {
	for i :=0; i < 1; i++ {
		//Generate sender private key & receiver payment address
		seed := privacy.RandomScalar().ToBytesS()
		masterKey, _ := wallet.NewMasterKey(seed)
		childSender, _ := masterKey.NewChildKey(uint32(1))
		privKeyB58 := childSender.Base58CheckSerialize(wallet.PriKeyType)
		childReceiver, _ := masterKey.NewChildKey(uint32(2))
		paymentAddressB58 := childReceiver.Base58CheckSerialize(wallet.PaymentAddressType)

		// sender key
		senderKey, err := wallet.Base58CheckDeserialize(privKeyB58)
		assert.Equal(t, nil, err)

		err = senderKey.KeySet.InitFromPrivateKey(&senderKey.KeySet.PrivateKey)
		assert.Equal(t, nil, err)

		//receiver key
		receiverKey, _ := wallet.Base58CheckDeserialize(paymentAddressB58)
		receiverPaymentAddress := receiverKey.KeySet.PaymentAddress

		shardID := common.GetShardIDFromLastByte(senderKey.KeySet.PaymentAddress.Pk[len(senderKey.KeySet.PaymentAddress.Pk)-1])

		// message to receiver
		msg := "Incognito-chain"
		receiverTK , _:= new(privacy.Point).FromBytesS(senderKey.KeySet.PaymentAddress.Tk)
		msgCipherText, _ := privacy.HybridEncrypt([]byte(msg), receiverTK)

		initAmount := uint64(10000)
		paymentInfo := []*privacy.PaymentInfo{{PaymentAddress: senderKey.KeySet.PaymentAddress, Amount: initAmount, Message: msgCipherText.Bytes() }}

		inputCoinsPRV := []*privacy.InputCoin{}
		paymentInfoPRV := []*privacy.PaymentInfo{}

		// token param for init new token
		tokenParam := &CustomTokenPrivacyParamTx{
			PropertyID:     "",
			PropertyName:   "Token 1",
			PropertySymbol: "Token 1",
			Amount:         initAmount,
			TokenTxType:    CustomTokenInit,
			Receiver:       paymentInfo,
			TokenInput:     []*privacy.InputCoin{},
			Mintable:       false,
			Fee:            0,
		}

		hasPrivacyForPRV := false
		hasPrivacyForToken := false

		paramToCreateTx := NewTxPrivacyTokenInitParams(&senderKey.KeySet.PrivateKey,
			paymentInfoPRV, inputCoinsPRV, 0, tokenParam, db, nil,
			hasPrivacyForPRV, hasPrivacyForToken, shardID, []byte{}, TxVersion2)

		// init tx
		tx := new(TxCustomTokenPrivacy)
		err = tx.Init(paramToCreateTx)
		assert.Equal(t, nil, err)

		assert.Equal(t, len(msgCipherText.Bytes()), len(tx.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins()[0].CoinDetails.GetInfo()))

		//fmt.Printf("Tx: %v\n", tx.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins()[0].CoinDetails.GetInfo())

		// convert to JSON string and revert
		txJsonString := tx.JSONString()
		txHash := tx.Hash()

		tx1 := new(TxCustomTokenPrivacy)
		tx1.UnmarshalJSON([]byte(txJsonString))
		txHash1 := tx1.Hash()
		assert.Equal(t, txHash, txHash1)

		// get actual tx size
		txActualSize := tx.GetTxActualSize()
		assert.Greater(t, txActualSize, uint64(0))

		txPrivacyTokenActualSize := tx.GetTxPrivacyTokenActualSize()
		assert.Greater(t, txPrivacyTokenActualSize, uint64(0))

		//isValidFee := tx.CheckTransactionFee(uint64(0))
		//assert.Equal(t, true, isValidFee)

		//isValidFeeToken := tx.CheckTransactionFeeByFeeToken(uint64(0))
		//assert.Equal(t, true, isValidFeeToken)
		//
		//isValidFeeTokenForTokenData := tx.CheckTransactionFeeByFeeTokenForTokenData(uint64(0))
		//assert.Equal(t, true, isValidFeeTokenForTokenData)

		isValidType := tx.ValidateType()
		assert.Equal(t, true, isValidType)

		//err = tx.ValidateTxWithCurrentMempool(nil)
		//assert.Equal(t, nil, err)

		err = tx.ValidateTxWithBlockChain(nil, shardID, db)
		assert.Equal(t, nil, err)

		isValidSanity, err := tx.ValidateSanityData(nil)
		assert.Equal(t, true, isValidSanity)
		assert.Equal(t, nil, err)

		isValidTxItself, err := tx.ValidateTxByItself(hasPrivacyForPRV, db, nil, shardID)
		assert.Equal(t, true, isValidTxItself)
		assert.Equal(t, nil, err)

		//isValidTx, err := tx.ValidateTransaction(hasPrivacyForPRV, db, shardID, tx.GetTokenID())
		//fmt.Printf("Err: %v\n", err)
		//assert.Equal(t, true, isValidTx)
		//assert.Equal(t, nil, err)

		_ = tx.GetProof()
		//assert.Equal(t, nil, proof)

		pubKeyReceivers, amounts := tx.GetTokenReceivers()
		assert.Equal(t, 1, len(pubKeyReceivers))
		assert.Equal(t, 1, len(amounts))
		assert.Equal(t, initAmount, amounts[0])

		isUniqueReceiver, uniquePubKey, uniqueAmount, tokenID := tx.GetTransferData()
		assert.Equal(t, true, isUniqueReceiver)
		assert.Equal(t, initAmount, uniqueAmount)
		assert.Equal(t, tx.GetTokenID(), tokenID)
		receiverPubKeyBytes := make([]byte, common.PublicKeySize)
		copy(receiverPubKeyBytes, senderKey.KeySet.PaymentAddress.Pk)
		assert.Equal(t, uniquePubKey, receiverPubKeyBytes)

		isCoinBurningTx := tx.IsCoinsBurning()
		assert.Equal(t, false, isCoinBurningTx)

		txValue := tx.CalculateTxValue()
		assert.Equal(t, initAmount, txValue)

		listSerialNumber := tx.ListSerialNumbersHashH()
		assert.Equal(t, 0, len(listSerialNumber))

		sigPubKey := tx.GetSigPubKey()
		assert.Equal(t, common.SigPubKeySize, len(sigPubKey))

		// store init tx

		// get output coin token from tx
		outputCoins := ConvertOutputCoinToInputCoin(tx.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins())

		// calculate serial number for input coins
		serialNumber := new(privacy.Point).Derive(privacy.PedCom.G[privacy.PedersenPrivateKeyIndex],
			new(privacy.Scalar).FromBytesS(senderKey.KeySet.PrivateKey),
			outputCoins[0].CoinDetails.GetSNDerivatorRandom())
		outputCoins[0].CoinDetails.SetSerialNumber(serialNumber)

		db.StorePrivacyToken(*tx.GetTokenID(), tx.Hash()[:])
		db.StoreCommitments(*tx.GetTokenID(), senderKey.KeySet.PaymentAddress.Pk[:], [][]byte{outputCoins[0].CoinDetails.GetCoinCommitment().ToBytesS()}, shardID)

		//listTokens, err := db.ListPrivacyToken()
		//assert.Equal(t, nil, err)
		//assert.Equal(t, 1, len(listTokens))

		transferAmount := uint64(10)

		paymentInfo2 := []*privacy.PaymentInfo{{PaymentAddress: receiverPaymentAddress, Amount: transferAmount, Message: msgCipherText.Bytes()}}

		// token param for transfer token
		tokenParam2 := &CustomTokenPrivacyParamTx{
			PropertyID:     tx.GetTokenID().String(),
			PropertyName:   "Token 1",
			PropertySymbol: "Token 1",
			Amount:         transferAmount,
			TokenTxType:    CustomTokenTransfer,
			Receiver:       paymentInfo2,
			TokenInput:     outputCoins,
			Mintable:       false,
			Fee:            0,
		}

		paramToCreateTx2 := NewTxPrivacyTokenInitParams(&senderKey.KeySet.PrivateKey,
			paymentInfoPRV, inputCoinsPRV, 0, tokenParam2, db, nil,
			hasPrivacyForPRV, true, shardID, []byte{}, TxVersion2)

		// init tx
		tx2 := new(TxCustomTokenPrivacy)
		err = tx2.Init(paramToCreateTx2)
		assert.Equal(t, nil, err)

		assert.Equal(t, len(msgCipherText.Bytes()), len(tx2.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins()[0].CoinDetails.GetInfo()))

		err = tx2.ValidateTxWithBlockChain(nil, shardID, db)
		assert.Equal(t, nil, err)

		isValidSanity, err = tx2.ValidateSanityData(nil)
		assert.Equal(t, true, isValidSanity)
		assert.Equal(t, nil, err)

		isValidTxItself, err = tx2.ValidateTxByItself(hasPrivacyForPRV, db, nil, shardID)
		assert.Equal(t, true, isValidTxItself)
		assert.Equal(t, nil, err)

		txValue2 := tx2.CalculateTxValue()
		assert.Equal(t, uint64(0), txValue2)
	}
}


func TestInitTxPTokenV1(t *testing.T) {
	for i := 0; i < 1; i++ {
		fmt.Println("********************* Coin base tx ********************* ")

		/****** Generate sender private key & receiver payment address ********/
		seed := privacy.RandomScalar().ToBytesS()
		masterKey, _ := wallet.NewMasterKey(seed)
		childSender, _ := masterKey.NewChildKey(uint32(1))
		childReceiver, _ := masterKey.NewChildKey(uint32(2))

		senderPrivKeyB58 := childSender.Base58CheckSerialize(wallet.PriKeyType)
		receiverPrivKeyB58 := childReceiver.Base58CheckSerialize(wallet.PriKeyType)
		receiverKeyWallet, _ := wallet.Base58CheckDeserialize(receiverPrivKeyB58)
		receiverKeyWallet.KeySet.InitFromPrivateKeyByte(receiverKeyWallet.KeySet.PrivateKey)
		//receiverPubKey := receiverKeyWallet.KeySet.PaymentAddress.Pk

		senderKey, err := wallet.Base58CheckDeserialize(senderPrivKeyB58)
		assert.Equal(t, nil, err)
		err = senderKey.KeySet.InitFromPrivateKey(&senderKey.KeySet.PrivateKey)
		assert.Equal(t, nil, err)

		senderPaymentAddress := senderKey.KeySet.PaymentAddress
		//senderPublicKey := senderPaymentAddress.Pk

		senderShardID := common.GetShardIDFromLastByte(senderKey.KeySet.PaymentAddress.Pk[len(senderKey.KeySet.PaymentAddress.Pk)-1])

		/******************************** INIT PTOKEN ********************************/
		// message to receiver
		msg := "Incognito-chain"
		receiverTK, _ := new(privacy.Point).FromBytesS(senderKey.KeySet.PaymentAddress.Tk)
		msgCipherText, _ := privacy.HybridEncrypt([]byte(msg), receiverTK)

		initAmount := uint64(10000)
		paymentInfo := []*privacy.PaymentInfo{{PaymentAddress: senderKey.KeySet.PaymentAddress, Amount: initAmount, Message: msgCipherText.Bytes()}}

		inputCoinsPRV := []*privacy.InputCoin{}
		paymentInfoPRV := []*privacy.PaymentInfo{}

		// token param for init new token
		tokenParam := &CustomTokenPrivacyParamTx{
			PropertyID:     "",
			PropertyName:   "Token 1",
			PropertySymbol: "Token 1",
			Amount:         initAmount,
			TokenTxType:    CustomTokenInit,
			Receiver:       paymentInfo,
			TokenInput:     []*privacy.InputCoin{},
			Mintable:       false,
			Fee:            0,
		}

		hasPrivacyForPRV := false
		hasPrivacyForToken := false

		paramToCreateTx1 := NewTxPrivacyTokenInitParams(&senderKey.KeySet.PrivateKey,
			paymentInfoPRV, inputCoinsPRV, 0, tokenParam, db, nil,
			hasPrivacyForPRV, hasPrivacyForToken, senderShardID, []byte{}, TxVersion2)

		// init tx
		tx1 := new(TxCustomTokenPrivacy)
		err = tx1.Init(paramToCreateTx1)
		assert.Equal(t, nil, err)

		isValid, err := tx1.ValidateTransaction(hasPrivacyForToken, db, senderShardID, nil)

		fmt.Printf("Error: %v\n", err)
		assert.Equal(t, true, isValid)
		assert.Equal(t, nil, err)

		tokenID := tx1.TxPrivacyTokenData.TxNormal.GetTokenID()
		tokenIDStr := tokenID.String()

		outputFromTx1 := tx1.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins()

		// store output coin's coin commitments in coin base tx
		db.StoreCommitments(
			*tokenID,
			senderPaymentAddress.Pk,
			[][]byte{outputFromTx1[0].CoinDetails.GetCoinCommitment().ToBytesS()},
			senderShardID)
		//db.StorePrivacyToken(tokenID, )

		/******** get output coins from coin base tx to create new tx ********/
		inputFromTx1 := ConvertOutputCoinToInputCoin(outputFromTx1)

		//fmt.Printf("inputFromTx1[0]GetValue: %v\n", inputFromTx1[0].CoinDetails.GetValue())
		//fmt.Printf("inputFromTx1[0]GetSNDerivatorRandom: %v\n", inputFromTx1[0].CoinDetails.GetSNDerivatorRandom())
		//fmt.Printf("inputFromTx1[0]GetCoinCommitment: %v\n", inputFromTx1[0].CoinDetails.GetCoinCommitment())
		//fmt.Printf("inputFromTx1[0]GetPrivRandOTA: %v\n", inputFromTx1[0].CoinDetails.GetPrivRandOTA())
		//fmt.Printf("inputFromTx1[0]GetRandomness: %v\n", inputFromTx1[0].CoinDetails.GetRandomness())
		//fmt.Printf("inputFromTx1[0]GetInfo: %v\n", inputFromTx1[0].CoinDetails.GetInfo())
		//fmt.Printf("inputFromTx1[0]GetPublicKey: %v\n", inputFromTx1[0].CoinDetails.GetPublicKey())
		//fmt.Printf("inputFromTx1[0]GetSerialNumber: %v\n", inputFromTx1[0].CoinDetails.GetSerialNumber())

		/******** init tx with mode no privacy ********/

		// calculate serial number for input coins from coin base tx
		serialNumber1 := new(privacy.Point).Derive(
			privacy.PedCom.G[privacy.PedersenPrivateKeyIndex],
			new(privacy.Scalar).FromBytesS(senderKey.KeySet.PrivateKey),
			inputFromTx1[0].CoinDetails.GetSNDerivatorRandom(),
		)
		inputFromTx1[0].CoinDetails.SetSerialNumber(serialNumber1)

		fmt.Println("********************* Tx2 ********************* ")

		// message to receiver
		msg2 := "Incognito-chain"
		receiverTK2, _ := new(privacy.Point).FromBytesS(senderKey.KeySet.PaymentAddress.Tk)
		msgCipherText2, _ := privacy.HybridEncrypt([]byte(msg2), receiverTK2)

		transferAmount2 := uint64(10000)
		paymentInfo2 := []*privacy.PaymentInfo{{PaymentAddress: senderKey.KeySet.PaymentAddress, Amount: transferAmount2, Message: msgCipherText2.Bytes()}}

		inputCoinsPRV2 := []*privacy.InputCoin{}
		paymentInfoPRV2 := []*privacy.PaymentInfo{}

		// token param for init new token
		tokenParam2 := &CustomTokenPrivacyParamTx{
			PropertyID:     tokenIDStr,
			PropertyName:   "Token 1",
			PropertySymbol: "Token 1",
			Amount:         transferAmount2,
			TokenTxType:    CustomTokenTransfer,
			Receiver:       paymentInfo2,
			TokenInput:     []*privacy.InputCoin{},
			Mintable:       false,
			Fee:            0,
		}

		hasPrivacyForPRV2 := false
		hasPrivacyForToken2 := false

		paramToCreateTx2 := NewTxPrivacyTokenInitParams(&senderKey.KeySet.PrivateKey,
			paymentInfoPRV2, inputCoinsPRV2, 0, tokenParam2, db, nil,
			hasPrivacyForPRV2, hasPrivacyForToken2, senderShardID, []byte{}, TxVersion2)

		// init tx
		tx2 := new(TxCustomTokenPrivacy)
		err = tx2.Init(paramToCreateTx2)
		assert.Equal(t, nil, err)

		//isValidSanity, err := tx2.ValidateSanityData(nil)
		//assert.Equal(t, true, isValidSanity)
		//assert.Equal(t, nil, err)

		isValid, err = tx2.ValidateTransaction(hasPrivacyForToken2, db, senderShardID, nil)

		fmt.Printf("Error: %v\n", err)
		assert.Equal(t, true, isValid)
		assert.Equal(t, nil, err)

		outputFromTx2 := tx2.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins()

		db.StoreCommitments(
			*tokenID,
			senderPaymentAddress.Pk,
			[][]byte{outputFromTx2[1].CoinDetails.GetCoinCommitment().ToBytesS()},
			senderShardID)

		outputFromTx2[1].Decrypt(senderKey.KeySet.ReadonlyKey)

		inputFromTx2 := ConvertOutputCoinToInputCoin([]*privacy.OutputCoin{outputFromTx2[1]})

		serialNumber2 := new(privacy.Point).Derive(
			privacy.PedCom.G[privacy.PedersenPrivateKeyIndex],
			new(privacy.Scalar).FromBytesS(senderKey.KeySet.PrivateKey),
			inputFromTx2[0].CoinDetails.GetSNDerivatorRandom(),
		)
		inputFromTx2[0].CoinDetails.SetSerialNumber(serialNumber2)

		// store output coin's coin commitments in coin base tx

		/******** get output coins from coin base tx to create new tx ********/
	}
}