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
		CoreIndexFileURI: op.CID(),
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

	CoreIndexFileURI string
	CoreIndexFile    *CoreIndexFile

	CoreProofFileURI string
	CoreProofFile    *CoreProofFile

	ProvisionalIndexFileURI string
	ProvisionalIndexFile    *ProvisionalIndexFile

	ProvisionalProofFileURI string
	ProvisionalProofFile    *ProvisionalProofFile

	// Version 1 only has a single Chunk file No need for Array here yet
	ChunkFileURI string
	ChunkFile    *ChunkFile

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

	d.createMappingArray = []string{}
	d.recoveryMappingArray = []string{}
	d.updateMappingArray = []string{}

	if err := d.fetchCoreIndexFile(); err != nil {
		//TODO Define Errors
		ops.Error = err
		return ops
	}

	// https://identity.foundation/sidetree/spec/#base-fee-variable
	if d.baseFeeFn != nil {
		d.baseFee = d.baseFeeFn(d.op.Operations(), string(d.op.Sequence))
	}

	// https://identity.foundation/sidetree/spec/#per-operation-fee
	if d.perOpFeeFn != nil {
		if !d.perOpFeeFn(d.baseFee, d.op.Operations(), string(d.op.Sequence)) {
			ops.Error = fmt.Errorf("per op fee is not valid")
			return ops
		}
	}

	// https://identity.foundation/sidetree/spec/#value-locking
	if d.valueLockFn != nil {
		if !d.valueLockFn(d.CoreIndexFile.WriterLockId, d.op.Operations(), d.baseFee, string(d.op.Sequence)) {
			ops.Error = fmt.Errorf("value lock is not valid")
			return ops
		}
	}

	if err := d.CoreIndexFile.Process(); err != nil {
		ops.Error = err
		return ops
	}

	if d.CoreProofFileURI != "" {

		if err := d.fetchCoreProofFile(); err != nil {
			ops.Error = err
			return ops
		}

		if err := d.CoreProofFile.Process(); err != nil {
			ops.Error = err
			return ops
		}
	}

	if d.ProvisionalIndexFileURI != "" {

		if err := d.fetchProvisionalIndexFile(); err != nil {
			ops.Error = err
			return ops
		}

		if err := d.ProvisionalIndexFile.Process(); err != nil {
			ops.Error = err
			return ops
		}

		if len(d.ProvisionalIndexFile.Operations.Update) > 0 {

			if err := d.fetchProvisionalProofFile(); err != nil {
				ops.Error = err
				return ops
			}

			if err := d.ProvisionalProofFile.Process(); err != nil {
				ops.Error = err
				return ops
			}
		}

		if len(d.ProvisionalIndexFile.Chunks) > 0 {
			if err := d.fetchChunkFile(); err != nil {
				ops.Error = err
				return ops
			}

			if err := d.ChunkFile.Process(); err != nil {
				ops.Error = err
				return ops
			}
		}
	}

	//Check for duplicate dids file invalid if duplicates exist
	if d.hasDuplicateDIDs() {
		ops.Error = fmt.Errorf("duplicate dids found")
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

func (d *OperationsProcessor) hasDuplicateDIDs() bool {
	dids := make(map[string]struct{})
	for _, id := range d.createMappingArray {
		if _, ok := dids[id]; ok {
			return true
		}
		dids[id] = struct{}{}
	}
	for _, id := range d.updateMappingArray {
		if _, ok := dids[id]; ok {
			return true
		}
		dids[id] = struct{}{}
	}
	for _, id := range d.recoveryMappingArray {
		if _, ok := dids[id]; ok {
			return true
		}
		dids[id] = struct{}{}
	}

	for id, _ := range d.deactivateOps {
		if _, ok := dids[id]; ok {
			return true
		}
		dids[id] = struct{}{}
	}

	return false
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

func (d *OperationsProcessor) fetchCoreIndexFile() error {

	if d.CoreIndexFileURI == "" {
		return fmt.Errorf("core index file URI is empty")
	}
	coreData, err := d.cas.Get(d.CoreIndexFileURI)
	if err != nil {
		return fmt.Errorf("failed to get core index file: %w", err)
	}

	d.CoreIndexFile, err = NewCoreIndexFile(d, coreData)
	if err != nil {
		return fmt.Errorf("failed to create core index file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchCoreProofFile() error {

	if d.CoreProofFileURI == "" {
		return fmt.Errorf("core proof file URI is empty")
	}

	coreProofData, err := d.cas.Get(d.CoreProofFileURI)
	if err != nil {
		return fmt.Errorf("failed to get core proof file: %w", err)
	}

	d.CoreProofFile, err = NewCoreProofFile(d, coreProofData)
	if err != nil {
		return fmt.Errorf("failed to create core proof file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchProvisionalIndexFile() error {

	if d.ProvisionalIndexFileURI == "" {
		return fmt.Errorf("no provisional index file URI")
	}

	provisionalData, err := d.cas.Get(d.ProvisionalIndexFileURI)
	if err != nil {
		return fmt.Errorf("failed to get provisional index file: %w", err)
	}

	d.ProvisionalIndexFile, err = NewProvisionalIndexFile(d, provisionalData)
	if err != nil {
		return fmt.Errorf("failed to create provisional index file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchProvisionalProofFile() error {

	if d.ProvisionalProofFileURI == "" {
		return fmt.Errorf("no provisional proof file URI")
	}

	provisionalProofData, err := d.cas.Get(d.ProvisionalProofFileURI)
	if err != nil {
		return fmt.Errorf("failed to get provisional proof file: %w", err)
	}

	d.ProvisionalProofFile, err = NewProvisionalProofFile(d, provisionalProofData)
	if err != nil {
		return fmt.Errorf("failed to create provisional proof file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchChunkFile() error {
	if d.ChunkFileURI == "" {
		return fmt.Errorf("no chunk file URI")
	}

	chunkData, err := d.cas.Get(d.ChunkFileURI)
	if err != nil {
		return fmt.Errorf("failed to get chunk file: %w", err)
	}

	d.ChunkFile, err = NewChunkFile(chunkData,
		WithMappingArrays(d.createMappingArray, d.recoveryMappingArray, d.updateMappingArray),
		WithOperations(d.createOps, d.recoverOps, d.updateOps),
	)
	if err != nil {
		return fmt.Errorf("failed to create chunk file: %w", err)
	}

	return nil
}

func (p *OperationsProcessor) populateDeltaMappingArray() error {
	coreIndex := p.CoreIndexFile
	if coreIndex == nil {
		return fmt.Errorf("core index file is nil")
	}

	provisionalIndex := p.ProvisionalIndexFile
	if provisionalIndex == nil {
		return fmt.Errorf("provisional index file is nil")
	}

	for _, op := range coreIndex.Operations.Create {
		uri, err := op.SuffixData.URI()
		if err != nil {
			return fmt.Errorf("failed to get uri from create operation: %w", err)
		}

		createOp := operations.CreateOperation(
			op.SuffixData,
		)

		p.createOps[uri] = createOp
		p.createMappingArray = append(p.createMappingArray, uri)
	}

	for _, op := range coreIndex.Operations.Recover {
		p.recoveryMappingArray = append(p.recoveryMappingArray, op.DIDSuffix)
	}

	for _, op := range provisionalIndex.Operations.Update {
		p.updateMappingArray = append(p.updateMappingArray, op.DIDSuffix)
	}

	return nil
}
