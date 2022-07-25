package sidetree

import (
	"encoding/json"
	"fmt"
)

func NewProvisionalProofFile(processor *OperationsProcessor, data []byte) (*ProvisionalProofFile, error) {
	var p ProvisionalProofFile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provisional proof file: %w", err)
	}
	p.processor = processor
	return &p, nil
}

type ProvisionalProofFile struct {
	Operations PorvProofOperations `json:"operations"`

	verifiedOps map[string]string
	processor   *OperationsProcessor
}

func (p *ProvisionalProofFile) Process() error {
	// p.processor.log.Infof("Processing provisional proof file %s", p.processor.ProvisionalProofFileURI)
	//TODO Check Max Provisional Proof File Size

	if len(p.Operations.Update) == len(p.processor.ProvisionalIndexFile.Operations.Update) {

		if len(p.processor.updateMappingArray) < len(p.Operations.Update) {
			return fmt.Errorf("update operation mapping array contains less entries than update entries")
		}

		p.verifiedOps = map[string]string{}

		for i, op := range p.Operations.Update {
			if err := p.processUpdate(i, op); err != nil {
				p.processor.log.Errorf(
					"core index: %s - failed to process provisional proof operation %d: %w",
					p.processor.CoreIndexFileURI,
					i,
					err,
				)
			}
		}

	} else {
		return fmt.Errorf("provisional proof and provisional index file do not match")
	}

	return nil
}

func (p *ProvisionalProofFile) processUpdate(index int, update SignedUpdateDataOp) error {
	id := p.processor.updateMappingArray[index]

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

type PorvProofOperations struct {
	Update []SignedUpdateDataOp `json:"update"`
}
