package aggregaterange

import (
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/privacyv1"
)

type InnerProductWitness struct {
	a []*privacyv1.Scalar
	b []*privacyv1.Scalar

	p *privacyv1.Point
}

type InnerProductProof struct {
	l []*privacyv1.Point
	r []*privacyv1.Point
	a *privacyv1.Scalar
	b *privacyv1.Scalar

	p *privacyv1.Point
}

func (proof InnerProductProof) ValidateSanity() bool {
	if len(proof.l) != len(proof.r) {
		return false
	}

	for i := 0; i < len(proof.l); i++ {
		if !proof.l[i].PointValid() {
			return false
		}

		if !proof.r[i].PointValid() {
			return false
		}
	}

	if !proof.a.ScalarValid() {
		return false
	}
	if !proof.b.ScalarValid() {
		return false
	}

	return proof.p.PointValid()
}

func (proof InnerProductProof) Bytes() []byte {
	var res []byte

	res = append(res, byte(len(proof.l)))
	for _, l := range proof.l {
		res = append(res, l.ToBytesS()...)
	}

	for _, r := range proof.r {
		res = append(res, r.ToBytesS()...)
	}

	res = append(res, proof.a.ToBytesS()...)
	res = append(res, proof.b.ToBytesS()...)
	res = append(res, proof.p.ToBytesS()...)

	return res
}

func (proof *InnerProductProof) SetBytes(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	lenLArray := int(bytes[0])
	offset := 1
	var err error

	proof.l = make([]*privacyv1.Point, lenLArray)
	for i := 0; i < lenLArray; i++ {
		proof.l[i], err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
		if err != nil{
			return err
		}
		offset += privacyv1.Ed25519KeySize
	}

	proof.r = make([]*privacyv1.Point, lenLArray)
	for i := 0; i < lenLArray; i++ {
		proof.r[i], err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
		if err != nil{
			return err
		}
		offset += privacyv1.Ed25519KeySize
	}

	proof.a = new(privacyv1.Scalar).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	offset += privacyv1.Ed25519KeySize

	proof.b = new(privacyv1.Scalar).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	offset += privacyv1.Ed25519KeySize

	proof.p, err = new(privacyv1.Point).FromBytesS(bytes[offset : offset+privacyv1.Ed25519KeySize])
	if err != nil{
		return err
	}

	return nil
}

func (wit InnerProductWitness) Prove(AggParam *bulletproofParams) (*InnerProductProof, error) {
	//var AggParam = newBulletproofParams(1)
	if len(wit.a) != len(wit.b) {
		return nil, errors.New("invalid inputs")
	}

	n := len(wit.a)

	a := make([]*privacyv1.Scalar, n)
	b := make([]*privacyv1.Scalar, n)

	for i := range wit.a {
		a[i] = new(privacyv1.Scalar).Set(wit.a[i])
		b[i] = new(privacyv1.Scalar).Set(wit.b[i])
	}

	p := new(privacyv1.Point).Set(wit.p)
	G := make([]*privacyv1.Point, n)
	H := make([]*privacyv1.Point, n)
	for i := range G {
		G[i] = new(privacyv1.Point).Set(AggParam.g[i])
		H[i] = new(privacyv1.Point).Set(AggParam.h[i])
	}

	proof := new(InnerProductProof)
	proof.l = make([]*privacyv1.Point, 0)
	proof.r = make([]*privacyv1.Point, 0)
	proof.p = new(privacyv1.Point).Set(wit.p)

	for n > 1 {
		nPrime := n / 2

		cL, err := innerProduct(a[:nPrime], b[nPrime:])
		if err != nil {
			return nil, err
		}

		cR, err := innerProduct(a[nPrime:], b[:nPrime])
		if err != nil {
			return nil, err
		}

		L, err := encodeVectors(a[:nPrime], b[nPrime:], G[nPrime:], H[:nPrime])
		if err != nil {
			return nil, err
		}
		L.Add(L, new(privacyv1.Point).ScalarMult(AggParam.u, cL))
		proof.l = append(proof.l, L)

		R, err := encodeVectors(a[nPrime:], b[:nPrime], G[:nPrime], H[nPrime:])
		if err != nil {
			return nil, err
		}
		R.Add(R, new(privacyv1.Point).ScalarMult(AggParam.u, cR))
		proof.r = append(proof.r, R)

		// calculate challenge x = hash(G || H || u || p ||  l || r)
		x := generateChallengeForAggRange(AggParam, [][]byte{p.ToBytesS(), L.ToBytesS(), R.ToBytesS()})
		xInverse := new(privacyv1.Scalar).Invert(x)
		xSquare := new(privacyv1.Scalar).Mul(x, x)
		xSquareInverse := new(privacyv1.Scalar).Mul(xInverse, xInverse)

		// calculate GPrime, HPrime, PPrime for the next loop
		GPrime := make([]*privacyv1.Point, nPrime)
		HPrime := make([]*privacyv1.Point, nPrime)

		for i := range GPrime {
			//GPrime[i] = new(privacyv1.Point).ScalarMult(G[i], xInverse)
			//GPrime[i].Add(GPrime[i], new(privacyv1.Point).ScalarMult(G[i+nPrime], x))
			//GPrime[i] = new(privacyv1.Point).AddPedersen(xInverse, G[i], x, G[i+nPrime])
			GPrime[i] = new(privacyv1.Point).AddPedersen(xInverse, G[i], x, G[i+nPrime])

			//HPrime[i] = new(privacyv1.Point).ScalarMult(H[i], x)
			//HPrime[i].Add(HPrime[i], new(privacyv1.Point).ScalarMult(H[i+nPrime], xInverse))
			HPrime[i] = new(privacyv1.Point).AddPedersen(x, H[i], xInverse, H[i+nPrime])
		}

		// x^2 * l + P + xInverse^2 * r
		PPrime := new(privacyv1.Point).AddPedersen(xSquare, L, xSquareInverse, R)
		PPrime.Add(PPrime, p)

		// calculate aPrime, bPrime
		aPrime := make([]*privacyv1.Scalar, nPrime)
		bPrime := make([]*privacyv1.Scalar, nPrime)

		for i := range aPrime {
			aPrime[i] = new(privacyv1.Scalar).Mul(a[i], x)
			aPrime[i] = new(privacyv1.Scalar).MulAdd(a[i+nPrime], xInverse, aPrime[i])

			bPrime[i] = new(privacyv1.Scalar).Mul(b[i], xInverse)
			bPrime[i]= new(privacyv1.Scalar).MulAdd(b[i+nPrime], x, bPrime[i])
		}

		a = aPrime
		b = bPrime
		p.Set(PPrime)
		G = GPrime
		H = HPrime
		n = nPrime
	}

	proof.a = new(privacyv1.Scalar).Set(a[0])
	proof.b = new(privacyv1.Scalar).Set(b[0])

	return proof, nil
}

func (proof InnerProductProof) Verify(AggParam *bulletproofParams) bool {
	//var AggParam = newBulletproofParams(1)
	p := new(privacyv1.Point)
	p.Set(proof.p)

	n := len(AggParam.g)

	G := make([]*privacyv1.Point, n)
	H := make([]*privacyv1.Point, n)
	for i := range G {
		G[i] = new(privacyv1.Point).Set(AggParam.g[i])
		H[i] = new(privacyv1.Point).Set(AggParam.h[i])
	}

	for i := range proof.l {
		nPrime := n / 2
		// calculate challenge x = hash(G || H || u || p ||  l || r)
		x := generateChallengeForAggRange(AggParam, [][]byte{p.ToBytesS(), proof.l[i].ToBytesS(), proof.r[i].ToBytesS()})
		xInverse := new(privacyv1.Scalar).Invert(x)
		xSquare := new(privacyv1.Scalar).Mul(x, x)
		xSquareInverse := new(privacyv1.Scalar).Mul(xInverse, xInverse)

		// calculate GPrime, HPrime, PPrime for the next loop
		GPrime := make([]*privacyv1.Point, nPrime)
		HPrime := make([]*privacyv1.Point, nPrime)

		for j := 0; j < len(GPrime); j++ {
			//GPrime[j] = new(privacyv1.Point).ScalarMult(G[j], xInverse)
			//GPrime[j].Add(GPrime[j], new(privacyv1.Point).ScalarMult(G[j+nPrime], x))
			GPrime[j] = new(privacyv1.Point).AddPedersen(xInverse, G[j], x, G[j+nPrime])

			//HPrime[j] = new(privacyv1.Point).ScalarMult(H[j], x)
			//HPrime[j].Add(HPrime[j], new(privacyv1.Point).ScalarMult(H[j+nPrime], xInverse))
			HPrime[j] = new(privacyv1.Point).AddPedersen(x, H[j], xInverse, H[j+nPrime])
		}

		//PPrime := l.ScalarMul(xSquare).Add(p).Add(r.ScalarMul(xSquareInverse)) // x^2 * l + P + xInverse^2 * r
		PPrime := new(privacyv1.Point).AddPedersen(xSquare, proof.l[i], xSquareInverse, proof.r[i])
		PPrime.Add(PPrime, p) // x^2 * l + P + xInverse^2 * r

		p = PPrime
		G = GPrime
		H = HPrime
		n = nPrime
	}

	c := new(privacyv1.Scalar).Mul(proof.a, proof.b)

	rightPoint := new(privacyv1.Point).AddPedersen(proof.a, G[0], proof.b, H[0])
	rightPoint.Add(rightPoint, new(privacyv1.Point).ScalarMult(AggParam.u, c))

	res := privacyv1.IsPointEqual(rightPoint, p)
	if !res {
		privacyv1.Logger.Log.Error("Inner product argument failed:")
		privacyv1.Logger.Log.Error("p: %v\n", p)
		privacyv1.Logger.Log.Error("rightPoint: %v\n", rightPoint)
		fmt.Printf("Inner product argument failed:")
		fmt.Printf("p: %v\n", p)
		fmt.Printf("rightPoint: %v\n", rightPoint)
	}

	return res
}
