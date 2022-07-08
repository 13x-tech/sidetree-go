package sidetree

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
)

func PatchData(log Logger, prefix string, delta did.Delta, doc *did.Document) error {

	for _, p := range delta.Patches {

		patch := new(Patch)

		patch.prefix = prefix
		patch.log = log
		patch.doc = doc
		patch.patch = p

		if err := patch.patchDelta(); err != nil {
			return err
		}
	}

	return nil
}

type Patch struct {
	patch  map[string]interface{}
	doc    *did.Document
	prefix string
	log    Logger
}

func (p *Patch) patchDelta() error {

	id := p.doc.Document.ID

	action, ok := p.patch["action"]
	if !ok {
		return p.log.Errorf("%s patch does not have action: %s", id, p.patch)
	}
	switch action {
	case "replace":
		if err := p.replaceDocEntries(id, p.patch); err != nil {
			return p.log.Errorf("failed to replace doc entries for %s: %w", id, err)
		}
		return nil
	case "add-public-keys":
		if err := p.addPublicKeys(id, p.patch); err != nil {
			return p.log.Errorf("failed to add public keys to %s: %w", id, err)
		}
		return nil
	case "remove-public-keys":
		if err := p.removePublicKeys(id, p.patch); err != nil {
			return p.log.Errorf("%s failed to remove public keys: %w", id, err)
		}
		return nil
	case "add-services":
		if err := p.addServices(id, p.patch); err != nil {
			return p.log.Errorf("%s failed to add services: %w", id, err)
		}
		return nil
	case "remove-services":
		if err := p.removeServices(id, p.patch); err != nil {
			return p.log.Errorf("%s failed to remove services: %w", id, err)
		}
		return nil
	case "ietf-json-patch":
		if err := p.ietfJSONPatch(id, p.patch); err != nil {
			return p.log.Errorf("failed to ietf json patch %s: %w", id, err)
		}
		return nil
	default:
		return p.log.Errorf("%s unknown patch type: %s", id, p.patch)
	}
}

func (p *Patch) replaceDocEntries(id string, patch map[string]interface{}) error {

	doc, ok := patch["document"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s patch does not have document: %+v", id, patch)
	}

	p.doc.Document.ResetData()

	publicKeys, err := p.processKeys(id, doc)
	if err == nil {
		p.doc.Document.AddPublicKeys(publicKeys)
	}

	didServices, err := p.processServices(doc)
	if err == nil {
		p.doc.Document.AddServices(didServices)
	}

	return nil
}

func (p *Patch) addPublicKeys(id string, patch map[string]interface{}) error {

	pubKeys, err := p.processKeys(id, patch)
	if err == nil {
		p.doc.Document.AddPublicKeys(pubKeys)
	}

	return nil
}

func (p *Patch) removePublicKeys(id string, patch map[string]interface{}) error {

	pKeyInterfaces, ok := patch["ids"].([]interface{})
	if !ok {
		return fmt.Errorf("%s patch does not have publicKey ids: %s", id, patch)
	} else {

		var pubKeys []string
		for _, pubKey := range pKeyInterfaces {
			key, ok := pubKey.(string)
			if !ok {
				return fmt.Errorf("%s patch pubKey ids do not cast to string: %s", id, patch)
			}
			pubKeys = append(pubKeys, key)
		}

		if err := p.doc.Document.RemovePublicKeys(pubKeys); err != nil {
			return fmt.Errorf("%s failed to remove public keys: %w", id, err)
		}

	}
	return nil
}

func (p *Patch) addServices(id string, patch map[string]interface{}) error {

	services, err := p.processServices(patch)
	if err == nil {
		p.doc.Document.AddServices(services)
	}
	return nil
}

func (p *Patch) removeServices(id string, patch map[string]interface{}) error {
	sInterfaces, ok := patch["ids"].([]interface{})
	if !ok {
		return fmt.Errorf("%s patch does not have service ids: %s", id, patch)
	} else {
		var services []string
		for _, s := range sInterfaces {
			service, ok := s.(string)
			if !ok {
				return fmt.Errorf("%s patch service ids do not cast to string: %s", id, patch)
			}
			services = append(services, service)
		}

		if err := p.doc.Document.RemoveServices(services); err != nil {
			return fmt.Errorf("%s failed to remove services: %w", id, err)
		}
	}
	return nil
}

func (p *Patch) ietfJSONPatch(id string, patch map[string]interface{}) error {
	fmt.Printf("%s ietf-json not supported: %s\n", id, patch)
	return nil
}

func (p *Patch) processKeys(id string, patch map[string]interface{}) ([]did.KeyInfo, error) {

	keys, ok := patch["publicKeys"]
	if !ok {
		return nil, fmt.Errorf("publicKeys not found")
	}

	var publicKeys []did.KeyInfo
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
		if key.Controller == "" {
			key.Controller = fmt.Sprintf("did:%s:%s", p.prefix, id)
		}
		publicKeys[i] = key
	}

	return publicKeys, nil
}

func (p *Patch) processServices(patch map[string]interface{}) ([]did.Service, error) {

	services, ok := patch["services"]
	if !ok {
		return nil, fmt.Errorf("services not found")
	}

	var didServices []did.Service
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
