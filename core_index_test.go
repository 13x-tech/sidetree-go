package sidetree

import (
	"encoding/json"
	"testing"
)

var testCoreFile = CoreIndexFile{
	Operations: CoreOperations{
		Create: []CreateOperation{},
		Recover: []Operation{{
			DIDSuffix: "abcdefg",
		}},
		Deactivate: []Operation{{
			DIDSuffix: "abcdefg",
		}},
	},
}

func TestDuplicateOperations(t *testing.T) {
	t.Run("test duplicates", func(t *testing.T) {
		t.Run("recover & deactivate", func(t *testing.T) {

			var testCoreFile = CoreIndexFile{
				Operations: CoreOperations{
					Create: []CreateOperation{},
					Recover: []Operation{{
						DIDSuffix: "abcdefg",
					}},
					Deactivate: []Operation{{
						DIDSuffix: "abcdefg",
					}},
				},
			}

			testFileJSON, err := json.Marshal(testCoreFile)
			if err != nil {
				t.Errorf("Error marshalling test core file: %v", err)
			}

			c, err := NewCoreIndexFile(nil, testFileJSON)
			if err != nil {
				t.Errorf("failed to create core index file: %v", err)
			}

			if err := c.populateCoreOperationArray(); err == nil {
				t.Errorf("expected error when populating multiple core operations for the same id")
			}
		})
		t.Run("recover", func(t *testing.T) {

			var testCoreFile = CoreIndexFile{
				Operations: CoreOperations{
					Create: []CreateOperation{},
					Recover: []Operation{{
						DIDSuffix: "abcdefg",
					}, {
						DIDSuffix: "abcdefg",
					}},
					Deactivate: []Operation{},
				},
			}

			testFileJSON, err := json.Marshal(testCoreFile)
			if err != nil {
				t.Errorf("Error marshalling test core file: %v", err)
			}

			c, err := NewCoreIndexFile(nil, testFileJSON)
			if err != nil {
				t.Errorf("failed to create core index file: %v", err)
			}

			if err := c.populateCoreOperationArray(); err == nil {
				t.Errorf("expected error when populating multiple core operations for the same id")
			}
		})
	})

	t.Run("no duplicates", func(t *testing.T) {
		var testCoreFile = CoreIndexFile{
			Operations: CoreOperations{
				Create: []CreateOperation{},
				Recover: []Operation{{
					DIDSuffix: "abcdefg",
				}},
				Deactivate: []Operation{{
					DIDSuffix: "hijklmn",
				}},
			},
		}

		testFileJSON, err := json.Marshal(testCoreFile)
		if err != nil {
			t.Errorf("Error marshalling test core file: %v", err)
		}

		c, err := NewCoreIndexFile(nil, testFileJSON)
		if err != nil {
			t.Errorf("failed to create core index file: %v", err)
		}

		if err := c.populateCoreOperationArray(); err != nil {
			t.Errorf("failed to populate core operations array: %v", err)
		}
	})

}
