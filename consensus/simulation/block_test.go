package main

import (
	"github.com/incognitochain/incognito-chain/common"
	"testing"
)

func TestBlockGraph_AddBlock(t *testing.T) {
	root := Block{
		1,
		1,
		1,
		0,
		common.Hash{},
	}
	bg := NewBlockGraph("n1", &root)

	b2a := &Block{
		2,
		2,
		2,
		1,
		root.Hash(),
	}
	b2b := &Block{
		2,
		3,
		3,
		2,
		root.Hash(),
	}

	b3a := &Block{
		3,
		4,
		4,
		3,
		b2a.Hash(),
	}

	b3b := &Block{
		3,
		5,
		5,
		0,
		b2b.Hash(),
	}

	b4 := &Block{
		4,
		6,
		6,
		1,
		b3a.Hash(),
	}
	b5 := &Block{
		5,
		7,
		7,
		2,
		b4.Hash(),
	}

	bg.AddBlock(b2a)
	bg.AddBlock(b2b)
	bg.AddBlock(b3a)
	bg.AddBlock(b3b)
	bg.AddBlock(b4)
	bg.AddBlock(b5)

	bg.Print()
}
