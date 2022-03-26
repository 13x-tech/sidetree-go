package sidetree

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	mh "github.com/multiformats/go-multihash"
)

type SideTree interface {
	Index() error
	Process() error
}

type Config struct {
	Logger              Logger
	ChainService        Service
	Storage             Storage
	StartBlock          int64
	MaxConcurrentBlocks int
	Prefix              string
}

type SideTreeIndexer struct {
	log        Logger
	config     Config
	srv        Service
	bestBlock  int64
	indexStore Indexer

	wg         sync.WaitGroup
	blockGuard chan struct{}
}

func New(config Config) SideTree {

	cas, err := config.Storage.CAS()
	if err != nil {
		panic(fmt.Errorf("failed to get CAS: %w", err))
	}

	indexer, err := config.Storage.Indexer()
	if err != nil {
		panic(fmt.Errorf("failed to get DIDOps: %w", err))
	}

	//TODO this sucks, find a cleaner way
	if err := cas.Start(); err != nil {
		panic(fmt.Errorf("failed to start CAS: %w", err))
	}

	return &SideTreeIndexer{
		log:        config.Logger,
		indexStore: indexer,
		srv:        config.ChainService,
		config:     config,
	}
}

func (d *SideTreeIndexer) Index() error {
	d.log.Info("Starting to index...")
	d.log.Info("Will print status every 100 blocks")

	startTime := time.Now()

	if !d.srv.IsCurrent() {
		if err := d.srv.WaitForSync(); err != nil {
			return fmt.Errorf("failed to wait for sync: %w", err)
		}
	}

	d.blockGuard = make(chan struct{}, d.config.MaxConcurrentBlocks)

	bb, err := d.srv.BestBlock()
	if err != nil {
		return fmt.Errorf("indexer failed to get best block: %w", err)
	}

	d.bestBlock = bb.Height()

	count := 0
	for i := d.config.StartBlock; i <= bb.Height(); i++ {
		_, err := d.indexStore.GetBlockOps(i)
		if err != nil {
			count++
			d.blockGuard <- struct{}{}
			d.wg.Add(1)
			go d.processBlock(i)
		}
	}

	d.wg.Wait()
	d.log.Infof("Indexed %d blocks in %s\n", count, time.Since(startTime))
	return nil
}

func (d *SideTreeIndexer) processBlock(blockheigt int64) error {
	if blockheigt%100 == 0 {
		d.log.Infof("Processing block %d...\n", blockheigt)
	}

	defer d.wg.Done()
	defer func() {
		<-d.blockGuard
	}()

	var ops []SideTreeOp

	hash, err := d.srv.GetBlockHash(blockheigt)
	if err != nil {
		return err
	}

	block, err := d.srv.GetBlock(hash)
	if err != nil {
		return err
	}

	for i, tx := range block.Transactions() {
		for n, txout := range tx.TxOut() {
			if d.checkSignature(txout.PkScript()) {
				opCount, CID := d.parseTxOut(txout.PkScript())
				if opCount != 0 && CID != "" {
					ops = append(ops, SideTreeOp{
						BlockHash:    block.Hash().String(),
						Height:       blockheigt,
						BlockTxIndex: i,
						TxOutpoint:   fmt.Sprintf("%s:%d", tx.Hash().String(), n),
						Ops:          opCount,
						CID:          CID,
						Processed:    false,
					})
				}
			}
		}
	}

	if err := d.indexStore.PutBlockOps(blockheigt, ops); err != nil {
		return err
	}

	return nil
}

func (d *SideTreeIndexer) checkSignature(b []byte) bool {

	if len(b) < 6 {
		return false
	}

	if b[0] != 0x6a {
		return false
	}
	pushBytes := int(b[1])

	if len(b) < pushBytes+2 {
		return false
	}

	return string(b[2:6]) == d.config.Prefix
}

func (d *SideTreeIndexer) Process() error {

	totalProcessed := 0

	for i := d.config.StartBlock; i <= d.bestBlock; i++ {
		if i%100 == 0 {
			d.log.Infof("Processing operations for block %d - %d ops processed so far...\n", i, totalProcessed)
		}

		ops, err := d.indexStore.GetBlockOps(i)
		if err != nil {
			return fmt.Errorf("failed to get block ops for height %d: %w", i, err)
		}

		totalOps := 0
		for _, op := range ops {
			totalOps = totalOps + op.Ops
		}
		totalProcessed = totalProcessed + totalOps

		if err := d.processSideTreeOperations(ops); err != nil {
			d.log.Error(err)
		}
	}

	return nil
}

func (d *SideTreeIndexer) processSideTreeOperations(ops []SideTreeOp) error {
	for _, op := range ops {
		processor, err := NewOperationsProcessor(op.Ops, op.CID, d.log, d.config.Storage)
		if err != nil {
			return fmt.Errorf("failed to create operations processor: %w", err)
		}

		if err := processor.Process(); err != nil {
			return fmt.Errorf("failed to process operations: %w", err)
		}
	}
	return nil
}

func (d *SideTreeIndexer) parseTxOut(b []byte) (operations int, address string) {
	if !d.checkSignature(b) {
		return 0, ""
	}

	//TODO Clean this up. It's a bit of a hack.
	//Measure start + end bytes using options for did method
	pushBytes := int(b[1])
	parts := strings.Split(string(b[6:2+pushBytes]), ".")
	if len(parts) != 2 {
		return 0, ""
	}
	operations, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, ""
	}
	return operations, parts[1]
}

func checkReveal(reveal string, commitment string) bool {
	rawReveal, err := base64.RawURLEncoding.DecodeString(reveal)
	if err != nil {
		return false
	}

	decoded, err := mh.Decode(rawReveal)
	if err != nil {
		return false
	}

	h256 := sha256.Sum256(decoded.Digest)
	revealHashed, err := mh.Encode(h256[:], mh.SHA2_256)
	if err != nil {
		return false
	}

	b64 := base64.RawURLEncoding.EncodeToString(revealHashed)

	return commitment == string(b64)
}

func hashReveal(data []byte) (string, error) {
	hashedReveal := sha256.Sum256(data)
	revealMH, err := mh.Encode(hashedReveal[:], mh.SHA2_256)
	if err != nil {
		return "", fmt.Errorf("failed to hash revieal: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(revealMH), nil
}
