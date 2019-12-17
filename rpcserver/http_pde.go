package rpcserver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/common/base58"
	"github.com/incognitochain/incognito-chain/database/lvdb"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/rpcserver/bean"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

type PDEWithdrawal struct {
	WithdrawalTokenIDStr string
	WithdrawerAddressStr string
	DeductingPoolValue   uint64
	DeductingShares      uint64
	PairToken1IDStr      string
	PairToken2IDStr      string
	TxReqID              common.Hash
	ShardID              byte
	Status               string
	BeaconHeight         uint64
}

type PDETrade struct {
	TraderAddressStr    string
	ReceivingTokenIDStr string
	ReceiveAmount       uint64
	Token1IDStr         string
	Token2IDStr         string
	ShardID             byte
	RequestedTxID       common.Hash
	Status              string
	BeaconHeight        uint64
}

type PDEContribution struct {
	PDEContributionPairID string
	ContributorAddressStr string
	ContributedAmount     uint64
	TokenIDStr            string
	TxReqID               common.Hash
	ShardID               byte
	Status                string
	BeaconHeight          uint64
}

type PDEInfoFromBeaconBlock struct {
	PDEContributions []*PDEContribution
	PDETrades        []*PDETrade
	PDEWithdrawals   []*PDEWithdrawal
}

type ConvertedPrice struct {
	FromTokenIDStr string
	ToTokenIDStr   string
	Amount         uint64
	Price          uint64
}

func (httpServer *HttpServer) handleCreateRawTxWithPRVContribution(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	pdeContributionPairID, ok := data["PDEContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributorAddressStr, ok := data["ContributorAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmountData, ok := data["ContributedAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmount := uint64(contributedAmountData)
	tokenIDStr, ok := data["TokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	meta, _ := metadata.NewPDEContribution(
		pdeContributionPairID,
		contributorAddressStr,
		contributedAmount,
		tokenIDStr,
		metadata.PDEContributionMeta,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta, *httpServer.config.Database)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPRVContribution(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPRVContribution(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPTokenContribution(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) >= 7 {
		hasPrivacyToken := int(arrayParams[6].(float64)) > 0
		if hasPrivacyToken {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("The privacy mode must be disabled"))
		}
	}
	tokenParamsRaw := arrayParams[4].(map[string]interface{})

	pdeContributionPairID, ok := tokenParamsRaw["PDEContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributorAddressStr, ok := tokenParamsRaw["ContributorAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmountData, ok := tokenParamsRaw["ContributedAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	contributedAmount := uint64(contributedAmountData)
	tokenIDStr := tokenParamsRaw["TokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	meta, _ := metadata.NewPDEContribution(
		pdeContributionPairID,
		contributorAddressStr,
		contributedAmount,
		tokenIDStr,
		metadata.PDEContributionMeta,
	)

	customTokenTx, rpcErr := httpServer.txService.BuildRawPrivacyCustomTokenTransaction(params, meta, *httpServer.config.Database)
	if rpcErr != nil {
		Logger.log.Error(rpcErr)
		return nil, rpcErr
	}

	byteArrays, err2 := json.Marshal(customTokenTx)
	if err2 != nil {
		Logger.log.Error(err2)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            customTokenTx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPTokenContribution(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPTokenContribution(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	// sendResult, err1 := httpServer.handleSendRawCustomTokenTransaction(newParam, closeChan)
	sendResult, err1 := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	return sendResult, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPRVTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tokenIDToBuyStr, ok := data["TokenIDToBuyStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tokenIDToSellStr, ok := data["TokenIDToSellStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	sellAmount := uint64(data["SellAmount"].(float64))
	traderAddressStr, ok := data["TraderAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	minAcceptableAmountData, ok := data["MinAcceptableAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	minAcceptableAmount := uint64(minAcceptableAmountData)
	tradingFeeData, ok := data["TradingFee"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tradingFee := uint64(tradingFeeData)
	meta, _ := metadata.NewPDETradeRequest(
		tokenIDToBuyStr,
		tokenIDToSellStr,
		sellAmount,
		minAcceptableAmount,
		tradingFee,
		traderAddressStr,
		metadata.PDETradeRequestMeta,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta, *httpServer.config.Database)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPRVTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPRVTradeReq(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithPTokenTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	if len(arrayParams) >= 7 {
		hasPrivacyToken := int(arrayParams[6].(float64)) > 0
		if hasPrivacyToken {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, errors.New("The privacy mode must be disabled"))
		}
	}
	tokenParamsRaw := arrayParams[4].(map[string]interface{})

	tokenIDToBuyStr, ok := tokenParamsRaw["TokenIDToBuyStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	tokenIDToSellStr, ok := tokenParamsRaw["TokenIDToSellStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	sellAmountData, ok := tokenParamsRaw["SellAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	sellAmount := uint64(sellAmountData)

	traderAddressStr, ok := tokenParamsRaw["TraderAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	minAcceptableAmountData, ok := tokenParamsRaw["MinAcceptableAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	minAcceptableAmount := uint64(minAcceptableAmountData)

	tradingFeeData, ok := tokenParamsRaw["TradingFee"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	tradingFee := uint64(tradingFeeData)

	meta, _ := metadata.NewPDETradeRequest(
		tokenIDToBuyStr,
		tokenIDToSellStr,
		sellAmount,
		minAcceptableAmount,
		tradingFee,
		traderAddressStr,
		metadata.PDETradeRequestMeta,
	)

	customTokenTx, rpcErr := httpServer.txService.BuildRawPrivacyCustomTokenTransaction(params, meta, *httpServer.config.Database)
	if rpcErr != nil {
		Logger.log.Error(rpcErr)
		return nil, rpcErr
	}

	byteArrays, err2 := json.Marshal(customTokenTx)
	if err2 != nil {
		Logger.log.Error(err2)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            customTokenTx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithPTokenTradeReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithPTokenTradeReq(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	// sendResult, err1 := httpServer.handleSendRawCustomTokenTransaction(newParam, closeChan)
	sendResult, err1 := httpServer.handleSendRawPrivacyCustomTokenTransaction(newParam, closeChan)
	if err1 != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	return sendResult, nil
}

func (httpServer *HttpServer) handleCreateRawTxWithWithdrawalReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)

	// get meta data from params
	data, ok := arrayParams[4].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawerAddressStr, ok := data["WithdrawerAddressStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalToken1IDStr, ok := data["WithdrawalToken1IDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalToken2IDStr, ok := data["WithdrawalToken2IDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}

	withdrawalShareAmtData, ok := data["WithdrawalShareAmt"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("metadata is invalid"))
	}
	withdrawalShareAmt := uint64(withdrawalShareAmtData)

	meta, _ := metadata.NewPDEWithdrawalRequest(
		withdrawerAddressStr,
		withdrawalToken1IDStr,
		withdrawalToken2IDStr,
		withdrawalShareAmt,
		metadata.PDEWithdrawalRequestMeta,
	)

	// create new param to build raw tx from param interface
	createRawTxParam, errNewParam := bean.NewCreateRawTxParam(params)
	if errNewParam != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errNewParam)
	}

	tx, err1 := httpServer.txService.BuildRawTransaction(createRawTxParam, meta, *httpServer.config.Database)
	if err1 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err1)
	}

	byteArrays, err2 := json.Marshal(tx)
	if err2 != nil {
		Logger.log.Error(err1)
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err2)
	}
	result := jsonresult.CreateTransactionResult{
		TxID:            tx.Hash().String(),
		Base58CheckData: base58.Base58Check{}.Encode(byteArrays, 0x00),
	}
	return result, nil
}

func (httpServer *HttpServer) handleCreateAndSendTxWithWithdrawalReq(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	data, err := httpServer.handleCreateRawTxWithWithdrawalReq(params, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	tx := data.(jsonresult.CreateTransactionResult)
	base58CheckData := tx.Base58CheckData
	newParam := make([]interface{}, 0)
	newParam = append(newParam, base58CheckData)
	sendResult, err := httpServer.handleSendRawTransaction(newParam, closeChan)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}
	result := jsonresult.NewCreateTransactionResult(nil, sendResult.(jsonresult.CreateTransactionResult).TxID, nil, sendResult.(jsonresult.CreateTransactionResult).ShardID)
	return result, nil
}

func (httpServer *HttpServer) handleGetPDEState(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	beaconHeight, ok := data["BeaconHeight"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Beacon height is invalid"))
	}
	pdeState, err := blockchain.InitCurrentPDEStateFromDB(httpServer.config.BlockChain.GetDatabase(), uint64(beaconHeight))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return pdeState, nil
}

func (httpServer *HttpServer) handleConvertNativeTokenToPrivacyToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	beaconHeight, ok := data["BeaconHeight"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	nativeTokenAmount, ok := data["NativeTokenAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	tokenIDStr, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	res, err := metadata.ConvertNativeTokenToPrivacyToken(
		uint64(nativeTokenAmount),
		tokenID,
		int64(beaconHeight),
		httpServer.config.BlockChain.GetDatabase(),
	)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return res, nil
}

func (httpServer *HttpServer) handleConvertPrivacyTokenToNativeToken(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	beaconHeight, ok := data["BeaconHeight"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	privacyTokenAmount, ok := data["PrivacyTokenAmount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	tokenIDStr, ok := data["TokenID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	tokenID, err := common.Hash{}.NewHashFromStr(tokenIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	res, err := metadata.ConvertPrivacyTokenToNativeToken(
		uint64(privacyTokenAmount),
		tokenID,
		int64(beaconHeight),
		httpServer.config.BlockChain.GetDatabase(),
	)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return res, nil
}

func (httpServer *HttpServer) handleGetPDEContributionStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	contributionPairID, ok := data["ContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	status, err := httpServer.databaseService.GetPDEStatus(lvdb.PDEContributionStatusPrefix, []byte(contributionPairID))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetPDEContributionStatusV2(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	contributionPairID, ok := data["ContributionPairID"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	contributionStatus, err := httpServer.databaseService.GetPDEContributionStatus(lvdb.PDEContributionStatusPrefix, []byte(contributionPairID))
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return contributionStatus, nil
}

func (httpServer *HttpServer) handleGetPDETradeStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	txRequestIDStr, ok := data["TxRequestIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	txIDHash, err := common.Hash{}.NewHashFromStr(txRequestIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	status, err := httpServer.databaseService.GetPDEStatus(lvdb.PDETradeStatusPrefix, txIDHash[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return status, nil
}

func (httpServer *HttpServer) handleGetPDEWithdrawalStatus(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	arrayParams := common.InterfaceSlice(params)
	data := arrayParams[0].(map[string]interface{})
	txRequestIDStr, ok := data["TxRequestIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload is invalid"))
	}
	txIDHash, err := common.Hash{}.NewHashFromStr(txRequestIDStr)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	status, err := httpServer.databaseService.GetPDEStatus(lvdb.PDEWithdrawalStatusPrefix, txIDHash[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	return status, nil
}

func parsePDEContributionInst(inst []string, beaconHeight uint64) (*PDEContribution, error) {
	status := inst[2]
	shardID, err := strconv.Atoi(inst[1])
	if err != nil {
		return nil, err
	}
	if status == common.PDEContributionMatchedChainStatus {
		matchedContribContent := []byte(inst[3])
		var matchedContrib metadata.PDEMatchedContribution
		err := json.Unmarshal(matchedContribContent, &matchedContrib)
		if err != nil {
			return nil, err
		}
		return &PDEContribution{
			PDEContributionPairID: matchedContrib.PDEContributionPairID,
			ContributorAddressStr: matchedContrib.ContributorAddressStr,
			ContributedAmount:     matchedContrib.ContributedAmount,
			TokenIDStr:            matchedContrib.TokenIDStr,
			TxReqID:               matchedContrib.TxReqID,
			ShardID:               byte(shardID),
			Status:                common.PDEContributionMatchedChainStatus,
			BeaconHeight:          beaconHeight,
		}, nil
	}
	if status == common.PDEContributionRefundChainStatus {
		refundedContribContent := []byte(inst[3])
		var refundedContrib metadata.PDERefundContribution
		err := json.Unmarshal(refundedContribContent, &refundedContrib)
		if err != nil {
			return nil, err
		}
		return &PDEContribution{
			PDEContributionPairID: refundedContrib.PDEContributionPairID,
			ContributorAddressStr: refundedContrib.ContributorAddressStr,
			ContributedAmount:     refundedContrib.ContributedAmount,
			TokenIDStr:            refundedContrib.TokenIDStr,
			TxReqID:               refundedContrib.TxReqID,
			ShardID:               byte(shardID),
			Status:                common.PDEContributionRefundChainStatus,
			BeaconHeight:          beaconHeight,
		}, nil
	}
	return nil, nil
}

func parsePDETradeInst(inst []string, beaconHeight uint64) (*PDETrade, error) {
	status := inst[2]
	shardID, err := strconv.Atoi(inst[1])
	if err != nil {
		return nil, err
	}
	if status == common.PDETradeRefundChainStatus {
		contentBytes, err := base64.StdEncoding.DecodeString(inst[3])
		if err != nil {
			return nil, err
		}
		var pdeTradeReqAction metadata.PDETradeRequestAction
		err = json.Unmarshal(contentBytes, &pdeTradeReqAction)
		if err != nil {
			return nil, err
		}
		tokenIDStrs := []string{pdeTradeReqAction.Meta.TokenIDToBuyStr, pdeTradeReqAction.Meta.TokenIDToSellStr}
		sort.Slice(tokenIDStrs, func(i, j int) bool {
			return tokenIDStrs[i] < tokenIDStrs[j]
		})
		return &PDETrade{
			TraderAddressStr:    pdeTradeReqAction.Meta.TraderAddressStr,
			ReceivingTokenIDStr: pdeTradeReqAction.Meta.TokenIDToSellStr,
			ReceiveAmount:       pdeTradeReqAction.Meta.SellAmount + pdeTradeReqAction.Meta.TradingFee,
			Token1IDStr:         tokenIDStrs[0],
			Token2IDStr:         tokenIDStrs[1],
			ShardID:             byte(shardID),
			RequestedTxID:       pdeTradeReqAction.TxReqID,
			Status:              "refunded",
			BeaconHeight:        beaconHeight,
		}, nil
	}
	if status == common.PDETradeAcceptedChainStatus {
		tradeAcceptedContentBytes := []byte(inst[3])
		var tradeAcceptedContent metadata.PDETradeAcceptedContent
		err := json.Unmarshal(tradeAcceptedContentBytes, &tradeAcceptedContent)
		if err != nil {
			return nil, err
		}
		tokenIDStrs := []string{tradeAcceptedContent.Token1IDStr, tradeAcceptedContent.Token2IDStr}
		sort.Slice(tokenIDStrs, func(i, j int) bool {
			return tokenIDStrs[i] < tokenIDStrs[j]
		})
		return &PDETrade{
			TraderAddressStr:    tradeAcceptedContent.TraderAddressStr,
			ReceivingTokenIDStr: tradeAcceptedContent.TokenIDToBuyStr,
			ReceiveAmount:       tradeAcceptedContent.ReceiveAmount,
			Token1IDStr:         tokenIDStrs[0],
			Token2IDStr:         tokenIDStrs[1],
			ShardID:             byte(shardID),
			RequestedTxID:       tradeAcceptedContent.RequestedTxID,
			Status:              "accepted",
			BeaconHeight:        beaconHeight,
		}, nil
	}
	return nil, nil
}

func parsePDEWithdrawalInst(inst []string, beaconHeight uint64) (*PDEWithdrawal, error) {
	status := inst[2]
	shardID, err := strconv.Atoi(inst[1])
	if err != nil {
		return nil, err
	}
	if status == common.PDEWithdrawalAcceptedChainStatus {
		withdrawalAcceptedContentBytes := []byte(inst[3])
		var withdrawalAcceptedContent metadata.PDEWithdrawalAcceptedContent
		err := json.Unmarshal(withdrawalAcceptedContentBytes, &withdrawalAcceptedContent)
		if err != nil {
			return nil, err
		}
		tokenIDStrs := []string{withdrawalAcceptedContent.PairToken1IDStr, withdrawalAcceptedContent.PairToken2IDStr}
		sort.Slice(tokenIDStrs, func(i, j int) bool {
			return tokenIDStrs[i] < tokenIDStrs[j]
		})
		return &PDEWithdrawal{
			WithdrawalTokenIDStr: withdrawalAcceptedContent.WithdrawalTokenIDStr,
			WithdrawerAddressStr: withdrawalAcceptedContent.WithdrawerAddressStr,
			DeductingPoolValue:   withdrawalAcceptedContent.DeductingPoolValue,
			DeductingShares:      withdrawalAcceptedContent.DeductingShares,
			PairToken1IDStr:      tokenIDStrs[0],
			PairToken2IDStr:      tokenIDStrs[1],
			TxReqID:              withdrawalAcceptedContent.TxReqID,
			ShardID:              byte(shardID),
			Status:               "accepted",
			BeaconHeight:         beaconHeight,
		}, nil
	}
	return nil, nil
}

func (httpServer *HttpServer) handleExtractPDEInstsFromBeaconBlock(
	params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError,
) {
	arrayParams := common.InterfaceSlice(params)
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	beaconHeight, ok := data["BeaconHeight"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Beacon height is invalid"))
	}

	bcHeight := uint64(beaconHeight)
	beaconBlocks, err := blockchain.FetchBeaconBlockFromHeight(
		httpServer.config.BlockChain.GetDatabase(),
		bcHeight,
		bcHeight,
	)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, err)
	}
	if len(beaconBlocks) == 0 {
		return nil, nil
	}
	bcBlk := beaconBlocks[0]
	pdeInfoFromBeaconBlock := PDEInfoFromBeaconBlock{
		PDEContributions: []*PDEContribution{},
		PDETrades:        []*PDETrade{},
		PDEWithdrawals:   []*PDEWithdrawal{},
	}
	insts := bcBlk.Body.Instructions
	for _, inst := range insts {
		if len(inst) < 2 {
			continue // Not error, just not PDE instruction
		}
		switch inst[0] {
		case strconv.Itoa(metadata.PDEContributionMeta):
			pdeContrib, err := parsePDEContributionInst(inst, bcHeight)
			if err != nil || pdeContrib == nil {
				continue
			}
			pdeInfoFromBeaconBlock.PDEContributions = append(pdeInfoFromBeaconBlock.PDEContributions, pdeContrib)
		case strconv.Itoa(metadata.PDETradeRequestMeta):
			pdeTrade, err := parsePDETradeInst(inst, bcHeight)
			if err != nil || pdeTrade == nil {
				continue
			}
			pdeInfoFromBeaconBlock.PDETrades = append(pdeInfoFromBeaconBlock.PDETrades, pdeTrade)
		case strconv.Itoa(metadata.PDEWithdrawalRequestMeta):
			pdeWithdrawal, err := parsePDEWithdrawalInst(inst, bcHeight)
			if err != nil || pdeWithdrawal == nil {
				continue
			}
			pdeInfoFromBeaconBlock.PDEWithdrawals = append(pdeInfoFromBeaconBlock.PDEWithdrawals, pdeWithdrawal)
		}
	}
	return pdeInfoFromBeaconBlock, nil
}

func convertPrice(
	latestBcHeight uint64,
	toTokenIDStr string,
	fromTokenIDStr string,
	convertingAmt uint64,
	pdePoolPairs map[string]*lvdb.PDEPoolForPair,
) *ConvertedPrice {
	poolPairKey := lvdb.BuildPDEPoolForPairKey(
		latestBcHeight,
		toTokenIDStr,
		fromTokenIDStr,
	)
	poolPair, found := pdePoolPairs[string(poolPairKey)]
	if !found || poolPair == nil {
		return nil
	}
	if poolPair.Token1PoolValue == 0 || poolPair.Token2PoolValue == 0 {
		return nil
	}

	tokenPoolValueToBuy := poolPair.Token1PoolValue
	tokenPoolValueToSell := poolPair.Token2PoolValue
	if poolPair.Token1IDStr == fromTokenIDStr {
		tokenPoolValueToBuy = poolPair.Token2PoolValue
		tokenPoolValueToSell = poolPair.Token1PoolValue
	}

	invariant := big.NewInt(0)
	invariant.Mul(big.NewInt(int64(tokenPoolValueToSell)), big.NewInt(int64(tokenPoolValueToBuy)))
	newTokenPoolValueToSell := big.NewInt(0)
	newTokenPoolValueToSell.Add(big.NewInt(int64(tokenPoolValueToSell)), big.NewInt(int64(convertingAmt)))

	newTokenPoolValueToBuy := big.NewInt(0).Div(invariant, newTokenPoolValueToSell).Uint64()
	modValue := big.NewInt(0).Mod(invariant, newTokenPoolValueToSell)
	if modValue.Cmp(big.NewInt(0)) != 0 {
		newTokenPoolValueToBuy++
	}
	if tokenPoolValueToBuy <= newTokenPoolValueToBuy {
		return nil
	}
	return &ConvertedPrice{
		FromTokenIDStr: fromTokenIDStr,
		ToTokenIDStr:   toTokenIDStr,
		Amount:         convertingAmt,
		Price:          tokenPoolValueToBuy - newTokenPoolValueToBuy,
	}
}

func (httpServer *HttpServer) handleConvertPDEPrices(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	latestBcHeight := httpServer.config.BlockChain.BestState.Beacon.BeaconHeight

	arrayParams := common.InterfaceSlice(params)
	data, ok := arrayParams[0].(map[string]interface{})
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Payload data is invalid"))
	}
	fromTokenIDStr, ok := data["FromTokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("FromTokenIDStr is invalid"))
	}
	toTokenIDStr, ok := data["ToTokenIDStr"].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("ToTokenIDStr is invalid"))
	}
	amount, ok := data["Amount"].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Amount is invalid"))
	}
	convertingAmt := uint64(amount)
	if convertingAmt == 0 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Amount is invalid"))
	}
	pdeState, err := blockchain.InitCurrentPDEStateFromDB(httpServer.config.BlockChain.GetDatabase(), latestBcHeight)
	if err != nil || pdeState == nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetPDEStateError, err)
	}
	pdePoolPairs := pdeState.PDEPoolPairs
	results := []*ConvertedPrice{}
	if toTokenIDStr != "all" {
		convertedPrice := convertPrice(
			latestBcHeight,
			toTokenIDStr,
			fromTokenIDStr,
			convertingAmt,
			pdePoolPairs,
		)
		if convertedPrice == nil {
			return results, nil
		}
		return append(results, convertedPrice), nil
	}
	// compute price of "from" token against all tokens else
	for poolPairKey, poolPair := range pdePoolPairs {
		if !strings.Contains(poolPairKey, fromTokenIDStr) {
			continue
		}
		var convertedPrice *ConvertedPrice
		if poolPair.Token1IDStr == fromTokenIDStr {
			convertedPrice = convertPrice(
				latestBcHeight,
				poolPair.Token2IDStr,
				fromTokenIDStr,
				convertingAmt,
				pdePoolPairs,
			)
		} else if poolPair.Token2IDStr == fromTokenIDStr {
			convertedPrice = convertPrice(
				latestBcHeight,
				poolPair.Token1IDStr,
				fromTokenIDStr,
				convertingAmt,
				pdePoolPairs,
			)
		}
		if convertedPrice == nil {
			continue
		}
		results = append(results, convertedPrice)
	}
	return results, nil
}
