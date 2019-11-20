package serialnumberprivacy

import (
	"errors"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacyv1"
	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/utils"
)

type SerialNumberPrivacyStatement struct {
	sn       *privacyv1.Point // serial number
	comSK    *privacyv1.Point // commitment to private key
	comInput *privacyv1.Point // commitment to input of the pseudo-random function
}

type SNPrivacyWitness struct {
	stmt *SerialNumberPrivacyStatement // statement to be proved

	sk     *privacyv1.Scalar // private key
	rSK    *privacyv1.Scalar // blinding factor in the commitment to private key
	input  *privacyv1.Scalar // input of pseudo-random function
	rInput *privacyv1.Scalar // blinding factor in the commitment to input
}

type SNPrivacyProof struct {
	stmt *SerialNumberPrivacyStatement // statement to be proved

	tSK    *privacyv1.Point // random commitment related to private key
	tInput *privacyv1.Point // random commitment related to input
	tSN    *privacyv1.Point // random commitment related to serial number

	zSK     *privacyv1.Scalar // first challenge-dependent information to open the commitment to private key
	zRSK    *privacyv1.Scalar // second challenge-dependent information to open the commitment to private key
	zInput  *privacyv1.Scalar // first challenge-dependent information to open the commitment to input
	zRInput *privacyv1.Scalar // second challenge-dependent information to open the commitment to input
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

	proof.tSK = new(privacyv1.Point)
	proof.tInput = new(privacyv1.Point)
	proof.tSN = new(privacyv1.Point)

	proof.zSK = new(privacyv1.Scalar)
	proof.zRSK = new(privacyv1.Scalar)
	proof.zInput = new(privacyv1.Scalar)
	proof.zRInput = new(privacyv1.Scalar)

	return proof
}

// Set sets Statement
func (stmt *SerialNumberPrivacyStatement) Set(
	SN *privacyv1.Point,
	comSK *privacyv1.Point,
	comInput *privacyv1.Point) {
	stmt.sn = SN
	stmt.comSK = comSK
	stmt.comInput = comInput
}

// Set sets Witness
func (wit *SNPrivacyWitness) Set(
	stmt *SerialNumberPrivacyStatement,
	SK *privacyv1.Scalar,
	rSK *privacyv1.Scalar,
	input *privacyv1.Scalar,
	rInput *privacyv1.Scalar) {

	wit.stmt = stmt
	wit.sk = SK
	wit.rSK = rSK
	wit.input = input
	wit.rInput = rInput
}

// Set sets Proof
func (proof *SNPrivacyProof) Set(
	stmt *SerialNumberPrivacyStatement,
	tSK *privacyv1.Point,
	tInput *privacyv1.Point,
	tSN *privacyv1.Point,
	zSK *privacyv1.Scalar,
	zRSK *privacyv1.Scalar,
	zInput *privacyv1.Scalar,
	zRInput *privacyv1.Scalar) {
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

	proof.stmt.sn = new(privacyv1.Point)
	proof.stmt.sn, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacyv1.Ed25519KeySize

	proof.stmt.comSK = new(privacyv1.Point)
	proof.stmt.comSK, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacyv1.Ed25519KeySize
	proof.stmt.comInput = new(privacyv1.Point)
	proof.stmt.comInput, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacyv1.Ed25519KeySize
	proof.tSK = new(privacyv1.Point)
	proof.tSK, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacyv1.Ed25519KeySize
	proof.tInput = new(privacyv1.Point)
	proof.tInput, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacyv1.Ed25519KeySize
	proof.tSN = new(privacyv1.Point)
	proof.tSN, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}

	offset += privacyv1.Ed25519KeySize
	proof.zSK = new(privacyv1.Scalar).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])

	offset += privacyv1.Ed25519KeySize
	proof.zRSK = new(privacyv1.Scalar).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])

	offset += privacyv1.Ed25519KeySize
	proof.zInput = new(privacyv1.Scalar).FromBytesS(bytes[offset : offset+common.BigIntSize])

	offset += privacyv1.Ed25519KeySize
	proof.zRInput = new(privacyv1.Scalar).FromBytesS(bytes[offset : offset+common.BigIntSize])

	return nil
}

func (wit SNPrivacyWitness) Prove(mess []byte) (*SNPrivacyProof, error) {

	eSK := privacyv1.RandomScalar()
	eSND := privacyv1.RandomScalar()
	dSK := privacyv1.RandomScalar()
	dSND := privacyv1.RandomScalar()

	// calculate tSeed = g_SK^eSK * h^dSK
	tSeed := privacyv1.PedCom.CommitAtIndex(eSK, dSK, privacyv1.PedersenPrivateKeyIndex)

	// calculate tSND = g_SND^eSND * h^dSND
	tInput := privacyv1.PedCom.CommitAtIndex(eSND, dSND, privacyv1.PedersenSndIndex)

	// calculate tSND = g_SK^eSND * h^dSND2
	tOutput := new(privacyv1.Point).ScalarMult(wit.stmt.sn, new(privacyv1.Scalar).Add(eSK, eSND))

	// calculate x = hash(tSeed || tInput || tSND2 || tOutput)
	x := new(privacyv1.Scalar)
	if mess == nil {
		x = utils.GenerateChallenge([][]byte{
			tSeed.ToBytesS(),
			tInput.ToBytesS(),
			tOutput.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Calculate zSeed = sk * x + eSK
	zSeed := new(privacyv1.Scalar).Mul(wit.sk, x)
	zSeed.Add(zSeed, eSK)
	//zSeed.Mod(zSeed, privacyv1.Curve.Params().N)

	// Calculate zRSeed = rSK * x + dSK
	zRSeed := new(privacyv1.Scalar).Mul(wit.rSK, x)
	zRSeed.Add(zRSeed, dSK)
	//zRSeed.Mod(zRSeed, privacyv1.Curve.Params().N)

	// Calculate zInput = input * x + eSND
	zInput := new(privacyv1.Scalar).Mul(wit.input, x)
	zInput.Add(zInput, eSND)
	//zInput.Mod(zInput, privacyv1.Curve.Params().N)

	// Calculate zRInput = rInput * x + dSND
	zRInput := new(privacyv1.Scalar).Mul(wit.rInput, x)
	zRInput.Add(zRInput, dSND)
	//zRInput.Mod(zRInput, privacyv1.Curve.Params().N)

	proof := new(SNPrivacyProof).Init()
	proof.Set(wit.stmt, tSeed, tInput, tOutput, zSeed, zRSeed, zInput, zRInput)
	return proof, nil
}

func (proof SNPrivacyProof) Verify(mess []byte) (bool, error) {
	// re-calculate x = hash(tSeed || tInput || tSND2 || tOutput)
	x := new(privacyv1.Scalar)
	if mess == nil {
		x = utils.GenerateChallenge([][]byte{
			proof.tSK.ToBytesS(),
			proof.tInput.ToBytesS(),
			proof.tSN.ToBytesS()})
	} else {
		x.FromBytesS(mess)
	}

	// Check gSND^zInput * h^zRInput = input^x * tInput
	leftPoint1 := privacyv1.PedCom.CommitAtIndex(proof.zInput, proof.zRInput, privacyv1.PedersenSndIndex)

	rightPoint1 := new(privacyv1.Point).ScalarMult(proof.stmt.comInput, x)
	rightPoint1.Add(rightPoint1, proof.tInput)

	if !privacyv1.IsPointEqual(leftPoint1, rightPoint1) {
		privacyv1.Logger.Log.Errorf("verify serial number privacy proof statement 1 failed")
		return false, errors.New("verify serial number privacy proof statement 1 failed")
	}

	// Check gSK^zSeed * h^zRSeed = vKey^x * tSeed
	leftPoint2 := privacyv1.PedCom.CommitAtIndex(proof.zSK, proof.zRSK, privacyv1.PedersenPrivateKeyIndex)

	rightPoint2 := new(privacyv1.Point).ScalarMult(proof.stmt.comSK, x)
	rightPoint2.Add(rightPoint2, proof.tSK)

	if !privacyv1.IsPointEqual(leftPoint2, rightPoint2) {
		privacyv1.Logger.Log.Errorf("verify serial number privacy proof statement 2 failed")
		return false, errors.New("verify serial number privacy proof statement 2 failed")
	}

	// Check sn^(zSeed + zInput) = gSK^x * tOutput
	leftPoint3 := new(privacyv1.Point).ScalarMult(proof.stmt.sn, new(privacyv1.Scalar).Add(proof.zSK, proof.zInput))

	rightPoint3 := new(privacyv1.Point).ScalarMult(privacyv1.PedCom.G[privacyv1.PedersenPrivateKeyIndex], x)
	rightPoint3.Add(rightPoint3, proof.tSN)

	if !privacyv1.IsPointEqual(leftPoint3, rightPoint3) {
		//privacyv1.Logger.Log.Errorf("verify serial number privacy proof statement 3 failed")
		return false, errors.New("verify serial number privacy proof statement 3 failed")
	}

	return true, nil
}
