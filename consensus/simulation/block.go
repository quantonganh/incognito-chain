package main

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
)

type Block struct {
	Height      uint64
	Timestamp   int64
	TimeSlot    int
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
	name         string
	root         *GraphNode
	node         map[common.Hash]*GraphNode
	leaf         map[common.Hash]*GraphNode
	edgeStr      string
	bestView     *GraphNode
	confirmBlock *GraphNode
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
	s.confirmBlock = s.root
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

func (s *BlockGraph) GetBestViewBlock() *Block {
	s.traverse(s.root)
	fmt.Println(s.bestView.block.Height, s.bestView.block.TimeSlot)
	return s.bestView.block
}

func (s *BlockGraph) Print() {
	s.edgeStr = ""
	s.traverse(s.root)

	dotContent := `digraph {
node [shape=record];
//    rankdir="LR";
newrank=true;
`
	maxTimeSlot := 0
	for k, v := range s.node {
		shortK := k.String()[0:5]
		dotContent += fmt.Sprintf(`%s_%d_%s [label = "%d:%s"]`, s.name, v.block.Height, string(shortK), v.block.Height, string(shortK)) + "\n"
		dotContent += fmt.Sprintf(`{rank=same; %s_%d_%s; slot_%d;}`, s.name, v.block.Height, string(shortK), v.block.TimeSlot) + "\n"
		if v.block.TimeSlot > maxTimeSlot {
			maxTimeSlot = v.block.TimeSlot
		}
	}

	for i := 1; i < maxTimeSlot; i++ {
		dotContent += fmt.Sprintf("slot_%d -> slot_%d;", i, i+1) + "\n"
	}

	dotContent += s.edgeStr
	dotContent += `}`
	fmt.Println(dotContent)
	s.updateConfirmBlock(s.bestView)

	b, _ := json.MarshalIndent(s.bestView.block, "", "\t")
	fmt.Println(string(b))
	c, _ := json.MarshalIndent(s.confirmBlock.block, "", "\t")
	fmt.Println(string(c))

}

func (s *BlockGraph) traverse(n *GraphNode) {
	if n.next != nil && len(n.next) != 0 {
		for h, v := range n.next {
			s.edgeStr += fmt.Sprintf("%s_%d_%s -> %s_%d_%s;\n", s.name, n.block.Height, string(n.block.Hash().String()[0:5]), s.name, v.block.Height, string(h.String()[0:5]))
			s.traverse(v)
		}
	} else {
		if s.bestView == nil {
			s.bestView = n
		} else {
			if n.block.Height > s.bestView.block.Height {
				s.bestView = n
			}
			if n.block.Height == s.bestView.block.Height && n.block.TimeSlot < s.bestView.block.TimeSlot {
				s.bestView = n
			}
		}
	}
}

func (s *BlockGraph) updateConfirmBlock(node *GraphNode) {
	_1block := node.prev
	_2block := _1block.prev
	if _1block == nil {
		s.confirmBlock = node
		return
	}
	if _2block == nil {
		s.confirmBlock = _1block
		return
	}
	if _2block.block.TimeSlot == _1block.block.TimeSlot-1 && _2block.block.TimeSlot == node.block.TimeSlot-2 {
		s.confirmBlock = _2block
		return
	}
	s.updateConfirmBlock(_1block)
}
