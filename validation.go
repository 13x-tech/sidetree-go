package sidetree

import (
	"fmt"

	"github.com/gowebpki/jcs"
)

// Per-field caps enforced at file-parse time, matching what the Sidetree v1
// reference implementation actually checks when parsing the index, proof, and
// chunk files. A field that exceeds its cap makes the whole anchored batch
// permanently invalid. The helpers return RAW sentinel-wrapped errors so each
// call site can classify them with the surrounding convention (the file
// Process() methods return raw errors that Process() wraps with classifyMalformed).
//
// Deliberately NOT enforced here (to avoid diverging from the reference by
// rejecting anchors it accepts):
//   - the anchor's own core-index CID is not length-checked (the reference only
//     applies maxCasUriLength to EMBEDDED URIs);
//   - reveal values are not length-checked (the reference validates them as
//     supported-algorithm multihashes; ion-sdk-go's did.CheckReveal enforces the
//     SHA-256 algorithm — hashAlgorithmsInMultihashCode=[18] — at apply time).

// checkCASURI rejects an embedded CAS/IPFS URI longer than MaxCASURILength.
// Empty URIs (optional fields) pass.
func checkCASURI(name, uri string) error {
	if len(uri) > MaxCASURILength {
		return fmt.Errorf("%w: %s is %d bytes (max %d)", ErrCASURITooLong, name, len(uri), MaxCASURILength)
	}
	return nil
}

// checkWriterLockID rejects a writerLockId longer than MaxWriterLockIDInBytes.
func checkWriterLockID(lockID string) error {
	if len(lockID) > MaxWriterLockIDInBytes {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrWriterLockIDTooLong, len(lockID), MaxWriterLockIDInBytes)
	}
	return nil
}

// checkDeltaSize rejects an operation delta whose canonicalized (JCS) size
// exceeds MaxDeltaSizeInBytes. It measures the RAW on-wire delta bytes (not a
// re-marshaled struct), matching the reference — which canonicalizes the parsed
// delta object, unknown fields included — so a writer cannot understate the size
// by hiding bytes in fields our struct would drop.
func checkDeltaSize(rawDelta []byte) error {
	canonical, err := jcs.Transform(rawDelta)
	if err != nil {
		return fmt.Errorf("failed to canonicalize delta for size check: %w", err)
	}
	if len(canonical) > MaxDeltaSizeInBytes {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrDeltaTooLarge, len(canonical), MaxDeltaSizeInBytes)
	}
	return nil
}
