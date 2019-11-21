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
	bg := NewBlockGraph("n1", &root)
	b2 := &Block{
		2,
		time.Now().Unix(),
		0,
		root.Hash(),
	}
	b3 := &Block{
		3,
		time.Now().Unix(),
		1,
		root.Hash(),
	}
	b4 := &Block{
		4,
		time.Now().Unix(),
		1,
		b2.Hash(),
	}

	bg.AddBlock(b2)
	bg.AddBlock(b3)
	bg.AddBlock(b4)

	bg.Print()
}
