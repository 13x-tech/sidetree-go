package sidetree

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/go-jose/go-jose/v3"
	"github.com/gowebpki/jcs"
)

type SignedDataOp struct {
	SignedData string
}

func (s *SignedDataOp) ValidateReveal(revealValue string) (bool, error) {

	parsed, err := jose.ParseSigned(s.SignedData)
	if err != nil {
		return false, fmt.Errorf("failed to parse signed data: %w", err)
	}

	payload := parsed.UnsafePayloadWithoutVerification()

	var protectedPayload ProtectedPayload
	if err := json.Unmarshal(payload, &protectedPayload); err != nil {
		return false, fmt.Errorf("failed to unmarshal protected payload: %w", err)
	}

	jsonKey, err := protectedPayload.GetKeyData()
	if err != nil {
		return false, fmt.Errorf("failed to get key data: %w", err)
	}

	reveal, err := hashReveal(jsonKey)
	if err != nil {
		return false, fmt.Errorf("failed to hash reveal value: %w", err)
	}

	if reveal != revealValue {
		return false, fmt.Errorf("failed to validate reveal value: want %s got %s", revealValue, reveal)
	}

	key := jose.JSONWebKey{}
	if err := key.UnmarshalJSON(jsonKey); err != nil {
		return false, fmt.Errorf("failed to unmarshal json web keys: %w", err)
	}

	verified, err := parsed.Verify(&key)
	if err != nil {
		return false, fmt.Errorf("failed to verify signature: %w", err)
	}

	return bytes.Equal(payload, verified), nil
}

type ProtectedPayload struct {
	UpdateKey map[string]interface{} `json:"updateKey"`
	DeltaHash string                 `json:"deltaHash"`
}

func (p *ProtectedPayload) GetKeyData() ([]byte, error) {

	keyData, err := json.Marshal(p.UpdateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal provisional proof operation: %w", err)
	}

	jsonKeyData, err := jcs.Transform(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to transform provisional proof operation: %w", err)
	}

	return jsonKeyData, nil
}
