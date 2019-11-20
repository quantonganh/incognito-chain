package blockchain

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/pkg/errors"
)

type ShardChain struct {
	BlockGen   *BlockGenerator
	Blockchain *BlockChain
	ChainName  string
	lock       sync.RWMutex

	views    map[string]*ShardView
	bestView *ShardView
}

func (chain *ShardChain) GetLastBlockTimeStamp() int64 {
	return chain.bestView.BestBlock.Header.Timestamp
}

func (chain *ShardChain) GetMinBlkInterval() time.Duration {
	return chain.bestView.BlockInterval
}

func (chain *ShardChain) GetMaxBlkCreateTime() time.Duration {
	return chain.bestView.BlockMaxCreateTime
}

func (chain *ShardChain) IsReady() bool {
	return chain.Blockchain.Synker.IsLatest(true, chain.bestView.ShardID)
}

func (chain *ShardChain) CurrentHeight() uint64 {
	return chain.bestView.BestBlock.Header.Height
}

func (chain *ShardChain) GetCommittee() []incognitokey.CommitteePublicKey {
	result := []incognitokey.CommitteePublicKey{}
	return append(result, chain.bestView.ShardCommittee...)
}

func (chain *ShardChain) GetCommitteeSize() int {
	return len(chain.bestView.ShardCommittee)
}

func (chain *ShardChain) GetPubKeyCommitteeIndex(pubkey string) int {
	for index, key := range chain.bestView.ShardCommittee {
		if key.GetMiningKeyBase58(chain.bestView.ConsensusAlgorithm) == pubkey {
			return index
		}
	}
	return -1
}

func (chain *ShardChain) GetLastProposerIndex() int {
	return chain.bestView.ShardProposerIdx
}

func (chain *ShardChain) CreateNewBlock(round int) (common.BlockInterface, error) {
	chain.lock.Lock()
	defer chain.lock.Unlock()
	start := time.Now()
	Logger.log.Infof("Begin Create New Block %+v", start)
	beaconHeight := chain.Blockchain.Synker.States.ClosestState.ClosestBeaconState
	if chain.Blockchain.BestView.Beacon.BeaconHeight < beaconHeight {
		beaconHeight = chain.Blockchain.BestView.Beacon.BeaconHeight
	} else {
		if beaconHeight < chain.Blockchain.BestView.Shard[byte(chain.GetShardID())].BeaconHeight {
			beaconHeight = chain.Blockchain.BestView.Shard[byte(chain.GetShardID())].BeaconHeight
		}
	}
	Logger.log.Infof("Begin Enter New Block Shard %+v", time.Now())
	newBlock, err := chain.BlockGen.NewBlockShard(byte(chain.GetShardID()), round, chain.Blockchain.Synker.GetClosestCrossShardPoolState(), beaconHeight, start)
	Logger.log.Infof("Begin Finish New Block Shard %+v", time.Now())
	if err != nil {
		return nil, err
	}
	Logger.log.Infof("Finish Create New Block %+v", start)
	return newBlock, nil
}

// func (chain *ShardChain) ValidateAndInsertBlock(block common.BlockInterface) error {
// 	//@Bahamoot review later
// 	chain.lock.Lock()
// 	defer chain.lock.Unlock()
// 	var shardbestView ShardbestView
// 	shardBlock := block.(*ShardBlock)
// 	shardbestView.cloneShardbestViewFrom(chain.bestView)
// 	producerPublicKey := shardBlock.Header.Producer
// 	producerPosition := (shardbestView.ShardProposerIdx + shardBlock.Header.Round) % len(shardbestView.ShardCommittee)
// 	tempProducer := shardbestView.ShardCommittee[producerPosition].GetMiningKeyBase58(shardbestView.ConsensusAlgorithm)
// 	if strings.Compare(tempProducer, producerPublicKey) != 0 {
// 		return NewBlockChainError(BeaconBlockProducerError, fmt.Errorf("Expect Producer Public Key to be equal but get %+v From Index, %+v From Header", tempProducer, producerPublicKey))
// 	}
// 	if err := chain.ValidateBlockSignatures(block, shardbestView.ShardCommittee); err != nil {
// 		return err
// 	}
// 	return chain.Blockchain.InsertShardBlock(shardBlock, false)
// }

func (chain *ShardChain) ValidateBlockSignatures(block common.BlockInterface, committee []incognitokey.CommitteePublicKey) error {
	if err := chain.Blockchain.config.ConsensusEngine.ValidateProducerSig(block, chain.GetConsensusType()); err != nil {
		return err
	}
	if err := chain.Blockchain.config.ConsensusEngine.ValidateBlockCommitteSig(block, committee, chain.GetConsensusType()); err != nil {
		return nil
	}
	return nil
}

func (chain *ShardChain) InsertBlk(block common.BlockInterface) error {
	if chain.Blockchain.config.ConsensusEngine.IsOngoing(chain.ChainName) {
		return NewBlockChainError(ConsensusIsOngoingError, errors.New(fmt.Sprint(chain.ChainName, block.Hash())))
	}
	return chain.Blockchain.InsertShardBlock(block.(*ShardBlock), false)
}

func (chain *ShardChain) InsertAndBroadcastBlock(block common.BlockInterface) error {
	go chain.Blockchain.config.Server.PushBlockToAll(block, false)
	err := chain.Blockchain.InsertShardBlock(block.(*ShardBlock), true)
	if err != nil {
		return err
	}
	return nil
}

func (chain *ShardChain) GetActiveShardNumber() int {
	return 0
}

func (chain *ShardChain) GetChainName() string {
	return chain.ChainName
}

func (chain *ShardChain) GetConsensusType() string {
	return chain.bestView.ConsensusAlgorithm
}

func (chain *ShardChain) GetShardID() int {
	return int(chain.bestView.ShardID)
}

func (chain *ShardChain) GetPubkeyRole(pubkey string, round int) (string, byte) {
	return chain.bestView.GetPubkeyRole(pubkey, round), chain.bestView.ShardID
}

func (chain *ShardChain) UnmarshalBlock(blockString []byte) (common.BlockInterface, error) {
	var shardBlk ShardBlock
	err := json.Unmarshal(blockString, &shardBlk)
	if err != nil {
		return nil, err
	}
	return &shardBlk, nil
}

func (chain *ShardChain) ValidatePreSignBlock(block common.BlockInterface) error {
	return chain.Blockchain.VerifyPreSignShardBlock(block.(*ShardBlock), chain.bestView.ShardID)
}

func (chain *ShardChain) GetBestViewConsensusType() string {
	return ""
}
func (chain *ShardChain) GetBestViewLastBlockTimeStamp() int64 {
	return 0
}
func (chain *ShardChain) GetBestViewMinBlkInterval() time.Duration {
	return 0
}
func (chain *ShardChain) GetBestViewMaxBlkCreateTime() time.Duration {
	return 0
}
func (chain *ShardChain) CurrentBestViewHeight() uint64 {
	return 0
}
func (chain *ShardChain) GetBestViewCommitteeSize() int {
	return 0
}
func (chain *ShardChain) GetBestViewCommittee() []incognitokey.CommitteePublicKey {
	return nil
}
func (chain *ShardChain) GetBestViewPubKeyCommitteeIndex(string) int {
	return 0
}
func (chain *ShardChain) GetBestViewLastProposerIndex() int {
	return 0
}
