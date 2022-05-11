package security

// Signer is an interface for signing packets.
// Note: there is a mutual dependency problem for validator and signer:
// Validator -> Data -> Signer
// To solve this problem, as well as support different spec, we choose to have Data and Interest interface.
type Signer interface {
}

type Validator interface {
}
