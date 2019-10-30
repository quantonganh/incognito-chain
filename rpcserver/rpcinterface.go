package rpcserver

import "github.com/incognitochain/incognito-chain/rpcserver/rpcservice"

type httpHandler func(*HttpServer, interface{}, <-chan struct{}) (interface{}, *rpcservice.RPCError)
type wsHandler func(*WsServer, interface{}, string, chan RpcSubResult, <-chan struct{})

// Commands valid for normal user
var HttpHandler = map[string]httpHandler{
	//Test Rpc Server
	testHttpServer: (*HttpServer).handleTestHttpServer,

	//profiling
	startProfiling: (*HttpServer).handleStartProfiling,
	stopProfiling:  (*HttpServer).handleStopProfiling,

	// node
	getNodeRole:              (*HttpServer).handleGetNodeRole,
	getNetworkInfo:           (*HttpServer).handleGetNetWorkInfo,
	getConnectionCount:       (*HttpServer).handleGetConnectionCount,
	getAllConnectedPeers:     (*HttpServer).handleGetAllConnectedPeers,
	getInOutMessages:         (*HttpServer).handleGetInOutMessages,
	getInOutMessageCount:     (*HttpServer).handleGetInOutMessageCount,
	getAllPeers:              (*HttpServer).handleGetAllPeers,
	estimateFee:              (*HttpServer).handleEstimateFee,
	estimateFeeWithEstimator: (*HttpServer).handleEstimateFeeWithEstimator,
	getActiveShards:          (*HttpServer).handleGetActiveShards,
	getMaxShardsNumber:       (*HttpServer).handleGetMaxShardsNumber,

	//tx pool
	getRawMempool:           (*HttpServer).handleGetRawMempool,
	getNumberOfTxsInMempool: (*HttpServer).handleGetNumberOfTxsInMempool,
	getMempoolEntry:         (*HttpServer).handleMempoolEntry,
	removeTxInMempool:       (*HttpServer).handleRemoveTxInMempool,
	getMempoolInfo:          (*HttpServer).handleGetMempoolInfo,
	getPendingTxsInBlockgen: (*HttpServer).handleGetPendingTxsInBlockgen,

	// block pool ver.2
	getShardToBeaconPoolStateV2: (*HttpServer).handleGetShardToBeaconPoolStateV2,
	getCrossShardPoolStateV2:    (*HttpServer).handleGetCrossShardPoolStateV2,
	getShardPoolStateV2:         (*HttpServer).handleGetShardPoolStateV2,
	getBeaconPoolStateV2:        (*HttpServer).handleGetBeaconPoolStateV2,
	// ver.1
	//getShardToBeaconPoolState: (*HttpServer).handleGetShardToBeaconPoolState,
	//getCrossShardPoolState:    (*HttpServer).handleGetCrossShardPoolState,
	getNextCrossShard: (*HttpServer).handleGetNextCrossShard,

	// block
	getBestBlock:        (*HttpServer).handleGetBestBlock,
	getBestBlockHash:    (*HttpServer).handleGetBestBlockHash,
	retrieveBlock:       (*HttpServer).handleRetrieveBlock,
	retrieveBeaconBlock: (*HttpServer).handleRetrieveBeaconBlock,
	getBlocks:           (*HttpServer).handleGetBlocks,
	getBlockChainInfo:   (*HttpServer).handleGetBlockChainInfo,
	getBlockCount:       (*HttpServer).handleGetBlockCount,
	getBlockHash:        (*HttpServer).handleGetBlockHash,
	checkHashValue:      (*HttpServer).handleCheckHashValue, // get data in blockchain from hash value
	getBlockHeader:      (*HttpServer).handleGetBlockHeader, // Current committee, next block committee and candidate is included in block header
	getCrossShardBlock:  (*HttpServer).handleGetCrossShardBlock,

	// transaction
	listOutputCoins:                         (*HttpServer).handleListOutputCoins,
	createRawTransaction:                    (*HttpServer).handleCreateRawTransaction,
	sendRawTransaction:                      (*HttpServer).handleSendRawTransaction,
	createAndSendTransaction:                (*HttpServer).handleCreateAndSendTx,
	getTransactionByHash:                    (*HttpServer).handleGetTransactionByHash,
	gettransactionhashbyreceiver:            (*HttpServer).handleGetTransactionHashByReceiver,
	createAndSendStakingTransaction:         (*HttpServer).handleCreateAndSendStakingTx,
	createAndSendStopAutoStakingTransaction: (*HttpServer).handleCreateAndSendStopAutoStakingTransaction,
	randomCommitments:                       (*HttpServer).handleRandomCommitments,
	hasSerialNumbers:                        (*HttpServer).handleHasSerialNumbers,
	hasSnDerivators:                         (*HttpServer).handleHasSnDerivators,
	listSerialNumbers:                       (*HttpServer).handleListSerialNumbers,
	listCommitments:                         (*HttpServer).handleListCommitments,
	listCommitmentIndices:                   (*HttpServer).handleListCommitmentIndices,

	//======Testing and Benchmark======
	getAndSendTxsFromFile:   (*HttpServer).handleGetAndSendTxsFromFile,
	getAndSendTxsFromFileV2: (*HttpServer).handleGetAndSendTxsFromFileV2,
	unlockMempool:           (*HttpServer).handleUnlockMempool,
	//=================================

	// Beststate
	getCandidateList:              (*HttpServer).handleGetCandidateList,
	getCommitteeList:              (*HttpServer).handleGetCommitteeList,
	getShardBestState:             (*HttpServer).handleGetShardBestState,
	getShardBestStateDetail:       (*HttpServer).handleGetShardBestStateDetail,
	getBeaconBestState:            (*HttpServer).handleGetBeaconBestState,
	getBeaconBestStateDetail:      (*HttpServer).handleGetBeaconBestStateDetail,
	getBeaconPoolState:            (*HttpServer).handleGetBeaconPoolState,
	getShardPoolState:             (*HttpServer).handleGetShardPoolState,
	getShardPoolLatestValidHeight: (*HttpServer).handleGetShardPoolLatestValidHeight,
	canPubkeyStake:                (*HttpServer).handleCanPubkeyStake,
	getTotalTransaction:           (*HttpServer).handleGetTotalTransaction,

	// custom token
	createRawCustomTokenTransaction:     (*HttpServer).handleCreateRawCustomTokenTransaction,
	sendRawCustomTokenTransaction:       (*HttpServer).handleSendRawCustomTokenTransaction,
	createAndSendCustomTokenTransaction: (*HttpServer).handleCreateAndSendCustomTokenTransaction,
	listUnspentCustomToken:              (*HttpServer).handleListUnspentCustomToken,
	getBalanceCustomToken:               (*HttpServer).handleGetBalanceCustomToken,
	listCustomToken:                     (*HttpServer).handleListCustomToken,
	customTokenTxs:                      (*HttpServer).handleCustomTokenDetail,
	listCustomTokenHolders:              (*HttpServer).handleGetListCustomTokenHolders,
	getListCustomTokenBalance:           (*HttpServer).handleGetListCustomTokenBalance,

	// custom token which support privacy
	createRawPrivacyCustomTokenTransaction:     (*HttpServer).handleCreateRawPrivacyCustomTokenTransaction,
	sendRawPrivacyCustomTokenTransaction:       (*HttpServer).handleSendRawPrivacyCustomTokenTransaction,
	createAndSendPrivacyCustomTokenTransaction: (*HttpServer).handleCreateAndSendPrivacyCustomTokenTransaction,
	listPrivacyCustomToken:                     (*HttpServer).handleListPrivacyCustomToken,
	privacyCustomTokenTxs:                      (*HttpServer).handlePrivacyCustomTokenDetail,
	getListPrivacyCustomTokenBalance:           (*HttpServer).handleGetListPrivacyCustomTokenBalance,
	getBalancePrivacyCustomToken:               (*HttpServer).handleGetBalancePrivacyCustomToken,

	// Bridge
	createIssuingRequest:            (*HttpServer).handleCreateIssuingRequest,
	sendIssuingRequest:              (*HttpServer).handleSendIssuingRequest,
	createAndSendIssuingRequest:     (*HttpServer).handleCreateAndSendIssuingRequest,
	createAndSendContractingRequest: (*HttpServer).handleCreateAndSendContractingRequest,
	checkETHHashIssued:              (*HttpServer).handleCheckETHHashIssued,
	getAllBridgeTokens:              (*HttpServer).handleGetAllBridgeTokens,
	getETHHeaderByHash:              (*HttpServer).handleGetETHHeaderByHash,
	getBridgeReqWithStatus:          (*HttpServer).handleGetBridgeReqWithStatus,
	generateTokenID:                 (*HttpServer).handleGenerateTokenID,

	// wallet
	getPublicKeyFromPaymentAddress:   (*HttpServer).handleGetPublicKeyFromPaymentAddress,
	defragmentAccount:                (*HttpServer).handleDefragmentAccount,
	getStackingAmount:                (*HttpServer).handleGetStakingAmount,
	hashToIdenticon:                  (*HttpServer).handleHashToIdenticon,
	createAndSendBurningRequest:      (*HttpServer).handleCreateAndSendBurningRequest,
	createAndSendTxWithIssuingETHReq: (*HttpServer).handleCreateAndSendTxWithIssuingETHReq,

	// Incognito -> Ethereum bridge
	getBeaconSwapProof:       (*HttpServer).handleGetBeaconSwapProof,
	getLatestBeaconSwapProof: (*HttpServer).handleGetLatestBeaconSwapProof,
	getBridgeSwapProof:       (*HttpServer).handleGetBridgeSwapProof,
	getLatestBridgeSwapProof: (*HttpServer).handleGetLatestBridgeSwapProof,
	getBurnProof:             (*HttpServer).handleGetBurnProof,

	//reward
	CreateRawWithDrawTransaction: (*HttpServer).handleCreateAndSendWithDrawTransaction,
	getRewardAmount:              (*HttpServer).handleGetRewardAmount,
	listRewardAmount:             (*HttpServer).handleListRewardAmount,

	// revert
	revertbeaconchain: (*HttpServer).handleRevertBeacon,
	revertshardchain:  (*HttpServer).handleRevertShard,

	// mining info
	getMiningInfo:               (*HttpServer).handleGetMiningInfo,
	enableMining:                (*HttpServer).handleEnableMining,
	getChainMiningStatus:        (*HttpServer).handleGetChainMiningStatus,
	getPublickeyMining:          (*HttpServer).handleGetPublicKeyMining,
	getPublicKeyRole:            (*HttpServer).handleGetPublicKeyRole,
	getIncognitoPublicKeyRole:   (*HttpServer).handleGetIncognitoPublicKeyRole,
	getMinerRewardFromMiningKey: (*HttpServer).handleGetMinerRewardFromMiningKey,
	getProducersBlackList:       (*HttpServer).handleGetProducersBlackList,
	getProducersBlackListDetail: (*HttpServer).handleGetProducersBlackListDetail,

	// pde
	getPDEState:                           (*HttpServer).handleGetPDEState,
	createAndSendTxWithWithdrawalReq:      (*HttpServer).handleCreateAndSendTxWithWithdrawalReq,
	createAndSendTxWithPTokenTradeReq:     (*HttpServer).handleCreateAndSendTxWithPTokenTradeReq,
	createAndSendTxWithPRVTradeReq:        (*HttpServer).handleCreateAndSendTxWithPRVTradeReq,
	createAndSendTxWithPTokenContribution: (*HttpServer).handleCreateAndSendTxWithPTokenContribution,
	createAndSendTxWithPRVContribution:    (*HttpServer).handleCreateAndSendTxWithPRVContribution,
}

// Commands that are available to a limited user
var LimitedHttpHandler = map[string]httpHandler{
	// local WALLET
	listAccounts:               (*HttpServer).handleListAccounts,
	getAccount:                 (*HttpServer).handleGetAccount,
	getAddressesByAccount:      (*HttpServer).handleGetAddressesByAccount,
	getAccountAddress:          (*HttpServer).handleGetAccountAddress,
	dumpPrivkey:                (*HttpServer).handleDumpPrivkey,
	importAccount:              (*HttpServer).handleImportAccount,
	removeAccount:              (*HttpServer).handleRemoveAccount,
	listUnspentOutputCoins:     (*HttpServer).handleListUnspentOutputCoins,
	getBalance:                 (*HttpServer).handleGetBalance,
	getBalanceByPrivatekey:     (*HttpServer).handleGetBalanceByPrivatekey,
	getBalanceByPaymentAddress: (*HttpServer).handleGetBalanceByPaymentAddress,
	getReceivedByAccount:       (*HttpServer).handleGetReceivedByAccount,
	setTxFee:                   (*HttpServer).handleSetTxFee,
}

var WsHandler = map[string]wsHandler{
	testSubcrice:                                (*WsServer).handleTestSubcribe,
	subcribeNewShardBlock:                       (*WsServer).handleSubscribeNewShardBlock,
	subcribeNewBeaconBlock:                      (*WsServer).handleSubscribeNewBeaconBlock,
	subcribePendingTransaction:                  (*WsServer).handleSubscribePendingTransaction,
	subcribeShardCandidateByPublickey:           (*WsServer).handleSubcribeShardCandidateByPublickey,
	subcribeShardCommitteeByPublickey:           (*WsServer).handleSubcribeShardCommitteeByPublickey,
	subcribeShardPendingValidatorByPublickey:    (*WsServer).handleSubcribeShardPendingValidatorByPublickey,
	subcribeBeaconCandidateByPublickey:          (*WsServer).handleSubcribeBeaconCandidateByPublickey,
	subcribeBeaconPendingValidatorByPublickey:   (*WsServer).handleSubcribeBeaconPendingValidatorByPublickey,
	subcribeBeaconCommitteeByPublickey:          (*WsServer).handleSubcribeBeaconCommitteeByPublickey,
	subcribeMempoolInfo:                         (*WsServer).handleSubcribeMempoolInfo,
	subcribeCrossOutputCoinByPrivateKey:         (*WsServer).handleSubcribeCrossOutputCoinByPrivateKey,
	subcribeCrossCustomTokenByPrivateKey:        (*WsServer).handleSubcribeCrossCustomTokenByPrivateKey,
	subcribeCrossCustomTokenPrivacyByPrivateKey: (*WsServer).handleSubcribeCrossCustomTokenPrivacyByPrivateKey,
	subcribeShardBestState:                      (*WsServer).handleSubscribeShardBestState,
	subcribeBeaconBestState:                     (*WsServer).handleSubscribeBeaconBestState,
	subcribeBeaconPoolBeststate:                 (*WsServer).handleSubscribeBeaconPoolBestState,
	subcribeShardPoolBeststate:                  (*WsServer).handleSubscribeShardPoolBeststate,
}
