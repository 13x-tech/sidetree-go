package did

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/13x-tech/sidetree-go/internal/keys"

	"github.com/gowebpki/jcs"
	mh "github.com/multiformats/go-multihash"
)

func New(id, recoveryCommitment, prefix string, published bool) *Document {
	var didContext []interface{}
	didContext = append(didContext, "https://www.w3.org/ns/did/v1")

	contextBase := map[string]interface{}{}
	contextBase["@base"] = fmt.Sprintf("did:%s:%s", prefix, id)
	didContext = append(didContext, contextBase)

	return &Document{
		Context: "https://w3id.org/did-resolution/v1",
		Document: &DocumentData{
			ID:      id,
			DocID:   fmt.Sprintf("did:%s:%s", prefix, id),
			Context: didContext,
		},
		Metadata: Metadata{
			CanonicalId: fmt.Sprintf("did:%s:%s", prefix, id),
			Method: MetadataMethod{
				Published:          published,
				RecoveryCommitment: recoveryCommitment,
			},
		},
	}
}

type Document struct {
	Context  string        `json:"@context"`
	Document *DocumentData `json:"didDocument"`
	Metadata Metadata      `json:"didDocumentMetadata"`
}

type DocumentData struct {
	ID                   string        `json:"-"`
	DocID                string        `json:"id"`
	Context              []interface{} `json:"@context"`
	Services             []Service     `json:"service,omitempty"`
	Verification         []KeyInfo     `json:"verificationMethod,omitempty"`
	Authentication       []string      `json:"authentication,omitempty"`
	Assertion            []string      `json:"assertionMethod,omitempty"`
	CapabilityDelegation []string      `json:"capabilityDelegation,omitempty"`
	CapabilityInvocation []string      `json:"capabilityInvocation,omitempty"`
	KeyAgreement         []string      `json:"keyAgreement,omitempty"`
}

func (d *DocumentData) ResetData() {
	d.Services = []Service{}
	d.Verification = []KeyInfo{}
	d.Authentication = []string{}
	d.Assertion = []string{}
	d.CapabilityDelegation = []string{}
	d.CapabilityInvocation = []string{}
	d.KeyAgreement = []string{}
}

func (d *DocumentData) AddPublicKeys(publicKeys []KeyInfo) {
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

func (d *DocumentData) removePurpose(keyId string) {
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

func (d *DocumentData) RemovePublicKeys(publicKeys []string) error {
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

func (d *DocumentData) AddServices(services []Service) {
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

func (d *DocumentData) RemoveServices(services []string) error {
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

type KeyInfo struct {
	ID         string                 `json:"id"`
	Controller string                 `json:"controller,omitempty"`
	Type       string                 `json:"type"`
	PubKey     map[string]interface{} `json:"publicKeyJwk,omitempty"`
	Multibase  string                 `json:"publicKeyMultibase,omitempty"`
	Purposes   []string               `json:"purposes,omitempty"`
}

type Service struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"`
	ServiceEndpoint ServiceEndpoint `json:"serviceEndpoint"`
}

type ServiceEndpoint interface{}

type Metadata struct {
	Method      MetadataMethod `json:"method"`
	CanonicalId string         `json:"canonicalId"`
}

type MetadataMethod struct {
	Published          bool   `json:"published"`
	RecoveryCommitment string `json:"recoveryCommitment"`
	UpdateCommitment   string `json:"updateCommitment"`
}

func GenerateKeys() (updateKey, recoveryKey *keys.JSONWebKey, err error) {
	updateKey, err = keys.GenerateES256K(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate update key: %v", err)
	}

	recoveryKey, err = keys.GenerateES256K(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate recovery key: %v", err)
	}

	return
}

type Option func(*craete) error

func WithPrefix(prefix string) Option {
	return func(d *craete) error {
		d.prefix = prefix
		return nil
	}
}

func WithUpdateKey(key *keys.JSONWebKey) Option {
	return func(d *craete) error {
		d.updateKey = key
		return nil
	}
}

func WithRecoverKey(key *keys.JSONWebKey) Option {
	return func(d *craete) error {
		d.recoveryKey = key
		return nil
	}
}

func WithGenerateKeys() Option {
	return func(d *craete) error {
		updateKey, recoveryKey, err := GenerateKeys()
		if err != nil {
			return err
		}
		d.updateKey = updateKey
		d.recoveryKey = recoveryKey
		return nil
	}
}

func WithServices(services ...Service) Option {
	return func(d *craete) error {
		d.addServices(services...)
		return nil
	}
}

func WithPubKeys(keys ...KeyInfo) Option {
	return func(d *craete) error {
		d.addPublicKeys(keys...)
		return nil
	}
}

// Create a DID Identity with the given DID and options
func Create(options ...Option) (*craete, error) {
	d := &craete{}
	for _, option := range options {
		if err := option(d); err != nil {
			return nil, err
		}
	}

	if d.prefix == "" {
		d.prefix = "ion"
	}

	if d.updateKey == nil || d.recoveryKey == nil {
		return nil, fmt.Errorf("update and recovery keys must be provided")
	}

	if err := d.generate(); err != nil {
		return nil, err
	}

	return d, nil
}

func ParseDID(did string) (*Document, error) {
	parts := strings.Split(did, ":")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid did: %s", did)
	}

	if parts[0] != "did" {
		return nil, fmt.Errorf("invalid did: %s", did)
	}

	method := parts[1]
	if method != "ion" {
		return nil, fmt.Errorf("invalid did - only ion method currently supported: %s", did)
	}

	id := parts[2]
	longForm := parts[3]

	longFormData, err := base64.RawURLEncoding.DecodeString(longForm)
	if err != nil {
		return nil, fmt.Errorf("invalid did - invalid long form: %s", did)
	}

	var docInfo struct {
		Delta      Delta      `json:"delta"`
		SuffixData SuffixData `json:"suffixData"`
	}

	if err := json.Unmarshal(longFormData, &docInfo); err != nil {
		return nil, fmt.Errorf("invalid did - invalid long form: %s", did)
	}

	recoverCommitment := docInfo.SuffixData.RecoveryCommitment

	doc := New(id, recoverCommitment, method, false)

	return doc, nil
}

type craete struct {
	delta      *Delta
	suffixData *SuffixData

	updateReveal       *string
	updateCommitment   *string
	recoveryReveal     *string
	recoveryCommitment *string

	prefix      string
	recoveryKey *keys.JSONWebKey
	updateKey   *keys.JSONWebKey
	pubKeys     []KeyInfo
	services    []Service
}

func (d *craete) addPublicKeys(keys ...KeyInfo) {
	d.pubKeys = append(d.pubKeys, keys...)
}

func (d *craete) addServices(services ...Service) {
	d.services = append(d.services, services...)
}

func (d *craete) generate() error {
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

	delta, err := createReplaceDelta(*d.updateCommitment, d.pubKeys, d.services)
	if err != nil {
		return fmt.Errorf("failed to create delta: %w", err)
	}

	d.delta = &delta

	deltaHash, err := delta.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash delta: %w", err)
	}

	suffixData := SuffixData{
		DeltaHash:          deltaHash,
		RecoveryCommitment: *d.recoveryCommitment,
	}

	d.suffixData = &suffixData

	return nil
}

func generateReveal(key *keys.JSONWebKey) (reveal, commitment string, err error) {

	updateKeyData, err := key.MarshalJSON()
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal update key: %w", err)
	}

	updateKeyJSON, err := jcs.Transform(updateKeyData)
	if err != nil {
		return "", "", fmt.Errorf("failed to transform update key: %w", err)
	}

	reveal, err = HashReveal(updateKeyJSON)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash reveal: %w", err)
	}

	commitment, err = hashCommitment(updateKeyJSON)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash commitment: %w", err)
	}

	return
}

func (d *craete) LongFormURI() (string, error) {

	didSuffix, err := d.suffixData.URI()
	if err != nil {
		return "", fmt.Errorf("failed to create suffix: %w", err)
	}

	marshalStruct := struct {
		Delta      Delta      `json:"delta"`
		SuffixData SuffixData `json:"suffixData"`
	}{
		Delta:      *d.delta,
		SuffixData: *d.suffixData,
	}

	didData, err := json.Marshal(marshalStruct)
	if err != nil {
		return "", fmt.Errorf("failed to marshal DID: %w", err)
	}

	jsonData, err := jcs.Transform(didData)
	if err != nil {
		return "", fmt.Errorf("failed to transform DID: %w", err)
	}

	encodedSuffixData := base64.RawURLEncoding.EncodeToString(jsonData)

	return fmt.Sprintf("did:%s:%s:%s", d.prefix, didSuffix, encodedSuffixData), nil
}

func (d *craete) URI() (string, error) {
	didSuffix, err := d.suffixData.URI()
	if err != nil {
		return "", fmt.Errorf("failed to create suffix: %w", err)
	}

	return fmt.Sprintf("did:%s:%s", d.prefix, didSuffix), nil

}

func createReplaceDelta(updateCommitment string, publicKeys []KeyInfo, services []Service) (Delta, error) {

	if len(publicKeys) == 0 && len(services) == 0 {
		return Delta{}, fmt.Errorf("public keys or services must not be empty")
	}

	document := map[string]interface{}{}
	if len(publicKeys) > 0 {
		pubKeyMap, err := createPublicKeyMap(publicKeys)
		if err != nil {
			return Delta{}, fmt.Errorf("failed to create public key map: %w", err)
		}
		document["publicKeys"] = pubKeyMap
	}

	if len(services) > 0 {
		serviceMap, err := createServicesMap(services)
		if err != nil {
			return Delta{}, fmt.Errorf("failed to create services map: %w", err)
		}
		document["services"] = serviceMap
	}

	patch := map[string]interface{}{
		"action":   "replace",
		"document": document,
	}

	d := Delta{
		UpdateCommitment: updateCommitment,
		Patches:          []map[string]interface{}{patch},
	}

	if b, err := underMaxSize(d, 1000); !b {
		return Delta{}, fmt.Errorf("delta size exceeded: %w", err)
	}

	return d, nil
}

//TODO Implement size checks where needed
// Delta, etc.
func underMaxSize(i interface{}, max int) (bool, error) {
	dataJSON, err := json.Marshal(i)
	if err != nil {
		return false, fmt.Errorf("failed to marshal data: %w", err)
	}

	jcsData, err := jcs.Transform(dataJSON)
	if err != nil {
		return false, fmt.Errorf("failed to transform data: %w", err)
	}

	if len(jcsData) < max {
		return true, nil
	} else {
		return false, fmt.Errorf("data is too large (max: %d) got %d", max, len(jcsData))
	}
}

func createPublicKeyMap(pubKeys []KeyInfo) (interface{}, error) {
	keyData, err := json.Marshal(pubKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	keyJSON, err := jcs.Transform(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to transform public key: %w", err)
	}

	var keyMap interface{}

	if err := json.Unmarshal(keyJSON, &keyMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	return keyMap, nil
}

func createServicesMap(services []Service) (interface{}, error) {
	serviceData, err := json.Marshal(services)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal services: %w", err)
	}

	serviceJSON, err := jcs.Transform(serviceData)
	if err != nil {
		return nil, fmt.Errorf("failed to transform services: %w", err)
	}

	var serviceMap interface{}

	if err := json.Unmarshal(serviceJSON, &serviceMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal services: %w", err)
	}

	return serviceMap, nil
}

func CheckReveal(reveal string, commitment string) bool {
	rawReveal, err := base64.RawURLEncoding.DecodeString(reveal)
	if err != nil {
		return false
	}

	decoded, err := mh.Decode(rawReveal)
	if err != nil {
		return false
	}

	h256 := sha256.Sum256(decoded.Digest)
	revealHashed, err := mh.Encode(h256[:], mh.SHA2_256)
	if err != nil {
		return false
	}

	b64 := base64.RawURLEncoding.EncodeToString(revealHashed)

	return commitment == string(b64)
}

func HashReveal(data []byte) (string, error) {
	hashedReveal := sha256.Sum256(data)
	revealMH, err := mh.Encode(hashedReveal[:], mh.SHA2_256)
	if err != nil {
		return "", fmt.Errorf("failed to hash revieal: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(revealMH), nil
}

func hashCommitment(data []byte) (string, error) {
	hashedReveal := sha256.Sum256(data)
	hashedCommitment := sha256.Sum256(hashedReveal[:])

	commitmentMH, err := mh.Encode(hashedCommitment[:], mh.SHA2_256)
	if err != nil {
		return "", fmt.Errorf("failed to hash commitment: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(commitmentMH), nil
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

type Delta struct {
	Patches          []map[string]interface{} `json:"patches"`
	UpdateCommitment string                   `json:"updateCommitment"`
}

func (d *Delta) Hash() (string, error) {
	deltaBytes, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("failed to marshal delta for hashing: %w", err)
	}

	deltaJSON, err := jcs.Transform(deltaBytes)
	if err != nil {
		return "", fmt.Errorf("failed to transform delta for hashing: %w", err)
	}

	shaDelta := sha256.Sum256(deltaJSON)
	hashed, err := mh.Encode(shaDelta[:], mh.SHA2_256)

	if err != nil {
		return "", fmt.Errorf("failed to multihash encode sha256: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(hashed), nil
}
