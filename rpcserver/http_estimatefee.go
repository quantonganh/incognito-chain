package rpcserver

import (
	"errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/rpcserver/rpcservice"
	"github.com/incognitochain/incognito-chain/transaction"
)

/*
handleEstimateFee - RPC estimates the transaction fee per kilobyte that needs to be paid for a transaction to be included within a certain number of blocks.
*/
func (httpServer *HttpServer) handleEstimateFee(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleEstimateFee params: %+v", params)
	/******* START Fetch all component to ******/
	// all component
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 4 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Not enough params"))
	}
	// param #1: private key of sender
	senderKeyParam, ok := arrayParams[0].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Sender private key is invalid"))
	}
	// param #3: estimation fee coin per kb
	defaultFeeCoinPerKbtemp, ok := arrayParams[2].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Default FeeCoinPerKbtemp is invalid"))
	}
	defaultFeeCoinPerKb := int64(defaultFeeCoinPerKbtemp)
	// param #4: hasPrivacy flag for PRV
	hashPrivacyTemp, ok := arrayParams[3].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("hasPrivacy is invalid"))
	}
	hasPrivacy := int(hashPrivacyTemp) > 0

	senderKeySet, shardIDSender, err := rpcservice.GetKeySetFromPrivateKeyParams(senderKeyParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.InvalidSenderPrivateKeyError, err)
	}

	outCoins, err := httpServer.outputCoinService.ListOutputCoinsByKeySet(senderKeySet, shardIDSender)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetOutputCoinError, err)
	}

	// remove out coin in mem pool
	outCoins, err = httpServer.txMemPoolService.FilterMemPoolOutcoinsToSpent(outCoins)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.GetOutputCoinError, err)
	}

	estimateFeeCoinPerKb := uint64(0)
	estimateTxSizeInKb := uint64(0)
	if len(outCoins) > 0 {
		// param #2: list receiver
		receiversPaymentAddressStrParam := make(map[string]interface{})
		if arrayParams[1] != nil {
			receiversPaymentAddressStrParam, ok = arrayParams[1].(map[string]interface{})
			if !ok {
				return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("receivers payment address is invalid"))
			}
		}

		paymentInfos, err := rpcservice.NewPaymentInfosFromReceiversParam(receiversPaymentAddressStrParam)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.InvalidReceiverPaymentAddressError, err)
		}

		// Check custom token param
		var customTokenParams *transaction.CustomTokenParamTx
		var customPrivacyTokenParam *transaction.CustomTokenPrivacyParamTx
		isGetPTokenFee := false
		if len(arrayParams) > 4 {
			// param #5: token params
			tokenParamsRaw, ok := arrayParams[4].(map[string]interface{})
			if !ok {
				return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token param is invalid"))
			}

			customTokenParams, customPrivacyTokenParam, err = httpServer.txService.BuildTokenParam(tokenParamsRaw, senderKeySet, shardIDSender)
			if err.(*rpcservice.RPCError) != nil {
				return nil, err.(*rpcservice.RPCError)
			}
		}

		beaconState, err := httpServer.blockService.BlockChain.BestState.GetClonedBeaconBestState()
		beaconHeight := beaconState.BeaconHeight

		var err2 error
		_, estimateFeeCoinPerKb, estimateTxSizeInKb, err2 = httpServer.txService.EstimateFee(
			defaultFeeCoinPerKb, isGetPTokenFee, outCoins, paymentInfos, shardIDSender, 8, hasPrivacy, nil,
			customTokenParams, customPrivacyTokenParam, *httpServer.config.Database, int64(beaconHeight))
		if err2 != nil{
			return nil, rpcservice.NewRPCError(rpcservice.RejectInvalidFeeError, err2)
		}
	}
	result := jsonresult.NewEstimateFeeResult(estimateFeeCoinPerKb, estimateTxSizeInKb)
	Logger.log.Debugf("handleEstimateFee result: %+v", result)
	return result, nil
}

// handleEstimateFeeWithEstimator -- get fee from estimator
func (httpServer *HttpServer) handleEstimateFeeWithEstimator(params interface{}, closeChan <-chan struct{}) (interface{}, *rpcservice.RPCError) {
	Logger.log.Debugf("handleEstimateFeeWithEstimator params: %+v", params)
	// all params
	arrayParams := common.InterfaceSlice(params)
	if arrayParams == nil || len(arrayParams) < 2 {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("Not enough params"))
	}
	// param #1: estimation fee coin per kb from client
	defaultFeeCoinPerKbTemp, ok := arrayParams[0].(float64)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("defaultFeeCoinPerKbTemp is invalid"))
	}
	defaultFeeCoinPerKb := int64(defaultFeeCoinPerKbTemp)

	// param #2: payment address
	paymentAddressParam, ok := arrayParams[1].(string)
	if !ok {
		return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("sender key param is invalid"))
	}
	_, shardIDSender, err := rpcservice.GetKeySetFromPaymentAddressParam(paymentAddressParam)
	if err != nil {
		return nil, rpcservice.NewRPCError(rpcservice.InvalidSenderPrivateKeyError, err)
	}

	// param #2: numbloc
	numblock := uint64(8)
	if len(arrayParams) >= 3 {
		numBlockParam, ok := arrayParams[2].(float64)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("num block param is invalid"))
		}
		numblock = uint64(numBlockParam)
	}

	// param #3: tokenId
	// if tokenID != nil, return fee for privacy token
	// if tokenID != nil, return fee for native token
	var tokenId *common.Hash
	if len(arrayParams) >= 4 && arrayParams[3] != nil {
		tokenIdParam, ok := arrayParams[3].(string)
		if !ok {
			return nil, rpcservice.NewRPCError(rpcservice.RPCInvalidParamsError, errors.New("token id param is invalid"))
		}
		tokenId, err = common.Hash{}.NewHashFromStr(tokenIdParam)
		if err != nil {
			return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
		}
	}

	beaconState, err := httpServer.blockService.BlockChain.BestState.GetClonedBeaconBestState()
	beaconHeight := beaconState.BeaconHeight

	estimateFeeCoinPerKb, err := httpServer.txService.EstimateFeeWithEstimator(defaultFeeCoinPerKb, shardIDSender, numblock, tokenId, int64(beaconHeight), *httpServer.config.Database)
	if err != nil{
		return nil, rpcservice.NewRPCError(rpcservice.UnexpectedError, err)
	}

	result := jsonresult.NewEstimateFeeResult(estimateFeeCoinPerKb, 0)
	Logger.log.Debugf("handleEstimateFeeWithEstimator result: %+v", result)
	return result, nil
}
