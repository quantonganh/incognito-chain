package oneoutofmany

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/privacyv1"
	"github.com/incognitochain/incognito-chain/privacyv1/zeroknowledge/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	m.Run()
}

var _ = func() (_ struct{}) {
	fmt.Println("This runs before init()!")
	privacyv1.Logger.Init(common.NewBackend(nil).Logger("test", true))
	return
}()

//TestPKOneOfMany test protocol for one of many Commitment is Commitment to zero
func TestPKOneOfMany(t *testing.T) {
	// prepare witness for Out out of many protocol
	for i := 0; i < 10; i++ {
		witness := new(OneOutOfManyWitness)

		//indexIsZero := int(common.RandInt() % privacyv1.CommitmentRingSize)
		indexIsZero := 0

		// list of commitments
		commitments := make([]*privacyv1.Point, privacyv1.CommitmentRingSize)
		values := make([]*privacyv1.Scalar, privacyv1.CommitmentRingSize)
		randoms := make([]*privacyv1.Scalar, privacyv1.CommitmentRingSize)

		for i := 0; i < privacyv1.CommitmentRingSize; i++ {
			values[i] = privacyv1.RandomScalar()
			randoms[i] = privacyv1.RandomScalar()
			commitments[i] = privacyv1.PedCom.CommitAtIndex(values[i], randoms[i], privacyv1.PedersenSndIndex)
		}

		// create Commitment to zero at indexIsZero
		values[indexIsZero] = new(privacyv1.Scalar).FromUint64(0)
		commitments[indexIsZero] = privacyv1.PedCom.CommitAtIndex(values[indexIsZero], randoms[indexIsZero], privacyv1.PedersenSndIndex)

		witness.Set(commitments, randoms[indexIsZero], uint64(indexIsZero))
		start := time.Now()
		proof, err := witness.Prove()
		assert.Equal(t, nil, err)
		end := time.Since(start)
		//fmt.Printf("One out of many proving time: %v\n", end)

		//fmt.Printf("Proof: %v\n", proof)

		// validate sanity for proof
		isValidSanity := proof.ValidateSanity()
		assert.Equal(t, true, isValidSanity)

		// verify the proof
		start = time.Now()
		res, err := proof.Verify()
		end = time.Since(start)
		fmt.Printf("One out of many verification time: %v\n", end)
		assert.Equal(t, true, res)
		assert.Equal(t, nil, err)

		//Convert proof to bytes array
		proofBytes := proof.Bytes()
		assert.Equal(t, utils.OneOfManyProofSize, len(proofBytes))

		// revert bytes array to proof
		proof2 := new(OneOutOfManyProof).Init()
		err = proof2.SetBytes(proofBytes)
		assert.Equal(t, nil, err)
		proof2.Statement.Commitments = commitments
		assert.Equal(t, proof, proof2)

		// verify the proof
		start = time.Now()
		res, err = proof2.Verify()
		end = time.Since(start)
		fmt.Printf("One out of many verification time: %v\n", end)
		assert.Equal(t, true, res)
		assert.Equal(t, nil, err)

	}
}

