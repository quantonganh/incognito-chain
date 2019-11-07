package metadata

import (
	"encoding/json"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/database/lvdb"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy/zeroknowledge"
)

// Interface for all types of metadata in tx
type Metadata interface {
	GetType() int
	Hash() *common.Hash
	CheckTransactionFee(Transaction, uint64, int64, database.DatabaseInterface) bool
	ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, b byte, db database.DatabaseInterface) (bool, error)
	ValidateSanityData(bcr BlockchainRetriever, tx Transaction) (bool, bool, error)
	ValidateMetadataByItself() bool
	BuildReqActions(tx Transaction, bcr BlockchainRetriever, shardID byte) ([][]string, error)
	CalculateSize() uint64
	VerifyMinerCreatedTxBeforeGettingInBlock([]Transaction, []int, [][]string, []int, byte, Transaction, BlockchainRetriever, *AccumulatedValues) (bool, error)
	IsMinerCreatedMetaType() bool
}

// This is tx struct which is really saved in tx mempool
type TxDesc struct {
	// Tx is the transaction associated with the entry.
	Tx Transaction

	// Height is the best block's height when the entry was added to the the source pool.
	Height uint64

	// Fee is the total fee the transaction associated with the entry pays.
	Fee uint64

	// FeeToken is the total token fee the transaction associated with the entry pays.
	// FeeToken is zero if tx is PRV transaction
	FeeToken uint64

	// FeePerKB is the fee the transaction pays in coin per 1000 bytes.
	FeePerKB int32
}

// Interface for mempool which is used in metadata
type MempoolRetriever interface {
	GetSerialNumbersHashH() map[common.Hash][]common.Hash
	GetTxsInMem() map[common.Hash]TxDesc
}

// Interface for blockchain which is used in metadata
type BlockchainRetriever interface {
	GetStakingAmountShard() uint64
	GetTxChainHeight(tx Transaction) (uint64, error)
	GetChainHeight(byte) uint64
	GetBeaconHeight() uint64
	GetCustomTokenTxs(*common.Hash) (map[common.Hash]Transaction, error)
	GetTransactionByHash(common.Hash) (byte, common.Hash, int, Transaction, error)
	GetCurrentBeaconBlockHeight(byte) uint64
	GetAllCommitteeValidatorCandidate() (map[byte][]incognitokey.CommitteePublicKey, map[byte][]incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, []incognitokey.CommitteePublicKey, error)
	GetAllCommitteeValidatorCandidateFlattenListFromDatabase() ([]string, error)
	GetStakingTx(byte) map[string]string
	GetAutoStakingList() map[string]bool
	GetDatabase() database.DatabaseInterface
	GetTxValue(txid string) (uint64, error)
	GetShardIDFromTx(txid string) (byte, error)
	GetCentralizedWebsitePaymentAddress() string
}

// Interface for all type of transaction
type Transaction interface {
	// GET/SET FUNC
	GetMetadataType() int
	GetType() string
	GetLockTime() int64
	GetTxActualSize() uint64
	GetSenderAddrLastByte() byte
	GetTxFee() uint64
	GetTxFeeToken() uint64
	GetMetadata() Metadata
	SetMetadata(Metadata)
	GetInfo() []byte
	GetSender() []byte
	GetSigPubKey() []byte
	GetProof() *zkp.PaymentProof
	// Get receivers' data for tx
	GetReceivers() ([][]byte, []uint64)
	GetUniqueReceiver() (bool, []byte, uint64)
	GetTransferData() (bool, []byte, uint64, *common.Hash)

	// Get receivers' data for custom token tx (nil for normal tx)
	GetTokenReceivers() ([][]byte, []uint64)
	GetTokenUniqueReceiver() (bool, []byte, uint64)

	GetMetadataFromVinsTx(BlockchainRetriever) (Metadata, error)
	GetTokenID() *common.Hash

	ListSerialNumbersHashH() []common.Hash
	Hash() *common.Hash

	// VALIDATE FUNC
	CheckTxVersion(int8) bool
	// CheckTransactionFee(minFeePerKbTx uint64) bool
	ValidateTxWithCurrentMempool(MempoolRetriever) error
	ValidateTxWithBlockChain(BlockchainRetriever, byte, database.DatabaseInterface) error
	ValidateDoubleSpendWithBlockchain(BlockchainRetriever, byte, database.DatabaseInterface, *common.Hash) error
	ValidateSanityData(BlockchainRetriever) (bool, error)
	ValidateTxByItself(bool, database.DatabaseInterface, BlockchainRetriever, byte) (bool, error)
	ValidateType() bool
	ValidateTransaction(bool, database.DatabaseInterface, byte, *common.Hash) (bool, error)
	VerifyMinerCreatedTxBeforeGettingInBlock([]Transaction, []int, [][]string, []int, byte, BlockchainRetriever, *AccumulatedValues) (bool, error)

	IsPrivacy() bool
	IsCoinsBurning() bool
	CalculateTxValue() uint64
	IsSalaryTx() bool
}

func getPDEPoolPair(
	prvIDStr, tokenIDStr string,
	beaconHeight int64,
	db database.DatabaseInterface,
) (*lvdb.PDEPoolForPair, error) {
	var pdePoolForPair lvdb.PDEPoolForPair
	var err error
	poolPairBytes := []byte{}
	if beaconHeight == -1 {
		poolPairBytes, err = db.GetLatestPDEPoolForPair(prvIDStr, tokenIDStr)
	} else {
		poolPairBytes, err = db.GetPDEPoolForPair(uint64(beaconHeight), prvIDStr, tokenIDStr)
	}
	if err != nil {
		return nil, err
	}
	if len(poolPairBytes) == 0 {
		return nil, NewMetadataTxError(CouldNotGetExchangeRateError, fmt.Errorf("Could not find out pdePoolForPair with token ids: %s & %s", prvIDStr, tokenIDStr))
	}
	err = json.Unmarshal(poolPairBytes, &pdePoolForPair)
	if err != nil {
		return nil, err
	}
	return &pdePoolForPair, nil
}

func convertValueBetweenCurrencies(
	amount uint64,
	currentCurrencyIDStr string,
	tokenID *common.Hash,
	beaconHeight int64,
	db database.DatabaseInterface,
) (float64, error) {
	prvIDStr := common.PRVCoinID.String()
	tokenIDStr := tokenID.String()
	pdePoolForPair, err := getPDEPoolPair(prvIDStr, tokenIDStr, beaconHeight, db)
	if err != nil {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
	}
	invariant := float64(0)
	invariant = float64(pdePoolForPair.Token1PoolValue) * float64(pdePoolForPair.Token2PoolValue)
	if invariant == 0 {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
	}
	if pdePoolForPair.Token1IDStr == currentCurrencyIDStr {
		remainingValue := invariant / (float64(pdePoolForPair.Token1PoolValue) + float64(amount))
		if float64(pdePoolForPair.Token2PoolValue) <= remainingValue {
			return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
		}
		return float64(pdePoolForPair.Token2PoolValue) - remainingValue, nil
	}
	remainingValue := invariant / (float64(pdePoolForPair.Token2PoolValue) + float64(amount))
	if float64(pdePoolForPair.Token1PoolValue) <= remainingValue {
		return 0, NewMetadataTxError(CouldNotGetExchangeRateError, err)
	}
	return float64(pdePoolForPair.Token1PoolValue) - remainingValue, nil
}

// return error if there is no exchange rate between native token and privacy token
// beaconHeight = -1: get the latest beacon height
func ConvertNativeTokenToPrivacyToken(
	nativeTokenAmount uint64,
	tokenID *common.Hash,
	beaconHeight int64,
	db database.DatabaseInterface,
) (float64, error) {
	return convertValueBetweenCurrencies(
		nativeTokenAmount,
		common.PRVCoinID.String(),
		tokenID,
		beaconHeight,
		db,
	)
}

// return error if there is no exchange rate between native token and privacy token
// beaconHeight = -1: get the latest beacon height
func ConvertPrivacyTokenToNativeToken(
	privacyTokenAmount uint64,
	tokenID *common.Hash,
	beaconHeight int64,
	db database.DatabaseInterface,
) (float64, error) {
	return convertValueBetweenCurrencies(
		privacyTokenAmount,
		tokenID.String(),
		tokenID,
		beaconHeight,
		db,
	)
}