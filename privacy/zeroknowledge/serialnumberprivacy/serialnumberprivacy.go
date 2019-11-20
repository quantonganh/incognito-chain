package serialnumberprivacy

import (
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge/utils"
)

type SerialNumberPrivacyStatement struct {
	sn       *privacy.Point // serial number
	comSK    *privacy.Point // commitment to private key
	comInput *privacy.Point // commitment to input of the pseudo-random function
}

type SNPrivacyWitness struct {
	stmt *SerialNumberPrivacyStatement // statement to be proved

	sk     *privacy.Scalar // private key
	rSK    *privacy.Scalar // blinding factor in the commitment to private key
	input  *privacy.Scalar // input of pseudo-random function
	rInput *privacy.Scalar // blinding factor in the commitment to input
}

type SNPrivacyProof struct {
	stmt *SerialNumberPrivacyStatement // statement to be proved

	tSK    *privacy.Point // random commitment related to private key
	tInput *privacy.Point // random commitment related to input
	tSN    *privacy.Point // random commitment related to serial number

	zSK     *privacy.Scalar // first challenge-dependent information to open the commitment to private key
	zRSK    *privacy.Scalar // second challenge-dependent information to open the commitment to private key
	zInput  *privacy.Scalar // first challenge-dependent information to open the commitment to input
	zRInput *privacy.Scalar // second challenge-dependent information to open the commitment to input
}

// ValidateSanity validates sanity of proof
func (proof SNPrivacyProof) ValidateSanity() bool {
	if !proof.stmt.sn.PointValid() {
		return false
	}
	if !proof.stmt.comSK.PointValid() {
		return false
	}
	if !proof.stmt.comInput.PointValid() {
		return false
	}
	if !proof.tSK.PointValid() {
		return false
	}
	if !proof.tInput.PointValid() {
		return false
	}
	if !proof.tSN.PointValid() {
		return false
	}
	if !proof.zSK.ScalarValid() {
		return false
	}
	if !proof.zRSK.ScalarValid() {
		return false
	}
	if !proof.zInput.ScalarValid() {
		return false
	}
	if !proof.zRInput.ScalarValid() {
		return false
	}
	return true
}

func (proof SNPrivacyProof) isNil() bool {
	if proof.stmt.sn == nil {
		return true
	}
	if proof.stmt.comSK == nil {
		return true
	}
	if proof.stmt.comInput == nil {
		return true
	}
	if proof.tSK == nil {
		return true
	}
	if proof.tInput == nil {
		return true
	}
	if proof.tSN == nil {
		return true
	}
	if proof.zSK == nil {
		return true
	}
	if proof.zRSK == nil {
		return true
	}
	if proof.zInput == nil {
		return true
	}
	return proof.zRInput == nil
}

// Init inits Proof
func (proof *SNPrivacyProof) Init() *SNPrivacyProof {
	proof.stmt = new(SerialNumberPrivacyStatement)

	proof.tSK = new(privacy.Point)
	proof.tInput = new(privacy.Point)
	proof.tSN = new(privacy.Point)

	proof.zSK = new(privacy.Scalar)
	proof.zRSK = new(privacy.Scalar)
	proof.zInput = new(privacy.Scalar)
	proof.zRInput = new(privacy.Scalar)

	return proof
}

// Set sets Statement
func (stmt *SerialNumberPrivacyStatement) Set(
	SN *privacy.Point,
	comSK *privacy.Point,
	comInput *privacy.Point) {
	stmt.sn = SN
	stmt.comSK = comSK
	stmt.comInput = comInput
}

// Set sets Witness
func (wit *SNPrivacyWitness) Set(
	stmt *SerialNumberPrivacyStatement,
	SK *privacy.Scalar,
	rSK *privacy.Scalar,
	input *privacy.Scalar,
	rInput *privacy.Scalar) {

	wit.stmt = stmt
	wit.sk = SK
	wit.rSK = rSK
	wit.input = input
	wit.rInput = rInput
}

// Set sets Proof
func (proof *SNPrivacyProof) Set(
	stmt *SerialNumberPrivacyStatement,
	tSK *privacy.Point,
	tInput *privacy.Point,
	tSN *privacy.Point,
	zSK *privacy.Scalar,
	zRSK *privacy.Scalar,
	zInput *privacy.Scalar,
	zRInput *privacy.Scalar) {
	proof.stmt = stmt
	proof.tSK = tSK
	proof.tInput = tInput
	proof.tSN = tSN

	proof.zSK = zSK
	proof.zRSK = zRSK
	proof.zInput = zInput
	proof.zRInput = zRInput
}

func (proof SNPrivacyProof) Bytes() []byte {
	// if proof is nil, return an empty array
	if proof.isNil() {
		return []byte{}
	}

	var bytes []byte
	bytes = append(bytes, proof.stmt.sn.ToBytesS()...)
	bytes = append(bytes, proof.stmt.comSK.ToBytesS()...)
	bytes = append(bytes, proof.stmt.comInput.ToBytesS()...)

	bytes = append(bytes, proof.tSK.ToBytesS()...)
	bytes = append(bytes, proof.tInput.ToBytesS()...)
	bytes = append(bytes, proof.tSN.ToBytesS()...)

	bytes = append(bytes, proof.zSK.ToBytesS()...)
	bytes = append(bytes, proof.zRSK.ToBytesS()...)
	bytes = append(bytes, proof.zInput.ToBytesS()...)
	bytes = append(bytes, proof.zRInput.ToBytesS()...)

	return bytes
}

func (proof *SNPrivacyProof) SetBytes(bytes []byte) error {
	if len(bytes) == 0 {
		return errors.New("Bytes array is empty")
	}

	offset := 0
	var err error

	proof.stmt.sn = new(privacy.Point)
	proof.stmt.sn, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacy.Ed25519KeySize

	proof.stmt.comSK = new(privacy.Point)
	proof.stmt.comSK, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacy.Ed25519KeySize
	proof.stmt.comInput = new(privacy.Point)
	proof.stmt.comInput, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacy.Ed25519KeySize
	proof.tSK = new(privacy.Point)
	proof.tSK, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacy.Ed25519KeySize
	proof.tInput = new(privacy.Point)
	proof.tInput, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacy.Ed25519KeySize
	proof.tSN = new(privacy.Point)
	proof.tSN, err = new(privacy.Point).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacy.Ed25519KeySize
	proof.zSK = new(privacy.Scalar).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])

	offset += privacy.Ed25519KeySize
	proof.zRSK = new(privacy.Scalar).FromBytesS(bytes[offset : offset+privacy.Ed25519KeySize])

	offset += privacy.Ed25519KeySize
	proof.zInput = new(privacy.Scalar).FromBytesS(bytes[offset : offset+common.BigIntSize])

	offset += privacy.Ed25519KeySize
	proof.zRInput = new(privacy.Scalar).FromBytesS(bytes[offset : offset+common.BigIntSize])

	return nil
}

func (wit SNPrivacyWitness) Prove(mess []byte) (*SNPrivacyProof, error) {

	eSK := privacy.RandomScalar()
	eSND := privacy.RandomScalar()
	dSK := privacy.RandomScalar()
	dSND := privacy.RandomScalar()

	// calculate tSeed = g_SK^eSK * h^dSK
	tSeed := privacy.PedCom.CommitAtIndex(eSK, dSK, privacy.PedersenPrivateKeyIndex)

	// calculate tSND = g_SND^eSND * h^dSND
	tInput := privacy.PedCom.CommitAtIndex(eSND, dSND, privacy.PedersenSndIndex)

	// calculate tSND = g_SK^eSND * h^dSND2
	tOutput := new(privacy.Point).ScalarMult(wit.stmt.sn, new(privacy.Scalar).Add(eSK, eSND))

	// calculate x = hash(tSeed || tInput || tSND2 || tOutput)
	x := new(privacy.Scalar)
	if mess == nil {
		x = utils.GenerateChallenge([][]byte{
			tSeed.ToBytesS(),
			tInput.ToBytesS(),
			tOutput.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Calculate zSeed = sk * x + eSK
	zSeed := new(privacy.Scalar).Mul(wit.sk, x)
	zSeed.Add(zSeed, eSK)
	//zSeed.Mod(zSeed, privacy.Curve.Params().N)

	// Calculate zRSeed = rSK * x + dSK
	zRSeed := new(privacy.Scalar).Mul(wit.rSK, x)
	zRSeed.Add(zRSeed, dSK)
	//zRSeed.Mod(zRSeed, privacy.Curve.Params().N)

	// Calculate zInput = input * x + eSND
	zInput := new(privacy.Scalar).Mul(wit.input, x)
	zInput.Add(zInput, eSND)
	//zInput.Mod(zInput, privacy.Curve.Params().N)

	// Calculate zRInput = rInput * x + dSND
	zRInput := new(privacy.Scalar).Mul(wit.rInput, x)
	zRInput.Add(zRInput, dSND)
	//zRInput.Mod(zRInput, privacy.Curve.Params().N)

	proof := new(SNPrivacyProof).Init()
	proof.Set(wit.stmt, tSeed, tInput, tOutput, zSeed, zRSeed, zInput, zRInput)
	return proof, nil
}

func (proof SNPrivacyProof) Verify(mess []byte) (bool, error) {
	// re-calculate x = hash(tSeed || tInput || tSND2 || tOutput)
	x := new(privacy.Scalar)
	if mess == nil {
		x = utils.GenerateChallenge([][]byte{
			proof.tSK.ToBytesS(),
			proof.tInput.ToBytesS(),
			proof.tSN.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Check gSND^zInput * h^zRInput = input^x * tInput
	leftPoint1 := privacy.PedCom.CommitAtIndex(proof.zInput, proof.zRInput, privacy.PedersenSndIndex)

	rightPoint1 := new(privacy.Point).ScalarMult(proof.stmt.comInput, x)
	rightPoint1.Add(rightPoint1, proof.tInput)

	if !privacy.IsPointEqual(leftPoint1, rightPoint1) {
		privacy.Logger.Log.Errorf("verify serial number privacy proof statement 1 failed")
		return false, errors.New("verify serial number privacy proof statement 1 failed")
	}

	// Check gSK^zSeed * h^zRSeed = vKey^x * tSeed
	leftPoint2 := privacy.PedCom.CommitAtIndex(proof.zSK, proof.zRSK, privacy.PedersenPrivateKeyIndex)

	rightPoint2 := new(privacy.Point).ScalarMult(proof.stmt.comSK, x)
	rightPoint2.Add(rightPoint2, proof.tSK)

	if !privacy.IsPointEqual(leftPoint2, rightPoint2) {
		privacy.Logger.Log.Errorf("verify serial number privacy proof statement 2 failed")
		return false, errors.New("verify serial number privacy proof statement 2 failed")
	}

	// Check sn^(zSeed + zInput) = gSK^x * tOutput
	leftPoint3 := new(privacy.Point).ScalarMult(proof.stmt.sn, new(privacy.Scalar).Add(proof.zSK, proof.zInput))

	rightPoint3 := new(privacy.Point).ScalarMult(privacy.PedCom.G[privacy.PedersenPrivateKeyIndex], x)
	rightPoint3.Add(rightPoint3, proof.tSN)

	if !privacy.IsPointEqual(leftPoint3, rightPoint3) {
		//privacy.Logger.Log.Errorf("verify serial number privacy proof statement 3 failed")
		return false, errors.New("verify serial number privacy proof statement 3 failed")
	}

	return true, nil
}
