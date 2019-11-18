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
	DurationHalfLifeRewardForDev = uint64(31536000) // 5 years, after 5 year, reward for devs = 0
	GetValidBlock                = 20
	GenesisBlockTime             = "2019-10-14T00:00:10.000Z"
	TestRandom                   = true
)

// CONSTANT for network MAINNET
const (
	// ------------- Mainnet ---------------------------------------------
	Mainnet            = 0x01
	MainetName         = "mainnet"
	MainnetDefaultPort = "9333"
	MainnetEpoch       = 30000
	MainnetRandomTime  = 15000
	MainnetOffset      = 2
	MainnetSwapOffset  = 1

	MainNetShardCommitteeSize  = 3
	MainNetBeaconCommitteeSize = 3
	MainNetActiveShards        = 2
	MainNetStakingAmountShard  = 1750000000000 // 1750 PRV = 1750 * 10^9 nano PRV

	MainnetMinBeaconBlkInterval = 10 * time.Second //second
	MainnetMaxBeaconBlkCreation = 8 * time.Second  //second
	MainnetMinShardBlkInterval  = 10 * time.Second //second
	MainnetMaxShardBlkCreation  = 6 * time.Second  //second

	//board and proposal parameters
	MainnetBasicReward                      = 400000000 //40 mili PRV
	MainnetRewardHalflife                   = 3155760   //1 year, reduce 12.5% per year
	MainETHContractAddressStr               = "0x10e492e6383DfE37d0d0B7B86015AE0876e88663"
	MainnetDevAddress                       = "1Uv2vrb74e6ScxuQiXvW9UcKoEbXnRMbuBJ6W2FBWxqhtHNGHi3sUP1D14rNEnWWzkYSMsZCmA4DKV6igmjd7qaJfj9TuMmyqz2ZG2SNx"
	MainnetCentralizedWebsitePaymentAddress = "1Uv2zzR4LgfX8ToQe8ub3bYcCLk3uDU1sm9U9hiu9EKYXoS77UdikfT9s8d5YjhsTJm61eazsMwk2otFZBYpPHwiMn8z6bKWWJRspsLky"
	// ------------- end Mainnet --------------------------------------
)

// VARIABLE for mainnet
var (
	MainnetInitPRV = []string{}
	// for beacon
	// public key
	PreSelectBeaconNodeMainnetSerializedPubkey = PreSelectBeaconNodeTestnetSerializedPubkey
	// For shard
	// public key
	PreSelectShardNodeMainnetSerializedPubkey = PreSelectShardNodeTestnetSerializedPubkey
	MaxTxsInBlock                             = 600
	MaxTxsProcessTimeInBlockCreation          = float64(0.85)
	TxsAverageProcessTime                     = int64(5000) // count in nano second ~= 5 mili seconds
	DefaultTxsAverageProcessTime              = int64(5000) // count in nano second
)

// END CONSTANT for network MAINNET

// CONSTANT for network TESTNET
const (
	Testnet            = 0x16
	TestnetName        = "testnet"
	TestnetDefaultPort = "9444"
	TestnetEpoch       = 100
	TestnetRandomTime  = 50
	TestnetOffset      = 1
	TestnetSwapOffset  = 1

	TestNetShardCommitteeSize     = 16
	TestNetMinShardCommitteeSize  = 4
	TestNetBeaconCommitteeSize    = 4
	TestNetMinBeaconCommitteeSize = 4
	TestNetActiveShards           = 2
	TestNetStakingAmountShard     = 1750000000000 // 1750 PRV = 1750 * 10^9 nano PRV

	TestNetMinBeaconBlkInterval = 10 * time.Second //second
	TestNetMaxBeaconBlkCreation = 8 * time.Second  //second
	TestNetMinShardBlkInterval  = 10 * time.Second //second
	TestNetMaxShardBlkCreation  = 6 * time.Second  //second

	//board and proposal parameters
	TestnetBasicReward                      = 400000000 //40 mili PRV
	TestnetRewardHalflife                   = 3155760   //1 year, reduce 12.5% per year
	TestnetETHContractAddressStr            = "0x862078F5e2B36924DdB704C677b962521b3433AB"
	TestnetDevAddress                       = "1Uv2vrb74e6ScxuQiXvW9UcKoEbXnRMbuBJ6W2FBWxqhtHNGHi3sUP1D14rNEnWWzkYSMsZCmA4DKV6igmjd7qaJfj9TuMmyqz2ZG2SNx"
	TestnetCentralizedWebsitePaymentAddress = "1Uv2zzR4LgfX8ToQe8ub3bYcCLk3uDU1sm9U9hiu9EKYXoS77UdikfT9s8d5YjhsTJm61eazsMwk2otFZBYpPHwiMn8z6bKWWJRspsLky"
)

// for beacon
// public key
var PreSelectBeaconNodeTestnetSerializedPubkey = []string{}
var PreSelectBeaconNodeTestnetSerializedPaymentAddress = []string{}
var PreSelectShardNodeTestnetSerializedPubkey = []string{}
var PreSelectShardNodeTestnetSerializedPaymentAddress = []string{}

func init() {
	if len(os.Args) > 0 && (strings.Contains(os.Args[0], "test") || strings.Contains(os.Args[0], "Test")) {
		return
	}
	keyData, err := ioutil.ReadFile("keylist.json")
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

// ---------------------------------------------
var TestnetInitPRV = []string{
	`{
  "Version": 1,
  "Type": "s",
  "LockTime": 1563438751,
  "Fee": 0,
  "Info": null,
  "SigPubKey": "AsKCsGYkt3JthzFysVzWHxkESGfEoSRFeWafGB+DZRQA",
  "Sig": "OA3DSbUjZt28zPtTRdbHRvwI8CfZvLeVpsBggHnDMusfpkGmE3MgkmTuhqh9/rOwlEgB1ULgU3yxmdYRSUQpOA==",
  "Proof": "1111111dP9RnNmXbXtb5GKjmThj1fuurPVnBJjr5Nw15gvMRyNfy8QdqGFnPrYmeQe5NpYwgRvx7hRsgDaYGwZmM8rNGBszCM5CGyTcFsHUP95AqhTzZFugrmRU3EFt8TnfM3LktX13eD9ep7V51Ww2UcQ2PewVLz3VwktfUAvmZ3tbPWtQoQLmSFmZ4z7A47gkk7q6WjjRDLtfUbF1yj6CcswkKwMN",
  "PubKeyLastByteSender": 0,
  "Metadata": null
}`,
}
var IntegrationTestInitPRV = []string{`{"Version":1,"Type":"s","LockTime":1564213226,"Fee":0,"Info":null,"SigPubKey":"A6zmFqIlTKgsV23Qk9jz2roo3VhisVy5Flg6EGuOKaQA","Sig":"f+JDTKpO7+veF6DVYobNp6l0l6rAYxCZjYCNRrsFN0lx7aOMOwXhZK0OGrKiDLfqSIMX7CXr9ProBz7TIx3yqg==","Proof":"1111111dP9RnNnGCD9afUsg4bvrBHNWfjZijttFU2bkFYLYFGqCoK6i6RCeSEk2NUmv7p8B4kyhi1qaoMjvYCotjhDogGiuYrEqUT4NQLXatq2xqkfxgX8DURcv9xCgrgqVceQ2DrBR5NcgbMQHHBnW1xV3Dte2kmq837EeufP3KoQpz3m5N3oN6x1UssfWSeHAuw4t2dUinKDTe7SgRnFFhfF59dvy","PubKeyLastByteSender":0,"Metadata":null}`,
	`{"Version":1,"Type":"s","LockTime":1564213226,"Fee":0,"Info":null,"SigPubKey":"As3StzeOJhR5qheXo9stChC6WqQJChZNqmPqdgNOFtkA","Sig":"ccWpvPZjitORv6+9WOWv7K5e8purHA4sX7mfBNE9m9YYFyPJ2awx5+1iHuWKD7BH9oum64XCiLYtW9iihVGlDw==","Proof":"1111111dP9RnNmZen93jhEW3eXaKkne72tbWVGtcdfAEfnbdf7fPDQmwYaTve2a9MBA56HHWXzXCbDxx79KCrtrArUqQKnxgun69qQpCjDZhaBdpKNZAAvYf7uBHrnxpm7qxRA4XLGSKbuLS6mBtrCUFPnit9BDbSAu9ZxQsPnr7XPPyHdbBofrBzFLqf2zTPMrqCAZqBqapA5AMtd8J8yknUHX6hWJ","PubKeyLastByteSender":0,"Metadata":null}`,
	`{"Version":1,"Type":"s","LockTime":1564502136,"Fee":0,"Info":null,"SigPubKey":"AmusT4yw6LoipXRBH10JL7D1I9B2jwN5gVsQA6SexgoB","Sig":"1aZeIjgrFhe9P16J9vd0V4pCOemknsJ/Ljy/a0fhqimyZL+YUpo+Q+rD0T2Tan9e8StbXQi944VD4EItqYhuWw==","Proof":"1111111dP9RnNmFpBcsd8WSQtTxPB9QfuMN8YS39CkSCi7zvR9pRxSNgVgXADCBjkCdMDH9K9VC3SQ1DstvsTSGuJ1XkjHghTWtMbGEeedBai4f4DjByeLzStJagXtuwQAxoia7Gowg7rutuJVLThVEHFDNVjdgmy8h7NCYZrL4YQiy3QLqeLqKwzoBULxEW2rw62HM2xsFjCsk7twTJCpHW1kc9ThT","PubKeyLastByteSender":1,"Metadata":null}`}
