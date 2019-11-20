package privacyv1

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOneTimeAddress(t *testing.T) {
	for i := 0; i < 1000; i++ {
		seed := RandomScalar()
		privSpendKey := GeneratePrivateKey(seed.ToBytesS())
		paymentAddress := GeneratePaymentAddress(privSpendKey)
		viewingKey := GenerateViewingKey(privSpendKey)

		fmt.Printf("Public Spend key: %v\n", paymentAddress.Pk)

		rand := RandomScalar()
		index := 10

		oneTimeAddr, err := GenerateOneTimeAddrFromPaymentAddr(paymentAddress, rand, index)
		assert.Equal(t, nil, err)
		assert.Equal(t, Ed25519KeySize, len(oneTimeAddr.ToBytesS()))
		fmt.Printf("oneTimeAddr: %v\n", oneTimeAddr.ToBytesS())

		cmRand := new(Point).ScalarMult(PedCom.G[PedersenPrivateKeyIndex], rand)

		pubSpendKeyFromOneTimeAddr, _ , err := GetPublicKeyFromOneTimeAddress(oneTimeAddr, cmRand, viewingKey.Rk, index)
		fmt.Printf("Public Spend key from one time address: %v\n", pubSpendKeyFromOneTimeAddr.ToBytesS())

		res, _,  err := IsPairOneTimeAddr(oneTimeAddr, cmRand, viewingKey, index)
		assert.Equal(t, true, res)
	}

}