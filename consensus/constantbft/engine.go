package constantbft

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/constant-money/constant-chain/blockchain"
	"github.com/constant-money/constant-chain/cashec"
	"github.com/constant-money/constant-chain/common"
	"github.com/constant-money/constant-chain/wire"
)

type Engine struct {
	sync.Mutex
	started bool

	// channel
	cQuit   chan struct{}
	cBFTMsg chan wire.Message

	config EngineConfig
}

type EngineConfig struct {
	BlockChain        *blockchain.BlockChain
	ChainParams       *blockchain.Params
	BlockGen          *blockchain.BlkTmplGenerator
	UserKeySet        *cashec.KeySet
	NodeMode          string
	Server            serverInterface
	ShardToBeaconPool blockchain.ShardToBeaconPool
	CrossShardPool    map[byte]blockchain.CrossShardPool
}

//Init apply configuration to consensus engine
func (engine Engine) Init(cfg *EngineConfig) (*Engine, error) {
	return &Engine{
		config: *cfg,
	}, nil
}

func (engine *Engine) Start() error {
	engine.Lock()
	defer engine.Unlock()
	if engine.started {
		return errors.New("Consensus engine is already started")
	}
	engine.cQuit = make(chan struct{})
	engine.cBFTMsg = make(chan wire.Message)
	engine.started = true
	Logger.log.Info("Start consensus with key", engine.config.UserKeySet.GetPublicKeyB58())
	fmt.Println(engine.config.BlockChain.BestState.Beacon.BeaconCommittee)

	time.AfterFunc(DelayTime*time.Millisecond, func() {
		currentPBFTBlkHeight := uint64(0)
		currentPBFTRound := 1
		prevRoundUserRole := ""
		resetRound := func() {
			prevRoundUserRole = ""
			currentPBFTRound = 1
		}
		for {
			select {
			case <-engine.cQuit:
				return
			default:
				if engine.config.BlockChain.IsReady(false, 0) {
					if prevRoundUserRole == common.BEACON_ROLE {
						if currentPBFTBlkHeight <= engine.config.BlockChain.BestState.Beacon.BeaconHeight {
							// reset round
							currentPBFTBlkHeight = engine.config.BlockChain.BestState.Beacon.BeaconHeight + 1
							currentPBFTRound = 1
						}
					}
					userRole, shardID := engine.config.BlockChain.BestState.Beacon.GetPubkeyRole(engine.config.UserKeySet.GetPublicKeyB58(), currentPBFTRound)
					if engine.config.NodeMode == common.NODEMODE_BEACON && userRole == common.SHARD_ROLE {
						userRole = common.EmptyString
					}
					if engine.config.NodeMode == common.NODEMODE_SHARD && userRole != common.SHARD_ROLE {
						userRole = common.EmptyString
					}
					nodeRole := userRole
					switch userRole {
					case common.VALIDATOR_ROLE, common.PROPOSER_ROLE:
						nodeRole = common.BEACON_ROLE
					}
					engine.config.Server.UpdateConsensusState(nodeRole, engine.config.UserKeySet.GetPublicKeyB58(), nil, engine.config.BlockChain.BestState.Beacon.BeaconCommittee, engine.config.BlockChain.BestState.Beacon.ShardCommittee)

					if userRole != common.EmptyString {
						bftProtocol := &BFTProtocol{
							cQuit:             engine.cQuit,
							cBFTMsg:           engine.cBFTMsg,
							BlockGen:          engine.config.BlockGen,
							UserKeySet:        engine.config.UserKeySet,
							BlockChain:        engine.config.BlockChain,
							Server:            engine.config.Server,
							ShardToBeaconPool: engine.config.ShardToBeaconPool,
							CrossShardPool:    engine.config.CrossShardPool,
						}

						if (engine.config.NodeMode == common.NODEMODE_BEACON || engine.config.NodeMode == common.NODEMODE_AUTO) && userRole != common.SHARD_ROLE {
							fmt.Printf("Node mode %+v, user role %+v, shardID %+v \n currentPBFTRound %+v, beacon height %+v, currentPBFTBlkHeight %+v, prevRoundUserRole %+v \n ", engine.config.NodeMode, userRole, shardID, currentPBFTRound, engine.config.BlockChain.BestState.Beacon.BeaconHeight, currentPBFTBlkHeight, prevRoundUserRole)
							bftProtocol.RoundData.ProposerOffset = (currentPBFTRound - 1) % len(engine.config.BlockChain.BestState.Beacon.BeaconCommittee)
							bftProtocol.RoundData.BestStateHash = engine.config.BlockChain.BestState.Beacon.Hash()
							bftProtocol.RoundData.Layer = common.BEACON_ROLE
							bftProtocol.RoundData.Committee = make([]string, len(engine.config.BlockChain.BestState.Beacon.BeaconCommittee))
							copy(bftProtocol.RoundData.Committee, engine.config.BlockChain.BestState.Beacon.BeaconCommittee)
							roundRole, _ := engine.config.BlockChain.BestState.Beacon.GetPubkeyRole(engine.config.UserKeySet.GetPublicKeyB58(), bftProtocol.RoundData.ProposerOffset)
							var (
								err    error
								resBlk interface{}
							)
							switch roundRole {
							case common.PROPOSER_ROLE:
								bftProtocol.RoundData.IsProposer = true
								currentPBFTBlkHeight = engine.config.BlockChain.BestState.Beacon.BeaconHeight + 1
								resBlk, err = bftProtocol.Start()
								if err != nil {
									currentPBFTRound++
									prevRoundUserRole = nodeRole
								}
							case common.VALIDATOR_ROLE:
								bftProtocol.RoundData.IsProposer = false
								currentPBFTBlkHeight = engine.config.BlockChain.BestState.Beacon.BeaconHeight + 1
								resBlk, err = bftProtocol.Start()
								if err != nil {
									currentPBFTRound++
									prevRoundUserRole = nodeRole
								}
							default:
								err = errors.New("Not your turn yet")
							}

							if err == nil {
								fmt.Println(resBlk.(*blockchain.BeaconBlock))
								err = engine.config.BlockChain.InsertBeaconBlock(resBlk.(*blockchain.BeaconBlock), false)
								if err != nil {
									Logger.log.Error("Insert beacon block error", err)
									continue
								}
								//PUSH BEACON TO ALL
								newBeaconBlock := resBlk.(*blockchain.BeaconBlock)
								newBeaconBlockMsg, err := MakeMsgBeaconBlock(newBeaconBlock)
								if err != nil {
									Logger.log.Error("Make new beacon block message error", err)
								} else {
									engine.config.Server.PushMessageToAll(newBeaconBlockMsg)
								}
								//reset round
								prevRoundUserRole = ""
								currentPBFTRound = 1
							} else {
								Logger.log.Error(err)
							}
							continue
						}
						if (engine.config.NodeMode == common.NODEMODE_SHARD || engine.config.NodeMode == common.NODEMODE_AUTO) && userRole == common.SHARD_ROLE {
							if currentPBFTBlkHeight <= engine.config.BlockChain.BestState.Shard[shardID].ShardHeight {
								// reset
								currentPBFTBlkHeight = engine.config.BlockChain.BestState.Shard[shardID].ShardHeight + 1
								currentPBFTRound = 1
							}
							fmt.Printf("Node mode %+v, user role %+v, shardID %+v \n currentPBFTRound %+v, beacon height %+v, currentPBFTBlkHeight %+v, prevRoundUserRole %+v \n ", engine.config.NodeMode, userRole, shardID, currentPBFTRound, engine.config.BlockChain.BestState.Shard[shardID].ShardCommittee, currentPBFTBlkHeight, prevRoundUserRole)
							engine.config.BlockChain.SyncShard(shardID)
							engine.config.BlockChain.StopSyncUnnecessaryShard()
							bftProtocol.RoundData.ProposerOffset = (currentPBFTRound - 1) % len(engine.config.BlockChain.BestState.Shard[shardID].ShardCommittee)
							bftProtocol.RoundData.BestStateHash = engine.config.BlockChain.BestState.Shard[shardID].Hash()
							bftProtocol.RoundData.Layer = common.SHARD_ROLE
							bftProtocol.RoundData.ShardID = shardID
							bftProtocol.RoundData.Committee = make([]string, len(engine.config.BlockChain.BestState.Shard[shardID].ShardCommittee))
							copy(bftProtocol.RoundData.Committee, engine.config.BlockChain.BestState.Shard[shardID].ShardCommittee)
							var (
								err    error
								resBlk interface{}
							)
							if engine.config.BlockChain.IsReady(true, shardID) {
								roundRole := engine.config.BlockChain.BestState.Shard[shardID].GetPubkeyRole(engine.config.UserKeySet.GetPublicKeyB58(), bftProtocol.RoundData.ProposerOffset)
								fmt.Println("My shard role", roundRole)
								switch roundRole {
								case common.PROPOSER_ROLE:
									bftProtocol.RoundData.IsProposer = true
									currentPBFTBlkHeight = engine.config.BlockChain.BestState.Shard[shardID].ShardHeight + 1
									resBlk, err = bftProtocol.Start()
									if err != nil {
										Logger.log.Error(err)
										currentPBFTRound++
										prevRoundUserRole = nodeRole
									}
								case common.VALIDATOR_ROLE:
									bftProtocol.RoundData.IsProposer = false
									currentPBFTBlkHeight = engine.config.BlockChain.BestState.Shard[shardID].ShardHeight + 1
									resBlk, err = bftProtocol.Start()
									if err != nil {
										currentPBFTRound++
										prevRoundUserRole = nodeRole
									}
								default:
									err = errors.New("Not your turn yet")
								}
								if err == nil {
									shardBlk := resBlk.(*blockchain.ShardBlock)
									fmt.Println("========NEW SHARD BLOCK=======", shardBlk.Header.Height)
									isProducer := false
									if strings.Compare(engine.config.UserKeySet.GetPublicKeyB58(), shardBlk.Header.Producer) == 0 {
										isProducer = true
									}
									err = engine.config.BlockChain.InsertShardBlock(shardBlk, isProducer)
									if err != nil {
										Logger.log.Error("Insert shard block error", err)
										continue
									}
									go func() {
										//PUSH SHARD TO BEACON
										fmt.Println("Create And Push Shard To Beacon Block")
										newShardToBeaconBlock := shardBlk.CreateShardToBeaconBlock(engine.config.BlockChain)
										newShardToBeaconMsg, err := MakeMsgShardToBeaconBlock(newShardToBeaconBlock)
										//TODO: check lock later
										if err == nil {
											go engine.config.Server.PushMessageToBeacon(newShardToBeaconMsg)
										}
										fmt.Println("Create and Push all Cross Shard Block")
										//PUSH CROSS-SHARD
										newCrossShardBlocks := shardBlk.CreateAllCrossShardBlock(engine.config.BlockChain.BestState.Beacon.ActiveShards)
										fmt.Println("New Cross Shard Blocks ", newCrossShardBlocks, shardBlk.Header.Height, shardBlk.Header.CrossShards)

										for sID, newCrossShardBlock := range newCrossShardBlocks {
											newCrossShardMsg, err := MakeMsgCrossShardBlock(newCrossShardBlock)
											if err == nil {
												engine.config.Server.PushMessageToShard(newCrossShardMsg, sID)
											}
										}
									}()

									//reset round
									resetRound()
								} else {
									Logger.log.Error(err)
								}
							} else {
								//reset round
								time.Sleep(time.Millisecond * 500)
								resetRound()
							}
						}
					}
				} else {
					//reset round
					time.Sleep(time.Millisecond * 500)
					resetRound()
				}
			}
		}
	})

	return nil
}

func (engine *Engine) Stop() error {
	engine.Lock()
	defer engine.Unlock()
	if !engine.started {
		return errors.New("Consensus engine is already stopped")
	}

	engine.started = false
	close(engine.cQuit)
	return nil
}
