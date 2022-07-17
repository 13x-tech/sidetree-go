package sidetree

import (
	"encoding/json"
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/api"
	"github.com/13x-tech/ion-sdk-go/pkg/did"
	"github.com/13x-tech/ion-sdk-go/pkg/operations"
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

func (c *ChunkFile) createdDeltaHash(id string) (string, bool) {
	if c.processor.createdDeltaHash == nil {
		return "", false
	}

	deltaHash, ok := c.processor.createdDeltaHash[id]
	return deltaHash, ok
}

func (c *ChunkFile) recoveryDeltaHash(id string) (string, bool) {
	if c.processor.CoreProofFile == nil || c.processor.CoreProofFile.recoveryDeltaHash == nil {
		return "", false
	}

	deltaHash, ok := c.processor.CoreProofFile.recoveryDeltaHash[id]
	return deltaHash, ok
}

func (c *ChunkFile) updateDeltaHash(id string) (string, bool) {
	if c.processor.ProvisionalProofFile == nil {
		return "", false
	}

	if c.processor.ProvisionalProofFile.verifiedOps == nil {
		return "", false
	}

	deltaHash, ok := c.processor.ProvisionalProofFile.verifiedOps[id]
	return deltaHash, ok
}

func (c *ChunkFile) checkDeltaHash(id, hash string) error {
	if deltaHash, ok := c.createdDeltaHash(id); ok {

		if deltaHash != hash {
			return fmt.Errorf("delta hash does not match for created: %s", id)
		}

	} else if deltaHash, ok = c.recoveryDeltaHash(id); ok {

		if deltaHash != hash {
			return fmt.Errorf("delta hash does not match for recovery: %s", id)
		}

	} else if deltaHash, ok = c.updateDeltaHash(id); ok {

		if deltaHash != hash {
			return fmt.Errorf("delta hash does not match for operation: %s", id)
		}
	}
	return nil
}

func (c *ChunkFile) processDelta(index int, delta did.Delta) error {
	id := c.processor.deltaMappingArray[index]

	suffixData, isCreate := c.processor.CoreIndexFile.createSuffix[id]
	if isCreate {

		createOp := api.CreateOperation(suffixData, delta)
		didOps, err := operations.New(
			operations.WithMethod(c.processor.method),
			operations.WithOperations(
				createOp,
			),
		)
		if err != nil {
			return fmt.Errorf("could not create new operation: %w", err)
		}

		opsData, err := didOps.SerializedOps()
		if err != nil {
			return fmt.Errorf("could not serialize ops data: %w", err)
		}
		if err := c.processor.didStore.PutOps(id, opsData); err != nil {
			return fmt.Errorf("could not store ops: %w", err)
		}
		return nil
	}

	didOpsB, err := c.processor.didStore.GetOps(id)
	if err != nil {
		return fmt.Errorf("could not get operations: %w", err)
	}

	_, err = api.ParseOps(didOpsB)
	if err != nil {
		return fmt.Errorf("could not parse ops: %w", err)
	}

	return nil
}
