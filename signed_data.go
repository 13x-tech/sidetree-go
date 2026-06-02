package sidetree

// The proof files carry the compact-JWS signed data for recover/update/deactivate
// operations. The Sidetree spec and reference ION serialize this field as
// "signedData" (lowercase). Without the explicit json tag Go (un)marshals it as
// "SignedData", so real ION proof files would decode with an empty SignedData
// (silently skipping signature verification) and files written by this library
// would be non-interoperable. The explicit tags make both directions spec-correct.

type SignedUpdateDataOp struct {
	SignedData string `json:"signedData"`
}

type SignedRecoverDataOp struct {
	SignedData string `json:"signedData"`
}

type SignedDeactivateDataOp struct {
	SignedData string `json:"signedData"`
}
