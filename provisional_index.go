package sidetree

import (
	"fmt"

	"github.com/13x-tech/sidetree-go/internal/did"
)

type ProvisionalIndexFile struct {
	ProvisionalProofURI string      `json:"provisionalProofFileUri"`
	Operations          ProvOPS     `json:"operations,omitempty"`
	Chunks              []ProvChunk `json:"chunks"`

	revealValues map[string]string
	processor    *OperationsProcessor
}

func (p *ProvisionalIndexFile) Process() error {
	// p.processor.log.Infof("Processing provisional index file %s", p.processor.ProvisionalIndexFileURI)

	// TODO Check Max Provisional Index File Size

	p.processor.ProvisionalProofFileURI = p.ProvisionalProofURI

	if err := p.processor.populateDeltaMappingArray(); err != nil {
		return fmt.Errorf("failed to populate delta mapping array: %w", err)
	}

	if err := p.populateCoreOperationArray(); err != nil {
		return fmt.Errorf("failed to populate core operation storage array: %w", err)
	}

	if err := p.processRevealValues(); err != nil {
		return fmt.Errorf("failed to process reveal values: %w", err)
	}

	// Version 1 of SideTree only contains a single chunk in the chunks array
	if len(p.Chunks) != 1 {
		return fmt.Errorf("provisional index file contains more than one chunk")
	}

	chunk := p.Chunks[0]
	if chunk.ChunkFileURI == "" {
		return fmt.Errorf("chunk file uri is empty")
	}

	p.processor.ChunkFileURI = chunk.ChunkFileURI

	return nil
}

func (p *ProvisionalIndexFile) processRevealValues() error {

	if len(p.Operations.Update) == 0 {
		return nil
	}

	p.revealValues = map[string]string{}

	for _, op := range p.Operations.Update {
		updateCommitment, err := p.processor.getUpdateCommitment(op.DIDSuffix)
		if err != nil {
			p.processor.log.Errorf(
				"core index: %s - failed to get update commitment for %s: %w",
				p.processor.CoreIndexFileURI,
				op.DIDSuffix,
				err,
			)
			continue
		}
		if did.CheckReveal(op.RevealValue, updateCommitment) {
			p.revealValues[op.DIDSuffix] = op.RevealValue
		}
	}
	return nil
}

func (p *ProvisionalIndexFile) populateCoreOperationArray() error {

	for _, op := range p.Operations.Update {
		if _, ok := p.processor.CoreIndexFile.suffixMap[op.DIDSuffix]; ok {
			return fmt.Errorf("duplicate operation found in recover")
		}

		p.processor.CoreIndexFile.suffixMap[op.DIDSuffix] = struct{}{}
	}

	if len(p.Operations.Update) > 0 && p.ProvisionalProofURI == "" {
		return fmt.Errorf("provisional proof uri is empty")
	}

	return nil
}

type ProvOPS struct {
	Update []Operation `json:"update"`
}

type ProvChunk struct {
	ChunkFileURI string `json:"chunkFileUri"`
}
