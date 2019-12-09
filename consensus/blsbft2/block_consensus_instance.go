package blsbftv2

import (
	"sync"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
)

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
