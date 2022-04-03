package sidetree

import (
	"fmt"

	"github.com/13x-tech/sidetree-go/pkg/did"
)

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
	if c.processor.createdDelaHash == nil {
		return "", false
	}

	deltaHash, ok := c.processor.createdDelaHash[id]
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

func (c *ChunkFile) processDelta(index int, delta did.Delta) error {
	id := c.processor.deltaMappingArray[index]

	if deltaHash, ok := c.createdDeltaHash(id); ok {

		hashed, err := delta.Hash()
		if err != nil {
			return fmt.Errorf("failed to hash delta: %w", err)
		}

		if deltaHash != hashed {
			return fmt.Errorf("delta hash does not match for created: %s", id)
		}

	} else if deltaHash, ok = c.recoveryDeltaHash(id); ok {

		hashed, err := delta.Hash()
		if err != nil {
			return fmt.Errorf("failed to hash delta: %w", err)
		}

		if deltaHash != hashed {
			return fmt.Errorf("delta hash does not match for recovery: %s", id)
		}

	} else if deltaHash, ok = c.updateDeltaHash(id); ok {

		hashed, err := delta.Hash()
		if err != nil {
			return fmt.Errorf("failed to hash delta: %w", err)
		}

		if hashed != deltaHash {
			return fmt.Errorf("delta hash does not match for operation: %s", id)
		}
	}

	if err := c.processor.setUpdateCommitment(id, delta.UpdateCommitment); err != nil {
		return fmt.Errorf("failed to set update commitment for %s: %w", id, err)
	}

	for _, patch := range delta.Patches {
		if err := c.processor.patchDelta(id, patch); err != nil {
			c.processor.log.Errorf("core index: %s - failed to patch delta: %w", c.processor.CoreIndexFileURI, err)
		}
	}

	return nil
}
