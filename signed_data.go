package sidetree

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/go-jose/go-jose/v3"
	"github.com/gowebpki/jcs"
)

type SignedUpdateDataOp struct {
	SignedData string

	parsed           *jose.JSONWebSignature
	protectedPayload *UpdateProtectedPayload
}

func (s *SignedUpdateDataOp) DeltaHash() (string, error) {
	if s.parsed == nil || s.protectedPayload == nil {
		if err := s.parse(); err != nil {
			return "", fmt.Errorf("failed to parse signed data op: %w", err)
		}
	}

	return s.protectedPayload.DeltaHash, nil
}

func (s *SignedUpdateDataOp) parse() error {
	var err error
	s.parsed, err = jose.ParseSigned(s.SignedData)
	if err != nil {
		return fmt.Errorf("failed to parse signed data: %w", err)
	}

	payload := s.parsed.UnsafePayloadWithoutVerification()
	var protectedPayload UpdateProtectedPayload

	if err := json.Unmarshal(payload, &protectedPayload); err != nil {
		return fmt.Errorf("failed to unmarshal protected payload: %w", err)
	}

	s.protectedPayload = &protectedPayload
	return nil
}

func (s *SignedUpdateDataOp) ValidateReveal(revealValue string) (bool, error) {

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

type UpdateProtectedPayload struct {
	UpdateKey map[string]interface{} `json:"updateKey"`
	DeltaHash string                 `json:"deltaHash"`
}

func (p *UpdateProtectedPayload) GetKeyData() ([]byte, error) {

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

type SignedRecoverDataOp struct {
	SignedData string

	parsed           *jose.JSONWebSignature
	protectedPayload *RecoverProtectedPayload
}

func (s *SignedRecoverDataOp) DeltaHash() (string, error) {
	if s.parsed == nil || s.protectedPayload == nil {
		if err := s.parse(); err != nil {
			return "", fmt.Errorf("failed to parse signed data op: %w", err)
		}
	}

	return s.protectedPayload.DeltaHash, nil
}

func (s *SignedRecoverDataOp) parse() error {
	var err error
	s.parsed, err = jose.ParseSigned(s.SignedData)
	if err != nil {
		return fmt.Errorf("failed to parse signed data: %w", err)
	}

	payload := s.parsed.UnsafePayloadWithoutVerification()
	var protectedPayload RecoverProtectedPayload

	if err := json.Unmarshal(payload, &protectedPayload); err != nil {
		return fmt.Errorf("failed to unmarshal protected payload: %w", err)
	}

	s.protectedPayload = &protectedPayload
	return nil
}

func (s *SignedRecoverDataOp) ValidateReveal(revealValue string) (bool, error) {

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

type RecoverProtectedPayload struct {
	RecoveryCommitment string                 `json:"recoveryCommitment"`
	RecoveryKey        map[string]interface{} `json:"recoveryKey"`
	DeltaHash          string                 `json:"deltaHash"`
	AnchorOrigin       string                 `json:"anchorOrigin,omitempty"`
}

func (p *RecoverProtectedPayload) GetKeyData() ([]byte, error) {

	keyData, err := json.Marshal(p.RecoveryKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal provisional proof operation: %w", err)
	}

	jsonKeyData, err := jcs.Transform(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to transform provisional proof operation: %w", err)
	}

	return jsonKeyData, nil
}

type SignedDeactivateDataOp struct {
	SignedData string

	parsed           *jose.JSONWebSignature
	protectedPayload *DeactivateProtectedPayload
}

func (s *SignedDeactivateDataOp) parse() error {
	var err error
	s.parsed, err = jose.ParseSigned(s.SignedData)
	if err != nil {
		return fmt.Errorf("failed to parse signed data: %w", err)
	}

	payload := s.parsed.UnsafePayloadWithoutVerification()
	var protectedPayload DeactivateProtectedPayload

	if err := json.Unmarshal(payload, &protectedPayload); err != nil {
		return fmt.Errorf("failed to unmarshal protected payload: %w", err)
	}

	s.protectedPayload = &protectedPayload
	return nil
}

func (s *SignedDeactivateDataOp) ValidateReveal(revealValue string) (bool, error) {

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

type DeactivateProtectedPayload struct {
	DIDSuffix   string                 `json:"didSuffix"`
	RecoveryKey map[string]interface{} `json:"recoveryKey"`
}

func (p *DeactivateProtectedPayload) GetKeyData() ([]byte, error) {

	keyData, err := json.Marshal(p.RecoveryKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal provisional proof operation: %w", err)
	}

	jsonKeyData, err := jcs.Transform(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to transform provisional proof operation: %w", err)
	}

	return jsonKeyData, nil
}
