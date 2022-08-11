package sidetree

import (
	"encoding/json"
	"fmt"
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
				operations.Anchor{Anchor: "1.core-index-uri"},
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
