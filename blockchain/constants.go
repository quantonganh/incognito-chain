package blockchain

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

//Network fixed params
const (
	// SHARD_BLOCK_VERSION is the current latest supported block version.
	VERSION                      = 1
	RANDOM_NUMBER                = 3
	SHARD_BLOCK_VERSION          = 1
	BEACON_BLOCK_VERSION         = 1
	DefaultMaxBlkReqPerPeer      = 600
	DefaultMaxBlkReqPerTime      = 1200
	MinCommitteeSize             = 3                // min size to run bft
	DefaultBroadcastStateTime    = 6 * time.Second  // in second
	DefaultStateUpdateTime       = 8 * time.Second  // in second
	DefaultMaxBlockSyncTime      = 1 * time.Second  // in second
	DefaultCacheCleanupTime      = 30 * time.Second // in second
	WorkerNumber                 = 5
	MAX_S2B_BLOCK                = 30
	MAX_BEACON_BLOCK             = 5
	LowerBoundPercentForIncDAO   = 3
	UpperBoundPercentForIncDAO   = 10
	GetValidBlock                = 20
	TestRandom                   = false
	NumberOfFixedBlockValidators = 22
)

// CONSTANT for network MAINNET
const (
	// ------------- Mainnet ---------------------------------------------
	Mainnet                 = 0x01
	MainetName              = "mainnet"
	MainnetDefaultPort      = "9333"
	MainnetGenesisBlockTime = "2019-10-29T00:00:00.000Z"
	MainnetEpoch            = 350
	MainnetRandomTime       = 175
	MainnetOffset           = 4
	MainnetSwapOffset       = 4
	MainnetAssignOffset     = 8

	MainNetShardCommitteeSize     = 32
	MainNetMinShardCommitteeSize  = 22
	MainNetBeaconCommitteeSize    = 32
	MainNetMinBeaconCommitteeSize = 7
	MainNetActiveShards           = 8
	MainNetStakingAmountShard     = 1750000000000 // 1750 PRV = 1750 * 10^9 nano PRV

	MainnetMinBeaconBlkInterval = 40 * time.Second //second
	MainnetMaxBeaconBlkCreation = 10 * time.Second //second
	MainnetMinShardBlkInterval  = 40 * time.Second //second
	MainnetMaxShardBlkCreation  = 10 * time.Second //second

	//board and proposal parameters
	MainnetBasicReward                      = 1386666000                                                                                                //1.386666 PRV
	MainETHContractAddressStr               = "0x0261DB5AfF8E5eC99fBc8FBBA5D4B9f8EcD44ec7"                                                              // v2-main - mainnet, branch master-temp-B-deploy, support erc20 with decimals > 18
	MainnetIncognitoDAOAddress              = "12S32fSyF4h8VxFHt4HfHvU1m9KHvBQsab5zp4TpQctmMdWuveXFH9KYWNemo7DRKvaBEvMgqm4XAuq1a1R4cNk2kfUfvXR3DdxCho3" // community fund
	MainnetCentralizedWebsitePaymentAddress = "12Rvjw6J3FWY3YZ1eDZ5uTy6DTPjFeLhCK7SXgppjivg9ShX2RRq3s8pdoapnH8AMoqvUSqZm1Gqzw7rrKsNzRJwSK2kWbWf1ogy885"
	// ------------- end Mainnet --------------------------------------
)

// VARIABLE for mainnet
var PreSelectBeaconNodeMainnetSerializedPubkey = []string{}
var PreSelectBeaconNodeMainnetSerializedPaymentAddress = []string{}
var PreSelectShardNodeMainnetSerializedPubkey = []string{}
var PreSelectShardNodeMainnetSerializedPaymentAddress = []string{}

// END CONSTANT for network MAINNET

// CONSTANT for network TESTNET
const (
	Testnet                 = 0x16
	TestnetName             = "testnet"
	TestnetDefaultPort      = "9444"
	TestnetGenesisBlockTime = "2019-11-29T00:00:00.000Z"
	TestnetEpoch            = 100
	TestnetRandomTime       = 50
	TestnetOffset           = 1
	TestnetSwapOffset       = 1
	TestnetAssignOffset     = 2

	TestNetShardCommitteeSize     = 32
	TestNetMinShardCommitteeSize  = 4
	TestNetBeaconCommitteeSize    = 4
	TestNetMinBeaconCommitteeSize = 4
	TestNetActiveShards           = 8
	TestNetStakingAmountShard     = 1750000000000 // 1750 PRV = 1750 * 10^9 nano PRV

	TestNetMinBeaconBlkInterval = 10 * time.Second //second
	TestNetMaxBeaconBlkCreation = 8 * time.Second  //second
	TestNetMinShardBlkInterval  = 10 * time.Second //second
	TestNetMaxShardBlkCreation  = 6 * time.Second  //second

	//board and proposal parameters
	TestnetBasicReward                      = 400000000 //40 mili PRV
	TestnetETHContractAddressStr            = "0x6e8CDB333ba1573Fffe195A545F3031Cff9Da008"
	TestnetIncognitoDAOAddress              = "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci" // community fund
	TestnetCentralizedWebsitePaymentAddress = "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci"
)

// VARIABLE for testnet
var PreSelectBeaconNodeTestnetSerializedPubkey = []string{}
var PreSelectBeaconNodeTestnetSerializedPaymentAddress = []string{}
var PreSelectShardNodeTestnetSerializedPubkey = []string{}
var PreSelectShardNodeTestnetSerializedPaymentAddress = []string{}

func init() {
	if len(os.Args) > 0 && (strings.Contains(os.Args[0], "test") || strings.Contains(os.Args[0], "Test")) {
		return
	}
	var keyData []byte
	var err error

	keyData, err = ioutil.ReadFile("keylist.json")
	if err != nil {
		panic(err)
	}

	type AccountKey struct {
		PrivateKey     string
		PaymentAddress string
		// PubKey     string
		CommitteePublicKey string
	}

	type KeyList struct {
		Shard  map[int][]AccountKey
		Beacon []AccountKey
	}

	keylist := KeyList{}

	err = json.Unmarshal(keyData, &keylist)
	if err != nil {
		panic(err)
	}

	var IsTestNet = false
	if IsTestNet {
		for i := 0; i < TestNetMinBeaconCommitteeSize; i++ {
			PreSelectBeaconNodeTestnetSerializedPubkey = append(PreSelectBeaconNodeTestnetSerializedPubkey, keylist.Beacon[i].CommitteePublicKey)
			PreSelectBeaconNodeTestnetSerializedPaymentAddress = append(PreSelectBeaconNodeTestnetSerializedPaymentAddress, keylist.Beacon[i].PaymentAddress)
		}

		for i := 0; i < TestNetActiveShards; i++ {
			for j := 0; j < TestNetMinShardCommitteeSize; j++ {
				PreSelectShardNodeTestnetSerializedPubkey = append(PreSelectShardNodeTestnetSerializedPubkey, keylist.Shard[i][j].CommitteePublicKey)
				PreSelectShardNodeTestnetSerializedPaymentAddress = append(PreSelectShardNodeTestnetSerializedPaymentAddress, keylist.Shard[i][j].PaymentAddress)
			}
		}
	} else {
		for i := 0; i < MainNetMinBeaconCommitteeSize; i++ {
			PreSelectBeaconNodeMainnetSerializedPubkey = append(PreSelectBeaconNodeMainnetSerializedPubkey, keylist.Beacon[i].CommitteePublicKey)
			PreSelectBeaconNodeMainnetSerializedPaymentAddress = append(PreSelectBeaconNodeMainnetSerializedPaymentAddress, keylist.Beacon[i].PaymentAddress)
		}

		for i := 0; i < MainNetActiveShards; i++ {
			for j := 0; j < MainNetMinShardCommitteeSize; j++ {
				PreSelectShardNodeMainnetSerializedPubkey = append(PreSelectShardNodeMainnetSerializedPubkey, keylist.Shard[i][j].CommitteePublicKey)
				PreSelectShardNodeMainnetSerializedPaymentAddress = append(PreSelectShardNodeMainnetSerializedPaymentAddress, keylist.Shard[i][j].PaymentAddress)
			}
		}
	}
}

// For shard
// public key

// END CONSTANT for network TESTNET

// -------------- FOR INSTRUCTION --------------
// Action for instruction
const (
	SetAction     = "set"
	SwapAction    = "swap"
	RandomAction  = "random"
	StakeAction   = "stake"
	AssignAction  = "assign"
	StopAutoStake = "stopautostake"
)
