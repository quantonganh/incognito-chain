package blockchain

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/database/lvdb"
	"github.com/incognitochain/incognito-chain/metadata"
)

func buildWaitingContributionInst(
	pdeContributionPairID string,
	contributorAddressStr string,
	contributedAmount uint64,
	tokenIDStr string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
) []string {
	waitingContribution := metadata.PDEWaitingContribution{
		PDEContributionPairID: pdeContributionPairID,
		ContributorAddressStr: contributorAddressStr,
		ContributedAmount:     contributedAmount,
		TokenIDStr:            tokenIDStr,
		TxReqID:               txReqID,
	}
	waitingContributionBytes, _ := json.Marshal(waitingContribution)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		"waiting",
		string(waitingContributionBytes),
	}
}

func buildRefundContributionInst(
	pdeContributionPairID string,
	contributorAddressStr string,
	contributedAmount uint64,
	tokenIDStr string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
) []string {
	refundContribution := metadata.PDERefundContribution{
		PDEContributionPairID: pdeContributionPairID,
		ContributorAddressStr: contributorAddressStr,
		ContributedAmount:     contributedAmount,
		TokenIDStr:            tokenIDStr,
		TxReqID:               txReqID,
		ShardID:               shardID,
	}
	refundContributionBytes, _ := json.Marshal(refundContribution)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		"refund",
		string(refundContributionBytes),
	}
}

func buildMatchedContributionInst(
	pdeContributionPairID string,
	contributorAddressStr string,
	contributedAmount uint64,
	tokenIDStr string,
	metaType int,
	shardID byte,
	txReqID common.Hash,
) []string {
	matchedContribution := metadata.PDEMatchedContribution{
		PDEContributionPairID: pdeContributionPairID,
		ContributorAddressStr: contributorAddressStr,
		ContributedAmount:     contributedAmount,
		TokenIDStr:            tokenIDStr,
		TxReqID:               txReqID,
	}
	matchedContributionBytes, _ := json.Marshal(matchedContribution)
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		"matched",
		string(matchedContributionBytes),
	}
}

func isRightRatio(
	waitingContribution1 *lvdb.PDEContribution,
	waitingContribution2 *lvdb.PDEContribution,
	poolPair *lvdb.PDEPoolForPair,
) bool {
	if poolPair == nil { // first contribution on the pair
		return true
	}
	if poolPair.Token1PoolValue == 0 || poolPair.Token2PoolValue == 0 {
		return true
	}
	if waitingContribution1.TokenIDStr == poolPair.Token1IDStr {
		expectedContribAmt := big.NewInt(0)
		expectedContribAmt.Mul(
			big.NewInt(int64(waitingContribution1.Amount)),
			big.NewInt(int64(poolPair.Token2PoolValue)),
		)
		expectedContribAmt.Div(
			expectedContribAmt,
			big.NewInt(int64(poolPair.Token1PoolValue)),
		)
		return expectedContribAmt.Uint64() == waitingContribution2.Amount
	}
	if waitingContribution1.TokenIDStr == poolPair.Token2IDStr {
		expectedContribAmt := big.NewInt(0)
		expectedContribAmt.Mul(
			big.NewInt(int64(waitingContribution1.Amount)),
			big.NewInt(int64(poolPair.Token1PoolValue)),
		)
		expectedContribAmt.Div(
			expectedContribAmt,
			big.NewInt(int64(poolPair.Token2PoolValue)),
		)
		return expectedContribAmt.Uint64() == waitingContribution2.Amount
	}
	return false
}

func (blockchain *BlockChain) buildInstructionsForPDEContribution(
	contentStr string,
	shardID byte,
	metaType int,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
) ([][]string, error) {
	if currentPDEState == nil {
		Logger.log.Warn("WARN - [buildInstructionsForPDEContribution]: Current PDE state is null.")
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			"refund",
			contentStr,
		}
		return [][]string{inst}, nil
	}
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde withdrawal action: %+v", err)
		return [][]string{}, nil
	}
	var pdeContributionAction metadata.PDEContributionAction
	err = json.Unmarshal(contentBytes, &pdeContributionAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde contribution action: %+v", err)
		return [][]string{}, nil
	}
	meta := pdeContributionAction.Meta
	waitingContribPairKey := string(lvdb.BuildWaitingPDEContributionKey(beaconHeight, meta.PDEContributionPairID))
	waitingContribution, found := currentPDEState.WaitingPDEContributions[waitingContribPairKey]
	if !found || waitingContribution == nil {
		currentPDEState.WaitingPDEContributions[waitingContribPairKey] = &lvdb.PDEContribution{
			ContributorAddressStr: meta.ContributorAddressStr,
			TokenIDStr:            meta.TokenIDStr,
			Amount:                meta.ContributedAmount,
			TxReqID:               pdeContributionAction.TxReqID,
		}
		inst := buildWaitingContributionInst(
			meta.PDEContributionPairID,
			meta.ContributorAddressStr,
			meta.ContributedAmount,
			meta.TokenIDStr,
			metaType,
			shardID,
			pdeContributionAction.TxReqID,
		)
		return [][]string{inst}, nil
	}
	if waitingContribution.TokenIDStr == meta.TokenIDStr ||
		waitingContribution.ContributorAddressStr != meta.ContributorAddressStr {
		delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
		refundInst1 := buildRefundContributionInst(
			meta.PDEContributionPairID,
			meta.ContributorAddressStr,
			meta.ContributedAmount,
			meta.TokenIDStr,
			metaType,
			shardID,
			pdeContributionAction.TxReqID,
		)
		refundInst2 := buildRefundContributionInst(
			meta.PDEContributionPairID,
			waitingContribution.ContributorAddressStr,
			waitingContribution.Amount,
			waitingContribution.TokenIDStr,
			metaType,
			shardID,
			waitingContribution.TxReqID,
		)
		return [][]string{refundInst1, refundInst2}, nil
	}
	// contributed to 2 diff sides of a pair and its a first contribution of this pair
	poolPairs := currentPDEState.PDEPoolPairs
	poolPairKey := string(lvdb.BuildPDEPoolForPairKey(beaconHeight, waitingContribution.TokenIDStr, meta.TokenIDStr))
	poolPair, found := poolPairs[poolPairKey]
	incomingWaitingContribution := &lvdb.PDEContribution{
		ContributorAddressStr: meta.ContributorAddressStr,
		TokenIDStr:            meta.TokenIDStr,
		Amount:                meta.ContributedAmount,
		TxReqID:               pdeContributionAction.TxReqID,
	}

	if !found || poolPair == nil ||
		isRightRatio(waitingContribution, incomingWaitingContribution, poolPair) {
		delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
		updateWaitingContributionPairToPoolV2(
			beaconHeight,
			waitingContribution,
			incomingWaitingContribution,
			currentPDEState,
		)
		matchedInst := buildMatchedContributionInst(
			meta.PDEContributionPairID,
			meta.ContributorAddressStr,
			meta.ContributedAmount,
			meta.TokenIDStr,
			metaType,
			shardID,
			pdeContributionAction.TxReqID,
		)
		return [][]string{matchedInst}, nil
	}

	// the contribution was not right with the current ratio of the pair
	delete(currentPDEState.WaitingPDEContributions, waitingContribPairKey)
	refundInst1 := buildRefundContributionInst(
		meta.PDEContributionPairID,
		meta.ContributorAddressStr,
		meta.ContributedAmount,
		meta.TokenIDStr,
		metaType,
		shardID,
		pdeContributionAction.TxReqID,
	)
	refundInst2 := buildRefundContributionInst(
		meta.PDEContributionPairID,
		waitingContribution.ContributorAddressStr,
		waitingContribution.Amount,
		waitingContribution.TokenIDStr,
		metaType,
		shardID,
		waitingContribution.TxReqID,
	)
	return [][]string{refundInst1, refundInst2}, nil
}

func (blockchain *BlockChain) buildInstructionsForPDETrade(
	contentStr string,
	shardID byte,
	metaType int,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
) ([][]string, error) {
	if currentPDEState == nil ||
		(currentPDEState.PDEPoolPairs == nil || len(currentPDEState.PDEPoolPairs) == 0) {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			"refund",
			contentStr,
		}
		return [][]string{inst}, nil
	}
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde trade instruction: %+v", err)
		return [][]string{}, nil
	}
	var pdeTradeReqAction metadata.PDETradeRequestAction
	err = json.Unmarshal(contentBytes, &pdeTradeReqAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde trade instruction: %+v", err)
		return [][]string{}, nil
	}
	pairKey := string(lvdb.BuildPDEPoolForPairKey(beaconHeight, pdeTradeReqAction.Meta.TokenIDToBuyStr, pdeTradeReqAction.Meta.TokenIDToSellStr))

	pdePoolPair, found := currentPDEState.PDEPoolPairs[pairKey]
	if !found || (pdePoolPair.Token1PoolValue == 0 || pdePoolPair.Token2PoolValue == 0) {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			"refund",
			contentStr,
		}
		return [][]string{inst}, nil
	}
	// trade accepted
	tokenPoolValueToBuy := pdePoolPair.Token1PoolValue
	tokenPoolValueToSell := pdePoolPair.Token2PoolValue
	if pdePoolPair.Token1IDStr == pdeTradeReqAction.Meta.TokenIDToSellStr {
		tokenPoolValueToSell = pdePoolPair.Token1PoolValue
		tokenPoolValueToBuy = pdePoolPair.Token2PoolValue
	}
	invariant := big.NewInt(0)
	invariant.Mul(big.NewInt(int64(tokenPoolValueToSell)), big.NewInt(int64(tokenPoolValueToBuy)))
	fee := pdeTradeReqAction.Meta.TradingFee
	newTokenPoolValueToSell := big.NewInt(0)
	newTokenPoolValueToSell.Add(big.NewInt(int64(tokenPoolValueToSell)), big.NewInt(int64(pdeTradeReqAction.Meta.SellAmount)))

	newTokenPoolValueToBuy := big.NewInt(0).Div(invariant, newTokenPoolValueToSell).Uint64()
	modValue := big.NewInt(0).Mod(invariant, newTokenPoolValueToSell)
	if modValue.Cmp(big.NewInt(0)) != 0 {
		newTokenPoolValueToBuy++
	}
	if tokenPoolValueToBuy <= newTokenPoolValueToBuy {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			"refund",
			contentStr,
		}
		return [][]string{inst}, nil
	}

	receiveAmt := tokenPoolValueToBuy - newTokenPoolValueToBuy
	if pdeTradeReqAction.Meta.MinAcceptableAmount > receiveAmt {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			"refund",
			contentStr,
		}
		return [][]string{inst}, nil
	}

	// update current pde state on mem
	newTokenPoolValueToSell.Add(newTokenPoolValueToSell, big.NewInt(int64(fee)))
	pdePoolPair.Token1PoolValue = newTokenPoolValueToBuy
	pdePoolPair.Token2PoolValue = newTokenPoolValueToSell.Uint64()
	if pdePoolPair.Token1IDStr == pdeTradeReqAction.Meta.TokenIDToSellStr {
		pdePoolPair.Token1PoolValue = newTokenPoolValueToSell.Uint64()
		pdePoolPair.Token2PoolValue = newTokenPoolValueToBuy
	}

	pdeTradeAcceptedContent := metadata.PDETradeAcceptedContent{
		TraderAddressStr: pdeTradeReqAction.Meta.TraderAddressStr,
		TokenIDToBuyStr:  pdeTradeReqAction.Meta.TokenIDToBuyStr,
		ReceiveAmount:    receiveAmt,
		Token1IDStr:      pdePoolPair.Token1IDStr,
		Token2IDStr:      pdePoolPair.Token2IDStr,
		ShardID:          shardID,
		RequestedTxID:    pdeTradeReqAction.TxReqID,
	}
	pdeTradeAcceptedContent.Token1PoolValueOperation = metadata.TokenPoolValueOperation{
		Operator: "-",
		Value:    receiveAmt,
	}
	pdeTradeAcceptedContent.Token2PoolValueOperation = metadata.TokenPoolValueOperation{
		Operator: "+",
		Value:    pdeTradeReqAction.Meta.SellAmount + fee,
	}
	if pdePoolPair.Token1IDStr == pdeTradeReqAction.Meta.TokenIDToSellStr {
		pdeTradeAcceptedContent.Token1PoolValueOperation = metadata.TokenPoolValueOperation{
			Operator: "+",
			Value:    pdeTradeReqAction.Meta.SellAmount + fee,
		}
		pdeTradeAcceptedContent.Token2PoolValueOperation = metadata.TokenPoolValueOperation{
			Operator: "-",
			Value:    receiveAmt,
		}
	}
	pdeTradeAcceptedContentBytes, err := json.Marshal(pdeTradeAcceptedContent)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while marshaling pdeTradeAcceptedContent: %+v", err)
		return [][]string{}, nil
	}
	inst := []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		"accepted",
		string(pdeTradeAcceptedContentBytes),
	}
	return [][]string{inst}, nil
}

func buildPDEWithdrawalAcceptedInst(
	wdMeta metadata.PDEWithdrawalRequest,
	shardID byte,
	metaType int,
	withdrawalTokenIDStr string,
	deductingPoolValue uint64,
	deductingShares uint64,
	txReqID common.Hash,
) ([]string, error) {
	wdAcceptedContent := metadata.PDEWithdrawalAcceptedContent{
		WithdrawalTokenIDStr: withdrawalTokenIDStr,
		WithdrawerAddressStr: wdMeta.WithdrawerAddressStr,
		DeductingPoolValue:   deductingPoolValue,
		DeductingShares:      deductingShares,
		PairToken1IDStr:      wdMeta.WithdrawalToken1IDStr,
		PairToken2IDStr:      wdMeta.WithdrawalToken2IDStr,
		TxReqID:              txReqID,
		ShardID:              shardID,
	}
	wdAcceptedContentBytes, err := json.Marshal(wdAcceptedContent)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while marshaling PDEWithdrawalAcceptedContent: %+v", err)
		return []string{}, nil
	}
	return []string{
		strconv.Itoa(metaType),
		strconv.Itoa(int(shardID)),
		"accepted",
		string(wdAcceptedContentBytes),
	}, nil
}

func deductPDEAmountsV2(
	wdMeta metadata.PDEWithdrawalRequest,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
) *DeductingAmountsByWithdrawal {
	var deductingAmounts *DeductingAmountsByWithdrawal
	pairKey := string(lvdb.BuildPDEPoolForPairKey(
		beaconHeight,
		wdMeta.WithdrawalToken1IDStr, wdMeta.WithdrawalToken2IDStr,
	))
	pdePoolPair, found := currentPDEState.PDEPoolPairs[pairKey]
	if !found || pdePoolPair == nil {
		return deductingAmounts
	}
	shareForWithdrawerKey := string(lvdb.BuildPDESharesKeyV2(
		beaconHeight,
		wdMeta.WithdrawalToken1IDStr, wdMeta.WithdrawalToken2IDStr, wdMeta.WithdrawerAddressStr,
	))
	currentSharesForWithdrawer, found := currentPDEState.PDEShares[shareForWithdrawerKey]
	if !found || currentSharesForWithdrawer == 0 {
		return deductingAmounts
	}

	totalSharesForPairPrefix := string(lvdb.BuildPDESharesKeyV2(
		beaconHeight,
		wdMeta.WithdrawalToken1IDStr, wdMeta.WithdrawalToken2IDStr, "",
	))
	totalSharesForPair := uint64(0)
	for shareKey, shareAmt := range currentPDEState.PDEShares {
		if strings.Contains(shareKey, totalSharesForPairPrefix) {
			totalSharesForPair += shareAmt
		}
	}
	if totalSharesForPair == 0 {
		return deductingAmounts
	}
	wdSharesForWithdrawer := wdMeta.WithdrawalShareAmt
	if wdSharesForWithdrawer > currentSharesForWithdrawer {
		wdSharesForWithdrawer = currentSharesForWithdrawer
	}
	if wdSharesForWithdrawer == 0 {
		return deductingAmounts
	}

	deductingAmounts = &DeductingAmountsByWithdrawal{}
	deductingPoolValueToken1 := big.NewInt(0)
	deductingPoolValueToken1.Mul(big.NewInt(int64(pdePoolPair.Token1PoolValue)), big.NewInt(int64(wdSharesForWithdrawer)))
	deductingPoolValueToken1.Div(deductingPoolValueToken1, big.NewInt(int64(totalSharesForPair)))
	if pdePoolPair.Token1PoolValue < deductingPoolValueToken1.Uint64() {
		pdePoolPair.Token1PoolValue = 0
	} else {
		pdePoolPair.Token1PoolValue -= deductingPoolValueToken1.Uint64()
	}
	deductingAmounts.Token1IDStr = pdePoolPair.Token1IDStr
	deductingAmounts.PoolValue1 = deductingPoolValueToken1.Uint64()

	deductingPoolValueToken2 := big.NewInt(0)
	deductingPoolValueToken2.Mul(big.NewInt(int64(pdePoolPair.Token2PoolValue)), big.NewInt(int64(wdSharesForWithdrawer)))
	deductingPoolValueToken2.Div(deductingPoolValueToken2, big.NewInt(int64(totalSharesForPair)))
	if pdePoolPair.Token2PoolValue < deductingPoolValueToken2.Uint64() {
		pdePoolPair.Token2PoolValue = 0
	} else {
		pdePoolPair.Token2PoolValue -= deductingPoolValueToken2.Uint64()
	}
	deductingAmounts.Token2IDStr = pdePoolPair.Token2IDStr
	deductingAmounts.PoolValue2 = deductingPoolValueToken2.Uint64()

	if currentPDEState.PDEShares[shareForWithdrawerKey] < wdSharesForWithdrawer {
		currentPDEState.PDEShares[shareForWithdrawerKey] = 0
	} else {
		currentPDEState.PDEShares[shareForWithdrawerKey] -= wdSharesForWithdrawer
	}
	deductingAmounts.Shares = wdSharesForWithdrawer
	return deductingAmounts
}

func (blockchain *BlockChain) buildInstructionsForPDEWithdrawal(
	contentStr string,
	shardID byte,
	metaType int,
	currentPDEState *CurrentPDEState,
	beaconHeight uint64,
) ([][]string, error) {
	if currentPDEState == nil {
		Logger.log.Warn("WARN - [buildInstructionsForPDEWithdrawal]: Current PDE state is null.")
		return [][]string{}, nil
	}
	contentBytes, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while decoding content string of pde withdrawal action: %+v", err)
		return [][]string{}, nil
	}
	var pdeWithdrawalRequestAction metadata.PDEWithdrawalRequestAction
	err = json.Unmarshal(contentBytes, &pdeWithdrawalRequestAction)
	if err != nil {
		Logger.log.Errorf("ERROR: an error occured while unmarshaling pde withdrawal request action: %+v", err)
		return [][]string{}, nil
	}
	wdMeta := pdeWithdrawalRequestAction.Meta
	deductingAmounts := deductPDEAmountsV2(
		wdMeta,
		currentPDEState,
		beaconHeight,
	)

	if deductingAmounts == nil {
		inst := []string{
			strconv.Itoa(metaType),
			strconv.Itoa(int(shardID)),
			"rejected",
			contentStr,
		}
		return [][]string{inst}, nil
	}

	insts := [][]string{}
	inst1, err := buildPDEWithdrawalAcceptedInst(
		wdMeta,
		shardID,
		metaType,
		wdMeta.WithdrawalToken1IDStr,
		deductingAmounts.PoolValue1,
		deductingAmounts.Shares,
		pdeWithdrawalRequestAction.TxReqID,
	)
	if err != nil {
		return [][]string{}, nil
	}
	insts = append(insts, inst1)
	inst2, err := buildPDEWithdrawalAcceptedInst(
		wdMeta,
		shardID,
		metaType,
		wdMeta.WithdrawalToken2IDStr,
		deductingAmounts.PoolValue2,
		0,
		pdeWithdrawalRequestAction.TxReqID,
	)
	if err != nil {
		return [][]string{}, nil
	}
	insts = append(insts, inst2)
	return insts, nil
}
