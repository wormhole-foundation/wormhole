package common

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/openpgp/armor" //nolint:staticcheck // Package is deprecated but we need it in the codebase still.
	"google.golang.org/protobuf/proto"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
)

const (
	GuardianKeyArmoredBlock    = "WORMHOLE GUARDIAN PRIVATE KEY"
	onlyOwnerReadWritePermMask = 0600
	forbiddenPermMask          = 0077
)

// LoadGuardianKey loads a serialized guardian key from disk.
func LoadGuardianKey(filename string, unsafeDevMode bool) (*ecdsa.PrivateKey, error) {
	return LoadArmoredKey(filename, GuardianKeyArmoredBlock, unsafeDevMode)
}

// LoadArmoredKey loads a serialized key from disk.
func LoadArmoredKey(filename string, blockType string, unsafeDevMode bool) (*ecdsa.PrivateKey, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Validate the permissions on the open handle so we check the same file we
	// read, closing the gap between a separate stat and open.
	if err := ValidatePrivateKeyFilePermissions(f); err != nil {
		return nil, err
	}

	p, err := armor.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read armored file: %w", err)
	}

	if p.Type != blockType {
		return nil, fmt.Errorf("invalid block type: %s", p.Type)
	}

	b, err := SafeRead(p.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var m nodev1.GuardianKey
	err = proto.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize protobuf: %w", err)
	}

	if !unsafeDevMode && m.UnsafeDeterministicKey {
		return nil, errors.New("refusing to use deterministic key in production")
	}

	gk, err := ethcrypto.ToECDSA(m.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize raw key data: %w", err)
	}

	return gk, nil
}

// ValidatePrivateKeyFilePermissions returns an error when f is a regular file that grants any permission to group
// or other users. It takes an open file handle, not a path, so that the permissions it checks belong to the same
// file the caller reads. Non-regular files (for example FIFOs or /dev/fd entries) are not checked.
func ValidatePrivateKeyFilePermissions(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat private key file: %w", err)
	}

	// Only regular files are checked. Non-regular sources (FIFOs, /dev/fd/N from
	// process substitution, etc.) carry no meaningful at-rest permissions and let
	// an operator supply a key without it ever touching disk, so they are allowed.
	if info.Mode().IsRegular() && info.Mode().Perm()&forbiddenPermMask != 0 {
		return fmt.Errorf("private key file %q has insecure permissions %o; expected no group or other permissions", f.Name(), info.Mode().Perm())
	}

	return nil
}

// WriteArmoredKey serializes a key and writes it to disk. The function creates a new file with read/write permissions
// for owner only.
func WriteArmoredKey(key *ecdsa.PrivateKey, description string, filename string, blockType string, unsafe bool) error {
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return errors.New("refusing to override existing key")
	}

	m := &nodev1.GuardianKey{
		Data:                   ethcrypto.FromECDSA(key),
		UnsafeDeterministicKey: unsafe,
	}

	// The private key is a really long-lived piece of data, and we really want to use the stable binary
	// protobuf encoding with field tags to make sure that we can safely evolve it in the future.
	b, err := proto.Marshal(m)
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, onlyOwnerReadWritePermMask)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	headers := map[string]string{
		"PublicKey": ethcrypto.PubkeyToAddress(key.PublicKey).String(),
	}
	if description != "" {
		headers["Description"] = description
	}
	a, err := armor.Encode(f, blockType, headers)
	if err != nil {
		panic(err)
	}
	_, err = a.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	err = a.Close()
	if err != nil {
		return err
	}
	return f.Close()
}
