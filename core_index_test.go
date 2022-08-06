package sidetree

import (
	"encoding/json"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
)

var testCoreFile = CoreIndexFile{
	Operations: CoreOperations{
		Create: []CreateOperation{},
		Recover: []Operation{{
			DIDSuffix: "abcdefg",
		}},
		Deactivate: []Operation{{
			DIDSuffix: "abcdefg",
		}},
	},
}

func TestDuplicateOperations(t *testing.T) {

	t.Run("test duplicates", func(t *testing.T) {
		t.Run("recover create", func(t *testing.T) {

			var testCoreFile = CoreIndexFile{
				Operations: CoreOperations{
					Create: []CreateOperation{
						{
							SuffixData: did.SuffixData{
								DeltaHash:          "EiC9KeEZci-n4WSQAFHBfXbH-gjGCLRiwOwfn5oDM-ulfg",
								RecoveryCommitment: "EiBZrU3AmONK1yieZzw_BsgGWKoidwEWaBifQXFsY2jLwQ",
							},
						},
					},
					Recover: []Operation{{
						DIDSuffix: "EiDiWRhmzQnRS18dt5Pzqxo2YcxffJR6LiOQp5Fr1K2uCw",
					}},
					Deactivate: []Operation{},
				},
			}

			testFileJSON, err := json.Marshal(testCoreFile)
			if err != nil {
				t.Errorf("Error marshalling test core file: %v", err)
			}

			c, err := NewCoreIndexFile(nil, testFileJSON)
			if err != nil {
				t.Errorf("failed to create core index file: %v", err)
			}

			if err := c.populateCoreOperationArray(); err == nil {
				t.Errorf("expected error when populating multiple core operations for the same id")
			}
		})
		t.Run("recover & deactivate", func(t *testing.T) {

			var testCoreFile = CoreIndexFile{
				Operations: CoreOperations{
					Create: []CreateOperation{},
					Recover: []Operation{{
						DIDSuffix: "abcdefg",
					}},
					Deactivate: []Operation{{
						DIDSuffix: "abcdefg",
					}},
				},
			}

			testFileJSON, err := json.Marshal(testCoreFile)
			if err != nil {
				t.Errorf("Error marshalling test core file: %v", err)
			}

			c, err := NewCoreIndexFile(nil, testFileJSON)
			if err != nil {
				t.Errorf("failed to create core index file: %v", err)
			}

			if err := c.populateCoreOperationArray(); err == nil {
				t.Errorf("expected error when populating multiple core operations for the same id")
			}
		})

		t.Run("recover duplicate", func(t *testing.T) {

			var testCoreFile = CoreIndexFile{
				Operations: CoreOperations{
					Create: []CreateOperation{},
					Recover: []Operation{{
						DIDSuffix: "abcdefg",
					}, {
						DIDSuffix: "abcdefg",
					}},
					Deactivate: []Operation{},
				},
			}

			testFileJSON, err := json.Marshal(testCoreFile)
			if err != nil {
				t.Errorf("Error marshalling test core file: %v", err)
				return
			}

			c, err := NewCoreIndexFile(nil, testFileJSON)
			if err != nil {
				t.Errorf("failed to create core index file: %v", err)
				return
			}

			if err := c.populateCoreOperationArray(); err == nil {
				t.Errorf("expected error when populating multiple core operations for the same id")
				return
			}
		})
	})

	t.Run("no duplicates", func(t *testing.T) {
		var testCoreFile = CoreIndexFile{
			Operations: CoreOperations{
				Create: []CreateOperation{},
				Recover: []Operation{{
					DIDSuffix: "abcdefg",
				}},
				Deactivate: []Operation{{
					DIDSuffix: "hijklmn",
				}},
			},
		}

		testFileJSON, err := json.Marshal(testCoreFile)
		if err != nil {
			t.Errorf("Error marshalling test core file: %v", err)
			return
		}

		c, err := NewCoreIndexFile(nil, testFileJSON)
		if err != nil {
			t.Errorf("failed to create core index file: %v", err)
			return
		}

		if err := c.populateCoreOperationArray(); err != nil {
			t.Errorf("failed to populate core operations array: %v", err)
			return
		}
	})

}

func TestCoreIndexProcess(t *testing.T) {
	t.Run("test no proof", func(t *testing.T) {
		t.Run("with deactivate operation", func(t *testing.T) {
			testCoreIndex := CoreIndexFile{
				Operations: CoreOperations{
					Deactivate: []Operation{{}},
				},
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

			if err := cif.Process(); err == nil || err != ErrNoCoreProof {
				t.Errorf("expected error when processing core index file without proof")
				return
			}
		})
		t.Run("without operations ", func(t *testing.T) {
			testCoreIndex := CoreIndexFile{
				Operations: CoreOperations{},
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
				t.Errorf("unexpected error when processing file without operations or proofs")
				return
			}
		})
	})
	t.Run("test setting processor uris", func(t *testing.T) {
		t.Run("provisional index file", func(t *testing.T) {
			testCoreIndex := CoreIndexFile{
				Operations:          CoreOperations{},
				ProvisionalIndexURI: "provisional-index-uri",
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

			if p.ProvisionalIndexFileURI != "provisional-index-uri" {
				t.Errorf("expected provisional index uri to be %s but got %s", "provisional-index-uri", p.ProvisionalIndexFileURI)
				return
			}

		})
		t.Run("core proof file", func(t *testing.T) {
			testCoreIndex := CoreIndexFile{
				Operations:   CoreOperations{},
				CoreProofURI: "core-proof-uri",
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

			if p.CoreProofFileURI != "core-proof-uri" {
				t.Errorf("expected provisional index uri to be %s but got %s", "core-proof-uri", p.CoreProofFileURI)
				return
			}

		})
	})
}
