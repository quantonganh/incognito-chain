package main

import (
	"github.com/incognitochain/incognito-chain/common"
	"testing"
	"time"
)

func TestBlockGraph_AddBlock(t *testing.T) {
	root := Block{
		1,
		time.Now().Unix(),
		0,
		common.Hash{},
	}
	bg := NewBlockGraph(&root)
	bg.AddBlock(&Block{
		2,
		time.Now().Unix(),
		0,
		root.Hash(),
	})

	bg.AddBlock(&Block{
		3,
		time.Now().Unix(),
		0,
		root.Hash(),
	})

}
