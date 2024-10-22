package sidetree

import (
	"encoding/json"
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

var (
	ErrProofIndexMismatch    = fmt.Errorf("provisional proof and provisional index file do not match")
	ErrUpdateMappingMismatch = fmt.Errorf("update operation mapping array contains less entries than update entries")
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
	Operations ProvProofOperations `json:"operations"`

	verifiedOps map[string]string
	processor   *OperationsProcessor
}

func (p *ProvisionalProofFile) Process() error {
	// p.processor.log.Infof("Processing provisional proof file %s", p.processor.ProvisionalProofFileURI)
	//TODO Check Max Provisional Proof File Size

	if len(p.Operations.Update) == len(p.processor.provisionalIndexFile.Operations.Update) {

		if len(p.processor.updateMappingArray) < len(p.Operations.Update) {
			return ErrUpdateMappingMismatch
		}

		p.verifiedOps = map[string]string{}

		for i, op := range p.Operations.Update {
			p.setUpdateOp(i, op)
		}

	} else {
		return ErrProofIndexMismatch
	}

	return nil
}

func (p *ProvisionalProofFile) setUpdateOp(index int, update SignedUpdateDataOp) {
	id := p.processor.updateMappingArray[index]

	reveal, ok := p.processor.provisionalIndexFile.revealValues[id]
	if !ok {
		//fmt.Errorf("core index: %s - failed to find reveal value for id %s", p.processor.CoreIndexFileURI, id)
		return
	}

	p.processor.updateOps[id] = operations.UpdateOperation(
		id,
		reveal,
		update.SignedData,
	)
}

type ProvProofOperations struct {
	Update []SignedUpdateDataOp `json:"update"`
}
