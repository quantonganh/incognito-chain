package main

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
)

type Block struct {
	Height      uint64
	Timestamp   int64
	ProposerIdx int
	PrevBlock   common.Hash
}

func (s *Block) Hash() common.Hash {
	b, _ := json.Marshal(s)
	return common.HashH(b)
}

type GraphNode struct {
	block *Block
	hash  common.Hash
	prev  *GraphNode
	next  map[common.Hash]*GraphNode
}

type BlockGraph struct {
	name string
	root *GraphNode
	node map[common.Hash]*GraphNode
	leaf map[common.Hash]*GraphNode
}

func NewBlockGraph(name string, rootBlock *Block) *BlockGraph {
	s := &BlockGraph{name: name}
	s.leaf = make(map[common.Hash]*GraphNode)
	s.node = make(map[common.Hash]*GraphNode)
	s.root = &GraphNode{
		rootBlock,
		rootBlock.Hash(),
		nil,
		make(map[common.Hash]*GraphNode),
	}
	s.leaf[rootBlock.Hash()] = s.root
	s.node[rootBlock.Hash()] = s.root
	return s
}

func (s *BlockGraph) AddBlock(b *Block) {
	newBlockHash := b.Hash()
	for h, v := range s.node {
		if h == b.PrevBlock {
			delete(s.leaf, h)
			s.leaf[newBlockHash] = &GraphNode{
				b,
				newBlockHash,
				v,
				make(map[common.Hash]*GraphNode),
			}
			v.next[newBlockHash] = s.leaf[newBlockHash]
			s.node[newBlockHash] = s.leaf[newBlockHash]
		}
	}
}

func (s *BlockGraph) Print() {
	traverse(s.root)
}

func traverse(n *GraphNode) {
	fmt.Println("traverse", n.block.Height)
	if n.next != nil && len(n.next) != 0 {
		//b, _ := json.MarshalIndent(n.block, "", "\t")
		//fmt.Println("branch", string(b))
		for _, v := range n.next {
			traverse(v)
		}
	} else {
		//b, _ := json.MarshalIndent(n.block, "", "\t")
		//fmt.Println("leaf", string(b))
	}
}
