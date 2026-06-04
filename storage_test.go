package sidetree

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

type Closer struct{}

func (c *Closer) Close() error {
	return nil
}

func NewTestCAS() *TestCASStorage {
	return &TestCASStorage{
		cas:      make(map[string][]byte),
		maxSizes: make(map[string]int),
	}
}

type TestCASStorage struct {
	Closer
	mu  sync.Mutex
	cas map[string][]byte
	// maxSizes records the maxSizeInBytes the last Get for each id was called
	// with, so tests can assert the caller passed the correct per-file cap.
	maxSizes map[string]int
}

func (t *TestCASStorage) Start() error {
	return nil
}

func (t *TestCASStorage) Type() CASType {
	return CASType("test")
}

func (t *TestCASStorage) Put(data []byte) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	id := fmt.Sprintf("%x", sha256.Sum256(data))
	t.cas[id] = data
	return id, nil
}

// Get records the requested cap (for assertions) but intentionally does NOT
// enforce it, so tests can stage oversized content and exercise the caller's
// defensive checkFileSize guard. A real CAS bounds the download itself.
func (t *TestCASStorage) Get(id string, maxSizeInBytes int) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.maxSizes[id] = maxSizeInBytes
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

var storageTestOps = []operations.Anchor{
	{
		Sequence: "1234:abcd:56:efgh:7",
		Anchor:   "1.QmXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
	},
	{
		Sequence: "1234:efgh:75:efgh:7",
		Anchor:   "1.QmXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
	},
	{
		Sequence: "1234:ijkl:89:efgh:7",
		Anchor:   "1.QmXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
	},
}

func TestCAS(t *testing.T) {

	testObject, err := json.Marshal(storageTestOps)
	if err != nil {
		t.Errorf("Error marshalling test object: %v", err)
	}

	cas := NewTestCAS()
	if err != nil {
		t.Errorf("Error creating CAS: %v", err)
	}
	if cas == nil {
		t.Errorf("CAS is nil")
	}

	if err := cas.Start(); err != nil {
		t.Errorf("Error starting CAS: %v", err)
	}

	if err := cas.insertObject("QmXXXXX", testObject); err != nil {
		t.Errorf("Error inserting test object: %v", err)
	}

	fetchedObject, err := cas.Get("QmXXXXX", MaxCoreIndexFileSizeInBytes)
	if err != nil {
		t.Errorf("Error getting gzip: %v", err)
	}

	if !bytes.Equal(testObject, fetchedObject) {
		t.Errorf("Objects not equal: %v", testObject)
	}

}
