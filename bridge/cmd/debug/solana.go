package debug

import (
	"context"
	"encoding/hex"
	"github.com/certusone/wormhole/bridge/pkg/solana"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"log"
	"strings"
)

var (
	agentRPC *string
)

func init() {
	agentRPC = postVaaSolanaCmd.Flags().String("agentRPC", "", "Solana agent sidecar gRPC socket path")
	DebugCmd.AddCommand(postVaaSolanaCmd)
}

var postVaaSolanaCmd = &cobra.Command{
	Use:   "post-vaa-solana [DATA]",
	Short: "Submit a hex-encoded VAA to Solana using the specified agent sidecar",
	Run: func(cmd *cobra.Command, args []string) {
		vaaQueue := make(chan *vaa.VAA)
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		supervisor.New(context.Background(), logger, func(ctx context.Context) error {
			if err := supervisor.Run(ctx, "solvaa",
				solana.NewSolanaVAASubmitter(*agentRPC, vaaQueue).Run); err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return nil
			}
		})

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

			vaaQueue <- v
		}

		select {}
	},
}
