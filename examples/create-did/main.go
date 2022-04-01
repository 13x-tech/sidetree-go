package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/13x-tech/sidetree-go/internal/did"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/gowebpki/jcs"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func genKey(crv elliptic.Curve, purposes []string) (did.KeyInfo, error) {
	key, err := ecdsa.GenerateKey(crv, rand.Reader)
	if err != nil {
		panic(err)
	}

	keyInfo, err := jwk.FromRaw(key)
	if err != nil {
		panic(err)
	}

	return JWKtoDIDKey(keyInfo, purposes)
}

//This is temporary
func main() {
	fmt.Println("Generating Test Identity...")

	crv := secp256k1.S256()

	key1, err := genKey(crv, []string{"authentication"})
	if err != nil {
		panic(err)
	}
	key2, err := genKey(elliptic.P256(), []string{"authentication", "assertionMethod", "capabilityDelegation", "capabilityInvocation", "keyAgreement"})
	if err != nil {
		panic(err)
	}

	services := []did.Service{
		{
			ID:   "linkeddomains",
			Type: "LinkedDomains",
			ServiceEndpoint: map[string]interface{}{
				"origins": []string{"https://www.linkeddomains.com"},
			},
		},
		{
			ID:   "dwa",
			Type: "DecentralizedWebApp",
			ServiceEndpoint: map[string]interface{}{
				"nodes": []string{"https://dwn.example.com", "https://dwn2.example.com"},
			},
		},
	}

	did, err := did.Create(
		did.WithGenerateKeys(crv),
		did.WithPubKeys(key1, key2),
		did.WithServices(services...),
	)

	if err != nil {
		panic(err)
	}

	longform, err := did.LongFormURI()
	if err != nil {
		panic(err)
	}

	shortform, err := did.URI()
	if err != nil {
		panic(err)
	}

	fmt.Printf("ShortForm DID: %s\n", shortform)
	fmt.Printf("LongForm DID: %s\n", longform)

}

func JWKtoDIDKey(key jwk.Key, purposes []string) (did.KeyInfo, error) {

	didKey := did.KeyInfo{}

	fingerPrint, err := key.Thumbprint(crypto.SHA256)
	if err != nil {
		return didKey, fmt.Errorf("failed to get key id: %w", err)
	}

	didKey.ID = fmt.Sprintf("sig_%x", fingerPrint[len(fingerPrint)-4:])
	didKey.Type, err = keyType(key)
	if err != nil {
		return did.KeyInfo{}, fmt.Errorf("failed to get key type: %w", err)
	}

	keyData, err := json.Marshal(key)
	if err != nil {
		return did.KeyInfo{}, fmt.Errorf("failed to marshal key: %w", err)
	}

	keyJSON, err := jcs.Transform(keyData)
	if err != nil {
		return did.KeyInfo{}, fmt.Errorf("failed to transform key: %w", err)
	}

	var keyMap map[string]interface{}
	if err := json.Unmarshal(keyJSON, &keyMap); err != nil {
		return did.KeyInfo{}, fmt.Errorf("failed to unmarshal key: %w", err)
	}

	didKey.PubKey = keyMap
	didKey.Purposes = purposes

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
