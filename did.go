package sidetree

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/gowebpki/jcs"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type DIDDoc struct {
	Context     string      `json:"@context"`
	DIDDocument *DIDDocData `json:"didDocument"`
	Metadata    DIDMetadata `json:"didDocumentMetadata"`
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
	Controller string                 `json:"controller,omitempty"`
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

func GenerateKeys() (updateKey, recoveryKey jwk.Key, err error) {
	updateECDSA, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		err = fmt.Errorf("failed to generate update key: %w", err)
		return
	}

	updateKey, err = jwk.FromRaw(updateECDSA)
	if err != nil {
		err = fmt.Errorf("failed to transform update key: %w", err)
		return
	}

	recoveryECDSA, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		err = fmt.Errorf("failed to generate recovery key: %w", err)
		return
	}

	recoveryKey, err = jwk.FromRaw(recoveryECDSA)
	if err != nil {
		err = fmt.Errorf("failed to transform recovery key: %w", err)
		return
	}

	return
}

type DIDOption func(*DID) error

func WithUpdateKey(key jwk.Key) DIDOption {
	return func(d *DID) error {
		d.updateKey = key
		return nil
	}
}

func WithRecoverKey(key jwk.Key) DIDOption {
	return func(d *DID) error {
		d.recoveryKey = key
		return nil
	}
}

func WithGenerateKeys() DIDOption {
	return func(d *DID) error {
		updateKey, recoveryKey, err := GenerateKeys()
		if err != nil {
			return err
		}
		d.updateKey = updateKey
		d.recoveryKey = recoveryKey
		return nil
	}
}

func WithServices(services ...DIDService) DIDOption {
	return func(d *DID) error {
		return d.AddServices(services...)
	}
}

func WithPubKeys(keys ...jwk.Key) DIDOption {
	return func(d *DID) error {
		return d.AddPublicKeys(keys...)
	}
}

func NewDID(options ...DIDOption) (*DID, error) {
	d := &DID{}
	for _, option := range options {
		if err := option(d); err != nil {
			return nil, err
		}
	}

	if d.updateKey == nil || d.recoveryKey == nil {
		return nil, fmt.Errorf("update and recovery keys must be provided")
	}

	if err := d.generateReveals(); err != nil {
		return nil, err
	}
	return d, nil
}

type DID struct {
	Delta      *Delta      `json:"delta"`
	SuffixData *SuffixData `json:"suffixData"`

	updateReveal       *string
	updateCommitment   *string
	recoveryReveal     *string
	recoveryCommitment *string

	recoveryKey jwk.Key
	updateKey   jwk.Key
	pubKeys     []DIDKeyInfo
	services    []DIDService
}

func (d *DID) AddPublicKeys(keys ...jwk.Key) error {
	for _, key := range keys {
		didKey, err := joseKeyToDIDKeyInfo(key)
		if err != nil {
			return fmt.Errorf("failed to convert key to DIDKeyInfo: %w", err)
		}
		d.pubKeys = append(d.pubKeys, didKey)
	}
	return nil
}

func (d *DID) AddServices(services ...DIDService) error {
	d.services = append(d.services, services...)
	return nil
}

func (d *DID) generateReveals() error {
	updateReveal, updateCommitment, err := generateReveal(d.updateKey)
	if err != nil {
		return fmt.Errorf("failed to generate update reveal: %w", err)
	}
	d.updateReveal = &updateReveal
	d.updateCommitment = &updateCommitment

	recoveryReveal, recoveryCommitment, err := generateReveal(d.recoveryKey)
	if err != nil {
		return fmt.Errorf("failed to generate recovery reveal: %w", err)
	}
	d.recoveryReveal = &recoveryReveal
	d.recoveryCommitment = &recoveryCommitment

	return nil
}

func generateReveal(key jwk.Key) (reveal, commitment string, err error) {

	updateKeyData, err := json.Marshal(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal update key: %w", err)
	}

	updateKeyJSON, err := jcs.Transform(updateKeyData)
	if err != nil {
		return "", "", fmt.Errorf("failed to transform update key: %w", err)
	}

	reveal, err = hashReveal(updateKeyJSON)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash reveal: %w", err)
	}

	commitment, err = hashCommitment(updateKeyJSON)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash commitment: %w", err)
	}

	return
}

func (d *DID) LongFormURI() (string, error) {

	delta, err := createReplaceDelta(*d.updateCommitment, d.pubKeys, d.services)
	if err != nil {
		return "", fmt.Errorf("failed to create delta: %w", err)
	}

	d.Delta = &delta

	deltaHash, err := delta.Hash()
	if err != nil {
		return "", fmt.Errorf("failed to hash delta: %w", err)
	}

	suffixData := SuffixData{
		DeltaHash:          deltaHash,
		RecoveryCommitment: *d.recoveryCommitment,
	}

	d.SuffixData = &suffixData

	didSuffix, err := d.SuffixData.URI()
	if err != nil {
		return "", fmt.Errorf("failed to create suffix: %w", err)
	}

	didData, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("failed to marshal DID: %w", err)
	}

	jsonData, err := jcs.Transform(didData)
	if err != nil {
		return "", fmt.Errorf("failed to transform DID: %w", err)
	}

	b64Data := base64.RawURLEncoding.EncodeToString(jsonData)

	return fmt.Sprintf("did:ion:%s:%s", didSuffix, b64Data), nil
}
