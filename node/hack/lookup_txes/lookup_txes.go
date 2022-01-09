package main

import (
	"context"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"golang.org/x/time/rate"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

/*
	1/0def15a24423e1edd1a5ab16f557b9060303ddbab8c803d2ee48f4b78a1cfd6b/118
		-> wPkMzrFNdXFtATPFYDMh9EMJNZyAd4un7TCezG7AgY2
	1/ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5/66
		-> Gv1KWf8DT1jKv5pKBmGaTmVszqa56Xn8YGx2Pg7i7qAk
*/

const (
	mainnetApi = "http://localhost:8899"
)

var (
	emitter = solana.MustPublicKeyFromBase58("Gv1KWf8DT1jKv5pKBmGaTmVszqa56Xn8YGx2Pg7i7qAk")

	want = []int{
		41437,
		42300,
		42577,
		42578,
	}
)

func main() {
	ctx := context.Background()
	c := rpc.New(mainnetApi)

	limiter := rate.NewLimiter(rate.Every(1*time.Second), 10)

	var limit uint64 = 100
	var before solana.Signature

	sigs := make([]solana.Signature, 0, limit*2)

	sort.Slice(want, func(i, j int) bool {
		return want[i] < want[j]
	})

	minWant := want[0]
	maxWant := want[len(want)-1]

	for {
		err := limiter.Wait(ctx)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("requesting before: %s", before)
		out, err := c.GetConfirmedSignaturesForAddress2(ctx, emitter, &rpc.GetConfirmedSignaturesForAddress2Opts{
			Limit:  &limit,
			Before: before,
		})
		if err != nil {
			log.Fatalf("GetConfirmedSignaturesForAddress2 %s: %v", emitter, err)
		}

		if len(out) == 0 {
			log.Println("fetched all txes")
			break
		}

		last := out[len(out)-1].Signature

		log.Printf("fetched %d transactions, from %s to %s",
			len(out), out[0].Signature, last)

		for _, sig := range out {
			sigs = append(sigs, sig.Signature)
		}

		before = last
	}

	log.Printf("found a total of %d transactions", len(sigs))

	skip := 0

OUTER:
	for _, sig := range sigs {
		if skip > 0 {
			skip--
			continue
		}

		out, err := c.GetConfirmedTransaction(ctx, sig)
		if err != nil {
			log.Fatalf("GetConfirmedTransaction %s: %v", sig, err)
		}

		for _, msg := range out.Meta.LogMessages {
			if strings.HasPrefix(msg, "Program log: Sequence:") {
				seq := msg[23:]
				log.Printf("%s %s", sig, seq)

				seqInt, err := strconv.Atoi(seq)
				if err != nil {
					log.Printf("failed to parse seq %s: %v", seq, err)
					continue
				}

				for _, w := range want {
					if w == seqInt {
						log.Printf("FOUND https://explorer.solana.com/tx/%s seq %s", sig, seq)
					}
				}

				if seqInt > maxWant {
					skip = (seqInt - maxWant) / 2
					log.Printf("max=%d, cur=%d, skipping %d", maxWant, seqInt, skip)
					continue OUTER
				}

				if seqInt < minWant {
					log.Printf("min=%d, cur=%d, we're done", minWant, seqInt)
					break OUTER
				}
			}
		}
	}
}
