package serialnumbernoprivacy

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/utils"
	"github.com/incognitochain/incognito-chain/privacyv1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPKSNNoPrivacy(t *testing.T) {
	for i:=0; i<1000; i++{
		// prepare witness for Serial number no privacy protocol
		sk := privacyv1.GeneratePrivateKey(privacyv1.RandBytes(10))
		skScalar := new(privacyv1.Scalar).FromBytesS(sk)
		if skScalar.ScalarValid() == false {
			fmt.Println("Invalid key value")
		}

		pk := privacyv1.GeneratePublicKey(sk)
		pkPoint, err := new(privacyv1.Point).FromBytesS(pk)
		if err != nil {
			fmt.Println("Invalid point key valu")
		}
		SND := privacyv1.RandomScalar()

		serialNumber := new(privacyv1.Point).Derive(privacyv1.PedCom.G[privacyv1.PedersenPrivateKeyIndex], skScalar, SND)

		witness := new(SNNoPrivacyWitness)
		witness.Set(serialNumber, pkPoint, SND, skScalar)

		// proving
		proof, err := witness.Prove(nil)
		assert.Equal(t, nil, err)

		//validate sanity proof
		isValidSanity := proof.ValidateSanity()
		assert.Equal(t, true, isValidSanity)

		// verify proof
		res, err := proof.Verify(nil)
		assert.Equal(t, true, res)
		assert.Equal(t, nil, err)

		// convert proof to bytes array
		proofBytes := proof.Bytes()
		assert.Equal(t, utils.SnNoPrivacyProofSize, len(proofBytes))

		// new SNPrivacyProof to set bytes array
		proof2 := new(SNNoPrivacyProof).Init()
		err = proof2.SetBytes(proofBytes)
		assert.Equal(t, nil, err)
		assert.Equal(t, proof, proof2)

		// verify proof
		res2, err := proof2.Verify(nil)
		assert.Equal(t, true, res2)
		assert.Equal(t, nil, err)
	}

}