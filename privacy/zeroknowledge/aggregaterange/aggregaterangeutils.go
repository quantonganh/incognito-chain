package aggregaterange

import (
	"errors"
	"github.com/incognitochain/incognito-chain/privacy"
	C25519 "github.com/incognitochain/incognito-chain/privacy/curve25519"
	"math"
)

// pad returns number has format 2^k that it is the nearest number to num
func pad(num int) int {
	if num == 1 || num == 2 {
		return num
	}
	tmp := 2
	for i := 2; ; i++ {
		tmp *= 2
		if tmp >= num {
			num = tmp
			break
		}
	}
	return num
}

/*-----------------------------Vector Functions-----------------------------*/
// The length here always has to be a power of two

//vectorAdd adds two vector and returns result vector
func vectorAdd(a []*privacy.Scalar, b []*privacy.Scalar) ([]*privacy.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("VectorAdd: Arrays not of the same length")
	}

	res := make([]*privacy.Scalar, len(a))
	for i := range a {
		res[i] = new(privacy.Scalar).Add(a[i], b[i])
	}
	return res, nil
}

// innerProduct calculates inner product between two vectors a and b
func innerProduct(a []*privacy.Scalar, b []*privacy.Scalar) (*privacy.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("InnerProduct: Arrays not of the same length")
	}
	res := new(privacy.Scalar).FromUint64(uint64(0))
	for i := range a {
		//res = a[i]*b[i] + res % l
		res.MulAdd(a[i], b[i], res)
	}
	return res, nil
}

// hadamardProduct calculates hadamard product between two vectors a and b
func hadamardProduct(a []*privacy.Scalar, b []*privacy.Scalar) ([]*privacy.Scalar, error) {
	if len(a) != len(b) {
		return nil, errors.New("InnerProduct: Arrays not of the same length")
	}

	res := make([]*privacy.Scalar, len(a))
	for i := 0; i < len(res); i++ {
		res[i] = new(privacy.Scalar).Mul(a[i], b[i])
	}
	return res, nil
}

// powerVector calculates base^n
func powerVector(base *privacy.Scalar, n int) []*privacy.Scalar {
	res := make([]*privacy.Scalar, n)
	res[0] = new(privacy.Scalar).FromUint64(1)
	if n >1 {
		res[1] = new(privacy.Scalar).Set(base)
		for i := 2; i < n; i++ {
			res[i] = new(privacy.Scalar).Mul(res[i-1], base)
		}
	}
	return res
}

// vectorAddScalar adds a vector to a big int, returns big int array
func vectorAddScalar(v []*privacy.Scalar, s *privacy.Scalar) []*privacy.Scalar {
	res := make([]*privacy.Scalar, len(v))

	for i := range v {
		res[i] = new(privacy.Scalar).Add(v[i], s)
	}
	return res
}

// vectorMulScalar mul a vector to a big int, returns a vector
func vectorMulScalar(v []*privacy.Scalar, s *privacy.Scalar) []*privacy.Scalar {
	res := make([]*privacy.Scalar, len(v))

	for i := range v {
		res[i] = new(privacy.Scalar).Mul(v[i], s)
	}
	return res
}

// CommitAll commits a list of PCM_CAPACITY value(s)
func encodeVectors(l []*privacy.Scalar, r []*privacy.Scalar, g []*privacy.Point, h []*privacy.Point) (*privacy.Point, error) {
	// MultiscalarMul Approach
	if len(l) != len(r) || len(g) != len(l) || len(h) != len(g) {
		return nil, errors.New("invalid input")
	}
	tmp1 := new(privacy.Point).MultiScalarMult(l, g)
	tmp2 := new(privacy.Point).MultiScalarMult(r, h)

	res := new(privacy.Point).Add(tmp1, tmp2)
	return res, nil

	////AddPedersen Approach
	//if len(l) != len(r) || len(g) != len(l) || len(h) != len(g) {
	//	return nil, errors.New("invalid input")
	//}
	//
	//res := new(privacy.Point).Identity()
	//
	//for i := 0; i < len(l); i++ {
	//	tmp := new(privacy.Point).AddPedersen(l[i], g[i], r[i], h[i])
	//	res.Add(res, tmp)
	//}
	//return res, nil
}

func encodeCachedVectors(l []*privacy.Scalar, r []*privacy.Scalar, gPre [][8]C25519.CachedGroupElement, hPre [][8]C25519.CachedGroupElement) (*privacy.Point, error) {
	// MultiscalarMul Approach
	//if len(l) != len(r) || len(gPre) != len(l) || len(hPre) != len(gPre) {
	//	return nil, errors.New("invalid input")
	//}
	//tmp1 := new(privacy.Point).MultiScalarMultCached(l, gPre)
	//tmp2 := new(privacy.Point).MultiScalarMultCached(r, hPre)
	//
	//res := new(privacy.Point).Add(tmp1, tmp2)
	//return res, nil


	//CacheAddPedersen Approach
	if len(l) != len(r) || len(gPre) != len(l) || len(hPre) != len(hPre) {
		return nil, errors.New("invalid input")
	}

	res := new(privacy.Point).Identity()

	for i := 0; i < len(l); i++ {
		tmp := new(privacy.Point).AddPedersenCached(l[i], gPre[i], r[i], hPre[i])
		res.Add(res, tmp)
	}
	return res, nil
}


// estimateMultiRangeProofSize estimate multi range proof size
func EstimateMultiRangeProofSize(nOutput int) uint64 {
	return uint64((nOutput+2*int(math.Log2(float64(maxExp*pad(nOutput))))+5)*privacy.Ed25519KeySize + 5*privacy.Ed25519KeySize + 2)
}
