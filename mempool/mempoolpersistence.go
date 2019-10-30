package mempool

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/databasemp/lvdb"
	"github.com/incognitochain/incognito-chain/transaction"
)

type TempDesc struct {
	StartTime     time.Time
	IsPushMessage bool
	Height        uint64
	Fee           uint64
	FeePerKB      int32
}

// addTransactionToDatabaseMempool - Add a transaction data into mempool database
func (tp *TxPool) addTransactionToDatabaseMempool(txHash *common.Hash, txDesc TxDesc) error {
	tx := txDesc.Desc.Tx
	tempDesc := TempDesc{
		StartTime:     txDesc.StartTime,
		IsPushMessage: txDesc.IsFowardMessage,
		Height:        txDesc.Desc.Height,
		Fee:           txDesc.Desc.Fee,
		FeePerKB:      txDesc.Desc.FeePerKB,
	}
	switch tx.GetType() {
	//==================For PRV Transfer Only
	case common.TxNormalType:
		{
			normalTx := tx.(*transaction.Tx)
			valueTx, err := json.Marshal(normalTx)
			if err != nil {
				return err
			}
			valueDesc, err := json.Marshal(tempDesc)
			if err != nil {
				return err
			}
			err = tp.config.DataBaseMempool.AddTransaction(txHash, common.TxNormalType, valueTx, valueDesc)
			if err != nil {
				return err
			}
		}
	//==================For PRV & TxNormalToken Transfer
	case common.TxCustomTokenType:
		{
			customTokenTx := tx.(*transaction.TxNormalToken)
			valueTx, err := json.Marshal(customTokenTx)
			if err != nil {
				return err
			}
			valueDesc, err := json.Marshal(tempDesc)
			if err != nil {
				return err
			}
			err = tp.config.DataBaseMempool.AddTransaction(txHash, common.TxCustomTokenType, valueTx, valueDesc)
			if err != nil {
				return err
			}
		}
	case common.TxCustomTokenPrivacyType:
		{
			customTokenPrivacyTx := tx.(*transaction.TxCustomTokenPrivacy)
			valueTx, err := json.Marshal(customTokenPrivacyTx)
			if err != nil {
				return err
			}
			valueDesc, err := json.Marshal(tempDesc)
			if err != nil {
				return err
			}
			err = tp.config.DataBaseMempool.AddTransaction(txHash, common.TxCustomTokenPrivacyType, valueTx, valueDesc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// getTransactionFromDatabaseMempool - get tx from mempool database
func (tp *TxPool) getTransactionFromDatabaseMempool(txHash *common.Hash) (*TxDesc, error) {
	value, err := tp.config.DataBaseMempool.GetTransaction(txHash)
	values := strings.Split(string(value), string(lvdb.Splitter))
	if len(values) != 3 {
		return nil, err
	}
	txDesc, err := unMarshallTxDescFromDatabase(values[0], []byte(values[1]), []byte(values[2]))
	if err != nil {
		return nil, err
	}
	return txDesc, nil
}

// resetDatabaseMempool - reset data in data mempool
func (tp *TxPool) resetDatabaseMempool() error {
	return tp.config.DataBaseMempool.Reset()
}

// loadDatabaseMP - Get all tx in mempool database persistence
func (tp *TxPool) loadDatabaseMP() ([]TxDesc, error) {
	txDescs := []TxDesc{}
	allTxHashes, allTxs, err := tp.config.DataBaseMempool.Load()
	ttl := time.Duration(tp.config.TxLifeTime) * time.Second
	if err != nil {
		return txDescs, err
	}
	for index, tx := range allTxs {
		values := strings.Split(string(tx), string(lvdb.Splitter))
		txDesc, err := unMarshallTxDescFromDatabase(values[0], []byte(values[1]), []byte(values[2]))
		if err != nil {
			Logger.log.Error(err)
			txHash, err := common.Hash{}.NewHash(allTxHashes[index][3:])
			if err != nil {
				Logger.log.Error(err)
				continue
			}
			// fail to ummarshall transaction then remove
			err1 := tp.removeTransactionFromDatabaseMP(txHash)
			if err1 != nil {
				Logger.log.Error(err1)
			}
			continue
		}
		//if transaction is timeout then remove
		if time.Since(txDesc.StartTime) > ttl {
			err1 := tp.removeTransactionFromDatabaseMP(txDesc.Desc.Tx.Hash())
			if err1 != nil {
				Logger.log.Error(err1)
			}
		}
		//if not validated by current blockchain db then remove
		err = tp.validateTransaction(txDesc.Desc.Tx, -1)
		if err != nil {
			Logger.log.Error(err)
			err1 := tp.removeTransactionFromDatabaseMP(txDesc.Desc.Tx.Hash())
			if err1 != nil {
				Logger.log.Error(err1)
			}
			continue
		}

		err = tp.addTx(txDesc, false)
		if err != nil {
			Logger.log.Error(err)
		}
		txDescs = append(txDescs, *txDesc)
	}
	return txDescs, nil
}

// removeTransactionFromDatabaseMP - remove tx from mempool db persistence
func (tp *TxPool) removeTransactionFromDatabaseMP(txHash *common.Hash) error {
	if has, _ := tp.config.DataBaseMempool.HasTransaction(txHash); has {
		err := tp.config.DataBaseMempool.RemoveTransaction(txHash)
		return err
	}
	return nil
}

// unMarshallTxDescFromDatabase - convert tx data in mempool database persistence into TxDesc
func unMarshallTxDescFromDatabase(txType string, valueTx []byte, valueDesc []byte) (*TxDesc, error) {
	txDesc := TxDesc{}
	switch txType {
	case common.TxNormalType:
		{
			tx := transaction.Tx{}
			err := json.Unmarshal(valueTx, &tx)
			if err != nil {
				return nil, err
			}

			txDesc.Desc.Tx = &tx
		}
	case common.TxCustomTokenType:
		{
			customTokenTx := transaction.TxNormalToken{}
			err := json.Unmarshal(valueTx, &customTokenTx)
			if err != nil {
				return nil, err
			}
			txDesc.Desc.Tx = &customTokenTx
		}
	case common.TxCustomTokenPrivacyType:
		{
			customTokenPrivacyTx := transaction.TxCustomTokenPrivacy{}
			err := json.Unmarshal(valueTx, &customTokenPrivacyTx)
			if err != nil {
				return nil, err
			}
			txDesc.Desc.Tx = &customTokenPrivacyTx
		}
	}
	tempDesc := TempDesc{}
	err := json.Unmarshal(valueDesc, &tempDesc)
	if err != nil {
		return nil, err
	}
	txDesc.IsFowardMessage = tempDesc.IsPushMessage
	txDesc.StartTime = tempDesc.StartTime
	txDesc.Desc.Height = tempDesc.Height
	txDesc.Desc.Fee = tempDesc.Fee
	txDesc.Desc.FeePerKB = tempDesc.FeePerKB
	return &txDesc, nil
}
