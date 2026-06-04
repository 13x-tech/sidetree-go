package sidetree

import (
	"fmt"
	"math"
)

// Value-time-lock verifier — a direct port of the Sidetree reference
// ValueTimeLockVerifier (decentralized-identity/sidetree v1.0.6). It is the
// protocol policy that decides how many operations an anchor may carry given a
// resolved on-chain value-time-lock and the block's normalized fee.
//
// sidetree-go owns this policy; the Bitcoin layer (ion-node) owns the data it
// needs — resolving the writerLockId to a ValueTimeLock (#55), tracking the
// per-block normalized fee (#54), and identifying the anchoring transaction's
// writer. ion-node plugs the policy in by implementing the ValueLocking callback
// (see sidetree.go) as a thin adapter that resolves those inputs and calls
// VerifyLockAmount. Until that resolver exists, the reader default-rejects
// over-quota locked anchors (see OperationsProcessor.checkOperationLimit), and
// ION mainnet runs with value-locking disabled so no canonical anchor needs it.

var (
	// ErrValueLockInvalidOwner: the lock's funds owner is not the anchoring
	// transaction's writer, so the writer may not use it.
	ErrValueLockInvalidOwner = fmt.Errorf("value-time-lock owner does not match the transaction writer")

	// ErrValueLockTimeOutOfRange: the anchor's block time is outside the lock's
	// active window [lockTransactionTime, unlockTransactionTime).
	ErrValueLockTimeOutOfRange = fmt.Errorf("anchor time is outside the value-time-lock window")

	// ErrValueLockInsufficientForOps: the anchor declares more operations than the
	// locked value (at the block's normalized fee) permits.
	ErrValueLockInsufficientForOps = fmt.Errorf("anchor operation count exceeds the value-time-lock allowance")
)

// ValueTimeLock is a resolved on-chain value-time-lock — the Bitcoin layer
// resolves a writerLockId to one of these (mirrors the reference
// ValueTimeLockModel). The lock is active for block times in
// [LockTransactionTime, UnlockTransactionTime).
type ValueTimeLock struct {
	// AmountLocked is the locked value in satoshis.
	AmountLocked int64
	// Owner identifies the locked-funds owner; it must equal the anchoring
	// transaction's writer for the lock to apply to that writer's anchors.
	Owner string
	// LockTransactionTime is the block height at which the lock starts (inclusive).
	LockTransactionTime int
	// UnlockTransactionTime is the block height at which the lock ends (exclusive).
	UnlockTransactionTime int
}

// CalculateMaxNumberOfOperationsAllowed returns the maximum number of operations
// an anchor may carry given a (possibly absent) value-time-lock and the block's
// normalized fee, porting the reference function of the same name:
//
//	allowed = floor(amountLocked / (normalizedFee * 0.001 * 60000)), floored to 100.
//
// With no lock the allowance is the free quota (MaxNumberOfOperationsForNoValue
// TimeLock = 100). The absolute ceiling MaxOperationsPerBatch (10000) is NOT
// applied here — it is enforced separately by the reader's op-count gate.
func CalculateMaxNumberOfOperationsAllowed(lock *ValueTimeLock, normalizedFee float64) int {
	if lock == nil {
		return MaxNumberOfOperationsForNoValueTimeLock
	}

	feePerOperation := normalizedFee * NormalizedFeeToPerOperationFeeMultiplier
	lockAmountPerOperation := feePerOperation * float64(ValueTimeLockAmountMultiplier)
	if lockAmountPerOperation <= 0 {
		// Defensive: a non-positive fee would make the division meaningless.
		// normalizedFee is always > 0 in practice (initialNormalizedFee = 1000),
		// so this only guards against invalid input; fall back to the free quota.
		return MaxNumberOfOperationsForNoValueTimeLock
	}

	numberOfOpsAllowed := int(math.Floor(float64(lock.AmountLocked) / lockAmountPerOperation))
	if numberOfOpsAllowed < MaxNumberOfOperationsForNoValueTimeLock {
		return MaxNumberOfOperationsForNoValueTimeLock
	}
	return numberOfOpsAllowed
}

// VerifyLockAmount verifies that an anchor declaring opCount operations is
// permitted, porting the reference verifyLockAmountAndThrowOnError. It returns
// nil when the anchor is allowed and a classified-free error otherwise (the
// caller wraps it with classifyMalformed):
//
//   - opCount <= 100: always allowed (the free quota needs no lock).
//   - opCount > 100 with a lock: the lock's owner must equal txWriter, the anchor
//     block time must fall in [LockTransactionTime, UnlockTransactionTime), and
//     opCount must not exceed CalculateMaxNumberOfOperationsAllowed.
//   - opCount > 100 with no lock: rejected (allowance is 100).
//
// anchorTime is the block height of the anchoring transaction. As in the
// reference, the owner/window checks are skipped when lock is nil; the final
// allowance check then rejects the over-quota anchor.
func VerifyLockAmount(lock *ValueTimeLock, opCount int, normalizedFee float64, txWriter string, anchorTime int) error {
	if opCount <= MaxNumberOfOperationsForNoValueTimeLock {
		return nil
	}

	if lock != nil {
		if lock.Owner != txWriter {
			return fmt.Errorf("%w: owner %q, writer %q", ErrValueLockInvalidOwner, lock.Owner, txWriter)
		}
		if anchorTime < lock.LockTransactionTime || anchorTime >= lock.UnlockTransactionTime {
			return fmt.Errorf("%w: anchor time %d not in [%d,%d)", ErrValueLockTimeOutOfRange, anchorTime, lock.LockTransactionTime, lock.UnlockTransactionTime)
		}
	}

	maxOps := CalculateMaxNumberOfOperationsAllowed(lock, normalizedFee)
	if opCount > maxOps {
		return fmt.Errorf("%w: %d > %d", ErrValueLockInsufficientForOps, opCount, maxOps)
	}
	return nil
}
