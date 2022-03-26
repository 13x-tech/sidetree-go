package sidetree

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func NewOperationsProcessor(operations int, indexURI string, logger Logger, storage Storage) (*OperationsProcessor, error) {

	didStore, err := storage.DIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to get did store: %w", err)
	}

	casStore, err := storage.CAS()
	if err != nil {
		return nil, fmt.Errorf("failed to get cas store: %w", err)
	}

	return &OperationsProcessor{
		log:              logger,
		opsCount:         operations,
		CoreIndexFileURI: indexURI,
		didStore:         didStore,
		casStore:         casStore,
	}, nil
}

type OperationsProcessor struct {
	log Logger

	CoreIndexFileURI string
	CoreIndexFile    *CoreIndexFile

	CoreProofFileURI string
	CoreProofFile    *CoreProofFile

	ProvisionalIndexFileURI string
	ProvisionalIndexFile    *ProvisionalIndexFile

	ProvisionalProofFileURI string
	ProvisionalProofFile    *ProvisionalProofFile

	// Version 1 only has a single Chunk file No need for Array here yet
	ChunkFileURI string
	ChunkFile    *ChunkFile

	didStore DIDs
	casStore CAS

	opsCount int
}

func (d *OperationsProcessor) Process() error {
	if err := d.fetchCoreIndexFile(); err != nil {
		return fmt.Errorf("core index: %s - failed to fetch core index file: %w", d.CoreIndexFileURI, err)
	}

	if d.CoreIndexFile == nil {
		return fmt.Errorf("core index: %s - core index file is nil", d.CoreIndexFileURI)
	}

	if err := d.CoreIndexFile.Process(); err != nil {
		return fmt.Errorf("core index: %s failed to process core index file: %w", d.CoreIndexFileURI, err)
	}

	if d.CoreProofFileURI != "" {

		if err := d.fetchCoreProofFile(); err != nil {
			return fmt.Errorf("core index: %s - failed to fetch core proof file: %w", d.CoreIndexFileURI, err)
		}

		if d.CoreProofFile == nil {
			return fmt.Errorf("core index: %s - core proof file is nil", d.CoreIndexFileURI)
		}

		if err := d.CoreProofFile.Process(); err != nil {
			return fmt.Errorf("core index: %s - failed to process core proof file: %w", d.CoreIndexFileURI, err)
		}
	}

	if d.ProvisionalIndexFileURI != "" {

		if err := d.fetchProvisionalIndexFile(); err != nil {
			return fmt.Errorf("core index: %s - failed to fetch provisional index file: %w", d.CoreIndexFileURI, err)
		}

		if d.ProvisionalIndexFile == nil {
			return fmt.Errorf("core index: %s - provisional index file is nil", d.CoreIndexFileURI)
		}

		if err := d.ProvisionalIndexFile.Process(); err != nil {
			return fmt.Errorf("core index: %s - failed to process provisional index file: %w", d.CoreIndexFileURI, err)
		}

		if len(d.ProvisionalIndexFile.Operations.Update) > 0 {

			if err := d.fetchProvisionalProofFile(); err != nil {
				return fmt.Errorf("core index: %s - failed to fetch provisional proof file: %w", d.CoreIndexFileURI, err)
			}

			if d.ProvisionalProofFile == nil {
				return fmt.Errorf("core index: %s - provisional proof file is nil", d.CoreIndexFileURI)
			}

			if err := d.ProvisionalProofFile.Process(); err != nil {
				return fmt.Errorf("core index: %s - failed to process provisional proof file: %w", d.CoreIndexFileURI, err)
			}
		}

		if len(d.ProvisionalIndexFile.Chunks) > 0 {
			if err := d.fetchChunkFile(); err != nil {
				return fmt.Errorf("core index: %s - failed to fetch chunk file: %w", d.CoreIndexFileURI, err)
			}

			if d.ChunkFile == nil {
				return fmt.Errorf("core index: %s - chunk file is nil", d.CoreIndexFileURI)
			}

			if err := d.ChunkFile.Process(); err != nil {
				return fmt.Errorf("core index: %s - failed to process chunk file: %w", d.CoreIndexFileURI, err)
			}
		}
	}

	return nil

}

func (d *OperationsProcessor) fetchCoreIndexFile() error {

	if d.CoreIndexFileURI == "" {
		return fmt.Errorf("core index file URI is empty")
	}

	coreData, err := d.casStore.GetGZip(d.CoreIndexFileURI)
	if err != nil {
		return fmt.Errorf("failed to get core index file: %w", err)
	}

	var coreIndexFile CoreIndexFile
	if err := json.Unmarshal(coreData, &coreIndexFile); err != nil {
		return fmt.Errorf("failed to unmarshal core index file: %w", err)
	}

	coreIndexFile.processor = d
	d.CoreIndexFile = &coreIndexFile

	return nil
}

func (d *OperationsProcessor) fetchCoreProofFile() error {

	if d.CoreProofFileURI == "" {
		return fmt.Errorf("core proof file URI is empty")
	}

	coreProofData, err := d.casStore.GetGZip(d.CoreProofFileURI)
	if err != nil {
		return fmt.Errorf("failed to get core proof file: %w", err)
	}

	var coreProofFile CoreProofFile
	if err := json.Unmarshal(coreProofData, &coreProofFile); err != nil {
		return fmt.Errorf("failed to unmarshal core proof file: %w", err)
	}

	coreProofFile.processor = d
	d.CoreProofFile = &coreProofFile

	return nil
}

func (d *OperationsProcessor) fetchProvisionalIndexFile() error {

	if d.ProvisionalIndexFileURI == "" {
		return fmt.Errorf("no provisional index file URI")
	}

	provisionalData, err := d.casStore.GetGZip(d.ProvisionalIndexFileURI)
	if err != nil {
		return fmt.Errorf("failed to get provisional index file: %w", err)
	}

	var provisionalIndexFile ProvisionalIndexFile
	if err := json.Unmarshal(provisionalData, &provisionalIndexFile); err != nil {
		return fmt.Errorf("failed to unmarshal provisional index file: %w", err)
	}

	provisionalIndexFile.processor = d
	d.ProvisionalIndexFile = &provisionalIndexFile

	return nil
}

func (d *OperationsProcessor) fetchProvisionalProofFile() error {

	if d.ProvisionalProofFileURI == "" {
		return fmt.Errorf("no provisional proof file URI")
	}

	provisionalProofData, err := d.casStore.GetGZip(d.ProvisionalProofFileURI)
	if err != nil {
		return fmt.Errorf("failed to get provisional proof file: %w", err)
	}

	var provisionalProofFile ProvisionalProofFile
	if err := json.Unmarshal(provisionalProofData, &provisionalProofFile); err != nil {
		return fmt.Errorf("failed to unmarshal provisional proof file: %w", err)
	}

	provisionalProofFile.processor = d
	d.ProvisionalProofFile = &provisionalProofFile

	return nil
}

func (d *OperationsProcessor) fetchChunkFile() error {

	if d.ChunkFileURI == "" {
		return fmt.Errorf("no chunk file URI")
	}

	chunkData, err := d.casStore.GetGZip(d.ChunkFileURI)
	if err != nil {
		return fmt.Errorf("failed to get chunk file: %w", err)
	}

	var chunkFile ChunkFile
	if err := json.Unmarshal(chunkData, &chunkFile); err != nil {
		return fmt.Errorf("failed to unmarshal chunk file: %w", err)
	}

	chunkFile.processor = d
	d.ChunkFile = &chunkFile

	return nil
}

func (d *OperationsProcessor) createDID(id string, recoverCommitment string) error {

	didDoc := NewDIDDoc(id, recoverCommitment)
	if err := d.didStore.Put(didDoc); err != nil {
		return fmt.Errorf("failed to put did document: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) patchDelta(id string, patch map[string]interface{}) error {
	action, ok := patch["action"]
	if !ok {
		return fmt.Errorf("%s patch does not have action: %s", id, patch)
	}
	switch action {
	case "replace":
		if err := d.replaceDocEntries(id, patch); err != nil {
			return fmt.Errorf("failed to replace doc entries for %s: %w", id, err)
		}
		return nil
	case "add-public-keys":
		if err := d.addPublicKeys(id, patch); err != nil {
			return fmt.Errorf("failed to add public keys to %s: %w", id, err)
		}
		return nil
	case "remove-public-keys":
		if err := d.removePublicKeys(id, patch); err != nil {
			return fmt.Errorf("%s failed to remove public keys: %w", id, err)
		}
		return nil
	case "add-services":
		if err := d.addServices(id, patch); err != nil {
			return fmt.Errorf("%s failed to add services: %w", id, err)
		}
		return nil
	case "remove-services":
		if err := d.removeServices(id, patch); err != nil {
			return fmt.Errorf("%s failed to remove services: %w", id, err)
		}
		return nil
	case "ietf-json-patch":
		if err := d.ietfJSONPatch(id, patch); err != nil {
			return fmt.Errorf("failed to ietf json patch %s: %w", id, err)
		}
		return nil
	default:
		return fmt.Errorf("%s unknown patch type: %s", id, patch)
	}
}

func (d *OperationsProcessor) replaceDocEntries(id string, patch map[string]interface{}) error {

	didDoc, err := d.didStore.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get did document: %w", err)
	}

	doc, ok := patch["document"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s patch does not have document: %+v", id, patch)
	}

	didDoc.DIDDocument.ResetData()

	publicKeys, err := processKeys(id, doc)
	if err == nil {
		didDoc.DIDDocument.AddPublicKeys(publicKeys)
	}

	didServices, err := processServices(doc)
	if err == nil {
		didDoc.DIDDocument.AddServices(didServices)
	}

	if err := d.didStore.Put(didDoc); err != nil {
		return fmt.Errorf("failed to update %s: %w", id, err)
	}

	return nil
}

func (d *OperationsProcessor) addPublicKeys(id string, patch map[string]interface{}) error {
	doc, err := d.didStore.Get(id)
	if err != nil {
		return fmt.Errorf("%s failed to get DID for add-public-key: %w", id, err)
	}
	pubKeys, err := processKeys(id, patch)
	if err == nil {
		doc.DIDDocument.AddPublicKeys(pubKeys)
		if err := d.didStore.Put(doc); err != nil {
			return fmt.Errorf("%s failed to put DID: %w", id, err)
		}
	}
	return nil
}

func (d *OperationsProcessor) removePublicKeys(id string, patch map[string]interface{}) error {

	doc, err := d.didStore.Get(id)
	if err != nil {
		return fmt.Errorf("%s failed to get DID for remove-public-key: %w", id, err)
	}

	pKeyInterfaces, ok := patch["ids"].([]interface{})
	if !ok {
		return fmt.Errorf("%s patch does not have publicKey ids: %s", id, patch)
	} else {

		var pubKeys []string
		for _, pubKey := range pKeyInterfaces {
			key, ok := pubKey.(string)
			if !ok {
				return fmt.Errorf("%s patch pubKey ids do not cast to string: %s", doc.DIDDocument.ID, patch)
			}
			pubKeys = append(pubKeys, key)
		}

		if err := doc.DIDDocument.RemovePublicKeys(pubKeys); err != nil {
			return fmt.Errorf("%s failed to remove public keys: %w", id, err)
		} else {
			if err := d.didStore.Put(doc); err != nil {
				return fmt.Errorf("%s failed to put DID: %w", id, err)
			}
		}
	}
	return nil
}

func (d *OperationsProcessor) addServices(id string, patch map[string]interface{}) error {
	doc, err := d.didStore.Get(id)
	if err != nil {
		return fmt.Errorf("%s failed to get DID for add-service: %w", id, err)
	}

	services, err := processServices(patch)
	if err == nil {
		doc.DIDDocument.AddServices(services)
		if err := d.didStore.Put(doc); err != nil {
			return fmt.Errorf("%s failed to put DID: %w", id, err)
		}
	}
	return nil
}

func (d *OperationsProcessor) removeServices(id string, patch map[string]interface{}) error {
	doc, err := d.didStore.Get(id)
	if err != nil {
		return fmt.Errorf("%s failed to get DID for remove-service: %w", id, err)
	}

	sInterfaces, ok := patch["ids"].([]interface{})
	if !ok {
		return fmt.Errorf("%s patch does not have service ids: %s", doc.DIDDocument.ID, patch)
	} else {

		var services []string
		for _, s := range sInterfaces {
			service, ok := s.(string)
			if !ok {
				return fmt.Errorf("%s patch service ids do not cast to string: %s", doc.DIDDocument.ID, patch)
			}
			services = append(services, service)
		}

		if err := doc.DIDDocument.RemoveServices(services); err != nil {
			return fmt.Errorf("%s failed to remove services: %w", doc.DIDDocument.ID, err)
		} else {
			if err := d.didStore.Put(doc); err != nil {
				return fmt.Errorf("%s failed to put DID: %w", doc.DIDDocument.ID, err)
			}
		}
	}
	return nil
}

func (d *OperationsProcessor) ietfJSONPatch(id string, patch map[string]interface{}) error {
	fmt.Printf("%s ietf-json not supported: %s\n", id, patch)
	return nil
}

func (d *OperationsProcessor) setUpdateCommitment(id string, commitment string) error {
	didDoc, err := d.didStore.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get did doc for %s: %w", id, err)
	}

	didDoc.Metadata.Method.UpdateCommitment = commitment

	if err := d.didStore.Put(didDoc); err != nil {
		return fmt.Errorf("failed to put did document: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) getUpdateCommitment(id string) (string, error) {
	didDoc, err := d.didStore.Get(id)
	if err != nil {
		return "", fmt.Errorf("failed to get did doc for %s: %w", id, err)
	}

	if didDoc.Metadata.Method.UpdateCommitment == "" {
		return "", fmt.Errorf("no update commitment for %s", id)
	}

	return didDoc.Metadata.Method.UpdateCommitment, nil
}

func processKeys(id string, patch map[string]interface{}) ([]DIDKeyInfo, error) {

	keys, ok := patch["publicKeys"]
	if !ok {
		return nil, fmt.Errorf("publicKeys not found")
	}

	var publicKeys []DIDKeyInfo
	keyBytes, err := json.Marshal(keys)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal publicKeys: %w", err)
	}

	if err := json.Unmarshal(keyBytes, &publicKeys); err != nil {
		return nil, fmt.Errorf("failed to unmarshal publicKeys: %w", err)
	}

	for i, key := range publicKeys {
		if len(base64.RawURLEncoding.EncodeToString([]byte(key.ID))) > 50 {
			return nil, fmt.Errorf("public key id %s is too long", key.ID)
		}

		key.ID = fmt.Sprintf("#%s", key.ID)
		key.Controller = fmt.Sprintf("did:ion:%s", id)
		publicKeys[i] = key
	}

	return publicKeys, nil
}

func processServices(patch map[string]interface{}) ([]DIDService, error) {

	services, ok := patch["services"]
	if !ok {
		return nil, fmt.Errorf("services not found")
	}

	var didServices []DIDService
	serviceBytes, err := json.Marshal(services)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal services: %w", err)
	}
	if err := json.Unmarshal(serviceBytes, &didServices); err != nil {
		return nil, fmt.Errorf("failed to unmarshal services: %w", err)
	}

	for i, service := range didServices {
		if len(base64.URLEncoding.EncodeToString([]byte(service.ID))) > 50 {
			return nil, fmt.Errorf("service id %s is too long", service.ID)
		}

		service.ID = fmt.Sprintf("#%s", service.ID)
		didServices[i] = service
	}

	return didServices, nil
}
