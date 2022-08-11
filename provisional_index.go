package sidetree

import (
	"encoding/json"
	"fmt"
)

var (
	ErrProvisionalProofURIEmpty = fmt.Errorf("provisional proof uri is empty")
	ErrMultipleChunks           = fmt.Errorf("provisional index file contains invalid chunk count")
)

func NewProvisionalIndexFile(processor *OperationsProcessor, data []byte) (*ProvisionalIndexFile, error) {
	var p ProvisionalIndexFile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provisional index file: %w", err)
	}
	p.processor = processor
	return &p, nil
}

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

	p.processor.provisionalProofFileURI = p.ProvisionalProofURI

	if err := p.processor.populateDeltaMappingArray(); err != nil {
		return fmt.Errorf("failed to populate delta mapping array: %w", err)
	}

	if err := p.populateCoreOperationArray(); err != nil {
		return fmt.Errorf("failed to populate core operation storage array: %w", err)
	}

	p.setRevealValues()

	// Version 1 of SideTree only contains a single chunk in the chunks array
	if len(p.Chunks) != 1 {
		return ErrMultipleChunks
	}

	chunk := p.Chunks[0]
	if chunk.ChunkFileURI == "" {
		return fmt.Errorf("chunk file uri is empty")
	}

	p.processor.chunkFileURI = chunk.ChunkFileURI

	return nil
}

func (p *ProvisionalIndexFile) setRevealValues() {
	p.revealValues = map[string]string{}
	for _, op := range p.Operations.Update {
		p.revealValues[op.DIDSuffix] = op.RevealValue
	}
}

func (p *ProvisionalIndexFile) populateCoreOperationArray() error {

	for _, op := range p.Operations.Update {
		if _, ok := p.processor.coreIndexFile.suffixMap[op.DIDSuffix]; ok {
			return ErrDuplicateOperation
		}

		p.processor.coreIndexFile.suffixMap[op.DIDSuffix] = struct{}{}
	}

	if len(p.Operations.Update) > 0 && p.ProvisionalProofURI == "" {
		return ErrProvisionalProofURIEmpty
	}

	return nil
}

type ProvOPS struct {
	Update []Operation `json:"update"`
}

type ProvChunk struct {
	ChunkFileURI string `json:"chunkFileUri"`
}
