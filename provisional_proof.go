package sidetree

import (
	"fmt"
)

type ProvisionalProofFile struct {
	Operations PorvProofOperations `json:"operations"`

	deltaMappingArray []string
	verifiedOps       map[string]string
	processor         *OperationsProcessor
}

func (p *ProvisionalProofFile) Process() error {
	// p.processor.log.Infof("Processing provisional proof file %s", p.processor.ProvisionalProofFileURI)
	//TODO Check Max Provisional Proof File Size

	if len(p.Operations.Update) == len(p.processor.ProvisionalIndexFile.Operations.Update) {

		if err := p.populateDeltaMappingArray(); err != nil {
			return fmt.Errorf("failed to populate delta mapping array: %w", err)
		}

		p.verifiedOps = make(map[string]string, len(p.Operations.Update))
		for i, op := range p.Operations.Update {
			if err := p.processUpdate(i, op); err != nil {
				p.processor.log.Errorf("Failed to process provisional proof operation %d: %w", i, err)
			}
		}

	} else {
		return fmt.Errorf("provisional proof and provisional index file do not match")
	}

	return nil
}

func (p *ProvisionalProofFile) processUpdate(index int, update SignedUpdateDataOp) error {
	id := p.deltaMappingArray[index]

	revealValue, ok := p.processor.ProvisionalIndexFile.revealValues[id]
	if !ok {
		return fmt.Errorf("failed to find reveal value for %s", id)
	}

	if ok, err := update.ValidateReveal(revealValue); err != nil {
		return fmt.Errorf("failed to validate reveal value in provisional update for %s: %w", id, err)
	} else if !ok {
		return fmt.Errorf("failed to validate reveal value in provisional update for %s", id)
	}

	deltaHash, err := update.DeltaHash()
	if err != nil {
		return fmt.Errorf("failed to get delta hash for %s: %w", id, err)
	}

	p.verifiedOps[id] = deltaHash

	return nil
}

func (p *ProvisionalProofFile) populateDeltaMappingArray() error {
	coreIndex := p.processor.CoreIndexFile
	if coreIndex == nil {
		return fmt.Errorf("core index file is nil")
	}

	provisionalIndex := p.processor.ProvisionalIndexFile

	for _, op := range coreIndex.Operations.Create {
		uri, err := op.SuffixData.URI()
		if err != nil {
			return fmt.Errorf("failed to get uri from create operation: %w", err)
		}

		p.deltaMappingArray = append(p.deltaMappingArray, uri)
	}

	for _, ok := range coreIndex.Operations.Recover {
		p.deltaMappingArray = append(p.deltaMappingArray, ok.DIDSuffix)
	}

	if provisionalIndex != nil {
		for _, op := range provisionalIndex.Operations.Update {
			p.deltaMappingArray = append(p.deltaMappingArray, op.DIDSuffix)
		}
	}

	return nil
}

type PorvProofOperations struct {
	Update []SignedUpdateDataOp `json:"update"`
}
