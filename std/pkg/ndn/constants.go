package ndn

// ContentType represents the type of Data content in MetaInfo.
type ContentType uint

const (
	ContentTypeBlob ContentType = 0
	ContentTypeLink ContentType = 1
	ContentTypeKey  ContentType = 2
	ContentTypeNack ContentType = 3
)

// SigType represents the type of signature.
type SigType int

const (
	SignatureNone            SigType = -1
	SignatureDigestSha256    SigType = 0
	SignatureSha256WithRsa   SigType = 1
	SignatureSha256WithEcdsa SigType = 3
	SignatureHmacWithSha256  SigType = 4
	SignatureEd25519         SigType = 5
	SignatureEmptyTest       SigType = 200
)

// InterestResult represents the result of Interest expression.
// Can be Data fetched (succeeded), NetworkNack received, or Timeout.
// Note that AppNack is considered as Data.
type InterestResult int

const (
	// Empty result. Not used by the engine.
	// Used by high-level part if the setting to construct an Interest is incorrect.
	InterestResultNone InterestResult = iota
	// Data is fetched
	InterestResultData
	// NetworkNack is received
	InterestResultNack
	// Timeout
	InterestResultTimeout
	// Cancelled due to disconnection
	InterestCancelled
	// Failed of validation. Not used by the engine itself.
	InterestResultUnverified
	// Other error happens during handling the fetched data. Not used by the engine itself.
	InterestResultError
)
