package transaction

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestUnmarshalJSON(t *testing.T) {
	key, err := wallet.Base58CheckDeserialize("112t8rnXCqbbNYBquntyd6EvDT4WiDDQw84ZSRDKmazkqrzi6w8rWyCVt7QEZgAiYAV4vhJiX7V9MCfuj4hGLoDN7wdU1LoWGEFpLs59X7K3")
	assert.Equal(t, nil, err)
	err = key.KeySet.InitFromPrivateKey(&key.KeySet.PrivateKey)
	assert.Equal(t, nil, err)
	paymentAddress := key.KeySet.PaymentAddress
	responseMeta, err := metadata.NewWithDrawRewardResponse(&common.Hash{})
	tx, err := BuildCoinBaseTxByCoinID(NewBuildCoinBaseTxByCoinIDParams(&paymentAddress, 10, &key.KeySet.PrivateKey, db, responseMeta, common.Hash{}, NormalCoinType, "PRV", 0))
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, tx)
	assert.Equal(t, uint64(10), tx.(*Tx).Proof.GetOutputCoins()[0].CoinDetails.GetValue())
	assert.Equal(t, common.PRVCoinID.String(), tx.GetTokenID().String())

	jsonStr, err := json.Marshal(tx)
	assert.Equal(t, nil, err)
	fmt.Println(string(jsonStr))

	tx1 := Tx{}
	//err = json.Unmarshal(jsonStr, &tx1)
	err = tx1.UnmarshalJSON(jsonStr)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint64(10), tx1.Proof.GetOutputCoins()[0].CoinDetails.GetValue())
	assert.Equal(t, common.PRVCoinID.String(), tx1.GetTokenID().String())
}

func TestInitTx(t *testing.T) {
	for i := 0; i < 1; i++ {
		//Generate sender private key & receiver payment address
		seed := privacy.RandomScalar().ToBytesS()
		masterKey, _ := wallet.NewMasterKey(seed)
		childSender, _ := masterKey.NewChildKey(uint32(1))
		privKeyB58 := childSender.Base58CheckSerialize(wallet.PriKeyType)
		childReceiver, _ := masterKey.NewChildKey(uint32(2))
		paymentAddressB58 := childReceiver.Base58CheckSerialize(wallet.PaymentAddressType)

		senderKey, err := wallet.Base58CheckDeserialize(privKeyB58)
		assert.Equal(t, nil, err)

		err = senderKey.KeySet.InitFromPrivateKey(&senderKey.KeySet.PrivateKey)
		assert.Equal(t, nil, err)

		senderPaymentAddress := senderKey.KeySet.PaymentAddress
		senderPublicKey := senderPaymentAddress.Pk

		shardID := common.GetShardIDFromLastByte(senderKey.KeySet.PaymentAddress.Pk[len(senderKey.KeySet.PaymentAddress.Pk)-1])

		// coin base tx to mint PRV
		mintedAmount := 1000
		coinBaseTx, err := BuildCoinBaseTxByCoinID(NewBuildCoinBaseTxByCoinIDParams(&senderPaymentAddress, uint64(mintedAmount), &senderKey.KeySet.PrivateKey, db, nil, common.Hash{}, NormalCoinType, "PRV", 0))

		isValidSanity, err := coinBaseTx.ValidateSanityData(nil)
		assert.Equal(t, nil, err)
		assert.Equal(t, true, isValidSanity)

		// store output coin's coin commitments in coin base tx
		db.StoreCommitments(
			common.PRVCoinID,
			senderPaymentAddress.Pk,
			[][]byte{coinBaseTx.(*Tx).Proof.GetOutputCoins()[0].CoinDetails.GetCoinCommitment().ToBytesS()},
			shardID)

		// get output coins from coin base tx to create new tx
		coinBaseOutput := ConvertOutputCoinToInputCoin(coinBaseTx.(*Tx).Proof.GetOutputCoins())

		// init new tx without privacy
		tx1 := Tx{}
		// calculate serial number for input coins
		serialNumber := new(privacy.Point).Derive(privacy.PedCom.G[privacy.PedersenPrivateKeyIndex],
			new(privacy.Scalar).FromBytesS(senderKey.KeySet.PrivateKey),
			coinBaseOutput[0].CoinDetails.GetSNDerivatorRandom())

		coinBaseOutput[0].CoinDetails.SetSerialNumber(serialNumber)

		receiverPaymentAddress, _ := wallet.Base58CheckDeserialize(paymentAddressB58)

		// transfer amount
		transferAmount := 5
		hasPrivacy := false
		fee := 1

		// message to receiver
		msg := "Incognito-chain"
		receiverTK, _ := new(privacy.Point).FromBytesS(senderKey.KeySet.PaymentAddress.Tk)
		msgCipherText, _ := privacy.HybridEncrypt([]byte(msg), receiverTK)

		fmt.Printf("msgCipherText: %v - len : %v\n", msgCipherText.Bytes(), len(msgCipherText.Bytes()))
		err = tx1.Init(
			NewTxPrivacyInitParams(
				&senderKey.KeySet.PrivateKey,
				[]*privacy.PaymentInfo{{PaymentAddress: receiverPaymentAddress.KeySet.PaymentAddress, Amount: uint64(transferAmount), Message: msgCipherText.Bytes()}},
				coinBaseOutput, uint64(fee), hasPrivacy, db, nil, nil, []byte{}, TxVersion2,
			),
		)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, len(msgCipherText.Bytes()), len(tx1.Proof.GetOutputCoins()[0].CoinDetails.GetInfo()))

		actualSize := tx1.GetTxActualSize()
		fmt.Printf("actualSize: %v\n", actualSize)

		senderPubKeyLastByte := tx1.GetSenderAddrLastByte()
		assert.Equal(t, senderKey.KeySet.PaymentAddress.Pk[len(senderKey.KeySet.PaymentAddress.Pk)-1], senderPubKeyLastByte)

		actualFee := tx1.GetTxFee()
		assert.Equal(t, uint64(fee), actualFee)

		actualFeeToken := tx1.GetTxFeeToken()
		assert.Equal(t, uint64(0), actualFeeToken)

		unique, pubk, amount := tx1.GetUniqueReceiver()
		assert.Equal(t, true, unique)
		assert.Equal(t, string(pubk[:]), string(receiverPaymentAddress.KeySet.PaymentAddress.Pk[:]))
		assert.Equal(t, uint64(5), amount)

		unique, pubk, amount, coinID := tx1.GetTransferData()
		assert.Equal(t, true, unique)
		assert.Equal(t, common.PRVCoinID.String(), coinID.String())
		assert.Equal(t, string(pubk[:]), string(receiverPaymentAddress.KeySet.PaymentAddress.Pk[:]))

		a, b := tx1.GetTokenReceivers()
		assert.Equal(t, 0, len(a))
		assert.Equal(t, 0, len(b))

		e, d, c := tx1.GetTokenUniqueReceiver()
		assert.Equal(t, false, e)
		assert.Equal(t, 0, len(d))
		assert.Equal(t, uint64(0), c)

		listInputSerialNumber := tx1.ListSerialNumbersHashH()
		assert.Equal(t, 1, len(listInputSerialNumber))
		assert.Equal(t, common.HashH(coinBaseOutput[0].CoinDetails.GetSerialNumber().ToBytesS()), listInputSerialNumber[0])

		isValidSanity, err = tx1.ValidateSanityData(nil)
		assert.Equal(t, true, isValidSanity)
		assert.Equal(t, nil, err)

		isValid, err := tx1.ValidateTransaction(hasPrivacy, db, shardID, nil)

		fmt.Printf("Error: %v\n", err)
		assert.Equal(t, true, isValid)
		assert.Equal(t, nil, err)

		isValidTxVersion := tx1.CheckTxVersion(1)
		assert.Equal(t, true, isValidTxVersion)

		//isValidTxFee := tx1.CheckTransactionFee(0)
		//assert.Equal(t, true, isValidTxFee)

		isSalaryTx := tx1.IsSalaryTx()
		assert.Equal(t, false, isSalaryTx)

		actualSenderPublicKey := tx1.GetSender()
		expectedSenderPublicKey := make([]byte, common.PublicKeySize)
		copy(expectedSenderPublicKey, senderPublicKey[:])
		assert.Equal(t, expectedSenderPublicKey, actualSenderPublicKey[:])

		//err = tx1.ValidateTxWithCurrentMempool(nil)
		//	assert.Equal(t, nil, err)

		err = tx1.ValidateDoubleSpendWithBlockchain(nil, shardID, db, nil)
		assert.Equal(t, nil, err)

		err = tx1.ValidateTxWithBlockChain(nil, shardID, db)
		assert.Equal(t, nil, err)

		isValid, err = tx1.ValidateTxByItself(hasPrivacy, db, nil, shardID)
		assert.Equal(t, nil, err)
		assert.Equal(t, true, isValid)

		metaDataType := tx1.GetMetadataType()
		assert.Equal(t, metadata.InvalidMeta, metaDataType)

		metaData := tx1.GetMetadata()
		assert.Equal(t, nil, metaData)

		info := tx1.GetInfo()
		assert.Equal(t, 0, len(info))

		lockTime := tx1.GetLockTime()
		now := time.Now().Unix()
		assert.LessOrEqual(t, lockTime, now)

		actualSigPubKey := tx1.GetSigPubKey()
		assert.Equal(t, expectedSenderPublicKey, actualSigPubKey)

		proof := tx1.GetProof()
		assert.NotEqual(t, nil, proof)

		isValidTxType := tx1.ValidateType()
		assert.Equal(t, true, isValidTxType)

		isCoinsBurningTx := tx1.IsCoinsBurning()
		assert.Equal(t, false, isCoinsBurningTx)

		actualTxValue := tx1.CalculateTxValue()
		assert.Equal(t, uint64(transferAmount), actualTxValue)

		// store output coin's coin commitments in tx1
		//for i:=0; i < len(tx1.Proof.GetOutputCoins()); i++ {
		//	db.StoreCommitments(
		//		common.PRVCoinID,
		//		tx1.Proof.GetOutputCoins()[i].CoinDetails.GetPublicKey().Compress(),
		//		[][]byte{tx1.Proof.GetOutputCoins()[i].CoinDetails.GetCoinCommitment().Compress()},
		//		shardID)
		//}

		// init tx with privacy
		//tx2 := Tx{}
		//
		//err = tx2.Init(
		//	NewTxPrivacyInitParams(
		//		&senderKey.KeySet.PrivateKey,
		//		[]*privacy.PaymentInfo{{PaymentAddress: senderPaymentAddress, Amount: uint64(transferAmount)}},
		//		coinBaseOutput, 1, true, db, nil, nil, []byte{}))
		//if err != nil {
		//	t.Error(err)
		//}
		//
		//isValidSanity, err = tx2.ValidateSanityData(nil)
		//assert.Equal(t, nil, err)
		//assert.Equal(t, true, isValidSanity)
		//
		//isValidTx, err := tx2.ValidateTransaction(true, db, shardID, &common.PRVCoinID)
		//assert.Equal(t, true, isValidTx)

	}
}

func TestInitTxWithMultiScenario(t *testing.T) {
	for i := 0; i < 50; i++ {
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

		senderPaymentAddress := senderKey.KeySet.PaymentAddress

		//receiver key
		receiverKey, _ := wallet.Base58CheckDeserialize(paymentAddressB58)
		receiverPaymentAddress := receiverKey.KeySet.PaymentAddress

		// shard ID of sender
		shardID := common.GetShardIDFromLastByte(senderKey.KeySet.PaymentAddress.Pk[len(senderKey.KeySet.PaymentAddress.Pk)-1])

		// create coin base tx to mint PRV
		mintedAmount := 1000
		coinBaseTx, err := BuildCoinBaseTxByCoinID(NewBuildCoinBaseTxByCoinIDParams(&senderPaymentAddress, uint64(mintedAmount), &senderKey.KeySet.PrivateKey, db, nil, common.Hash{}, NormalCoinType, "PRV", 0))

		isValidSanity, err := coinBaseTx.ValidateSanityData(nil)
		assert.Equal(t, nil, err)
		assert.Equal(t, true, isValidSanity)

		// store output coin's coin commitments in coin base tx
		db.StoreCommitments(
			common.PRVCoinID,
			senderPaymentAddress.Pk,
			[][]byte{coinBaseTx.(*Tx).Proof.GetOutputCoins()[0].CoinDetails.GetCoinCommitment().ToBytesS()},
			shardID)

		// get output coins from coin base tx to create new tx
		coinBaseOutput := ConvertOutputCoinToInputCoin(coinBaseTx.(*Tx).Proof.GetOutputCoins())

		// init new tx with privacy
		tx1 := Tx{}
		// calculate serial number for input coins
		serialNumber := new(privacy.Point).Derive(privacy.PedCom.G[privacy.PedersenPrivateKeyIndex],
			new(privacy.Scalar).FromBytesS(senderKey.KeySet.PrivateKey),
			coinBaseOutput[0].CoinDetails.GetSNDerivatorRandom())

		coinBaseOutput[0].CoinDetails.SetSerialNumber(serialNumber)

		// transfer amount
		transferAmount := 5
		hasPrivacy := true
		fee := 1
		err = tx1.Init(
			NewTxPrivacyInitParams(
				&senderKey.KeySet.PrivateKey,
				[]*privacy.PaymentInfo{{PaymentAddress: receiverPaymentAddress, Amount: uint64(transferAmount)}},
				coinBaseOutput, uint64(fee), hasPrivacy, db, nil, nil, []byte{}, TxVersion2,
			),
		)
		assert.Equal(t, nil, err)

		isValidSanity, err = tx1.ValidateSanityData(nil)
		assert.Equal(t, true, isValidSanity)
		assert.Equal(t, nil, err)
		fmt.Println("Hello")
		isValid, err := tx1.ValidateTransaction(hasPrivacy, db, shardID, nil)
		assert.Equal(t, true, isValid)
		assert.Equal(t, nil, err)
		fmt.Println("Hello")
		err = tx1.ValidateDoubleSpendWithBlockchain(nil, shardID, db, nil)
		assert.Equal(t, nil, err)

		err = tx1.ValidateTxWithBlockChain(nil, shardID, db)
		assert.Equal(t, nil, err)

		isValid, err = tx1.ValidateTxByItself(hasPrivacy, db, nil, shardID)
		assert.Equal(t, nil, err)
		assert.Equal(t, true, isValid)

		// modify Sig
		tx1.Sig[len(tx1.Sig)-1] = tx1.Sig[len(tx1.Sig)-1] ^ tx1.Sig[0]
		tx1.Sig[len(tx1.Sig)-2] = tx1.Sig[len(tx1.Sig)-2] ^ tx1.Sig[1]
		isValid, err = tx1.ValidateTransaction(hasPrivacy, db, shardID, nil)
		assert.Equal(t, false, isValid)
		assert.NotEqual(t, nil, err)
		tx1.Sig[len(tx1.Sig)-1] = tx1.Sig[len(tx1.Sig)-1] ^ tx1.Sig[0]
		tx1.Sig[len(tx1.Sig)-2] = tx1.Sig[len(tx1.Sig)-2] ^ tx1.Sig[1]

		// modify verification key
		tx1.SigPubKey[len(tx1.SigPubKey)-1] = tx1.SigPubKey[len(tx1.SigPubKey)-1] ^ tx1.SigPubKey[0]
		tx1.SigPubKey[len(tx1.SigPubKey)-2] = tx1.SigPubKey[len(tx1.SigPubKey)-2] ^ tx1.SigPubKey[1]

		isValid, err = tx1.ValidateTransaction(hasPrivacy, db, shardID, nil)
		assert.Equal(t, false, isValid)
		assert.NotEqual(t, nil, err)

		tx1.SigPubKey[len(tx1.SigPubKey)-1] = tx1.SigPubKey[len(tx1.SigPubKey)-1] ^ tx1.SigPubKey[0]
		tx1.SigPubKey[len(tx1.SigPubKey)-2] = tx1.SigPubKey[len(tx1.SigPubKey)-2] ^ tx1.SigPubKey[1]

		// modify proof
		originProof := tx1.Proof.Bytes()

		//var modifiedProof = make ([]byte, len(originProof))
		//copy(modifiedProof, originProof)
		//modifiedProof[7] = modifiedProof[8]
		//modifiedProof[5] = modifiedProof[6]
		//modifiedProof[6] = modifiedProof[16]
		//
		//tx1.Proof.SetBytes(modifiedProof)
		//
		//isValid, err = tx1.ValidateTransaction(hasPrivacy, db, shardID, nil)
		//assert.Equal(t, false, isValid)
		//assert.NotEqual(t, nil, err)

		tx1.Proof.SetBytes(originProof)

		// back to correct case
		isValid, err = tx1.ValidateTxByItself(hasPrivacy, db, nil, shardID)
		assert.Equal(t, nil, err)
		assert.Equal(t, true, isValid)
	}
}

func TestInitSalaryTx(t *testing.T) {
	salary := uint64(1000)

	privateKey := privacy.GeneratePrivateKey([]byte{123})
	senderKey := new(wallet.KeyWallet)
	err := senderKey.KeySet.InitFromPrivateKey(&privateKey)
	assert.Equal(t, nil, err)

	senderPaymentAddress := senderKey.KeySet.PaymentAddress
	receiverAddr := senderPaymentAddress

	tx := new(Tx)
	err = tx.InitTxSalary(salary, &receiverAddr, &senderKey.KeySet.PrivateKey, db, nil)
	assert.Equal(t, nil, err)

	isValid, err := tx.ValidateTxSalary(db)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, isValid)

	isSalaryTx := tx.IsSalaryTx()
	assert.Equal(t, true, isSalaryTx)
}

type CoinObject struct {
	PublicKey      string
	CoinCommitment string
	SNDerivator    string
	SerialNumber   string
	Randomness     string
	Value          uint64
	Info           string
}

func ParseCoinObjectToStruct(coinObjects []CoinObject) ([]*privacy.InputCoin, uint64) {
	coins := make([]*privacy.InputCoin, len(coinObjects))
	sumValue := uint64(0)

	for i := 0; i < len(coins); i++ {

		publicKey, _, _ := base58.Base58Check{}.Decode(coinObjects[i].PublicKey)
		publicKeyPoint := new(privacy.Point)
		publicKeyPoint.FromBytesS(publicKey)

		coinCommitment, _, _ := base58.Base58Check{}.Decode(coinObjects[i].CoinCommitment)
		coinCommitmentPoint := new(privacy.Point)
		coinCommitmentPoint.FromBytesS(coinCommitment)

		snd, _, _ := base58.Base58Check{}.Decode(coinObjects[i].SNDerivator)
		sndBN := new(privacy.Scalar).FromBytesS(snd)

		serialNumber, _, _ := base58.Base58Check{}.Decode(coinObjects[i].SerialNumber)
		serialNumberPoint := new(privacy.Point)
		serialNumberPoint.FromBytesS(serialNumber)

		randomness, _, _ := base58.Base58Check{}.Decode(coinObjects[i].Randomness)
		randomnessBN := new(privacy.Scalar).FromBytesS(randomness)

		coins[i] = new(privacy.InputCoin).Init()
		coins[i].CoinDetails.SetPublicKey(publicKeyPoint)
		coins[i].CoinDetails.SetCoinCommitment(coinCommitmentPoint)
		coins[i].CoinDetails.SetSNDerivatorRandom(sndBN)
		coins[i].CoinDetails.SetSerialNumber(serialNumberPoint)
		coins[i].CoinDetails.SetRandomness(randomnessBN)
		coins[i].CoinDetails.SetValue(coinObjects[i].Value)

		sumValue += coinObjects[i].Value

	}

	return coins, sumValue
}

func TestInitTxV1(t *testing.T) {
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

		/******** coin base tx to mint PRV ********/
		mintedAmount := 1000
		coinBaseTx, err := BuildCoinBaseTxByCoinID(NewBuildCoinBaseTxByCoinIDParams(&senderPaymentAddress, uint64(mintedAmount), &senderKey.KeySet.PrivateKey, db, nil, common.Hash{}, NormalCoinType, "PRV", 0))

		isValidSanity, err := coinBaseTx.ValidateSanityData(nil)
		assert.Equal(t, nil, err)
		assert.Equal(t, true, isValidSanity)

		// store output coin's coin commitments in coin base tx
		db.StoreCommitments(
			common.PRVCoinID,
			senderPaymentAddress.Pk,
			[][]byte{coinBaseTx.(*Tx).Proof.GetOutputCoins()[0].CoinDetails.GetCoinCommitment().ToBytesS()},
			senderShardID)

		/******** get output coins from coin base tx to create new tx ********/
		coinBaseOutput := ConvertOutputCoinToInputCoin(coinBaseTx.(*Tx).Proof.GetOutputCoins())

		fmt.Printf("coinBaseOutput[0]GetValue: %v\n", coinBaseOutput[0].CoinDetails.GetValue())
		fmt.Printf("coinBaseOutput[0]GetSNDerivatorRandom: %v\n", coinBaseOutput[0].CoinDetails.GetSNDerivatorRandom())
		fmt.Printf("coinBaseOutput[0]GetCoinCommitment: %v\n", coinBaseOutput[0].CoinDetails.GetCoinCommitment())
		fmt.Printf("coinBaseOutput[0]GetPrivRandOTA: %v\n", coinBaseOutput[0].CoinDetails.GetPrivRandOTA())
		fmt.Printf("coinBaseOutput[0]GetRandomness: %v\n", coinBaseOutput[0].CoinDetails.GetRandomness())
		fmt.Printf("coinBaseOutput[0]GetInfo: %v\n", coinBaseOutput[0].CoinDetails.GetInfo())
		fmt.Printf("coinBaseOutput[0]GetPublicKey: %v\n", coinBaseOutput[0].CoinDetails.GetPublicKey())
		fmt.Printf("coinBaseOutput[0]GetSerialNumber: %v\n", coinBaseOutput[0].CoinDetails.GetSerialNumber())

		/******** init tx with mode no privacy ********/
		fmt.Println("********************* Tx1 ********************* ")
		tx1 := Tx{}
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
		err = tx1.Init(
			NewTxPrivacyInitParams(
				&senderKey.KeySet.PrivateKey,
				[]*privacy.PaymentInfo{{PaymentAddress: receiverKeyWallet.KeySet.PaymentAddress, Amount: uint64(transferAmount), Message: msgCipherText.Bytes()}},
				coinBaseOutput, uint64(fee), hasPrivacy, db, nil, nil, []byte{}, TxVersion2,
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
		fmt.Println("********************* Tx4 ********************* ")
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

		//isValidTxVersion := tx1.CheckTxVersion(1)
		//assert.Equal(t, true, isValidTxVersion)
		//
		////isValidTxFee := tx1.CheckTransactionFee(0)
		////assert.Equal(t, true, isValidTxFee)
		//
		//isSalaryTx := tx1.IsSalaryTx()
		//assert.Equal(t, false, isSalaryTx)
		//
		//actualSenderPublicKey := tx1.GetSender()
		//expectedSenderPublicKey := make([]byte, common.PublicKeySize)
		//copy(expectedSenderPublicKey, senderPublicKey[:])
		//assert.Equal(t, expectedSenderPublicKey, actualSenderPublicKey[:])
		//
		////err = tx1.ValidateTxWithCurrentMempool(nil)
		////	assert.Equal(t, nil, err)
		//
		//err = tx1.ValidateDoubleSpendWithBlockchain(nil, senderShardID, db, nil)
		//assert.Equal(t, nil, err)
		//
		//err = tx1.ValidateTxWithBlockChain(nil, senderShardID, db)
		//assert.Equal(t, nil, err)
		//
		//isValid, err = tx1.ValidateTxByItself(hasPrivacy, db, nil, senderShardID)
		//assert.Equal(t, nil, err)
		//assert.Equal(t, true, isValid)
		//
		//metaDataType := tx1.GetMetadataType()
		//assert.Equal(t, metadata.InvalidMeta, metaDataType)
		//
		//metaData := tx1.GetMetadata()
		//assert.Equal(t, nil, metaData)
		//
		//info := tx1.GetInfo()
		//assert.Equal(t, 0, len(info))
		//
		//lockTime := tx1.GetLockTime()
		//now := time.Now().Unix()
		//assert.LessOrEqual(t, lockTime, now)
		//
		//actualSigPubKey := tx1.GetSigPubKey()
		//assert.Equal(t, expectedSenderPublicKey, actualSigPubKey)
		//
		//proof := tx1.GetProof()
		//assert.NotEqual(t, nil, proof)
		//
		//isValidTxType := tx1.ValidateType()
		//assert.Equal(t, true, isValidTxType)
		//
		//isCoinsBurningTx := tx1.IsCoinsBurning()
		//assert.Equal(t, false, isCoinsBurningTx)
		//
		//actualTxValue := tx1.CalculateTxValue()
		//assert.Equal(t, uint64(transferAmount), actualTxValue)

		// store output coin's coin commitments in tx1
		//for i:=0; i < len(tx1.Proof.GetOutputCoins()); i++ {
		//	db.StoreCommitments(
		//		common.PRVCoinID,
		//		tx1.Proof.GetOutputCoins()[i].CoinDetails.GetPublicKey().Compress(),
		//		[][]byte{tx1.Proof.GetOutputCoins()[i].CoinDetails.GetCoinCommitment().Compress()},
		//		senderShardID)
		//}

		// init tx with privacy
		//tx2 := Tx{}
		//
		//err = tx2.Init(
		//	NewTxPrivacyInitParams(
		//		&senderKey.KeySet.PrivateKey,
		//		[]*privacy.PaymentInfo{{PaymentAddress: senderPaymentAddress, Amount: uint64(transferAmount)}},
		//		coinBaseOutput, 1, true, db, nil, nil, []byte{}))
		//if err != nil {
		//	t.Error(err)
		//}
		//
		//isValidSanity, err = tx2.ValidateSanityData(nil)
		//assert.Equal(t, nil, err)
		//assert.Equal(t, true, isValidSanity)
		//
		//isValidTx, err := tx2.ValidateTransaction(true, db, senderShardID, &common.PRVCoinID)
		//assert.Equal(t, true, isValidTx)

	}
}
