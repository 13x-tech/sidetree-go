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
