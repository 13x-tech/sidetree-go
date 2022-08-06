package sidetree

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

var testChunk = ChunkFile{
	Deltas: []did.Delta{
		{
			UpdateCommitment: "abcdefg",
		},
		{
			UpdateCommitment: "hijklmn",
		},
		{
			UpdateCommitment: "opqrstu",
		},
	},
}

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

	index := []string{"x", "y", "z"}

	t.Run("set delta", func(t *testing.T) {
		d := &TestSetDeltas{}

		c, err := NewChunkFile(
			[]byte(`{"deltas":[]}`),
			WithOperations(map[string]operations.CreateInterface{
				"x": &TestDeltaCreate{t: d},
			}, map[string]operations.RecoverInterface{
				"z": &TestDeltaRecover{t: d},
			}, map[string]operations.UpdateInterface{
				"y": &TestDeltaUpdate{t: d},
			}),
		)
		if err != nil {
			t.Errorf("failed to create chunk: %v", err)
		}
		for n, delta := range testChunk.Deltas {
			c.setDelta(index[n], delta)
		}
		if d.setDeltaCount != len(testChunk.Deltas) {
			t.Errorf("expected %d delta, got %d", len(testChunk.Deltas), len(c.Deltas))
		}
	})
}

func TestChunkFileProcess(t *testing.T) {

	testDataJSON, err := json.Marshal(testChunk)
	if err != nil {
		t.Errorf("failed to marshal test data: %v", err)
	}

	t.Run("process chunk invalid chunk file", func(t *testing.T) {
		_, err := NewChunkFile(
			[]byte(``),
		)
		if err == nil {
			t.Errorf("should have failed to process invalid json")
		}
		if strings.Contains(err.Error(), "unable to unmarshal") {
			return
		}
	})

	t.Run("process chunk invalid mapping array count", func(t *testing.T) {
		d := &TestSetDeltas{}

		c, err := NewChunkFile(
			testDataJSON,
			WithMappingArrays([]string{}, []string{"z"}, []string{"y"}),
			WithOperations(map[string]operations.CreateInterface{}, map[string]operations.RecoverInterface{
				"z": &TestDeltaRecover{t: d},
			}, map[string]operations.UpdateInterface{
				"y": &TestDeltaUpdate{t: d},
			}),
		)
		if err != nil {
			t.Errorf("failed to create chunk: %v", err)
		}

		err = c.Process()
		if err == nil || err != ErrInvalidDeltaCount {
			t.Errorf("expected error %v, got %v", ErrInvalidDeltaCount, err)
		}

	})

	t.Run("process empty chunk", func(t *testing.T) {

		c, err := NewChunkFile(
			[]byte(`{"deltas":[]}`),
		)
		if err != nil {
			t.Errorf("failed to create chunk: %v", err)
		}

		err = c.Process()
		if err != nil {
			t.Error("expected no error, got", err)
		}
	})

	t.Run("process chunk", func(t *testing.T) {
		d := &TestSetDeltas{}

		c, err := NewChunkFile(
			testDataJSON,
			WithMappingArrays([]string{"x"}, []string{"z"}, []string{"y"}),
			WithOperations(map[string]operations.CreateInterface{
				"x": &TestDeltaCreate{t: d},
			}, map[string]operations.RecoverInterface{
				"z": &TestDeltaRecover{t: d},
			}, map[string]operations.UpdateInterface{
				"y": &TestDeltaUpdate{t: d},
			}),
		)
		if err != nil {
			t.Errorf("failed to create chunk: %v", err)
		}

		if err := c.Process(); err != nil {
			t.Errorf("failed to process chunk: %v", err)
		}
	})
}
