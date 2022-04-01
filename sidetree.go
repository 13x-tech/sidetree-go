package sidetree

import (
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	mh "github.com/multiformats/go-multihash"
)

type SidetreeOption func(interface{})

func WithPrefix(prefix string) SidetreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			t.prefix = prefix
		case *OperationsProcessor:
			t.prefix = prefix
		}
	}
}

func WithStorage(storage Storage) SidetreeOption {
	return func(d interface{}) {

		switch t := d.(type) {
		case *SideTree:
			t.store = storage
		case *OperationsProcessor:
			didStore, err := storage.DIDs()
			if err != nil {
				return
			}

			casStore, err := storage.CAS()
			if err != nil {
				return
			}

			indexStore, err := storage.Indexer()
			if err != nil {
				return
			}
			t.didStore = didStore
			t.casStore = casStore
			t.indexStore = indexStore
		}

	}
}

func WithLogger(log Logger) SidetreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			t.log = log
		case *OperationsProcessor:
			t.log = log
		}
	}
}

func New(options ...SidetreeOption) *SideTree {
	return &SideTree{}
}

type SideTree struct {
	prefix string
	store  Storage
	log    Logger
}

func (s *SideTree) ProcessOperations(ops []SideTreeOp) error {

	for _, op := range ops {
		processor, err := Processor(
			op,
			WithPrefix(s.prefix),
			WithStorage(s.store),
			WithLogger(s.log),
		)
		if err != nil {
			return fmt.Errorf("failed to create operations processor: %w", err)
		}

		if err := processor.Process(); err != nil {
			return fmt.Errorf("failed to process operations: %w", err)
		}
	}

	return nil
}

func Create(updateKey, recoveryKey jwk.Key, publicKeys []jwk.Key, services []DIDService) (Delta, CreateOperation, error) {

	var delta Delta
	var create CreateOperation

	updateKeyData, err := json.Marshal(updateKey)
	if err != nil {
		return delta, create, fmt.Errorf("failed to marshal update key: %w", err)
	}

	updateKeyJSON, err := jcs.Transform(updateKeyData)
	if err != nil {
		return delta, create, fmt.Errorf("failed to transform update key: %w", err)
	}

	commitment, err := hashCommitment(updateKeyJSON)
	if err != nil {
		return delta, create, fmt.Errorf("failed to hash commitment: %w", err)
	}

	var pubKeys []DIDKeyInfo
	for _, key := range publicKeys {
		didKey, err := joseKeyToDIDKeyInfo(key)
		if err != nil {
			return delta, create, fmt.Errorf("failed to convert key to DIDKeyInfo: %w", err)
		}
		pubKeys = append(pubKeys, didKey)
	}

	delta, err = createReplaceDelta(commitment, pubKeys, services)
	if err != nil {
		return delta, create, fmt.Errorf("failed to create delta: %w", err)
	}

	deltaHash, err := delta.Hash()
	if err != nil {
		return delta, create, fmt.Errorf("failed to hash delta: %w", err)
	}

	recoverKeyData, err := json.Marshal(recoveryKey)
	if err != nil {
		return delta, create, fmt.Errorf("failed to marshal recovery key: %w", err)
	}

	recoverKeyJSON, err := jcs.Transform(recoverKeyData)
	if err != nil {
		return delta, create, fmt.Errorf("failed to transform recovery key: %w", err)
	}

	recoveryCommitment, err := hashCommitment(recoverKeyJSON)
	if err != nil {
		return delta, create, fmt.Errorf("failed to hash recovery commitment: %w", err)
	}

	create = CreateOperation{
		SuffixData: SuffixData{
			DeltaHash:          deltaHash,
			RecoveryCommitment: recoveryCommitment,
		},
	}

	return delta, create, nil
}

func joseKeyToDIDKeyInfo(key jwk.Key) (DIDKeyInfo, error) {

	didKey := DIDKeyInfo{}

	fingerPrint, err := key.Thumbprint(crypto.SHA256)
	if err != nil {
		return didKey, fmt.Errorf("failed to get key id: %w", err)
	}

	didKey.ID = fmt.Sprintf("sig_%x", fingerPrint[len(fingerPrint)-4:])
	didKey.Type, err = keyType(key)
	if err != nil {
		return DIDKeyInfo{}, fmt.Errorf("failed to get key type: %w", err)
	}

	keyData, err := json.Marshal(key)
	if err != nil {
		return DIDKeyInfo{}, fmt.Errorf("failed to marshal key: %w", err)
	}

	keyJSON, err := jcs.Transform(keyData)
	if err != nil {
		return DIDKeyInfo{}, fmt.Errorf("failed to transform key: %w", err)
	}

	var keyMap map[string]interface{}
	if err := json.Unmarshal(keyJSON, &keyMap); err != nil {
		return DIDKeyInfo{}, fmt.Errorf("failed to unmarshal key: %w", err)
	}

	didKey.PubKey = keyMap
	return didKey, nil

}

//TODO: this is wrong
func keyType(key jwk.Key) (string, error) {
	switch key.Algorithm() {
	case jwa.ES256K:
		return "EcdsaSecp256k1VerificationKey2019", nil
	case jwa.Ed25519:
		return "Ed25519VerificationKey2018", nil
	case jwa.RSA, jwa.RSA1_5, jwa.RSA_OAEP, jwa.RSA_OAEP_256:
		return "RsaVerificationKey2018", nil
	case jwa.X25519:
		return "X25519KeyAgreementKey2019", nil
	default:
		return "JsonWebKey2020", nil
		// return "Bls12381G2Key2020", nil
		// return "Bls12381G1Key2020", nil
		// return "SchnorrSecp256k1VerificationKey2019", nil
		// return "PgpVerificationKey2021", nil
	}
}

func createReplaceDelta(updateCommitment string, publicKeys []DIDKeyInfo, services []DIDService) (Delta, error) {

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

	return Delta{
		UpdateCommitment: updateCommitment,
		Patches:          []map[string]interface{}{patch},
	}, nil
}

func createPublicKeyMap(pubKeys []DIDKeyInfo) (interface{}, error) {
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

func createServicesMap(services []DIDService) (interface{}, error) {
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

func checkReveal(reveal string, commitment string) bool {
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

func hashReveal(data []byte) (string, error) {
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
