package sidetree

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/13x-tech/ion-sdk-go/pkg/crypto/util"
	"github.com/13x-tech/ion-sdk-go/pkg/did"
	"github.com/13x-tech/ion-sdk-go/pkg/keys"

	"github.com/gowebpki/jcs"
)

type Processable interface {
	Process() error
}

type Serializable interface {
	Serialize() ([]byte, error)
}

type Create interface {
	Serializable
	Processable
	Operation() (did.SuffixData, did.Delta, error)
}

type Update interface {
	Serializable
	Processable
	Operation() (didSuffix, revealValue, signedData string, delta did.Delta, err error)
}

type Deactivate interface {
	Serializable
	Processable
	Operation() (didSuffix, revealValue, signature string, err error)
}

type Recover interface {
	Serializable
	Processable
	Operation() (didSuffix, revealValue string, delta did.Delta, signature string, err error)
}

type DID struct {
	method string

	document   *did.Document
	suffixData did.SuffixData
	ops        []interface{}

	logger Logger
}

func (d *DID) SerializedOps() ([]byte, error) {
	serializedOps := []map[string]interface{}{}
	for _, op := range d.ops {
		serialized, err := serializeOp(op)
		if err != nil {
			return nil, fmt.Errorf("could not serialize did: %w", err)
		}
		serializedOps = append(serializedOps, serialized)
	}
	return json.Marshal(serializedOps)
}

func serializeOp(o interface{}) (map[string]interface{}, error) {
	var data []byte
	var err error
	switch op := o.(type) {
	case Create:
		data, err = op.Serialize()
	case Update:
		data, err = op.Serialize()
	case Deactivate:
		data, err = op.Serialize()
	case Recover:
		data, err = op.Serialize()
	default:
		return nil, fmt.Errorf("invalid operation")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to serialize operation: %w", err)
	}

	var opJSON map[string]interface{}
	if err := json.Unmarshal(data, &opJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return opJSON, nil
}

type Option func(d *DID) error

func WithOperations(ops ...interface{}) Option {
	return func(d *DID) error {
		d.ops = ops
		return nil
	}
}

func WithMethod(method string) Option {
	return func(d *DID) error {
		d.method = method
		return nil
	}
}

func NewDID(opts ...Option) (*DID, error) {
	d := new(DID)
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, fmt.Errorf("options error: %w", err)
		}
	}

	firstOp := d.firstOp()
	if firstOp == nil {
		return nil, fmt.Errorf("could not get first op")
	}

	if err := d.processCreate(firstOp); err != nil {
		return nil, fmt.Errorf("could not process create on first record")
	}

	return d, nil
}

func (d *DID) firstOp() Create {
	if len(d.ops) == 0 {
		return nil
	}
	switch op := d.ops[0].(type) {
	case Create:
		return op
	default:
		return nil
	}
}

func (d *DID) Process(operation interface{}) error {
	switch op := operation.(type) {
	case Update:
		if err := d.processUpdate(op); err != nil {
			return fmt.Errorf("could not process update: %w", err)
		}
		d.ops = append(d.ops, op)
		return nil
	case Recover:
		if err := d.processRecover(op); err != nil {
			return fmt.Errorf("could not process recover: %w", err)
		}
		d.ops = append(d.ops, op)
		return nil
	case Deactivate:
		if err := d.processDeactivate(op); err != nil {
			return fmt.Errorf("could not process deactivate: %w", err)
		}
		d.ops = append(d.ops, op)
		return nil
	default:
		return fmt.Errorf("unsupported operation type")
	}
}

func (d *DID) processCreate(op Create) error {
	suffixData, delta, err := op.Operation()
	if err != nil {
		return fmt.Errorf("could not get operation data: %w", err)
	}

	d.suffixData = suffixData
	d.setRecoveryCommitment(suffixData.RecoveryCommitment)

	didSuffix, err := suffixData.URI()
	if err != nil {
		return fmt.Errorf("could not get did suffix: %w", err)
	}

	d.document = did.New(
		didSuffix,
		suffixData.RecoveryCommitment,
		d.method,
		false,
	)

	hash, err := delta.Hash()
	if err != nil {
		return fmt.Errorf("could not extract delta hash: %w", err)
	}

	if suffixData.DeltaHash != hash {
		return fmt.Errorf("delta hash did not match suffix data")
	}

	if err := d.processDelta(delta); err != nil {
		return fmt.Errorf("could not process delta")
	}

	return nil
}

func (d *DID) processUpdate(op Update) error {

	_, revealValue, signedData, delta, err := op.Operation()
	if err != nil {
		return fmt.Errorf("could not get operation: %d", err)
	}

	if !did.CheckReveal(revealValue, d.getUpdateCommitment()) {
		return fmt.Errorf("invalid reveal value")
	}

	jws, err := keys.ParseSigned(signedData)
	if err != nil {
		return fmt.Errorf("could not parse signed: %w", err)
	}

	payload := jws.Payload()
	var updatePayload UpdateProtectedPayload
	if err := json.Unmarshal(payload, &updatePayload); err != nil {
		return fmt.Errorf("invalid protected payload: %w", err)
	}

	signingKeyJSON, err := json.Marshal(updatePayload.UpdateKey)
	if err != nil {
		return fmt.Errorf("could not marshal signing key: %w", err)
	}

	calcReveal, err := util.HashReveal(signingKeyJSON)
	if err != nil {
		return fmt.Errorf("could not generate reveal")
	}

	if calcReveal != revealValue {
		return fmt.Errorf("reveals do not match: expected: %s got %s", revealValue, calcReveal)
	}

	signingKey, err := keys.ParseKey(signingKeyJSON)
	if err != nil {
		return fmt.Errorf("could not parse signing key: %w", err)
	}

	payload, err = jws.Verify(signingKey)
	if err != nil {
		return fmt.Errorf("could not verify signed data: %w", err)
	}

	if err := json.Unmarshal(payload, &updatePayload); err != nil {
		return fmt.Errorf("invalid verified payload: %w", err)
	}

	deltaHash, err := delta.Hash()
	if err != nil {
		return fmt.Errorf("could not get delta hash")
	}

	if updatePayload.DeltaHash != deltaHash {
		return fmt.Errorf("delta hash doesn't match: expected %s got %s", updatePayload.DeltaHash, deltaHash)
	}

	if err := d.processDelta(delta); err != nil {
		return fmt.Errorf("could not match delta hash: %w", err)
	}

	return nil
}

func (d *DID) processDeactivate(op Deactivate) error {
	_, revealValue, signedData, err := op.Operation()
	if err != nil {
		return fmt.Errorf("could not get operation: %d", err)
	}

	if !did.CheckReveal(revealValue, d.getUpdateCommitment()) {
		return fmt.Errorf("invalid reveal value")
	}

	jws, err := keys.ParseSigned(signedData)
	if err != nil {
		return fmt.Errorf("could not parse signed: %w", err)
	}

	payload := jws.Payload()
	var deactivatePayload DeactivateProtectedPayload
	if err := json.Unmarshal(payload, &deactivatePayload); err != nil {
		return fmt.Errorf("invalid protected payload: %w", err)
	}

	signingKeyJSON, err := json.Marshal(deactivatePayload.RecoveryKey)
	if err != nil {
		return fmt.Errorf("could not marshal signing key: %w", err)
	}

	calcReveal, err := util.HashReveal(signingKeyJSON)
	if err != nil {
		return fmt.Errorf("could not generate reveal")
	}

	if calcReveal != revealValue {
		return fmt.Errorf("reveals do not match: expected: %s got %s", revealValue, calcReveal)
	}

	signingKey, err := keys.ParseKey(signingKeyJSON)
	if err != nil {
		return fmt.Errorf("could not parse signing key: %w", err)
	}

	payload, err = jws.Verify(signingKey)
	if err != nil {
		return fmt.Errorf("could not verify signed data: %w", err)
	}

	if err := json.Unmarshal(payload, &deactivatePayload); err != nil {
		return fmt.Errorf("invalid verified payload: %w", err)
	}

	d.document = nil
	return nil
}

func (d *DID) processRecover(op Recover) error {
	_, revealValue, delta, signedData, err := op.Operation()
	if err != nil {
		return fmt.Errorf("could not get operation data: %w", err)
	}

	if !did.CheckReveal(revealValue, d.getRecoveryCommitment()) {
		return fmt.Errorf("invalid reveal value")
	}

	jws, err := keys.ParseSigned(signedData)
	if err != nil {
		return fmt.Errorf("could not parse signed: %w", err)
	}

	payload := jws.Payload()
	var recoverPayload RecoverProtectedPayload
	if err := json.Unmarshal(payload, &recoverPayload); err != nil {
		return fmt.Errorf("invalid protected payload: %w", err)
	}

	signingKeyJSON, err := json.Marshal(recoverPayload.RecoveryKey)
	if err != nil {
		return fmt.Errorf("could not marshal signing key: %w", err)
	}

	calcReveal, err := util.HashReveal(signingKeyJSON)
	if err != nil {
		return fmt.Errorf("could not generate reveal")
	}

	if calcReveal != revealValue {
		return fmt.Errorf("reveals do not match: expected: %s got %s", revealValue, calcReveal)
	}

	signingKey, err := keys.ParseKey(signingKeyJSON)
	if err != nil {
		return fmt.Errorf("could not parse signing key: %w", err)
	}

	payload, err = jws.Verify(signingKey)
	if err != nil {
		return fmt.Errorf("could not verify signed data: %w", err)
	}

	if err := json.Unmarshal(payload, &recoverPayload); err != nil {
		return fmt.Errorf("invalid verified payload: %w", err)
	}

	deltaHash, err := delta.Hash()
	if err != nil {
		return fmt.Errorf("could not get delta hash")
	}

	if recoverPayload.DeltaHash != deltaHash {
		return fmt.Errorf("delta hash doesn't match: expected %s got %s", recoverPayload.DeltaHash, deltaHash)
	}
	d.setRecoveryCommitment(recoverPayload.RecoveryCommitment)

	if err := d.processDelta(delta); err != nil {
		return fmt.Errorf("could not match delta hash: %w", err)
	}

	return nil
}

func (d *DID) processDelta(delta did.Delta) error {
	d.setUpdateCommitment(delta.UpdateCommitment)
	return PatchData(d.logger, d.method, delta, d.document)
}

func (d *DID) setRecoveryCommitment(commitment string) {
	d.document.Metadata.Method.RecoveryCommitment = commitment
}

func (d *DID) getRecoveryCommitment() string {
	return d.document.Metadata.Method.RecoveryCommitment
}

func (d *DID) setUpdateCommitment(commitment string) {
	d.document.Metadata.Method.UpdateCommitment = commitment
}

func (d *DID) getUpdateCommitment() string {
	return d.document.Metadata.Method.UpdateCommitment
}

func (d *DID) Suffix() (string, error) {
	firstOp := d.firstOp()
	if firstOp == nil {
		return "", fmt.Errorf("could not get first op")
	}

	suffixData, _, err := d.firstOp().Operation()
	if err != nil {
		return "", fmt.Errorf("could not get first operation: %w", err)
	}

	return suffixData.URI()
}

func (d *DID) LongForm() (string, error) {
	firstOp := d.firstOp()
	if firstOp == nil {
		return "", fmt.Errorf("could not get first op")
	}

	suffixData, delta, err := d.firstOp().Operation()
	if err != nil {
		return "", fmt.Errorf("could not get first operation: %w", err)
	}

	didSuffix, err := suffixData.URI()
	if err != nil {
		return "", fmt.Errorf("could not get suffixData string")
	}

	marshalStruct := struct {
		Delta      did.Delta      `json:"delta"`
		SuffixData did.SuffixData `json:"suffixData"`
	}{
		Delta:      delta,
		SuffixData: suffixData,
	}

	didData, err := json.Marshal(marshalStruct)
	if err != nil {
		return "", fmt.Errorf("failed to marshal DID: %w", err)
	}

	jsonData, err := jcs.Transform(didData)
	if err != nil {
		return "", fmt.Errorf("failed to transform DID: %w", err)
	}

	encodedSuffixData := base64.RawURLEncoding.EncodeToString(jsonData)
	return fmt.Sprintf("did:%s:%s:%s", d.method, didSuffix, encodedSuffixData), nil
}

func (d *DID) URI() (string, error) {
	longForm, err := d.LongForm()
	if err != nil {
		return "", fmt.Errorf("could not get long form: %w", err)
	}

	splitLF := strings.Split(longForm, ":")
	if len(splitLF) < 4 {
		return "", fmt.Errorf("invalid long form")
	}

	return strings.Join(splitLF[:len(splitLF)-2], ":"), nil
}
