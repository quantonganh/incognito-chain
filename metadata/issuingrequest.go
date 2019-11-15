package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/privacy"
	"github.com/incognitochain/incognito-chain/wallet"
)

// only centralized website can send this type of tx
type IssuingRequest struct {
	ReceiverAddress privacy.PaymentAddress
	DepositedAmount uint64
	TokenID         common.Hash
	TokenName       string
	MetadataBase
}

type IssuingReqAction struct {
	Meta    IssuingRequest `json:"meta"`
	TxReqID common.Hash    `json:"txReqId"`
}

type IssuingAcceptedInst struct {
	ShardID         byte                   `json:"shardId"`
	DepositedAmount uint64                 `json:"issuingAmount"`
	ReceiverAddr    privacy.PaymentAddress `json:"receiverAddrStr"`
	IncTokenID      common.Hash            `json:"incTokenId"`
	IncTokenName    string                 `json:"incTokenName"`
	TxReqID         common.Hash            `json:"txReqId"`
}

func ParseIssuingInstContent(instContentStr string) (*IssuingReqAction, error) {
	contentBytes, err := base64.StdEncoding.DecodeString(instContentStr)
	if err != nil {
		return nil, NewMetadataTxError(IssuingRequestDecodeInstructionError, err)
	}
	var issuingReqAction IssuingReqAction
	err = json.Unmarshal(contentBytes, &issuingReqAction)
	if err != nil {
		return nil, NewMetadataTxError(IssuingRequestUnmarshalJsonError, err)
	}
	return &issuingReqAction, nil
}

func NewIssuingRequest(
	receiverAddress privacy.PaymentAddress,
	depositedAmount uint64,
	tokenID common.Hash,
	tokenName string,
	metaType int,
) (*IssuingRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	issuingReq := &IssuingRequest{
		ReceiverAddress: receiverAddress,
		DepositedAmount: depositedAmount,
		TokenID:         tokenID,
		TokenName:       tokenName,
	}
	issuingReq.MetadataBase = metadataBase
	return issuingReq, nil
}

func NewIssuingRequestFromMap(data map[string]interface{}) (Metadata, error) {
	tokenID, err := common.Hash{}.NewHashFromStr(data["TokenID"].(string))
	if err != nil {
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenID incorrect"))
	}

	tokenName, ok := data["TokenName"].(string)
	if !ok {
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenName incorrect"))
	}

	depositedAmount, ok := data["DepositedAmount"]
	if !ok {
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("DepositedAmount incorrect"))
	}
	depositedAmountFloat, ok := depositedAmount.(float64)
	if !ok {
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("DepositedAmount incorrect"))
	}
	depositedAmt := uint64(depositedAmountFloat)
	keyWallet, err := wallet.Base58CheckDeserialize(data["ReceiveAddress"].(string))
	if err != nil {
		return nil, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("ReceiveAddress incorrect"))
	}

	return NewIssuingRequest(
		keyWallet.KeySet.PaymentAddress,
		depositedAmt,
		*tokenID,
		tokenName,
		IssuingRequestMeta,
	)
}

func (iReq IssuingRequest) ValidateTxWithBlockChain(
	txr Transaction,
	bcr BlockchainRetriever,
	shardID byte,
	db database.DatabaseInterface,
) (bool, error) {
	keySet, err := wallet.Base58CheckDeserialize(bcr.GetCentralizedWebsitePaymentAddress())
	if err != nil || !bytes.Equal(txr.GetSigPubKey(), keySet.KeySet.PaymentAddress.Pk) {
		return false, NewMetadataTxError(IssuingRequestValidateTxWithBlockChainError, errors.New("the issuance request must be called by centralized website"))
	}
	return true, nil
}

func (iReq IssuingRequest) ValidateSanityData(bcr BlockchainRetriever, txr Transaction) (bool, bool, error) {
	if len(iReq.ReceiverAddress.Pk) == 0 {
		return false, false, NewMetadataTxError(IssuingRequestValidateSanityDataError, errors.New("Wrong request info's receiver address"))
	}
	if iReq.DepositedAmount == 0 {
		return false, false, errors.New("Wrong request info's deposited amount")
	}
	if iReq.Type != IssuingRequestMeta {
		return false, false, NewMetadataTxError(IssuingRequestValidateSanityDataError, errors.New("Wrong request info's meta type"))
	}
	if iReq.TokenName == "" {
		return false, false, NewMetadataTxError(IssuingRequestValidateSanityDataError, errors.New("Wrong request info's token name"))
	}
	return true, true, nil
}

func (iReq IssuingRequest) ValidateMetadataByItself() bool {
	return iReq.Type == IssuingRequestMeta
}

func (iReq IssuingRequest) Hash() *common.Hash {
	record := iReq.ReceiverAddress.String()
	record += iReq.TokenID.String()
	record += string(iReq.DepositedAmount)
	record += iReq.TokenName
	record += iReq.MetadataBase.Hash().String()

	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (iReq *IssuingRequest) BuildReqActions(tx Transaction, bcr BlockchainRetriever, shardID byte) ([][]string, error) {
	txReqID := *(tx.Hash())
	actionContent := map[string]interface{}{
		"meta":    *iReq,
		"txReqId": txReqID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, NewMetadataTxError(IssuingRequestBuildReqActionsError, err)
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(IssuingRequestMeta), actionContentBase64Str}
	// track the request status to leveldb
	err = bcr.GetDatabase().TrackBridgeReqWithStatus(txReqID, byte(common.BridgeRequestProcessingStatus), nil)
	if err != nil {
		return [][]string{}, NewMetadataTxError(IssuingRequestBuildReqActionsError, err)
	}
	return [][]string{action}, nil
}

func (iReq *IssuingRequest) CalculateSize() uint64 {
	return calculateSize(iReq)
}
