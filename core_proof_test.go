package sidetree

import (
	"encoding/json"
	"testing"
)

func TestNewCoreProofFile(t *testing.T) {
	t.Run("invalid core proof", func(t *testing.T) {
		_, err := NewCoreProofFile(nil, []byte("invalid"))
		if err == nil {
			t.Errorf("should have failed to create core proof file")
		}
	})
	t.Run("valid core proof", func(t *testing.T) {
		cpf := CoreProofFile{
			Operations: CoreProofOperations{
				Recover: []SignedRecoverDataOp{
					{SignedData: "some-data"},
				},
				Deactivate: []SignedDeactivateDataOp{
					{SignedData: "some-data"},
				},
			},
		}

		cpfJson, err := json.Marshal(cpf)
		if err != nil {
			t.Errorf("Error marshalling core proof file: %v", err)
			return
		}

		_, err = NewCoreProofFile(nil, cpfJson)
		if err != nil {
			t.Errorf("should have succeeded to create core proof file, got error: %v", err)
		}
	})
}

func TestCoreProofFile(t *testing.T) {
	t.Run("setDeactivateOp", func(t *testing.T) {

	})
	t.Run("setRecoverOp", func(t *testing.T) {

	})
}
