package debug

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/mr-tron/base58"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"regexp"
	"strings"
)

type observedAndSignedLockup struct {
	SourceChain string `json:"source_chain"`
	TargetChain string `json:"target_chain"`
	Txhash      string `json:"txhash"`
	Digest      string `json:"digest"`
	Signature   string `json:"signature"`
}

func init() {
	DebugCmd.AddCommand(lookupSolanaCmd)
}

const githubMessage = `
The transaction was successfully processed by the Wormhole network, but the client-side transaction submission may have failed. Try clearing your cache and reloading wormholebridge.com to see if the transaction can be completed there. Make sure both wallets are connected.

If wormholebridge.com doesn't work, you can manually submit the signed VAA bytes below to the submitVAA method on the Wormhole contract: 1https://etherscan.io/address/0xf92cd566ea4864356c5491c177a430c222d7e678#writeContract

---

Digest (for reference only): %s

Signed VAA bytes to submit to the contract:

> 0x%s

`

var lookupSolanaCmd = &cobra.Command{
	Use:   "lookup-from-log [LOGFILE] [TRANSACTION]",
	Short: "Looks up the signed VAA for a specific Solana or Eth address from the given logfile",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		tx := args[1]

		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}

		file, err := os.Open(path)
		if err != nil {
			logger.Fatal("failed to open logfile", zap.Error(err))
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		reJSON, err := regexp.Compile(`\{.*}`)
		if err != nil {
			panic(err)
		}

		needle, err := parseTxArg(tx)
		if err != nil {
			logger.Fatal("invalid tx", zap.Error(err))
		}

		digests := make(map[string]bool)

		for scanner.Scan() {
			line := scanner.Text()

			if strings.Contains(line, "observed and signed confirmed lockup") {
				m := reJSON.FindStringSubmatch(line)
				var msg *observedAndSignedLockup
				err = json.Unmarshal([]byte(m[0]), &msg)
				if err != nil {
					logger.Warn("failed to unmarshal entry", zap.Error(err))
					fmt.Println(m[0])
					continue
				}

				if msg.Txhash == "0x"+needle {
					if digests[msg.Digest] {
						continue
					}
					logger.Info("found matching lockup", zap.Any("txhash", needle), zap.String("digest", msg.Digest))
					digests[msg.Digest] = true
				}
			}

			if strings.Contains(line, "submitting signed VAA to Solana") {
				m := reJSON.FindStringSubmatch(line)
				var msg *submittedVAAMessage
				err = json.Unmarshal([]byte(m[0]), &msg)
				if err != nil {
					logger.Warn("failed to unmarshal entry", zap.Error(err))
					fmt.Println(m[0])
					continue
				}

				if digests[msg.Digest] {
					fmt.Printf(githubMessage, msg.Digest, msg.Bytes)
				}
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Fatal("failed reading logfile", zap.Error(err))
		}
	},
}

func parseTxArg(tx string) (string, error) {
	if strings.HasPrefix(tx, "0x") {
		return tx, nil
	} else {
		b, err := base58.Decode(tx)
		if err != nil {
			return "", err
		}

		return hex.EncodeToString(b), nil
	}
}
