package sidetree

import (
	"fmt"
)

type DIDDoc struct {
	Context     string      `json:"@context"`
	DIDDocument *DIDDocData `json:"didDocument"`
	Metadata    DIDMetadata `json:"didDocumentMetadata"`
}

func NewDIDDoc(id string, recoveryCommitment string) *DIDDoc {

	var didContext []interface{}
	didContext = append(didContext, "https://www.w3.org/ns/did/v1")

	contextBase := make(map[string]interface{})
	contextBase["@base"] = fmt.Sprintf("did:ion:%s", id)
	didContext = append(didContext, contextBase)

	return &DIDDoc{
		Context: "https://w3id.org/did-resolution/v1",
		DIDDocument: &DIDDocData{
			ID:      id,
			DocID:   fmt.Sprintf("did:ion:%s", id),
			Context: didContext,
		},
		Metadata: DIDMetadata{
			CanonicalId: fmt.Sprintf("did:ion:%s", id),
			Method: DIDMetadataMethod{
				Published:          true,
				RecoveryCommitment: recoveryCommitment,
			},
		},
	}
}

type DIDDocData struct {
	ID                   string        `json:"-"`
	DocID                string        `json:"id"`
	Context              []interface{} `json:"@context"`
	Services             []DIDService  `json:"service,omitempty"`
	Verification         []DIDKeyInfo  `json:"verificationMethod,omitempty"`
	Authentication       []string      `json:"authentication,omitempty"`
	Assertion            []string      `json:"assertionMethod,omitempty"`
	CapabilityDelegation []string      `json:"capabilityDelegation,omitempty"`
	CapabilityInvocation []string      `json:"capabilityInvocation,omitempty"`
	KeyAgreement         []string      `json:"keyAgreement,omitempty"`
}

func (d *DIDDocData) ResetData() {
	d.Services = []DIDService{}
	d.Verification = []DIDKeyInfo{}
	d.Authentication = []string{}
	d.Assertion = []string{}
	d.CapabilityDelegation = []string{}
	d.CapabilityInvocation = []string{}
	d.KeyAgreement = []string{}
}

func (d *DIDDocData) AddPublicKeys(publicKeys []DIDKeyInfo) {
	for _, pubKey := range publicKeys {
		// Check for existing key by ID Error for entire statechange if key already exists
		// TODO: Check spec if this is legit
		for i, key := range d.Verification {
			if key.ID == pubKey.ID {
				d.Verification = append(d.Verification[:i], d.Verification[i+1:]...)
				d.removePurpose(key.ID)
				break
			}
		}

		for _, purpose := range pubKey.Purposes {
			switch purpose {
			case "authentication":
				d.Authentication = append(d.Authentication, pubKey.ID)
			case "keyAgreement":
				d.KeyAgreement = append(d.KeyAgreement, pubKey.ID)
			case "assertionMethod":
				d.Assertion = append(d.Assertion, pubKey.ID)
			case "capabilityDelegation":
				d.CapabilityDelegation = append(d.CapabilityDelegation, pubKey.ID)
			case "capabilityInvocation":
				d.CapabilityInvocation = append(d.CapabilityInvocation, pubKey.ID)
			}
		}
		pubKey.Purposes = nil
		d.Verification = append(d.Verification, pubKey)
	}
}

func (d *DIDDocData) removePurpose(keyId string) {
	for i, authKey := range d.Authentication {
		if authKey == keyId {
			d.Authentication = append(d.Authentication[:i], d.Authentication[i+1:]...)
			break
		}
	}
	for i, keyAgreeKey := range d.KeyAgreement {
		if keyAgreeKey == keyId {
			d.KeyAgreement = append(d.KeyAgreement[:i], d.KeyAgreement[i+1:]...)
			break
		}
	}
	for i, assertionKey := range d.Assertion {
		if assertionKey == keyId {
			d.Assertion = append(d.Assertion[:i], d.Assertion[i+1:]...)
			break
		}
	}
	for i, capDelegationKey := range d.CapabilityDelegation {
		if capDelegationKey == keyId {
			d.CapabilityDelegation = append(d.CapabilityDelegation[:i], d.CapabilityDelegation[i+1:]...)
			break
		}
	}
	for i, capInvocationKey := range d.CapabilityInvocation {
		if capInvocationKey == keyId {
			d.CapabilityInvocation = append(d.CapabilityInvocation[:i], d.CapabilityInvocation[i+1:]...)
			break
		}
	}
}

func (d *DIDDocData) RemovePublicKeys(publicKeys []string) error {
	modified := false
	for _, key := range publicKeys {

		if key[0] != '#' {
			key = "#" + key
		}

		for i, pubKey := range d.Verification {
			if pubKey.ID == key {
				modified = true
				d.Verification = append(d.Verification[:i], d.Verification[i+1:]...)
				d.removePurpose(key)
				break
			}
		}
	}
	if !modified {
		return fmt.Errorf("no keys to remove")
	}
	return nil
}

func (d *DIDDocData) AddServices(services []DIDService) {
	for _, service := range services {
		// Check for existing service by ID Error for entire statechange if service already exists
		for i, svc := range d.Services {
			if svc.ID == service.ID {
				d.Services = append(d.Services[:i], d.Services[i+1:]...)
				break
			}
		}
	}
	d.Services = append(d.Services, services...)
}

func (d *DIDDocData) RemoveServices(services []string) error {
	modified := false
	for _, service := range services {
		if service[0] != '#' {
			service = "#" + service
		}

		for i, svc := range d.Services {
			if svc.ID == service {
				modified = true
				d.Services = append(d.Services[:i], d.Services[i+1:]...)
				break
			}
		}
	}
	if !modified {
		return fmt.Errorf("no services to remove")
	}
	return nil
}

type DIDKeyInfo struct {
	ID         string                 `json:"id"`
	Controller string                 `json:"controller"`
	Type       string                 `json:"type"`
	PubKey     map[string]interface{} `json:"publicKeyJwk,omitempty"`
	Multibase  string                 `json:"publicKeyMultibase,omitempty"`
	Purposes   []string               `json:"purposes,omitempty"`
}

type DIDService struct {
	ID              string             `json:"id"`
	Type            string             `json:"type"`
	ServiceEndpoint DIDServiceEndpoint `json:"serviceEndpoint"`
}

type DIDServiceEndpoint interface{}

type DIDMetadata struct {
	Method      DIDMetadataMethod `json:"method"`
	CanonicalId string            `json:"canonicalId"`
}

type DIDMetadataMethod struct {
	Published          bool   `json:"published"`
	RecoveryCommitment string `json:"recoveryCommitment"`
	UpdateCommitment   string `json:"updateCommitment"`
}
