package blsbftv2

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/wire"
)

func (e BLSBFT) preValidateCheck(block *common.BlockInterface) bool {
	// blockViewHash := block.GetPreviousViewHash()

	return true
}

func (e BLSBFT) getProposeBlock(view blockchain.ChainViewInterface, timeslot uint64) (common.BlockInterface, error) {
	if e.bestProposeBlock == "" {
		block, err := view.CreateNewBlock(timeslot)
		if err != nil {
			return nil, err
		}
		return block, nil
	}
	return nil, nil
}

func (e BLSBFT) processProposeMsg(proposeMsg *BFTPropose) error {
	block, err := e.Chain.UnmarshalBlock(proposeMsg.Block)
	if err != nil {
		return err
	}
	blockHash := block.Hash().String()
	if _, ok := e.onGoingBlocks[blockHash]; ok {
		return errors.New("already received this propose block")
	}
	view, err := e.Chain.GetViewByHash(block.GetPreviousViewHash())
	if err != nil {
		return err
	}
	if view.IsBestView() {
		if len(e.onGoingBlocks) > 0 {
			if e.onGoingBlocks[e.bestProposeBlock].Timeslot == block.GetTimeslot() {

			}
		}
		view.ValidatePreSignBlock(block)
	}
	return nil
}

func (e *BLSBFT) processVoteMsg(vote *BFTVote) error {
	onGoingBlock, ok := e.onGoingBlocks[vote.BlockHash]
	if ok {
		return errors.New("already received this propose block")
	}
	e.lockOnGoingBlocks.RLock()
	defer e.lockOnGoingBlocks.RUnlock()
	if err := onGoingBlock.addVote(vote); err != nil {
		return err
	}
	voteMsg, err := MakeBFTVoteMsg(vote, e.ChainKey)
	if err != nil {
		return err
	}
	e.Node.PushMessageToChain(voteMsg, e.Chain)
	return nil
}

func (e *BLSBFT) processRequestBlkMsg(requestMsg *BFTRequestBlock) error {
	return nil
}

func (e *BLSBFT) ProcessBFTMsg(msg *wire.MessageBFT) {
	switch msg.Type {
	case MSG_PROPOSE:
		var msgPropose BFTPropose
		err := json.Unmarshal(msg.Content, &msgPropose)
		if err != nil {
			fmt.Println(err)
			return
		}
		go e.processProposeMsg(&msgPropose)
	case MSG_VOTE:
		var msgVote BFTVote
		err := json.Unmarshal(msg.Content, &msgVote)
		if err != nil {
			fmt.Println(err)
			return
		}
		go e.processVoteMsg(&msgVote)
	case MSG_REQUESTBLK:
		var msgRequest BFTRequestBlock
		err := json.Unmarshal(msg.Content, &msgRequest)
		if err != nil {
			fmt.Println(err)
			return
		}
		go e.processRequestBlkMsg(&msgRequest)
	default:
		e.Logger.Critical("Unknown BFT message type")
		return
	}
}

func (e *BLSBFT) isInTimeslot(view blockchain.ChainViewInterface) bool {
	return false
}

func (blockCI *viewConsensusInstance) addVote(vote *BFTVote) error {
	blockCI.lockVote.Lock()
	defer blockCI.lockVote.Unlock()
	if _, ok := blockCI.Votes[vote.Validator]; !ok {
		return errors.New("already received this vote")
	}
	err := validateVote(vote)
	if err != nil {
		return err
	}
	blockCI.Votes[vote.Validator] = vote
	return nil
}

func (blockCI *viewConsensusInstance) confirmVote(blockHash *common.Hash, vote *BFTVote) error {
	data := blockHash.GetBytes()
	data = append(data, vote.BLS...)
	data = append(data, vote.BRI...)
	data = common.HashB(data)
	var err error
	vote.VoteSig, err = blockCI.Engine.UserKeySet.BriSignData(data)
	return err
}

func (blockCI *viewConsensusInstance) createAndSendVote() (BFTVote, error) {
	var vote BFTVote

	pubKey := blockCI.Engine.UserKeySet.GetPublicKey()
	selfIdx := common.IndexOfStr(pubKey.GetMiningKeyBase58(consensusName), blockCI.CommitteeBLS.StringList)

	blsSig, err := blockCI.Engine.UserKeySet.BLSSignData(blockCI.Block.Hash().GetBytes(), selfIdx, blockCI.CommitteeBLS.ByteList)
	if err != nil {
		return vote, consensus.NewConsensusError(consensus.UnExpectedError, err)
	}
	bridgeSig := []byte{}
	if metadata.HasBridgeInstructions(blockCI.Block.GetInstructions()) {
		bridgeSig, err = blockCI.Engine.UserKeySet.BriSignData(blockCI.Block.Hash().GetBytes())
		if err != nil {
			return vote, consensus.NewConsensusError(consensus.UnExpectedError, err)
		}
	}

	vote.BLS = blsSig
	vote.BRI = bridgeSig
	vote.Validator = pubKey.GetMiningKeyBase58(consensusName)

	msg, err := MakeBFTVoteMsg(&vote, blockCI.Engine.ChainKey)
	if err != nil {
		return vote, consensus.NewConsensusError(consensus.UnExpectedError, err)
	}

	blockCI.Votes[pubKey.GetMiningKeyBase58(consensusName)] = &vote
	blockCI.Engine.Logger.Info("sending vote...")
	go blockCI.Engine.Node.PushMessageToChain(msg, blockCI.Engine.Chain)
	return vote, nil
}

func validateProposeBlock(block common.BlockInterface, view blockchain.ChainViewInterface) (BFTVote, error) {
	err := view.ValidatePreSignBlock(block)
	if err != nil {
		return BFTVote{}, err
	}
	var v BFTVote

	return v, nil
}

func (blockCI *viewConsensusInstance) initInstance(view blockchain.ChainViewInterface) error {
	return nil
}

func (vote *BFTVote) signVote(signFunc func(data []byte) ([]byte, error)) error {
	data := []byte(vote.BlockHash)
	data = append(data, vote.BLS...)
	data = append(data, vote.BRI...)
	data = common.HashB(data)
	var err error
	vote.VoteSig, err = signFunc(data)
	return err
}

func (e *BLSBFT) getTimeSlot() uint64 {
	return uint64(e.Chain.GetGenesisTime())
}

func validateProducerPosition(block common.BlockInterface, genesisTime int64, slotTime uint64, committee []incognitokey.CommitteePublicKey) error {
	// producerPosition := (lastProposerIndex + block.GetRound()) % len(committee)
	// tempProducer, err := committee[producerPosition].ToBase58()
	// if err != nil {
	// 	return err
	// }
	// if tempProducer == block.GetProducer() {
	// 	return nil
	// }
	// return consensus.NewConsensusError(consensus.UnExpectedError, errors.New("Producer should be should be :"+tempProducer))
	return nil
}
