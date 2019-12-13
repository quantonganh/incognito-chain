package rpcservice

import (
	"errors"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/rpcserver/jsonresult"
	"github.com/incognitochain/incognito-chain/wallet"
)

type CoinService struct {
	BlockChain *blockchain.BlockChain
}

func (coinService CoinService) ListOutputCoinsByKeySet(keySet *incognitokey.KeySet, shardID byte) ([]*privacy.OutputCoin, error) {
	prvCoinID := &common.Hash{}
	err := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, err
	}

	return coinService.BlockChain.GetListOutputCoinsByKeyset(keySet, shardID, prvCoinID)
}

func (coinService CoinService) ListUnspentOutputCoinsByKey(listKeyParams []interface{}) (*jsonresult.ListOutputCoins, *RPCError) {
	result := &jsonresult.ListOutputCoins{
		Outputs: make(map[string][]jsonresult.OutCoin),
	}
	for _, keyParam := range listKeyParams {
		keys, ok := keyParam.(map[string]interface{})
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("key param is invalid"))
		}

		// get keyset only contain pri-key by deserializing
		priKeyStr, ok := keys["PrivateKey"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("private key is invalid"))
		}
		keyWallet, err := wallet.Base58CheckDeserialize(priKeyStr)
		if err != nil || keyWallet.KeySet.PrivateKey == nil {
			Logger.log.Error("Check Deserialize err", err)
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("private key is invalid"))
		}

		keySetTmp, shardID, err := GetKeySetFromPrivateKey(keyWallet.KeySet.PrivateKey)
		if err != nil {
			return nil, NewRPCError(UnexpectedError, err)
		}
		keyWallet.KeySet = *keySetTmp

		outCoins, _, err := coinService.ListOutputCoinsByKeySetV2(&keyWallet.KeySet, shardID, 0, 0)
		if err != nil {
			return nil, NewRPCError(UnexpectedError, err)
		}

		item := make([]jsonresult.OutCoin, 0)
		for _, outCoin := range outCoins {
			if outCoin.CoinDetails.GetValue() == 0 {
				continue
			}
			item = append(item, jsonresult.NewOutCoin(outCoin))
		}
		result.Outputs[priKeyStr] = item
	}
	return result, nil
}

func (coinService CoinService) ListOutputCoinsByKey(listKeyParams []interface{}, tokenID common.Hash) (*jsonresult.ListOutputCoins, *RPCError) {
	result := &jsonresult.ListOutputCoins{
		Outputs: make(map[string][]jsonresult.OutCoin),
	}
	for _, keyParam := range listKeyParams {
		keys, ok := keyParam.(map[string]interface{})
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("key param is invalid"))
		}

		// get keyset only contain readonly-key by deserializing
		readonlyKeyStr, ok := keys["ReadonlyKey"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid readonly key"))
		}
		readonlyKey, err := wallet.Base58CheckDeserialize(readonlyKeyStr)
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(UnexpectedError, err)
		}

		// get keyset only contain pub-key by deserializing
		pubKeyStr, ok := keys["PaymentAddress"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid payment address"))
		}
		pubKey, err := wallet.Base58CheckDeserialize(pubKeyStr)
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(UnexpectedError, err)
		}

		// create a key set
		keySet := incognitokey.KeySet{
			ReadonlyKey:    readonlyKey.KeySet.ReadonlyKey,
			PaymentAddress: pubKey.KeySet.PaymentAddress,
		}
		lastByte := keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1]
		shardIDSender := common.GetShardIDFromLastByte(lastByte)
		outputCoins, err := coinService.BlockChain.GetListOutputCoinsByKeyset(&keySet, shardIDSender, &tokenID)
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(UnexpectedError, err)
		}
		item := make([]jsonresult.OutCoin, 0)

		for _, outCoin := range outputCoins {
			if outCoin.CoinDetails.GetValue() == 0 {
				continue
			}
			item = append(item, jsonresult.NewOutCoin(outCoin))
		}
		result.Outputs[readonlyKeyStr] = item
	}
	return result, nil
}

/* =================== TRANSACTION V2  =================== */

func (coinService CoinService) ListOutputCoinsByKeyV2(listKeyParams []interface{}, tokenID common.Hash, fromBlockHeightParam int64, toBlockHeightParam int64) (*jsonresult.ListOutputCoins, *RPCError) {
	result := &jsonresult.ListOutputCoins{
		Outputs:             make(map[string][]jsonresult.OutCoin),
		CurrentBlockHeights: make(map[string]uint64),
	}

	for _, keyParam := range listKeyParams {
		keys, ok := keyParam.(map[string]interface{})
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("key param is invalid"))
		}

		// get keyset only contain readonly-key by deserializing
		readonlyKeyStr, ok := keys["ReadonlyKey"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid readonly key"))
		}
		readonlyKey, err := wallet.Base58CheckDeserialize(readonlyKeyStr)
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(UnexpectedError, err)
		}

		// get keyset only contain pub-key by deserializing
		pubKeyStr, ok := keys["PaymentAddress"].(string)
		if !ok {
			return nil, NewRPCError(RPCInvalidParamsError, errors.New("invalid payment address"))
		}
		pubKey, err := wallet.Base58CheckDeserialize(pubKeyStr)
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(UnexpectedError, err)
		}

		// create a key set
		keySet := incognitokey.KeySet{
			ReadonlyKey:    readonlyKey.KeySet.ReadonlyKey,
			PaymentAddress: pubKey.KeySet.PaymentAddress,
		}
		lastByte := keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1]
		shardIDSender := common.GetShardIDFromLastByte(lastByte)

		// get outout coins from version 2 from fromBlockHeight to toBlockHeight
		outputCoins, currentBlockHeight, err := coinService.BlockChain.GetListOutputCoinsByKeysetV2(&keySet, shardIDSender, &tokenID, fromBlockHeightParam, toBlockHeightParam)
		if err != nil {
			Logger.log.Debugf("handleListOutputCoins result: %+v, err: %+v", nil, err)
			return nil, NewRPCError(UnexpectedError, err)
		}
		item := make([]jsonresult.OutCoin, 0)

		for _, outCoin := range outputCoins {
			if outCoin.CoinDetails.GetValue() == 0 {
				continue
			}
			item = append(item, jsonresult.NewOutCoin(outCoin))
		}
		result.Outputs[readonlyKeyStr] = item
		result.CurrentBlockHeights[readonlyKeyStr] = currentBlockHeight
	}
	return result, nil
}

func (coinService CoinService) ListOutputCoinsByKeySetV2(keySet *incognitokey.KeySet, shardID byte, fromBlockHeight int64, toBlockHeight int64) ([]*privacy.OutputCoin, uint64, error) {
	prvCoinID := &common.Hash{}
	err := prvCoinID.SetBytes(common.PRVCoinID[:])
	if err != nil {
		return nil, uint64(0), err
	}

	return coinService.BlockChain.GetListOutputCoinsByKeysetV2(keySet, shardID, prvCoinID, fromBlockHeight, toBlockHeight)
}
