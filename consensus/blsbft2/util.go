package blsbftv2

import (
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus"
)

func (e *BLSBFT) getTimeSinceLastBlock() time.Duration {
	return time.Since(time.Unix(int64(e.Chain.GetLastBlockTimeStamp()), 0))
}

func (e *BLSBFT) waitForNextTimeslot() bool {
	timeSinceLastBlk := e.getTimeSinceLastBlock()
	if timeSinceLastBlk >= e.Chain.GetMinBlkInterval() {
		return false
	} else {
		//fmt.Println("\n\nWait for", e.Chain.GetMinBlkInterval()-timeSinceLastBlk, "\n\n")
		return true
	}
}

func (e *BLSBFT) ExtractBridgeValidationData(block common.BlockInterface) ([][]byte, []int, error) {
	valData, err := DecodeValidationData(block.GetValidationField())
	if err != nil {
		return nil, nil, consensus.NewConsensusError(consensus.UnExpectedError, err)
	}
	return valData.BridgeSig, valData.ValidatiorsIdx, nil
}
