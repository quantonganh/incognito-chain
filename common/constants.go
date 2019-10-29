package common

// for common
const (
	EmptyString       = ""
	ZeroByte          = byte(0x00)
	DateOutputFormat  = "2006-01-02T15:04:05.999999"
	BigIntSize        = 32 // bytes
	CheckSumLen       = 4  // bytes
	AESKeySize        = 32 // bytes
	Int32Size         = 4  // bytes
	Uint32Size        = 4  // bytes
	Uint64Size        = 8  // bytes
	HashSize          = 32 // bytes
	MaxHashStringSize = HashSize * 2
	Base58Version     = 0
)

// size data for incognito key and signature
const (
	// for key size
	PrivateKeySize      = 32  // bytes
	PublicKeySize       = 32  // bytes
	BLSPublicKeySize    = 128 // bytes
	BriPublicKeySize    = 33  // bytes
	TransmissionKeySize = 32  //bytes
	ReceivingKeySize    = 32  // bytes
	PaymentAddressSize  = 64  // bytes
	// for signature size
	// it is used for both privacy and no privacy
	SigPubKeySize    = 32
	SigNoPrivacySize = 64
	SigPrivacySize   = 96
	IncPubKeyB58Size = 51
)

// for exit code
const (
	ExitCodeUnknow = iota
	ExitByOs
	ExitByLogging
	ExitCodeForceUpdate
)

// For all Transaction information
const (
	TxNormalType             = "n"  // normal tx(send and receive coin)
	TxRewardType             = "s"  // reward tx
	TxReturnStakingType      = "rs" //
	TxCustomTokenType        = "t"  // token  tx with no supporting privacy
	TxCustomTokenPrivacyType = "tp" // token  tx with supporting privacy
)

var (
	MaxTxSize    = uint64(100)  // unit KB = 100KB
	MaxBlockSize = uint64(2000) //unit kilobytes = 2 Megabyte
)

// special token ids (aka. PropertyID in custom token)
var (
	PRVCoinID = Hash{4} // To send PRV in custom token
)

// burning addresses
const (
	BurningAddress = "15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs"
)

// CONSENSUS
const (
	NodeModeRelay  = "relay"
	NodeModeShard  = "shard"
	NodeModeAuto   = "auto"
	NodeModeBeacon = "beacon"

	BeaconRole     = "beacon"
	ShardRole      = "shard"
	CommitteeRole  = "committee"
	ProposerRole   = "proposer"
	ValidatorRole  = "validator"
	PendingRole    = "pending"
	WaitingRole    = "waiting"
	MaxShardNumber = 8

	BlsConsensus    = "bls"
	BridgeConsensus = "dsa"
	IncKeyType      = "inc"
)

const (
	BeaconChainKey = "beacon"
	ShardChainKey  = "shard"
)

// Ethereum Decentralized bridge
const (
	AbiJson       = `[{"inputs":[{"internalType":"address","name":"admin","type":"address"},{"internalType":"address","name":"incognitoProxyAddress","type":"address"},{"internalType":"address","name":"_prevVault","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"claimer","type":"address"}],"name":"Claim","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"token","type":"address"},{"indexed":false,"internalType":"string","name":"incognitoAddress","type":"string"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"Deposit","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"uint256","name":"ndays","type":"uint256"}],"name":"Extend","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"newVault","type":"address"}],"name":"Migrate","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address[]","name":"assets","type":"address[]"}],"name":"MoveAssets","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"pauser","type":"address"}],"name":"Paused","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"pauser","type":"address"}],"name":"Unpaused","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"newIncognitoProxy","type":"address"}],"name":"UpdateIncognitoProxy","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"token","type":"address"},{"indexed":false,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"}],"name":"Withdraw","type":"event"},{"constant":true,"inputs":[],"name":"ETH_TOKEN","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"admin","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"claim","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"string","name":"incognitoAddress","type":"string"}],"name":"deposit","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"incognitoAddress","type":"string"}],"name":"depositERC20","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":true,"inputs":[],"name":"expire","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"uint256","name":"n","type":"uint256"}],"name":"extend","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"incognito","outputs":[{"internalType":"contractIncognito","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"hash","type":"bytes32"}],"name":"isWithdrawed","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"addresspayable","name":"_newVault","type":"address"}],"name":"migrate","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"address[]","name":"assets","type":"address[]"}],"name":"moveAssets","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"newVault","outputs":[{"internalType":"addresspayable","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes","name":"inst","type":"bytes"}],"name":"parseBurnInst","outputs":[{"internalType":"uint8","name":"","type":"uint8"},{"internalType":"uint8","name":"","type":"uint8"},{"internalType":"address","name":"","type":"address"},{"internalType":"addresspayable","name":"","type":"address"},{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"pure","type":"function"},{"constant":false,"inputs":[],"name":"pause","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"paused","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"prevVault","outputs":[{"internalType":"contractWithdrawable","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"_successor","type":"address"}],"name":"retire","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"successor","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"unpause","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"address","name":"newIncognitoProxy","type":"address"}],"name":"updateIncognitoProxy","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"internalType":"bytes","name":"inst","type":"bytes"},{"internalType":"uint256[2]","name":"heights","type":"uint256[2]"},{"internalType":"bytes32[][2]","name":"instPaths","type":"bytes32[][2]"},{"internalType":"bool[][2]","name":"instPathIsLefts","type":"bool[][2]"},{"internalType":"bytes32[2]","name":"instRoots","type":"bytes32[2]"},{"internalType":"bytes32[2]","name":"blkData","type":"bytes32[2]"},{"internalType":"uint256[][2]","name":"sigIdxs","type":"uint256[][2]"},{"internalType":"uint8[][2]","name":"sigVs","type":"uint8[][2]"},{"internalType":"bytes32[][2]","name":"sigRs","type":"bytes32[][2]"},{"internalType":"bytes32[][2]","name":"sigSs","type":"bytes32[][2]"}],"name":"withdraw","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":true,"inputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"name":"withdrawed","outputs":[{"internalType":"bool","name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"}]`
	BridgeShardID = 1
	EthAddrStr    = "0x0000000000000000000000000000000000000000"
)

const (
	BridgeRequestNotFoundStatus   = 0
	BridgeRequestProcessingStatus = 1
	BridgeRequestAcceptedStatus   = 2
	BridgeRequestRejectedStatus   = 3
)
