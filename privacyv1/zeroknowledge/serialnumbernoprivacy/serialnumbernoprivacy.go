package serialnumbernoprivacy

import (
	"errors"
	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/utils"
	"github.com/incognitochain/incognito-chain/privacyv1"
)

type SerialNumberNoPrivacyStatement struct {
	output *privacyv1.Point
	vKey   *privacyv1.Point
	input  *privacyv1.Scalar
}

// SNNoPrivacyWitness is a protocol for Zero-knowledge Proof of Knowledge of one out of many commitments containing 0
// include Witness: CommitedValue, r []byte
type SNNoPrivacyWitness struct {
	stmt SerialNumberNoPrivacyStatement
	seed *privacyv1.Scalar
}

// serialNumberNNoPrivacyProof contains Proof's value
type SNNoPrivacyProof struct {
	// general info
	stmt SerialNumberNoPrivacyStatement

	tSeed   *privacyv1.Point
	tOutput *privacyv1.Point

	zSeed *privacyv1.Scalar
}

func (proof SNNoPrivacyProof) ValidateSanity() bool {
	if !proof.stmt.output.PointValid() {
		return false
	}
	if !proof.stmt.vKey.PointValid() {
		return false
	}
	if !proof.stmt.input.ScalarValid() {
		return false
	}

	if !proof.tSeed.PointValid() {
		return false
	}
	if !proof.tOutput.PointValid() {
		return false
	}
	return proof.zSeed.ScalarValid()
}

func (pro SNNoPrivacyProof) isNil() bool {
	if pro.stmt.output == nil {
		return true
	}
	if pro.stmt.vKey == nil {
		return true
	}
	if pro.stmt.input == nil {
		return true
	}
	if pro.tSeed == nil {
		return true
	}
	if pro.tOutput == nil {
		return true
	}
	if pro.zSeed == nil {
		return true
	}
	return false
}

func (pro *SNNoPrivacyProof) Init() *SNNoPrivacyProof {
	pro.stmt.output = new(privacyv1.Point)
	pro.stmt.vKey = new(privacyv1.Point)
	pro.stmt.input = new(privacyv1.Scalar)

	pro.tSeed = new(privacyv1.Point)
	pro.tOutput = new(privacyv1.Point)

	pro.zSeed = new(privacyv1.Scalar)

	return pro
}

// Set sets Witness
func (wit *SNNoPrivacyWitness) Set(
	output *privacyv1.Point,
	vKey *privacyv1.Point,
	input *privacyv1.Scalar,
	seed *privacyv1.Scalar) {

	if wit == nil {
		wit = new(SNNoPrivacyWitness)
	}

	wit.stmt.output = output
	wit.stmt.vKey = vKey
	wit.stmt.input = input

	wit.seed = seed
}

// Set sets Proof
func (pro *SNNoPrivacyProof) Set(
	output *privacyv1.Point,
	vKey *privacyv1.Point,
	input *privacyv1.Scalar,
	tSeed *privacyv1.Point,
	tOutput *privacyv1.Point,
	zSeed *privacyv1.Scalar) {

	if pro == nil {
		pro = new(SNNoPrivacyProof)
	}

	pro.stmt.output = output
	pro.stmt.vKey = vKey
	pro.stmt.input = input

	pro.tSeed = tSeed
	pro.tOutput = tOutput

	pro.zSeed = zSeed
}

func (pro SNNoPrivacyProof) Bytes() []byte {
	// if proof is nil, return an empty array
	if pro.isNil() {
		return []byte{}
	}

	var bytes []byte
	bytes = append(bytes, pro.stmt.output.ToBytesS()...)
	bytes = append(bytes, pro.stmt.vKey.ToBytesS()...)
	bytes = append(bytes, pro.stmt.input.ToBytesS()...)

	bytes = append(bytes, pro.tSeed.ToBytesS()...)
	bytes = append(bytes, pro.tOutput.ToBytesS()...)

	bytes = append(bytes, pro.zSeed.ToBytesS()...)

	return bytes
}

func (pro *SNNoPrivacyProof) SetBytes(bytes []byte) error {
	if len(bytes) == 0 {
		return errors.New("Bytes array is empty")
	}

	offset := 0
	var err error
	pro.stmt.output, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacyv1.Ed25519KeySize

	pro.stmt.vKey, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacyv1.Ed25519KeySize

	pro.stmt.input.FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	offset += privacyv1.Ed25519KeySize

	pro.tSeed, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacyv1.Ed25519KeySize

	pro.tOutput, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacyv1.Ed25519KeySize

	pro.zSeed.FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])

	return nil
}

func (wit SNNoPrivacyWitness) Prove(mess []byte) (*SNNoPrivacyProof, error) {
	// randomness
	eSK := privacyv1.RandomScalar()

	// calculate tSeed = g_SK^eSK
	tSK := new(privacyv1.Point).ScalarMult(privacyv1.PedCom.G[privacyv1.PedersenPrivateKeyIndex], eSK)

	// calculate tOutput = sn^eSK
	tE := new(privacyv1.Point).ScalarMult(wit.stmt.output, eSK)

	x := new(privacyv1.Scalar)
	if mess == nil {
		// calculate x = hash(tSeed || tInput || tSND2 || tOutput)
		// recheck frombytes is valid scalar
		x = utils.GenerateChallenge([][]byte{tSK.ToBytesS(), tE.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Calculate zSeed = SK * x + eSK
	zSK := new(privacyv1.Scalar).Mul(wit.seed, x)
	zSK.Add(zSK, eSK)

	proof := new(SNNoPrivacyProof).Init()
	proof.Set(wit.stmt.output, wit.stmt.vKey, wit.stmt.input, tSK, tE, zSK)
	return proof, nil
}

func (pro SNNoPrivacyProof) Verify(mess []byte) (bool, error) {
	// re-calculate x = hash(tSeed || tOutput)
	x := new(privacyv1.Scalar)
	if mess == nil {
		// calculate x = hash(tSeed || tInput || tSND2 || tOutput)
		x = utils.GenerateChallenge([][]byte{pro.tSeed.ToBytesS(), pro.tOutput.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Check gSK^zSeed = vKey^x * tSeed
	leftPoint1 := new(privacyv1.Point).ScalarMult(privacyv1.PedCom.G[privacyv1.PedersenPrivateKeyIndex], pro.zSeed)

	rightPoint1 := new(privacyv1.Point).ScalarMult(pro.stmt.vKey, x)
	rightPoint1 = rightPoint1.Add(rightPoint1, pro.tSeed)

	if !privacyv1.IsPointEqual(leftPoint1, rightPoint1) {
		privacyv1.Logger.Log.Errorf("verify serial number no privacy proof statement 1 failed")
		return false, errors.New("verify serial number no privacy proof statement 1 failed")
	}

	// Check sn^(zSeed + x*input) = gSK^x * tOutput
	tmp := new(privacyv1.Scalar).Add(pro.zSeed, new(privacyv1.Scalar).Mul(x, pro.stmt.input))
	leftPoint2 := new(privacyv1.Point).ScalarMult(pro.stmt.output, tmp)

	rightPoint2 := new (privacyv1.Point).ScalarMult(privacyv1.PedCom.G[privacyv1.PedersenPrivateKeyIndex], x)
	rightPoint2 = rightPoint2.Add(rightPoint2, pro.tOutput)

	if !privacyv1.IsPointEqual(leftPoint2, rightPoint2) {
		privacyv1.Logger.Log.Errorf("verify serial number no privacy proof statement 2 failed")
		return false, errors.New("verify serial number no privacy proof statement 2 failed")
	}

	return true, nil
}
