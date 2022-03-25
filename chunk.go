package sidetree

import "fmt"

type ChunkFile struct {
	Deltas            []Delta `json:"deltas"`
	deltaMappingArray []string

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
	if err := c.populateDeltaMappingArray(); err != nil {
		return fmt.Errorf("failed to populate operation mapping array: %w", err)
	}

	if len(c.deltaMappingArray) > len(c.Deltas) {
		return fmt.Errorf("operation mapping array contains more entries than delta entries")
	}

	for i, delta := range c.Deltas {
		if err := c.processDelta(i, delta); err != nil {
			c.processor.log.Errorf("core index: %s - failed to process delta: %w", c.processor.CoreIndexFileURI, err)
		}
	}

	return nil
}

func (c *ChunkFile) processDelta(index int, delta Delta) error {
	id := c.deltaMappingArray[index]

	// TODO Check validation of each delta in order to proceed to patching

	if _, ok := c.processor.CoreIndexFile.createdOps[id]; !ok {
		if c.processor.ProvisionalProofFile == nil {
			return fmt.Errorf("provisional proof file is nil")
		}

		if c.processor.ProvisionalProofFile.verifiedOps == nil {
			return fmt.Errorf("provisional proof file verified ops is nil")
		}

		verifiedId, ok := c.processor.ProvisionalProofFile.verifiedOps[index]
		if !ok {
			return fmt.Errorf("operation not found in provisional proof file")
		}
		if id != verifiedId {
			return fmt.Errorf("operation id mismatch")
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

func (c *ChunkFile) populateDeltaMappingArray() error {
	coreIndex := c.processor.CoreIndexFile
	if coreIndex == nil {
		return fmt.Errorf("core index file is nil")
	}

	provisionalIndex := c.processor.ProvisionalIndexFile

	for _, op := range coreIndex.Operations.Create {
		uri, err := op.SuffixData.URI()
		if err != nil {
			return fmt.Errorf("failed to get uri from create operation: %w", err)
		}

		c.deltaMappingArray = append(c.deltaMappingArray, uri)
	}

	for _, ok := range coreIndex.Operations.Recover {
		c.deltaMappingArray = append(c.deltaMappingArray, ok.DIDSuffix)
	}

	if provisionalIndex != nil {
		for _, op := range provisionalIndex.Operations.Update {
			c.deltaMappingArray = append(c.deltaMappingArray, op.DIDSuffix)
		}
	}

	return nil
}

type Delta struct {
	Patches          []map[string]interface{} `json:"patches"`
	UpdateCommitment string                   `json:"updateCommitment"`
}
