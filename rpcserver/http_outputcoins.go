package rpcserver

import (
	"errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
)

//handleListUnspentOutputCoins - use private key to get all tx which contains output coin of account
// by private key, it return full tx outputcoin with amount and receiver address in txs
//component:
//Parameter #1—the minimum number of confirmations an output must have
//Parameter #2—the maximum number of confirmations an output may have
//Parameter #3—the list priv-key which be used to view utxo
//
func (httpServer *HttpServer) handleListUnspentOutputCoins(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleListUnspentOutputCoins params: %+v", params)

	// get component
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 3 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}

	var min int
	var max int
	if paramsArray[0] != nil {
		minParam, ok := paramsArray[0].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("min param is invalid"))
		}
		min = int(minParam)
	}

	if paramsArray[1] != nil {
		maxParam, ok := paramsArray[1].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("max param is invalid"))
		}
		max = int(maxParam)
	}
	_ = min
	_ = max

	listKeyParams := common.InterfaceSlice(paramsArray[2])
	if listKeyParams == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("list key is invalid"))
	}

	result, err := httpServer.outputCoinService.ListUnspentOutputCoinsByKey(listKeyParams)
	if err != nil {
		return nil, err
	}

	Logger.log.Debugf("handleListUnspentOutputCoins result: %+v", result)
	return result, nil
}

//handleListOutputCoins - use readonly key to get all tx which contains output coin of account
// by private key, it return full tx outputcoin with amount and receiver address in txs
//component:
//Parameter #1—the minimum number of confirmations an output must have
//Parameter #2—the maximum number of confirmations an output may have
//Parameter #3—the list paymentaddress-readonlykey which be used to view list outputcoin
//Parameter #4 - optional - token id - default prv coin
func (httpServer *HttpServer) handleListOutputCoins(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleListOutputCoins params: %+v", params)

	// get component
	paramsArray := common.InterfaceSlice(params)
	if paramsArray == nil || len(paramsArray) < 3 {
		Logger.log.Debugf("handleListOutputCoins result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("param must be an array at least 3 elements"))
	}

	minTemp, ok := paramsArray[0].(float64)
	if !ok {
		Logger.log.Debugf("handleListOutputCoins result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("min param is invalid"))
	}
	min := int(minTemp)

	maxTemp, ok := paramsArray[1].(float64)
	if !ok {
		Logger.log.Debugf("handleListOutputCoins result: %+v", nil)
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("max param is invalid"))
	}
	max := int(maxTemp)

	_ = min
	_ = max

	//#3: list key component
	listKeyParams := common.InterfaceSlice(paramsArray[2])
	if listKeyParams == nil {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("list key is invalid"))
	}

	//#4: optional token type - default prv coin
	tokenID := &common.Hash{}
	err := tokenID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.TokenIsInvalidError, err)
	}
	if len(paramsArray) > 3 {
		var err1 error
		tokenIdParam, ok := paramsArray[3].(string)
		if !ok {
			Logger.log.Debugf("handleListOutputCoins result: %+v", nil)
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token id param is invalid"))
		}

		tokenID, err1 = common.Hash{}.NewHashFromStr(tokenIdParam)
		if err1 != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err1)
			return nil, rpcservice.NewRPCError(rpcservice.ListCustomTokenNotFoundError, err1)
		}
	}
	result, err1 := httpServer.outputCoinService.ListOutputCoinsByKey(listKeyParams, *tokenID)
	if err1 != nil {
		return nil, err1
	}
	Logger.log.Debugf("handleListOutputCoins result: %+v", result)
	return result, nil
}
