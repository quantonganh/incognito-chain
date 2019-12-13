package transaction

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
