package sidetree

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
	mh "github.com/multiformats/go-multihash"
)

type CoreIndexFile struct {
	ProvisionalIndexURI string         `json:"provisionalIndexFileUri"`
	CoreProofURI        string         `json:"coreProofFileUri"`
	WriterLockId        string         `json:"writerLockId,omitempty"`
	Operations          CoreOperations `json:"operations"`

	createdOps map[string]struct{}
	processor  *OperationsProcessor
}

func (c *CoreIndexFile) Process() error {
	// c.processor.log.Infof("Processing core index file %s with %d operations", c.processor.CoreIndexFileURI, c.processor.opsCount)
	// Core Index File Processing Procedure
	// https://identity.foundation/sidetree/spec/#core-index-file-processing

	//TODO Check Max Core Index File Size

	c.processor.ProvisionalIndexFileURI = c.ProvisionalIndexURI

	if (len(c.Operations.Deactivate) > 0 || len(c.Operations.Recover) > 0) && c.CoreProofURI == "" {
		return fmt.Errorf("core proof uri is empty")
	} else {
		c.processor.CoreProofFileURI = c.CoreProofURI
	}

	if err := c.populateCoreOperationArray(); err != nil {
		return fmt.Errorf("failed to populate core operation storage array: %w", err)
	}

	if err := c.processCreateOperations(); err != nil {
		return fmt.Errorf("failed to process create operations: %w", err)
	}

	// TODO Process Recovery Ops
	// TODO Process Deactivate Ops

	return nil
}

func (c *CoreIndexFile) populateCoreOperationArray() error {

	// a Core Index File MUST NOT include multiple operations in the operations
	// section of the Core Index File for the same DID Suffix
	// - if any duplicates are found, cease processing, discard the file data,
	// and retain a reference that the whole batch of anchored operations and all
	// its files are to be ignored.
	suffixMap := make(map[string]struct{})

	for _, op := range c.Operations.Create {
		uri, err := op.SuffixData.URI()
		if err != nil {
			return fmt.Errorf("failed to get uri: %w", err)
		}

		if _, ok := suffixMap[uri]; ok {
			return fmt.Errorf("duplicate operation found in create")
		}

		c.processor.operationStorage = append(c.processor.operationStorage, uri)
	}

	for _, op := range c.Operations.Recover {
		if _, ok := suffixMap[op.DIDSuffix]; ok {
			return fmt.Errorf("duplicate operation found in recover")
		}

		c.processor.operationStorage = append(c.processor.operationStorage, op.DIDSuffix)
	}

	for _, op := range c.Operations.Deactivate {
		if _, ok := suffixMap[op.DIDSuffix]; ok {
			return fmt.Errorf("duplicate operation found in deactivate")
		}

		c.processor.operationStorage = append(c.processor.operationStorage, op.DIDSuffix)
	}

	return nil
}

func (c *CoreIndexFile) processCreateOperations() error {

	c.createdOps = make(map[string]struct{}, len(c.Operations.Create))

	for _, op := range c.Operations.Create {
		uri, err := op.SuffixData.URI()
		if err != nil {
			return fmt.Errorf("failed to get uri: %w", err)
		}
		if err := c.processor.createDID(uri, op.SuffixData.RecoveryCommitment); err != nil {
			return fmt.Errorf("failed to create did: %w", err)
		}
		c.createdOps[uri] = struct{}{}
	}

	return nil
}

type CoreOperations struct {
	Create     []CreateOperation `json:"create,omitempty"`
	Recover    []Operation       `json:"recover,omitempty"`
	Deactivate []Operation       `json:"deactivate,omitempty"`
}

type CreateOperation struct {
	SuffixData SuffixData `json:"suffixData"`
}

type Operation struct {
	DIDSuffix   string `json:"didSuffix"`
	RevealValue string `json:"revealValue"`
}

type SuffixData struct {
	Type               string `json:"type,omitempty"`
	DeltaHash          string `json:"deltaHash"`
	RecoveryCommitment string `json:"recoveryCommitment"`
	AnchorOrigin       string `json:"anchorOrigin,omitempty"`
}

func (s SuffixData) URI() (string, error) {
	// Short Form DID URI
	// https://identity.foundation/sidetree/spec/#short-form-did

	bytes, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("failed to marshal suffix data: %w", err)
	}

	jcsBytes, err := jcs.Transform(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to transform bytes: %w", err)
	}

	h256 := sha256.Sum256(jcsBytes)
	hash, err := mh.Encode(h256[:], mh.SHA2_256)
	if err != nil {
		return "", fmt.Errorf("failed to create hash: %w", err)
	}
	encoder := base64.RawURLEncoding
	return encoder.EncodeToString(hash), nil
}
