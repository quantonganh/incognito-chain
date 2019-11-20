package zkp

import (
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacyv1"
	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/aggregaterange"
	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/oneoutofmany"
	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/serialnumbernoprivacy"
	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/serialnumberprivacy"
	
)

// PaymentWitness contains all of witness for proving when spending coins
type PaymentWitness struct {
	privateKey          *privacyv1.Scalar
	inputCoins          []*privacyv1.InputCoin
	outputCoins         []*privacyv1.OutputCoin
	commitmentIndices   []uint64
	myCommitmentIndices []uint64

	oneOfManyWitness             []*oneoutofmany.OneOutOfManyWitness
	serialNumberWitness          []*serialnumberprivacy.SNPrivacyWitness
	serialNumberNoPrivacyWitness []*serialnumbernoprivacy.SNNoPrivacyWitness

	aggregatedRangeWitness *aggregaterange.AggregatedRangeWitness

	comOutputValue                 []*privacyv1.Point
	comOutputSerialNumberDerivator []*privacyv1.Point
	comOutputShardID               []*privacyv1.Point

	comInputSecretKey             *privacyv1.Point
	comInputValue                 []*privacyv1.Point
	comInputSerialNumberDerivator []*privacyv1.Point
	comInputShardID               *privacyv1.Point

	randSecretKey *privacyv1.Scalar
}

func (paymentWitness PaymentWitness) GetRandSecretKey() *privacyv1.Scalar {
	return paymentWitness.randSecretKey
}

type PaymentWitnessParam struct {
	HasPrivacy              bool
	PrivateKey              *privacyv1.Scalar
	InputCoins              []*privacyv1.InputCoin
	OutputCoins             []*privacyv1.OutputCoin
	PublicKeyLastByteSender byte
	Commitments             []*privacyv1.Point
	CommitmentIndices       []uint64
	MyCommitmentIndices     []uint64
	Fee                     uint64
}

// Build prepares witnesses for all protocol need to be proved when create tx
// if hashPrivacy = false, witness includes spending key, input coins, output coins
// otherwise, witness includes all attributes in PaymentWitness struct
func (wit *PaymentWitness) Init(PaymentWitnessParam PaymentWitnessParam) *privacyv1.PrivacyError {

	hasPrivacy := PaymentWitnessParam.HasPrivacy
	privateKey := PaymentWitnessParam.PrivateKey
	inputCoins := PaymentWitnessParam.InputCoins
	outputCoins := PaymentWitnessParam.OutputCoins
	publicKeyLastByteSender := PaymentWitnessParam.PublicKeyLastByteSender
	commitments := PaymentWitnessParam.Commitments
	commitmentIndices := PaymentWitnessParam.CommitmentIndices
	myCommitmentIndices := PaymentWitnessParam.MyCommitmentIndices
	_ = PaymentWitnessParam.Fee

	if !hasPrivacy {
		for _, outCoin := range outputCoins {
			outCoin.CoinDetails.SetRandomness(privacyv1.RandomScalar())
			err := outCoin.CoinDetails.CommitAll()
			if err != nil {
				return privacyv1.NewPrivacyErr(privacyv1.CommitNewOutputCoinNoPrivacyErr, nil)
			}
		}
		wit.privateKey = privateKey
		wit.inputCoins = inputCoins
		wit.outputCoins = outputCoins

		if len(inputCoins) > 0 {
			publicKey := inputCoins[0].CoinDetails.GetPublicKey()

			wit.serialNumberNoPrivacyWitness = make([]*serialnumbernoprivacy.SNNoPrivacyWitness, len(inputCoins))
			for i := 0; i < len(inputCoins); i++ {
				/***** Build witness for proving that serial number is derived from the committed derivator *****/

				inputSNDTmp := new(privacyv1.Scalar)
				snd := inputCoins[i].CoinDetails.GetSNDerivator()
				privRandOTA := inputCoins[i].CoinDetails.GetPrivRandOTA()
				if snd != nil && !snd.IsZero() {
					// input from tx version 0 or tx version 1 no privacy
					inputSNDTmp = snd
				} else if privRandOTA != nil && !privRandOTA.IsZero() {
					// input from tx version 1 has privacy
					inputSNDTmp = privRandOTA
				}
				if wit.serialNumberNoPrivacyWitness[i] == nil {
					wit.serialNumberNoPrivacyWitness[i] = new(serialnumbernoprivacy.SNNoPrivacyWitness)
				}
				wit.serialNumberNoPrivacyWitness[i].Set(inputCoins[i].CoinDetails.GetSerialNumber(), publicKey, inputSNDTmp, wit.privateKey)
			}
		}

		return nil
	}

	wit.privateKey = privateKey
	wit.inputCoins = inputCoins
	wit.outputCoins = outputCoins
	wit.commitmentIndices = commitmentIndices
	wit.myCommitmentIndices = myCommitmentIndices

	numInputCoin := len(wit.inputCoins)

	randInputSK := privacyv1.RandomScalar()
	// set rand sk for Schnorr signature
	wit.randSecretKey = new(privacyv1.Scalar).Set(randInputSK)

	cmInputSK := privacyv1.PedCom.CommitAtIndex(wit.privateKey, randInputSK, privacyv1.PedersenPrivateKeyIndex)
	wit.comInputSecretKey = new(privacyv1.Point).Set(cmInputSK)

	randInputShardID := privacyv1.RandomScalar()
	senderShardID := common.GetShardIDFromLastByte(publicKeyLastByteSender)
	wit.comInputShardID = privacyv1.PedCom.CommitAtIndex(new(privacyv1.Scalar).FromUint64(uint64(senderShardID)), randInputShardID, privacyv1.PedersenShardIDIndex)

	wit.comInputValue = make([]*privacyv1.Point, numInputCoin)
	wit.comInputSerialNumberDerivator = make([]*privacyv1.Point, numInputCoin)
	// It is used for proving 2 commitments commit to the same value (input)
	//cmInputSNDIndexSK := make([]*privacyv1.Point, numInputCoin)

	randInputValue := make([]*privacyv1.Scalar, numInputCoin)
	randInputSND := make([]*privacyv1.Scalar, numInputCoin)
	//randInputSNDIndexSK := make([]*big.Int, numInputCoin)

	// cmInputValueAll is sum of all input coins' value commitments
	cmInputValueAll := new(privacyv1.Point).Identity()
	randInputValueAll := new(privacyv1.Scalar).FromUint64(0)

	// Summing all commitments of each input coin into one commitment and proving the knowledge of its Openings
	cmInputSum := make([]*privacyv1.Point, numInputCoin)
	randInputSum := make([]*privacyv1.Scalar, numInputCoin)
	// randInputSumAll is sum of all randomess of coin commitments
	randInputSumAll := new(privacyv1.Scalar).FromUint64(0)

	wit.oneOfManyWitness = make([]*oneoutofmany.OneOutOfManyWitness, numInputCoin)
	wit.serialNumberWitness = make([]*serialnumberprivacy.SNPrivacyWitness, numInputCoin)

	commitmentTemps := make([][]*privacyv1.Point, numInputCoin)
	randInputIsZero := make([]*privacyv1.Scalar, numInputCoin)

	preIndex := 0

	for i, inputCoin := range wit.inputCoins {
		// commit each component of coin commitment
		randInputValue[i] = privacyv1.RandomScalar()
		randInputSND[i] = privacyv1.RandomScalar()

		wit.comInputValue[i] = privacyv1.PedCom.CommitAtIndex(new(privacyv1.Scalar).FromUint64(inputCoin.CoinDetails.GetValue()), randInputValue[i], privacyv1.PedersenValueIndex)
		wit.comInputSerialNumberDerivator[i] = privacyv1.PedCom.CommitAtIndex(inputCoin.CoinDetails.GetSNDerivator(), randInputSND[i], privacyv1.PedersenSndIndex)

		cmInputValueAll.Add(cmInputValueAll, wit.comInputValue[i])
		randInputValueAll.Add(randInputValueAll, randInputValue[i])

		/***** Build witness for proving one-out-of-N commitments is a commitment to the coins being spent *****/
		cmInputSum[i] = new(privacyv1.Point).Add(cmInputSK, wit.comInputValue[i])
		cmInputSum[i].Add(cmInputSum[i], wit.comInputSerialNumberDerivator[i])
		cmInputSum[i].Add(cmInputSum[i], wit.comInputShardID)

		randInputSum[i] = new(privacyv1.Scalar).Set(randInputSK)
		randInputSum[i].Add(randInputSum[i], randInputValue[i])
		randInputSum[i].Add(randInputSum[i], randInputSND[i])
		randInputSum[i].Add(randInputSum[i], randInputShardID)

		randInputSumAll.Add(randInputSumAll, randInputSum[i])

		// commitmentTemps is a list of commitments for protocol one-out-of-N
		commitmentTemps[i] = make([]*privacyv1.Point, privacyv1.CommitmentRingSize)

		randInputIsZero[i] = new(privacyv1.Scalar).FromUint64(0)
		randInputIsZero[i].Sub(inputCoin.CoinDetails.GetRandomness(), randInputSum[i])

		for j := 0; j < privacyv1.CommitmentRingSize; j++ {
			commitmentTemps[i][j] = new(privacyv1.Point).Sub(commitments[preIndex+j], cmInputSum[i])
		}

		if wit.oneOfManyWitness[i] == nil {
			wit.oneOfManyWitness[i] = new(oneoutofmany.OneOutOfManyWitness)
		}
		indexIsZero := myCommitmentIndices[i] % privacyv1.CommitmentRingSize

		wit.oneOfManyWitness[i].Set(commitmentTemps[i], randInputIsZero[i], indexIsZero)
		preIndex = privacyv1.CommitmentRingSize * (i + 1)
		// ---------------------------------------------------

		/***** Build witness for proving that serial number is derived from the committed derivator *****/
		if wit.serialNumberWitness[i] == nil {
			wit.serialNumberWitness[i] = new(serialnumberprivacy.SNPrivacyWitness)
		}
		stmt := new(serialnumberprivacy.SerialNumberPrivacyStatement)
		stmt.Set(inputCoin.CoinDetails.GetSerialNumber(), cmInputSK, wit.comInputSerialNumberDerivator[i])
		wit.serialNumberWitness[i].Set(stmt, privateKey, randInputSK, inputCoin.CoinDetails.GetSNDerivator(), randInputSND[i])
		// ---------------------------------------------------
	}

	numOutputCoin := len(wit.outputCoins)

	randOutputValue := make([]*privacyv1.Scalar, numOutputCoin)
	randOutputSND := make([]*privacyv1.Scalar, numOutputCoin)
	cmOutputValue := make([]*privacyv1.Point, numOutputCoin)
	cmOutputSND := make([]*privacyv1.Point, numOutputCoin)

	cmOutputSum := make([]*privacyv1.Point, numOutputCoin)
	randOutputSum := make([]*privacyv1.Scalar, numOutputCoin)

	cmOutputSumAll := new(privacyv1.Point).Identity()

	// cmOutputValueAll is sum of all value coin commitments
	cmOutputValueAll := new(privacyv1.Point).Identity()

	randOutputValueAll := new(privacyv1.Scalar).FromUint64(0)

	randOutputShardID := make([]*privacyv1.Scalar, numOutputCoin)
	cmOutputShardID := make([]*privacyv1.Point, numOutputCoin)

	for i, outputCoin := range wit.outputCoins {
		if i == len(outputCoins)-1 {
			randOutputValue[i] = new(privacyv1.Scalar).Sub(randInputValueAll, randOutputValueAll)
		} else {
			randOutputValue[i] = privacyv1.RandomScalar()
		}

		randOutputSND[i] = privacyv1.RandomScalar()
		randOutputShardID[i] = privacyv1.RandomScalar()

		cmOutputValue[i] = privacyv1.PedCom.CommitAtIndex(new(privacyv1.Scalar).FromUint64(outputCoin.CoinDetails.GetValue()), randOutputValue[i], privacyv1.PedersenValueIndex)
		cmOutputSND[i] = privacyv1.PedCom.CommitAtIndex(outputCoin.CoinDetails.GetSNDerivator(), randOutputSND[i], privacyv1.PedersenSndIndex)

		receiverShardID := common.GetShardIDFromLastByte(outputCoins[i].CoinDetails.GetPubKeyLastByte())
		cmOutputShardID[i] = privacyv1.PedCom.CommitAtIndex(new(privacyv1.Scalar).FromUint64(uint64(receiverShardID)), randOutputShardID[i], privacyv1.PedersenShardIDIndex)

		randOutputSum[i] = new(privacyv1.Scalar).FromUint64(0)
		randOutputSum[i].Add(randOutputValue[i], randOutputSND[i])
		randOutputSum[i].Add(randOutputSum[i], randOutputShardID[i])

		cmOutputSum[i] = new(privacyv1.Point).Identity()
		cmOutputSum[i].Add(cmOutputValue[i], cmOutputSND[i])
		cmOutputSum[i].Add(cmOutputSum[i], outputCoins[i].CoinDetails.GetPublicKey())
		cmOutputSum[i].Add(cmOutputSum[i], cmOutputShardID[i])

		cmOutputValueAll.Add(cmOutputValueAll, cmOutputValue[i])
		randOutputValueAll.Add(randOutputValueAll, randOutputValue[i])

		// calculate final commitment for output coins
		outputCoins[i].CoinDetails.SetCoinCommitment(cmOutputSum[i])
		outputCoins[i].CoinDetails.SetRandomness(randOutputSum[i])

		cmOutputSumAll.Add(cmOutputSumAll, cmOutputSum[i])
	}

	// For Multi Range Protocol
	// proving each output value is less than vmax
	// proving sum of output values is less than vmax
	outputValue := make([]uint64, numOutputCoin)
	for i := 0; i < numOutputCoin; i++ {
		if outputCoins[i].CoinDetails.GetValue() > 0 {
			outputValue[i] = outputCoins[i].CoinDetails.GetValue()
		} else {
			return privacyv1.NewPrivacyErr(privacyv1.UnexpectedErr, errors.New("output coin's value is less than 0"))
		}
	}
	if wit.aggregatedRangeWitness == nil {
		wit.aggregatedRangeWitness = new(aggregaterange.AggregatedRangeWitness)
	}
	wit.aggregatedRangeWitness.Set(outputValue, randOutputValue)
	// ---------------------------------------------------

	// save partial commitments (value, input, shardID)
	wit.comOutputValue = cmOutputValue
	wit.comOutputSerialNumberDerivator = cmOutputSND
	wit.comOutputShardID = cmOutputShardID

	return nil
}

// Prove creates big proof
func (wit *PaymentWitness) Prove(hasPrivacy bool) (*PaymentProof, *privacyv1.PrivacyError) {
	proof := new(PaymentProof)
	proof.Init()

	proof.inputCoins = wit.inputCoins
	proof.outputCoins = wit.outputCoins
	proof.commitmentOutputValue = wit.comOutputValue
	proof.commitmentOutputSND = wit.comOutputSerialNumberDerivator
	proof.commitmentOutputShardID = wit.comOutputShardID

	proof.commitmentInputSecretKey = wit.comInputSecretKey
	proof.commitmentInputValue = wit.comInputValue
	proof.commitmentInputSND = wit.comInputSerialNumberDerivator
	proof.commitmentInputShardID = wit.comInputShardID
	proof.commitmentIndices = wit.commitmentIndices

	// if hasPrivacy == false, don't need to create the zero knowledge proof
	// proving user has spending key corresponding with public key in input coins
	// is proved by signing with spending key
	if !hasPrivacy {
		// Proving that serial number is derived from the committed derivator
		for i := 0; i < len(wit.inputCoins); i++ {
			snNoPrivacyProof, err := wit.serialNumberNoPrivacyWitness[i].Prove(nil)
			if err != nil {
				return nil, privacyv1.NewPrivacyErr(privacyv1.ProveSerialNumberNoPrivacyErr, err)
			}
			proof.serialNumberNoPrivacyProof = append(proof.serialNumberNoPrivacyProof, snNoPrivacyProof)
		}
		return proof, nil
	}

	// if hasPrivacy == true
	numInputCoins := len(wit.oneOfManyWitness)

	for i := 0; i < numInputCoins; i++ {
		// Proving one-out-of-N commitments is a commitment to the coins being spent
		oneOfManyProof, err := wit.oneOfManyWitness[i].Prove()
		if err != nil {
			return nil, privacyv1.NewPrivacyErr(privacyv1.ProveOneOutOfManyErr, err)
		}
		proof.oneOfManyProof = append(proof.oneOfManyProof, oneOfManyProof)

		// Proving that serial number is derived from the committed derivator
		serialNumberProof, err := wit.serialNumberWitness[i].Prove(nil)
		if err != nil {
			return nil, privacyv1.NewPrivacyErr(privacyv1.ProveSerialNumberPrivacyErr, err)
		}
		proof.serialNumberProof = append(proof.serialNumberProof, serialNumberProof)
	}
	var err error

	// Proving that each output values and sum of them does not exceed v_max
	proof.aggregatedRangeProof, err = wit.aggregatedRangeWitness.Prove()
	if err != nil {
		return nil, privacyv1.NewPrivacyErr(privacyv1.ProveAggregatedRangeErr, err)
	}

	if len(proof.inputCoins) == 0 {
		proof.commitmentIndices = nil
		proof.commitmentInputSecretKey = nil
		proof.commitmentInputShardID = nil
		proof.commitmentInputSND = nil
		proof.commitmentInputValue = nil
	}

	if len(proof.outputCoins) == 0 {
		proof.commitmentOutputValue = nil
		proof.commitmentOutputSND = nil
		proof.commitmentOutputShardID = nil
	}

	//privacyv1.Logger.Log.Debug("Privacy log: PROVING DONE!!!")
	return proof, nil
}