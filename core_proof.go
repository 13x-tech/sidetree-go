package sidetree

import (
	"encoding/json"
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
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

	if len(p.Operations.Recover) != len(p.processor.CoreIndexFile.Operations.Recover) ||
		len(p.Operations.Deactivate) != len(p.processor.CoreIndexFile.Operations.Deactivate) {
		return fmt.Errorf("core proof and core index file do not match")
	}

	for i, op := range p.Operations.Recover {
		coreOp := p.processor.CoreIndexFile.Operations.Recover[i]
		if err := p.processRecover(coreOp.DIDSuffix, coreOp.RevealValue, op); err != nil {
			p.processor.log.Errorf(
				"core index: %s - failed to process core proof recovery operation for %s: %w",
				p.processor.CoreIndexFileURI,
				coreOp.DIDSuffix,
				err,
			)
		}
	}

	for i, op := range p.Operations.Deactivate {
		coreOp := p.processor.CoreIndexFile.Operations.Deactivate[i]
		if err := p.processDeactivate(coreOp.DIDSuffix, coreOp.RevealValue, op); err != nil {
			p.processor.log.Errorf(
				"core index: %s - failed to process core proof deactivate operation for %s: %w",
				p.processor.CoreIndexFileURI,
				coreOp.DIDSuffix,
				err,
			)
		}
	}

	return nil
}

func (p *CoreProofFile) processDeactivate(id string, revealValue string, op SignedDeactivateDataOp) error {
	deactivate := operations.DeactivateOperation(
		p.processor.Anchor(),
		p.processor.SystemAnchor(),
		id,
		revealValue,
		op.SignedData,
	)
	p.processor.deactivateOps[id] = deactivate

	//TODO Process this Deactivate Here

	return nil
}

func (p *CoreProofFile) processRecover(id string, revealValue string, op SignedRecoverDataOp) error {
	recover := operations.RecoverOperation(
		p.processor.Anchor(),
		p.processor.SystemAnchor(),
		id,
		revealValue,
		op.SignedData,
	)
	p.processor.recoverOps[id] = recover

	return nil
}

type CoreProofOperations struct {
	Recover    []SignedRecoverDataOp    `json:"recover"`
	Deactivate []SignedDeactivateDataOp `json:"deactivate"`
}
