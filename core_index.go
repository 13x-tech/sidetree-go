package sidetree

import (
	"encoding/json"
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
)

func NewCoreIndexFile(processor *OperationsProcessor, data []byte) (*CoreIndexFile, error) {

	c := CoreIndexFile{}
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal core index file: %w", err)
	}

	c.processor = processor

	return &c, nil
}

type CoreIndexFile struct {
	ProvisionalIndexURI string         `json:"provisionalIndexFileUri"`
	CoreProofURI        string         `json:"coreProofFileUri"`
	WriterLockId        string         `json:"writerLockId,omitempty"`
	Operations          CoreOperations `json:"operations"`

	suffixMap map[string]struct{}

	processor *OperationsProcessor
}

func (c *CoreIndexFile) Process() error {
	// c.processor.log.Infof("Processing core index file %s with %d operations", c.processor.CoreIndexFileURI, c.processor.opsCount)
	// Core Index File Processing Procedure
	// https://identity.foundation/sidetree/spec/#core-index-file-processing

	//TODO Check Max Core Index File Size

	c.processor.ProvisionalIndexFileURI = c.ProvisionalIndexURI

	if (len(c.Operations.Deactivate) > 0 || len(c.Operations.Recover) > 0) && c.CoreProofURI == "" {
		return ErrNoCoreProof
	}
	c.processor.CoreProofFileURI = c.CoreProofURI

	if err := c.populateCoreOperationArray(); err != nil {
		return fmt.Errorf("failed to populate core operations array: %w", err)
	}

	// TODO Need to Process Create, Recover, Deactivate Ops here?
	return nil
}

func (c *CoreIndexFile) populateCoreOperationArray() error {

	// a Core Index File MUST NOT include multiple operations in the operations
	// section of the Core Index File for the same DID Suffix
	// - if any duplicates are found, cease processing, discard the file data,
	// and retain a reference that the whole batch of anchored operations and all
	// its files are to be ignored.

	c.suffixMap = map[string]struct{}{}

	for _, op := range c.Operations.Create {
		uri, err := op.SuffixData.URI()
		if err != nil {
			return ErrURINotFound
		}
		if _, ok := c.suffixMap[uri]; ok {
			return ErrDuplicateOperation
		}
		c.suffixMap[uri] = struct{}{}
	}

	for _, op := range c.Operations.Recover {
		if _, ok := c.suffixMap[op.DIDSuffix]; ok {
			return ErrDuplicateOperation
		}
		c.suffixMap[op.DIDSuffix] = struct{}{}
	}

	for _, op := range c.Operations.Deactivate {
		if _, ok := c.suffixMap[op.DIDSuffix]; ok {
			return ErrDuplicateOperation
		}
		c.suffixMap[op.DIDSuffix] = struct{}{}
	}

	return nil
}

type CoreOperations struct {
	Create     []CreateOperation `json:"create,omitempty"`
	Recover    []Operation       `json:"recover,omitempty"`
	Deactivate []Operation       `json:"deactivate,omitempty"`
}

type CreateOperation struct {
	SuffixData did.SuffixData `json:"suffixData"`
}

type Operation struct {
	DIDSuffix   string `json:"didSuffix"`
	RevealValue string `json:"revealValue"`
}
