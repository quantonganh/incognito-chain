package mempool

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/incognitochain/incognito-chain/metrics"
	"github.com/incognitochain/incognito-chain/pubsub"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/databasemp"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/transaction"
)

// default value
const (
	defaultScanTime          = 10 * time.Minute
	defaultIsUnlockMempool   = true
	defaultIsBlockGenStarted = false
	defaultRoleInCommittees  = -1
	defaultIsTest            = false
	defaultReplaceFeeRatio   = 1.1
)

// config is a descriptor containing the memory pool configuration.
type Config struct {
	BlockChain        *blockchain.BlockChain       // Block chain of node
	DataBase          database.DatabaseInterface   // main database of blockchain
	DataBaseMempool   databasemp.DatabaseInterface // database is used for storage data in mempool into lvdb
	ChainParams       *blockchain.Params
	FeeEstimator      map[byte]*FeeEstimator // FeeEstimatator provides a feeEstimator. If it is not nil, the mempool records all new transactions it observes into the feeEstimator.
	TxLifeTime        uint                   // Transaction life time in pool
	MaxTx             uint64                 //Max transaction pool may have
	IsLoadFromMempool bool                   //Reset mempool database when run node
	PersistMempool    bool
	RelayShards       []byte
	// UserKeyset            *incognitokey.KeySet
	PubSubManager         *pubsub.PubSubManager
	RoleInCommitteesEvent pubsub.EventChannel
}

// TxDesc is transaction message in mempool
type TxDesc struct {
	Desc            metadata.TxDesc // transaction details
	StartTime       time.Time       //Unix Time that transaction enter mempool
	IsFowardMessage bool
}

type TxPool struct {
	// The following variables must only be used atomically.
	config                    Config
	lastUpdated               int64 // last time pool was updated
	pool                      map[common.Hash]*TxDesc
	poolSerialNumbersHashList map[common.Hash][]common.Hash // [txHash] -> list hash serialNumbers of input coin
	poolSerialNumberHash      map[common.Hash]common.Hash   // [hash from list of serialNumber] -> txHash
	mtx                       sync.RWMutex
	poolCandidate             map[common.Hash]string //Candidate List in mempool
	candidateMtx              sync.RWMutex
	poolRequestStopStaking    map[common.Hash]string //request stop staking list in mempool
	requestStopStakingMtx     sync.RWMutex
	poolTokenID               map[common.Hash]string //Token ID List in Mempool
	tokenIDMtx                sync.RWMutex
	CPendingTxs               chan<- metadata.Transaction // channel to deliver txs to block gen
	CRemoveTxs                chan<- metadata.Transaction // channel to deliver txs to block gen
	RoleInCommittees          int                         //Current Role of Node
	roleMtx                   sync.RWMutex
	ScanTime                  time.Duration
	IsBlockGenStarted         bool
	IsUnlockMempool           bool
	ReplaceFeeRatio           float64

	//for testing
	IsTest       bool
	duplicateTxs map[common.Hash]uint64 //For testing
}

/*
Init Txpool from config
*/
func (tp *TxPool) Init(cfg *Config) {
	tp.config = *cfg
	tp.pool = make(map[common.Hash]*TxDesc)
	tp.poolSerialNumbersHashList = make(map[common.Hash][]common.Hash)
	tp.poolSerialNumberHash = make(map[common.Hash]common.Hash)
	tp.poolTokenID = make(map[common.Hash]string)
	tp.poolCandidate = make(map[common.Hash]string)
	tp.poolRequestStopStaking = make(map[common.Hash]string)
	tp.duplicateTxs = make(map[common.Hash]uint64)
	_, subChanRole, _ := tp.config.PubSubManager.RegisterNewSubscriber(pubsub.ShardRoleTopic)
	tp.config.RoleInCommitteesEvent = subChanRole
	tp.ScanTime = defaultScanTime
	tp.IsUnlockMempool = defaultIsUnlockMempool
	tp.IsBlockGenStarted = defaultIsBlockGenStarted
	tp.RoleInCommittees = defaultRoleInCommittees
	tp.IsTest = defaultIsTest
	tp.ReplaceFeeRatio = defaultReplaceFeeRatio
}

// InitChannelMempool - init channel
func (tp *TxPool) InitChannelMempool(cPendingTxs chan metadata.Transaction, cRemoveTxs chan metadata.Transaction) {
	tp.CPendingTxs = cPendingTxs
	tp.CRemoveTxs = cRemoveTxs
}

func (tp *TxPool) AnnouncePersisDatabaseMempool() {
	if tp.config.PersistMempool {
		Logger.log.Critical("Turn on Mempool Persistence Database")
	} else {
		Logger.log.Critical("Turn off Mempool Persistence Database")
	}
}

// LoadOrResetDatabaseMempool - Load and reset database of mempool when start node
func (tp *TxPool) LoadOrResetDatabaseMempool() error {
	if !tp.config.IsLoadFromMempool {
		err := tp.resetDatabaseMempool()
		if err != nil {
			Logger.log.Errorf("Fail to reset mempool database, error: %+v \n", err)
			return NewMempoolTxError(DatabaseError, err)
		} else {
			Logger.log.Critical("Successfully Reset from database")
		}
	} else {
		txDescs, err := tp.loadDatabaseMP()
		if err != nil {
			Logger.log.Errorf("Fail to load mempool database, error: %+v \n", err)
			return NewMempoolTxError(DatabaseError, err)
		} else {
			Logger.log.Criticalf("Successfully load %+v from database \n", len(txDescs))
		}
	}
	return nil
}

// loop forever in mempool
// receive data from other package
func (tp *TxPool) Start(cQuit chan struct{}) {
	for {
		select {
		case <-cQuit:
			return
		case msg := <-tp.config.RoleInCommitteesEvent:
			{
				shardID, ok := msg.Value.(int)
				if !ok {
					continue
				}
				go func() {
					tp.roleMtx.Lock()
					defer tp.roleMtx.Unlock()
					tp.RoleInCommittees = shardID
				}()
			}
		}
	}
}

func (tp *TxPool) MonitorPool() {
	if tp.config.TxLifeTime == 0 {
		return
	}
	ticker := time.NewTicker(tp.ScanTime)
	defer ticker.Stop()
	for _ = range ticker.C {
		tp.mtx.Lock()
		ttl := time.Duration(tp.config.TxLifeTime) * time.Second
		txsToBeRemoved := []*TxDesc{}
		for _, txDesc := range tp.pool {
			if time.Since(txDesc.StartTime) > ttl {
				txsToBeRemoved = append(txsToBeRemoved, txDesc)
			}
		}
		for _, txDesc := range txsToBeRemoved {
			txHash := *txDesc.Desc.Tx.Hash()
			startTime := txDesc.StartTime
			tp.removeTx(txDesc.Desc.Tx)
			tp.TriggerCRemoveTxs(txDesc.Desc.Tx)
			tp.removeCandidateByTxHash(txHash)
			tp.removeRequestStopStakingByTxHash(txHash)
			tp.removeTokenIDByTxHash(txHash)
			err := tp.config.DataBaseMempool.RemoveTransaction(txDesc.Desc.Tx.Hash())
			if err != nil {
				Logger.log.Error(err)
			}
			txSize := txDesc.Desc.Tx.GetTxActualSize()
			go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
				metrics.Measurement:      metrics.TxPoolRemoveAfterLifeTime,
				metrics.MeasurementValue: float64(time.Since(startTime).Seconds()),
				metrics.Tag:              metrics.TxSizeTag,
				metrics.TagValue:         fmt.Sprintf("%d", txSize),
			})
			size := len(tp.pool)
			go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
				metrics.Measurement:      metrics.PoolSize,
				metrics.MeasurementValue: float64(size)})
		}
		tp.mtx.Unlock()
	}
}

// MaybeAcceptTransaction is the main workhorse for handling insertion of new
// free-standing transactions into a memory pool.  It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, detecting orphan transactions, and insertion into the memory pool.
//
// If the transaction is an orphan (missing parent transactions), the
// transaction is NOT added to the orphan pool, but each unknown referenced
// parent is returned.  Use ProcessTransaction instead if new orphans should
// be added to the orphan pool.
//
// This function is safe for concurrent access.
// #1: tx
// #2: default nil, contain input coins hash, which are used for creating this tx
func (tp *TxPool) MaybeAcceptTransaction(tx metadata.Transaction, beaconHeight int64) (*common.Hash, *TxDesc, error) {
	//tp.config.BlockChain.BestState.Beacon.BeaconHeight
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	if tp.IsTest {
		return &common.Hash{}, &TxDesc{}, nil
	}
	go func(txHash common.Hash) {
		tp.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.TransactionHashEnterNodeTopic, txHash))
	}(*tx.Hash())
	if !tp.checkRelayShard(tx) && !tp.checkPublicKeyRole(tx) {
		senderShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
		err := NewMempoolTxError(UnexpectedTransactionError, errors.New("Unexpected Transaction From Shard "+fmt.Sprintf("%d", senderShardID)))
		Logger.log.Error(err)
		return &common.Hash{}, &TxDesc{}, err
	}
	txType := tx.GetType()
	if txType == common.TxNormalType {
		if tx.IsPrivacy() {
			txType = metrics.TxNormalPrivacy
		} else {
			txType = metrics.TxNormalNoPrivacy
		}
	}
	txSize := fmt.Sprintf("%d", tx.GetTxActualSize())
	txTypePrivacyOrNot := metrics.TxPrivacy
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolTxBeginEnter,
		metrics.MeasurementValue: float64(1),
		metrics.Tag:              metrics.TxTypeTag,
		metrics.TagValue:         txType})
	//==========
	if uint64(len(tp.pool)) >= tp.config.MaxTx {
		return nil, nil, NewMempoolTxError(MaxPoolSizeError, errors.New("Pool reach max number of transaction"))
	}
	startAdd := time.Now()
	hash, txDesc, err := tp.maybeAcceptTransaction(tx, tp.config.PersistMempool, true, beaconHeight)
	elapsed := float64(time.Since(startAdd).Seconds())
	//==========
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolEntered,
		metrics.MeasurementValue: elapsed,
		metrics.Tag:              metrics.TxSizeTag,
		metrics.TagValue:         txSize})
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolEnteredWithType,
		metrics.MeasurementValue: elapsed,
		metrics.Tag:              metrics.TxSizeTag,
		metrics.TagValue:         txSize})
	size := len(tp.pool)
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.PoolSize,
		metrics.MeasurementValue: float64(size)})
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxAddedIntoPoolType,
		metrics.MeasurementValue: float64(1),
		metrics.Tag:              metrics.TxTypeTag,
		metrics.TagValue:         txType,
	})
	if !tx.IsPrivacy() {
		txTypePrivacyOrNot = metrics.TxNoPrivacy
	}
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolPrivacyOrNot,
		metrics.MeasurementValue: float64(1),
		metrics.Tag:              metrics.TxPrivacyOrNotTag,
		metrics.TagValue:         txTypePrivacyOrNot,
	})
	if err != nil {
		Logger.log.Error(err)
	} else {
		if tp.IsBlockGenStarted {
			if tp.IsUnlockMempool {
				go func(tx metadata.Transaction) {
					tp.CPendingTxs <- tx
				}(tx)
			}
		}
		// Publish Message
		go tp.config.PubSubManager.PublishMessage(pubsub.NewMessage(pubsub.MempoolInfoTopic, tp.listTxs()))
	}
	return hash, txDesc, err
}

// This function is safe for concurrent access.
func (tp *TxPool) MaybeAcceptTransactionForBlockProducing(tx metadata.Transaction, beaconHeight int64) (*metadata.TxDesc, error) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	_, txDesc, err := tp.maybeAcceptTransaction(tx, false, false, beaconHeight)
	if err != nil {
		Logger.log.Error(err)
		return nil, err
	}
	tempTxDesc := &txDesc.Desc
	return tempTxDesc, err
}

/*
// maybeAcceptTransaction into pool
// #1: tx
// #2: store into db
// #3: default nil, contain input coins hash, which are used for creating this tx
*/
func (tp *TxPool) maybeAcceptTransaction(tx metadata.Transaction, isStore bool, isNewTransaction bool, beaconHeight int64) (*common.Hash, *TxDesc, error) {
	txType := tx.GetType()
	if txType == common.TxNormalType {
		if tx.IsPrivacy() {
			txType = metrics.TxNormalPrivacy
		} else {
			txType = metrics.TxNormalNoPrivacy
		}
	}
	txSize := fmt.Sprintf("%d", tx.GetTxActualSize())
	startValidate := time.Now()
	// validate tx
	err := tp.validateTransaction(tx, beaconHeight)
	if err != nil {
		return nil, nil, err
	}
	elapsed := float64(time.Since(startValidate).Seconds())

	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidated,
		metrics.MeasurementValue: elapsed,
		metrics.Tag:              metrics.TxSizeTag,
		metrics.TagValue:         txSize})
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidatedWithType,
		metrics.MeasurementValue: elapsed,
		metrics.Tag:              metrics.TxSizeTag,
		metrics.TagValue:         txSize})
	shardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	bestHeight := tp.config.BlockChain.BestState.Shard[shardID].BestBlock.Header.Height
	txFee := tx.GetTxFee()
	txFeeToken := tx.GetTxFeeToken()
	txD := createTxDescMempool(tx, bestHeight, txFee, txFeeToken)
	startAdd := time.Now()
	err = tp.addTx(txD, isStore)
	if err != nil {
		return nil, nil, err
	}
	if isNewTransaction {
		Logger.log.Infof("Add New Txs Into Pool %+v FROM SHARD %+v\n", *tx.Hash(), shardID)
		go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
			metrics.Measurement:      metrics.TxPoolAddedAfterValidation,
			metrics.MeasurementValue: float64(time.Since(startAdd).Seconds()),
			metrics.Tag:              metrics.TxSizeTag,
			metrics.TagValue:         txSize})
	}
	return tx.Hash(), txD, nil
}

// createTxDescMempool - return an object TxDesc for mempool from original Tx
func createTxDescMempool(tx metadata.Transaction, height uint64, fee uint64, feeToken uint64) *TxDesc {
	txDesc := &TxDesc{
		Desc: metadata.TxDesc{
			Tx:       tx,
			Height:   height,
			Fee:      fee,
			FeeToken: feeToken,
		},
		StartTime:       time.Now(),
		IsFowardMessage: false,
	}
	return txDesc
}

/*
// maybeAcceptTransaction is the internal function which implements the public
// See the comment for MaybeAcceptTransaction for more details.
// This function MUST be called with the mempool lock held (for writes).
In Param#2: isStore: store transaction to persistence storage only work for transaction come from user (not for validation process)
1. Validate sanity data of tx
2. Validate duplicate tx
3. Do not accept a salary tx
4. Validate fee with tx size
5. Validate with other txs in mempool
5.1 Check for Replacement or Cancel transaction
6. Validate data in tx: privacy proof, metadata,...
7. Validate tx with blockchain: douple spend, ...
8. CustomInitToken: Check Custom Init Token try to init exist token ID
9. Staking Transaction: Check Duplicate stake public key in pool ONLY with staking transaction
10. RequestStopAutoStaking
*/
func (tp *TxPool) validateTransaction(tx metadata.Transaction, beaconHeight int64) error {
	var shardID byte
	var err error
	var now time.Time
	txHash := tx.Hash()
	txType := tx.GetType()
	if txType == common.TxNormalType {
		if tx.IsPrivacy() {
			txType = metrics.TxNormalPrivacy
		} else {
			txType = metrics.TxNormalNoPrivacy
		}
	}

	// Condition 1: sanity data
	now = time.Now()
	validated, err := tx.ValidateSanityData(tp.config.BlockChain)
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.Condition1,
		metrics.Tag:              metrics.ValidateConditionTag,
	})
	if !validated {
		return NewMempoolTxError(RejectSansityTx, fmt.Errorf("transaction's sansity %v is error %v", txHash.String(), err))
	}

	// Condition 2: Don't accept the transaction if it already exists in the pool.
	now = time.Now()
	isTxInPool := tp.isTxInPool(txHash)
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.Condition2,
		metrics.Tag:              metrics.ValidateConditionTag,
	})
	if isTxInPool {
		str := fmt.Sprintf("already have transaction %+v", txHash.String())
		go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
			metrics.Measurement:      metrics.TxPoolDuplicateTxs,
			metrics.MeasurementValue: float64(1),
		})
		return NewMempoolTxError(RejectDuplicateTx, fmt.Errorf("%+v", str))
	}
	// Condition 3: A standalone transaction must not be a salary transaction.
	now = time.Now()
	isSalaryTx := tx.IsSalaryTx()
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.Condition3,
		metrics.Tag:              metrics.ValidateConditionTag,
	})
	if isSalaryTx {
		return NewMempoolTxError(RejectSalaryTx, fmt.Errorf("%+v is salary tx", txHash.String()))
	}
	// Condition 4: check fee PRV of tx
	now = time.Now()
	switch tx.GetType() {
	case common.TxCustomTokenPrivacyType:
		{
			txPrivacyToken := tx.(*transaction.TxCustomTokenPrivacy)
			tokenID := txPrivacyToken.TxPrivacyTokenData.PropertyID
			feeNativeToken := tx.GetTxFee()
			feePToken := tx.GetTxFeeToken()

			//convert fee in Ptoken to fee in native token (if feePToken > 0)
			if feePToken > 0 {
				feePTokenToNativeToken, err := ConvertPrivacyTokenToNativeToken(feePToken, &tokenID, beaconHeight, tp.config.DataBase)
				if err != nil{
					return NewMempoolTxError(RejectInvalidFee,
						fmt.Errorf("transaction %+v: %+v %v can not convert to native token",
							txHash.String(), feePToken, tokenID))
				}

				feeNativeToken += feePTokenToNativeToken
			}

			// get limit fee in native token
			actualTxSize := tx.GetTxActualSize()
			limitFee := tp.config.FeeEstimator[shardID].GetLimitFeeForNativeToken()

			// check fee in native token
			minFee := actualTxSize * limitFee
			if feeNativeToken < minFee {
				return NewMempoolTxError(RejectInvalidFee,
					fmt.Errorf("transaction %+v has %d fees PRV which is under the required amount of %d, tx size %d",
						txHash.String(), feeNativeToken, minFee, actualTxSize))
			}

			//txPrivacyToken := tx.(*transaction.TxCustomTokenPrivacy)
			//isPaidByPRV := false
			//isPaidPartiallyPRV := false
			//// check PRV element and pToken element
			//if txPrivacyToken.Tx.Proof != nil || txPrivacyToken.TxPrivacyTokenData.Type == transaction.CustomTokenInit {
			//	// tx contain PRV data -> check with PRV fee
			//	// @notice: check limit fee but apply for token fee
			//	limitFee := tp.config.FeeEstimator[shardID].limitFee
			//	if limitFee > 0 {
			//		if txPrivacyToken.GetTxFeeToken() == 0 || // not paid with token -> use PRV for paying fee
			//			txPrivacyToken.TxPrivacyTokenData.Type == transaction.CustomTokenInit { // or init token -> need to use PRV for paying fee
			//			// paid all with PRV
			//			ok := txPrivacyToken.CheckTransactionFee(limitFee)
			//			if !ok {
			//				return NewMempoolTxError(RejectInvalidFee, fmt.Errorf("transaction %+v has %d fees PRV which is under the required amount of %d",
			//					txHash.String(),
			//					txPrivacyToken.Tx.GetTxFee(),
			//					limitFee*txPrivacyToken.GetTxActualSize()))
			//			}
			//			isPaidPartiallyPRV = false
			//		} else {
			//			// paid partial with PRV
			//			ok := txPrivacyToken.Tx.CheckTransactionFee(limitFee)
			//			if !ok {
			//				return NewMempoolTxError(RejectInvalidFee, fmt.Errorf("transaction %+v has %d fees PRV which is under the required amount of %d",
			//					txHash.String(),
			//					txPrivacyToken.Tx.GetTxFee(),
			//					limitFee*txPrivacyToken.Tx.GetTxActualSize()))
			//			}
			//			isPaidPartiallyPRV = true
			//		}
			//	}
			//	isPaidByPRV = true
			//}
			//if txPrivacyToken.TxPrivacyTokenData.TxNormal.Proof != nil {
			//	tokenID := txPrivacyToken.TxPrivacyTokenData.PropertyID
			//	limitFeeToken, isFeePToken := tp.config.FeeEstimator[shardID].GetLimitFeeForNativeToken(&tokenID)
			//	if limitFeeToken > 0 {
			//		if !isPaidByPRV {
			//			// not paid anything by PRV
			//			// -> check fee on total tx size(prv tx container + pToken tx) and use token as fee
			//			ok := txPrivacyToken.CheckTransactionFeeByFeeToken(limitFeeToken)
			//			if !ok {
			//				return NewMempoolTxError(RejectInvalidFee, fmt.Errorf("transaction %+v has %d fees which is under the required amount of %d",
			//					txHash.String(),
			//					tx.GetTxFeeToken(),
			//					limitFeeToken*txPrivacyToken.GetTxActualSize()))
			//			}
			//		} else {
			//			// paid by PRV
			//			if isPaidPartiallyPRV {
			//				// paid partially -> check fee on pToken tx data size(only for pToken tx)
			//				ok := txPrivacyToken.CheckTransactionFeeByFeeTokenForTokenData(limitFeeToken)
			//				if !ok {
			//					return NewMempoolTxError(RejectInvalidFee, fmt.Errorf("transaction %+v has %d fees which is under the required amount of %d",
			//						txHash.String(),
			//						tx.GetTxFeeToken(),
			//						limitFeeToken*txPrivacyToken.GetTxPrivacyTokenActualSize()))
			//				}
			//			}
			//		}
			//	}
			//}
		}
	case common.TxCustomTokenType:
		{
			// Only check like normal tx with PRV
			limitFee := tp.config.FeeEstimator[shardID].limitFee
			txCustomToken := tx.(*transaction.TxNormalToken)
			if limitFee > 0 {
				ok := txCustomToken.CheckTransactionFee(limitFee)
				if !ok {
					return NewMempoolTxError(RejectInvalidFee, fmt.Errorf("transaction %+v has %d fees which is under the required amount of %d", txHash.String(), txCustomToken.GetTxFee(), limitFee*tx.GetTxActualSize()))
				}
			}
		}
	default:
		{
			// This is a normal tx -> only check like normal tx with PRV
			limitFee := tp.config.FeeEstimator[shardID].limitFee
			txNormal := tx.(*transaction.Tx)
			if limitFee > 0 {
				txFee := txNormal.GetTxFee()
				ok := tx.CheckTransactionFee(limitFee)
				if !ok {
					return NewMempoolTxError(RejectInvalidFee,
						fmt.Errorf("transaction %+v has %d fees which is under the required amount of %d",
							txHash.String(), txFee, limitFee*tx.GetTxActualSize()))
				}
			}
		}
	}
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.Condition4,
		metrics.Tag:              metrics.ValidateConditionTag,
	})

	// Condition 5: check tx with all txs in current mempool
	now = time.Now()
	err = tx.ValidateTxWithCurrentMempool(tp)
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.Condition5,
		metrics.Tag:              metrics.ValidateConditionTag,
	})
	if err != nil {
		now := time.Now()
		replaceErr, isReplacedTx := tp.validateTransactionReplacement(tx)
		go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
			metrics.Measurement:      metrics.TxPoolValidationDetails,
			metrics.MeasurementValue: float64(time.Since(now).Seconds()),
			metrics.TagValue:         metrics.ReplaceTxMetic,
			metrics.Tag:              metrics.ValidateConditionTag,
		})
		// if replace tx success (no replace error found) then continue with next validate condition
		if isReplacedTx {
			if replaceErr != nil {
				return replaceErr
			}
		} else {
			// replace fail
			return NewMempoolTxError(RejectDoubleSpendWithMempoolTx, err)
		}
	}
	// Condition 6: ValidateTransaction tx by it self
	shardID = common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	now = time.Now()
	validated, errValidateTxByItself := tx.ValidateTxByItself(tx.IsPrivacy(), tp.config.DataBase, tp.config.BlockChain, shardID)
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.VTBITxTypeMetic,
		metrics.Tag:              metrics.ValidateConditionTag,
	})
	if !validated {
		return NewMempoolTxError(RejectInvalidTx, fmt.Errorf("Invalid tx - %+v", errValidateTxByItself))
	}

	// Condition 7: validate tx with data of blockchain
	now = time.Now()
	err = tx.ValidateTxWithBlockChain(tp.config.BlockChain, shardID, tp.config.DataBase)
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.Condition7,
		metrics.Tag:              metrics.ValidateConditionTag,
	})
	if err != nil {
		return NewMempoolTxError(RejectDoubleSpendWithBlockchainTx, err)
	}
	// Condition 8: init exist custom token or not
	now = time.Now()
	foundTokenID := -1
	tokenID := ""
	if tx.GetType() == common.TxCustomTokenType {
		customTokenTx := tx.(*transaction.TxNormalToken)
		if customTokenTx.TxTokenData.Type == transaction.CustomTokenInit {
			tokenID = customTokenTx.TxTokenData.PropertyID.String()
			tp.tokenIDMtx.RLock()
			foundTokenID = common.IndexOfStrInHashMap(tokenID, tp.poolTokenID)
			tp.tokenIDMtx.RUnlock()
		}
	}
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.Condition8,
		metrics.Tag:              metrics.ValidateConditionTag,
	})
	if foundTokenID > 0 {
		return NewMempoolTxError(RejectDuplicateInitTokenTx, fmt.Errorf("Init Transaction of this Token is in pool already %+v", tokenID))
	}
	// Condition 9: check duplicate stake public key ONLY with staking transaction
	now = time.Now()
	pubkey := ""
	foundPubkey := -1
	if tx.GetMetadata() != nil {
		if tx.GetMetadata().GetType() == metadata.ShardStakingMeta || tx.GetMetadata().GetType() == metadata.BeaconStakingMeta {
			stakingMetadata, ok := tx.GetMetadata().(*metadata.StakingMetadata)
			if !ok {
				return NewMempoolTxError(GetStakingMetadataError, fmt.Errorf("Expect metadata type to be *metadata.StakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata())))
			}
			pubkey = stakingMetadata.CommitteePublicKey
			tp.candidateMtx.RLock()
			foundPubkey = common.IndexOfStrInHashMap(stakingMetadata.CommitteePublicKey, tp.poolCandidate)
			tp.candidateMtx.RUnlock()
		}
	}
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.Condition9,
		metrics.Tag:              metrics.ValidateConditionTag,
	})
	if foundPubkey > 0 {
		return NewMempoolTxError(RejectDuplicateStakePubkey, fmt.Errorf("This public key already stake and still in pool %+v", pubkey))
	}
	// Condition 10: check duplicate request stop auto staking
	now = time.Now()
	requestedPublicKey := ""
	foundRequestStopAutoStaking := -1
	if tx.GetMetadata() != nil {
		if tx.GetMetadata().GetType() == metadata.StopAutoStakingMeta {
			stopAutoStakingMetadata, ok := tx.GetMetadata().(*metadata.StopAutoStakingMetadata)
			if !ok {
				return NewMempoolTxError(GetStakingMetadataError, fmt.Errorf("Expect metadata type to be *metadata.StopAutoStakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata())))
			}
			requestedPublicKey = stopAutoStakingMetadata.CommitteePublicKey
			tp.requestStopStakingMtx.RLock()
			foundRequestStopAutoStaking = common.IndexOfStrInHashMap(stopAutoStakingMetadata.CommitteePublicKey, tp.poolRequestStopStaking)
			tp.requestStopStakingMtx.RUnlock()
		}
	}
	go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
		metrics.Measurement:      metrics.TxPoolValidationDetails,
		metrics.MeasurementValue: float64(time.Since(now).Seconds()),
		metrics.TagValue:         metrics.Condition10,
		metrics.Tag:              metrics.ValidateConditionTag,
	})
	if foundRequestStopAutoStaking > 0 {
		return NewMempoolTxError(RejectDuplicateRequestStopAutoStaking, fmt.Errorf("This public key already request to stop auto staking and still in pool %+v", requestedPublicKey))
	}
	return nil
}

// check transaction in pool
func (tp *TxPool) isTxInPool(hash *common.Hash) bool {
	if _, exists := tp.pool[*hash]; exists {
		return true
	}
	return false
}

func (tp *TxPool) validateTransactionReplacement(tx metadata.Transaction) (error, bool) {
	// calculate match serial number list in pool for replaced tx
	serialNumberHashList := tx.ListSerialNumbersHashH()
	hash := common.HashArrayOfHashArray(serialNumberHashList)
	// find replace tx in pool
	if txHashToBeReplaced, ok := tp.poolSerialNumberHash[hash]; ok {
		if txDescToBeReplaced, ok := tp.pool[txHashToBeReplaced]; ok {
			var baseReplaceFee float64
			var baseReplaceFeeToken float64
			var replaceFee float64
			var replaceFeeToken float64
			var isReplaced = false
			if txDescToBeReplaced.Desc.Fee > 0 && txDescToBeReplaced.Desc.FeeToken == 0 {
				// paid by prv fee only
				baseReplaceFee = float64(txDescToBeReplaced.Desc.Fee)
				replaceFee = float64(tx.GetTxFee())
				// not a higher enough fee than return error
				if baseReplaceFee*tp.ReplaceFeeRatio >= replaceFee {
					return NewMempoolTxError(RejectReplacementTxError, fmt.Errorf("Expect fee to be greater than %+v but get %+v ", baseReplaceFee, replaceFee)), true
				}
				isReplaced = true
			} else if txDescToBeReplaced.Desc.Fee == 0 && txDescToBeReplaced.Desc.FeeToken > 0 {
				//paid by token fee only
				baseReplaceFeeToken = float64(txDescToBeReplaced.Desc.FeeToken)
				replaceFeeToken = float64(tx.GetTxFeeToken())
				// not a higher enough fee than return error
				if baseReplaceFeeToken*tp.ReplaceFeeRatio >= replaceFeeToken {
					return NewMempoolTxError(RejectReplacementTxError, fmt.Errorf("Expect fee to be greater than %+v but get %+v ", baseReplaceFeeToken, replaceFeeToken)), true
				}
				isReplaced = true
			} else if txDescToBeReplaced.Desc.Fee > 0 && txDescToBeReplaced.Desc.FeeToken > 0 {
				// paid by both prv fee and token fee
				// then only one of fee is higher then it will be accepted
				baseReplaceFee = float64(txDescToBeReplaced.Desc.Fee)
				replaceFee = float64(tx.GetTxFee())
				baseReplaceFeeToken = float64(txDescToBeReplaced.Desc.FeeToken)
				replaceFeeToken = float64(tx.GetTxFeeToken())
				// not a higher enough fee than return error
				if baseReplaceFee*tp.ReplaceFeeRatio >= replaceFee || baseReplaceFeeToken*tp.ReplaceFeeRatio >= replaceFeeToken {
					return NewMempoolTxError(RejectReplacementTxError, fmt.Errorf("Expect fee to be greater than %+v but get %+v ", baseReplaceFee, replaceFee)), true
				}
				isReplaced = true
			}
			if isReplaced {
				txToBeReplaced := txDescToBeReplaced.Desc.Tx
				tp.removeTx(txToBeReplaced)
				tp.TriggerCRemoveTxs(txToBeReplaced)
				tp.removeRequestStopStakingByTxHash(*txToBeReplaced.Hash())
				tp.removeTokenIDByTxHash(*txToBeReplaced.Hash())
				// send tx into channel of CRmoveTxs
				tp.TriggerCRemoveTxs(tx)
				return nil, true
			} else {
				return NewMempoolTxError(RejectReplacementTxError, fmt.Errorf("Unexpected error occur")), true
			}
		} else {
			//found no tx to be replaced
			return nil, false
		}
	} else {
		// no match serial number list to be replaced
		return nil, false
	}
}

// TriggerCRemoveTxs - send a tx channel into CRemoveTxs of tx mempool
func (tp *TxPool) TriggerCRemoveTxs(tx metadata.Transaction) {
	if tp.IsBlockGenStarted {
		go func(tx metadata.Transaction) {
			tp.CRemoveTxs <- tx
		}(tx)
	}
}

/*
// add transaction into pool
// #1: tx
// #2: store into db
// #3: default nil, contain input coins hash, which are used for creating this tx
*/
func (tp *TxPool) addTx(txD *TxDesc, isStore bool) error {
	tx := txD.Desc.Tx
	txHash := tx.Hash()
	if isStore {
		err := tp.addTransactionToDatabaseMempool(txHash, *txD)
		if err != nil {
			Logger.log.Errorf("Fail to add tx %+v to mempool database %+v \n", *txHash, err)
		} else {
			Logger.log.Criticalf("Add tx %+v to mempool database success \n", *txHash)
		}
	}
	tp.pool[*txHash] = txD
	var serialNumberList []common.Hash
	serialNumberList = append(serialNumberList, txD.Desc.Tx.ListSerialNumbersHashH()...)
	serialNumberListHash := common.HashArrayOfHashArray(serialNumberList)
	tp.poolSerialNumberHash[serialNumberListHash] = *txD.Desc.Tx.Hash()
	tp.poolSerialNumbersHashList[*txHash] = serialNumberList
	atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())
	// Record this tx for fee estimation if enabled, apply for normal tx and privacy token tx
	if tp.config.FeeEstimator != nil {
		var shardID byte
		flag := false
		switch tx.GetType() {
		case common.TxNormalType:
			{
				shardID = common.GetShardIDFromLastByte(tx.(*transaction.Tx).PubKeyLastByteSender)
				flag = true
			}
		case common.TxCustomTokenPrivacyType:
			{
				shardID = common.GetShardIDFromLastByte(tx.(*transaction.TxCustomTokenPrivacy).PubKeyLastByteSender)
				flag = true
			}
		}
		if flag {
			if temp, ok := tp.config.FeeEstimator[shardID]; ok {
				temp.ObserveTransaction(txD)
			}
		}
	}
	// add candidate into candidate list ONLY with staking transaction
	if tx.GetMetadata() != nil {
		metadataType := tx.GetMetadata().GetType()
		switch metadataType {
		case metadata.ShardStakingMeta, metadata.BeaconStakingMeta:
			{
				stakingMetadata, ok := tx.GetMetadata().(*metadata.StakingMetadata)
				if !ok {
					return NewMempoolTxError(GetStakingMetadataError, fmt.Errorf("Expect metadata type to be *metadata.StakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata())))
				}
				tp.addCandidateToList(*txHash, stakingMetadata.CommitteePublicKey)
			}
		case metadata.StopAutoStakingMeta:
			{
				stopAutoStakingMetadata, ok := tx.GetMetadata().(*metadata.StopAutoStakingMetadata)
				if !ok {
					return NewMempoolTxError(GetStakingMetadataError, fmt.Errorf("Expect metadata type to be *metadata.StopAutoStakingMetadata but get %+v", reflect.TypeOf(tx.GetMetadata())))
				}
				tp.addRequestStopStakingToList(*txHash, stopAutoStakingMetadata.CommitteePublicKey)
			}
		default:
			{
				Logger.log.Debug("Metadata Type:", metadataType)
			}
		}
	}
	if tx.GetType() == common.TxCustomTokenType {
		customTokenTx := tx.(*transaction.TxNormalToken)
		if customTokenTx.TxTokenData.Type == transaction.CustomTokenInit {
			tokenID := customTokenTx.TxTokenData.PropertyID.String()
			tp.addTokenIDToList(*txHash, tokenID)
		}
	}
	Logger.log.Infof("Add Transaction %+v Successs \n", txHash.String())
	return nil
}

// Check relay shard and public key role before processing transaction
func (tp *TxPool) checkRelayShard(tx metadata.Transaction) bool {
	senderShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	if common.IndexOfByte(senderShardID, tp.config.RelayShards) > -1 {
		return true
	}
	return false
}

func (tp *TxPool) checkPublicKeyRole(tx metadata.Transaction) bool {
	senderShardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	tp.roleMtx.RLock()
	if tp.RoleInCommittees > -1 && byte(tp.RoleInCommittees) == senderShardID {
		tp.roleMtx.RUnlock()
		return true
	} else {
		tp.roleMtx.RUnlock()
		return false
	}
}

// RemoveTx safe remove transaction for pool
func (tp *TxPool) RemoveTx(txs []metadata.Transaction, isInBlock bool) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	// remove transaction from database mempool
	for _, tx := range txs {
		var now time.Time
		start := time.Now()
		now = time.Now()
		txDesc, ok := tp.pool[*tx.Hash()]
		if !ok {
			continue
		}
		txType := tx.GetType()
		if txType == common.TxNormalType {
			if tx.IsPrivacy() {
				txType = metrics.TxNormalPrivacy
			} else {
				txType = metrics.TxNormalNoPrivacy
			}
		}
		startTime := txDesc.StartTime
		if tp.config.PersistMempool {
			err := tp.removeTransactionFromDatabaseMP(tx.Hash())
			if err != nil {
				Logger.log.Error(err)
			}
		}
		go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
			metrics.Measurement:      metrics.TxPoolRemovedTimeDetails,
			metrics.MeasurementValue: float64(time.Since(now).Seconds()),
			metrics.Tag:              metrics.ValidateConditionTag,
			metrics.TagValue:         metrics.Condition1,
		})
		now = time.Now()
		tp.removeTx(tx)
		tp.TriggerCRemoveTxs(tx)
		go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
			metrics.Measurement:      metrics.TxPoolRemovedTimeDetails,
			metrics.MeasurementValue: float64(time.Since(now).Seconds()),
			metrics.Tag:              metrics.ValidateConditionTag,
			metrics.TagValue:         metrics.Condition2,
		})
		now = time.Now()
		if isInBlock {
			elapsed := float64(time.Since(startTime).Seconds())
			txSize := fmt.Sprintf("%d", tx.GetTxActualSize())
			go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
				metrics.Measurement:      metrics.TxPoolRemoveAfterInBlock,
				metrics.MeasurementValue: elapsed,
				metrics.Tag:              metrics.TxSizeTag,
				metrics.TagValue:         txSize,
			})
			go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
				metrics.Measurement:      metrics.TxPoolRemoveAfterInBlockWithType,
				metrics.MeasurementValue: elapsed,
				metrics.Tag:              metrics.TxSizeWithTypeTag,
				metrics.TagValue:         txType + txSize,
			})
		}
		go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
			metrics.Measurement:      metrics.TxPoolRemovedNumber,
			metrics.MeasurementValue: float64(1),
		})
		size := len(tp.pool)
		go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
			metrics.Measurement:      metrics.PoolSize,
			metrics.MeasurementValue: float64(size),
		})
		go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
			metrics.Measurement:      metrics.TxPoolRemovedTimeDetails,
			metrics.MeasurementValue: float64(time.Since(now).Seconds()),
			metrics.Tag:              metrics.ValidateConditionTag,
			metrics.TagValue:         metrics.Condition3,
		})
		go metrics.AnalyzeTimeSeriesMetricData(map[string]interface{}{
			metrics.Measurement:      metrics.TxPoolRemovedTime,
			metrics.MeasurementValue: float64(time.Since(start).Seconds()),
			metrics.Tag:              metrics.TxTypeTag,
			metrics.TagValue:         txType,
		})
	}
	return
}

/*
	- Remove transaction out of pool
		+ Tx Description pool
		+ List Serial Number Pool
		+ Hash of List Serial Number Pool
	- Transaction want to be removed maybe replaced by another transaction:
		+ New tx (Replacement tx) still exist in pool
		+ Using the same list serial number to delete new transaction out of pool
*/
func (tp *TxPool) removeTx(tx metadata.Transaction) {
	//Logger.log.Infof((*tx).Hash().String())
	if _, exists := tp.pool[*tx.Hash()]; exists {
		delete(tp.pool, *tx.Hash())
		atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())
	}
	if _, exists := tp.poolSerialNumbersHashList[*tx.Hash()]; exists {
		delete(tp.poolSerialNumbersHashList, *tx.Hash())
	}
	serialNumberHashList := tx.ListSerialNumbersHashH()
	hash := common.HashArrayOfHashArray(serialNumberHashList)
	if _, exists := tp.poolSerialNumberHash[hash]; exists {
		delete(tp.poolSerialNumberHash, hash)
		// Using the same list serial number to delete new transaction out of pool
		// this new transaction maybe not exist
		if _, exists := tp.pool[hash]; exists {
			delete(tp.pool, hash)
			atomic.StoreInt64(&tp.lastUpdated, time.Now().Unix())
		}
		if _, exists := tp.poolSerialNumbersHashList[hash]; exists {
			delete(tp.poolSerialNumbersHashList, hash)
		}
	}
}

func (tp *TxPool) addCandidateToList(txHash common.Hash, candidate string) {
	tp.candidateMtx.Lock()
	defer tp.candidateMtx.Unlock()
	tp.poolCandidate[txHash] = candidate
}

func (tp *TxPool) removeCandidateByTxHash(txHash common.Hash) {
	tp.candidateMtx.Lock()
	defer tp.candidateMtx.Unlock()
	if _, exist := tp.poolCandidate[txHash]; exist {
		delete(tp.poolCandidate, txHash)
	}
}

func (tp *TxPool) RemoveCandidateList(candidate []string) {
	tp.candidateMtx.Lock()
	defer tp.candidateMtx.Unlock()
	candidateToBeRemoved := []common.Hash{}
	for _, value := range candidate {
		for txHash, currentCandidate := range tp.poolCandidate {
			if strings.Compare(value, currentCandidate) == 0 {
				candidateToBeRemoved = append(candidateToBeRemoved, txHash)
				break
			}
		}
	}
	for _, txHash := range candidateToBeRemoved {
		delete(tp.poolCandidate, txHash)
	}
}
func (tp *TxPool) addRequestStopStakingToList(txHash common.Hash, requestStopStaking string) {
	tp.requestStopStakingMtx.Lock()
	defer tp.requestStopStakingMtx.Unlock()
	tp.poolRequestStopStaking[txHash] = requestStopStaking
}

func (tp *TxPool) removeRequestStopStakingByTxHash(txHash common.Hash) {
	tp.requestStopStakingMtx.Lock()
	defer tp.requestStopStakingMtx.Unlock()
	if _, exist := tp.poolRequestStopStaking[txHash]; exist {
		delete(tp.poolRequestStopStaking, txHash)
	}
}

func (tp *TxPool) RemoveRequestStopStakingList(requestStopStakings []string) {
	tp.requestStopStakingMtx.Lock()
	defer tp.requestStopStakingMtx.Unlock()
	requestStopStakingsToBeRemoved := []common.Hash{}
	for _, requestStopStaking := range requestStopStakings {
		for txHash, currentRequestStopStaking := range tp.poolRequestStopStaking {
			if strings.Compare(requestStopStaking, currentRequestStopStaking) == 0 {
				requestStopStakingsToBeRemoved = append(requestStopStakingsToBeRemoved, txHash)
				break
			}
		}
	}
	for _, txHash := range requestStopStakingsToBeRemoved {
		delete(tp.poolRequestStopStaking, txHash)
	}
}
func (tp *TxPool) addTokenIDToList(txHash common.Hash, tokenID string) {
	tp.tokenIDMtx.Lock()
	defer tp.tokenIDMtx.Unlock()
	tp.poolTokenID[txHash] = tokenID
}

func (tp *TxPool) removeTokenIDByTxHash(txHash common.Hash) {
	tp.tokenIDMtx.Lock()
	defer tp.tokenIDMtx.Unlock()
	if _, exist := tp.poolTokenID[txHash]; exist {
		delete(tp.poolTokenID, txHash)
	}
}

func (tp *TxPool) RemoveTokenIDList(tokenID []string) {
	tp.tokenIDMtx.Lock()
	defer tp.tokenIDMtx.Unlock()
	tokenToBeRemoved := []common.Hash{}
	for _, value := range tokenID {
		for txHash, currentToken := range tp.poolTokenID {
			if strings.Compare(value, currentToken) == 0 {
				tokenToBeRemoved = append(tokenToBeRemoved, txHash)
				break
			}
		}
	}
	for _, txHash := range tokenToBeRemoved {
		delete(tp.poolTokenID, txHash)
	}
}

//=======================Service for other package
// SendTransactionToBlockGen - push tx into channel and send to Block generate of consensus
func (tp *TxPool) SendTransactionToBlockGen() {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	for _, txdesc := range tp.pool {
		tp.CPendingTxs <- txdesc.Desc.Tx
	}
	tp.IsUnlockMempool = true
}

// MarkForwardedTransaction - mart a transaction is forward message
func (tp *TxPool) MarkForwardedTransaction(txHash common.Hash) {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	if tp.IsTest {
		return
	}
	if _, ok := tp.pool[txHash]; ok {
		tp.pool[txHash].IsFowardMessage = true
	}
}

// GetTx get transaction info by hash
func (tp *TxPool) GetTx(txHash *common.Hash) (metadata.Transaction, error) {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	Logger.log.Info(txHash.String())
	txDesc, exists := tp.pool[*txHash]
	if exists {
		return txDesc.Desc.Tx, nil
	}
	err := NewMempoolTxError(TransactionNotFoundError, fmt.Errorf("Transaction "+txHash.String()+" Not Found!"))
	Logger.log.Error(err)
	return nil, err
}

// // MiningDescs returns a slice of mining descriptors for all the transactions
// // in the pool.
func (tp *TxPool) MiningDescs() []*metadata.TxDesc {
	tp.mtx.Lock()
	defer tp.mtx.Unlock()
	descs := []*metadata.TxDesc{}
	for _, desc := range tp.pool {
		descs = append(descs, &desc.Desc)
	}
	return descs
}

func (tp TxPool) GetPool() map[common.Hash]*TxDesc {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	pool := make(map[common.Hash]*TxDesc)
	for k, v := range tp.pool {
		pool[k] = v
	}
	return tp.pool
}

// Count return len of transaction pool
func (tp *TxPool) Count() int {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	count := len(tp.pool)
	return count
}

func (tp TxPool) GetClonedPoolCandidate() map[common.Hash]string {
	tp.candidateMtx.RLock()
	defer tp.candidateMtx.RUnlock()
	result := make(map[common.Hash]string)
	for k, v := range tp.poolCandidate {
		result[k] = v
	}
	return result
}

/*
Sum of all transactions sizes
*/
func (tp *TxPool) Size() uint64 {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	size := uint64(0)
	for _, tx := range tp.pool {
		size += tx.Desc.Tx.GetTxActualSize()
	}
	return size
}

// Get Max fee
func (tp *TxPool) MaxFee() uint64 {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	fee := uint64(0)
	for _, tx := range tp.pool {
		if tx.Desc.Fee > fee {
			fee = tx.Desc.Fee
		}
	}
	return fee
}

/*
// LastUpdated returns the last time a transaction was added to or
	// removed from the source pool.
*/
func (tp *TxPool) LastUpdated() time.Time {
	return time.Unix(tp.lastUpdated, 0)
}

/*
// HaveTransaction returns whether or not the passed transaction hash
	// exists in the source pool.
*/
func (tp *TxPool) HaveTransaction(hash *common.Hash) bool {
	// Protect concurrent access.
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	haveTx := tp.isTxInPool(hash)
	return haveTx
}

/*
List all tx ids in mempool
*/
func (tp *TxPool) ListTxs() []string {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	return tp.listTxs()
}

/*
List all tx ids in mempool
*/
func (tp *TxPool) listTxs() []string {
	result := make([]string, 0)
	for _, tx := range tp.pool {
		result = append(result, tx.Desc.Tx.Hash().String())
	}
	return result
}

/*
List all tx ids in mempool
*/
func (tp *TxPool) ListTxsDetail() []metadata.Transaction {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	result := make([]metadata.Transaction, 0)
	for _, tx := range tp.pool {
		result = append(result, tx.Desc.Tx)
	}
	return result
}

// ValidateSerialNumberHashH - check serialNumberHashH which is
// used by a tx in mempool
func (tp *TxPool) ValidateSerialNumberHashH(serialNumber []byte) error {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()
	hash := common.HashH(serialNumber)
	for txHash, serialNumbersHashH := range tp.poolSerialNumbersHashList {
		for _, serialNumberHashH := range serialNumbersHashH {
			if serialNumberHashH.IsEqual(&hash) {
				return NewMempoolTxError(DuplicateSerialNumbersHashError, fmt.Errorf("Transaction %+v use duplicate current serial number in pool", txHash))
			}
		}
	}
	return nil
}

func (tp *TxPool) EmptyPool() bool {
	tp.candidateMtx.Lock()
	tp.tokenIDMtx.Lock()
	defer tp.candidateMtx.Unlock()
	defer tp.tokenIDMtx.Unlock()
	if len(tp.pool) == 0 && len(tp.poolSerialNumbersHashList) == 0 && len(tp.poolCandidate) == 0 && len(tp.poolTokenID) == 0 {
		return true
	}
	tp.pool = make(map[common.Hash]*TxDesc)
	tp.poolSerialNumbersHashList = make(map[common.Hash][]common.Hash)
	tp.poolSerialNumberHash = make(map[common.Hash]common.Hash)
	tp.poolCandidate = make(map[common.Hash]string)
	tp.poolTokenID = make(map[common.Hash]string)
	tp.poolRequestStopStaking = make(map[common.Hash]string)
	if len(tp.pool) == 0 && len(tp.poolSerialNumbersHashList) == 0 && len(tp.poolSerialNumberHash) == 0 && len(tp.poolCandidate) == 0 && len(tp.poolTokenID) == 0 && len(tp.poolRequestStopStaking) == 0 {
		return true
	}
	return false
}

func (tp *TxPool) calPoolSize() uint64 {
	var totalSize uint64
	for _, txDesc := range tp.pool {
		size := txDesc.Desc.Tx.GetTxActualSize()
		totalSize += size
	}
	return totalSize
}

// ----------- transaction.MempoolRetriever's implementation -----------------
func (tp TxPool) GetSerialNumbersHashH() map[common.Hash][]common.Hash {
	//tp.mtx.RLock()
	//defer tp.mtx.RUnlock()
	m := make(map[common.Hash][]common.Hash)
	for k, hashList := range tp.poolSerialNumbersHashList {
		m[k] = []common.Hash{}
		for _, v := range hashList {
			m[k] = append(m[k], v)
		}
	}
	return m
}

func (tp TxPool) GetTxsInMem() map[common.Hash]metadata.TxDesc {
	//tp.mtx.RLock()
	//defer tp.mtx.RUnlock()
	txsInMem := make(map[common.Hash]metadata.TxDesc)
	for hash, txDesc := range tp.pool {
		txsInMem[hash] = txDesc.Desc
	}
	return txsInMem
}

// ----------- end of transaction.MempoolRetriever's implementation -----------------
