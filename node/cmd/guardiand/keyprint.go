package guardiand

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/openpgp/armor" //nolint
	"google.golang.org/protobuf/proto"
)

var KeyprintCmd = &cobra.Command{
	Use:   "keyprint [KEYFILE]",
	Short: "Unmarshal and print armored guardian key in hex format",
	Run:   runKeyprint,
	Args:  cobra.ExactArgs(1),
}

func runKeyprint(cmd *cobra.Command, args []string) {
	keyFile := args[0]

	fmt.Println("Reading key from", keyFile)

	f, err := os.Open(keyFile)
	if err != nil {
		log.Fatalf("failed to open keyfile: %v", err)
	}

	p, err := armor.Decode(f)
	if err != nil {
		log.Fatalf("failed to read armored file: %v", err)
	}

	b, err := io.ReadAll(p.Body)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	var m nodev1.GuardianKey
	err = proto.Unmarshal(b, &m)
	if err != nil {
		log.Fatalf("failed to deserialize protobuf: %v", err)
	}

	fmt.Printf("Guardian key:\n")
	fmt.Printf("\tType: %s\n", p.Type)
	fmt.Printf("\tPrivatekey: %s\n", hex.EncodeToString(m.Data))
}
