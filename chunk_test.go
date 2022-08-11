package sidetree

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

type TestDeltaCreate struct {
	t *TestSetDeltas
	operations.CreateInterface
}

func (t *TestDeltaCreate) SetDelta(delta did.Delta) {
	t.t.setDeltaCount++
}

type TestDeltaUpdate struct {
	t *TestSetDeltas
	operations.UpdateInterface
}

func (t *TestDeltaUpdate) SetDelta(delta did.Delta) {
	t.t.setDeltaCount++
}

type TestDeltaRecover struct {
	t *TestSetDeltas
	operations.RecoverInterface
}

func (t *TestDeltaRecover) SetDelta(delta did.Delta) {
	t.t.setDeltaCount++
}

type TestSetDeltas struct {
	setDeltaCount int
}

func TestChunkSetDelta(t *testing.T) {

	tests := map[string]struct {
		deltas []did.Delta
		want   int
	}{
		"valid": {
			deltas: []did.Delta{
				{
					UpdateCommitment: "abc",
				},
				{
					UpdateCommitment: "def",
				},
				{
					UpdateCommitment: "xyz",
				},
			},
			want: 3,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			d := &TestSetDeltas{}
			fileJSON, err := json.Marshal(struct {
				Deltas []did.Delta `json:"deltas"`
			}{
				Deltas: test.deltas,
			})

			if err != nil {
				t.Errorf("failed to marshal test data: %v", err)
			}

			c, err := NewChunkFile(
				[]byte(fileJSON),
				WithMappingArrays([]string{"x"}, []string{"z"}, []string{"y"}),
				WithOperations(map[string]operations.CreateInterface{
					"x": &TestDeltaCreate{t: d},
				}, map[string]operations.RecoverInterface{
					"y": &TestDeltaRecover{t: d},
				}, map[string]operations.UpdateInterface{
					"z": &TestDeltaUpdate{t: d},
				}),
			)
			if err != nil {
				t.Fatalf("failed to create chunk: %v", err)
			}

			if err := c.Process(); err != nil {
				t.Fatalf("failed to process chunk: %v", err)
			}
			if d.setDeltaCount != test.want {
				t.Errorf("expected %d deltas to be set, got %d", test.want, d.setDeltaCount)
			}
		})
	}
}

func TestChunkFileProcess(t *testing.T) {

	t.Run("process chunk invalid chunk file", func(t *testing.T) {
		_, err := NewChunkFile([]byte("invalid file"))
		if err == nil {
			t.Errorf("should have failed to process invalid json")
		}
		if strings.Contains(err.Error(), "unable to unmarshal") {
			return
		}
	})

	t.Run("mapping array error", func(t *testing.T) {

		testChunk := ChunkFile{
			Deltas: []did.Delta{
				{
					UpdateCommitment: "abc",
				},
				{
					UpdateCommitment: "def",
				},
				{
					UpdateCommitment: "xyz",
				},
			},
		}

		tests := map[string]struct {
			file       ChunkFile
			createOps  map[string]operations.CreateInterface
			recoverOps map[string]operations.RecoverInterface
			updateOps  map[string]operations.UpdateInterface
			want       error
		}{
			"without create": {
				file:       testChunk,
				createOps:  map[string]operations.CreateInterface{},
				recoverOps: map[string]operations.RecoverInterface{"y": &TestDeltaRecover{t: &TestSetDeltas{}}},
				updateOps:  map[string]operations.UpdateInterface{"z": &TestDeltaUpdate{t: &TestSetDeltas{}}},
				want:       ErrInvalidDeltaCount,
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				createMappingArray := func(maping map[string]operations.CreateInterface) []string {
					ids := []string{}
					for id, _ := range maping {
						ids = append(ids, id)
					}
					return ids
				}(test.createOps)
				recoverMappingArray := func(maping map[string]operations.RecoverInterface) []string {
					ids := []string{}
					for id, _ := range maping {
						ids = append(ids, id)
					}
					return ids
				}(test.recoverOps)
				updateMappingArray := func(maping map[string]operations.UpdateInterface) []string {
					ids := []string{}
					for id, _ := range maping {
						ids = append(ids, id)
					}
					return ids
				}(test.updateOps)

				chunkFile, err := json.Marshal(test.file)
				if err != nil {
					t.Fatalf("failed to marshal test data: %v", err)
				}

				c, err := NewChunkFile(
					chunkFile,
					WithMappingArrays(createMappingArray, recoverMappingArray, updateMappingArray),
					WithOperations(test.createOps, test.recoverOps, test.updateOps),
				)
				if err != nil {
					t.Fatalf("failed to create chunk: %v", err)
				}
				if err := c.Process(); err != test.want {
					t.Errorf("expected error %v, got %v", test.want, err)
				}
			})
		}
	})
}
