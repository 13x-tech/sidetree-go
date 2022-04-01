package did

import (
	"testing"
)

func testDoc() *Document {

	return &Document{
		Context: "https://w3id.org/did-resolution/v1",
		Document: &DocumentData{
			ID:    "EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
			DocID: "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
			Context: []interface{}{
				"https://www.w3.org/ns/did/v1",
				map[string]interface{}{"@base": "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ"},
			},
			Services: []Service{{
				ID:   "#linkeddomains",
				Type: "LinkedDomains",
				ServiceEndpoint: map[string]interface{}{
					"origins": []string{"https://woodgrove.com/"},
				},
			}},
			Verification: []KeyInfo{{
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
		Metadata: Metadata{
			CanonicalId: "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
			Method: MetadataMethod{
				Published:          true,
				UpdateCommitment:   "EiAGj7alOM1_2pVQv_Phbw3928zlVWWvMYuLsvuDnSuImg",
				RecoveryCommitment: "EiB_FKDwQpnzkrD9Rwvu9puF8WUYdOvO06lX1F0LoF7WKw",
			},
		},
	}
}

var TestKey = KeyInfo{
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

var TestServie = Service{
	ID:   "#identityHub",
	Type: "IdentityHub",
	ServiceEndpoint: map[string]interface{}{
		"instances": []string{"https://identity.ion.org"},
	},
}

func TestResetData(t *testing.T) {

	doc := testDoc()
	doc.Document.ResetData()

	if len(doc.Document.Services) != 0 {
		t.Errorf("Services length should be 0 got %d", len(doc.Document.Services))
	}
	if len(doc.Document.Verification) != 0 {
		t.Errorf("Verification length should be 0 got %d", len(doc.Document.Verification))
	}
	if len(doc.Document.Authentication) != 0 {
		t.Errorf("Authentication length should be 0 got %d", len(doc.Document.Authentication))
	}
	if len(doc.Document.Assertion) != 0 {
		t.Errorf("Assertion length should be 0 got %d", len(doc.Document.Assertion))
	}
	if len(doc.Document.CapabilityDelegation) != 0 {
		t.Errorf("CapabilityDelegation length should be 0 got %d", len(doc.Document.CapabilityDelegation))
	}
	if len(doc.Document.CapabilityInvocation) != 0 {
		t.Errorf("CapabilityInvocation length should be 0 got %d", len(doc.Document.CapabilityInvocation))
	}
	if len(doc.Document.KeyAgreement) != 0 {
		t.Errorf("KeyAgreement length should be 0 got %d", len(doc.Document.KeyAgreement))
	}
}

func TestAddPublicKeys(t *testing.T) {

	doc := testDoc()

	if len(doc.Document.Verification) != 1 {
		t.Errorf("Verification length should be 1 got %d", len(doc.Document.Verification))
	}

	doc.Document.AddPublicKeys([]KeyInfo{TestKey})

	if len(doc.Document.Verification) != 2 {
		t.Errorf("Verification length should be 2 got %d", len(doc.Document.Verification))
	}

	foundKey := false
	for _, key := range doc.Document.Verification {
		if key.ID == "#sig_xxxxxx" {
			foundKey = true
			break
		}
	}

	if !foundKey {
		t.Error("Key not found")
	}

	foundAuth := false
	for _, authentication := range doc.Document.Authentication {
		if authentication == "#sig_xxxxxx" {
			foundAuth = true
			break
		}
	}

	if !foundAuth {
		t.Error("Authentication not found")
	}

	foundAssertion := false
	for _, assertion := range doc.Document.Assertion {
		if assertion == "#sig_xxxxxx" {
			foundAssertion = true
			break
		}
	}

	if !foundAssertion {
		t.Error("Assertion not found")
	}

	foundCapabilityDelegation := false
	for _, capabilityDelegation := range doc.Document.CapabilityDelegation {
		if capabilityDelegation == "#sig_xxxxxx" {
			foundCapabilityDelegation = true
			break
		}
	}

	if !foundCapabilityDelegation {
		t.Error("CapabilityDelegation not found")
	}

	foundCapabilityInvocation := false
	for _, capabilityInvocation := range doc.Document.CapabilityInvocation {
		if capabilityInvocation == "#sig_xxxxxx" {
			foundCapabilityInvocation = true
			break
		}
	}

	if !foundCapabilityInvocation {
		t.Error("CapabilityInvocation not found")
	}

	foundKeyAgreement := false
	for _, keyAgreement := range doc.Document.KeyAgreement {
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
	doc.Document.AddPublicKeys([]KeyInfo{TestKey})

	foundKey := false
	for _, key := range doc.Document.Verification {
		if key.ID == "#sig_xxxxxx" {
			foundKey = true
			break
		}
	}

	if !foundKey {
		t.Error("Key not found")
	}

	if err := doc.Document.RemovePublicKeys([]string{"#sig_aaaaaa"}); err == nil {
		t.Error("RemovePublicKey should return error for a key that does not exist")
	}

	if err := doc.Document.RemovePublicKeys([]string{"#sig_xxxxxx"}); err != nil {
		t.Errorf("Error removing key: %s", err)
	}

	foundKey = false
	for _, key := range doc.Document.Verification {
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
	doc.Document.AddServices([]Service{TestServie})

	foundService := false
	var testService Service
	for _, service := range doc.Document.Services {
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
	for _, service := range doc.Document.Services {
		if service.ID == "#linkeddomains" {
			foundService = true
			break
		}
	}

	if !foundService {
		t.Error("Service not found")
	}

	if err := doc.Document.RemoveServices([]string{"linkeddomainsss"}); err == nil {
		t.Error("RemoveServices should return error for a service that does not exist")
	}

	if err := doc.Document.RemoveServices([]string{"linkeddomains"}); err != nil {
		t.Errorf("Error removing service: %s", err)
	}

	foundService = false
	for _, service := range doc.Document.Services {
		if service.ID == "#linkeddomains" {
			foundService = true
			break
		}
	}

	if foundService {
		t.Error("Service found")
	}
}
