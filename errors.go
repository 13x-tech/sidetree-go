package sidetree

import (
	"errors"
	"fmt"
)

var (
	ErrURINotFound        = fmt.Errorf("URI not found")
	ErrDuplicateOperation = fmt.Errorf("duplicate operation")
	ErrNoCoreProof        = fmt.Errorf("core proof uri is empty")

	// Per-anchor operation-count limits (Sidetree protocol rule). A
	// spec-compliant ION node rejects the ENTIRE anchored batch — permanently,
	// never retried — when an anchor's operation count violates these, so they
	// are classified as ErrMalformed at the point of enforcement. See
	// docs/plans/2026-06-04-001-feat-ion-value-locking-protocol-rules-plan.md.

	// ErrInvalidOperationCount: the anchor string declares a non-positive or
	// unparseable operation count. AnchorString.Operations() returns 0 for both a
	// literal "0" and a non-numeric count, and a negative value parses through, so
	// any count < 1 is treated as a malformed anchor (ION rejects it at parse time).
	ErrInvalidOperationCount = fmt.Errorf("anchor declares a non-positive or unparseable operation count")

	// ErrTooManyOperations: the anchor declares more operations than
	// MaxOperationsPerBatch (the absolute ceiling; no value lock can exceed it).
	ErrTooManyOperations = fmt.Errorf("anchor operation count exceeds maxOperationsPerBatch")

	// ErrOperationLimitExceeded: the anchor declares more than
	// MaxNumberOfOperationsForNoValueTimeLock operations with no writer value lock.
	ErrOperationLimitExceeded = fmt.Errorf("anchor operation count exceeds maxNumberOfOperationsForNoValueTimeLock without a value-time-lock")

	// ErrUnverifiableValueLock: the anchor exceeds the free quota and presents a
	// writerLockId, but no value-lock verifier is configured to authorize it, so
	// the lock cannot be verified (no on-chain LockResolver / normalized fee yet).
	// Default-reject. ION mainnet runs with value-locking disabled, so no
	// canonical anchor reaches this branch today.
	ErrUnverifiableValueLock = fmt.Errorf("anchor exceeds the free operation quota with an unverifiable value lock")

	// ErrOperationCountMismatch: the anchored files contain more operations than
	// the anchor string declares — a writer must not understate the count to slip
	// past the operation-limit gate while packing more operations into the files.
	ErrOperationCountMismatch = fmt.Errorf("anchored operation count exceeds the anchor-string declared count")

	// ErrFileTooLarge: a Sidetree file's (decompressed) size exceeds the maximum
	// the protocol allows for its type (per-file cap × MaxMemoryDecompressionFactor).
	// The content is immutable, so this is a permanent rejection.
	ErrFileTooLarge = fmt.Errorf("sidetree file exceeds its maximum size")

	// ErrContentUnavailable marks a Sidetree-file fetch that failed because the
	// CAS could not return the content (IPFS timeout, not-found, peer
	// unreachable). The content may be published or become reachable later, so
	// the anchored operation is "not-yet-applied" rather than invalid — callers
	// should RECORD the anchor and RETRY. This is the failure mode late
	// publishing depends on (an anchor whose content surfaces later splices in
	// at its original txnum and can retroactively invalidate later operations).
	ErrContentUnavailable = errors.New("content unavailable")

	// ErrMalformed marks content that WAS retrieved but is structurally or
	// semantically invalid per the Sidetree spec (unparseable file, count
	// mismatch, duplicate operation in a batch, ...). Retrying cannot help —
	// IPFS content is immutable by CID — so callers should PERMANENTLY SKIP it.
	ErrMalformed = errors.New("malformed content")
)

// classifyFetch tags a CAS Get failure with its retryability class. A fetch
// failure is content-unavailable (retryable) by default: the content may
// publish or the CAS may reconnect later. A CAS that can prove the bytes are
// present-but-corrupt (e.g. gzip decompression failed) may itself wrap
// ErrMalformed, in which case the permanent-skip classification is preserved.
func classifyFetch(err error) error {
	if errors.Is(err, ErrMalformed) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrContentUnavailable, err)
}

// classifyMalformed tags content that was retrieved but is invalid per spec.
// The original error chain is preserved (double %w) so callers can still match
// the specific sentinel (ErrNoCoreProof, ErrDuplicateOperation, ...) alongside
// ErrMalformed. Already-classified errors pass through unchanged.
func classifyMalformed(err error) error {
	if errors.Is(err, ErrMalformed) || errors.Is(err, ErrContentUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrMalformed, err)
}
