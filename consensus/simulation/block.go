package main

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"os"
)

type GraphNode struct {
	block common.BlockInterface
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

func NewBlockGraph(name string, rootBlock common.BlockInterface) *BlockGraph {
	s := &BlockGraph{name: name}
	s.leaf = make(map[common.Hash]*GraphNode)
	s.node = make(map[common.Hash]*GraphNode)
	s.root = &GraphNode{
		rootBlock,
		*rootBlock.Hash(),
		nil,
		make(map[common.Hash]*GraphNode),
	}
	s.leaf[*rootBlock.Hash()] = s.root
	s.node[*rootBlock.Hash()] = s.root
	s.confirmBlock = s.root
	return s
}

func (s *BlockGraph) AddBlock(b common.BlockInterface) {
	newBlockHash := *b.Hash()
	for h, v := range s.node {
		if h == b.GetPrevBlockHash() {
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

func (s *BlockGraph) GetBestViewBlock() common.BlockInterface {
	s.traverse(s.root)
	s.updateConfirmBlock(s.bestView)
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
	maxTimeSlot := int64(0)
	for k, v := range s.node {
		shortK := k.String()[0:5]
		dotContent += fmt.Sprintf(`%s_%d_%s [label = "%d:%s"]`, s.name, v.block.GetHeight(), string(shortK), v.block.GetHeight(), string(shortK)) + "\n"
		dotContent += fmt.Sprintf(`{rank=same; %s_%d_%s; slot_%d;}`, s.name, v.block.GetHeight(), string(shortK), GetTimeSlot(v.block.GetTimeStamp())) + "\n"
		if GetTimeSlot(v.block.GetTimeStamp()) > maxTimeSlot {
			maxTimeSlot = GetTimeSlot(v.block.GetTimeStamp())
		}
	}

	for i := int64(0); i < maxTimeSlot; i++ {
		dotContent += fmt.Sprintf("slot_%d -> slot_%d;", i, i+1) + "\n"
	}

	dotContent += s.edgeStr
	dotContent += `}`

	fd, _ := os.OpenFile(s.name+".dot", os.O_WRONLY|os.O_CREATE, 0666)
	fd.Truncate(0)
	fd.Write([]byte(dotContent))
	fd.Close()

}

func (s *BlockGraph) traverse(n *GraphNode) {
	if n.next != nil && len(n.next) != 0 {
		for h, v := range n.next {
			s.edgeStr += fmt.Sprintf("%s_%d_%s -> %s_%d_%s;\n", s.name, n.block.GetHeight(), string(n.block.Hash().String()[0:5]), s.name, v.block.GetHeight(), string(h.String()[0:5]))
			s.traverse(v)
		}
	} else {
		if s.bestView == nil {
			s.bestView = n
		} else {
			if n.block.GetHeight() > s.bestView.block.GetHeight() {
				s.bestView = n
			}
			if n.block.GetHeight() == s.bestView.block.GetHeight() && GetTimeSlot(n.block.GetTimeStamp()) < GetTimeSlot(s.bestView.block.GetTimeStamp()) {
				s.bestView = n
			}
		}
	}
}

func (s *BlockGraph) updateConfirmBlock(node *GraphNode) {
	_1block := node.prev
	if _1block == nil {
		s.confirmBlock = node
		return
	}
	_2block := _1block.prev
	if _2block == nil {
		s.confirmBlock = _1block
		return
	}
	if GetTimeSlot(_2block.block.GetTimeStamp()) == GetTimeSlot(_1block.block.GetTimeStamp())-1 && GetTimeSlot(_2block.block.GetTimeStamp()) == GetTimeSlot(node.block.GetTimeStamp())-2 {
		s.confirmBlock = _2block
		return
	}
	s.updateConfirmBlock(_1block)
}
