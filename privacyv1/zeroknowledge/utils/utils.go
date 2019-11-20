package utils

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacyv1"
	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/aggregaterange"
)

// GenerateChallengeFromByte get hash of n points in G append with input values
// return blake_2b(G[0]||G[1]||...||G[CM_CAPACITY-1]||<values>)
// G[i] is list of all generator point of Curve
func GenerateChallenge(values [][]byte) *privacyv1.Scalar {
	bytes := []byte{}
	for i := 0; i < len(privacyv1.PedCom.G); i++ {
		bytes = append(bytes, privacyv1.PedCom.G[i].ToBytesS()...)
	}

	for i := 0; i < len(values); i++ {
		bytes = append(bytes, values[i]...)
	}

	hash := privacyv1.HashToScalar(bytes)
	//res := new(big.Int).SetBytes(hash)
	//res.Mod(res, privacyv1.Curve.Params().N)
	return hash
}

// EstimateProofSize returns the estimated size of the proof in bytes
func EstimateProofSize(nInput int, nOutput int, hasPrivacy bool) uint64 {
	if !hasPrivacy {
		FlagSize := 14 + 2*nInput + nOutput
		sizeSNNoPrivacyProof := nInput * SnNoPrivacyProofSize
		sizeInputCoins := nInput * inputCoinsNoPrivacySize
		sizeOutputCoins := nOutput * OutputCoinsNoPrivacySize

		sizeProof := uint64(FlagSize + sizeSNNoPrivacyProof + sizeInputCoins + sizeOutputCoins)
		return uint64(sizeProof)
	}

	FlagSize := 14 + 7*nInput + 4*nOutput

	sizeOneOfManyProof := nInput * OneOfManyProofSize
	sizeSNPrivacyProof := nInput * SnPrivacyProofSize
	sizeComOutputMultiRangeProof := int(aggregaterange.EstimateMultiRangeProofSize(nOutput))

	sizeInputCoins := nInput * inputCoinsPrivacySize
	sizeOutputCoins := nOutput * outputCoinsPrivacySize

	sizeComOutputValue := nOutput * privacyv1.Ed25519KeySize
	sizeComOutputSND := nOutput * privacyv1.Ed25519KeySize
	sizeComOutputShardID := nOutput * privacyv1.Ed25519KeySize

	sizeComInputSK := privacyv1.Ed25519KeySize
	sizeComInputValue := nInput * privacyv1.Ed25519KeySize
	sizeComInputSND := nInput * privacyv1.Ed25519KeySize
	sizeComInputShardID := privacyv1.Ed25519KeySize

	sizeCommitmentIndices := nInput * privacyv1.CommitmentRingSize * common.Uint64Size

	sizeProof := sizeOneOfManyProof + sizeSNPrivacyProof +
		sizeComOutputMultiRangeProof + sizeInputCoins + sizeOutputCoins +
		sizeComOutputValue + sizeComOutputSND + sizeComOutputShardID +
		sizeComInputSK + sizeComInputValue + sizeComInputSND + sizeComInputShardID +
		sizeCommitmentIndices + FlagSize

	return uint64(sizeProof)
}
