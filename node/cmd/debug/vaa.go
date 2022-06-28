package debug

import (
	"encoding/hex"
	"log"
	"strings"

	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
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

			spew.Dump(v)
		}
	},
}
