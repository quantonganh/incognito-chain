package blsbftv2

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus"
	"github.com/incognitochain/incognito-chain/consensus/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/wire"
)

func (e BLSBFT) preValidateCheck(block *common.BlockInterface) bool {
	// blockViewHash := block.GetPreviousViewHash()

	return true
}

func (e *BLSBFT) proposeBlock(timeslot uint64) error {
	bestView := e.Chain.GetBestView()
	bestViewHash := bestView.Hash().String()
	bestProposedBlockHash, ok := e.bestProposeBlockOfView[bestViewHash]

	if ok {
		//re-broadcast best proposed block

		// consensusCfg, _ := parseConsensusConfig(bestView.GetConsensusConfig())
		// consensusSlottime, _ := time.ParseDuration(consensusCfg.Slottime)
		// if e.currentTimeslotOfViews[bestViewHash] == getTimeSlot(bestView.GetGenesisTime(), time.Now().Unix(), int64(consensusSlottime.Seconds())) {
		// }

		blockData, _ := json.Marshal(e.onGoingBlocks[bestProposedBlockHash].Block)
		msg, _ := MakeBFTProposeMsg(blockData, e.ChainKey, e.UserKeySet)
		go e.Node.PushMessageToChain(msg, e.Chain)
	} else {
		//create block and boardcast block
		if isProducer(timeslot, bestView.GetCommittee(), e.UserKeySet.GetPublicKeyBase58()) != nil {
			return errors.New("I'm not the block producer")
		}
		block, err := bestView.CreateNewBlock(timeslot)
		if err != nil {
			return err
		}
		validationData := e.CreateValidationData(block)
		validationDataString, _ := EncodeValidationData(validationData)
		block.(blockValidation).AddValidationField(validationDataString)

		blockHash := block.Hash().String()

		if err := e.createBlockConsensusInstance(bestView, blockHash); err != nil {
			return err
		}

		e.lockOnGoingBlocks.RLock()
		instance := e.onGoingBlocks[blockHash]
		e.lockOnGoingBlocks.RUnlock()

		err = instance.addBlock(block)
		if err != nil {
			return err
		}

		e.bestProposeBlockOfView[bestViewHash] = blockHash
		e.proposedBlockOnView.BlockHash = blockHash
		e.proposedBlockOnView.ViewHash = bestViewHash
		blockData, _ := json.Marshal(block)
		msg, _ := MakeBFTProposeMsg(blockData, e.ChainKey, e.UserKeySet)
		go e.Node.PushMessageToChain(msg, e.Chain)
	}

	return nil
}

func (e BLSBFT) processProposeMsg(proposeMsg *BFTPropose) error {
	// proposer only propose once
	// voter can vote on multi-views

	block, err := e.Chain.UnmarshalBlock(proposeMsg.Block)
	if err != nil {
		return err
	}
	currentViewTimeslot := e.currentTimeslotOfViews[block.GetPreviousViewHash().String()]

	if block.GetTimeslot() > currentViewTimeslot {
		// hmm... something wrong with local clock?
		return fmt.Errorf("this propose block has timeslot higher than current timeslot. BLOCK:%v CURRENT:%v", block.GetTimeslot(), currentViewTimeslot)
	}
	blockHash := block.Hash().String()
	if blockHash == e.Chain.GetBestView().GetTipBlock().Hash().String() {
		//send this block
	}
	e.lockOnGoingBlocks.RLock()
	if _, ok := e.onGoingBlocks[blockHash]; ok {
		if e.onGoingBlocks[blockHash].Block != nil {
			e.lockOnGoingBlocks.RUnlock()
			return errors.New("already received this propose block")
		}
	}
	e.lockOnGoingBlocks.RUnlock()

	view, err := e.Chain.GetViewByHash(block.GetPreviousViewHash())
	if err != nil {
		if block.GetHeight() > e.Chain.GetBestView().CurrentHeight() {
			//request block
			return nil
		}
		return err
	}

	consensusCfg, err := parseConsensusConfig(view.GetConsensusConfig())
	if err != nil {
		return err
	}
	consensusSlottime, err := time.ParseDuration(consensusCfg.Slottime)
	if err != nil {
		return err
	}
	if view.CurrentHeight() == e.Chain.GetBestView().CurrentHeight() {
		if err := e.validateProducer(block, view, int64(consensusSlottime.Seconds()), view.GetCommittee(), e.Logger); err != nil {
			return err
		}
		if len(e.onGoingBlocks) > 0 {
			e.lockOnGoingBlocks.RLock()
			if bestBlockHash, ok := e.bestProposeBlockOfView[block.GetPreviousViewHash().String()]; ok {
				if block.GetTimeslot() < e.onGoingBlocks[bestBlockHash].Timeslot {
					e.lockOnGoingBlocks.RUnlock()

					if err := e.createBlockConsensusInstance(view, blockHash); err != nil {
						return err
					}
					e.bestProposeBlockOfView[block.GetPreviousViewHash().String()] = blockHash
				}
			} else {
				defer e.lockOnGoingBlocks.RUnlock()
				instance := e.onGoingBlocks[blockHash]
				err := instance.addBlock(block)
				if err != nil {
					return err
				}
			}
		} else {
			err := view.ValidatePreSignBlock(block)
			if err != nil {
				return err
			}
			if err := e.createBlockConsensusInstance(view, blockHash); err != nil {
				return err
			}
			e.bestProposeBlockOfView[block.GetPreviousViewHash().String()] = blockHash
		}
	}
	return nil
}

func (e *BLSBFT) processVoteMsg(vote *BFTVote) error {
	e.lockOnGoingBlocks.RLock()
	viewHash, err := common.Hash{}.NewHashFromStr(vote.ViewHash)
	if err != nil {
		return err
	}
	view, err := e.Chain.GetViewByHash(viewHash)
	if err != nil {
		return err
	}

	if _, ok := e.onGoingBlocks[vote.BlockHash]; !ok {
		e.lockOnGoingBlocks.RUnlock()
		if err := e.createBlockConsensusInstance(view, vote.BlockHash); err != nil {
			return err
		}
		e.lockOnGoingBlocks.RLock()
	}
	defer e.lockOnGoingBlocks.RUnlock()

	onGoingBlock := e.onGoingBlocks[vote.BlockHash]
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

func (blockCI *blockConsensusInstance) addVote(vote *BFTVote) error {
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

func (blockCI *blockConsensusInstance) confirmVote(blockHash *common.Hash, vote *BFTVote) error {
	data := blockHash.GetBytes()
	data = append(data, vote.BLS...)
	data = append(data, vote.BRI...)
	data = common.HashB(data)
	var err error
	vote.VoteSig, err = blockCI.Engine.UserKeySet.BriSignData(data)
	return err
}

func (blockCI *blockConsensusInstance) createAndSendVote() (BFTVote, error) {
	var vote BFTVote

	pubKey := blockCI.Engine.UserKeySet.GetPublicKey()
	selfIdx := common.IndexOfStr(pubKey.GetMiningKeyBase58(consensusName), blockCI.Committee.StringList)

	blsSig, err := blockCI.Engine.UserKeySet.BLSSignData(blockCI.Block.Hash().GetBytes(), selfIdx, blockCI.Committee.ByteList)
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

func (blockCI *blockConsensusInstance) initInstance(view blockchain.ChainViewInterface) error {
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

func getTimeSlot(genesisTime int64, pointInTime int64, slotTime int64) uint64 {
	slotTimeDur := time.Duration(slotTime)
	blockTime := time.Unix(pointInTime, 0)
	timePassed := blockTime.Sub(time.Unix(genesisTime, 0)).Round(slotTimeDur)
	timeSlot := uint64(int64(timePassed.Seconds()) / slotTime)
	return timeSlot
}

func validateProducerPosition(block common.BlockInterface, genesisTime int64, slotTime int64, committee []incognitokey.CommitteePublicKey) error {
	timeSlot := getTimeSlot(genesisTime, block.GetBlockTimestamp(), slotTime)
	if block.GetTimeslot() != timeSlot {
		return consensus.NewConsensusError(consensus.InvalidTimeslotError, fmt.Errorf("Timeslot should be %v instead of %v", timeSlot, block.GetTimeslot()))
	}
	return isProducer(timeSlot, committee, block.GetProducer())
}

func getProducerPosition(timeslot uint64, committeeLen uint64) uint64 {
	return timeslot % committeeLen
}

func isProducer(timeslot uint64, committee []incognitokey.CommitteePublicKey, producerPbk string) error {
	producerPosition := getProducerPosition(timeslot, uint64(len(committee)))
	tempProducer, err := committee[producerPosition].ToBase58()
	if err != nil {
		return err
	}
	if tempProducer != producerPbk {
		return consensus.NewConsensusError(consensus.UnExpectedError, fmt.Errorf("Producer should be should be %v instead of %v", tempProducer, producerPbk))
	}
	return nil
}

func (blockCI *blockConsensusInstance) addBlock(block common.BlockInterface) error {
	blockCI.Block = block
	blockCI.Timeslot = block.GetTimeslot()
	blockCI.Phase = votePhase
	return nil
}

func (e *BLSBFT) createBlockConsensusInstance(view blockchain.ChainViewInterface, blockHash string) error {
	e.lockOnGoingBlocks.Lock()
	defer e.lockOnGoingBlocks.Unlock()
	var blockCI blockConsensusInstance
	blockCI.View = view
	blockCI.Phase = listenPhase

	var cfg consensusConfig
	err := json.Unmarshal([]byte(view.GetConsensusConfig()), &cfg)
	if err != nil {
		return err
	}
	blockCI.ConsensusCfg = cfg
	cmHash := view.GetCommitteeHash()
	cmCache, ok := e.viewCommitteesCache.Get(cmHash.String())
	if !ok {
		committee := view.GetCommittee()
		var cmDecode committeeDecode
		cmDecode.Committee = committee
		cmDecode.ByteList = []blsmultisig.PublicKey{}
		cmDecode.StringList = []string{}
		for _, member := range cmDecode.Committee {
			cmDecode.ByteList = append(cmDecode.ByteList, member.MiningPubKey[consensusName])
		}
		committeeBLSString, err := incognitokey.ExtractPublickeysFromCommitteeKeyList(cmDecode.Committee, consensusName)
		if err != nil {
			return err
		}
		cmDecode.StringList = committeeBLSString
		e.viewCommitteesCache.Add(view.GetCommitteeHash().String(), cmDecode, committeeCacheCleanupTime)
		blockCI.Committee = cmDecode
	} else {
		blockCI.Committee = cmCache.(committeeDecode)
	}

	e.onGoingBlocks[blockHash] = &blockCI
	return nil
}
