package sidetree

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

var (
	ErrInvalidDeltaCount = errors.New("invalid delta count")
)

type ChunkOption func(c *ChunkFile)

func WithMappingArrays(
	createMappingArray []string,
	recoverMappingArray []string,
	updateMappingArray []string,
) ChunkOption {
	return func(c *ChunkFile) {
		c.createMappingArray = createMappingArray
		c.recoverMappingArray = recoverMappingArray
		c.updateMappingArray = updateMappingArray
	}
}

func WithOperations(
	createOps map[string]operations.CreateInterface,
	recoverOps map[string]operations.RecoverInterface,
	updateOps map[string]operations.UpdateInterface,
) ChunkOption {
	return func(c *ChunkFile) {
		c.createOps = createOps
		c.recoverOps = recoverOps
		c.updateOps = updateOps
	}
}

func NewChunkFile(data []byte, opts ...ChunkOption) (*ChunkFile, error) {
	var c ChunkFile
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unable to unmarshal: %w", err)
	}

	// Capture the raw on-wire bytes of each delta so the per-delta size cap (#32)
	// is measured on what was actually anchored, not a re-marshaled did.Delta
	// (which would silently drop unknown fields and undercount the size).
	var raw struct {
		Deltas []json.RawMessage `json:"deltas"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unable to unmarshal raw deltas: %w", err)
	}
	c.rawDeltas = raw.Deltas

	for _, opt := range opts {
		opt(&c)
	}

	return &c, nil
}

type ChunkFile struct {
	Deltas []did.Delta `json:"deltas"`

	// rawDeltas holds the unparsed on-wire bytes of each entry in Deltas (same
	// order/length), used only for the canonicalized size check.
	rawDeltas []json.RawMessage

	createMappingArray  []string
	recoverMappingArray []string
	updateMappingArray  []string

	createOps  map[string]operations.CreateInterface
	recoverOps map[string]operations.RecoverInterface
	updateOps  map[string]operations.UpdateInterface
}

func (c *ChunkFile) Process() error {
	// c.processor.log.Infof("Processing chunk file %s", c.processor.ChunkFileURI)
	// Max Chunk File Size is enforced at fetch time
	// (fetchChunkFile passes MaxChunkFileSizeInBytes to CAS.Get).

	// In order to process Chunk File Delta Entries in relation to the DIDs they
	// are bound to, they must be mapped back to the Create, Recovery,
	// and Update operation entries present in the Core Index File and
	// Provisional Index File. To create this mapping, concatenate the
	// Core Index File Create Entries, Core Index File Recovery Entries,
	// Provisional Index File Update Entries into a single array, in that order,
	// herein referred to as the Operation Delta Mapping Array
	var mappingArray []string
	if len(c.createMappingArray) > 0 {
		mappingArray = append(mappingArray, c.createMappingArray...)
	}
	if len(c.recoverMappingArray) > 0 {
		mappingArray = append(mappingArray, c.recoverMappingArray...)
	}
	if len(c.updateMappingArray) > 0 {
		mappingArray = append(mappingArray, c.updateMappingArray...)
	}

	if len(mappingArray) != len(c.Deltas) {
		return ErrInvalidDeltaCount
	}

	if len(c.rawDeltas) != len(c.Deltas) {
		return ErrInvalidDeltaCount
	}

	for i, delta := range c.Deltas {
		// Per-field cap (#32): each operation's canonicalized delta must not
		// exceed MaxDeltaSizeInBytes. Measured on the raw on-wire bytes.
		if err := checkDeltaSize(c.rawDeltas[i]); err != nil {
			return err
		}
		id := mappingArray[i]
		c.setDelta(id, delta)
	}

	return nil
}

func (c *ChunkFile) setDelta(id string, delta did.Delta) {
	if createOp, ok := c.createOps[id]; ok {
		createOp.SetDelta(delta)
	} else if recoverOp, ok := c.recoverOps[id]; ok {
		recoverOp.SetDelta(delta)
	} else if updateOp, ok := c.updateOps[id]; ok {
		updateOp.SetDelta(delta)
	}
}
