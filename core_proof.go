package sidetree

import "fmt"

type CoreProofFile struct {
	Operations CoreProofOperations `json:"operations"`

	recoveryDeltaHash map[string]string
	processor         *OperationsProcessor
}

func (p *CoreProofFile) Process() error {
	// p.processor.log.Infof("Processing core proof file %s", p.processor.CoreProofFileURI)
	//TODO Check Max Core Proof File Size

	if len(p.Operations.Recover) != len(p.processor.CoreIndexFile.Operations.Recover) ||
		len(p.Operations.Deactivate) != len(p.processor.CoreIndexFile.Operations.Deactivate) {
		return fmt.Errorf("core proof and core index file do not match")
	}

	p.recoveryDeltaHash = make(map[string]string, len(p.Operations.Recover))

	for i, op := range p.Operations.Recover {
		coreOp := p.processor.CoreIndexFile.Operations.Recover[i]
		if err := p.processRecover(coreOp.DIDSuffix, coreOp.RevealValue, op); err != nil {
			p.processor.log.Errorf("core index: %s - failed to process core proof recovery operation for %s: %w", p.processor.CoreIndexFileURI, coreOp.DIDSuffix, err)
		}
	}

	for i, op := range p.Operations.Deactivate {
		coreOp := p.processor.CoreIndexFile.Operations.Deactivate[i]
		if err := p.processDeactivate(coreOp.DIDSuffix, coreOp.RevealValue, op); err != nil {
			p.processor.log.Errorf("core index: %s - failed to process core proof deactivate operation for %s: %w", p.processor.CoreIndexFileURI, coreOp.DIDSuffix, err)
		}
	}

	return nil
}

func (p *CoreProofFile) processDeactivate(id string, revealValue string, op SignedDeactivateDataOp) error {
	if ok, err := op.ValidateReveal(revealValue); err != nil {
		return fmt.Errorf("failed to validate reveal value in core proof recover for %s: %w", id, err)
	} else if !ok {
		return fmt.Errorf("failed to validate reveal value in core proof recover for %s", id)
	}

	didDoc, err := p.processor.didStore.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get did document for %s: %w", id, err)
	}

	//TODO Better way to mark document as deactivated
	didDoc.Metadata.Method.UpdateCommitment = ""

	if err := p.processor.didStore.Put(didDoc); err != nil {
		return fmt.Errorf("failed to put did document for %s: %w", id, err)
	}

	if err := p.processor.didStore.Deactivate(id); err != nil {
		return fmt.Errorf("failed to deactivate did document for %s: %w", id, err)
	}

	return nil
}

func (p *CoreProofFile) processRecover(id string, revealValue string, op SignedRecoverDataOp) error {
	if ok, err := op.ValidateReveal(revealValue); err != nil {
		return fmt.Errorf("failed to validate reveal value in core proof recover for %s: %w", id, err)
	} else if !ok {
		return fmt.Errorf("failed to validate reveal value in core proof recover for %s", id)
	}

	deltaHash, err := op.DeltaHash()
	if err != nil {
		return fmt.Errorf("failed to get delta hash for %s: %w", id, err)
	}

	p.recoveryDeltaHash[id] = deltaHash

	if err := p.processor.didStore.Recover(id); err != nil {
		return fmt.Errorf("failed to recover did document for %s: %w", id, err)
	}

	didDoc, err := p.processor.didStore.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get did document for %s: %w", id, err)
	}

	didDoc.DIDDocument.ResetData()

	if err := p.processor.didStore.Put(didDoc); err != nil {
		return fmt.Errorf("failed to put did document for %s: %w", id, err)
	}

	return nil
}

type CoreProofOperations struct {
	Recover    []SignedRecoverDataOp    `json:"recover"`
	Deactivate []SignedDeactivateDataOp `json:"deactivate"`
}
