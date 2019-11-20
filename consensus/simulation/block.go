package main

import (
	"encoding/json"
	"github.com/incognitochain/incognito-chain/common"
)

type Block struct {
	height      uint64
	timestamp   int64
	proposerIdx int
	prevBlock   common.Hash
}

func (s *Block) Hash() common.Hash {
	b, _ := json.Marshal(s)
	return common.HashH(b)
}

type GraphNode struct {
	block *Block
	hash  common.Hash
	prev  *GraphNode
	next  *GraphNode
}

type BlockGraph struct {
	root *GraphNode
	leaf map[common.Hash]*GraphNode
}

func NewBlockGraph(rootBlock *Block) *BlockGraph {
	s := &BlockGraph{}
	s.leaf = make(map[common.Hash]*GraphNode)
	s.root = &GraphNode{
		rootBlock,
		rootBlock.Hash(),
		nil,
		nil,
	}
	s.leaf[rootBlock.Hash()] = s.root
	return s
}

func (s *BlockGraph) AddBlock(b *Block) {
	newBlockHash := b.Hash()
	for h, v := range s.leaf {
		if h == newBlockHash {
			delete(s.leaf, h)
			s.leaf[newBlockHash] = &GraphNode{
				b,
				newBlockHash,
				v,
				nil,
			}
			v.next = s.leaf[newBlockHash]
		}
	}
}

func (s *BlockGraph) Print() {

}
