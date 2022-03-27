package sidetree

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	mh "github.com/multiformats/go-multihash"
)

type Config interface {
	Logger() Logger
	Storage() Storage
	Prefix() string
}

type SideTree struct {
	log    Logger
	store  Storage
	prefix string
}

func New(config Config) (*SideTree, error) {

	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	if config.Logger() == nil {
		return nil, fmt.Errorf("logger is nil")
	}

	if config.Storage() == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	if config.Prefix() == "" {
		return nil, fmt.Errorf("prefix is empty")
	}

	return &SideTree{
		log:    config.Logger(),
		store:  config.Storage(),
		prefix: config.Prefix(),
	}, nil
}

func (s *SideTree) CheckAnchorSignature(b []byte) bool {

	if len(b) < 6 {
		return false
	}

	if b[0] != 0x6a {
		return false
	}

	prefix := s.prefix + ":"
	pushBytes := int(b[1])

	if len(b) < pushBytes+2 {
		return false
	}

	endIndex := 2 + len(prefix)

	return string(b[2:endIndex]) == prefix
}

// This is particular to ION and is not a general purpose function
// TODO: make this a general purpose function
func (d *SideTree) ParseAnchor(b []byte) string {
	if !d.CheckAnchorSignature(b) {
		return ""
	}

	prefix := d.prefix + ":"

	pushBytes := int(b[1])

	startIndex := 2 + len(prefix)
	endIndex := 2 + pushBytes

	return string(b[startIndex:endIndex])
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
