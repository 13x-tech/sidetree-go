package main

import (
	"encoding/json"
	"fmt"

	"github.com/13x-tech/sidetree-go/internal/keys"
	"github.com/13x-tech/sidetree-go/pkg/did"

	"github.com/gowebpki/jcs"
)

func genKey(purposes []string) (did.KeyInfo, error) {
	key, err := keys.GenerateES256K([]byte("testing shitty entropy"))
	if err != nil {
		return did.KeyInfo{}, fmt.Errorf("failed to generate key: %w", err)
	}

	return JWKtoDIDKey(key, purposes)
}

//This is temporary
func main() {
	fmt.Println("Generating Test Identity...")

	key1, err := genKey([]string{"authentication", "assertionMethod", "capabilityDelegation", "capabilityInvocation", "keyAgreement"})
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

	id, err := did.Create(
		did.WithGenerateKeys(),
		did.WithPubKeys(key1),
		did.WithServices(services...),
	)

	if err != nil {
		panic(err)
	}

	longform, err := id.LongFormURI()
	if err != nil {
		panic(err)
	}

	shortform, err := id.URI()
	if err != nil {
		panic(err)
	}

	fmt.Printf("ShortForm DID: %s\n", shortform)
	fmt.Printf("LongForm DID: %s\n", longform)

	didSuffix, _, err := did.ParseLongForm(longform)
	if err != nil {
		panic(err)
	}

	shortFormCheck, err := didSuffix.URI()
	if err != nil {
		panic(err)
	}

	fmt.Printf("DID Suffix: %s\n", shortFormCheck)

}

func JWKtoDIDKey(key *keys.JSONWebKey, purposes []string) (did.KeyInfo, error) {

	didKey := did.KeyInfo{}

	didKey.ID = key.KeyID()
	didKey.Type = key.KeyType().String()

	keyData, err := key.MarshalJSON()
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
