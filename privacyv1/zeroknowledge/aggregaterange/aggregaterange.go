package aggregaterange

import (
	"github.com/incognitochain/incognito-chain/privacyv1"
	"github.com/pkg/errors"
)

// This protocol proves in zero-knowledge that a list of committed values falls in [0, 2^64)

type AggregatedRangeWitness struct {
	values []uint64
	rands  []*privacyv1.Scalar
}

type AggregatedRangeProof struct {
	cmsValue          []*privacyv1.Point
	a                 *privacyv1.Point
	s                 *privacyv1.Point
	t1                *privacyv1.Point
	t2                *privacyv1.Point
	tauX              *privacyv1.Scalar
	tHat              *privacyv1.Scalar
	mu                *privacyv1.Scalar
	innerProductProof *InnerProductProof
}

func (proof AggregatedRangeProof) ValidateSanity() bool {
	for i := 0; i < len(proof.cmsValue); i++ {
		if !proof.cmsValue[i].PointValid() {
			return false
		}
	}
	if !proof.a.PointValid() {
		return false
	}
	if !proof.s.PointValid() {
		return false
	}
	if !proof.t1.PointValid() {
		return false
	}
	if !proof.t2.PointValid() {
		return false
	}
	if !proof.tauX.ScalarValid() {
		return false
	}
	if !proof.tHat.ScalarValid(){
		return false
	}
	if !proof.mu.ScalarValid() {
		return false
	}

	return proof.innerProductProof.ValidateSanity()
}

func (proof *AggregatedRangeProof) Init() {
	proof.a = new(privacyv1.Point).Identity()
	proof.s = new(privacyv1.Point).Identity()
	proof.t1 = new(privacyv1.Point).Identity()
	proof.t2 = new(privacyv1.Point).Identity()
	proof.tauX = new(privacyv1.Scalar)
	proof.tHat = new(privacyv1.Scalar)
	proof.mu = new(privacyv1.Scalar)
	proof.innerProductProof = new(InnerProductProof)
}

func (proof AggregatedRangeProof) IsNil() bool {
	if proof.a == nil {
		return true
	}
	if proof.s == nil {
		return true
	}
	if proof.t1 == nil {
		return true
	}
	if proof.t2 == nil {
		return true
	}
	if proof.tauX == nil {
		return true
	}
	if proof.tHat == nil {
		return true
	}
	if proof.mu == nil {
		return true
	}
	return proof.innerProductProof == nil
}

func (proof AggregatedRangeProof) Bytes() []byte {
	var res []byte

	if proof.IsNil() {
		return []byte{}
	}

	res = append(res, byte(len(proof.cmsValue)))
	for i := 0; i < len(proof.cmsValue); i++ {
		res = append(res, proof.cmsValue[i].ToBytesS()...)
	}

	res = append(res, proof.a.ToBytesS()...)
	res = append(res, proof.s.ToBytesS()...)
	res = append(res, proof.t1.ToBytesS()...)
	res = append(res, proof.t2.ToBytesS()...)

	res = append(res, proof.tauX.ToBytesS()...)
	res = append(res, proof.tHat.ToBytesS()...)
	res = append(res, proof.mu.ToBytesS()...)
	res = append(res, proof.innerProductProof.Bytes()...)

	//privacyv1.Logger.Log.Debugf("BYTES ------------ %v\n", res)
	return res

}

func (proof *AggregatedRangeProof) SetBytes(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	//privacyv1.Logger.Log.Debugf("BEFORE SETBYTES ------------ %v\n", bytes)

	lenValues := int(bytes[0])
	offset := 1
	var err error

	proof.cmsValue = make([]*privacyv1.Point, lenValues)
	for i := 0; i < lenValues; i++ {
		proof.cmsValue[i], err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
		if err != nil {
			return err
		}
		offset += privacyv1.Ed25519KeySize
	}

	proof.a, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacyv1.Ed25519KeySize

	proof.s, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacyv1.Ed25519KeySize

	proof.t1, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacyv1.Ed25519KeySize

	proof.t2, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil {
		return err
	}
	offset += privacyv1.Ed25519KeySize

	proof.tauX = new(privacyv1.Scalar).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	offset += privacyv1.Ed25519KeySize

	proof.tHat = new(privacyv1.Scalar).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	offset += privacyv1.Ed25519KeySize

	proof.mu = new(privacyv1.Scalar).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	offset += privacyv1.Ed25519KeySize

	proof.innerProductProof = new(InnerProductProof)
	proof.innerProductProof.SetBytes(bytes[offset:])

	//privacyv1.Logger.Log.Debugf("AFTER SETBYTES ------------ %v\n", proof.Bytes())
	return nil
}

func (wit *AggregatedRangeWitness) Set(values []uint64, rands []*privacyv1.Scalar) {
	numValue := len(values)
	wit.values = make([]uint64, numValue)
	wit.rands = make([]*privacyv1.Scalar, numValue)

	for i := range values {
		wit.values[i] = values[i]
		wit.rands[i] = new(privacyv1.Scalar).Set(rands[i])
	}
}

func (wit AggregatedRangeWitness) Prove() (*AggregatedRangeProof, error) {
	proof := new(AggregatedRangeProof)

	numValue := len(wit.values)
	numValuePad := pad(numValue)
	values := make([]uint64, numValuePad)
	rands := make([]*privacyv1.Scalar, numValuePad)

	for i := range wit.values {
		values[i] = wit.values[i]
		rands[i] = new(privacyv1.Scalar).Set(wit.rands[i])
	}

	for i := numValue; i < numValuePad; i++ {
		values[i] = uint64(0)
		rands[i] = new(privacyv1.Scalar).FromUint64(0)
	}

	aggParam := new(bulletproofParams)
	extraNumber := numValuePad - len(AggParam.g) / 64
	if extraNumber > 0 {
		aggParam = addBulletproofParams(extraNumber)
	} else {
		aggParam.g = AggParam.g[0:numValuePad*64]
		aggParam.h = AggParam.h[0:numValuePad*64]
		aggParam.u = AggParam.u
		//aggParam.gPrecomputed = AggParam.gPrecomputed[0:numValuePad*64]
		//aggParam.hPrecomputed = AggParam.hPrecomputed[0:numValuePad*64]
		//aggParam.gPreMultiScalar = AggParam.gPreMultiScalar[0:numValuePad*64]
		//aggParam.hPreMultiScalar = AggParam.hPreMultiScalar[0:numValuePad*64]
	}

	proof.cmsValue = make([]*privacyv1.Point, numValue)
	for i := 0; i < numValue; i++ {
		proof.cmsValue[i] = privacyv1.PedCom.CommitAtIndex(new(privacyv1.Scalar).FromUint64(values[i]), rands[i], privacyv1.PedersenValueIndex)
	}

	n := maxExp
	// Convert values to binary array
	aL := make([]*privacyv1.Scalar, numValuePad*n)
	for i, value := range values {
		tmp := privacyv1.ConvertUint64ToBinaryInBigInt(value, n)
		for j := 0; j < n; j++ {
			aL[i*n+j] = tmp[j]
		}
	}

	twoNumber := new(privacyv1.Scalar).FromUint64(2)
	twoVectorN := powerVector(twoNumber, n)

	aR := make([]*privacyv1.Scalar, numValuePad*n)

	for i := 0; i < numValuePad*n; i++ {
		aR[i] = new(privacyv1.Scalar).Sub(aL[i], new(privacyv1.Scalar).FromUint64(1))
	}

	// random alpha
	alpha := privacyv1.RandomScalar()

	// Commitment to aL, aR: A = h^alpha * G^aL * H^aR
	A, err := encodeVectors(aL, aR, aggParam.g, aggParam.h)
	if err != nil {
		return nil, err
	}
	A = A.Add(A, new(privacyv1.Point).ScalarMult(privacyv1.PedCom.G[privacyv1.PedersenRandomnessIndex], alpha))
	proof.a = A

	// Random blinding vectors sL, sR
	sL := make([]*privacyv1.Scalar, n*numValuePad)
	sR := make([]*privacyv1.Scalar, n*numValuePad)
	for i := range sL {
		sL[i] = privacyv1.RandomScalar()
		sR[i] = privacyv1.RandomScalar()
	}

	// random rho
	rho :=privacyv1.RandomScalar()

	// Commitment to sL, sR : S = h^rho * G^sL * H^sR
	S, err := encodeVectors(sL, sR, aggParam.g, aggParam.h)
	if err != nil {
		return nil, err
	}
	S = S.Add(S, new(privacyv1.Point).ScalarMult(privacyv1.PedCom.G[privacyv1.PedersenRandomnessIndex], rho))
	proof.s = S

	// challenge y, z
	y := generateChallengeForAggRange(aggParam, [][]byte{A.ToBytesS(), S.ToBytesS()})
	z := generateChallengeForAggRange(aggParam, [][]byte{A.ToBytesS(), S.ToBytesS(), y.ToBytesS()})
	zNeg := new(privacyv1.Scalar).Sub(new(privacyv1.Scalar).FromUint64(0), z)
	zSquare := new(privacyv1.Scalar).Mul(z, z)

	// l(X) = (aL -z*1^n) + sL*X
	yVector := powerVector(y, n*numValuePad)

	l0 := vectorAddScalar(aL, zNeg)
	l1 := sL

	// r(X) = y^n hada (aR +z*1^n + sR*X) + z^2 * 2^n
	hadaProduct, err := hadamardProduct(yVector, vectorAddScalar(aR, z))
	if err != nil {
		return nil, err
	}

	vectorSum := make([]*privacyv1.Scalar, n*numValuePad)
	zTmp := new(privacyv1.Scalar).Set(z)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		for i := 0; i < n; i++ {
			vectorSum[j*n+i] = new(privacyv1.Scalar).Mul(twoVectorN[i], zTmp)
		}
	}

	r0, err := vectorAdd(hadaProduct, vectorSum)
	if err != nil {
		return nil, err
	}

	r1, err := hadamardProduct(yVector, sR)
	if err != nil {
		return nil, err
	}

	//t(X) = <l(X), r(X)> = t0 + t1*X + t2*X^2

	//calculate t0 = v*z^2 + delta(y, z)
	deltaYZ := new(privacyv1.Scalar).Sub(z, zSquare)

	// innerProduct1 = <1^(n*m), y^(n*m)>
	innerProduct1 := new(privacyv1.Scalar).FromUint64(0)
	for i := 0; i < n*numValuePad; i++ {
		innerProduct1 = innerProduct1.Add(innerProduct1, yVector[i])
	}

	deltaYZ.Mul(deltaYZ, innerProduct1)

	// innerProduct2 = <1^n, 2^n>
	innerProduct2 := new(privacyv1.Scalar).FromUint64(0)
	for i := 0; i < n; i++ {
		innerProduct2 = innerProduct2.Add(innerProduct2, twoVectorN[i])
	}

	sum := new(privacyv1.Scalar).FromUint64(0)
	zTmp = new(privacyv1.Scalar).Set(zSquare)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		sum.Add(sum, zTmp)
	}
	sum.Mul(sum, innerProduct2)
	deltaYZ.Sub(deltaYZ, sum)

	// t1 = <l1, r0> + <l0, r1>
	innerProduct3, err := innerProduct(l1, r0)
	if err != nil {
		return nil, err
	}

	innerProduct4, err := innerProduct(l0, r1)
	if err != nil {
		return nil, err
	}

	t1 := new(privacyv1.Scalar).Add(innerProduct3, innerProduct4)

	// t2 = <l1, r1>
	t2, err := innerProduct(l1, r1)
	if err != nil {
		return nil, err
	}

	// commitment to t1, t2
	tau1 := privacyv1.RandomScalar()
	tau2 := privacyv1.RandomScalar()

	proof.t1 = privacyv1.PedCom.CommitAtIndex(t1, tau1, privacyv1.PedersenValueIndex)
	proof.t2 = privacyv1.PedCom.CommitAtIndex(t2, tau2, privacyv1.PedersenValueIndex)

	// challenge x = hash(G || H || A || S || T1 || T2)
	x := generateChallengeForAggRange(aggParam,
		[][]byte{proof.a.ToBytesS(), proof.s.ToBytesS(), proof.t1.ToBytesS(), proof.t2.ToBytesS()})
	xSquare := new(privacyv1.Scalar).Mul(x,x)

	// lVector = aL - z*1^n + sL*x
	lVector, err := vectorAdd(vectorAddScalar(aL, zNeg), vectorMulScalar(sL, x))
	if err != nil {
		return nil, err
	}

	// rVector = y^n hada (aR +z*1^n + sR*x) + z^2*2^n
	tmpVector, err := vectorAdd(vectorAddScalar(aR, z), vectorMulScalar(sR, x))
	if err != nil {
		return nil, err
	}
	rVector, err := hadamardProduct(yVector, tmpVector)
	if err != nil {
		return nil, err
	}

	vectorSum = make([]*privacyv1.Scalar, n*numValuePad)
	zTmp = new(privacyv1.Scalar).Set(z)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		for i := 0; i < n; i++ {
			vectorSum[j*n+i] = new(privacyv1.Scalar).Mul(twoVectorN[i], zTmp)
		}
	}

	rVector, err = vectorAdd(rVector, vectorSum)
	if err != nil {
		return nil, err
	}

	// tHat = <lVector, rVector>
	proof.tHat, err = innerProduct(lVector, rVector)
	if err != nil {
		return nil, err
	}

	// blinding value for tHat: tauX = tau2*x^2 + tau1*x + z^2*rand
	proof.tauX = new(privacyv1.Scalar).Mul(tau2, xSquare)
	proof.tauX.Add(proof.tauX, new(privacyv1.Scalar).Mul(tau1, x))
	zTmp = new(privacyv1.Scalar).Set(z)
	tmpBN := new(privacyv1.Scalar)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		proof.tauX.Add(proof.tauX, tmpBN.Mul(zTmp, rands[j]))
	}

	// alpha, rho blind A, S
	// mu = alpha + rho*x
	proof.mu = new(privacyv1.Scalar).Mul(rho, x)
	proof.mu.Add(proof.mu, alpha)

	// instead of sending left vector and right vector, we use inner sum argument to reduce proof size from 2*n to 2(log2(n)) + 2
	innerProductWit := new(InnerProductWitness)
	innerProductWit.a = lVector
	innerProductWit.b = rVector
	innerProductWit.p, err = encodeVectors(lVector, rVector, aggParam.g, aggParam.h)
	if err != nil {
		return nil, err
	}
	innerProductWit.p = innerProductWit.p.Add(innerProductWit.p, new(privacyv1.Point).ScalarMult(aggParam.u, proof.tHat))

	proof.innerProductProof, err = innerProductWit.Prove(aggParam)
	if err != nil {
		return nil, err
	}

	return proof, nil
}

func (proof AggregatedRangeProof) Verify() (bool, error) {
	numValue := len(proof.cmsValue)
	numValuePad := pad(numValue)

	tmpcmsValue := proof.cmsValue

	for i := numValue; i < numValuePad; i++ {
		identity := new(privacyv1.Point).Identity()
		tmpcmsValue = append(tmpcmsValue, identity)
	}

	aggParam := new(bulletproofParams)
	extraNumber := numValuePad - len(AggParam.g) / 64
	if extraNumber > 0 {
		aggParam = addBulletproofParams(extraNumber)
	} else {
		aggParam.g = AggParam.g[0:numValuePad*64]
		aggParam.h = AggParam.h[0:numValuePad*64]
		aggParam.u = AggParam.u
		//aggParam.gPrecomputed = AggParam.gPrecomputed[0:numValuePad*64]
		//aggParam.hPrecomputed = AggParam.hPrecomputed[0:numValuePad*64]
		//aggParam.gPreMultiScalar = AggParam.gPreMultiScalar[0:numValuePad*64]
		//aggParam.hPreMultiScalar = AggParam.hPreMultiScalar[0:numValuePad*64]
	}

	n := maxExp
	oneNumber := new(privacyv1.Scalar).FromUint64(1)
	twoNumber := new(privacyv1.Scalar).FromUint64(2)
	oneVector := powerVector(oneNumber, n*numValuePad)
	oneVectorN := powerVector(oneNumber, n)
	twoVectorN := powerVector(twoNumber, n)

	// recalculate challenge y, z
	y := generateChallengeForAggRange(aggParam, [][]byte{proof.a.ToBytesS(), proof.s.ToBytesS()})
	z := generateChallengeForAggRange(aggParam, [][]byte{proof.a.ToBytesS(), proof.s.ToBytesS(), y.ToBytesS()})
	zSquare := new(privacyv1.Scalar).Mul(z, z)

	// challenge x = hash(G || H || A || S || T1 || T2)
	//fmt.Printf("T2: %v\n", proof.t2)
	x := generateChallengeForAggRange(aggParam,[][]byte{proof.a.ToBytesS(), proof.s.ToBytesS(),proof.t1.ToBytesS(), proof.t2.ToBytesS()})
	xSquare := new(privacyv1.Scalar).Mul(x, x)

	yVector := powerVector(y, n*numValuePad)
	// HPrime = H^(y^(1-i)
	HPrime := make([]*privacyv1.Point, n*numValuePad)
	yInverse := new(privacyv1.Scalar).Invert(y)
	expyInverse := new(privacyv1.Scalar).FromUint64(1)
	for i := 0; i < n*numValuePad; i++ {
		HPrime[i] = new(privacyv1.Point).ScalarMult(aggParam.h[i], expyInverse)
		expyInverse.Mul(expyInverse, yInverse)
	}

	// g^tHat * h^tauX = V^(z^2) * g^delta(y,z) * T1^x * T2^(x^2)
	deltaYZ := new(privacyv1.Scalar).Sub(z, zSquare)

	// innerProduct1 = <1^(n*m), y^(n*m)>
	innerProduct1, err := innerProduct(oneVector, yVector)
	if err != nil {
		return false, privacyv1.NewPrivacyErr(privacyv1.CalInnerProductErr, err)
	}

	deltaYZ.Mul(deltaYZ, innerProduct1)

	// innerProduct2 = <1^n, 2^n>
	innerProduct2, err := innerProduct(oneVectorN, twoVectorN)
	if err != nil {
		return false, privacyv1.NewPrivacyErr(privacyv1.CalInnerProductErr, err)
	}

	sum := new(privacyv1.Scalar).FromUint64(0)
	zTmp := new(privacyv1.Scalar).Set(zSquare)
	for j := 0; j < numValuePad; j++ {
		zTmp.Mul(zTmp, z)
		sum.Add(sum, zTmp)
	}
	sum.Mul(sum, innerProduct2)
	deltaYZ.Sub(deltaYZ, sum)

	left1 := privacyv1.PedCom.CommitAtIndex(proof.tHat, proof.tauX, privacyv1.PedersenValueIndex)

	right1 := new(privacyv1.Point).ScalarMult( proof.t2, xSquare)
	right1.Add(right1, new(privacyv1.Point).AddPedersen(deltaYZ,privacyv1.PedCom.G[privacyv1.PedersenValueIndex], x, proof.t1))

	expVector := vectorMulScalar(powerVector(z, numValuePad), zSquare)
	right1.Add(right1, new(privacyv1.Point).MultiScalarMult(expVector, tmpcmsValue))

	if !privacyv1.IsPointEqual(left1, right1) {
		privacyv1.Logger.Log.Errorf("verify aggregated range proof statement 1 failed")
		return false, errors.New("verify aggregated range proof statement 1 failed")
	}

	innerProductArgValid := proof.innerProductProof.Verify(aggParam)
	if !innerProductArgValid {
		privacyv1.Logger.Log.Errorf("verify aggregated range proof statement 2 failed")
		return false, errors.New("verify aggregated range proof statement 2 failed")
	}

	return true, nil
}
