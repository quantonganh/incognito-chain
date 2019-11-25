package main

import (
	"github.com/incognitochain/incognito-chain/common"
	"testing"
)

func TestBlockGraph_AddBlock(t *testing.T) {

	root := NewBlock(1, START_TIME, common.Hash{})
	bg := NewBlockGraph("n1", root)
	b2a := NewBlock(2, NextTimeSlot(root.GetTimeStamp()), *root.Hash())
	b2b := NewBlock(2, NextTimeSlot(b2a.GetTimeStamp()), *root.Hash())
	b3a := NewBlock(3, NextTimeSlot(b2b.GetTimeStamp()), *b2a.Hash())
	b3b := NewBlock(3, NextTimeSlot(b3a.GetTimeStamp()), *b2b.Hash())
	b4 := NewBlock(4, NextTimeSlot(b3b.GetTimeStamp()), *b3a.Hash())
	b5 := NewBlock(5, NextTimeSlot(b4.GetTimeStamp()), *b4.Hash())
	b6 := NewBlock(6, NextTimeSlot(b5.GetTimeStamp()), *b5.Hash())

	bg.AddBlock(b2a)
	bg.AddBlock(b2b)
	bg.AddBlock(b3a)
	bg.AddBlock(b3b)
	bg.AddBlock(b4)
	bg.AddBlock(b5)
	bg.AddBlock(b6)

	bg.Print()
}
