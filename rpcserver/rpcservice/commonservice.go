package rpcservice

import (
	"errors"
	rCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/transaction"
	"github.com/incognitochain/incognito-chain/wallet"
	"log"
)

func NewContractingRequestMetadata(senderPrivateKeyStr string, tokenReceivers interface{}, tokenID string) (*metadata.ContractingRequest, *RPCError) {
	senderKey, err := wallet.Base58CheckDeserialize(senderPrivateKeyStr)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, err)
	}
	err = senderKey.KeySet.InitFromPrivateKey(&senderKey.KeySet.PrivateKey)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, err)
	}
	paymentAddr := senderKey.KeySet.PaymentAddress

	_, voutsAmount, err := transaction.CreateCustomTokenReceiverArray(tokenReceivers)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, err)
	}
	tokenIDHash, err := common.Hash{}.NewHashFromStr(tokenID)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, err)
	}

	meta, _ := metadata.NewContractingRequest(
		paymentAddr,
		uint64(voutsAmount),
		*tokenIDHash,
		metadata.ContractingRequestMeta,
	)

	return meta, nil
}

func NewBurningRequestMetadata(senderPrivateKeyStr string, tokenReceivers interface{}, tokenID string, tokenName string, remoteAddress string) (*metadata.BurningRequest, *RPCError) {
	senderKey, err := wallet.Base58CheckDeserialize(senderPrivateKeyStr)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, err)
	}
	err = senderKey.KeySet.InitFromPrivateKey(&senderKey.KeySet.PrivateKey)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, err)
	}
	paymentAddr := senderKey.KeySet.PaymentAddress

	_, voutsAmount, err := transaction.CreateCustomTokenReceiverArray(tokenReceivers)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, err)
	}
	tokenIDHash, err := common.Hash{}.NewHashFromStr(tokenID)
	if err != nil {
		return nil, NewRPCError(UnexpectedError, err)
	}

	meta, err := metadata.NewBurningRequest(
		paymentAddr,
		uint64(voutsAmount),
		*tokenIDHash,
		tokenName,
		remoteAddress,
		metadata.BurningRequestMeta,
	)
	if err != nil{
		return nil, NewRPCError(UnexpectedError, err)
	}

	return meta, nil
}

func GetETHHeaderByHash(ethBlockHash string) (*types.Header, error) {
	return metadata.GetETHHeader(rCommon.HexToHash(ethBlockHash))
}

// GetKeySetFromPrivateKeyParams - deserialize a private key string
// into keyWallet object and fill all keyset in keywallet with private key
// return key set and shard ID
func GetKeySetFromPrivateKeyParams(privateKeyWalletStr string) (*incognitokey.KeySet, byte, error) {
	// deserialize to crate keywallet object which contain private key
	keyWallet, err := wallet.Base58CheckDeserialize(privateKeyWalletStr)
	if err != nil {
		return nil, byte(0), err
	}

	return GetKeySetFromPrivateKey(keyWallet.KeySet.PrivateKey)
}

// GetKeySetFromPrivateKeyParams - deserialize a private key string
// into keyWallet object and fill all keyset in keywallet with private key
// return key set and shard ID
func GetKeySetFromPrivateKey(privateKey privacy.PrivateKey) (*incognitokey.KeySet, byte, error) {
	keySet := new(incognitokey.KeySet)
	// fill paymentaddress and readonly key with privatekey
	err := keySet.InitFromPrivateKey(&privateKey)
	if err != nil {
		return nil, byte(0), err
	}

	// calculate shard ID
	lastByte := keySet.PaymentAddress.Pk[len(keySet.PaymentAddress.Pk)-1]
	shardID := common.GetShardIDFromLastByte(lastByte)

	return keySet, shardID, nil
}

// GetKeySetFromPaymentAddressParam - deserialize a key string(wallet serialized)
// into keyWallet - this keywallet may contain
func GetKeySetFromPaymentAddressParam(paymentAddressStr string) (*incognitokey.KeySet, byte, error) {
	keyWallet, err := wallet.Base58CheckDeserialize(paymentAddressStr)
	if err != nil {
		return nil, byte(0), err
	}

	// calculate shard ID
	lastByte := keyWallet.KeySet.PaymentAddress.Pk[len(keyWallet.KeySet.PaymentAddress.Pk)-1]
	shardID := common.GetShardIDFromLastByte(lastByte)

	return &keyWallet.KeySet, shardID, nil
}

func NewPaymentInfosFromReceiversParam(receiversParam map[string]interface{}) ([]*privacy.PaymentInfo, error) {
	paymentInfos := make([]*privacy.PaymentInfo, 0)
	for paymentAddressStr, amount := range receiversParam {
		keyWalletReceiver, err := wallet.Base58CheckDeserialize(paymentAddressStr)
		if err != nil {
			return nil, err
		}
		paymentInfo := &privacy.PaymentInfo{
			Amount:         uint64(amount.(float64)),
			PaymentAddress: keyWalletReceiver.KeySet.PaymentAddress,
		}
		paymentInfos = append(paymentInfos, paymentInfo)
	}

	return paymentInfos, nil
}

func GetStakingAmount(stakingType int, stakingShardAmountParam uint64) uint64 {
	amount := uint64(0)
	stakingData, _ := metadata.NewStakingMetadata(metadata.ShardStakingMeta, "", "", stakingShardAmountParam, "", true)
	if stakingType == 1 {
		amount = stakingData.GetBeaconStakeAmount()
	}
	if stakingType == 0 {
		amount = stakingData.GetShardStateAmount()
	}

	return amount
}

func HashToIdenticon(hashStrs []interface{}) ([]string, error) {
	result := make([]string, 0)
	for _, hash := range hashStrs {
		temp, err := common.Hash{}.NewHashFromStr(hash.(string))
		if err != nil {
			return nil, errors.New("Hash string is invalid")
		}
		result = append(result, common.Render(temp.GetBytes()))
	}

	return result, nil
}

func GenerateTokenID(network string, name string) (common.Hash, error) {
	point := privacy.HashToPoint([]byte(network + "-" + name))
	hash := new(common.Hash)
	err := hash.SetBytes(point.ToBytesS())
	if err != nil {
		log.Println("Wrong param")
		return common.Hash{}, err
	}
	return *hash, nil
}
