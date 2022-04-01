package did

import (
	"testing"
)

func testDoc() *Doc {

	return &Doc{
		Context: "https://w3id.org/did-resolution/v1",
		DIDDocument: &DIDDocData{
			ID:    "EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
			DocID: "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
			Context: []interface{}{
				"https://www.w3.org/ns/did/v1",
				map[string]interface{}{"@base": "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ"},
			},
			Services: []DIDService{{
				ID:   "#linkeddomains",
				Type: "LinkedDomains",
				ServiceEndpoint: map[string]interface{}{
					"origins": []string{"https://woodgrove.com/"},
				},
			}},
			Verification: []DIDKeyInfo{{
				ID:         "#sig_44a9661f",
				Controller: "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
				Type:       "EcdsaSecp256k1VerificationKey2019",
				PubKey: map[string]interface{}{
					"kty": "EC",
					"crv": "secp256k1",
					"x":   "sE3ra-hJlRySLrZVSOwxnJtb2u9h_njbNKG8c53QEqo",
					"y":   "zERmPj751qx6-AL9n60eIojS-Qp9BcYB2IKEMrl0E3c",
				}},
			},
			Authentication:       []string{"#sig_44a9661f"},
			Assertion:            []string{"#sig_44a9661f"},
			CapabilityDelegation: []string{"#sig_44a9661f"},
			CapabilityInvocation: []string{"#sig_44a9661f"},
			KeyAgreement:         []string{"#sig_44a9661f"},
		},
		Metadata: DIDMetadata{
			CanonicalId: "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
			Method: DIDMetadataMethod{
				Published:          true,
				UpdateCommitment:   "EiAGj7alOM1_2pVQv_Phbw3928zlVWWvMYuLsvuDnSuImg",
				RecoveryCommitment: "EiB_FKDwQpnzkrD9Rwvu9puF8WUYdOvO06lX1F0LoF7WKw",
			},
		},
	}
}

var TestKey = DIDKeyInfo{
	ID:         "#sig_xxxxxx",
	Controller: "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
	Type:       "EcdsaSecp256k1VerificationKey2019",
	PubKey: map[string]interface{}{
		"kty": "EC",
		"crv": "secp256k1",
		"x":   "xxxxxxx",
		"y":   "yyyyyyy",
	},
	Purposes: []string{
		"authentication",
		"keyAgreement",
		"assertionMethod",
		"capabilityDelegation",
		"capabilityInvocation",
	},
}

var TestServie = DIDService{
	ID:   "#identityHub",
	Type: "IdentityHub",
	ServiceEndpoint: map[string]interface{}{
		"instances": []string{"https://identity.ion.org"},
	},
}

func TestResetData(t *testing.T) {

	doc := testDoc()
	doc.DIDDocument.ResetData()

	if len(doc.DIDDocument.Services) != 0 {
		t.Errorf("Services length should be 0 got %d", len(doc.DIDDocument.Services))
	}
	if len(doc.DIDDocument.Verification) != 0 {
		t.Errorf("Verification length should be 0 got %d", len(doc.DIDDocument.Verification))
	}
	if len(doc.DIDDocument.Authentication) != 0 {
		t.Errorf("Authentication length should be 0 got %d", len(doc.DIDDocument.Authentication))
	}
	if len(doc.DIDDocument.Assertion) != 0 {
		t.Errorf("Assertion length should be 0 got %d", len(doc.DIDDocument.Assertion))
	}
	if len(doc.DIDDocument.CapabilityDelegation) != 0 {
		t.Errorf("CapabilityDelegation length should be 0 got %d", len(doc.DIDDocument.CapabilityDelegation))
	}
	if len(doc.DIDDocument.CapabilityInvocation) != 0 {
		t.Errorf("CapabilityInvocation length should be 0 got %d", len(doc.DIDDocument.CapabilityInvocation))
	}
	if len(doc.DIDDocument.KeyAgreement) != 0 {
		t.Errorf("KeyAgreement length should be 0 got %d", len(doc.DIDDocument.KeyAgreement))
	}
}

func TestAddPublicKeys(t *testing.T) {

	doc := testDoc()

	if len(doc.DIDDocument.Verification) != 1 {
		t.Errorf("Verification length should be 1 got %d", len(doc.DIDDocument.Verification))
	}

	doc.DIDDocument.AddPublicKeys([]DIDKeyInfo{TestKey})

	if len(doc.DIDDocument.Verification) != 2 {
		t.Errorf("Verification length should be 2 got %d", len(doc.DIDDocument.Verification))
	}

	foundKey := false
	for _, key := range doc.DIDDocument.Verification {
		if key.ID == "#sig_xxxxxx" {
			foundKey = true
			break
		}
	}

	if !foundKey {
		t.Error("Key not found")
	}

	foundAuth := false
	for _, authentication := range doc.DIDDocument.Authentication {
		if authentication == "#sig_xxxxxx" {
			foundAuth = true
			break
		}
	}

	if !foundAuth {
		t.Error("Authentication not found")
	}

	foundAssertion := false
	for _, assertion := range doc.DIDDocument.Assertion {
		if assertion == "#sig_xxxxxx" {
			foundAssertion = true
			break
		}
	}

	if !foundAssertion {
		t.Error("Assertion not found")
	}

	foundCapabilityDelegation := false
	for _, capabilityDelegation := range doc.DIDDocument.CapabilityDelegation {
		if capabilityDelegation == "#sig_xxxxxx" {
			foundCapabilityDelegation = true
			break
		}
	}

	if !foundCapabilityDelegation {
		t.Error("CapabilityDelegation not found")
	}

	foundCapabilityInvocation := false
	for _, capabilityInvocation := range doc.DIDDocument.CapabilityInvocation {
		if capabilityInvocation == "#sig_xxxxxx" {
			foundCapabilityInvocation = true
			break
		}
	}

	if !foundCapabilityInvocation {
		t.Error("CapabilityInvocation not found")
	}

	foundKeyAgreement := false
	for _, keyAgreement := range doc.DIDDocument.KeyAgreement {
		if keyAgreement == "#sig_xxxxxx" {
			foundKeyAgreement = true
			break
		}
	}

	if !foundKeyAgreement {
		t.Error("KeyAgreement not found")
	}
}

func TestRemovePublicKeys(t *testing.T) {
	doc := testDoc()
	doc.DIDDocument.AddPublicKeys([]DIDKeyInfo{TestKey})

	foundKey := false
	for _, key := range doc.DIDDocument.Verification {
		if key.ID == "#sig_xxxxxx" {
			foundKey = true
			break
		}
	}

	if !foundKey {
		t.Error("Key not found")
	}

	if err := doc.DIDDocument.RemovePublicKeys([]string{"#sig_aaaaaa"}); err == nil {
		t.Error("RemovePublicKey should return error for a key that does not exist")
	}

	if err := doc.DIDDocument.RemovePublicKeys([]string{"#sig_xxxxxx"}); err != nil {
		t.Errorf("Error removing key: %s", err)
	}

	foundKey = false
	for _, key := range doc.DIDDocument.Verification {
		if key.ID == "#sig_xxxxxx" {
			foundKey = true
			break
		}
	}

	if foundKey {
		t.Error("Key found")
	}
}

func TestAddServices(t *testing.T) {

	doc := testDoc()
	doc.DIDDocument.AddServices([]DIDService{TestServie})

	foundService := false
	var testService DIDService
	for _, service := range doc.DIDDocument.Services {
		if service.ID == "#identityHub" {
			testService = service
			foundService = true
			break
		}
	}

	if !foundService {
		t.Error("Service not found")
	}

	foundInstace := false
	serviceEndpoint := testService.ServiceEndpoint.(map[string]interface{})
	for _, instance := range serviceEndpoint["instances"].([]string) {
		if instance == "https://identity.ion.org" {
			foundInstace = true
			break
		}
	}

	if !foundInstace {
		t.Error("Instance not found")
	}
}

func TestRemoveService(t *testing.T) {

	doc := testDoc()

	foundService := false
	for _, service := range doc.DIDDocument.Services {
		if service.ID == "#linkeddomains" {
			foundService = true
			break
		}
	}

	if !foundService {
		t.Error("Service not found")
	}

	if err := doc.DIDDocument.RemoveServices([]string{"linkeddomainsss"}); err == nil {
		t.Error("RemoveServices should return error for a service that does not exist")
	}

	if err := doc.DIDDocument.RemoveServices([]string{"linkeddomains"}); err != nil {
		t.Errorf("Error removing service: %s", err)
	}

	foundService = false
	for _, service := range doc.DIDDocument.Services {
		if service.ID == "#linkeddomains" {
			foundService = true
			break
		}
	}

	if foundService {
		t.Error("Service found")
	}
}
