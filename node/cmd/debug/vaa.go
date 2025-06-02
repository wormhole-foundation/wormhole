package debug

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var decodeVaaCmd = &cobra.Command{
	Use:   "decode-vaa [DATA]",
	Short: "Decode a hex-encoded VAA",
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			arg = strings.TrimPrefix(arg, "0x")
			b, err := hex.DecodeString(arg)
			if err != nil {
				log.Fatal(err)
			}

			v, err := vaa.Unmarshal(b)
			if err != nil {
				log.Fatal(err)
			}

			debugStr, err := v.DebugString()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("%s", debugStr)
		}
	},
}
