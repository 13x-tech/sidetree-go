package sidetree

import (
	"errors"
	"fmt"
)

var (
	ErrURINotFound        = fmt.Errorf("URI not found")
	ErrDuplicateOperation = fmt.Errorf("duplicate operation")
	ErrNoCoreProof        = fmt.Errorf("core proof uri is empty")

	// ErrContentUnavailable marks a Sidetree-file fetch that failed because the
	// CAS could not return the content (IPFS timeout, not-found, peer
	// unreachable). The content may be published or become reachable later, so
	// the anchored operation is "not-yet-applied" rather than invalid — callers
	// should RECORD the anchor and RETRY. This is the failure mode late
	// publishing depends on (an anchor whose content surfaces later splices in
	// at its original txnum and can retroactively invalidate later operations).
	ErrContentUnavailable = errors.New("content unavailable")

	// ErrMalformed marks content that WAS retrieved but is structurally or
	// semantically invalid per the Sidetree spec (unparseable file, count
	// mismatch, duplicate operation in a batch, ...). Retrying cannot help —
	// IPFS content is immutable by CID — so callers should PERMANENTLY SKIP it.
	ErrMalformed = errors.New("malformed content")
)

// classifyFetch tags a CAS Get failure with its retryability class. A fetch
// failure is content-unavailable (retryable) by default: the content may
// publish or the CAS may reconnect later. A CAS that can prove the bytes are
// present-but-corrupt (e.g. gzip decompression failed) may itself wrap
// ErrMalformed, in which case the permanent-skip classification is preserved.
func classifyFetch(err error) error {
	if errors.Is(err, ErrMalformed) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrContentUnavailable, err)
}

// classifyMalformed tags content that was retrieved but is invalid per spec.
// The original error chain is preserved (double %w) so callers can still match
// the specific sentinel (ErrNoCoreProof, ErrDuplicateOperation, ...) alongside
// ErrMalformed. Already-classified errors pass through unchanged.
func classifyMalformed(err error) error {
	if errors.Is(err, ErrMalformed) || errors.Is(err, ErrContentUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrMalformed, err)
}
