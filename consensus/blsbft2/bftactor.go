package blsbftv2

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus"
	"github.com/incognitochain/incognito-chain/consensus/signatureschemes/blsmultisig"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wire"
)

type BLSBFT struct {
	Chain    blockchain.ChainInterface
	Node     consensus.NodeInterface
	ChainKey string
	PeerID   string

	UserKeySet   *MiningKey
	BFTMessageCh chan wire.MessageBFT
	isStarted    bool
	StopCh       chan struct{}
	logger       common.Logger

	currentTimeslot          uint64
	bestProposeBlock         string
	ongoingViews             map[string]viewConsensusInstance
	lockBlocksToCollectVotes sync.RWMutex
}

type viewConsensusInstance struct {
	Engine         *BLSBFT
	View           blockchain.ChainViewInterface
	Block          common.BlockInterface
	Votes          map[string]BFTVote
	UnconfirmVotes []BFTVote
	lockVote       sync.RWMutex
	Timeslot       uint64

	Committee    []incognitokey.CommitteePublicKey
	CommitteeBLS struct {
		StringList []string
		ByteList   []blsmultisig.PublicKey
	}
}

func (e *BLSBFT) GetConsensusName() string {
	return consensusName
}

func (e *BLSBFT) Stop() error {
	if e.isStarted {
		select {
		case <-e.StopCh:
			return nil
		default:
			close(e.StopCh)
		}
		e.isStarted = false
	}
	return consensus.NewConsensusError(consensus.ConsensusAlreadyStoppedError, errors.New(e.ChainKey))
}

func (e *BLSBFT) Start() error {
	if e.isStarted {
		return consensus.NewConsensusError(consensus.ConsensusAlreadyStartedError, errors.New(e.ChainKey))
	}
	e.isStarted = true
	e.StopCh = make(chan struct{})

	ticker := time.Tick(500 * time.Millisecond)
	e.logger.Info("start bls-bftv2 consensus for chain", e.ChainKey)
	go func() {
		fmt.Println("action")
		for { //actor loop
			select {
			case <-e.StopCh:
				return

			case <-ticker:

				// metrics.SetGlobalParam("RoundKey", getRoundKey(e.RoundData.NextHeight, e.RoundData.Round), "Phase", e.RoundData.State)

				// pubKey := e.UserKeySet.GetPublicKey()
				// if common.IndexOfStr(pubKey.GetMiningKeyBase58(consensusName), e.RoundData.CommitteeBLS.StringList) == -1 {
				// 	e.enterNewRound()
				// 	continue
				// }

				// if !e.Chain.IsReady() {
				// 	e.isOngoing = false
				// 	//fmt.Println("CONSENSUS: ticker 1")
				// 	continue
				// }

				// if !e.isInTimeFrame() || e.RoundData.State == "" {
				// 	e.enterNewRound()
				// }

				// switch e.RoundData.State {
				// case listenPhase:
				// 	// timeout or vote nil?
				// 	//fmt.Println("CONSENSUS: listen phase 1")
				// 	if e.Chain.CurrentHeight() == e.RoundData.NextHeight {
				// 		e.enterNewRound()
				// 		continue
				// 	}
				// 	roundKey := getRoundKey(e.RoundData.NextHeight, e.RoundData.Round)
				// 	if e.Blocks[roundKey] != nil {
				// 		// metrics.SetGlobalParam("ReceiveBlockTime", time.Since(e.RoundData.TimeStart).Seconds())
				// 		//fmt.Println("CONSENSUS: listen phase 2")
				// 		if err := e.validatePreSignBlock(e.Blocks[roundKey]); err != nil {
				// 			delete(e.Blocks, roundKey)
				// 			e.logger.Error(err)
				// 			continue
				// 		}

				// 		if e.RoundData.Block == nil {
				// 			blockData, _ := json.Marshal(e.Blocks[roundKey])
				// 			msg, _ := MakeBFTProposeMsg(blockData, e.ChainKey, e.UserKeySet)
				// 			go e.Node.PushMessageToChain(msg, e.Chain)

				// 			e.RoundData.Block = e.Blocks[roundKey]
				// 			e.RoundData.BlockHash = *e.RoundData.Block.Hash()
				// 			valData, err := DecodeValidationData(e.RoundData.Block.GetValidationField())
				// 			if err != nil {
				// 				e.logger.Error(err)
				// 				continue
				// 			}
				// 			e.RoundData.BlockValidateData = *valData
				// 			e.enterVotePhase()
				// 		}
				// 	}
				// case votePhase:
				// 	if e.RoundData.NotYetSendVote {
				// 		err := e.sendVote()
				// 		if err != nil {
				// 			e.logger.Error(err)
				// 			continue
				// 		}
				// 	}
				// 	if !(new(common.Hash).IsEqual(&e.RoundData.BlockHash)) && e.isHasMajorityVotes() {
				// 		e.RoundData.lockVotes.Lock()
				// 		aggSig, brigSigs, validatorIdx, err := combineVotes(e.RoundData.Votes, e.RoundData.CommitteeBLS.StringList)
				// 		e.RoundData.lockVotes.Unlock()
				// 		if err != nil {
				// 			e.logger.Error(err)
				// 			continue
				// 		}

				// 		e.RoundData.BlockValidateData.AggSig = aggSig
				// 		e.RoundData.BlockValidateData.BridgeSig = brigSigs
				// 		e.RoundData.BlockValidateData.ValidatiorsIdx = validatorIdx

				// 		validationDataString, _ := EncodeValidationData(e.RoundData.BlockValidateData)
				// 		e.RoundData.Block.(blockValidation).AddValidationField(validationDataString)

				// 		//TODO: check issue invalid sig when swap
				// 		//TODO 0xakk0r0kamui trace who is malicious node if ValidateCommitteeSig return false
				// 		err = e.ValidateCommitteeSig(e.RoundData.Block, e.RoundData.Committee)
				// 		if err != nil {
				// 			fmt.Print("\n")
				// 			e.logger.Critical(e.RoundData.Block.GetValidationField())
				// 			fmt.Print("\n")
				// 			e.logger.Critical(e.RoundData.Committee)
				// 			fmt.Print("\n")
				// 			for _, member := range e.RoundData.Committee {
				// 				fmt.Println(base58.Base58Check{}.Encode(member.MiningPubKey[consensusName], common.Base58Version))
				// 			}
				// 			e.logger.Critical(err)
				// 			continue
				// 		}

				// 		if err := e.Chain.GetBestView().InsertAndBroadcastBlock(e.RoundData.Block); err != nil {
				// 			e.logger.Error(err)
				// 			if blockchainError, ok := err.(*blockchain.BlockChainError); ok {
				// 				if blockchainError.Code != blockchain.ErrCodeMessage[blockchain.DuplicateShardBlockError].Code {
				// 					e.logger.Error(err)
				// 				}
				// 			}
				// 			continue
				// 		}
				// 		// e.Node.PushMessageToAll()
				// 		// metrics.SetGlobalParam("CommitTime", time.Since(time.Unix(e.Chain.GetLastBlockTimeStamp(), 0)).Seconds())
				// 		e.logger.Warn("Commit block! Wait for next round")
				// 		e.enterNewRound()
				// 	}
				// }
			}
		}
	}()
	return nil
}

func (e BLSBFT) NewInstance(chain blockchain.ChainInterface, chainKey string, node consensus.NodeInterface, logger common.Logger) consensus.ConsensusInterface {
	var newInstance BLSBFT
	newInstance.Chain = chain
	newInstance.ChainKey = chainKey
	newInstance.Node = node
	newInstance.UserKeySet = e.UserKeySet
	newInstance.logger = logger
	return &newInstance
}

func init() {
	consensus.RegisterConsensus(common.BlsConsensus2, &BLSBFT{})
}
