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
	"github.com/patrickmn/go-cache"
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
	Logger       common.Logger

	currentTimeslot   uint64
	bestProposeBlock  string
	onGoingBlocks     map[string]*blockConsensusInstance
	lockOnGoingBlocks sync.RWMutex

	isAlreadyProposeBlock bool

	viewCommitteesCache *cache.Cache // [committeeHash]:committeeDecodeStruct
}

type blockConsensusInstance struct {
	Engine       *BLSBFT
	View         blockchain.ChainViewInterface
	ConsensusCfg consensusConfig
	Block        common.BlockInterface
	Votes        map[string]*BFTVote
	lockVote     sync.RWMutex
	Timeslot     uint64
	Phase        string
	Committee    committeeDecode
}

type consensusConfig struct {
	Slottime string
}
type committeeDecode struct {
	Committee  []incognitokey.CommitteePublicKey
	StringList []string
	ByteList   []blsmultisig.PublicKey
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

	ticker := time.Tick(1 * time.Second)
	e.Logger.Info("start bls-bftv2 consensus for chain", e.ChainKey)
	go func() {
		fmt.Println("action")
		for { //actor loop
			select {
			case <-e.StopCh:
				return
			case <-ticker:
				e.lockOnGoingBlocks.Lock()

				e.lockOnGoingBlocks.Unlock()
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
	newInstance.Logger = logger
	return &newInstance
}

func init() {
	consensus.RegisterConsensus(common.BlsConsensus2, &BLSBFT{})
}
