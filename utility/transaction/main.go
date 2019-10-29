package main

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/database"
	_ "github.com/incognitochain/incognito-chain/database/lvdb"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/wallet"
	"path/filepath"
)

func main() {
	db, err := database.Open("leveldb", filepath.Join("./", "./"))
	if err != nil {
		fmt.Print("could not open connection to leveldb")
		fmt.Print(err)
		panic(err)
	}

	// init an genesis tx
	initGenesisTx(db)

	// init thank tx
	initThankTx(db)
}

func initGenesisTx(db database.DatabaseInterface) {
	var initTxs []string
	testUserkeyList := map[string]uint64{
		"112t8rnXBS7jJ4iqFon5rM66ex1Fc7sstNrJA9iMKgNURMUf3rywYfJ4c5Kpxw1BgL1frj9Nu5uL5vpemn9mLUW25CD1w7khX88WdauTVyKa": uint64(5000000000000000),
	}
	for privateKey, initAmount := range testUserkeyList {

		testUserKey, _ := wallet.Base58CheckDeserialize(privateKey)
		testUserKey.KeySet.InitFromPrivateKey(&testUserKey.KeySet.PrivateKey)

		testSalaryTX := transaction.Tx{}
		testSalaryTX.InitTxSalary(initAmount, &testUserKey.KeySet.PaymentAddress, &testUserKey.KeySet.PrivateKey, db, nil)
		initTx, _ := json.MarshalIndent(testSalaryTX, "", "  ")
		initTxs = append(initTxs, string(initTx))
	}
	fmt.Println(initTxs)
}

func initThankTx(db database.DatabaseInterface) {
	var initTxs []string
	testUserkeyList := map[string]string{
		"112t8rnXBS7jJ4iqFon5rM66ex1Fc7sstNrJA9iMKgNURMUf3rywYfJ4c5Kpxw1BgL1frj9Nu5uL5vpemn9mLUW25CD1w7khX88WdauTVyKa": "@abc",
	}
	for privateKey, info := range testUserkeyList {

		testUserKey, _ := wallet.Base58CheckDeserialize(privateKey)
		testUserKey.KeySet.InitFromPrivateKey(&testUserKey.KeySet.PrivateKey)

		testSalaryTX := transaction.Tx{}
		testSalaryTX.InitTxSalary(0, &testUserKey.KeySet.PaymentAddress, &testUserKey.KeySet.PrivateKey, db, nil)
		testSalaryTX.Info = []byte(info)
		initTx, _ := json.MarshalIndent(testSalaryTX, "", "  ")
		initTxs = append(initTxs, string(initTx))
	}
	fmt.Println(initTxs)
}
