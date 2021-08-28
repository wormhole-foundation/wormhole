package debug

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	agentv1 "github.com/certusone/wormhole/bridge/pkg/proto/agent/v1"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var resubmitAgentRPC *string

func init() {
	resubmitAgentRPC = resubmitSolanaCmd.Flags().String("agentRPC", "", "Solana agent sidecar gRPC socket path")
	DebugCmd.AddCommand(resubmitSolanaCmd)
}

type submittedVAAMessage struct {
	Digest string `json:"digest"`
	Vaa    struct {
		Version          int `json:"Version"`
		GuardianSetIndex int `json:"GuardianSetIndex"`
		Signatures       []struct {
			Index     int   `json:"Index"`
			Signature []int `json:"Signature"`
		} `json:"Signatures"`
		Timestamp time.Time `json:"Timestamp"`
		Payload   struct {
			Nonce         int   `json:"Nonce"`
			SourceChain   int   `json:"SourceChain"`
			TargetChain   int   `json:"TargetChain"`
			SourceAddress []int `json:"SourceAddress"`
			TargetAddress []int `json:"TargetAddress"`
			Asset         struct {
				Chain    int   `json:"Chain"`
				Address  []int `json:"Address"`
				Decimals int   `json:"Decimals"`
			} `json:"Asset"`
			Amount int `json:"Amount"`
		} `json:"Payload"`
	} `json:"vaa"`
	Bytes string `json:"bytes"`
}

var resubmitSolanaCmd = &cobra.Command{
	Use:   "resubmit-from-log [LOGFILE]",
	Short: "Attempts to resubmit all signed VAAs in the given logfile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		vaaQueue := make(chan *vaa.VAA)
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}

		file, err := os.Open(args[0])
		if err != nil {
			logger.Fatal("failed to open logfile", zap.Error(err))
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		reJSON, err := regexp.Compile(`\{.*}`)
		if err != nil {
			panic(err)
		}

		wg := sync.WaitGroup{}

		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "submitting signed VAA to Solana") {
				m := reJSON.FindStringSubmatch(line)
				var msg *submittedVAAMessage
				err = json.Unmarshal([]byte(m[0]), &msg)
				if err != nil {
					logger.Warn("failed to unmarshal entry", zap.Error(err))
					fmt.Println(m[0])
					continue
				}

				go processMessage(msg, logger, vaaQueue, &wg)
				wg.Add(1)
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Fatal("failed reading logfile", zap.Error(err))
		}

		go resubmitProcessor(ctx, logger, *resubmitAgentRPC, vaaQueue, &wg)

		wg.Wait()
	},
}

func processMessage(msg *submittedVAAMessage, logger *zap.Logger, c chan<- *vaa.VAA, wg *sync.WaitGroup) {
	logger.Info("processing",
		zap.Time("timestamp", msg.Vaa.Timestamp),
		zap.String("digest", msg.Digest),
	)

	b, err := hex.DecodeString(msg.Bytes)
	if err != nil {
		logger.Warn("failed to decode hex string", zap.String("digest", msg.Digest))
		wg.Done()
		return
	}

	v, err := vaa.Unmarshal(b)
	if err != nil {
		logger.Warn("failed to unmarshal VAA", zap.String("digest", msg.Digest))
		wg.Done()
		return
	}

	c <- v
}

func resubmitProcessor(ctx context.Context, logger *zap.Logger, agentSock string, c chan *vaa.VAA, wg *sync.WaitGroup) {
	path := fmt.Sprintf("unix://%s", agentSock)

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(timeout, path, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		logger.Fatal("failed to dial agent", zap.String("agent", agentSock))
	}
	defer conn.Close()

	agent := agentv1.NewAgentClient(conn)

	for v := range c {
		processResubmit(ctx, v, logger, agent, wg)
	}

}

func processResubmit(ctx context.Context, v *vaa.VAA, logger *zap.Logger, agent agentv1.AgentClient, wg *sync.WaitGroup) {
	b, err := v.Marshal()
	if err != nil {
		panic(err)
	}

	m, err := v.SigningMsg()
	if err != nil {
		panic(err)
	}
	h := hex.EncodeToString(m.Bytes())

	logger.Info("resubmitting", zap.String("digest", h))

	timeout, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	res, err := agent.SubmitVAA(timeout, &agentv1.SubmitVAARequest{Vaa: b, SkipPreflight: false})
	if err != nil {
		_, ok := status.FromError(err)
		if !ok {
			panic("err not a status")
		}

		logger.Warn("failed to resubmit", zap.String("digest", h), zap.Error(err))
		wg.Done()
		return
	}

	logger.Info("submitted VAA",
		zap.String("tx_sig", res.Signature), zap.String("digest", h))

	wg.Done()
}
