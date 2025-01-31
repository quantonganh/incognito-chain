package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database"
	"github.com/incognitochain/incognito-chain/wallet"
)

// PDEWithdrawalRequest - privacy dex withdrawal request
type PDEWithdrawalRequest struct {
	WithdrawerAddressStr  string
	WithdrawalToken1IDStr string
	WithdrawalToken2IDStr string
	WithdrawalShareAmt    uint64
	MetadataBase
}

type PDEWithdrawalRequestAction struct {
	Meta    PDEWithdrawalRequest
	TxReqID common.Hash
	ShardID byte
}

type PDEWithdrawalAcceptedContent struct {
	WithdrawalTokenIDStr string
	WithdrawerAddressStr string
	DeductingPoolValue   uint64
	DeductingShares      uint64
	PairToken1IDStr      string
	PairToken2IDStr      string
	TxReqID              common.Hash
	ShardID              byte
}

func NewPDEWithdrawalRequest(
	withdrawerAddressStr string,
	withdrawalToken1IDStr string,
	withdrawalToken2IDStr string,
	withdrawalShareAmt uint64,
	metaType int,
) (*PDEWithdrawalRequest, error) {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	pdeWithdrawalRequest := &PDEWithdrawalRequest{
		WithdrawerAddressStr:  withdrawerAddressStr,
		WithdrawalToken1IDStr: withdrawalToken1IDStr,
		WithdrawalToken2IDStr: withdrawalToken2IDStr,
		WithdrawalShareAmt:    withdrawalShareAmt,
	}
	pdeWithdrawalRequest.MetadataBase = metadataBase
	return pdeWithdrawalRequest, nil
}

func (pc PDEWithdrawalRequest) ValidateTxWithBlockChain(
	txr Transaction,
	bcr BlockchainRetriever,
	shardID byte,
	db database.DatabaseInterface,
) (bool, error) {
	// NOTE: verify supported tokens pair as needed
	return true, nil
}

func (pc PDEWithdrawalRequest) ValidateSanityData(bcr BlockchainRetriever, txr Transaction) (bool, bool, error) {
	keyWallet, err := wallet.Base58CheckDeserialize(pc.WithdrawerAddressStr)
	if err != nil {
		return false, false, NewMetadataTxError(PDEWithdrawalRequestFromMapError, errors.New("WithdrawerAddressStr incorrect"))
	}
	withdrawerAddr := keyWallet.KeySet.PaymentAddress
	if len(withdrawerAddr.Pk) == 0 {
		return false, false, errors.New("Wrong request info's withdrawer address")
	}
	if !bytes.Equal(txr.GetSigPubKey()[:], withdrawerAddr.Pk[:]) {
		return false, false, errors.New("WithdrawerAddr incorrect")
	}
	_, err = common.Hash{}.NewHashFromStr(pc.WithdrawalToken1IDStr)
	if err != nil {
		return false, false, NewMetadataTxError(PDEWithdrawalRequestFromMapError, errors.New("WithdrawalTokenID1Str incorrect"))
	}
	_, err = common.Hash{}.NewHashFromStr(pc.WithdrawalToken2IDStr)
	if err != nil {
		return false, false, NewMetadataTxError(PDEWithdrawalRequestFromMapError, errors.New("WithdrawalTokenID2Str incorrect"))
	}
	if pc.WithdrawalShareAmt == 0 {
		return false, false, NewMetadataTxError(PDEWithdrawalRequestFromMapError, errors.New("WithdrawalShareAmt should be large than 0"))
	}
	return true, true, nil
}

func (pc PDEWithdrawalRequest) ValidateMetadataByItself() bool {
	return pc.Type == PDEWithdrawalRequestMeta
}

func (pc PDEWithdrawalRequest) Hash() *common.Hash {
	record := pc.MetadataBase.Hash().String()
	record += pc.WithdrawerAddressStr
	record += pc.WithdrawalToken1IDStr
	record += pc.WithdrawalToken2IDStr
	record += strconv.FormatUint(pc.WithdrawalShareAmt, 10)
	// final hash
	hash := common.HashH([]byte(record))
	return &hash
}

func (pc *PDEWithdrawalRequest) BuildReqActions(tx Transaction, bcr BlockchainRetriever, shardID byte) ([][]string, error) {
	actionContent := PDEWithdrawalRequestAction{
		Meta:    *pc,
		TxReqID: *tx.Hash(),
		ShardID: shardID,
	}
	actionContentBytes, err := json.Marshal(actionContent)
	if err != nil {
		return [][]string{}, err
	}
	actionContentBase64Str := base64.StdEncoding.EncodeToString(actionContentBytes)
	action := []string{strconv.Itoa(PDEWithdrawalRequestMeta), actionContentBase64Str}
	return [][]string{action}, nil
}

func (pc *PDEWithdrawalRequest) CalculateSize() uint64 {
	return calculateSize(pc)
}
