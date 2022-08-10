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
