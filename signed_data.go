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

	parsed           *jose.JSONWebSignature
	protectedPayload *ProtectedPayload
}

func (s *SignedDataOp) DeltaHash() (string, error) {
	if s.parsed == nil || s.protectedPayload == nil {
		if err := s.parse(); err != nil {
			return "", fmt.Errorf("failed to parse signed data op: %w", err)
		}
	}

	return s.protectedPayload.DeltaHash, nil
}

func (s *SignedDataOp) parse() error {
	var err error
	s.parsed, err = jose.ParseSigned(s.SignedData)
	if err != nil {
		return fmt.Errorf("failed to parse signed data: %w", err)
	}

	payload := s.parsed.UnsafePayloadWithoutVerification()
	var protectedPayload ProtectedPayload

	if err := json.Unmarshal(payload, &protectedPayload); err != nil {
		return fmt.Errorf("failed to unmarshal protected payload: %w", err)
	}

	s.protectedPayload = &protectedPayload
	return nil
}

func (s *SignedDataOp) ValidateReveal(revealValue string) (bool, error) {

	if s.parsed == nil {
		if err := s.parse(); err != nil {
			return false, fmt.Errorf("failed to parse signed data: %w", err)
		}
	}

	jsonKey, err := s.protectedPayload.GetKeyData()
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

	verified, err := s.parsed.Verify(&key)
	if err != nil {
		return false, fmt.Errorf("failed to verify signature: %w", err)
	}

	unsafePayload := s.parsed.UnsafePayloadWithoutVerification()

	return bytes.Equal(unsafePayload, verified), nil
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
