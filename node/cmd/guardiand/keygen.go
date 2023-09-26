package guardiand

import (
	"crypto/ecdsa"
	"crypto/rand"
	"log"

	"github.com/certusone/wormhole/node/pkg/common"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var keyDescription *string
var blockType *string

func init() {
	keyDescription = KeygenCmd.Flags().String("desc", "", "Human-readable key description (optional)")
	blockType = KeygenCmd.Flags().String("block-type", common.GuardianKeyArmoredBlock, "block type of armored file (optional)")
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

	err = common.WriteArmoredKey(gk, *keyDescription, args[0], *blockType, false)
	if err != nil {
		log.Fatalf("failed to write key: %v", err)
	}
}
