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
			hasPrivacyForPRV, hasPrivacyForToken, shardID, []byte{})

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
			hasPrivacyForPRV, true, shardID, []byte{})

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
		receiverPubKey := receiverKeyWallet.KeySet.PaymentAddress.Pk

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

		paramToCreateTx1 := NewTxPrivacyTokenInitParams(&senderKey.KeySet.PrivateKey,
			paymentInfoPRV, inputCoinsPRV, 0, tokenParam, db, nil,
			hasPrivacyForPRV, hasPrivacyForToken, senderShardID, []byte{})

		// init tx
		tx1 := new(TxCustomTokenPrivacy)
		err = tx1.Init(paramToCreateTx1)
		assert.Equal(t, nil, err)

		tokenID := tx1.TxPrivacyTokenData.TxNormal.GetTokenID()

		// store output coin's coin commitments in coin base tx
		db.StoreCommitments(
			*tokenID,
			senderPaymentAddress.Pk,
			[][]byte{tx1.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins()[0].CoinDetails.GetCoinCommitment().ToBytesS()},
			senderShardID)

		/******** get output coins from coin base tx to create new tx ********/
		coinBaseOutput := ConvertOutputCoinToInputCoin(tx1.TxPrivacyTokenData.TxNormal.Proof.GetOutputCoins())

		//fmt.Printf("coinBaseOutput[0]GetValue: %v\n", coinBaseOutput[0].CoinDetails.GetValue())
		//fmt.Printf("coinBaseOutput[0]GetSNDerivatorRandom: %v\n", coinBaseOutput[0].CoinDetails.GetSNDerivatorRandom())
		//fmt.Printf("coinBaseOutput[0]GetCoinCommitment: %v\n", coinBaseOutput[0].CoinDetails.GetCoinCommitment())
		//fmt.Printf("coinBaseOutput[0]GetPrivRandOTA: %v\n", coinBaseOutput[0].CoinDetails.GetPrivRandOTA())
		//fmt.Printf("coinBaseOutput[0]GetRandomness: %v\n", coinBaseOutput[0].CoinDetails.GetRandomness())
		//fmt.Printf("coinBaseOutput[0]GetInfo: %v\n", coinBaseOutput[0].CoinDetails.GetInfo())
		//fmt.Printf("coinBaseOutput[0]GetPublicKey: %v\n", coinBaseOutput[0].CoinDetails.GetPublicKey())
		//fmt.Printf("coinBaseOutput[0]GetSerialNumber: %v\n", coinBaseOutput[0].CoinDetails.GetSerialNumber())

		/******** init tx with mode no privacy ********/
		fmt.Println("********************* Tx2 ********************* ")
		tx2 := new(TxCustomTokenPrivacy)
		// calculate serial number for input coins from coin base tx
		serialNumber := new(privacy.Point).Derive(
			privacy.PedCom.G[privacy.PedersenPrivateKeyIndex],
			new(privacy.Scalar).FromBytesS(senderKey.KeySet.PrivateKey),
			coinBaseOutput[0].CoinDetails.GetSNDerivatorRandom(),
		)
		coinBaseOutput[0].CoinDetails.SetSerialNumber(serialNumber)

		// transfer amount
		transferAmount := 5
		hasPrivacy := true
		fee := 1

		// message to receiver
		msg := "Incognito-chain"
		receiverTK, _ := new(privacy.Point).FromBytesS(senderKey.KeySet.PaymentAddress.Tk)
		msgCipherText, _ := privacy.HybridEncrypt([]byte(msg), receiverTK)

		//fmt.Printf("msgCipherText: %v - len : %v\n", msgCipherText.Bytes(), len(msgCipherText.Bytes()))
		err = tx2.Init(
			NewTxPrivacyTokenInitParams(
				&senderKey.KeySet.PrivateKey,
				paymentInfoPRV, inputCoinsPRV, 0, tokenParam, db, nil,
				hasPrivacyForPRV, hasPrivacyForToken, senderShardID, []byte{},
			),
		)
		if err != nil {
			t.Error(err)
		}

		outputs := tx1.GetProof().GetOutputCoins()
		fmt.Printf("%v\n", len(outputs))
		for i := 0; i < len(outputs); i++ {
			fmt.Printf("outputs[i].CoinDetails.GetValue(): %v\n", outputs[i].CoinDetails.GetValue())
			fmt.Printf("outputs[i].CoinDetails.GetSNDerivatorRandom(): %v\n", outputs[i].CoinDetails.GetSNDerivatorRandom())
			fmt.Printf("outputs[i].CoinDetails.GetPublicKey(): %v\n", outputs[i].CoinDetails.GetPublicKey())
			fmt.Printf("outputs[i].CoinDetails.GetPrivRandOTA(): %v\n", outputs[i].CoinDetails.GetPrivRandOTA())
		}

		assert.Equal(t, len(msgCipherText.Bytes()), len(tx1.Proof.GetOutputCoins()[0].CoinDetails.GetInfo()))

		if hasPrivacy {
			assert.NotEqual(t, receiverPubKey, outputs[0].CoinDetails.GetPublicKey())
		} else {
			assert.Equal(t, receiverPubKey, outputs[0].CoinDetails.GetPublicKey())
		}

		isValidSanity, err = tx1.ValidateSanityData(nil)
		assert.Equal(t, true, isValidSanity)
		assert.Equal(t, nil, err)

		isValid, err := tx1.ValidateTransaction(hasPrivacy, db, senderShardID, nil)

		fmt.Printf("Error: %v\n", err)
		assert.Equal(t, true, isValid)
		assert.Equal(t, nil, err)

		outputs[1].Decrypt(senderKey.KeySet.ReadonlyKey)

		newOutput := ConvertOutputCoinToInputCoin([]*privacy.OutputCoin{outputs[1]})

		// store output coin's coin commitments from tx1
		db.StoreCommitments(
			common.PRVCoinID,
			senderPaymentAddress.Pk,
			[][]byte{newOutput[0].CoinDetails.GetCoinCommitment().ToBytesS()},
			senderShardID)

		fmt.Printf("newOutput[0].CoinDetails.GetCoinCommitment().ToBytesS(): %v\n", newOutput[0].CoinDetails.GetCoinCommitment().ToBytesS())

		/******** init tx with mode privacy from privacy input ********/
		fmt.Println("********************* Tx2 ********************* ")
		tx2 := Tx{}
		// prepare input: calculate SN, PrivRandOTA, Value, Randomness
		// calculate privRandOTA
		fmt.Printf("AA tx1.Proof.GetEphemeralPubKey(): %v\n", tx1.Proof.GetEphemeralPubKey())
		fmt.Printf("AAA newOutput[0].CoinDetails.GetPublicKey(): %v\n", newOutput[0].CoinDetails.GetPublicKey())
		isPair, privRandOTA, err := privacy.IsPairOneTimeAddr(newOutput[0].CoinDetails.GetPublicKey(), tx1.Proof.GetEphemeralPubKey(), senderKey.KeySet.ReadonlyKey, 1)
		fmt.Printf("err check one time address: %v\n", err)
		assert.Equal(t, true, isPair)
		newOutput[0].CoinDetails.SetPrivRandOTA(privRandOTA)
		fmt.Printf("privRandOTA : %v\n", privRandOTA)
		fmt.Printf("newOutput[0].CoinDetails.GetPrivRandOTA() : %v\n", newOutput[0].CoinDetails.GetPrivRandOTA())

		// calculate serial number for input coins from coin base tx
		serialNumber2 := new(privacy.Point).Derive(
			privacy.PedCom.G[privacy.PedersenPrivateKeyIndex],
			new(privacy.Scalar).FromBytesS(senderKey.KeySet.PrivateKey),
			newOutput[0].CoinDetails.GetPrivRandOTA(),
		)
		newOutput[0].CoinDetails.SetSerialNumber(serialNumber2)

		// decrypt Value, Randomness

		// transfer amount
		transferAmount2 := 5
		hasPrivacy2 := true
		fee2 := 1

		// message to receiver
		msg2 := "Incognito-chain"
		receiverTK2, _ := new(privacy.Point).FromBytesS(senderKey.KeySet.PaymentAddress.Tk)
		msgCipherText2, _ := privacy.HybridEncrypt([]byte(msg2), receiverTK2)

		//fmt.Printf("msgCipherText: %v - len : %v\n", msgCipherText.Bytes(), len(msgCipherText.Bytes()))
		err = tx2.Init(
			NewTxPrivacyInitParams(
				&senderKey.KeySet.PrivateKey,
				[]*privacy.PaymentInfo{{PaymentAddress: receiverKeyWallet.KeySet.PaymentAddress, Amount: uint64(transferAmount2), Message: msgCipherText2.Bytes()}},
				newOutput, uint64(fee2), hasPrivacy2, db, nil, nil, []byte{}, TxVersion2,
			),
		)
		if err != nil {
			t.Error(err)
		}

		outputs2 := tx2.GetProof().GetOutputCoins()
		fmt.Printf("%v\n", len(outputs2))
		for i := 0; i < len(outputs2); i++ {
			fmt.Printf("outputs[i].CoinDetails.GetValue(): %v\n", outputs2[i].CoinDetails.GetValue())
			fmt.Printf("outputs[i].CoinDetails.GetSNDerivatorRandom(): %v\n", outputs2[i].CoinDetails.GetSNDerivatorRandom())
			fmt.Printf("outputs[i].CoinDetails.GetPublicKey(): %v\n", outputs2[i].CoinDetails.GetPublicKey())
			fmt.Printf("outputs[i].CoinDetails.GetPrivRandOTA(): %v\n", outputs2[i].CoinDetails.GetPrivRandOTA())
		}

		assert.Equal(t, len(msgCipherText2.Bytes()), len(tx2.Proof.GetOutputCoins()[0].CoinDetails.GetInfo()))

		if hasPrivacy2 {
			assert.NotEqual(t, receiverPubKey, outputs2[0].CoinDetails.GetPublicKey())
		} else {
			assert.Equal(t, receiverPubKey, outputs2[0].CoinDetails.GetPublicKey())
		}

		isValidSanity2, err := tx2.ValidateSanityData(nil)
		assert.Equal(t, true, isValidSanity2)
		assert.Equal(t, nil, err)

		isValid2, err := tx2.ValidateTransaction(hasPrivacy2, db, senderShardID, nil)

		fmt.Printf("Error: %v\n", err)
		assert.Equal(t, true, isValid2)
		assert.Equal(t, nil, err)

		outputs2[1].Decrypt(senderKey.KeySet.ReadonlyKey)

		outputFromTx2 := ConvertOutputCoinToInputCoin([]*privacy.OutputCoin{outputs2[1]})

		// store output coin's coin commitments from tx1
		db.StoreCommitments(
			common.PRVCoinID,
			senderPaymentAddress.Pk,
			[][]byte{outputFromTx2[0].CoinDetails.GetCoinCommitment().ToBytesS()},
			senderShardID)

		/******** init tx with mode no privacy from privacy input ********/
		fmt.Println("********************* Tx3 ********************* ")
		tx3 := Tx{}
		// prepare input: calculate SN, PrivRandOTA, Value, Randomness
		// calculate privRandOTA
		fmt.Printf("AA tx1.Proof.GetEphemeralPubKey(): %v\n", tx2.Proof.GetEphemeralPubKey())
		fmt.Printf("AAA newOutput[0].CoinDetails.GetPublicKey(): %v\n", outputFromTx2[0].CoinDetails.GetPublicKey())
		isPair3, privRandOTA3, err := privacy.IsPairOneTimeAddr(outputFromTx2[0].CoinDetails.GetPublicKey(), tx2.Proof.GetEphemeralPubKey(), senderKey.KeySet.ReadonlyKey, 1)
		fmt.Printf("err check one time address: %v\n", err)
		assert.Equal(t, true, isPair3)
		outputFromTx2[0].CoinDetails.SetPrivRandOTA(privRandOTA3)
		fmt.Printf("privRandOTA : %v\n", privRandOTA3)
		fmt.Printf("newOutput[0].CoinDetails.GetPrivRandOTA() : %v\n", outputFromTx2[0].CoinDetails.GetPrivRandOTA())

		// calculate serial number for input coins from coin base tx
		serialNumber3 := new(privacy.Point).Derive(
			privacy.PedCom.G[privacy.PedersenPrivateKeyIndex],
			new(privacy.Scalar).FromBytesS(senderKey.KeySet.PrivateKey),
			outputFromTx2[0].CoinDetails.GetPrivRandOTA(),
		)
		outputFromTx2[0].CoinDetails.SetSerialNumber(serialNumber3)

		// decrypt Value, Randomness

		// transfer amount
		transferAmount3 := 5
		hasPrivacy3 := false
		fee3 := 1

		// message to receiver
		msg3 := "Incognito-chain"
		receiverTK3, _ := new(privacy.Point).FromBytesS(senderKey.KeySet.PaymentAddress.Tk)
		msgCipherText3, _ := privacy.HybridEncrypt([]byte(msg3), receiverTK3)

		//fmt.Printf("msgCipherText: %v - len : %v\n", msgCipherText.Bytes(), len(msgCipherText.Bytes()))
		err = tx3.Init(
			NewTxPrivacyInitParams(
				&senderKey.KeySet.PrivateKey,
				[]*privacy.PaymentInfo{{PaymentAddress: receiverKeyWallet.KeySet.PaymentAddress, Amount: uint64(transferAmount3), Message: msgCipherText3.Bytes()}},
				outputFromTx2, uint64(fee3), hasPrivacy3, db, nil, nil, []byte{}, TxVersion2,
			),
		)
		if err != nil {
			t.Error(err)
		}

		outputs3 := tx3.GetProof().GetOutputCoins()
		fmt.Printf("%v\n", len(outputs3))
		for i := 0; i < len(outputs3); i++ {
			fmt.Printf("outputs[i].CoinDetails.GetValue(): %v\n", outputs3[i].CoinDetails.GetValue())
			fmt.Printf("outputs[i].CoinDetails.GetSNDerivatorRandom(): %v\n", outputs3[i].CoinDetails.GetSNDerivatorRandom())
			fmt.Printf("outputs[i].CoinDetails.GetPublicKey(): %v\n", outputs3[i].CoinDetails.GetPublicKey())
			fmt.Printf("outputs[i].CoinDetails.GetPrivRandOTA(): %v\n", outputs3[i].CoinDetails.GetPrivRandOTA())
		}

		assert.Equal(t, len(msgCipherText3.Bytes()), len(tx3.Proof.GetOutputCoins()[0].CoinDetails.GetInfo()))

		if hasPrivacy2 {
			assert.NotEqual(t, receiverPubKey, outputs3[0].CoinDetails.GetPublicKey())
		} else {
			assert.Equal(t, receiverPubKey, outputs3[0].CoinDetails.GetPublicKey())
		}

		isValidSanity3, err := tx3.ValidateSanityData(nil)
		assert.Equal(t, true, isValidSanity3)
		assert.Equal(t, nil, err)

		isValid3, err := tx3.ValidateTransaction(hasPrivacy3, db, senderShardID, nil)

		fmt.Printf("Error: %v\n", err)
		assert.Equal(t, true, isValid3)
		assert.Equal(t, nil, err)

		//outputs3[1].Decrypt(senderKey.KeySet.ReadonlyKey)

		outputFromTx3 := ConvertOutputCoinToInputCoin([]*privacy.OutputCoin{outputs3[1]})

		// store output coin's coin commitments from tx1
		db.StoreCommitments(
			common.PRVCoinID,
			senderPaymentAddress.Pk,
			[][]byte{outputFromTx3[0].CoinDetails.GetCoinCommitment().ToBytesS()},
			senderShardID)

		/******** init tx with mode no privacy from privacy input ********/
		fmt.Println("********************* Tx3 ********************* ")
		tx4 := Tx{}
		// prepare input: calculate SN, PrivRandOTA, Value, Randomness

		// calculate serial number for input coins from coin base tx
		serialNumber4 := new(privacy.Point).Derive(
			privacy.PedCom.G[privacy.PedersenPrivateKeyIndex],
			new(privacy.Scalar).FromBytesS(senderKey.KeySet.PrivateKey),
			outputFromTx3[0].CoinDetails.GetSNDerivatorRandom(),
		)
		outputFromTx3[0].CoinDetails.SetSerialNumber(serialNumber4)

		// decrypt Value, Randomness

		// transfer amount
		transferAmount4 := 5
		hasPrivacy4 := false
		fee4 := 1

		// message to receiver
		msg4 := "Incognito-chain"
		receiverTK4, _ := new(privacy.Point).FromBytesS(senderKey.KeySet.PaymentAddress.Tk)
		msgCipherText4, _ := privacy.HybridEncrypt([]byte(msg4), receiverTK4)

		//fmt.Printf("msgCipherText: %v - len : %v\n", msgCipherText.Bytes(), len(msgCipherText.Bytes()))
		err = tx4.Init(
			NewTxPrivacyInitParams(
				&senderKey.KeySet.PrivateKey,
				[]*privacy.PaymentInfo{{PaymentAddress: receiverKeyWallet.KeySet.PaymentAddress, Amount: uint64(transferAmount4), Message: msgCipherText4.Bytes()}},
				outputFromTx3, uint64(fee4), hasPrivacy4, db, nil, nil, []byte{}, TxVersion2,
			),
		)
		if err != nil {
			t.Error(err)
		}

		outputs4 := tx4.GetProof().GetOutputCoins()
		fmt.Printf("%v\n", len(outputs4))
		for i := 0; i < len(outputs4); i++ {
			fmt.Printf("outputs[i].CoinDetails.GetValue(): %v\n", outputs4[i].CoinDetails.GetValue())
			fmt.Printf("outputs[i].CoinDetails.GetSNDerivatorRandom(): %v\n", outputs4[i].CoinDetails.GetSNDerivatorRandom())
			fmt.Printf("outputs[i].CoinDetails.GetPublicKey(): %v\n", outputs4[i].CoinDetails.GetPublicKey())
			fmt.Printf("outputs[i].CoinDetails.GetPrivRandOTA(): %v\n", outputs4[i].CoinDetails.GetPrivRandOTA())
		}

		assert.Equal(t, len(msgCipherText4.Bytes()), len(tx4.Proof.GetOutputCoins()[0].CoinDetails.GetInfo()))

		//if hasPrivacy4 {
		//	assert.NotEqual(t, receiverPubKey, outputs4[0].CoinDetails.GetPublicKey())
		//} else{
		//	assert.Equal(t, receiverPubKey, outputs4[0].CoinDetails.GetPublicKey())
		//}

		isValidSanity4, err := tx4.ValidateSanityData(nil)
		assert.Equal(t, true, isValidSanity4)
		assert.Equal(t, nil, err)

		isValid4, err := tx4.ValidateTransaction(hasPrivacy4, db, senderShardID, nil)

		fmt.Printf("Error: %v\n", err)
		assert.Equal(t, true, isValid4)
		assert.Equal(t, nil, err)