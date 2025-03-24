package error

import (
	"fmt"
)

type Service int

const (
	Fetcher Service = iota
	Aggregator
	Reporter
	BootAPI
	Admin
	Por
	Dal
	Logscribe
	LogscribeConsumer
	Others
)

type ErrorCode int

const (
	NetworkError ErrorCode = iota
	DatabaseError
	InternalError
	InvalidInputError
	UnknownCaseError
	InvalidRaftMessageError
	InvalidBusMessageError
	MultipleError
)

var ServiceNames = map[Service]string{
	Fetcher:    "Fetcher",
	Aggregator: "Aggregator",
	Reporter:   "Reporter",
	BootAPI:    "BootAPI",
	Admin:      "Admin",
	Por:        "POR",
	Others:     "Others",
}

var ErrorCodes = map[ErrorCode]string{
	NetworkError:            "NetworkError",
	DatabaseError:           "DatabaseError",
	InternalError:           "InternalError",
	InvalidInputError:       "InvalidInputError",
	UnknownCaseError:        "UnknownCaseError",
	InvalidRaftMessageError: "InvalidRaftMessageError",
	InvalidBusMessageError:  "InvalidBusMessageError",
	MultipleError:           "MultipleError",
}

func (s Service) String() string {
	name, ok := ServiceNames[s]
	if !ok {
		return "UnknownService"
	}
	return name
}

func (e ErrorCode) String() string {
	code, ok := ErrorCodes[e]
	if !ok {
		return "UnknownErrorCode"
	}
	return code
}

type CustomError struct {
	Service Service
	Code    ErrorCode
	Message string
}

func (e *CustomError) Error() string {
	return fmt.Sprintf("Service: %s, Code: %s, Message: %s", e.Service, e.Code, e.Message)
}

var (
	ErrAdminDbPoolNotFound     = &CustomError{Service: Admin, Code: InternalError, Message: "db pool not found"}
	ErrAdminRedisConnNotFound  = &CustomError{Service: Admin, Code: InternalError, Message: "redisconn not found"}
	ErrAdminMessageBusNotFound = &CustomError{Service: Admin, Code: InternalError, Message: "messagebus not found"}

	ErrAggregatorInvalidInitValue         = &CustomError{Service: Aggregator, Code: InvalidInputError, Message: "Invalid init value parameters"}
	ErrAggregatorUnhandledCustomMessage   = &CustomError{Service: Aggregator, Code: UnknownCaseError, Message: "Unhandled custom message"}
	ErrAggregatorInvalidRaftMessage       = &CustomError{Service: Aggregator, Code: InvalidRaftMessageError, Message: "Invalid raft message"}
	ErrAggregatorNonLeaderRaftMessage     = &CustomError{Service: Aggregator, Code: InvalidRaftMessageError, Message: "Invalid raft message: message sent from non-leader"}
	ErrAggregatorGlobalAggregateInsertion = &CustomError{Service: Aggregator, Code: DatabaseError, Message: "Failed to insert global aggregator"}
	ErrAggregatorNotFound                 = &CustomError{Service: Aggregator, Code: InternalError, Message: "Aggregator not found"}
	ErrAggregatorCancelNotFound           = &CustomError{Service: Aggregator, Code: InternalError, Message: "Aggregator cancel function not found"}
	ErrAggregatorEmptyProof               = &CustomError{Service: Aggregator, Code: InternalError, Message: "Empty proof"}

	ErrBootAPIDbPoolNotFound = &CustomError{Service: BootAPI, Code: InternalError, Message: "db pool not found"}

	ErrBusChannelNotFound  = &CustomError{Service: Others, Code: InternalError, Message: "Channel not found"}
	ErrBusMsgPublishFail   = &CustomError{Service: Others, Code: InternalError, Message: "Failed to send message to channel"}
	ErrBusParamNotFound    = &CustomError{Service: Others, Code: InternalError, Message: "Param not found in message"}
	ErrBusConvertParamFail = &CustomError{Service: Others, Code: InternalError, Message: "Failed to convert message param"}
	ErrBusParseParamFail   = &CustomError{Service: Others, Code: InternalError, Message: "Failed to parse message param"}
	ErrBusNonAdmin         = &CustomError{Service: Others, Code: InvalidBusMessageError, Message: "Non-admin bus message"}
	ErrBusUnknownCommand   = &CustomError{Service: Others, Code: InvalidBusMessageError, Message: "Unknown command"}

	ErrChainTransactionFail                  = &CustomError{Service: Others, Code: InternalError, Message: "transaction failed"}
	ErrChainEmptyNameParam                   = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty name param"}
	ErrChainFailedToFindMethodSignatureMatch = &CustomError{Service: Others, Code: InternalError, Message: "failed to find method signature match"}
	ErrChainInvalidSignatureLength           = &CustomError{Service: Others, Code: InvalidInputError, Message: "invalid signature length"}
	ErrChainProviderUrlNotFound              = &CustomError{Service: Others, Code: InternalError, Message: "provider url not found"}
	ErrChainReporterUnsupportedChain         = &CustomError{Service: Others, Code: InvalidInputError, Message: "unsupported chain type"}
	ErrChainDelegatorUrlNotFound             = &CustomError{Service: Others, Code: InvalidInputError, Message: "delegator url not found"}
	ErrChainEmptySignedRawTx                 = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty signed raw tx"}
	ErrChainPubKeyToECDSAFail                = &CustomError{Service: Others, Code: InternalError, Message: "failed to convert public key to ECDSA"}
	ErrChainSignerPKNotFound                 = &CustomError{Service: Others, Code: InvalidInputError, Message: "signer public key not found"}
	ErrChainEmptyClientParam                 = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty client param"}
	ErrChainEmptyAddressParam                = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty address param"}
	ErrChainEmptyReporterParam               = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty reporter param"}
	ErrChainEmptyFuncStringParam             = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty function string param"}
	ErrChainEmptyEventStringParam            = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty event string param"}
	ErrChainEmptyChainIdParam                = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty chain id param"}
	ErrChainEmptyToAddress                   = &CustomError{Service: Others, Code: InvalidInputError, Message: "to address is empty"}
	ErrChainEmptyGasPrice                    = &CustomError{Service: Others, Code: InvalidInputError, Message: "gas price is empty"}
	ErrChainWebsocketContractAddressNotfound = &CustomError{Service: Others, Code: InvalidInputError, Message: "websocket contract address not found"}
	ErrChainWebsocketChannelNotfound         = &CustomError{Service: Others, Code: InvalidInputError, Message: "websocket channel not found"}
	ErrChainEmptyEventNameStringParam        = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty event name string param"}
	ErrChainWebsocketUrlNotProvided          = &CustomError{Service: Others, Code: InvalidInputError, Message: "websocket url not provided"}
	ErrChainSubmissionProxyContractNotFound  = &CustomError{Service: Others, Code: InvalidInputError, Message: "submission proxy contract not found"}
	ErrChainFailedToParseContractResult      = &CustomError{Service: Others, Code: InvalidInputError, Message: "failed to parse contract result"}
	ErrChainCachedAbiNotFound                = &CustomError{Service: Others, Code: InvalidInputError, Message: "cached abi not found"}

	ErrDbDatabaseUrlNotFound            = &CustomError{Service: Others, Code: InternalError, Message: "DATABASE_URL not found"}
	ErrDbEmptyTableNameParam            = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty table name"}
	ErrDbEmptyColumnNamesParam          = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty column names"}
	ErrDbEmptyWhereColumnsParam         = &CustomError{Service: Others, Code: InvalidInputError, Message: "empty where columns"}
	ErrDbWhereColumnValueLengthMismatch = &CustomError{Service: Others, Code: InvalidInputError, Message: "where column and value length mismatch"}
	ErrRdbHostNotFound                  = &CustomError{Service: Others, Code: InternalError, Message: "REDIS_HOST not found"}
	ErrRdbPortNotFound                  = &CustomError{Service: Others, Code: InternalError, Message: "REDIS_PORT not found"}
	ErrRediscriberRouterNotFound        = &CustomError{Service: Others, Code: InternalError, Message: "Rediscriber router not found"}

	ErrFetcherNotFound                        = &CustomError{Service: Fetcher, Code: InternalError, Message: "Fetcher not found"}
	ErrFetcherCancelNotFound                  = &CustomError{Service: Fetcher, Code: InternalError, Message: "Fetcher cancel function not found"}
	ErrFetcherInvalidType                     = &CustomError{Service: Fetcher, Code: InvalidInputError, Message: "Invalid fetcher type"}
	ErrFetcherNoDataFetched                   = &CustomError{Service: Fetcher, Code: InternalError, Message: "No data fetched"}
	ErrFetcherInvalidDexFetcherDefinition     = &CustomError{Service: Fetcher, Code: InvalidInputError, Message: "Invalid dex fetcher definition"}
	ErrFetcherChainHelperNotFound             = &CustomError{Service: Fetcher, Code: InternalError, Message: "Chain helper not found"}
	ErrFetcherInvalidRawResult                = &CustomError{Service: Fetcher, Code: InternalError, Message: "Invalid raw result"}
	ErrFetcherConvertToBigInt                 = &CustomError{Service: Fetcher, Code: InternalError, Message: "Failed to convert to big.Int"}
	ErrFetcherInvalidInput                    = &CustomError{Service: Fetcher, Code: InvalidInputError, Message: "Invalid input"}
	ErrFetcherDivisionByZero                  = &CustomError{Service: Fetcher, Code: InternalError, Message: "Division by zero"}
	ErrLocalAggregatorCancelNotFound          = &CustomError{Service: Fetcher, Code: InternalError, Message: "LocalAggregator cancel function not found"}
	ErrLocalAggregatorZeroVolume              = &CustomError{Service: Fetcher, Code: InvalidInputError, Message: "Zero volume"}
	ErrFeedDataBulkWriterCancelNotFound       = &CustomError{Service: Fetcher, Code: InternalError, Message: "FeedDataBulkWriter cancel function not found"}
	ErrLocalAggregateBulkWriterCancelNotFound = &CustomError{Service: Fetcher, Code: InternalError, Message: "LocalAggregateBulkWriter cancel function not found"}
	ErrFetcherNoMatchingChainID               = &CustomError{Service: Fetcher, Code: InvalidInputError, Message: "No matching chain ID"}
	ErrFetcherFailedToGetDexResultSlice       = &CustomError{Service: Fetcher, Code: InternalError, Message: "Failed to get dex result slice"}
	ErrFetcherFailedBigIntConvert             = &CustomError{Service: Fetcher, Code: InternalError, Message: "Failed to convert to fetched data to big.Int"}
	ErrFetcherFeedNotFound                    = &CustomError{Service: Fetcher, Code: InvalidInputError, Message: "Feed not found"}

	ErrLibP2pEmptyNonLocalAddress = &CustomError{Service: Others, Code: InternalError, Message: "Host has no non-local addresses"}
	ErrLibP2pAddressSplitFail     = &CustomError{Service: Others, Code: InternalError, Message: "Failed to split address"}
	ErrLibP2pFailToConnectPeer    = &CustomError{Service: Others, Code: InternalError, Message: "Failed to connect to peer"}

	ErrPorProviderUrlNotFound = &CustomError{Service: Por, Code: InternalError, Message: "POR_PROVIDER_URL not found"}
	ErrPorReporterPkNotFound  = &CustomError{Service: Por, Code: InvalidInputError, Message: "POR_REPORTER_PK not found"}
	ErrPorRawResultCastFail   = &CustomError{Service: Por, Code: InternalError, Message: "Failed to cast raw result to slice"}
	ErrPorRoundIdCastFail     = &CustomError{Service: Por, Code: InternalError, Message: "Failed to cast round id to int32"}
	ErrPorUpdatedAtCastFail   = &CustomError{Service: Por, Code: InternalError, Message: "Failed to cast updated at to big.Int"}
	ErrPorAnswerCastFail      = &CustomError{Service: Por, Code: InternalError, Message: "Failed to cast answer to big.Int"}
	ErrPorJobFail             = &CustomError{Service: Por, Code: InternalError, Message: "job failed"}

	ErrRaftLeaderIdMismatch = &CustomError{Service: Others, Code: InternalError, Message: "Leader id mismatch"}

	ErrReporterSubmissionProxyContractNotFound  = &CustomError{Service: Reporter, Code: InternalError, Message: "SUBMISSION_PROXY_CONTRACT not found"}
	ErrReporterNoReportersSet                   = &CustomError{Service: Reporter, Code: InternalError, Message: "No reporters set"}
	ErrReporterNotFound                         = &CustomError{Service: Reporter, Code: InternalError, Message: "Reporter not found"}
	ErrReporterAlreadyRunning                   = &CustomError{Service: Reporter, Code: InternalError, Message: "Reporter already running"}
	ErrReporterCancelNotFound                   = &CustomError{Service: Reporter, Code: InternalError, Message: "Reporter cancel function not found"}
	ErrReporterConfigNotFound                   = &CustomError{Service: Reporter, Code: InternalError, Message: "Reporter config not found"}
	ErrReporterEmptyConfigs                     = &CustomError{Service: Reporter, Code: InternalError, Message: "Empty configs"}
	ErrReporterJobFailed                        = &CustomError{Service: Reporter, Code: InternalError, Message: "Job failed"}
	ErrReporterReportFailed                     = &CustomError{Service: Reporter, Code: InternalError, Message: "Report failed"}
	ErrReporterProofNotFound                    = &CustomError{Service: Reporter, Code: InternalError, Message: "Proof not found"}
	ErrReporterUnknownMessageType               = &CustomError{Service: Reporter, Code: InvalidRaftMessageError, Message: "Unknown message type"}
	ErrReporterKaiaHelperNotFound               = &CustomError{Service: Reporter, Code: InternalError, Message: "Kaia helper not found"}
	ErrReporterDeviationReportFail              = &CustomError{Service: Reporter, Code: InternalError, Message: "Deviation report failed"}
	ErrReporterEmptyValidAggregates             = &CustomError{Service: Reporter, Code: InternalError, Message: "Empty valid aggregates"}
	ErrReporterEmptyAggregatesParam             = &CustomError{Service: Reporter, Code: InvalidInputError, Message: "Empty aggregates param"}
	ErrReporterEmptySubmissionPairsParam        = &CustomError{Service: Reporter, Code: InvalidInputError, Message: "Empty submission pairs param"}
	ErrReporterEmptyProofParam                  = &CustomError{Service: Reporter, Code: InvalidInputError, Message: "Empty proof param"}
	ErrReporterInvalidAggregateFound            = &CustomError{Service: Reporter, Code: InternalError, Message: "Invalid aggregate found"}
	ErrReporterMissingProof                     = &CustomError{Service: Reporter, Code: InternalError, Message: "Missing proof"}
	ErrReporterResultCastToInterfaceFail        = &CustomError{Service: Reporter, Code: InternalError, Message: "Failed to cast result to interface"}
	ErrReporterResultCastToAddressFail          = &CustomError{Service: Reporter, Code: InternalError, Message: "Failed to cast result to address"}
	ErrReporterSignerNotWhitelisted             = &CustomError{Service: Reporter, Code: InternalError, Message: "Signer not whitelisted"}
	ErrReporterEmptyValidProofs                 = &CustomError{Service: Reporter, Code: InternalError, Message: "Empty valid proofs"}
	ErrReporterInvalidProofLength               = &CustomError{Service: Reporter, Code: InvalidInputError, Message: "Invalid proof length"}
	ErrReporterBusMessageByNonAdmin             = &CustomError{Service: Reporter, Code: InvalidBusMessageError, Message: "Bus message sent by non-admin"}
	ErrReporterClear                            = &CustomError{Service: Reporter, Code: InternalError, Message: "Failed to clear reporters"}
	ErrReporterStart                            = &CustomError{Service: Reporter, Code: InternalError, Message: "Failed to start reporters"}
	ErrReporterStop                             = &CustomError{Service: Reporter, Code: InternalError, Message: "Failed to stop reporters"}
	ErrReporterValidateAggregateTimestampValues = &CustomError{Service: Reporter, Code: InternalError, Message: "Failed to validate aggregate timestamp values"}
	ErrReporterDalApiKeyNotFound                = &CustomError{Service: Reporter, Code: InternalError, Message: "DAL API key not found in reporter"}
	ErrReporterDalRestEndpointNotFound          = &CustomError{Service: Reporter, Code: InternalError, Message: "DAL REST endpoint not found in reporter"}
	ErrReporterDalWsDataProcessingFailed        = &CustomError{Service: Reporter, Code: InternalError, Message: "Failed to process DAL WS data"}

	ErrDalEmptyProofParam      = &CustomError{Service: Dal, Code: InvalidInputError, Message: "Empty proof param"}
	ErrDalInvalidProofLength   = &CustomError{Service: Dal, Code: InvalidInputError, Message: "Invalid proof length"}
	ErrDalSignerNotWhitelisted = &CustomError{Service: Dal, Code: InternalError, Message: "Signer not whitelisted"}
	ErrDalEmptyValidProofs     = &CustomError{Service: Dal, Code: InternalError, Message: "Empty valid proofs"}
	ErrDalPortNotFound         = &CustomError{Service: Dal, Code: InternalError, Message: "Port not found"}
	ErrDalCollectorNotFound    = &CustomError{Service: Dal, Code: InternalError, Message: "Collector not found"}
	ErrDalHubNotFound          = &CustomError{Service: Dal, Code: InternalError, Message: "Hub not found"}
	ErrDalKeyCacheNotFound     = &CustomError{Service: Dal, Code: InternalError, Message: "Key cache not found"}
	ErrDalFeedHashNotFound     = &CustomError{Service: Dal, Code: InternalError, Message: "Feed hash not found"}
	ErrDalSymbolsNotFound      = &CustomError{Service: Dal, Code: InternalError, Message: "Symbols not found"}
	ErrDalChainEnvNotFound     = &CustomError{Service: Dal, Code: InternalError, Message: "Chain env not found"}

	ErrReducerCastToFloatFail          = &CustomError{Service: Others, Code: InternalError, Message: "Failed to cast to float"}
	ErrReducerIndexCastToInterfaceFail = &CustomError{Service: Others, Code: InternalError, Message: "Failed to cast to interface from INDEX"}
	ErrReducerParseCastToInterfaceFail = &CustomError{Service: Others, Code: InternalError, Message: "Failed to cast to interface from PARSE"}
	ErrReducerParseCastToStringFail    = &CustomError{Service: Others, Code: InternalError, Message: "Failed to cast to string from PARSE"}
	ErrReducerParseCastToMapFail       = &CustomError{Service: Others, Code: InternalError, Message: "Failed to cast to map from PARSE"}
	ErrReducerMulCastToFloatFail       = &CustomError{Service: Others, Code: InternalError, Message: "Failed to cast to float from MUL"}
	ErrReducerDivCastToFloatFail       = &CustomError{Service: Others, Code: InternalError, Message: "Failed to cast to float from DIV"}
	ErrReducerDivDivsionByZero         = &CustomError{Service: Others, Code: InternalError, Message: "Division by zero from DIV"}
	ErrReducerDivFromCastToFloatFail   = &CustomError{Service: Others, Code: InternalError, Message: "Failed to cast to float from DIVFROM"}
	ErrReducerUnknownReducerFunc       = &CustomError{Service: Others, Code: InternalError, Message: "Unknown reducer function"}
	ErrRequestStatusNotOk              = &CustomError{Service: Others, Code: InternalError, Message: "Request status not OK"}
	ErrRequestInvalidMethod            = &CustomError{Service: Others, Code: InvalidInputError, Message: "Invalid method"}
	ErrCalculatorEmptyArr              = &CustomError{Service: Others, Code: InternalError, Message: "Empty array"}
	ErrReducerIndexOutOfBounds         = &CustomError{Service: Others, Code: InvalidInputError, Message: "Index out of bounds"}

	ErrLogTimestampNotExist = &CustomError{Service: Others, Code: InvalidInputError, Message: "Log timestamp not exist"}
	ErrLogMsgNotExist       = &CustomError{Service: Others, Code: InvalidInputError, Message: "Log message not exist"}
	ErrLogLvlNotExist       = &CustomError{Service: Others, Code: InvalidInputError, Message: "Log level not exist"}
	ErrLogEmptyLogByte      = &CustomError{Service: Others, Code: InvalidInputError, Message: "Empty log byte"}

	ErrLogscribeDbPoolNotFound            = &CustomError{Service: Logscribe, Code: InternalError, Message: "db pool not found"}
	ErrLogscribeInsertFailed              = &CustomError{Service: Logscribe, Code: InternalError, Message: "Failed to insert logs to Logscribe"}
	ErrLogscribeServiceNotExist           = &CustomError{Service: Logscribe, Code: InvalidInputError, Message: "Service field not found in log entry"}
	ErrLogscribeGithubCredentialsNotFound = &CustomError{Service: Logscribe, Code: InternalError, Message: "Github credentials not found"}
	ErrLogscribeFailedToCreateGithubIssue = &CustomError{Service: Logscribe, Code: InternalError, Message: "Failed to create github issue"}

	ErrLogscribeConsumerServiceNotProvided   = &CustomError{Service: LogscribeConsumer, Code: InvalidInputError, Message: "Service field not provided in logscribeconsumer"}
	ErrLogscribeConsumerInvalidLevel         = &CustomError{Service: LogscribeConsumer, Code: InvalidInputError, Message: "Invalid log level provided to logscribeconsumer"}
	ErrLogscribeConsumerEndpointUnresponsive = &CustomError{Service: LogscribeConsumer, Code: NetworkError, Message: "Logscribe endpoint unresponsive"}

	ErrConditionTimedOut = &CustomError{Service: Others, Code: InternalError, Message: "Condition timed out"}
)
