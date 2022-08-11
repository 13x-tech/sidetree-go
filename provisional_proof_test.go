package sidetree

import (
	"encoding/json"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

func TestNewProvProof(t *testing.T) {
	t.Run("bad data", func(t *testing.T) {
		_, err := NewProvisionalProofFile(&OperationsProcessor{}, []byte("bad data"))
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
	t.Run("empty object", func(t *testing.T) {
		_, err := NewProvisionalProofFile(&OperationsProcessor{}, []byte("{}"))
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}

func TestProvProofProcess(t *testing.T) {
	tests := map[string]struct {
		updateMapping []string
		revealValues  map[string]string
		provUpdates   []Operation
		proofFile     ProvisionalProofFile
		want          error
	}{
		"no error": {
			updateMapping: []string{"id1", "id2"},
			revealValues:  map[string]string{"id1": "value1", "id2": "value2"},
			provUpdates:   []Operation{{RevealValue: "value1"}},
			proofFile: ProvisionalProofFile{
				Operations: ProvProofOperations{
					Update: []SignedUpdateDataOp{{SignedData: "signedData1"}},
				},
			},
			want: nil,
		},
		"update count mismatch between proof and index": {
			updateMapping: []string{"id1", "id2"},
			revealValues:  map[string]string{"id1": "value1", "id2": "value2"},
			provUpdates:   []Operation{{RevealValue: "value1"}, {RevealValue: "value2"}},
			proofFile: ProvisionalProofFile{
				Operations: ProvProofOperations{
					Update: []SignedUpdateDataOp{{SignedData: "signedData1"}},
				},
			},
			want: ErrProofIndexMismatch,
		},
		"update mapping array count mismatch": {
			updateMapping: []string{"id1"},
			revealValues:  map[string]string{"id1": "value1", "id2": "value2"},
			provUpdates:   []Operation{{RevealValue: "value1"}, {RevealValue: "value2"}},
			proofFile: ProvisionalProofFile{
				Operations: ProvProofOperations{
					Update: []SignedUpdateDataOp{{SignedData: "signedData1"}, {SignedData: "signedData2"}},
				},
			},
			want: ErrUpdateMappingMismatch,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := &OperationsProcessor{
				updateOps:          map[string]operations.UpdateInterface{},
				updateMappingArray: test.updateMapping,
				provisionalIndexFile: &ProvisionalIndexFile{
					Operations: ProvOPS{
						Update: test.provUpdates,
					},
					revealValues: test.revealValues,
				},
			}

			ppfJSON, err := json.Marshal(test.proofFile)
			if err != nil {
				t.Errorf("error marshalling proof file: %v", err)
			}

			ppf, err := NewProvisionalProofFile(p, ppfJSON)
			if err != nil {
				t.Errorf("error creating provisional proof file: %v", err)
			}

			if err := ppf.Process(); err != test.want {
				t.Errorf("expected %v, got %v", test.want, err)
			}

		})
	}
}

func TestProvProofSetUpdateOp(t *testing.T) {
	tests := map[string]struct {
		updateMapping []string
		revealValues  map[string]string
		updateOps     map[int]SignedUpdateDataOp
		expected      int
	}{
		"reveal value missing": {
			updateMapping: []string{"id1", "id2"},
			revealValues:  map[string]string{"id1": "value1"},
			updateOps: map[int]SignedUpdateDataOp{
				0: {SignedData: "signedData1"},
				1: {SignedData: "signedData2"},
			},
			expected: 1,
		},
		"nothing missing": {
			updateMapping: []string{"id1", "id2"},
			revealValues:  map[string]string{"id1": "value1", "id2": "value2"},
			updateOps: map[int]SignedUpdateDataOp{
				0: {SignedData: "signedData1"},
				1: {SignedData: "signedData2"},
			},
			expected: 2,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := &OperationsProcessor{
				updateOps:          map[string]operations.UpdateInterface{},
				updateMappingArray: test.updateMapping,
				provisionalIndexFile: &ProvisionalIndexFile{
					revealValues: test.revealValues,
				},
			}
			ppf := ProvisionalProofFile{
				processor: p,
			}

			for id, op := range test.updateOps {
				ppf.setUpdateOp(id, op)
			}

			if len(p.updateOps) != test.expected {
				t.Errorf("expected %d update ops, got %d", test.expected, len(p.updateOps))
			}

		})
	}
}
