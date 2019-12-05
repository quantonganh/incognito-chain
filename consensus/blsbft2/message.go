package blsbftv2

import (
	"encoding/json"

	"github.com/incognitochain/incognito-chain/consensus"
	"github.com/incognitochain/incognito-chain/wire"
)

const (
	MSG_PROPOSE    = "propose"
	MSG_VOTE       = "vote"
	MSG_REQUESTBLK = "getblk"
)

type BFTPropose struct {
	Block json.RawMessage
}

type BFTVote struct {
	BlockHash string
	Validator string
	BLS       []byte
	BRI       []byte
	VoteSig   []byte
}

type BFTRequestBlock struct {
	BlockHash string
	PeerID    string
}

func MakeBFTProposeMsg(block []byte, chainKey string, userKeySet *MiningKey) (wire.Message, error) {
	var proposeCtn BFTPropose
	proposeCtn.Block = block
	proposeCtnBytes, err := json.Marshal(proposeCtn)
	if err != nil {
		return nil, consensus.NewConsensusError(consensus.UnExpectedError, err)
	}
	msg, _ := wire.MakeEmptyMessage(wire.CmdBFT)
	msg.(*wire.MessageBFT).ChainKey = chainKey
	msg.(*wire.MessageBFT).Content = proposeCtnBytes
	msg.(*wire.MessageBFT).Type = MSG_PROPOSE
	return msg, nil
}

func MakeBFTVoteMsg(vote *BFTVote, chainKey string) (wire.Message, error) {
	voteCtnBytes, err := json.Marshal(vote)
	if err != nil {
		return nil, consensus.NewConsensusError(consensus.UnExpectedError, err)
	}
	msg, _ := wire.MakeEmptyMessage(wire.CmdBFT)
	msg.(*wire.MessageBFT).ChainKey = chainKey
	msg.(*wire.MessageBFT).Content = voteCtnBytes
	msg.(*wire.MessageBFT).Type = MSG_VOTE
	return msg, nil
}

func MakeBFTRequestBlk(request BFTRequestBlock, peerID string, chainKey string) (wire.Message, error) {
	requestCtnBytes, err := json.Marshal(request)
	if err != nil {
		return nil, consensus.NewConsensusError(consensus.UnExpectedError, err)
	}
	msg, _ := wire.MakeEmptyMessage(wire.CmdBFT)
	msg.(*wire.MessageBFT).ChainKey = chainKey
	msg.(*wire.MessageBFT).Content = requestCtnBytes
	msg.(*wire.MessageBFT).Type = MSG_REQUESTBLK
	return msg, nil
}
