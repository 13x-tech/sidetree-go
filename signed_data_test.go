package sidetree

import (
	"encoding/json"
	"testing"
)

// TestSignedDataJSONTag pins the spec-correct "signedData" wire key for all three
// proof-operation types, in both directions. Before the json tags were added the
// field (un)marshaled as "SignedData", so real ION proof files decoded to an
// empty SignedData and verification was silently skipped.
func TestSignedDataJSONTag(t *testing.T) {
	const jws = "eyJhbGciOiJFUzI1NksifQ.cGF5bG9hZA.c2ln"
	const wire = `{"signedData":"` + jws + `"}`

	t.Run("update", func(t *testing.T) {
		var op SignedUpdateDataOp
		if err := json.Unmarshal([]byte(wire), &op); err != nil {
			t.Fatal(err)
		}
		if op.SignedData != jws {
			t.Errorf("decoded signedData = %q, want %q", op.SignedData, jws)
		}
		b, _ := json.Marshal(op)
		if string(b) != wire {
			t.Errorf("encoded = %s, want %s", b, wire)
		}
	})
	t.Run("recover", func(t *testing.T) {
		var op SignedRecoverDataOp
		if err := json.Unmarshal([]byte(wire), &op); err != nil || op.SignedData != jws {
			t.Errorf("recover signedData = %q (err=%v), want %q", op.SignedData, err, jws)
		}
	})
	t.Run("deactivate", func(t *testing.T) {
		var op SignedDeactivateDataOp
		if err := json.Unmarshal([]byte(wire), &op); err != nil || op.SignedData != jws {
			t.Errorf("deactivate signedData = %q (err=%v), want %q", op.SignedData, err, jws)
		}
	})
}
