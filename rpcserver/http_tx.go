package rpcserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"github.com/incognitochain/incognito-chain/wallet"
)

/*
// handleCreateTransaction handles createtransaction commands.
*/
func (httpServer *HttpServer) handleCreateRawTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateRawTransaction params: %+v", params)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	txHash, txBytes, txShardID, err := httpServer.txService.CreateRawTransaction(createRawTxParam, nil, *httpServer.config.Database)
	if err != nil {
		// return hex for a new tx
		return nil, err
	}

	result := jsonresult.NewCreateTransactionResult(txHash, common.EmptyString, txBytes, txShardID)
	Logger.log.Debugf("handleCreateRawTransaction result: %+v", result)
	return result, nil
}

/*
// handleSendTransaction implements the sendtransaction command.
Parameter #1—a serialized transaction to broadcast
Parameter #2–whether to allow high fees
Result—a TXID or error Message
*/
func (httpServer *HttpServer) handleSendRawTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleSendRawTransaction params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	base58CheckData, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("base58 check data is invalid"))
	}

	txMsg, txHash, LastBytePubKeySender, err := httpServer.txService.SendRawTransaction(base58CheckData)
	if err != nil {
		return nil, err
	}

	err2 := httpServer.config.Server.PushMessageToAll(txMsg)
	if err2 == nil {
		Logger.log.Infof("handleSendRawTransaction result: %+v, err: %+v", nil, err2)
		httpServer.config.TxMemPool.MarkForwardedTransaction(*txHash)
	}

	result := jsonresult.NewCreateTransactionResult(txHash, common.EmptyString, nil, common.GetShardIDFromLastByte(LastBytePubKeySender))
	Logger.log.Debugf("\n\n\n\n\n\nhandleSendRawTransaction result: %+v\n\n\n\n\n", result)
	return result, nil
}

/*
handleCreateAndSendTx - RPC creates transaction and send to network
*/
func (httpServer *HttpServer) handleCreateAndSendTx(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateAndSendTx params: %+v", params)
	var err error
	data, err := httpServer.handleCreateRawTransaction(params, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		Logger.log.Debugf("handleCreateAndSendTx result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		Logger.log.Debugf("handleCreateAndSendTx result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.SendTxDataError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, tx.ShardID)
	Logger.log.Debugf("handleCreateAndSendTx result: %+v", result)
	return result, nil
}

func (httpServer *HttpServer) handleGetTransactionHashByReceiver(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	paymentAddress, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payment address"))
	}

	result, err := httpServer.txService.GetTransactionHashByReceiver(paymentAddress)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	return result, nil
}

func (httpServer *HttpServer) handleGetTransactionByReceiver(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	paramsArray := common.InterfaceSlice(params)
	keys, ok := paramsArray[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("key param is invalid"))
	}
	// get keyset only contain readonly-key by deserializing
	readonlyKeyStr, ok := keys["ReadonlyKey"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("invalid readonly key"))
	}
	readonlyKey, err := wallet.Base58CheckDeserialize(readonlyKeyStr)
	if err != nil {
		Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	// get keyset only contain pub-key by deserializing
	paymentAddressStr, ok := keys["PaymentAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("invalid payment address"))
	}
	paymentAddress, err := wallet.Base58CheckDeserialize(paymentAddressStr)
	if err != nil {
		Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	// create a key set
	keySet := incognitokey.KeySet{
		ReadonlyKey:    readonlyKey.KeySet.ReadonlyKey,
		PaymentAddress: paymentAddress.KeySet.PaymentAddress,
	}

	result, err := httpServer.txService.GetTransactionByReceiver(keySet)

	return result, nil
}

// Get transaction by Hash
func (httpServer *HttpServer) handleGetTransactionByHash(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleGetTransactionByHash params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	// param #1: transaction Hash
	txHashStr, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Tx hash is invalid"))
	}
	Logger.log.Debugf("Get TransactionByHash input Param %+v", txHashStr)
	return httpServer.txService.GetTransactionByHash(txHashStr)
}

// handleCreateRawCustomTokenTransaction - handle create a custom token command and return in hex string format.
func (httpServer *HttpServer) handleCreateRawCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateRawCustomTokenTransaction params: %+v", params)
	var err error
	tx, err := httpServer.txService.BuildRawCustomTokenTransaction(params, nil, *httpServer.config.Database)
	if err.(*rpcservice.RPCError) != nil {
		Logger.log.Error(err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}

	byteArrays, err := json.Marshal(tx)
	if err != nil {
		Logger.log.Error(err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	result := jsonresult.CreateTransactionTokenResult{
		ShardID:         common.GetShardIDFromLastByte(tx.Tx.PubKeyLastByteSender),
		TxID:            tx.Hash().String(),
		TokenID:         tx.TxTokenData.PropertyID.String(),
		TokenName:       tx.TxTokenData.PropertyName,
		TokenAmount:     tx.TxTokenData.Amount,
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	Logger.log.Debugf("handleCreateRawCustomTokenTransaction result: %+v", result)
	return result, nil
}

// handleSendRawTransaction...
func (httpServer *HttpServer) handleSendRawCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleSendRawCustomTokenTransaction params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	base58CheckData, ok := arrayParams[0].(string)
	if !ok {
		Logger.log.Debugf("handleSendRawCustomTokenTransaction result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param is invalid"))
	}

	txMsg, tx, err := httpServer.txService.SendRawCustomTokenTransaction(base58CheckData)
	if err != nil {
		return nil, err
	}

	err2 := httpServer.config.Server.PushMessageToAll(txMsg)
	//Mark Fowarded transaction
	if err2 == nil {
		httpServer.config.TxMemPool.MarkForwardedTransaction(*tx.Hash())
	}
	result := jsonresult.CreateTransactionTokenResult{
		TxID:        tx.Hash().String(),
		TokenID:     tx.TxTokenData.PropertyID.String(),
		TokenName:   tx.TxTokenData.PropertyName,
		TokenAmount: tx.TxTokenData.Amount,
		ShardID:     common.GetShardIDFromLastByte(tx.Tx.PubKeyLastByteSender),
	}
	Logger.log.Debugf("handleSendRawCustomTokenTransaction result: %+v", result)
	return result, nil
}

// handleCreateAndSendCustomTokenTransaction - create and send a tx which process on a custom token look like erc-20 on eth
func (httpServer *HttpServer) handleCreateAndSendCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateAndSendCustomTokenTransaction params: %+v", params)
	data, err := httpServer.handleCreateRawCustomTokenTransaction(params, closeChan)
	if err != nil {
		Logger.log.Debugf("handleCreateAndSendCustomTokenTransaction result: %+v, err: %+v", nil, err)
		return nil, err
	}
	tx := data.(jsonresult.CreateTransactionTokenResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	txID, err := httpServer.handleSendRawCustomTokenTransaction(newParam, closeChan)
	if err != nil {
		Logger.log.Debugf("handleCreateAndSendCustomTokenTransaction result: %+v, err: %+v", nil, err)
		return nil, err
	}
	Logger.log.Debugf("handleCreateAndSendCustomTokenTransaction result: %+v", txID)
	return tx, nil
}

// handleGetListCustomTokenHolders - return all custom token holder
func (httpServer *HttpServer) handleGetListCustomTokenHolders(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	tokenIDStr, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("TokenID is invalid"))
	}

	return httpServer.txService.GetListCustomTokenHolders(tokenIDStr)
}

// handleGetListCustomTokenBalance - return list token + balance for one account payment address
func (httpServer *HttpServer) handleGetListCustomTokenBalance(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleGetListCustomTokenBalance params: %+v", params)
	result := jsonresult.ListCustomTokenBalance{ListCustomTokenBalance: []jsonresult.CustomTokenBalance{}}
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	accountParam, ok := arrayParams[0].(string)
	if len(accountParam) == 0 || !ok {
		Logger.log.Debugf("handleGetListCustomTokenBalance result: %+v", nil)
		return result, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param is invalid"))
	}

	result, err := httpServer.txService.GetListCustomTokenBalance(accountParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	return result, nil
}

// handleGetListPrivacyCustomTokenBalance - return list privacy token + balance for one account payment address
func (httpServer *HttpServer) handleGetListPrivacyCustomTokenBalance(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleGetListPrivacyCustomTokenBalance params: %+v", params)

	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	privateKey, ok := arrayParams[0].(string)
	if len(privateKey) == 0 || !ok {
		Logger.log.Debugf("handleGetListPrivacyCustomTokenBalance result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param is invalid"))
	}

	result, err := httpServer.txService.GetListPrivacyCustomTokenBalance(privateKey)
	if err != nil {
		return nil, err
	}
	Logger.log.Debugf("handleGetListPrivacyCustomTokenBalance result: %+v", result)
	return result, nil
}

// handleGetListPrivacyCustomTokenBalance - return list privacy token + balance for one account payment address
func (httpServer *HttpServer) handleGetBalancePrivacyCustomToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleGetBalancePrivacyCustomToken params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		Logger.log.Debugf("handleGetBalancePrivacyCustomToken error: Need 2 params but get %+v", len(arrayParams))
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 elements"))
	}

	privateKey, ok := arrayParams[0].(string)
	if len(privateKey) == 0 || !ok {
		Logger.log.Debugf("handleGetBalancePrivacyCustomToken result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("private key is invalid"))
	}

	tokenID, ok := arrayParams[1].(string)
	if len(tokenID) == 0 || !ok {
		Logger.log.Debugf("handleGetBalancePrivacyCustomToken result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("tokenID is invalid"))
	}

	totalValue, err2 := httpServer.txService.GetBalancePrivacyCustomToken(privateKey, tokenID)
	if err2 != nil {
		return nil, err2
	}

	Logger.log.Debugf("handleGetBalancePrivacyCustomToken result: %+v", totalValue)
	return totalValue, nil
}

// handleCustomTokenDetail - return list tx which relate to custom token by token id
func (httpServer *HttpServer) handleCustomTokenDetail(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCustomTokenDetail params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		Logger.log.Debugf("handleCustomTokenDetail result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("param must be an array at least 1 element"))
	}

	tokenIDTemp, ok := arrayParams[0].(string)
	if !ok {
		Logger.log.Debugf("handleCustomTokenDetail result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("tokenID is invalid"))
	}

	txs, err := httpServer.txService.CustomTokenDetail(tokenIDTemp)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	result := jsonresult.CustomToken{
		ListTxs: []string{},
	}
	for _, tx := range txs {
		result.ListTxs = append(result.ListTxs, tx.String())
	}
	Logger.log.Debugf("handleCustomTokenDetail result: %+v", result)
	return result, nil
}

// handlePrivacyCustomTokenDetail - return list tx which relate to privacy custom token by token id
func (httpServer *HttpServer) handlePrivacyCustomTokenDetail(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handlePrivacyCustomTokenDetail params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		Logger.log.Debugf("handlePrivacyCustomTokenDetail result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("param must be an array at least 1 element"))
	}

	tokenIDTemp, ok := arrayParams[0].(string)
	if !ok {
		Logger.log.Debugf("handlePrivacyCustomTokenDetail result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("tokenID is invalid"))
	}

	txs, _, err := httpServer.txService.PrivacyCustomTokenDetail(tokenIDTemp)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	result := jsonresult.CustomToken{
		ListTxs:            []string{},
		ID:                 tokenIDTemp,
		Name:               "",
		IsPrivacy:          true,
		Symbol:             "",
		InitiatorPublicKey: "",
	}

	for _, tx := range txs {
		result.ListTxs = append(result.ListTxs, tx.String())
	}

	Logger.log.Debugf("handlePrivacyCustomTokenDetail result: %+v", result)
	return result, nil
}

// handleListUnspentCustomToken - return list utxo of custom token
func (httpServer *HttpServer) handleListUnspentCustomToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleListUnspentCustomToken params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		Logger.log.Debugf("handleListUnspentCustomToken result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 element"))
	}
	// param #1: paymentaddress of sender
	senderKeyParam, ok := arrayParams[0].(string)
	if !ok {
		Logger.log.Debugf("handleListUnspentCustomToken result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("senderKey is invalid"))
	}

	// param #2: tokenID
	tokenIDParam, ok := arrayParams[1].(string)
	if !ok {
		Logger.log.Debugf("handleListUnspentCustomToken result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("tokenID is invalid"))
	}

	unspentTxTokenOuts, err := httpServer.txService.ListUnspentCustomToken(senderKeyParam, tokenIDParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	result := []jsonresult.UnspentCustomToken{}
	for _, temp := range unspentTxTokenOuts {
		item := jsonresult.UnspentCustomToken{
			PaymentAddress: senderKeyParam,
			Index:          temp.GetIndex(),
			TxHash:         temp.GetTxCustomTokenID().String(),
			Value:          temp.Value,
		}
		result = append(result, item)
	}

	Logger.log.Debugf("handleListUnspentCustomToken result: %+v", result)
	return result, nil
}

// handleListUnspentCustomToken - return list utxo of custom token
func (httpServer *HttpServer) handleGetBalanceCustomToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleGetBalanceCustomToken params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		Logger.log.Debugf("handleListUnspentCustomToken result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 element"))
	}
	// param #1: paymentaddress of sender
	senderKeyParam, ok := arrayParams[0].(string)
	if !ok {
		Logger.log.Debugf("handleGetBalanceCustomToken result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("senderKey is invalid"))
	}

	// param #2: tokenID
	tokenIDParam, ok := arrayParams[1].(string)
	if !ok {
		Logger.log.Debugf("handleGetBalanceCustomToken result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("tokenID is invalid"))
	}

	totalValue, err := httpServer.txService.GetBalanceCustomToken(senderKeyParam, tokenIDParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	Logger.log.Debugf("handleGetBalanceCustomToken result: %+v", totalValue)
	return totalValue, nil
}

// handleCreateSignatureOnCustomTokenTx - return a signature which is signed on raw custom token tx
func (httpServer *HttpServer) handleCreateSignatureOnCustomTokenTx(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateSignatureOnCustomTokenTx params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 element"))
	}

	base58CheckData, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("base58 check data is invalid"))
	}

	senderPrivateKeyParam, ok := arrayParams[1].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("private key is invalid"))
	}

	result, err := httpServer.txService.CreateSignatureOnCustomTokenTx(base58CheckData, senderPrivateKeyParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}

	Logger.log.Debugf("handleCreateSignatureOnCustomTokenTx result: %+v", result)
	return result, nil
}

// handleRandomCommitments - from input of outputcoin, random to create data for create new tx
func (httpServer *HttpServer) handleRandomCommitments(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleRandomCommitments params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 element"))
	}

	// #1: payment address
	paymentAddressStr, ok := arrayParams[0].(string)
	if !ok {
		Logger.log.Debugf("handleRandomCommitments result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("PaymentAddress is invalid"))
	}

	// #2: available inputCoin from old outputcoin
	outputs, ok := arrayParams[1].([]interface{})
	if !ok {
		Logger.log.Debugf("handleRandomCommitments result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("outputs is invalid"))
	}

	//#3 - tokenID - default PRV
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 2 {
		tokenIDTemp, ok := arrayParams[2].(string)
		if !ok {
			Logger.log.Debugf("handleRandomCommitments result: %+v", nil)
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("tokenID is invalid"))
		}
		tokenID, err = common.Hash{}.NewHashFromStr(tokenIDTemp)
		if err != nil {
			Logger.log.Debugf("handleRandomCommitments result: %+v, err: %+v", nil, err)
			return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
		}
	}

	commitmentIndexs, myCommitmentIndexs, commitments, err2 := httpServer.txService.RandomCommitments(paymentAddressStr, outputs, tokenID)
	if err2 != nil {
		return nil, err2
	}

	result := jsonresult.NewRandomCommitmentResult(commitmentIndexs, myCommitmentIndexs, commitments)
	Logger.log.Debugf("handleRandomCommitments result: %+v", result)
	return result, nil
}

// handleListSerialNumbers - return list all serialnumber in shard for token ID
func (httpServer *HttpServer) handleListSerialNumbers(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	var err error
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 0 {
		tokenIDTemp, ok := arrayParams[0].(string)
		if !ok {
			Logger.log.Debugf("handleHasSerialNumbers result: %+v", nil)
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		if len(tokenIDTemp) > 0 {
			tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
			if err != nil {
				Logger.log.Debugf("handleHasSerialNumbers result: %+v, err: %+v", err)
				return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
			}
		}
	}
	shardID := 0
	if len(arrayParams) > 1 {
		shardIDParam, ok := arrayParams[1].(float64)
		if ok {
			shardID = int(shardIDParam)
		}
	}

	result, err := httpServer.databaseService.ListSerialNumbers(*tokenID, byte(shardID))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
	}
	return result, nil
}

// handleListSerialNumbers - return list all serialnumber in shard for token ID
func (httpServer *HttpServer) handleListSNDerivator(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	var err error
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 0 {
		tokenIDTemp, ok := arrayParams[0].(string)
		if !ok {
			Logger.log.Debugf("handleListSNDerivator result: %+v", nil)
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		if len(tokenIDTemp) > 0 {
			tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
			if err != nil {
				Logger.log.Debugf("handleListSNDerivator result: %+v, err: %+v", err)
				return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
			}
		}
	}

	result, err := httpServer.databaseService.ListSNDerivator(*tokenID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
	}
	return result, nil
}

// handleListCommitments - return list all commitments in shard for token ID
func (httpServer *HttpServer) handleListCommitments(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	var err error
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 0 {
		tokenIDTemp, ok := arrayParams[0].(string)
		if !ok {
			Logger.log.Debugf("handleHasSerialNumbers result: %+v", nil)
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		if len(tokenIDTemp) > 0 {
			tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
			if err != nil {
				Logger.log.Debugf("handleHasSerialNumbers result: %+v, err: %+v", err)
				return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
			}
		}
	}
	shardID := 0
	if len(arrayParams) > 1 {
		shardIDParam, ok := arrayParams[1].(float64)
		if ok {
			shardID = int(shardIDParam)
		}
	}

	result, err := httpServer.databaseService.ListCommitments(*tokenID, byte(shardID))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
	}
	return result, nil
}

// handleListCommitmentIndices - return list all commitment indices in shard for token ID
func (httpServer *HttpServer) handleListCommitmentIndices(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	var err error
	tokenID := &common.Hash{}
	err = tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 0 {
		tokenIDTemp, ok := arrayParams[0].(string)
		if !ok {
			Logger.log.Debugf("handleHasSerialNumbers result: %+v", nil)
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		if len(tokenIDTemp) > 0 {
			tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
			if err != nil {
				Logger.log.Debugf("handleHasSerialNumbers result: %+v, err: %+v", err)
				return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
			}
		}
	}
	shardID := byte(0)
	if len(arrayParams) > 1 {
		shardIDParam, ok := arrayParams[1].(float64)
		if ok {
			shardID = byte(shardIDParam)
		}
	}

	result, err := httpServer.databaseService.ListCommitmentIndices(*tokenID, shardID)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
	}
	return result, nil
}

// handleHasSerialNumbers - check list serial numbers existed in db of node
func (httpServer *HttpServer) handleHasSerialNumbers(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleHasSerialNumbers params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 elements"))
	}

	// #1: payment address
	paymentAddressStr, ok := arrayParams[0].(string)
	if !ok {
		Logger.log.Debugf("handleHasSerialNumbers result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("PaymentAddress is invalid"))
	}

	//#2: list serialnumbers in base58check encode string
	serialNumbersStr, ok := arrayParams[1].([]interface{})
	if !ok {
		Logger.log.Debugf("handleHasSerialNumbers result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
	}

	// #3: optional - token ID - default is prv coin
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 2 {
		tokenIDTemp, ok := arrayParams[2].(string)
		if !ok {
			Logger.log.Debugf("handleHasSerialNumbers result: %+v", nil)
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("serialNumbers is invalid"))
		}
		tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
		if err != nil {
			Logger.log.Debugf("handleHasSerialNumbers result: %+v, err: %+v", err)
			return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
		}
	}

	result, err := httpServer.databaseService.HasSerialNumbers(paymentAddressStr, serialNumbersStr, *tokenID)
	if err != nil {
		Logger.log.Debugf("handleHasSerialNumbers result: %+v, err: %+v", err)
		return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err)
	}

	Logger.log.Debugf("handleHasSerialNumbers result: %+v", result)
	return result, nil
}

// handleHasSerialNumbers - check list serial numbers existed in db of node
func (httpServer *HttpServer) handleHasSnDerivators(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleHasSnDerivators params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 2 elements"))
	}

	// #1: payment address
	paymentAddressStr, ok := arrayParams[0].(string)
	if !ok {
		Logger.log.Debugf("handleHasSnDerivators result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("paymentAddress is invalid"))
	}

	//#2: list serialnumbers in base58check encode string
	snDerivatorStr, ok := arrayParams[1].([]interface{})
	if !ok {
		Logger.log.Debugf("handleHasSnDerivators result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("snDerivatorStr is invalid"))
	}

	// #3: optional - token ID - default is prv coin
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:]) // default is PRV coin
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(arrayParams) > 2 {
		tokenIDTemp, ok := arrayParams[1].(string)
		if !ok {
			Logger.log.Debugf("handleHasSnDerivators result: %+v", nil)
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("tokenID is invalid"))
		}
		tokenID, err = (common.Hash{}).NewHashFromStr(tokenIDTemp)
		if err != nil {
			Logger.log.Debugf("handleHasSnDerivators result: %+v, err: %+v", nil, err)
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		}
	}
	result, err := httpServer.databaseService.HasSnDerivators(paymentAddressStr, snDerivatorStr, *tokenID)
	if err != nil {
		Logger.log.Debugf("handleHasSnDerivators result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	Logger.log.Debugf("handleHasSnDerivators result: %+v", result)
	return result, nil
}

// handleCreateRawCustomTokenTransaction - handle create a custom token command and return in hex string format.
func (httpServer *HttpServer) handleCreateRawPrivacyCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateRawPrivacyCustomTokenTransaction params: %+v", params)
	var err error
	tx, err := httpServer.txService.BuildRawPrivacyCustomTokenTransaction(params, nil, *httpServer.config.Database)
	if err.(*rpcservice.RPCError) != nil {
		Logger.log.Error(err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}

	byteArrays, err := json.Marshal(tx)
	if err != nil {
		Logger.log.Error(err)
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	result := jsonresult.CreateTransactionTokenResult{
		ShardID:         common.GetShardIDFromLastByte(tx.Tx.PubKeyLastByteSender),
		TxID:            tx.Hash().String(),
		TokenID:         tx.TxPrivacyTokenData.PropertyID.String(),
		TokenName:       tx.TxPrivacyTokenData.PropertyName,
		TokenAmount:     tx.TxPrivacyTokenData.Amount,
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	Logger.log.Debugf("handleCreateRawPrivacyCustomTokenTransaction result: %+v", result)
	return result, nil
}

// handleSendRawTransaction...
func (httpServer *HttpServer) handleSendRawPrivacyCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleSendRawPrivacyCustomTokenTransaction params: %+v", params)
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 1 {
		Logger.log.Debugf("handleSendRawPrivacyCustomTokenTransaction result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 1 element"))
	}

	base58CheckData, ok := arrayParams[0].(string)
	if !ok {
		Logger.log.Debugf("handleSendRawPrivacyCustomTokenTransaction result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Param is invalid"))
	}

	txMsg, tx, err := httpServer.txService.SendRawPrivacyCustomTokenTransaction(base58CheckData)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	err = httpServer.config.Server.PushMessageToAll(txMsg)
	//Mark forwarded message
	if err == nil {
		httpServer.config.TxMemPool.MarkForwardedTransaction(*tx.Hash())
	}
	result := jsonresult.CreateTransactionTokenResult{
		TxID:        tx.Hash().String(),
		TokenID:     tx.TxPrivacyTokenData.PropertyID.String(),
		TokenName:   tx.TxPrivacyTokenData.PropertyName,
		TokenAmount: tx.TxPrivacyTokenData.Amount,
		ShardID:     common.GetShardIDFromLastByte(tx.Tx.PubKeyLastByteSender),
	}
	Logger.log.Debugf("handleSendRawPrivacyCustomTokenTransaction result: %+v", result)
	return result, nil
}

// handleCreateAndSendCustomTokenTransaction - create and send a tx which process on a custom token look like erc-20 on eth
func (httpServer *HttpServer) handleCreateAndSendPrivacyCustomTokenTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateAndSendPrivacyCustomTokenTransaction params: %+v", params)
	data, err := httpServer.handleCreateRawPrivacyCustomTokenTransaction(params, closeChan)
	if err != nil {
		return nil, err
	}
	tx := data.(jsonresult.CreateTransactionTokenResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	txId, err := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err != nil {
		Logger.log.Errorf("handleCreateAndSendPrivacyCustomTokenTransaction result: %+v, err: %+v", nil, err)
		return nil, err
	}
	Logger.log.Debugf("handleCreateAndSendPrivacyCustomTokenTransaction result: %+v", txId)
	return tx, nil
}

/*
// handleCreateRawStakingTransaction handles create staking
*/
func (httpServer *HttpServer) handleCreateRawStakingTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	// get component
	Logger.log.Debugf("handleCreateRawStakingTransaction params: %+v", params)

	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 5 {
		Logger.log.Debugf("handleCreateRawStakingTransaction result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 5 element"))
	}

	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	keyWallet := new(wallet.KeyWallet)
	keyWallet.KeySet = *createRawTxParam.SenderKeySet
	funderPaymentAddress := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	Logger.log.Info("Staking Public Key: %v\n", funderPaymentAddress)

	// prepare meta data
	data, ok := paramsArray[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Data For Staking Transaction %+v", paramsArray[4]))
	}

	stakingType, ok := data["StakingType"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Staking Type For Staking Transaction %+v", data["StakingType"]))
	}

	candidatePaymentAddress, ok := data["CandidatePaymentAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Producer Payment Address for Staking Transaction %+v", data["CandidatePaymentAddress"]))
	}

	// Get private seed, a.k.a mining key
	privateSeed, ok := data["PrivateSeed"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Private Seed For Staking Transaction %+v", data["PrivateSeed"]))
	}
	privateSeedBytes, ver, errDecode := base58.Base58Check{}.Decode(privateSeed)
	if (errDecode != nil) || (ver != common.ZeroByte) {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("Decode privateseed failed!"))
	}

	//Get RewardReceiver Payment Address
	rewardReceiverPaymentAddress, ok := data["RewardReceiverPaymentAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Reward Receiver Payment Address For Staking Transaction %+v", data["RewardReceiverPaymentAddress"]))
	}

	//Get auto staking flag
	autoReStaking, ok := data["AutoReStaking"].(bool)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid auto restaking flag %+v", data["AutoReStaking"]))
	}

	// Get candidate publickey
	candidateWallet, err := wallet.Base58CheckDeserialize(candidatePaymentAddress)
	if err != nil || candidateWallet == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Base58CheckDeserialize candidate Payment Address failed"))
	}
	pk := candidateWallet.KeySet.PaymentAddress.Pk

	committeePK, err := incognitokey.NewCommitteeKeyFromSeed(privateSeedBytes, pk)
	if err != nil {
		Logger.log.Critical(err)
		Logger.log.Debugf("handleCreateRawStakingTransaction result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Cannot get payment address"))
	}

	committeePKBytes, err := committeePK.Bytes()
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Cannot import key set"))
	}

	stakingMetadata, err := metadata.NewStakingMetadata(
		int(stakingType), funderPaymentAddress, rewardReceiverPaymentAddress,
		httpServer.config.ChainParams.StakingAmountShard,
		base58.Base58Check{}.Encode(committeePKBytes, common.ZeroByte), autoReStaking)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	txID, txBytes, txShardID, err := httpServer.txService.CreateRawTransaction(createRawTxParam, stakingMetadata, *httpServer.config.Database)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}

	result := jsonresult.CreateTransactionResult{
		TxID:            txID.String(),
		Base58CheckData: base58.Base58Check{}.Encode(txBytes, common.ZeroByte),
		ShardID:         txShardID,
	}
	Logger.log.Debugf("handleCreateRawStakingTransaction result: %+v", result)
	return result, nil
}

/*
handleCreateAndSendStakingTx - RPC creates staking transaction and send to network
*/
func (httpServer *HttpServer) handleCreateAndSendStakingTx(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Infof("handleCreateAndSendStakingTx params: %+v", params)

	var err error
	data, err := httpServer.handleCreateRawStakingTransaction(params, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	Logger.log.Infof("handleCreateAndSendStakingTx create success tx=%+v", tx.TxID)
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		Logger.log.Debugf("handleCreateAndSendStakingTx result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.SendTxDataError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, tx.ShardID)
	Logger.log.Infof("handleCreateAndSendStakingTx result: %+v", result)
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawStopAutoStakingTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	// get component
	Logger.log.Debugf("handleCreateRawStopAutoStakingTransaction params: %+v", params)
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 5 {
		Logger.log.Debugf("handleCreateRawStopAutoStakingTransaction result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 5 element"))
	}

	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	keyWallet := new(wallet.KeyWallet)
	keyWallet.KeySet = *createRawTxParam.SenderKeySet
	funderPaymentAddress := keyWallet.Base58CheckSerialize(wallet.PaymentAddressType)
	Logger.log.Info("Staking Public Key: %v\n", funderPaymentAddress)

	//Get data to create meta data
	data, ok := paramsArray[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Staking Type For Staking Transaction %+v", paramsArray[4]))
	}

	//Get staking type
	stopAutoStakingType, ok := data["StopAutoStakingType"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Staking Type For Staking Transaction %+v", data["StopAutoStakingType"]))
	}

	//Get Candidate Payment Address
	candidatePaymentAddress, ok := data["CandidatePaymentAddress"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Producer Payment Address for Staking Transaction %+v", data["CandidatePaymentAddress"]))
	}
	// Get private seed, a.k.a mining key
	privateSeed, ok := data["PrivateSeed"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, fmt.Errorf("Invalid Private Seed for Staking Transaction %+v", data["PrivateSeed"]))
	}
	privateSeedBytes, ver, err := base58.Base58Check{}.Decode(privateSeed)
	if (err != nil) || (ver != common.ZeroByte) {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("Decode privateseed failed!"))
	}

	// Get candidate publickey
	candidateWallet, err := wallet.Base58CheckDeserialize(candidatePaymentAddress)
	if err != nil || candidateWallet == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Base58CheckDeserialize candidate Payment Address failed"))
	}
	pk := candidateWallet.KeySet.PaymentAddress.Pk

	committeePK, err := incognitokey.NewCommitteeKeyFromSeed(privateSeedBytes, pk)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	committeePKBytes, err := committeePK.Bytes()
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}

	stakingMetadata, err := metadata.NewStopAutoStakingMetadata(int(stopAutoStakingType), base58.Base58Check{}.Encode(committeePKBytes, common.ZeroByte))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	txID, txBytes, txShardID, err := httpServer.txService.CreateRawTransaction(createRawTxParam, stakingMetadata, *httpServer.config.Database)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}

	result := jsonresult.CreateTransactionResult{
		TxID:            txID.String(),
		Base58CheckData: base58.Base58Check{}.Encode(txBytes, common.ZeroByte),
		ShardID:         txShardID,
	}
	Logger.log.Debugf("handleCreateRawStakingTransaction result: %+v", result)
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendStopAutoStakingTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleCreateAndSendStopAutoStakingTransaction params: %+v", params)
	var err error
	data, err := httpServer.handleCreateRawStopAutoStakingTransaction(params, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		return nil, rpcservice.NewRPCError(rpcservice.CreateTxDataError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData

	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err.(*rpcservice.RPCError) != nil {
		Logger.log.Debugf("handleCreateAndSendStakingTx result: %+v, err: %+v", nil, err)
		return nil, rpcservice.NewRPCError(rpcservice.SendTxDataError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, tx.ShardID)
	Logger.log.Debugf("handleCreateAndSendStakingTx result: %+v", result)
	return result, nil
}
