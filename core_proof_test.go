package sidetree

import (
	"encoding/json"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

func TestNewCoreProofFile(t *testing.T) {
	t.Run("invalid core proof", func(t *testing.T) {
		_, err := NewCoreProofFile(nil, []byte("invalid"))
		if err == nil {
			t.Errorf("should have failed to create core proof file")
		}
	})

	t.Run("valid core proof", func(t *testing.T) {
		cpf := CoreProofFile{
			Operations: CoreProofOperations{
				Recover: []SignedRecoverDataOp{
					{SignedData: "some-data"},
				},
				Deactivate: []SignedDeactivateDataOp{
					{SignedData: "some-data"},
				},
			},
		}

		cpfJson, err := json.Marshal(cpf)
		if err != nil {
			t.Errorf("Error marshalling core proof file: %v", err)
			return
		}

		_, err = NewCoreProofFile(nil, cpfJson)
		if err != nil {
			t.Errorf("should have succeeded to create core proof file, got error: %v", err)
		}
	})
}

func TestCountOperations(t *testing.T) {
	t.Run("count operations", func(t *testing.T) {
		tests := map[string]struct {
			ProofFile CoreProofFile
			IndexFile CoreIndexFile
			want      error
		}{
			"invalid recover count": {
				want: ErrCoreProofCount,
				ProofFile: CoreProofFile{
					Operations: CoreProofOperations{
						Recover: []SignedRecoverDataOp{{
							SignedData: "some-data",
						}, {
							SignedData: "some-data-2",
						}},
						Deactivate: []SignedDeactivateDataOp{},
					},
				},
				IndexFile: CoreIndexFile{
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix:   "abc123",
							RevealValue: "some-data",
						}},
						Deactivate: []Operation{{
							DIDSuffix: "abc123",
						}},
					},
				},
			},
			"invalid deactivate count": {
				want: ErrCoreProofCount,
				ProofFile: CoreProofFile{
					Operations: CoreProofOperations{
						Recover: []SignedRecoverDataOp{
							{SignedData: "some-data"},
						},
						Deactivate: []SignedDeactivateDataOp{
							{SignedData: "some-data-2"},
						},
					},
				},
				IndexFile: CoreIndexFile{
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix:   "abc123",
							RevealValue: "some-data",
						}},
						Deactivate: []Operation{},
					},
				},
			},
			"valid": {
				want: nil,
				ProofFile: CoreProofFile{
					Operations: CoreProofOperations{
						Recover:    []SignedRecoverDataOp{{SignedData: "some-data"}},
						Deactivate: []SignedDeactivateDataOp{{SignedData: "some-data-2"}},
					},
				},
				IndexFile: CoreIndexFile{
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix:   "abc123",
							RevealValue: "some-data",
						}},
						Deactivate: []Operation{{DIDSuffix: "abc123"}},
					},
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				fileJSON, err := json.Marshal(test.ProofFile)
				if err != nil {
					t.Fatalf("failed to marshal core proof file: %v", err)
				}
				p := &OperationsProcessor{
					coreIndexFile: &test.IndexFile,
					deactivateOps: map[string]operations.DeactivateInterface{},
					recoverOps:    map[string]operations.RecoverInterface{},
				}
				c, err := NewCoreProofFile(p, fileJSON)
				if err != nil {
					t.Fatalf("failed to create core proof file: %v", err)
				}
				if err := c.Process(); err != test.want {
					t.Errorf("expected error %v, got %v", test.want, err)
				}
			})
		}

	})
}
