module github.com/13x-tech/sidetree-go

go 1.17

replace github.com/go-jose/go-jose/v3 v3.0.0 => github.com/13x-tech/go-jose/v3 v3.0.1-0.20220321223504-5b54fdf1b7df

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1
	github.com/go-jose/go-jose/v3 v3.0.0
	github.com/gowebpki/jcs v1.0.0
	github.com/multiformats/go-multihash v0.2.0
)

require (
	github.com/13x-tech/ion-api-go v0.0.0-20220707201757-fa7ad1ff5e66
	github.com/13x-tech/ion-sdk-go v0.0.0-20220717020508-e36127a6a877
	github.com/btcsuite/btcd/btcec/v2 v2.1.3 // indirect
	github.com/klauspost/cpuid/v2 v2.0.14 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/stretchr/testify v1.7.1 // indirect
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d // indirect
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 // indirect
	lukechampine.com/blake3 v1.1.7 // indirect
)
