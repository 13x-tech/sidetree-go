package sidetree

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/13x-tech/sidetree-go/internal/did"
)

type Closer struct{}

func (c *Closer) Close() error {
	return nil
}

func NewTestStorage() *TestStorage {
	return &TestStorage{
		index: &TestIndexerStorage{
			indexOps: map[int][]SideTreeOp{},
			didOps:   map[string][]SideTreeOp{},
		},
		dids: &TestDIDsStorage{
			dids:        map[string]*did.Doc{},
			deactivated: map[string]struct{}{},
			mu:          sync.Mutex{},
		},
		cas: &TestCASStorage{
			cas: map[string][]byte{},
			mu:  sync.Mutex{},
		},
	}
}

type TestStorage struct {
	Closer
	index *TestIndexerStorage
	dids  *TestDIDsStorage
	cas   *TestCASStorage
}

func (t *TestStorage) Indexer() (Indexer, error) {
	return t.index, nil
}

func (t *TestStorage) DIDs() (DIDs, error) {
	return t.dids, nil
}

func (t *TestStorage) CAS() (CAS, error) {
	return t.cas, nil
}

type TestCASStorage struct {
	Closer
	mu  sync.Mutex
	cas map[string][]byte
}

func (t *TestCASStorage) Start() error {
	return nil
}

func (t *TestCASStorage) PutGZip(data []byte) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	id := fmt.Sprintf("%x", sha256.Sum256(data))
	t.cas[id] = data
	return id, nil
}

func (t *TestCASStorage) GetGZip(id string) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	data, ok := t.cas[id]
	if !ok {
		return nil, fmt.Errorf("no data found for id %s", id)
	}
	return data, nil
}

func (t *TestCASStorage) insertObject(id string, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cas[id] = data
	return nil
}

type TestDIDsStorage struct {
	Closer
	mu          sync.Mutex
	dids        map[string]*did.Doc
	deactivated map[string]struct{}
}

func (t *TestDIDsStorage) Put(doc *did.Doc) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.dids[doc.DIDDocument.ID] = doc
	return nil
}

func (t *TestDIDsStorage) Deactivate(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.deactivated[id] = struct{}{}
	return nil
}

func (t *TestDIDsStorage) Recover(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.deactivated, id)
	return nil
}

func (t *TestDIDsStorage) Get(id string) (*did.Doc, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.deactivated[id]
	if ok {
		return nil, fmt.Errorf("did %s is deactivated", id)
	}

	doc, ok := t.dids[id]
	if !ok {
		return nil, fmt.Errorf("no DID found for id %s", id)
	}
	return doc, nil
}

func (t *TestDIDsStorage) List() ([]string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	var ids []string
	for id := range t.dids {
		_, ok := t.deactivated[id]
		if !ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

type TestIndexerStorage struct {
	Closer
	mu       sync.Mutex
	didOps   map[string][]SideTreeOp
	indexOps map[int][]SideTreeOp
}

func (t *TestIndexerStorage) PutOps(index int, ops []SideTreeOp) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.indexOps[index] = ops
	return nil
}

func (t *TestIndexerStorage) GetOps(index int) ([]SideTreeOp, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	ops, ok := t.indexOps[index]
	if !ok {
		return nil, fmt.Errorf("no ops found for index %d", index)
	}
	return ops, nil
}

func (t *TestIndexerStorage) PutDIDOps(id string, ops []SideTreeOp) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.didOps[id] = ops
	return nil
}

func (t *TestIndexerStorage) GetDIDOps(id string) ([]SideTreeOp, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	ops, ok := t.didOps[id]
	if !ok {
		return nil, fmt.Errorf("no ops found for id %s", id)
	}
	return ops, nil
}

var storageTestOps = []SideTreeOp{
	{
		SystemAnchorPoint: "1234:abcd:56:efgh:7",
		AnchorString:      "1.QmXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		Processed:         false,
	},
	{
		SystemAnchorPoint: "1234:efgh:75:efgh:7",
		AnchorString:      "1.QmXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		Processed:         false,
	},
	{
		SystemAnchorPoint: "1234:ijkl:89:efgh:7",
		AnchorString:      "1.QmXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		Processed:         false,
	},
}

func TestIndexeOps(t *testing.T) {
	storage := NewTestStorage()
	indexer, err := storage.Indexer()
	if err != nil {
		t.Errorf("Error creating indexer: %v", err)
	}
	if indexer == nil {
		t.Error("Indexer is nil")
	}

	if err := indexer.PutOps(1234, storageTestOps); err != nil {
		t.Errorf("Error putting ops: %v", err)
	}

	ops, err := indexer.GetOps(1234)
	if err != nil {
		t.Errorf("Error getting ops: %v", err)
	}

	if !reflect.DeepEqual(ops, storageTestOps) {
		t.Errorf("Ops not equal: %v", ops)
	}
}

func TestDIDOps(t *testing.T) {
	storage := NewTestStorage()
	indexer, err := storage.Indexer()
	if err != nil {
		t.Errorf("Error creating Indexer: %v", err)
	}
	if indexer == nil {
		t.Error("Indexer is nil")
	}

	if err := indexer.PutDIDOps("abc", storageTestOps); err != nil {
		t.Errorf("Error putting ops: %v", err)
	}

	ops, err := indexer.GetDIDOps("abc")
	if err != nil {
		t.Errorf("Error getting ops: %v", err)
	}

	if !reflect.DeepEqual(ops, storageTestOps) {
		t.Errorf("Ops not equal: %v", ops)
	}

}

func TestDIDs(t *testing.T) {
	storage := NewTestStorage()
	dids, err := storage.DIDs()
	if err != nil {
		t.Errorf("Error creating DIDs storage: %v", err)
	}
	if dids == nil {
		t.Errorf("DIDs storage is nil")
	}

	doc := testDoc()

	if err := dids.Put(doc); err != nil {
		t.Errorf("Error putting DID: %v", err)
	}

	doc2, err := dids.Get(doc.DIDDocument.ID)
	if err != nil {
		t.Errorf("Error getting DID: %v", err)
	}

	if !reflect.DeepEqual(doc, doc2) {
		t.Errorf("DIDs not equal: %v", doc)
	}

	if err := dids.Deactivate(doc.DIDDocument.ID); err != nil {
		t.Errorf("Error deactivating DID: %v", err)
	}

	_, err = dids.Get(doc.DIDDocument.ID)
	if err == nil {
		t.Errorf("Deactivated did should not be found")
	}

	if err := dids.Recover(doc.DIDDocument.ID); err != nil {
		t.Errorf("Error recovering DID: %v", err)
	}

	doc3, err := dids.Get(doc.DIDDocument.ID)
	if err != nil {
		t.Errorf("Error getting DID: %v", err)
	}

	if !reflect.DeepEqual(doc, doc3) {
		t.Errorf("DIDs not equal: %v", doc)
	}

}

func TestCAS(t *testing.T) {

	testObject, err := json.Marshal(storageTestOps)
	if err != nil {
		t.Errorf("Error marshalling test object: %v", err)
	}

	storage := NewTestStorage()
	cas, err := storage.CAS()
	if err != nil {
		t.Errorf("Error creating CAS: %v", err)
	}
	if cas == nil {
		t.Errorf("CAS is nil")
	}

	if err := cas.Start(); err != nil {
		t.Errorf("Error starting CAS: %v", err)
	}

	if err := cas.(*TestCASStorage).insertObject("QmXXXXX", testObject); err != nil {
		t.Errorf("Error inserting test object: %v", err)
	}

	fetchedObject, err := cas.GetGZip("QmXXXXX")
	if err != nil {
		t.Errorf("Error getting gzip: %v", err)
	}

	if !bytes.Equal(testObject, fetchedObject) {
		t.Errorf("Objects not equal: %v", testObject)
	}

}

func testDoc() *did.Doc {

	return &did.Doc{
		Context: "https://w3id.org/did-resolution/v1",
		DIDDocument: &did.DIDDocData{
			ID:    "EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
			DocID: "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
			Context: []interface{}{
				"https://www.w3.org/ns/did/v1",
				map[string]interface{}{"@base": "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ"},
			},
			Services: []did.DIDService{{
				ID:   "#linkeddomains",
				Type: "LinkedDomains",
				ServiceEndpoint: map[string]interface{}{
					"origins": []string{"https://woodgrove.com/"},
				},
			}},
			Verification: []did.DIDKeyInfo{{
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
		Metadata: did.DIDMetadata{
			CanonicalId: "did:ion:EiBCyVAW45f9xyh_RbA6ZK4aM2gndCOjg8-mYfCVHXShVQ",
			Method: did.DIDMetadataMethod{
				Published:          true,
				UpdateCommitment:   "EiAGj7alOM1_2pVQv_Phbw3928zlVWWvMYuLsvuDnSuImg",
				RecoveryCommitment: "EiB_FKDwQpnzkrD9Rwvu9puF8WUYdOvO06lX1F0LoF7WKw",
			},
		},
	}
}
