package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/wallet"
)

// PDETradeRequest - privacy dex trade
type PDETradeRequest struct {
	TokenIDToBuyStr  string
	TokenIDToSellStr string
	SellAmount       uint64 // must be equal to vout value
	TraderAddressStr string
	MetadataBase
}

type PDETradeRequestAction struct {
	Meta    PDETradeRequest
	TxReqID common.Hash
	ShardID byte
}

type TokenPoolValueOperation struct {
	Operator string
	Value    uint64
}

type PDETradeAcceptedContent struct {
	TraderAddressStr         string
	TokenIDToBuyStr          string
	ReceiveAmount            uint64
	Token1IDStr              string
	Token2IDStr              string
	Token1PoolValueOperation TokenPoolValueOperation
	Token2PoolValueOperation TokenPoolValueOperation
	ShardID                  byte
	RequestedTxID            common.Hash
}

func NewPDETradeRequest(
	tokenIDToBuyStr string,
	tokenIDToSellStr string,
	sellAmount uint64,
	traderAddressStr string,
	metaType int,
) (*PDETradeRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	pdeTradeRequest := &PDETradeRequest{
		TokenIDToBuyStr:  tokenIDToBuyStr,
		TokenIDToSellStr: tokenIDToSellStr,
		SellAmount:       sellAmount,
		TraderAddressStr: traderAddressStr,
	}
	pdeTradeRequest.MetadataBase = metadataBase
	return pdeTradeRequest, nil
}

func (pc PDETradeRequest) ValidateTxWithBlockChain(
	txr Transaction,
	bcr BlockchainRetriever,
	shardID byte,
	db database.DatabaseInterface,
) (bool, error) {
	// NOTE: verify supported tokens pair as needed
	return true, nil
}

func (pc PDETradeRequest) ValidateSanityData(bcr BlockchainRetriever, txr Transaction) (bool, bool, error) {
	// Note: the metadata was already verified with *transaction.TxCustomToken level so no need to verify with *transaction.Tx level again as *transaction.Tx is embedding property of *transaction.TxCustomToken
	if txr.GetType() == common.TxCustomTokenPrivacyType && reflect.TypeOf(txr).String() == "*transaction.Tx" {
		return true, true, nil
	}

	keyWallet, err := wallet.Base58CheckDeserialize(pc.TraderAddressStr)
	if err != nil {
		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TraderAddressStr incorrect"))
	}
	traderAddr := keyWallet.KeySet.PaymentAddress

	if len(traderAddr.Pk) == 0 {
		return false, false, errors.New("Wrong request info's trader address")
	}
	if !txr.IsCoinsBurning() {
		return false, false, errors.New("Must send coin to burning address")
	}
	if pc.SellAmount != txr.CalculateTxValue() {
		return false, false, errors.New("Contributed Amount should be equal to the tx value")
	}
	if !bytes.Equal(txr.GetSigPubKey()[:], traderAddr.Pk[:]) {
		return false, false, errors.New("TraderAddress incorrect")
	}

	_, err = common.Hash{}.NewHashFromStr(pc.TokenIDToBuyStr)
	if err != nil {
		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenIDToBuyStr incorrect"))
	}

	tokenIDToSell, err := common.Hash{}.NewHashFromStr(pc.TokenIDToSellStr)
	if err != nil {
		return false, false, NewMetadataTxError(IssuingRequestNewIssuingRequestFromMapEror, errors.New("TokenIDToSellStr incorrect"))
	}

	if !bytes.Equal(txr.GetTokenID()[:], tokenIDToSell[:]) {
		return false, false, errors.New("Wrong request info's token id, it should be equal to tx's token id.")
	}

	if txr.GetType() == common.TxNormalType && pc.TokenIDToSellStr != common.PRVCoinID.String() {
		return false, false, errors.New("With tx normal privacy, the tokenIDStr should be PRV, not custom token.")
	}

	if txr.GetType() == common.TxCustomTokenPrivacyType && pc.TokenIDToSellStr == common.PRVCoinID.String() {
		return false, false, errors.New("With tx custome token privacy, the tokenIDStr should not be PRV, but custom token.")
	}

	return true, true, nil
}

func (pc PDETradeRequest) ValidateMetadataByItself() bool {
	return pc.Type == PDETradeRequestMeta
}

func (pc PDETradeRequest) Hash() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.TokenIDToBuyStr
	record += pc.TokenIDToSellStr
	record += pc.TraderAddressStr
	record += strconv.FormatUint(pc.SellAmount, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc *PDETradeRequest) BuildReqActions(tx Transaction, bcr BlockchainRetriever, shardID byte) ([][]string, error) {
	actionContent := PDETradeRequestAction{
		Meta:    *pc,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(PDETradeRequestMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (pc *PDETradeRequest) CalculateSize() uint64 {
	return calculateSize(pc)
}
