package sidetree

import (
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

var (
	ErrInvalidMethod = fmt.Errorf("invalid method")
	ErrInvalidCAS    = fmt.Errorf("invalid cas")
	ErrEmptyURI      = fmt.Errorf("index anchor URI is empty")
)

func Processor(op operations.Anchor, options ...SideTreeOption) (*OperationsProcessor, error) {
	if op.CID() == "" {
		return nil, ErrEmptyURI
	}

	d := &OperationsProcessor{
		op:               op,
		coreIndexFileURI: op.CID(),
	}

	for _, option := range options {
		option(d)
	}

	if d.method == "" {
		return nil, ErrInvalidMethod
	}

	if d.cas == nil {
		return nil, ErrInvalidCAS
	}

	if d.filterDIDs == nil {
		d.filterDIDs = []string{}
	}

	return d, nil
}

type OperationsProcessor struct {
	cas        CAS
	filterDIDs []string
	method     string
	op         operations.Anchor

	coreIndexFileURI string
	coreIndexFile    *CoreIndexFile

	coreProofFileURI string
	coreProofFile    *CoreProofFile

	provisionalIndexFileURI string
	provisionalIndexFile    *ProvisionalIndexFile

	provisionalProofFileURI string
	provisionalProofFile    *ProvisionalProofFile

	// Version 1 only has a single Chunk file No need for Array here yet
	chunkFileURI string
	chunkFile    *ChunkFile

	createOps     map[string]operations.CreateInterface
	updateOps     map[string]operations.UpdateInterface
	deactivateOps map[string]operations.DeactivateInterface
	recoverOps    map[string]operations.RecoverInterface

	createMappingArray   []string
	recoveryMappingArray []string
	updateMappingArray   []string

	//TODO These Don't actually do Anything Yet
	baseFeeFn   BaseFeeAlgorithm
	perOpFeeFn  PerOperationFee
	valueLockFn ValueLocking

	baseFee int
}

type ProcessedOperations struct {
	AnchorString   string
	AnchorSequence string
	Error          error
	CreateOps      map[string]operations.CreateInterface
	UpdateOps      map[string]operations.UpdateInterface
	DeactivateOps  map[string]operations.DeactivateInterface
	RecoverOps     map[string]operations.RecoverInterface
}

func (b *OperationsProcessor) Anchor() string {
	return string(b.op.Anchor)
}

func (b *OperationsProcessor) SystemAnchor() string {
	return string(b.op.Sequence)
}

func (d *OperationsProcessor) Process() ProcessedOperations {

	d.createMappingArray = []string{}
	d.recoveryMappingArray = []string{}
	d.updateMappingArray = []string{}

	d.createOps = map[string]operations.CreateInterface{}
	d.updateOps = map[string]operations.UpdateInterface{}
	d.deactivateOps = map[string]operations.DeactivateInterface{}
	d.recoverOps = map[string]operations.RecoverInterface{}

	ops := ProcessedOperations{
		Error:          nil,
		AnchorString:   d.Anchor(),
		AnchorSequence: d.SystemAnchor(),
	}

	if err := d.fetchCoreIndexFile(); err != nil {
		ops.Error = err // already classified (unavailable vs malformed)
		return ops
	}

	// Per-anchor operation-count enforcement (Sidetree protocol rule). This runs
	// UNCONDITIONALLY — it does not depend on any configured fee/value-lock
	// callback — and rejects the entire batch the way a spec-compliant ION node
	// does. Rejection is permanent and never retried, so it routes through
	// classifyMalformed. Checked here, right after the core index file is
	// available (so writerLockId is known) and before any file downloads.
	declaredOps := d.op.Operations()
	if err := d.checkOperationLimit(declaredOps); err != nil {
		ops.Error = classifyMalformed(err)
		return ops
	}

	// https://identity.foundation/sidetree/spec/#base-fee-variable
	if d.baseFeeFn != nil {
		d.baseFee = d.baseFeeFn(d.op.Operations(), string(d.op.Sequence))
	}

	// https://identity.foundation/sidetree/spec/#per-operation-fee
	if d.perOpFeeFn != nil {
		if !d.perOpFeeFn(d.baseFee, d.op.Operations(), string(d.op.Sequence)) {
			ops.Error = classifyMalformed(fmt.Errorf("per op fee is not valid"))
			return ops
		}
	}

	// https://identity.foundation/sidetree/spec/#value-locking
	// The valueLockFn is the integration seam the Bitcoin layer (ion-node)
	// implements: it resolves the writerLockId to a ValueTimeLock, looks up the
	// block's normalized fee, identifies the transaction writer, and decides via
	// the ported policy VerifyLockAmount (see valuelock.go). It returns true iff
	// the anchor is permitted.
	//
	// NOTE: the opCount passed here is the writer-DECLARED anchor-string count
	// (the "paid" count). It is an upper bound and may exceed the operations
	// actually anchored (ION permits paying/locking for more than are used), so a
	// lock verifier must size the required lock against this declared count — not
	// against the post-parse actual count.
	if d.valueLockFn != nil {
		if !d.valueLockFn(d.coreIndexFile.WriterLockId, d.baseFee, d.op.Operations(), string(d.op.Sequence)) {
			ops.Error = classifyMalformed(fmt.Errorf("value lock is not valid"))
			return ops
		}
	}

	if err := d.coreIndexFile.Process(); err != nil {
		ops.Error = classifyMalformed(err)
		return ops
	}

	if d.coreProofFileURI != "" {

		if err := d.fetchCoreProofFile(); err != nil {
			ops.Error = err
			return ops
		}

		if err := d.coreProofFile.Process(); err != nil {
			ops.Error = classifyMalformed(err)
			return ops
		}
	}

	if d.provisionalIndexFileURI != "" {

		if err := d.fetchProvisionalIndexFile(); err != nil {
			ops.Error = err
			return ops
		}

		if err := d.provisionalIndexFile.Process(); err != nil {
			ops.Error = classifyMalformed(err)
			return ops
		}

		if len(d.provisionalIndexFile.Operations.Update) > 0 {

			if err := d.fetchProvisionalProofFile(); err != nil {
				ops.Error = err
				return ops
			}

			if err := d.provisionalProofFile.Process(); err != nil {
				ops.Error = classifyMalformed(err)
				return ops
			}
		}

		if len(d.provisionalIndexFile.Chunks) > 0 {
			if err := d.fetchChunkFile(); err != nil {
				ops.Error = err
				return ops
			}

			if err := d.chunkFile.Process(); err != nil {
				ops.Error = classifyMalformed(err)
				return ops
			}
		}
	}

	// A writer must not understate the anchor-string operation count to slip
	// past checkOperationLimit above while packing more operations into the
	// anchored files. Now that every file is parsed, reject if the anchored
	// operation count exceeds what the anchor string declared.
	if anchored := d.anchoredOperationCount(); anchored > declaredOps {
		ops.Error = classifyMalformed(fmt.Errorf("%w: declared %d, anchored %d", ErrOperationCountMismatch, declaredOps, anchored))
		return ops
	}

	return ProcessedOperations{
		Error:          nil,
		AnchorString:   d.Anchor(),
		AnchorSequence: d.SystemAnchor(),
		CreateOps:      d.CreateOps(),
		RecoverOps:     d.RecoverOps(),
		UpdateOps:      d.UpdateOps(),
		DeactivateOps:  d.DeactivateOps(),
	}
}

// checkOperationLimit enforces the Sidetree per-anchor operation-count rules
// against opCount (the anchor-string declared count), unconditionally:
//
//   - opCount > MaxOperationsPerBatch: hard ceiling; no value lock can exceed it.
//   - opCount > MaxNumberOfOperationsForNoValueTimeLock with an empty
//     writerLockId: too many operations and no value-time-lock.
//   - opCount > MaxNumberOfOperationsForNoValueTimeLock with a writerLockId but
//     no value-lock verifier installed: the lock cannot be verified yet (no
//     on-chain LockResolver / normalized fee; see #33/#55), so default-reject.
//     When a verifier IS installed, this defers to it (the valueLockFn block in
//     Process decides), so the seam stays meaningful.
//
// The returned error is unwrapped; the caller wraps it with classifyMalformed.
func (d *OperationsProcessor) checkOperationLimit(opCount int) error {
	// A valid anchor declares at least one operation. AnchorString.Operations()
	// collapses a non-numeric count to 0 (and lets a negative count through), so
	// guard the count's validity here — otherwise a malformed "abc.cid"/"0.cid"
	// would be waved past the quota gate as "under the free limit".
	if opCount < 1 {
		return fmt.Errorf("%w: %d", ErrInvalidOperationCount, opCount)
	}
	if opCount > MaxOperationsPerBatch {
		return fmt.Errorf("%w: %d > %d", ErrTooManyOperations, opCount, MaxOperationsPerBatch)
	}
	if opCount <= MaxNumberOfOperationsForNoValueTimeLock {
		return nil
	}
	if d.coreIndexFile.WriterLockId == "" {
		return fmt.Errorf("%w: %d > %d", ErrOperationLimitExceeded, opCount, MaxNumberOfOperationsForNoValueTimeLock)
	}
	if d.valueLockFn == nil {
		return fmt.Errorf("%w: %d operations, writerLockId %q", ErrUnverifiableValueLock, opCount, d.coreIndexFile.WriterLockId)
	}
	return nil
}

// anchoredOperationCount returns the number of operations actually present in
// the anchored files: core index Create/Recover/Deactivate plus provisional
// Update. (Provisional index may be absent.)
func (d *OperationsProcessor) anchoredOperationCount() int {
	n := 0
	if d.coreIndexFile != nil {
		n += len(d.coreIndexFile.Operations.Create)
		n += len(d.coreIndexFile.Operations.Recover)
		n += len(d.coreIndexFile.Operations.Deactivate)
	}
	if d.provisionalIndexFile != nil {
		n += len(d.provisionalIndexFile.Operations.Update)
	}
	return n
}

func (d *OperationsProcessor) CreateOps() map[string]operations.CreateInterface {
	if len(d.filterDIDs) == 0 {
		return d.createOps
	}

	ops := map[string]operations.CreateInterface{}
	for _, did := range d.filterDIDs {
		if _, ok := d.createOps[did]; ok {
			ops[did] = d.createOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) RecoverOps() map[string]operations.RecoverInterface {
	if len(d.filterDIDs) == 0 {
		return d.recoverOps
	}

	ops := map[string]operations.RecoverInterface{}
	for _, did := range d.filterDIDs {
		if _, ok := d.recoverOps[did]; ok {
			ops[did] = d.recoverOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) UpdateOps() map[string]operations.UpdateInterface {
	if len(d.filterDIDs) == 0 {
		return d.updateOps
	}

	ops := map[string]operations.UpdateInterface{}
	for _, did := range d.filterDIDs {
		if _, ok := d.updateOps[did]; ok {
			ops[did] = d.updateOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) DeactivateOps() map[string]operations.DeactivateInterface {
	if len(d.filterDIDs) == 0 {
		return d.deactivateOps
	}

	ops := map[string]operations.DeactivateInterface{}
	for _, did := range d.filterDIDs {
		if _, ok := d.deactivateOps[did]; ok {
			ops[did] = d.deactivateOps[did]
		}
	}

	return ops
}

// checkFileSize defensively enforces the protocol per-file cap on the
// (decompressed) bytes a CAS returned, in case a CAS does not honor the
// maxSizeInBytes contract on Get. The legal decompressed size is the per-file
// cap times MaxMemoryDecompressionFactor; anything larger is a permanently
// invalid file (ErrMalformed). name identifies the file for the error message.
func checkFileSize(name string, data []byte, maxSizeInBytes int) error {
	limit := maxSizeInBytes * MaxMemoryDecompressionFactor
	if len(data) > limit {
		return classifyMalformed(fmt.Errorf("%w: %s is %d bytes (limit %d)", ErrFileTooLarge, name, len(data), limit))
	}
	return nil
}

func (d *OperationsProcessor) fetchCoreIndexFile() error {

	coreData, err := d.cas.Get(d.coreIndexFileURI, MaxCoreIndexFileSizeInBytes)
	if err != nil {
		return fmt.Errorf("failed to get core index file: %w", classifyFetch(err))
	}
	if err := checkFileSize("core index file", coreData, MaxCoreIndexFileSizeInBytes); err != nil {
		return err
	}

	d.coreIndexFile, err = NewCoreIndexFile(d, coreData)
	if err != nil {
		return fmt.Errorf("failed to create core index file: %w", classifyMalformed(err))
	}

	return nil
}

func (d *OperationsProcessor) fetchCoreProofFile() error {

	coreProofData, err := d.cas.Get(d.coreProofFileURI, MaxProofFileSizeInBytes)
	if err != nil {
		return fmt.Errorf("failed to get core proof file: %w", classifyFetch(err))
	}
	if err := checkFileSize("core proof file", coreProofData, MaxProofFileSizeInBytes); err != nil {
		return err
	}

	d.coreProofFile, err = NewCoreProofFile(d, coreProofData)
	if err != nil {
		return fmt.Errorf("failed to create core proof file: %w", classifyMalformed(err))
	}

	return nil
}

func (d *OperationsProcessor) fetchProvisionalIndexFile() error {

	provisionalData, err := d.cas.Get(d.provisionalIndexFileURI, MaxProvisionalIndexFileSizeInBytes)
	if err != nil {
		return fmt.Errorf("failed to get provisional index file: %w", classifyFetch(err))
	}
	if err := checkFileSize("provisional index file", provisionalData, MaxProvisionalIndexFileSizeInBytes); err != nil {
		return err
	}

	d.provisionalIndexFile, err = NewProvisionalIndexFile(d, provisionalData)
	if err != nil {
		return fmt.Errorf("failed to create provisional index file: %w", classifyMalformed(err))
	}

	return nil
}

func (d *OperationsProcessor) fetchProvisionalProofFile() error {

	provisionalProofData, err := d.cas.Get(d.provisionalProofFileURI, MaxProofFileSizeInBytes)
	if err != nil {
		return fmt.Errorf("failed to get provisional proof file: %w", classifyFetch(err))
	}
	if err := checkFileSize("provisional proof file", provisionalProofData, MaxProofFileSizeInBytes); err != nil {
		return err
	}

	d.provisionalProofFile, err = NewProvisionalProofFile(d, provisionalProofData)
	if err != nil {
		return fmt.Errorf("failed to create provisional proof file: %w", classifyMalformed(err))
	}

	return nil
}

func (d *OperationsProcessor) fetchChunkFile() error {

	chunkData, err := d.cas.Get(d.chunkFileURI, MaxChunkFileSizeInBytes)
	if err != nil {
		return fmt.Errorf("failed to get chunk file: %w", classifyFetch(err))
	}
	if err := checkFileSize("chunk file", chunkData, MaxChunkFileSizeInBytes); err != nil {
		return err
	}

	d.chunkFile, err = NewChunkFile(chunkData,
		WithMappingArrays(d.createMappingArray, d.recoveryMappingArray, d.updateMappingArray),
		WithOperations(d.createOps, d.recoverOps, d.updateOps),
	)
	if err != nil {
		return fmt.Errorf("failed to create chunk file: %w", classifyMalformed(err))
	}

	return nil
}

func (p *OperationsProcessor) populateDeltaMappingArray() error {

	provisionalIndex := p.provisionalIndexFile
	if provisionalIndex == nil {
		return fmt.Errorf("provisional index file is nil")
	}

	for _, op := range p.coreIndexFile.Operations.Create {
		uri, _ := op.SuffixData.URI()

		createOp := operations.CreateOperation(
			op.SuffixData,
		)

		p.createOps[uri] = createOp
		p.createMappingArray = append(p.createMappingArray, uri)
	}

	for _, op := range p.coreIndexFile.Operations.Recover {
		p.recoveryMappingArray = append(p.recoveryMappingArray, op.DIDSuffix)
	}

	for _, op := range provisionalIndex.Operations.Update {
		p.updateMappingArray = append(p.updateMappingArray, op.DIDSuffix)
	}

	return nil
}
