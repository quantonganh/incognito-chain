package blsbftv2

import (
	"time"

	"github.com/incognitochain/incognito-chain/common"
)

const (
	proposePhase  = "PROPOSE"
	listenPhase   = "LISTEN"
	votePhase     = "VOTE"
	commitPhase   = "COMMIT"
	newphase      = "NEWPHASE"
	consensusName = common.BlsConsensus2
)

//
const (
	committeeCacheCleanupTime = 40 * time.Minute
)
