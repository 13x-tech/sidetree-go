package sidetree

import (
	"fmt"
	"strings"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

func TestProcessorOptions(t *testing.T) {
	tests := map[string]struct {
		anchorString operations.AnchorString
		method       string
		cas          CAS
		filterIds    []string
		want         error
	}{
		"test valid": {
			anchorString: "1.abc",
			method:       "test",
			cas:          NewTestCAS(),
			filterIds:    []string{"did:sidetree:test"},
			want:         nil,
		},
		"empty method": {
			anchorString: "1.abc",
			method:       "",
			cas:          NewTestCAS(),
			filterIds:    []string{"did:sidetree:test"},
			want:         ErrInvalidMethod,
		},
		"nil cas": {
			anchorString: "1.abc",
			method:       "test",
			cas:          nil,
			filterIds:    []string{"did:sidetree:test"},
			want:         ErrInvalidCAS,
		},
		"empty uri": {
			anchorString: "abc",
			method:       "test",
			cas:          NewTestCAS(),
			filterIds:    []string{"did:sidetree:test"},
			want:         ErrEmptyURI,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Processor(
				operations.Anchor{Anchor: test.anchorString},
				WithDIDs(test.filterIds),
				WithPrefix(test.method),
				WithCAS(test.cas),
				WithFeeFunctions(
					BaseFeeAlgorithm(func(opCount int, anchorPoint string) int { return 0 }),
					PerOperationFee(func(baseFee int, opCount int, anchorPoint string) bool { return true }),
					ValueLocking(func(writerLockId string, baseFee int, opCount int, anchorPoint string) bool { return true }),
				),
			)
			if err != test.want && (test.want != nil && err != nil && !strings.Contains(err.Error(), test.want.Error())) {
				t.Errorf("expected error %v, got %v", test.want, err)
			}
		})
	}
}

func TestProcessorProcess(t *testing.T) {
	tests := map[string]struct {
		anchor       operations.Anchor
		feeFunctions []interface{}
		want         ProcessedOperations
		cas          CAS
	}{
		"without fee functions": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: nil,
			},
			cas: func() CAS {
				cas := NewTestCAS()
				cas.Put([]byte("{}"))
				return cas
			}(),
		},
		"index file found": {
			anchor: operations.Anchor{Anchor: "1.44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"},
			want: ProcessedOperations{
				Error: nil,
			},
			cas: func() CAS {
				cas := NewTestCAS()
				cas.Put([]byte("{}"))
				return cas
			}(),
		},
		"index file not found": {
			anchor: operations.Anchor{Anchor: "1.44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"},
			want: ProcessedOperations{
				Error: fmt.Errorf("no data found"),
			},
			cas: NewTestCAS(),
		},
	}

	for name, test := range tests {

		t.Run(name, func(t *testing.T) {
			p, err := Processor(
				test.anchor,
				WithCAS(test.cas),
				WithPrefix("test"),
				WithFeeFunctions(test.feeFunctions...),
			)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			got := p.Process()
			if got.Error != test.want.Error && (test.want.Error != nil && err != nil && !strings.Contains(got.Error.Error(), test.want.Error.Error())) {
				t.Fatalf("expected %v, got %v", test.want.Error, got.Error)
			}

		})
	}
}
