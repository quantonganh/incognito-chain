package serialnumberprivacy

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/privacyv1"
	"testing"
	"time"

	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/utils"
	"github.com/stretchr/testify/assert"
)

func TestPKSNPrivacy(t *testing.T) {
	for i:= 0 ; i <1000; i++ {
		sk := privacyv1.GeneratePrivateKey(privacyv1.RandBytes(31))
		skScalar := new(privacyv1.Scalar).FromBytesS(sk)
		if skScalar.ScalarValid() == false {
			fmt.Println("Invalid scala key value")
		}

		SND := privacyv1.RandomScalar()
		rSK := privacyv1.RandomScalar()
		rSND := privacyv1.RandomScalar()

		serialNumber := new(privacyv1.Point).Derive(privacyv1.PedCom.G[privacyv1.PedersenPrivateKeyIndex], skScalar, SND)
		comSK := privacyv1.PedCom.CommitAtIndex(skScalar, rSK, privacyv1.PedersenPrivateKeyIndex)
		comSND := privacyv1.PedCom.CommitAtIndex(SND, rSND, privacyv1.PedersenSndIndex)

		stmt := new(SerialNumberPrivacyStatement)
		stmt.Set(serialNumber, comSK, comSND)

		witness := new(SNPrivacyWitness)
		witness.Set(stmt, skScalar, rSK, SND, rSND)

		// proving
		start := time.Now()
		proof, err := witness.Prove(nil)
		assert.Equal(t, nil, err)

		end := time.Since(start)
		fmt.Printf("Serial number proving time: %v\n", end)

		//validate sanity proof
		isValidSanity := proof.ValidateSanity()
		assert.Equal(t, true, isValidSanity)

		// convert proof to bytes array
		proofBytes := proof.Bytes()
		assert.Equal(t, utils.SnPrivacyProofSize, len(proofBytes))

		// new SNPrivacyProof to set bytes array
		proof2 := new(SNPrivacyProof).Init()
		err = proof2.SetBytes(proofBytes)
		assert.Equal(t, nil, err)
		assert.Equal(t, proof, proof2)

		start = time.Now()
		res, err := proof2.Verify(nil)
		end = time.Since(start)
		fmt.Printf("Serial number verification time: %v\n", end)
		assert.Equal(t, true, res)
		assert.Equal(t, nil, err)
	}
}
