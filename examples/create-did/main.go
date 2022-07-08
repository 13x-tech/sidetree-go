package main

import (
	"encoding/json"
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/api"
	"github.com/13x-tech/ion-sdk-go/pkg/did"
	"github.com/13x-tech/ion-sdk-go/pkg/keys"
	"github.com/13x-tech/ion-sdk-go/pkg/operations/create"

	"github.com/gowebpki/jcs"
)

func genKey(purposes []string) (did.KeyInfo, error) {
	key, err := keys.GenerateES256K([]byte("testing shitty entropy"))
	if err != nil {
		return did.KeyInfo{}, fmt.Errorf("failed to generate key: %w", err)
	}

	return JWK(key, purposes)
}

//This is temporary
func main() {

	fmt.Println("Generating Test Identity...")
	recoverKey, err := keys.GenerateES256K([]byte("some bullshit"))
	if err != nil {
		panic(err)
	}

	updateKey, err := keys.GenerateES256K([]byte("some bullshit 2"))
	if err != nil {
		panic(err)
	}
	deviceKey, err := genKey([]string{"authentication", "assertionMethod", "capabilityDelegation", "capabilityInvocation", "keyAgreement"})
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

	id, err := create.New(
		create.WithUpdateKey(updateKey),
		create.WithRecoverKey(recoverKey),
		create.WithPubKeys(deviceKey),
		create.WithServices(services...),
	)

	if err != nil {
		panic(err)
	}

	suffixData, delta, err := id.Operation()
	if err != nil {
		panic(err)
	}

	ion, err := api.New(
		api.WithEndpoint("https://kenny-1.cow-tone.ts.net/operations"),
		api.WithChallenge("https://beta.ion.msidentity.com/api/v1.0/proof-of-work-challenge"),
	)

	createOp := api.CreateOperation(suffixData, delta)

	response, err := ion.Submit(createOp)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Response: %s", response)
}

func JWK(key *keys.DIDKey, purposes []string) (did.KeyInfo, error) {
	return JWKWithID(key, purposes, "")
}

func JWKWithID(key *keys.DIDKey, purposes []string, id string) (did.KeyInfo, error) {

	didKey := did.KeyInfo{}

	if len(id) > 0 {
		didKey.ID = id
	} else {
		keyId, err := key.ID()
		if err != nil {
			return didKey, fmt.Errorf("could not generate KeyID: %w", err)
		}
		didKey.ID = keyId
	}

	didKey.Type = key.KeyType().String()
	keyData, err := key.Key().MarshalJSON()
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
