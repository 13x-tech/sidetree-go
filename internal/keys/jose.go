package keys

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/go-jose/go-jose/v3"
)

type Algoirthm string
type KeyType string

func (k KeyType) String() string {
	return string(k)
}

//Only secp256k1 is supported currently
var (
	ES256K Algoirthm = "ES256K"
)

func ParseKey(data []byte) (*JSONWebKey, error) {
	key := &jose.JSONWebKey{}

	if err := key.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	return &JSONWebKey{
		key: key,
	}, nil
}

type JSONWebKey struct {
	keyType KeyType
	key     *jose.JSONWebKey
}

func (k *JSONWebKey) KeyType() KeyType {
	return k.keyType
}

func (s *JSONWebKey) KeyID() string {
	return s.key.KeyID
}

func (s *JSONWebKey) MarshalJSON() ([]byte, error) {
	return s.key.MarshalJSON()
}

type JSONWebSignature struct {
	signature *jose.JSONWebSignature
}

func ParseSigned(data string) (*JSONWebSignature, error) {
	signature, err := jose.ParseSigned(data)
	if err != nil {
		return nil, err
	}

	return &JSONWebSignature{
		signature: signature,
	}, nil
}

func (s *JSONWebSignature) Payload() []byte {
	return s.signature.UnsafePayloadWithoutVerification()
}

func (s *JSONWebSignature) Verify(key *JSONWebKey) ([]byte, error) {
	return s.signature.Verify(key.key)
}

// GenerateKey generates a new key, the entropy is optional
// if entropy is nil, a new random key is generated
// Key ID will be last four bytes of the sha256 of the public key in hex
func GenerateES256K(entropy []byte) (*JSONWebKey, error) {
	var r io.Reader

	crv := secp256k1.S256()

	if entropy == nil {
		r = rand.Reader
	} else {
		padded := padEntropy(entropy, crv.Params().BitSize/8+8)
		r = bytes.NewReader(padded)
	}

	key, err := ecdsa.GenerateKey(crv, r)
	if err != nil {
		return nil, err
	}

	//Key ID will be last four bytes of the sha256 of the public key in hex
	pubKeyHash := sha256.Sum256(key.PublicKey.X.Bytes())

	joseKey := jose.JSONWebKey{
		KeyID:     fmt.Sprintf("sig_%x", pubKeyHash[len(pubKeyHash)-4:]),
		Key:       key,
		Algorithm: string(jose.ES256K),
	}

	return &JSONWebKey{
		keyType: KeyType("JSONWebKey2020"),
		key:     &joseKey,
	}, nil

}

func padEntropy(b []byte, length int) []byte {
	if len(b) >= length {
		return b[:length]
	}

	return append(b, make([]byte, length-len(b))...)
}
