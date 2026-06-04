package sidetree

import (
	"fmt"
	"strings"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

func checkError(got error, want error) bool {
	if want == nil {
		return got == nil
	} else if got == nil {
		return false
	}
	return strings.Contains(got.Error(), want.Error())
}

func TestSideTreeOptions(t *testing.T) {
	tests := map[string]struct {
		method      string
		cas         CAS
		baseFeeFn   BaseFeeAlgorithm
		perOpFeeFn  PerOperationFee
		valueLockFn ValueLocking
	}{
		"test valid": {
			method:      "test",
			cas:         NewTestCAS(),
			baseFeeFn:   func(opCount int, anchorPoint string) int { return 0 },
			perOpFeeFn:  func(baseFee int, opCount int, anchorPoint string) bool { return true },
			valueLockFn: func(writerLockId string, baseFee int, opCount int, anchorPoint string) bool { return true },
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			st := New(
				WithDIDs([]string{"did:sidetree:test"}),
				WithPrefix(test.method),
				WithCAS(test.cas),
				WithFeeFunctions(test.baseFeeFn, test.perOpFeeFn, test.valueLockFn),
			)

			if st.method != test.method {
				t.Errorf("expected prefix %s, got %s", test.method, st.method)
			}

			if st.cas != test.cas {
				t.Errorf("expected cas to be %v, got %v", test.cas, st.cas)
			}

			if st.baseFeeFn(10, "abc") != test.baseFeeFn(10, "abc") {
				t.Errorf("expected base fee to be %d, got %d", test.baseFeeFn(10, "abc"), st.baseFeeFn(10, "abc"))
			}

			if st.perOpFeeFn(10, 10, "abc") != test.perOpFeeFn(10, 10, "abc") {
				t.Errorf("expected per op fee to be %t, got %t", test.perOpFeeFn(10, 10, "abc"), st.perOpFeeFn(10, 10, "abc"))
			}

			if st.valueLockFn("abc", 10, 10, "abc") != test.valueLockFn("abc", 10, 10, "abc") {
				t.Errorf("expected value lock to be %t, got %t", test.valueLockFn("abc", 10, 10, "abc"), st.valueLockFn("abc", 10, 10, "abc"))
			}

		})
	}
}

func TestSideTreeProcessOperations(t *testing.T) {

	tests := map[string]struct {
		sidetree *SideTree
		ops      []operations.Anchor
		ids      []string
		wantErr  error
		want     int
	}{
		"without ops": {
			sidetree: New(
				WithPrefix("test"),
				WithCAS(NewTestCAS()),
			),
			ops:     []operations.Anchor{},
			wantErr: nil,
			want:    0,
		},
		"with ops": {
			sidetree: New(
				WithPrefix("test"),
				WithCAS(NewTestCAS()),
			),
			ops: []operations.Anchor{{
				Sequence: "1:abc:1:abc",
				Anchor:   "2.abc",
			}, {
				Sequence: "2:def:1:xyz",
				Anchor:   "1.xyz",
			}},
			wantErr: nil,
			want:    2,
		},
		"empty cid": {
			sidetree: New(
				WithPrefix("test"),
				WithCAS(NewTestCAS()),
			),
			ops: []operations.Anchor{{
				Sequence: "1:abc:1:abc",
				Anchor:   "abc",
			}},
			wantErr: fmt.Errorf("failed to create operations processor"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			opMap, err := test.sidetree.ProcessOperations(test.ops, test.ids)
			if !checkError(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
			if len(opMap) != test.want {
				t.Errorf("expected %d operations, got %d", test.want, len(opMap))
			}
		})
	}
}

// TestProcessOperationsForwardsFeeFunctions guards the dropped-seam bug: a
// SideTree built WithFeeFunctions must forward those callbacks to every
// per-anchor Processor, so the per-operation-fee and value-lock checks actually
// fire during ProcessOperations. Before the fix the callbacks were silently
// dropped and a >100-op / unlocked anchor would be accepted regardless.
func TestProcessOperationsForwardsFeeFunctions(t *testing.T) {
	op := operations.Anchor{Sequence: "1:abc:1:abc", Anchor: "1.abc"}

	tests := map[string]struct {
		valueLock ValueLocking
		perOpFee  PerOperationFee
		wantErr   error
	}{
		"value lock rejects": {
			valueLock: func(writerLockId string, baseFee int, opCount int, anchorPoint string) bool { return false },
			wantErr:   fmt.Errorf("value lock is not valid"),
		},
		"per op fee rejects": {
			perOpFee: func(baseFee int, opCount int, anchorPoint string) bool { return false },
			wantErr:  fmt.Errorf("per op fee is not valid"),
		},
		"callbacks accept": {
			valueLock: func(writerLockId string, baseFee int, opCount int, anchorPoint string) bool { return true },
			perOpFee:  func(baseFee int, opCount int, anchorPoint string) bool { return true },
			wantErr:   nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cas := NewTestCAS()
			cas.insertObject("abc", []byte("{}")) // valid, empty core index file

			called := false
			opts := []SideTreeOption{WithPrefix("test"), WithCAS(cas)}
			var fns []interface{}
			if test.valueLock != nil {
				vl := test.valueLock
				fns = append(fns, ValueLocking(func(writerLockId string, baseFee int, opCount int, anchorPoint string) bool {
					called = true
					return vl(writerLockId, baseFee, opCount, anchorPoint)
				}))
			}
			if test.perOpFee != nil {
				po := test.perOpFee
				fns = append(fns, PerOperationFee(func(baseFee int, opCount int, anchorPoint string) bool {
					called = true
					return po(baseFee, opCount, anchorPoint)
				}))
			}
			opts = append(opts, WithFeeFunctions(fns...))

			st := New(opts...)
			opMap, err := st.ProcessOperations([]operations.Anchor{op}, nil)
			if err != nil {
				t.Fatalf("unexpected top-level error: %v", err)
			}
			if !called {
				t.Fatal("fee/value-lock callback was not forwarded to the per-anchor processor")
			}
			if !checkError(opMap[op].Error, test.wantErr) {
				t.Errorf("expected per-anchor error %v, got %v", test.wantErr, opMap[op].Error)
			}
		})
	}
}
