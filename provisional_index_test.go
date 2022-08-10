package sidetree

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestSetRevealValues(t *testing.T) {
	tests := map[string]struct {
		Operations ProvOPS
		want       int
	}{
		"set reveals": {
			Operations: ProvOPS{
				Update: []Operation{{
					DIDSuffix:   "abc",
					RevealValue: "abc",
				}, {
					DIDSuffix:   "def",
					RevealValue: "def",
				}},
			},
			want: 2,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := ProvisionalIndexFile{Operations: test.Operations}
			p.setRevealValues()
			if len(p.revealValues) != test.want {
				t.Errorf("got %d, want %d", len(p.revealValues), test.want)
			}
		})
	}
}

func TestCoreOperationsArray(t *testing.T) {
	tests := map[string]struct {
		suffixMap map[string]struct{}
		ProofURI  string
		Update    []Operation
		want      error
	}{
		"duplicate operation": {
			suffixMap: map[string]struct{}{
				"abc": {},
			},
			ProofURI: "some-proof-uri",
			Update: []Operation{{
				DIDSuffix: "abc",
			}},
			want: ErrDuplicateOperation,
		},
		"non duplicate": {
			suffixMap: map[string]struct{}{},
			ProofURI:  "some-proof-uri",
			Update: []Operation{{
				DIDSuffix: "abc",
			}},
			want: nil,
		},
		"no proof uri": {
			suffixMap: map[string]struct{}{},
			Update: []Operation{{
				DIDSuffix: "abc",
			}},
			want: ErrProvisionalProofURIEmpty,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := &OperationsProcessor{
				CoreIndexFile: &CoreIndexFile{
					suffixMap: test.suffixMap,
				},
			}

			pi := ProvisionalIndexFile{
				processor: p,
				Operations: ProvOPS{
					Update: test.Update,
				},
				ProvisionalProofURI: test.ProofURI,
			}
			if err := pi.populateCoreOperationArray(); err != test.want {
				t.Errorf("got %v, want %v", err, test.want)
			}
		})
	}

}

func TestNewProvisionalIndexFile(t *testing.T) {
	t.Run("bad data", func(t *testing.T) {
		p := &OperationsProcessor{}
		_, err := NewProvisionalIndexFile(
			p,
			[]byte("bad data"),
		)
		if !strings.Contains(err.Error(), "failed to unmarshal") {
			t.Errorf("got %v, want a failed to unmarshal error", err)
		}
	})
	t.Run("good data", func(t *testing.T) {
		p := &OperationsProcessor{}
		_, err := NewProvisionalIndexFile(
			p,
			[]byte("{}"),
		)
		if err != nil {
			t.Errorf("got %v, want no error", err)
		}
	})

}

func TestProcessProvisionalIndexFile(t *testing.T) {
	tests := map[string]struct {
		ProofURI   string
		Operations ProvOPS
		IndexFile  *ProvisionalIndexFile
		Chunks     []ProvChunk
		want       error
	}{
		"multiple chunks": {
			ProofURI:   "some-proof-uri",
			Operations: ProvOPS{},
			IndexFile:  &ProvisionalIndexFile{},
			Chunks:     []ProvChunk{{}, {}},
			want:       ErrMultipleChunks,
		},
		"does not have index file": {
			ProofURI:   "some-proof-uri",
			Operations: ProvOPS{},
			Chunks:     []ProvChunk{{}},
			want:       fmt.Errorf("provisional index file is nil"),
		},
		"no proof uri": {
			ProofURI: "",
			Operations: ProvOPS{
				Update: []Operation{{
					DIDSuffix: "abc",
				}},
			},
			IndexFile: &ProvisionalIndexFile{},
			Chunks:    []ProvChunk{{}},
			want:      ErrProvisionalProofURIEmpty,
		},
		"no chunk file uri": {
			ProofURI:  "some-proof-uri",
			IndexFile: &ProvisionalIndexFile{},
			Operations: ProvOPS{
				Update: []Operation{{
					DIDSuffix: "abc",
				}},
			},
			Chunks: []ProvChunk{{
				ChunkFileURI: "",
			}},
			want: fmt.Errorf("chunk file uri is empty"),
		},
		"no error": {
			ProofURI:  "some-proof-uri",
			IndexFile: &ProvisionalIndexFile{},
			Operations: ProvOPS{
				Update: []Operation{{
					DIDSuffix: "abc",
				}},
			},
			Chunks: []ProvChunk{{
				ChunkFileURI: "some-chunk-file",
			}},
			want: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := &OperationsProcessor{
				CoreIndexFile: &CoreIndexFile{
					suffixMap: map[string]struct{}{},
				},
				ProvisionalIndexFile: test.IndexFile,
			}
			jsonData, err := json.Marshal(ProvisionalIndexFile{
				ProvisionalProofURI: test.ProofURI,
				Operations:          test.Operations,
				Chunks:              test.Chunks,
			})
			if err != nil {
				t.Fatalf("json marshal error got %v, want no error", err)
			}

			pi, err := NewProvisionalIndexFile(p, jsonData)
			if err != nil {
				t.Fatalf("new provisional index error got %v, want no error", err)
			}
			if err := pi.Process(); !checkError(err, test.want) {
				t.Errorf("process error got %v, want %v", err, test.want)
			}

		})
	}
}
