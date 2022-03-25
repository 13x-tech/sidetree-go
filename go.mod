module github.com/13x-tech/sidetree-go

go 1.17

//Replace Go-Jose to include secp256k1 curve from btcd/bcecc package
replace github.com/go-jose/go-jose/v3 v3.0.0 => github.com/13x-tech/go-jose/v3 v3.0.1-0.20220321223504-5b54fdf1b7df

require (
	github.com/go-jose/go-jose/v3 v3.0.0
	github.com/gowebpki/jcs v1.0.0
	github.com/multiformats/go-multihash v0.1.0
)

require (
	github.com/btcsuite/btcd/btcec/v2 v2.1.3 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/sys v0.0.0-20210309074719-68d13333faf2 // indirect
	lukechampine.com/blake3 v1.1.6 // indirect
)
