package sidetree

import (
	"encoding/json"
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
)

func NewChunkFile(processor *OperationsProcessor, data []byte) (*ChunkFile, error) {
	var c ChunkFile
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chunk: %w", err)
	}
	c.processor = processor
	return &c, nil
}

type ChunkFile struct {
	Deltas []did.Delta `json:"deltas"`

	processor *OperationsProcessor
}

func (c *ChunkFile) Process() error {
	// c.processor.log.Infof("Processing chunk file %s", c.processor.ChunkFileURI)
	// TODO Check Max Chunk File Size

	// In order to process Chunk File Delta Entries in relation to the DIDs they
	// are bound to, they must be mapped back to the Create, Recovery,
	// and Update operation entries present in the Core Index File and
	// Provisional Index File. To create this mapping, concatenate the
	// Core Index File Create Entries, Core Index File Recovery Entries,
	// Provisional Index File Update Entries into a single array, in that order,
	// herein referred to as the Operation Delta Mapping Array

	if len(c.processor.deltaMappingArray) > len(c.Deltas) {
		return fmt.Errorf("operation mapping array contains more entries than delta entries")
	}

	for i, delta := range c.Deltas {
		if err := c.processDelta(i, delta); err != nil {
			c.processor.log.Errorf("core index: %s - failed to process delta: %w", c.processor.CoreIndexFileURI, err)
		}
	}

	return nil
}

func (c *ChunkFile) processDelta(index int, delta did.Delta) error {
	id := c.processor.deltaMappingArray[index]

	if createOp, ok := c.processor.createOps[id]; ok {
		createOp.SetDelta(delta)
	} else if recoverOp, ok := c.processor.recoverOps[id]; ok {
		recoverOp.SetDelta(delta)
	} else if updateOp, ok := c.processor.updateOps[id]; ok {
		updateOp.SetDelta(delta)
	}

	return nil
}
