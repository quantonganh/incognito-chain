package blockchain

import (
	"fmt"
	"sync"

	"github.com/incognitochain/incognito-chain/common"
	libp2p "github.com/libp2p/go-libp2p-peer"
)

func (blockchain *BlockChain) OnPeerStateReceived(beacon *ChainState, shard *map[byte]ChainState, shardToBeaconPool *map[byte][]uint64, crossShardPool *map[byte]map[byte][]uint64, peerID libp2p.ID) {
	if blockchain.IsTest {
		return
	}
	if beacon.Timestamp < GetBeaconBestState().BestBlock.Header.Timestamp && beacon.Height > GetBeaconBestState().BestBlock.Header.Height {
		return
	}

	var (
		userRole    string
		userShardID byte
	)

	userRole, userShardIDInt := blockchain.config.ConsensusEngine.GetUserLayer()
	if userRole == common.ShardRole {
		userShardID = byte(userShardIDInt)
	}

	pState := &peerState{
		Shard:  make(map[byte]*ChainState),
		Beacon: beacon,
		Peer:   peerID,
	}
	nodeMode := blockchain.config.NodeMode
	if userRole == common.BeaconRole {
		pState.ShardToBeaconPool = shardToBeaconPool
		for shardID := byte(0); shardID < byte(common.MaxShardNumber); shardID++ {
			if shardState, ok := (*shard)[shardID]; ok {
				if shardState.Height > GetBeaconBestState().GetBestHeightOfShard(shardID) {
					pState.Shard[shardID] = &shardState
				}
			}
		}
	}
	if userRole == common.ShardRole && (nodeMode == common.NodeModeAuto || nodeMode == common.NodeModeBeacon) {
		if shardState, ok := (*shard)[userShardID]; ok && shardState.Height >= blockchain.BestState.Shard[userShardID].ShardHeight {
			pState.Shard[userShardID] = &shardState
			if pool, ok := (*crossShardPool)[userShardID]; ok {
				pState.CrossShardPool = make(map[byte]*map[byte][]uint64)
				pState.CrossShardPool[userShardID] = &pool
			}
		}
	}
	blockchain.Synker.Status.Lock()
	for shardID := 0; shardID < blockchain.BestState.Beacon.ActiveShards; shardID++ {
		if shardState, ok := (*shard)[byte(shardID)]; ok {
			if shardState.Height > GetBestStateShard(byte(shardID)).ShardHeight && (*shard)[byte(shardID)].Timestamp > GetBestStateShard(byte(shardID)).BestBlock.Header.Timestamp {
				pState.Shard[byte(shardID)] = &shardState
			}
		}
	}
	blockchain.Synker.Status.Unlock()

	blockchain.Synker.States.Lock()
	if blockchain.Synker.States.PeersState != nil {
		blockchain.Synker.States.PeersState[pState.Peer] = pState
	}
	blockchain.Synker.States.Unlock()
}

func (blockchain *BlockChain) OnBlockShardReceived(newBlk *ShardBlock) {
	if blockchain.IsTest {
		return
	}
	fmt.Println("Shard block received from shard", newBlk.Header.ShardID, newBlk.Header.Height)
	if newBlk.Header.Timestamp < GetBestStateShard(newBlk.Header.ShardID).BestBlock.Header.Timestamp { // not receive block older than current latest block
		//fmt.Println("Shard block received 0")
		//return
	}

	if _, ok := blockchain.Synker.Status.Shards[newBlk.Header.ShardID]; ok {
		if _, ok := currentInsert.Shards[newBlk.Header.ShardID]; !ok {
			currentInsert.Shards[newBlk.Header.ShardID] = &sync.Mutex{}
		}

		currentInsert.Shards[newBlk.Header.ShardID].Lock()
		defer currentInsert.Shards[newBlk.Header.ShardID].Unlock()
		currentShardBestState := blockchain.BestState.Shard[newBlk.Header.ShardID]

		if currentShardBestState.ShardHeight <= newBlk.Header.Height {
			currentShardBestState := blockchain.BestState.Shard[newBlk.Header.ShardID]

			if currentShardBestState.ShardHeight == newBlk.Header.Height && currentShardBestState.BestBlock.Header.Timestamp < newBlk.Header.Timestamp && currentShardBestState.BestBlock.Header.Round < newBlk.Header.Round {
				//fmt.Println("Shard block received 1", role)
				err := blockchain.InsertShardBlock(newBlk, false)
				if err != nil {
					Logger.log.Error(err)
				}
				return
			}

			err := blockchain.config.ShardPool[newBlk.Header.ShardID].AddShardBlock(newBlk)
			if err != nil {
				Logger.log.Errorf("Add block %+v from shard %+v error %+v: \n", newBlk.Header.Height, newBlk.Header.ShardID, err)
			}
		} else {
			//fmt.Println("Shard block received 2")
		}
	} else {
		//fmt.Println("Shard block received 1")
	}
}

func (blockchain *BlockChain) OnBlockBeaconReceived(newBlk *BeaconBlock) {
	if blockchain.IsTest {
		return
	}
	if blockchain.Synker.Status.Beacon {
		fmt.Println("Beacon block received", newBlk.Header.Height, blockchain.BestState.Beacon.BeaconHeight, newBlk.Header.Timestamp)
		if newBlk.Header.Timestamp < blockchain.BestState.Beacon.BestBlock.Header.Timestamp { // not receive block older than current latest block
			return
		}
		if blockchain.BestState.Beacon.BeaconHeight <= newBlk.Header.Height {
			currentBeaconBestState := blockchain.BestState.Beacon
			if currentBeaconBestState.BeaconHeight == newBlk.Header.Height && currentBeaconBestState.BestBlock.Header.Timestamp < newBlk.Header.Timestamp && currentBeaconBestState.BestBlock.Header.Round < newBlk.Header.Round {
				fmt.Println("Beacon block insert", newBlk.Header.Height)
				err := blockchain.InsertBeaconBlock(newBlk, false)
				if err != nil {
					Logger.log.Error(err)
					return
				}
				return
			}
			fmt.Println("Beacon block prepare add to pool", newBlk.Header.Height)
			err := blockchain.config.BeaconPool.AddBeaconBlock(newBlk)
			if err != nil {
				fmt.Println("Beacon block add pool err", err)
			}
		}

	}
}

func (blockchain *BlockChain) OnShardToBeaconBlockReceived(block *ShardToBeaconBlock) {
	if blockchain.IsTest {
		return
	}
	if blockchain.config.NodeMode == common.NodeModeBeacon || blockchain.config.NodeMode == common.NodeModeAuto {
		layer, role, _ := blockchain.config.ConsensusEngine.GetUserRole()
		if layer != common.BeaconRole || role != common.CommitteeRole {
			return
		}
	} else {
		return
	}

	if blockchain.Synker.IsLatest(false, 0) {
		if block.Header.Version != SHARD_BLOCK_VERSION {
			Logger.log.Debugf("Invalid Verion of block height %+v in Shard %+v", block.Header.Height, block.Header.ShardID)
			return
		}

		//err := blockchain.config.ConsensusEngine.ValidateProducerSig(block, block.Header.ConsensusType)
		//if err != nil {
		//	Logger.log.Error(err)
		//	return
		//}

		from, to, err := blockchain.config.ShardToBeaconPool.AddShardToBeaconBlock(block)
		if err != nil {
			if err.Error() != "receive old block" && err.Error() != "receive duplicate block" {
				Logger.log.Error(err)
				return
			}
		}
		if from != 0 && to != 0 {
			fmt.Printf("Message/SyncBlockShardToBeacon, from %+v to %+v \n", from, to)
			blockchain.Synker.SyncBlockShardToBeacon(block.Header.ShardID, false, false, false, nil, nil, from, to, "")
		}
	}
}

func (blockchain *BlockChain) OnCrossShardBlockReceived(block *CrossShardBlock) {
	if blockchain.IsTest {
		return
	}
	Logger.log.Info("Received CrossShardBlock", block.Header.Height, block.Header.ShardID)
	if blockchain.IsTest {
		return
	}
	if blockchain.config.NodeMode == common.NodeModeShard || blockchain.config.NodeMode == common.NodeModeAuto {
		layer, role, _ := blockchain.config.ConsensusEngine.GetUserRole()
		if layer != common.ShardRole || role != common.CommitteeRole {
			return
		}
	} else {
		return
	}
	expectedHeight, toShardID, err := blockchain.config.CrossShardPool[block.ToShardID].AddCrossShardBlock(block)
	for fromShardID, height := range expectedHeight {
		// fmt.Printf("Shard %+v request CrossShardBlock with Height %+v from shard %+v \n", toShardID, height, fromShardID)
		blockchain.Synker.SyncBlockCrossShard(false, false, []common.Hash{}, []uint64{height}, fromShardID, toShardID, "")
	}
	if err != nil {
		if err.Error() != "receive old block" && err.Error() != "receive duplicate block" {
			Logger.log.Error(err)
			return
		}
	}

}
