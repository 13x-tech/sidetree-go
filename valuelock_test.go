package sidetree

import (
	"errors"
	"testing"
)

// feePerOpAt1000 documents the canonical mainnet-ish arithmetic used below:
// normalizedFee = 1000 sat -> feePerOp = 1000 * 0.001 = 1 sat ->
// lockAmountPerOp = 1 * 60000 = 60000 sat. So a lock of N*60000 sat permits N ops
// (floored to 100).
const normalizedFee1000 = 1000.0

func TestCalculateMaxNumberOfOperationsAllowed(t *testing.T) {
	tests := map[string]struct {
		lock          *ValueTimeLock
		normalizedFee float64
		want          int
	}{
		"no lock returns the free quota": {
			lock:          nil,
			normalizedFee: normalizedFee1000,
			want:          100,
		},
		"lock smaller than the free quota still returns 100": {
			// 60000 sat permits exactly 1 op, floored up to the 100 free quota.
			lock:          &ValueTimeLock{AmountLocked: 60000},
			normalizedFee: normalizedFee1000,
			want:          100,
		},
		"lock for exactly 100 ops returns 100": {
			lock:          &ValueTimeLock{AmountLocked: 100 * 60000},
			normalizedFee: normalizedFee1000,
			want:          100,
		},
		"lock for 200 ops returns 200": {
			lock:          &ValueTimeLock{AmountLocked: 200 * 60000},
			normalizedFee: normalizedFee1000,
			want:          200,
		},
		"allowance floors toward zero": {
			// 200*60000 + 59999 -> still only 200 whole ops.
			lock:          &ValueTimeLock{AmountLocked: 200*60000 + 59999},
			normalizedFee: normalizedFee1000,
			want:          200,
		},
		"higher normalized fee reduces the allowance": {
			// fee 2000 -> feePerOp 2 -> lockPerOp 120000; 200*60000=12,000,000 ->
			// 12,000,000/120000 = 100 ops.
			lock:          &ValueTimeLock{AmountLocked: 200 * 60000},
			normalizedFee: 2000,
			want:          100,
		},
		"non-positive fee falls back to the free quota": {
			lock:          &ValueTimeLock{AmountLocked: 1_000_000_000},
			normalizedFee: 0,
			want:          100,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := CalculateMaxNumberOfOperationsAllowed(test.lock, test.normalizedFee); got != test.want {
				t.Errorf("CalculateMaxNumberOfOperationsAllowed = %d, want %d", got, test.want)
			}
		})
	}
}

func TestVerifyLockAmount(t *testing.T) {
	// A lock owned by "writer" that permits 200 ops and is active for [100, 200).
	lock200 := &ValueTimeLock{
		AmountLocked:          200 * 60000,
		Owner:                 "writer",
		LockTransactionTime:   100,
		UnlockTransactionTime: 200,
	}

	tests := map[string]struct {
		lock       *ValueTimeLock
		opCount    int
		txWriter   string
		anchorTime int
		wantErr    error
	}{
		"at the free quota needs no lock": {
			lock:       nil,
			opCount:    100,
			txWriter:   "writer",
			anchorTime: 150,
			wantErr:    nil,
		},
		"over the free quota with no lock is rejected": {
			lock:       nil,
			opCount:    101,
			txWriter:   "writer",
			anchorTime: 150,
			wantErr:    ErrValueLockInsufficientForOps,
		},
		"over the free quota within a sufficient lock is allowed": {
			lock:       lock200,
			opCount:    200,
			txWriter:   "writer",
			anchorTime: 150,
			wantErr:    nil,
		},
		"over the lock allowance is rejected": {
			lock:       lock200,
			opCount:    201,
			txWriter:   "writer",
			anchorTime: 150,
			wantErr:    ErrValueLockInsufficientForOps,
		},
		"wrong owner is rejected": {
			lock:       lock200,
			opCount:    200,
			txWriter:   "someone-else",
			anchorTime: 150,
			wantErr:    ErrValueLockInvalidOwner,
		},
		"anchor before the lock window is rejected": {
			lock:       lock200,
			opCount:    200,
			txWriter:   "writer",
			anchorTime: 99,
			wantErr:    ErrValueLockTimeOutOfRange,
		},
		"anchor at the unlock height (exclusive) is rejected": {
			lock:       lock200,
			opCount:    200,
			txWriter:   "writer",
			anchorTime: 200,
			wantErr:    ErrValueLockTimeOutOfRange,
		},
		"anchor at the lock start (inclusive) is allowed": {
			lock:       lock200,
			opCount:    200,
			txWriter:   "writer",
			anchorTime: 100,
			wantErr:    nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := VerifyLockAmount(test.lock, test.opCount, normalizedFee1000, test.txWriter, test.anchorTime)
			if test.wantErr == nil {
				if err != nil {
					t.Errorf("expected nil, got %v", err)
				}
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Errorf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}
