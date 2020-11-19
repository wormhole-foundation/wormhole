package guardiand

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	nodev1 "github.com/certusone/wormhole/bridge/pkg/proto/node/v1"
)

var keyDescription *string

func init() {
	keyDescription = KeygenCmd.Flags().String("desc", "", "Human-readable key description (optional)")
}

var KeygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Create guardian key at the specified path",
	Run:   runKeygen,
	Args:  cobra.ExactArgs(1),
}

func runKeygen(cmd *cobra.Command, args []string) {
	lockMemory()
	setRestrictiveUmask()

	log.Print("Creating new key at ", args[0])

	gk, err := ecdsa.GenerateKey(ethcrypto.S256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate key: %v", err)
	}

	err = writeGuardianKey(gk, *keyDescription, args[0])
	if err != nil {
		log.Fatalf("failed to write key: %v", err)
	}
}

// loadGuardianKey loads a serialized guardian key from disk.
func loadGuardianKey(filename string) (*ecdsa.PrivateKey, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read guardian private key from disk: %w", err)
	}

	var m nodev1.GuardianKey
	err = prototext.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize private key from disk: %w", err)
	}

	gk, err := ethcrypto.ToECDSA(m.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize key data: %w", err)
	}

	return gk, nil
}

// writeGuardianKey serializes a guardian key and writes it to disk.
func writeGuardianKey(key *ecdsa.PrivateKey, description string, filename string) error {
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return errors.New("refusing to override existing key")
	}

	m := &nodev1.GuardianKey{
		Description: description,
		Data:        ethcrypto.FromECDSA(key),
		Pubkey:      ethcrypto.PubkeyToAddress(key.PublicKey).String(),
	}

	b, err := prototext.MarshalOptions{Multiline: true, EmitASCII: true}.Marshal(m)
	if err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(filename, b, 0600); err != nil {
		return err
	}

	return nil
}

// generateDevnetGuardianKey returns a deterministic testnet key.
func generateDevnetGuardianKey() (*ecdsa.PrivateKey, error) {
	// Figure out our devnet index
	idx, err := devnet.GetDevnetIndex()
	if err != nil {
		return nil, err
	}

	// Generate guardian key
	return devnet.DeterministicEcdsaKeyByIndex(ethcrypto.S256(), uint64(idx)), nil
}
