package transaction

const (
	// TxVersion1 is the current latest supported transaction version.
	TxVersion1 = 1
	TxVersion2 = 2
)

const (
	CustomTokenInit = iota
	CustomTokenTransfer
	CustomTokenCrossShard
)

const (
	NormalCoinType = iota
	CustomTokenType
	CustomTokenPrivacyType
)

const MaxSizeInfo = 512
