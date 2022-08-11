package sidetree

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
)

func TestBadOperationsData(t *testing.T) {
	_, err := NewCoreIndexFile(nil, []byte("bad data"))
	if err == nil {
		t.Errorf("should have failed to create core index file")
	}
}

func TestDuplicateOperations(t *testing.T) {
	tests := map[string]struct {
		create     []CreateOperation
		recover    []Operation
		deactivate []Operation
		want       error
	}{
		"duplicate create": {
			create: []CreateOperation{{
				SuffixData: did.SuffixData{
					DeltaHash:          "abc123",
					RecoveryCommitment: "xyz789",
				},
			}, {
				SuffixData: did.SuffixData{
					DeltaHash:          "abc123",
					RecoveryCommitment: "xyz789",
				},
			}},
			want: ErrDuplicateOperation,
		},
		"duplicate recover": {
			recover: []Operation{{
				DIDSuffix: "abc123",
			}, {
				DIDSuffix: "abc123",
			}},
			want: ErrDuplicateOperation,
		},
		"duplicate deactivate": {
			deactivate: []Operation{{
				DIDSuffix: "abc123",
			}, {
				DIDSuffix: "abc123",
			}},
			want: ErrDuplicateOperation,
		},
		"duplicate mix": {
			create: []CreateOperation{{
				SuffixData: did.SuffixData{
					DeltaHash:          "abc123",
					RecoveryCommitment: "xyz789",
				},
			}, {
				SuffixData: did.SuffixData{
					DeltaHash:          "def456",
					RecoveryCommitment: "uvw456",
				},
			}},
			deactivate: []Operation{{
				DIDSuffix: "abc123",
			}, {
				DIDSuffix: "EiAcia-ZeClGSDCnIi7WRip4sm-jF9QvmsR0QDpPn64Kyw",
			}},
			want: ErrDuplicateOperation,
		},
		"no duplicates": {
			create: []CreateOperation{{
				SuffixData: did.SuffixData{
					DeltaHash:          "abc123",
					RecoveryCommitment: "xyz789",
				},
			}, {
				SuffixData: did.SuffixData{
					DeltaHash:          "def456",
					RecoveryCommitment: "uvw456",
				},
			}},
			deactivate: []Operation{{
				DIDSuffix: "abc123",
			}, {
				DIDSuffix: "def456",
			}},
			recover: []Operation{{
				DIDSuffix: "xyz789",
			}, {
				DIDSuffix: "uvw456",
			}},
			want: nil,
		},
	}

	t.Run("populateCoreOperationArray", func(t *testing.T) {

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				testFile := CoreIndexFile{
					Operations: CoreOperations{
						Create:     test.create,
						Recover:    test.recover,
						Deactivate: test.deactivate,
					},
					CoreProofURI: "core-proof-uri",
				}

				coreJson, err := json.Marshal(testFile)
				if err != nil {
					t.Fatal("failed to marshal core index file: %w", err)
				}
				p := &OperationsProcessor{}
				cif, err := NewCoreIndexFile(p, coreJson)
				if err != nil {
					t.Fatal("failed to create core index file: %w", err)
				}

				if err := cif.Process(); errors.Unwrap(err) != test.want {
					t.Errorf("got %v, want %v", err, test.want)
				}
			})
		}
	})
}

func TestCoreIndexProcess(t *testing.T) {

	t.Run("test without proof uri", func(t *testing.T) {
		tests := map[string]struct {
			create     []CreateOperation
			recover    []Operation
			deactivate []Operation
			coreProof  string
			want       error
		}{
			"deactivate": {
				deactivate: []Operation{{
					DIDSuffix: "abc123",
				}},
				want: ErrNoCoreProof,
			},
			"recover": {
				recover: []Operation{{
					DIDSuffix: "abc123",
				}},
				want: ErrNoCoreProof,
			},
			"create": {
				create: []CreateOperation{{
					SuffixData: did.SuffixData{
						DeltaHash:          "abc123",
						RecoveryCommitment: "xyz789",
					},
				}},
				want: nil,
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				testFile := CoreIndexFile{
					Operations: CoreOperations{
						Create:     test.create,
						Recover:    test.recover,
						Deactivate: test.deactivate,
					},
					CoreProofURI: test.coreProof,
				}

				coreJson, err := json.Marshal(testFile)
				if err != nil {
					t.Fatal("failed to marshal core index file: %w", err)
				}
				p := &OperationsProcessor{}
				cif, err := NewCoreIndexFile(p, coreJson)
				if err != nil {
					t.Fatal("failed to create core index file: %w", err)
				}
				if err := cif.Process(); err != test.want {
					t.Errorf("got %v, want %v", err, test.want)
				}
			})
		}
	})

	t.Run("test setting processor uris", func(t *testing.T) {

		t.Run("provisional index file", func(t *testing.T) {

			tests := map[string]struct {
				IndexURI string
			}{
				"with uri": {
					IndexURI: "provisional-index-uri",
				},
				"without uri": {
					IndexURI: "",
				},
			}

			for name, test := range tests {

				t.Run(name, func(t *testing.T) {
					testCoreIndex := CoreIndexFile{
						Operations:          CoreOperations{},
						ProvisionalIndexURI: test.IndexURI,
					}

					coreIndexJSON, err := json.Marshal(testCoreIndex)
					if err != nil {
						t.Fatalf("Error marshalling test core index file: %v", err)
					}
					p := &OperationsProcessor{}

					cif, err := NewCoreIndexFile(p, coreIndexJSON)
					if err != nil {
						t.Fatalf("failed to create core index file: %v", err)
					}
					if err := cif.Process(); err != nil {
						t.Fatalf("unexpected error when processing file without operations or proofs: %v", err)
					}

					if p.provisionalIndexFileURI != test.IndexURI {
						t.Fatalf("expected provisional index uri to be %s but got %s", test.IndexURI, p.provisionalIndexFileURI)
					}
				})
			}
		})

		t.Run("core proof file", func(t *testing.T) {
			tests := map[string]struct {
				CoreProofURI string
			}{
				"with uri": {
					CoreProofURI: "core-proof-uri",
				},
				"without uri": {
					CoreProofURI: "",
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {

					testCoreIndex := CoreIndexFile{
						Operations:   CoreOperations{},
						CoreProofURI: test.CoreProofURI,
					}
					coreIndexJSON, err := json.Marshal(testCoreIndex)
					if err != nil {
						t.Errorf("Error marshalling test core index file: %v", err)
						return
					}
					p := &OperationsProcessor{}

					cif, err := NewCoreIndexFile(p, coreIndexJSON)
					if err != nil {
						t.Errorf("failed to create core index file: %v", err)
						return
					}
					if err := cif.Process(); err != nil {
						t.Errorf("unexpected error when processing file without operations or proofs: %v", err)
						return
					}
					if p.coreProofFileURI != test.CoreProofURI {
						t.Errorf("expected provisional index uri to be %s but got %s", test.CoreProofURI, p.coreProofFileURI)
						return
					}
				})
			}
		})
	})
}
