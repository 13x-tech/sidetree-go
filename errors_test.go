package sidetree

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

// malformedCAS is a CAS whose Get always fails with an ErrMalformed-wrapped
// error, simulating a CAS that retrieved bytes but found them corrupt (e.g. a
// gzip decompression failure). It exercises the classifyFetch passthrough.
type malformedCAS struct{}

func (malformedCAS) Close() error               { return nil }
func (malformedCAS) Start() error               { return nil }
func (malformedCAS) Type() CASType              { return CASType("malformed") }
func (malformedCAS) Put([]byte) (string, error) { return "", nil }
func (malformedCAS) Get(id string, maxSizeInBytes int) ([]byte, error) {
	return nil, fmt.Errorf("corrupt object %s: %w", id, ErrMalformed)
}

// TestProcessErrorClassification verifies that ProcessedOperations.Error carries
// the right retryability class: a CAS fetch failure is ErrContentUnavailable
// (retry — the late-publishing case), while fetched-but-invalid content is
// ErrMalformed (permanent skip). The two classes are mutually exclusive, and
// malformed errors preserve their specific underlying sentinel.
func TestProcessErrorClassification(t *testing.T) {
	tests := map[string]struct {
		anchor         operations.Anchor
		cas            CAS
		wantClass      error // ErrContentUnavailable or ErrMalformed
		notClass       error // the class that must NOT match
		wantUnderlying error // optional: a specific sentinel that must remain matchable
	}{
		"missing core index is unavailable": {
			anchor:    operations.Anchor{Anchor: "1.missing"},
			cas:       NewTestCAS(),
			wantClass: ErrContentUnavailable,
			notClass:  ErrMalformed,
		},
		"unparseable core index is malformed": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			cas: func() CAS {
				c := NewTestCAS()
				c.insertObject("abc", []byte("not json"))
				return c
			}(),
			wantClass: ErrMalformed,
			notClass:  ErrContentUnavailable,
		},
		"process validation error is malformed and keeps its sentinel": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			cas: func() CAS {
				c := NewTestCAS()
				// A recover op with no core-proof URI -> coreIndexFile.Process
				// returns ErrNoCoreProof (validation, not a fetch failure).
				ci := CoreIndexFile{Operations: CoreOperations{
					Recover: []Operation{{DIDSuffix: "did:abc:123", RevealValue: "r"}},
				}}
				b, _ := json.Marshal(ci)
				c.insertObject("abc", b)
				return c
			}(),
			wantClass:      ErrMalformed,
			notClass:       ErrContentUnavailable,
			wantUnderlying: ErrNoCoreProof,
		},
		"missing provisional index is unavailable": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			cas: func() CAS {
				c := NewTestCAS()
				ci := CoreIndexFile{ProvisionalIndexURI: "gone", Operations: CoreOperations{}}
				b, _ := json.Marshal(ci)
				c.insertObject("abc", b)
				return c
			}(),
			wantClass: ErrContentUnavailable,
			notClass:  ErrMalformed,
		},
		"CAS-signalled corruption stays malformed (no retry)": {
			anchor:    operations.Anchor{Anchor: "1.whatever"},
			cas:       malformedCAS{},
			wantClass: ErrMalformed,
			notClass:  ErrContentUnavailable,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p, err := Processor(test.anchor, WithCAS(test.cas), WithPrefix("test"))
			if err != nil {
				t.Fatalf("Processor: %v", err)
			}
			got := p.Process()
			if got.Error == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !errors.Is(got.Error, test.wantClass) {
				t.Errorf("error %q is not classified %v", got.Error, test.wantClass)
			}
			if errors.Is(got.Error, test.notClass) {
				t.Errorf("error %q must not be classified %v", got.Error, test.notClass)
			}
			if test.wantUnderlying != nil && !errors.Is(got.Error, test.wantUnderlying) {
				t.Errorf("error %q lost its underlying sentinel %v", got.Error, test.wantUnderlying)
			}
		})
	}
}
