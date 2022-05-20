package guardiand

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/certusone/wormhole/node/pkg/common"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/openpgp/armor" //nolint
	"google.golang.org/protobuf/proto"

	"github.com/certusone/wormhole/node/pkg/devnet"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
)

var keyDescription *string

const (
	GuardianKeyArmoredBlock = "WORMHOLE GUARDIAN PRIVATE KEY"
)

func init() {
	keyDescription = KeygenCmd.Flags().String("desc", "", "Human-readable key description (optional)")
}

var KeygenCmd = &cobra.Command{
	Use:   "keygen [KEYFILE]",
	Short: "Create guardian key at the specified path",
	Run:   runKeygen,
	Args:  cobra.ExactArgs(1),
}

func runKeygen(cmd *cobra.Command, args []string) {
	common.LockMemory()
	common.SetRestrictiveUmask()

	log.Print("Creating new key at ", args[0])

	gk, err := ecdsa.GenerateKey(ethcrypto.S256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate key: %v", err)
	}

	err = writeGuardianKey(gk, *keyDescription, args[0], false)
	if err != nil {
		log.Fatalf("failed to write key: %v", err)
	}
}

// loadGuardianKey loads a serialized guardian key from disk.
func loadGuardianKey(filename string) (*ecdsa.PrivateKey, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	p, err := armor.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read armored file: %w", err)
	}

	if p.Type != GuardianKeyArmoredBlock {
		return nil, fmt.Errorf("invalid block type: %s", p.Type)
	}

	b, err := ioutil.ReadAll(p.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var m nodev1.GuardianKey
	err = proto.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize protobuf: %w", err)
	}

	if !*unsafeDevMode && m.UnsafeDeterministicKey {
		return nil, errors.New("refusing to use deterministic key in production")
	}

	gk, err := ethcrypto.ToECDSA(m.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize raw key data: %w", err)
	}

	return gk, nil
}

// writeGuardianKey serializes a guardian key and writes it to disk.
func writeGuardianKey(key *ecdsa.PrivateKey, description string, filename string, unsafe bool) error {
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

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	headers := map[string]string{
		"PublicKey": ethcrypto.PubkeyToAddress(key.PublicKey).String(),
	}
	if description != "" {
		headers["Description"] = description
	}
	a, err := armor.Encode(f, GuardianKeyArmoredBlock, headers)
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

// generateDevnetGuardianKey returns a deterministic testnet key.
func generateDevnetGuardianKey() (*ecdsa.PrivateKey, error) {
	// Figure out our devnet index
	idx, err := devnet.GetDevnetIndex()
	if err != nil {
		return nil, err
	}

	// Generate guardian key
	return devnet.InsecureDeterministicEcdsaKeyByIndex(ethcrypto.S256(), uint64(idx)), nil
}
