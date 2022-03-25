package sidetree

type CoreProofFile struct {
	Operations CoreProofOperations `json:"operations"`

	processor *OperationsProcessor
}

func (p *CoreProofFile) Process() error {
	// p.processor.log.Infof("Processing core proof file %s", p.processor.CoreProofFileURI)
	//TODO Check Max Core Proof File Size

	return nil
}

type CoreProofOperations struct {
	Recover    []SignedDataOp `json:"recover"`
	Deactivate []SignedDataOp `json:"deactivate"`
}
