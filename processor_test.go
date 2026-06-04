package sidetree

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
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
			if !checkError(err, test.want) {
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
		"with fee functions": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			feeFunctions: []interface{}{
				BaseFeeAlgorithm(func(opCount int, anchorPoint string) int { return 0 }),
				PerOperationFee(func(baseFee int, opCount int, anchorPoint string) bool { return true }),
				ValueLocking(func(writerLockId string, baseFee int, opCount int, anchorPoint string) bool { return true }),
			},
			want: ProcessedOperations{
				Error: nil,
			},
			cas: func() CAS {
				cas := NewTestCAS()
				cas.insertObject("abc", []byte("{}"))
				return cas
			}(),
		},
		"per op fee returns false": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			feeFunctions: []interface{}{
				BaseFeeAlgorithm(func(opCount int, anchorPoint string) int { return 0 }),
				PerOperationFee(func(baseFee int, opCount int, anchorPoint string) bool { return false }),
				ValueLocking(func(writerLockId string, baseFee int, opCount int, anchorPoint string) bool { return true }),
			},
			want: ProcessedOperations{
				Error: fmt.Errorf("per op fee is not valid"),
			},
			cas: func() CAS {
				cas := NewTestCAS()
				cas.insertObject("abc", []byte("{}"))
				return cas
			}(),
		},
		"value locking returns false": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			feeFunctions: []interface{}{
				BaseFeeAlgorithm(func(opCount int, anchorPoint string) int { return 0 }),
				PerOperationFee(func(baseFee int, opCount int, anchorPoint string) bool { return true }),
				ValueLocking(func(writerLockId string, baseFee int, opCount int, anchorPoint string) bool { return false }),
			},
			want: ProcessedOperations{
				Error: fmt.Errorf("value lock is not valid"),
			},
			cas: func() CAS {
				cas := NewTestCAS()
				cas.insertObject("abc", []byte("{}"))
				return cas
			}(),
		},
		"index file found": {
			anchor: operations.Anchor{Anchor: "1.cas"},
			want: ProcessedOperations{
				Error: nil,
			},
			cas: func() CAS {
				cas := NewTestCAS()
				cas.insertObject("cas", []byte("{}"))
				return cas
			}(),
		},
		"index file not found": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to get core index file"),
			},
			cas: NewTestCAS(),
		},
		"bad core index data": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to create core index file"),
			},
			cas: func() CAS {
				cas := NewTestCAS()
				cas.insertObject("abc", []byte("bad data"))
				return cas
			}(),
		},
		"invalid core index": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: ErrNoCoreProof,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				coreIndex := CoreIndexFile{
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
				}
				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("abc", ciJSON)
				return cas
			}(),
		},
		"with core proof not found": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to get core proof file"),
			},
			cas: func() CAS {
				cas := NewTestCAS()

				coreIndex := CoreIndexFile{
					CoreProofURI: "core-proof-uri",
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
				}
				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("abc", ciJSON)
				return cas
			}(),
		},
		"bad core proof data": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to create core proof file"),
			},
			cas: func() CAS {
				cas := NewTestCAS()
				cas.insertObject("core-proof-uri", []byte("bad data"))

				coreIndex := CoreIndexFile{
					CoreProofURI: "core-proof-uri",
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
				}
				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("abc", ciJSON)
				return cas
			}(),
		},
		"with core proof process error": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: fmt.Errorf("core proof count mismatch"),
			},
			cas: func() CAS {
				cas := NewTestCAS()

				coreIndex := CoreIndexFile{
					CoreProofURI: "xyz",
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
				}
				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("abc", ciJSON)
				cas.insertObject("xyz", []byte("{}"))
				return cas
			}(),
		},
		"cannot fetch provisional index": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to get provisional index file"),
			},
			cas: func() CAS {
				cas := NewTestCAS()

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "xyz",
					Operations:          CoreOperations{},
				}
				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("abc", ciJSON)
				return cas
			}(),
		},
		"bad provisional index data": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to create provisional index file"),
			},
			cas: func() CAS {
				cas := NewTestCAS()
				cas.insertObject("xyz", []byte("bad data"))
				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "xyz",
					Operations:          CoreOperations{},
				}
				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("abc", ciJSON)
				return cas
			}(),
		},
		"empty provisional index": {
			anchor: operations.Anchor{Anchor: "1.abc"},
			want: ProcessedOperations{
				Error: ErrMultipleChunks,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "xyz",
					Operations:          CoreOperations{},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("abc", ciJSON)
				cas.insertObject("xyz", []byte("{}"))

				return cas
			}(),
		},
		"fetch provisional proof error": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to get provisional proof file"),
			},
			cas: func() CAS {
				cas := NewTestCAS()

				chunk := ChunkFile{
					Deltas: []did.Delta{{}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					Operations:          CoreOperations{},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"provisional proof bad data": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to create provisional proof file"),
			},
			cas: func() CAS {
				cas := NewTestCAS()

				cas.insertObject("prov-proof-uri", []byte("bad data"))

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					Operations:          CoreOperations{},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"provisional proof process error": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrProofIndexMismatch,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{
						Update: []SignedUpdateDataOp{},
					},
				}
				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				chunk := ChunkFile{
					Deltas: []did.Delta{{}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					Operations:          CoreOperations{},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"cannot fetch chunk": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to get chunk file"),
			},
			cas: func() CAS {
				cas := NewTestCAS()

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{
						Update: []SignedUpdateDataOp{{
							SignedData: "signed-data",
						}},
					},
				}
				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					Operations:          CoreOperations{},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"bad chunk data": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: fmt.Errorf("failed to create chunk file"),
			},
			cas: func() CAS {
				cas := NewTestCAS()

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{
						Update: []SignedUpdateDataOp{{
							SignedData: "signed-data",
						}},
					},
				}
				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)
				cas.insertObject("chunk-uri", []byte("bad chunk data"))

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					Operations:          CoreOperations{},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"cannot process chunk": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrInvalidDeltaCount,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{
						Update: []SignedUpdateDataOp{{
							SignedData: "signed-data",
						}},
					},
				}

				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				chunk := ChunkFile{
					Deltas: []did.Delta{},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					Operations:          CoreOperations{},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate create": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{},
				}

				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				chunk := ChunkFile{
					Deltas: []did.Delta{{}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				coreProof := CoreProofFile{
					Operations: CoreProofOperations{},
				}
				cpJSON, err := json.Marshal(coreProof)
				if err != nil {
					t.Fatal("failed to marshal core proof")
				}
				cas.insertObject("core-proof-uri", cpJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations:          ProvOPS{},
					Chunks:              []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					CoreProofURI:        "core-proof-uri",
					Operations: CoreOperations{
						Create: []CreateOperation{{
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
					},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate recover": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{},
				}

				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				chunk := ChunkFile{
					Deltas: []did.Delta{{}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				coreProof := CoreProofFile{
					Operations: CoreProofOperations{
						Recover: []SignedRecoverDataOp{{}, {}},
					},
				}
				cpJSON, err := json.Marshal(coreProof)
				if err != nil {
					t.Fatal("failed to marshal core proof")
				}
				cas.insertObject("core-proof-uri", cpJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations:          ProvOPS{},
					Chunks:              []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					CoreProofURI:        "core-proof-uri",
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix:   "abc123",
							RevealValue: "reveal-value",
						}, {
							DIDSuffix:   "abc123",
							RevealValue: "reveal-value",
						}},
					},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate update": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{
						Update: []SignedUpdateDataOp{{}, {}},
					},
				}

				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				chunk := ChunkFile{
					Deltas: []did.Delta{{}, {}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix: "abc123",
						}, {
							DIDSuffix: "abc123",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					Operations:          CoreOperations{},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate deactivate": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				coreProof := CoreProofFile{
					Operations: CoreProofOperations{
						Deactivate: []SignedDeactivateDataOp{{}, {}},
					},
				}
				cpJSON, err := json.Marshal(coreProof)
				if err != nil {
					t.Fatal("failed to marshal core proof")
				}
				cas.insertObject("core-proof-uri", cpJSON)

				coreIndex := CoreIndexFile{
					CoreProofURI: "core-proof-uri",
					Operations: CoreOperations{
						Deactivate: []Operation{{
							DIDSuffix:   "abc123",
							RevealValue: "reveal-value",
						}, {
							DIDSuffix:   "abc123",
							RevealValue: "reveal-value",
						}},
					},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate create + update": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{
						Update: []SignedUpdateDataOp{{
							SignedData: "signed-data",
						}},
					},
				}

				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				chunk := ChunkFile{
					Deltas: []did.Delta{{}, {}, {}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "EiAcia-ZeClGSDCnIi7WRip4sm-jF9QvmsR0QDpPn64Kyw",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					Operations: CoreOperations{
						Create: []CreateOperation{{
							SuffixData: did.SuffixData{
								DeltaHash:          "def123",
								RecoveryCommitment: "xyz789",
							},
						}, {
							SuffixData: did.SuffixData{
								DeltaHash:          "abc123",
								RecoveryCommitment: "xyz789",
							},
						}},
					},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate create + deactivate": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				chunk := ChunkFile{
					Deltas: []did.Delta{{}, {}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provIndex := ProvisionalIndexFile{
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreProof := CoreProofFile{
					Operations: CoreProofOperations{
						Deactivate: []SignedDeactivateDataOp{{}},
					},
				}

				cpJSON, err := json.Marshal(coreProof)
				if err != nil {
					t.Fatal("failed to marshal core proof")
				}
				cas.insertObject("core-proof-uri", cpJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					CoreProofURI:        "core-proof-uri",
					Operations: CoreOperations{
						Create: []CreateOperation{{
							SuffixData: did.SuffixData{
								DeltaHash:          "def123",
								RecoveryCommitment: "xyz789",
							},
						}, {
							SuffixData: did.SuffixData{
								DeltaHash:          "abc123",
								RecoveryCommitment: "xyz789",
							},
						}},
						Deactivate: []Operation{{
							DIDSuffix: "EiAcia-ZeClGSDCnIi7WRip4sm-jF9QvmsR0QDpPn64Kyw",
						}},
					},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate create + recover": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				chunk := ChunkFile{
					Deltas: []did.Delta{{}, {}, {}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provIndex := ProvisionalIndexFile{
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreProof := CoreProofFile{
					Operations: CoreProofOperations{
						Recover: []SignedRecoverDataOp{{}},
					},
				}

				cpJSON, err := json.Marshal(coreProof)
				if err != nil {
					t.Fatal("failed to marshal core proof")
				}
				cas.insertObject("core-proof-uri", cpJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					CoreProofURI:        "core-proof-uri",
					Operations: CoreOperations{
						Create: []CreateOperation{{
							SuffixData: did.SuffixData{
								DeltaHash:          "def123",
								RecoveryCommitment: "xyz789",
							},
						}, {
							SuffixData: did.SuffixData{
								DeltaHash:          "abc123",
								RecoveryCommitment: "xyz789",
							},
						}},
						Recover: []Operation{{
							DIDSuffix: "EiAcia-ZeClGSDCnIi7WRip4sm-jF9QvmsR0QDpPn64Kyw",
						}},
					},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate update + recover": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				chunk := ChunkFile{
					Deltas: []did.Delta{{}, {}, {}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{
						Update: []SignedUpdateDataOp{{}, {}},
					},
				}

				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "xyz",
							RevealValue: "reveal-value",
						}, {
							DIDSuffix:   "abc",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreProof := CoreProofFile{
					Operations: CoreProofOperations{
						Recover: []SignedRecoverDataOp{{}},
					},
				}

				cpJSON, err := json.Marshal(coreProof)
				if err != nil {
					t.Fatal("failed to marshal core proof")
				}
				cas.insertObject("core-proof-uri", cpJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					CoreProofURI:        "core-proof-uri",
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix: "abc",
						}},
					},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate update + deactivate": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				chunk := ChunkFile{
					Deltas: []did.Delta{{}, {}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{
						Update: []SignedUpdateDataOp{{}, {}},
					},
				}

				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "xyz",
							RevealValue: "reveal-value",
						}, {
							DIDSuffix:   "abc",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreProof := CoreProofFile{
					Operations: CoreProofOperations{
						Deactivate: []SignedDeactivateDataOp{{}},
					},
				}

				cpJSON, err := json.Marshal(coreProof)
				if err != nil {
					t.Fatal("failed to marshal core proof")
				}
				cas.insertObject("core-proof-uri", cpJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					CoreProofURI:        "core-proof-uri",
					Operations: CoreOperations{
						Deactivate: []Operation{{
							DIDSuffix: "abc",
						}},
					},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"duplicate recover + deactivate": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: ErrDuplicateOperation,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				chunk := ChunkFile{
					Deltas: []did.Delta{{}, {}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provIndex := ProvisionalIndexFile{
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreProof := CoreProofFile{
					Operations: CoreProofOperations{
						Recover:    []SignedRecoverDataOp{{}, {}},
						Deactivate: []SignedDeactivateDataOp{{}},
					},
				}

				cpJSON, err := json.Marshal(coreProof)
				if err != nil {
					t.Fatal("failed to marshal core proof")
				}
				cas.insertObject("core-proof-uri", cpJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					CoreProofURI:        "core-proof-uri",
					Operations: CoreOperations{
						Recover: []Operation{{
							DIDSuffix: "xyz",
						}, {
							DIDSuffix: "abc",
						}},
						Deactivate: []Operation{{
							DIDSuffix: "abc",
						}},
					},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
		},
		"valid provisional index": {
			anchor: operations.Anchor{Anchor: "1.core-index-uri"},
			want: ProcessedOperations{
				Error: nil,
			},
			cas: func() CAS {
				cas := NewTestCAS()

				provProof := ProvisionalProofFile{
					Operations: ProvProofOperations{
						Update: []SignedUpdateDataOp{{
							SignedData: "signed-data",
						}},
					},
				}

				ppJSON, err := json.Marshal(provProof)
				if err != nil {
					t.Fatal("failed to marshal provisional proof")
				}
				cas.insertObject("prov-proof-uri", ppJSON)

				chunk := ChunkFile{
					Deltas: []did.Delta{{}},
				}

				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					t.Fatal("failed to marshal chunk")
				}
				cas.insertObject("chunk-uri", chunkJSON)

				provIndex := ProvisionalIndexFile{
					ProvisionalProofURI: "prov-proof-uri",
					Operations: ProvOPS{
						Update: []Operation{{
							DIDSuffix:   "did:abc:123",
							RevealValue: "reveal-value",
						}},
					},
					Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
				}

				piJSON, err := json.Marshal(provIndex)
				if err != nil {
					t.Fatal("failed to marshal provisional index")
				}
				cas.insertObject("prov-index-uri", piJSON)

				coreIndex := CoreIndexFile{
					ProvisionalIndexURI: "prov-index-uri",
					Operations:          CoreOperations{},
				}

				ciJSON, err := json.Marshal(coreIndex)
				if err != nil {
					t.Fatal("failed to marshal core index")
				}
				cas.insertObject("core-index-uri", ciJSON)

				return cas
			}(),
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
			if !checkError(got.Error, test.want.Error) {
				t.Errorf("expected error %v, got %v", test.want.Error, got.Error)
			}

		})
	}
}

func TestProcessFilterDIDs(t *testing.T) {
	testCas := func() CAS {
		cas := NewTestCAS()

		provProof := ProvisionalProofFile{
			Operations: ProvProofOperations{
				Update: []SignedUpdateDataOp{{
					SignedData: "signed-data",
				}},
			},
		}

		ppJSON, err := json.Marshal(provProof)
		if err != nil {
			t.Fatal("failed to marshal provisional proof")
		}
		cas.insertObject("prov-proof-uri", ppJSON)

		chunk := ChunkFile{
			Deltas: []did.Delta{{}, {}, {}},
		}

		chunkJSON, err := json.Marshal(chunk)
		if err != nil {
			t.Fatal("failed to marshal chunk")
		}
		cas.insertObject("chunk-uri", chunkJSON)

		provIndex := ProvisionalIndexFile{
			ProvisionalProofURI: "prov-proof-uri",
			Operations: ProvOPS{
				Update: []Operation{{
					DIDSuffix:   "update-did",
					RevealValue: "reveal-value",
				}},
			},
			Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}},
		}

		piJSON, err := json.Marshal(provIndex)
		if err != nil {
			t.Fatal("failed to marshal provisional index")
		}
		cas.insertObject("prov-index-uri", piJSON)

		coreProof := CoreProofFile{
			Operations: CoreProofOperations{
				Recover: []SignedRecoverDataOp{{
					SignedData: "signed-data",
				}},
				Deactivate: []SignedDeactivateDataOp{{
					SignedData: "signed-data",
				}},
			},
		}

		cpJSON, err := json.Marshal(coreProof)
		if err != nil {
			t.Fatal("failed to marshal core proof")
		}
		cas.insertObject("core-proof-uri", cpJSON)

		coreIndex := CoreIndexFile{
			ProvisionalIndexURI: "prov-index-uri",
			CoreProofURI:        "core-proof-uri",
			Operations: CoreOperations{
				Recover: []Operation{{
					DIDSuffix: "recover-did",
				}},
				Deactivate: []Operation{{
					DIDSuffix: "deactivate-did",
				}},
				Create: []CreateOperation{{
					SuffixData: did.SuffixData{
						DeltaHash:          "abc123",
						RecoveryCommitment: "xyz789",
					},
				}},
			},
		}

		ciJSON, err := json.Marshal(coreIndex)
		if err != nil {
			t.Fatal("failed to marshal core index")
		}
		cas.insertObject("core-index-uri", ciJSON)

		return cas
	}()

	tests := map[string]struct {
		cas    CAS
		filter []string
		want   int
	}{
		"no filter": {
			want: 4,
			cas:  testCas,
		},
		"update filter": {
			filter: []string{"update-did"},
			want:   1,
			cas:    testCas,
		},
		"recover filter": {
			filter: []string{"recover-did"},
			want:   1,
			cas:    testCas,
		},
		"deactivate filter": {
			filter: []string{"deactivate-did"},
			want:   1,
			cas:    testCas,
		},
		"create filter": {
			filter: []string{"EiAcia-ZeClGSDCnIi7WRip4sm-jF9QvmsR0QDpPn64Kyw"},
			want:   1,
			cas:    testCas,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p, err := Processor(
				// Declares 4 operations: 1 create + 1 recover + 1 deactivate
				// (core) + 1 update (provisional), matching the fixture so the
				// anchored-count cross-check passes.
				operations.Anchor{Anchor: "4.core-index-uri"},
				WithCAS(test.cas),
				WithPrefix("test"),
				WithDIDs(test.filter),
			)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			got := p.Process()
			if got.Error != nil {
				t.Errorf("expected no error, got %v", got.Error)
			}
			total := len(got.CreateOps) +
				len(got.UpdateOps) +
				len(got.DeactivateOps) +
				len(got.RecoverOps)
			if total != test.want {
				t.Fatalf("expected %d operations, got %d", test.want, total)
			}

		})
	}
}

// TestProcessorOperationLimit verifies the unconditional per-anchor
// operation-count enforcement (the P0 consensus rule, #28): a spec-compliant
// ION node rejects the entire batch when the anchor-string operation count
// violates the value-time-lock limits. The rejection must be permanent
// (ErrMalformed), so the observer skips the batch rather than retrying it.
func TestProcessorOperationLimit(t *testing.T) {
	casWith := func(t *testing.T, cid string, ci CoreIndexFile) CAS {
		t.Helper()
		cas := NewTestCAS()
		b, err := json.Marshal(ci)
		if err != nil {
			t.Fatalf("failed to marshal core index: %v", err)
		}
		cas.insertObject(cid, b)
		return cas
	}

	accept := ValueLocking(func(string, int, int, string) bool { return true })
	reject := ValueLocking(func(string, int, int, string) bool { return false })

	tests := map[string]struct {
		anchor    operations.AnchorString
		coreIndex CoreIndexFile
		valueLock ValueLocking
		wantErr   error
	}{
		"zero declared count rejects": {
			anchor:    "0.cid",
			coreIndex: CoreIndexFile{},
			wantErr:   ErrInvalidOperationCount,
		},
		"non-numeric declared count rejects": {
			anchor:    "abc.cid",
			coreIndex: CoreIndexFile{},
			wantErr:   ErrInvalidOperationCount,
		},
		"negative declared count rejects": {
			anchor:    "-5.cid",
			coreIndex: CoreIndexFile{},
			wantErr:   ErrInvalidOperationCount,
		},
		"at no-lock limit accepts": {
			anchor:    "100.cid",
			coreIndex: CoreIndexFile{},
			wantErr:   nil,
		},
		"over no-lock limit without a lock rejects": {
			anchor:    "101.cid",
			coreIndex: CoreIndexFile{},
			wantErr:   ErrOperationLimitExceeded,
		},
		"over batch max without a lock rejects": {
			anchor:    "10001.cid",
			coreIndex: CoreIndexFile{},
			wantErr:   ErrTooManyOperations,
		},
		"over batch max rejects even with a verified lock": {
			anchor:    "10001.cid",
			coreIndex: CoreIndexFile{WriterLockId: "lock-123"},
			valueLock: accept,
			wantErr:   ErrTooManyOperations,
		},
		"over quota with a lock but no verifier rejects": {
			anchor:    "200.cid",
			coreIndex: CoreIndexFile{WriterLockId: "lock-123"},
			wantErr:   ErrUnverifiableValueLock,
		},
		"over quota with a verifier that accepts is allowed": {
			anchor:    "200.cid",
			coreIndex: CoreIndexFile{WriterLockId: "lock-123"},
			valueLock: accept,
			wantErr:   nil,
		},
		"over quota with a verifier that rejects is rejected": {
			anchor:    "200.cid",
			coreIndex: CoreIndexFile{WriterLockId: "lock-123"},
			valueLock: reject,
			wantErr:   fmt.Errorf("value lock is not valid"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cas := casWith(t, test.anchor.CID(), test.coreIndex)
			opts := []SideTreeOption{WithCAS(cas), WithPrefix("test")}
			if test.valueLock != nil {
				opts = append(opts, WithFeeFunctions(test.valueLock))
			}

			p, err := Processor(operations.Anchor{Anchor: test.anchor}, opts...)
			if err != nil {
				t.Fatalf("expected no error creating processor, got %v", err)
			}

			got := p.Process()
			if !checkError(got.Error, test.wantErr) {
				t.Errorf("expected error %v, got %v", test.wantErr, got.Error)
			}
			// Every rejection must be classified permanent (ErrMalformed).
			if test.wantErr != nil && !errors.Is(got.Error, ErrMalformed) {
				t.Errorf("expected rejection to be ErrMalformed (permanent), got %v", got.Error)
			}
		})
	}
}

// TestProcessorAnchoredCountExceedsDeclared verifies the anti-bypass
// cross-check: a writer that understates the anchor-string operation count to
// pass the limit gate while packing more operations into the anchored files is
// still rejected once every file is parsed.
func TestProcessorAnchoredCountExceedsDeclared(t *testing.T) {
	cas := NewTestCAS()

	ci := CoreIndexFile{}
	for i := 0; i <= MaxNumberOfOperationsForNoValueTimeLock; i++ { // 101 distinct creates
		ci.Operations.Create = append(ci.Operations.Create, CreateOperation{
			SuffixData: did.SuffixData{
				DeltaHash:          fmt.Sprintf("delta-%d", i),
				RecoveryCommitment: "recovery-commitment",
			},
		})
	}
	b, err := json.Marshal(ci)
	if err != nil {
		t.Fatalf("failed to marshal core index: %v", err)
	}
	cas.insertObject("cid", b)

	// Anchor string declares only 1 operation but the file packs 101.
	p, err := Processor(operations.Anchor{Anchor: "1.cid"}, WithCAS(cas), WithPrefix("test"))
	if err != nil {
		t.Fatalf("expected no error creating processor, got %v", err)
	}

	got := p.Process()
	if !checkError(got.Error, ErrOperationCountMismatch) {
		t.Errorf("expected %v, got %v", ErrOperationCountMismatch, got.Error)
	}
	if !errors.Is(got.Error, ErrMalformed) {
		t.Errorf("expected rejection to be ErrMalformed (permanent), got %v", got.Error)
	}
}

// TestProcessorRejectsOversizedFile verifies the defensive size guard (#31): if
// a CAS returns a (decompressed) file larger than the protocol per-file cap ×
// MaxMemoryDecompressionFactor, the batch is permanently rejected (ErrMalformed)
// rather than parsed. Uses the core index file, whose cap is the smallest.
func TestProcessorRejectsOversizedFile(t *testing.T) {
	cas := NewTestCAS()
	oversized := make([]byte, MaxCoreIndexFileSizeInBytes*MaxMemoryDecompressionFactor+1)
	cas.insertObject("cid", oversized)

	p, err := Processor(operations.Anchor{Anchor: "1.cid"}, WithCAS(cas), WithPrefix("test"))
	if err != nil {
		t.Fatalf("expected no error creating processor, got %v", err)
	}

	got := p.Process()
	if !checkError(got.Error, ErrFileTooLarge) {
		t.Errorf("expected %v, got %v", ErrFileTooLarge, got.Error)
	}
	if !errors.Is(got.Error, ErrMalformed) {
		t.Errorf("expected rejection to be ErrMalformed (permanent), got %v", got.Error)
	}
}

// TestFetchPassesPerFileSizeCaps verifies that each fetch path passes the
// correct protocol per-file cap to CAS.Get (#31), exercising all five Sidetree
// files in one batch. The TestCAS records the maxSizeInBytes it was called with
// per URI.
func TestFetchPassesPerFileSizeCaps(t *testing.T) {
	cas := NewTestCAS()

	insert := func(uri string, v interface{}) {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("failed to marshal %s: %v", uri, err)
		}
		cas.insertObject(uri, b)
	}

	// create + recover + update => 3 chunk deltas; deactivate has no delta.
	insert("chunk-uri", ChunkFile{Deltas: []did.Delta{{}, {}, {}}})
	insert("prov-proof-uri", ProvisionalProofFile{
		Operations: ProvProofOperations{Update: []SignedUpdateDataOp{{SignedData: "signed-data"}}},
	})
	insert("prov-index-uri", ProvisionalIndexFile{
		ProvisionalProofURI: "prov-proof-uri",
		Operations:          ProvOPS{Update: []Operation{{DIDSuffix: "update-did", RevealValue: "reveal-value"}}},
		Chunks:              []ProvChunk{{ChunkFileURI: "chunk-uri"}},
	})
	insert("core-proof-uri", CoreProofFile{
		Operations: CoreProofOperations{
			Recover:    []SignedRecoverDataOp{{SignedData: "signed-data"}},
			Deactivate: []SignedDeactivateDataOp{{SignedData: "signed-data"}},
		},
	})
	insert("core-index-uri", CoreIndexFile{
		ProvisionalIndexURI: "prov-index-uri",
		CoreProofURI:        "core-proof-uri",
		Operations: CoreOperations{
			Recover:    []Operation{{DIDSuffix: "recover-did"}},
			Deactivate: []Operation{{DIDSuffix: "deactivate-did"}},
			Create:     []CreateOperation{{SuffixData: did.SuffixData{DeltaHash: "abc123", RecoveryCommitment: "xyz789"}}},
		},
	})

	// 4 operations (create + recover + deactivate + update).
	p, err := Processor(operations.Anchor{Anchor: "4.core-index-uri"}, WithCAS(cas), WithPrefix("test"))
	if err != nil {
		t.Fatalf("expected no error creating processor, got %v", err)
	}
	if got := p.Process(); got.Error != nil {
		t.Fatalf("expected the batch to process cleanly, got %v", got.Error)
	}

	want := map[string]int{
		"core-index-uri": MaxCoreIndexFileSizeInBytes,
		"core-proof-uri": MaxProofFileSizeInBytes,
		"prov-index-uri": MaxProvisionalIndexFileSizeInBytes,
		"prov-proof-uri": MaxProofFileSizeInBytes,
		"chunk-uri":      MaxChunkFileSizeInBytes,
	}
	for uri, cap := range want {
		got, ok := cas.maxSizes[uri]
		if !ok {
			t.Errorf("%s was never fetched", uri)
			continue
		}
		if got != cap {
			t.Errorf("%s fetched with cap %d, want %d", uri, got, cap)
		}
	}
}

// TestProcessorPerFieldCaps verifies the per-field structural caps enforced when
// parsing the core index file (#32): writerLockId and the embedded CAS URIs.
// Each violation is a permanent rejection (ErrMalformed); a value exactly at the
// cap is accepted.
func TestProcessorPerFieldCaps(t *testing.T) {
	tooLongURI := strings.Repeat("u", MaxCASURILength+1)         // 101
	tooLongLock := strings.Repeat("l", MaxWriterLockIDInBytes+1) // 201
	atCapLock := strings.Repeat("l", MaxWriterLockIDInBytes)     // 200 (must pass)

	tests := map[string]struct {
		anchor    operations.AnchorString
		coreIndex CoreIndexFile
		wantErr   error // nil means the batch must process cleanly
	}{
		"writer lock id too long": {
			anchor:    "1.cid",
			coreIndex: CoreIndexFile{WriterLockId: tooLongLock},
			wantErr:   ErrWriterLockIDTooLong,
		},
		"writer lock id at cap accepts": {
			anchor:    "1.cid",
			coreIndex: CoreIndexFile{WriterLockId: atCapLock},
			wantErr:   nil,
		},
		"core proof uri too long": {
			anchor:    "1.cid",
			coreIndex: CoreIndexFile{CoreProofURI: tooLongURI},
			wantErr:   ErrCASURITooLong,
		},
		"provisional index uri too long": {
			anchor:    "1.cid",
			coreIndex: CoreIndexFile{ProvisionalIndexURI: tooLongURI},
			wantErr:   ErrCASURITooLong,
		},
		"all fields within caps accepts": {
			anchor:    "1.cid",
			coreIndex: CoreIndexFile{WriterLockId: "lock"},
			wantErr:   nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cas := NewTestCAS()
			b, err := json.Marshal(test.coreIndex)
			if err != nil {
				t.Fatalf("failed to marshal core index: %v", err)
			}
			cas.insertObject(test.anchor.CID(), b)

			p, err := Processor(operations.Anchor{Anchor: test.anchor}, WithCAS(cas), WithPrefix("test"))
			if err != nil {
				t.Fatalf("expected no error creating processor, got %v", err)
			}

			got := p.Process()
			if !checkError(got.Error, test.wantErr) {
				t.Errorf("expected %v, got %v", test.wantErr, got.Error)
			}
			if test.wantErr != nil && !errors.Is(got.Error, ErrMalformed) {
				t.Errorf("expected rejection to be ErrMalformed (permanent), got %v", got.Error)
			}
		})
	}
}

// TestProcessorAcceptsRealisticAnchor guards against over-rejection (#32): a
// realistic canonical batch — a 59-char CIDv1 anchor CID, embedded CAS URIs,
// reveal values longer than the (removed) 50-byte cap, and a real ~900-byte
// delta — must process cleanly. This is the regression test for the two
// over-rejections (reveal length, anchor-CID length) the reference does not have.
func TestProcessorAcceptsRealisticAnchor(t *testing.T) {
	cas := NewTestCAS()
	insert := func(uri string, v interface{}) {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("failed to marshal %s: %v", uri, err)
		}
		cas.insertObject(uri, b)
	}

	// A reveal value longer than the old 50-byte cap must not be rejected.
	longReveal := strings.Repeat("a", 70)
	// A real delta well under 1000 canonical bytes (one add-public-keys patch).
	realDelta := did.Delta{
		UpdateCommitment: "EiB_realistic_update_commitment_value_000000",
		Patches: []map[string]interface{}{{
			"action": "add-public-keys",
			"publicKeys": []map[string]interface{}{{
				"id":           "key-1",
				"type":         "EcdsaSecp256k1VerificationKey2019",
				"publicKeyJwk": map[string]string{"kty": "EC", "crv": "secp256k1", "x": strings.Repeat("x", 43), "y": strings.Repeat("y", 43)},
				"purposes":     []string{"authentication"},
			}},
		}},
	}

	insert("chunk-uri", ChunkFile{Deltas: []did.Delta{realDelta, realDelta, realDelta}})
	insert("prov-proof-uri", ProvisionalProofFile{
		Operations: ProvProofOperations{Update: []SignedUpdateDataOp{{SignedData: "signed-data"}}},
	})
	insert("prov-index-uri", ProvisionalIndexFile{
		ProvisionalProofURI: "prov-proof-uri",
		Operations:          ProvOPS{Update: []Operation{{DIDSuffix: "update-did", RevealValue: longReveal}}},
		Chunks:              []ProvChunk{{ChunkFileURI: "chunk-uri"}},
	})
	insert("core-proof-uri", CoreProofFile{
		Operations: CoreProofOperations{
			Recover:    []SignedRecoverDataOp{{SignedData: "signed-data"}},
			Deactivate: []SignedDeactivateDataOp{{SignedData: "signed-data"}},
		},
	})
	insert("core-index-uri", CoreIndexFile{
		ProvisionalIndexURI: "prov-index-uri",
		CoreProofURI:        "core-proof-uri",
		Operations: CoreOperations{
			Recover:    []Operation{{DIDSuffix: "recover-did", RevealValue: longReveal}},
			Deactivate: []Operation{{DIDSuffix: "deactivate-did", RevealValue: longReveal}},
			Create:     []CreateOperation{{SuffixData: did.SuffixData{DeltaHash: "abc123", RecoveryCommitment: "xyz789"}}},
		},
	})

	// 4 operations (create + recover + deactivate + update); 3 deltas
	// (create + recover + update).
	p, err := Processor(operations.Anchor{Anchor: "4.core-index-uri"}, WithCAS(cas), WithPrefix("test"))
	if err != nil {
		t.Fatalf("expected no error creating processor, got %v", err)
	}
	if got := p.Process(); got.Error != nil {
		t.Fatalf("realistic anchor was wrongly rejected: %v", got.Error)
	}
}

// TestProcessorProvisionalFieldCaps verifies the per-field caps enforced when
// parsing the provisional index file (#32): its embedded CAS URIs.
func TestProcessorProvisionalFieldCaps(t *testing.T) {
	tooLongURI := strings.Repeat("u", MaxCASURILength+1)

	tests := map[string]struct {
		provIndex ProvisionalIndexFile
		wantErr   error
	}{
		"provisional proof uri too long": {
			provIndex: ProvisionalIndexFile{ProvisionalProofURI: tooLongURI},
			wantErr:   ErrCASURITooLong,
		},
		"chunk file uri too long": {
			provIndex: ProvisionalIndexFile{Chunks: []ProvChunk{{ChunkFileURI: tooLongURI}}},
			wantErr:   ErrCASURITooLong,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cas := NewTestCAS()

			pi, err := json.Marshal(test.provIndex)
			if err != nil {
				t.Fatalf("failed to marshal provisional index: %v", err)
			}
			cas.insertObject("prov-index-uri", pi)

			ci, err := json.Marshal(CoreIndexFile{ProvisionalIndexURI: "prov-index-uri"})
			if err != nil {
				t.Fatalf("failed to marshal core index: %v", err)
			}
			cas.insertObject("cid", ci)

			p, err := Processor(operations.Anchor{Anchor: "1.cid"}, WithCAS(cas), WithPrefix("test"))
			if err != nil {
				t.Fatalf("expected no error creating processor, got %v", err)
			}

			got := p.Process()
			if !checkError(got.Error, test.wantErr) {
				t.Errorf("expected %v, got %v", test.wantErr, got.Error)
			}
			if !errors.Is(got.Error, ErrMalformed) {
				t.Errorf("expected rejection to be ErrMalformed (permanent), got %v", got.Error)
			}
		})
	}
}

// TestProcessorDeltaSizeCap verifies that an operation delta whose canonicalized
// size exceeds MaxDeltaSizeInBytes is rejected when the chunk file is processed (#32).
func TestProcessorDeltaSizeCap(t *testing.T) {
	oversizedDelta := did.Delta{UpdateCommitment: strings.Repeat("x", MaxDeltaSizeInBytes+200)}
	chunk, err := json.Marshal(ChunkFile{Deltas: []did.Delta{oversizedDelta}})
	if err != nil {
		t.Fatalf("failed to marshal chunk: %v", err)
	}
	runDeltaSizeTest(t, chunk, ErrDeltaTooLarge)
}

// TestProcessorDeltaSizeCapCountsUnknownFields verifies the cap is measured on
// the raw on-wire delta bytes, not a re-marshaled did.Delta. A delta that is
// small once parsed (unknown fields dropped) but large on the wire (the bytes
// the reference canonicalizes and sizes) must still be rejected — otherwise a
// writer could hide bytes in fields our struct ignores to evade the cap.
func TestProcessorDeltaSizeCapCountsUnknownFields(t *testing.T) {
	// Hand-build the chunk JSON so the delta carries an unknown top-level field
	// that json.Unmarshal into did.Delta would drop.
	padding := strings.Repeat("z", MaxDeltaSizeInBytes+200)
	rawChunk := []byte(`{"deltas":[{"patches":[],"updateCommitment":"c","unknownField":"` + padding + `"}]}`)
	runDeltaSizeTest(t, rawChunk, ErrDeltaTooLarge)
}

// runDeltaSizeTest stages a minimal create-only batch whose single delta is the
// provided chunk file, and asserts Process() rejects it with wantErr (permanent).
func runDeltaSizeTest(t *testing.T, chunkJSON []byte, wantErr error) {
	t.Helper()
	cas := NewTestCAS()
	cas.insertObject("chunk-uri", chunkJSON)

	pi, err := json.Marshal(ProvisionalIndexFile{Chunks: []ProvChunk{{ChunkFileURI: "chunk-uri"}}})
	if err != nil {
		t.Fatalf("failed to marshal provisional index: %v", err)
	}
	cas.insertObject("prov-index-uri", pi)

	ci, err := json.Marshal(CoreIndexFile{
		ProvisionalIndexURI: "prov-index-uri",
		Operations: CoreOperations{
			Create: []CreateOperation{{SuffixData: did.SuffixData{DeltaHash: "d", RecoveryCommitment: "r"}}},
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal core index: %v", err)
	}
	cas.insertObject("cid", ci)

	p, err := Processor(operations.Anchor{Anchor: "1.cid"}, WithCAS(cas), WithPrefix("test"))
	if err != nil {
		t.Fatalf("expected no error creating processor, got %v", err)
	}

	got := p.Process()
	if !checkError(got.Error, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, got.Error)
	}
	if !errors.Is(got.Error, ErrMalformed) {
		t.Errorf("expected rejection to be ErrMalformed (permanent), got %v", got.Error)
	}
}
