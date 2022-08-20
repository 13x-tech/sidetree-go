package sidetree

import (
	"encoding/json"
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

var (
	ErrCoreProofCount = fmt.Errorf("core proof count mismatch")
)

func NewCoreProofFile(processor *OperationsProcessor, data []byte) (*CoreProofFile, error) {
	var c CoreProofFile
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal core proof file: %w", err)
	}
	c.processor = processor

	return &c, nil
}

type CoreProofFile struct {
	Operations CoreProofOperations `json:"operations"`

	processor *OperationsProcessor
}

func (p *CoreProofFile) Process() error {
	// p.processor.log.Infof("Processing core proof file %s", p.processor.CoreProofFileURI)
	//TODO Check Max Core Proof File Size

	if len(p.Operations.Recover) != len(p.processor.coreIndexFile.Operations.Recover) ||
		len(p.Operations.Deactivate) != len(p.processor.coreIndexFile.Operations.Deactivate) {
		return ErrCoreProofCount
	}

	for i, op := range p.Operations.Recover {
		coreOp := p.processor.coreIndexFile.Operations.Recover[i]
		p.setRecoverOp(coreOp.DIDSuffix, coreOp.RevealValue, op)
	}

	for i, op := range p.Operations.Deactivate {
		coreOp := p.processor.coreIndexFile.Operations.Deactivate[i]
		p.setDeactivateOp(coreOp.DIDSuffix, coreOp.RevealValue, op)
	}

	return nil
}

func (p *CoreProofFile) setDeactivateOp(id string, revealValue string, op SignedDeactivateDataOp) {
	p.processor.deactivateOps[id] = operations.DeactivateOperation(
		id,
		revealValue,
		op.SignedData,
	)
}

func (p *CoreProofFile) setRecoverOp(id string, revealValue string, op SignedRecoverDataOp) {
	p.processor.recoverOps[id] = operations.RecoverOperation(
		id,
		revealValue,
		op.SignedData,
	)
}

type CoreProofOperations struct {
	Recover    []SignedRecoverDataOp    `json:"recover"`
	Deactivate []SignedDeactivateDataOp `json:"deactivate"`
}
