package sidetree

// Canonical Sidetree v1 protocol parameters. Values are verbatim from the
// reference implementation, decentralized-identity/sidetree tag v1.0.6:
// lib/core/versions/1.0/protocol-parameters.json (and Operation.ts for the
// reveal-value length, and the spec for the decompression factor). They are the
// single source of truth for both the reader (this package) and writers that
// must not emit anchors a compliant reader will reject.
//
// These are protocol-version-scoped and live here in the core. Network-specific
// Bitcoin parameters (valueTimeLockDurationInBlocks, the normalized-fee seed /
// look-back / fluctuation) are NOT here — they belong to the Bitcoin wrapper
// (ion-node config), per the ION reference layering.
const (
	// MaxOperationsPerBatch is the hard ceiling on operations in one anchor,
	// regardless of any value lock. An anchor whose anchor-string operation count
	// exceeds this is rejected outright (MAX_OPERATION_COUNT).
	MaxOperationsPerBatch = 10000

	// MaxNumberOfOperationsForNoValueTimeLock is the maximum operations an anchor
	// may carry with NO value-time-lock. This is the consensus-critical default:
	// ION mainnet runs with value-locking disabled, so every canonical anchor is
	// capped here. An anchor with more operations and no writer lock is rejected.
	MaxNumberOfOperationsForNoValueTimeLock = 100

	// NormalizedFeeToPerOperationFeeMultiplier converts a block's normalized fee
	// into the satoshi cost attributed to a single operation:
	// feePerOperation = normalizedFee * NormalizedFeeToPerOperationFeeMultiplier.
	NormalizedFeeToPerOperationFeeMultiplier = 0.001

	// ValueTimeLockAmountMultiplier is how much locked value is required per
	// operation above the free quota:
	// requiredLock = normalizedFee * NormalizedFeeToPerOperationFeeMultiplier * ops * ValueTimeLockAmountMultiplier.
	// Inverted by the reader: opsAllowed = floor(amountLocked / (normalizedFee * 0.001 * 60000)).
	ValueTimeLockAmountMultiplier = 60000

	// MaxDeltaSizeInBytes caps a single operation's canonicalized delta. This is
	// the de-facto per-operation size bound; Sidetree v1 has no separate
	// maxOperationSize constant.
	MaxDeltaSizeInBytes = 1000

	// MaxCASURILength caps the byte length of any CAS/IPFS URI in an index or
	// proof file (also the spec's MAX_OPERATION_HASH_LENGTH).
	MaxCASURILength = 100

	// MaxEncodedRevealValueLength caps an operation's encoded reveal value
	// (from Operation.ts; not present in protocol-parameters.json).
	MaxEncodedRevealValueLength = 50

	// MaxWriterLockIDInBytes caps the writerLockId string in the core index file.
	MaxWriterLockIDInBytes = 200

	// Bounded-download caps for each Sidetree file (a download exceeding its cap is
	// rejected). MaxProofFileSizeInBytes is SHARED by the core proof and the
	// provisional proof file.
	MaxCoreIndexFileSizeInBytes        = 1000000
	MaxProvisionalIndexFileSizeInBytes = 1000000
	MaxProofFileSizeInBytes            = 2500000
	MaxChunkFileSizeInBytes            = 10000000

	// MaxMemoryDecompressionFactor bounds gzip expansion: a decompressed file may
	// be at most this many times its on-disk (compressed) size (zip-bomb guard).
	// Spec-level (MAX_MEMORY_DECOMPRESSION_FACTOR), not in protocol-parameters.json.
	MaxMemoryDecompressionFactor = 3

	// Per-blockchain-time (per-block) aggregate ceilings the observer enforces
	// across all Sidetree transactions at one transaction time.
	MaxNumberOfTransactionsPerTransactionTime = 300
	MaxNumberOfOperationsPerTransactionTime   = 600000

	// SHA256MultihashCode (0x12) is the only multihash code allowed for Sidetree
	// hashes, commitments, and reveal values (hashAlgorithmsInMultihashCode).
	SHA256MultihashCode = 18
)
