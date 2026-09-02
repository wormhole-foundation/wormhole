package watchers

import (
	"slices"
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestRegisterRPCURL(t *testing.T) {
	rpcURLsMu.Lock()
	rpcURLs = map[vaa.ChainID][]string{}
	rpcURLsMu.Unlock()

	RegisterRPCURL(vaa.ChainIDEthereum, "")
	if urls := RPCURLs(vaa.ChainIDEthereum); urls != nil {
		t.Fatalf("expected empty URLs to be ignored, got %v", urls)
	}

	RegisterRPCURL(vaa.ChainIDEthereum, "https://eth.example")
	RegisterRPCURL(vaa.ChainIDEthereum, "https://eth.example")
	RegisterRPCURL(vaa.ChainIDEthereum, "https://eth-backup.example")

	urls := RPCURLs(vaa.ChainIDEthereum)
	want := []string{"https://eth.example", "https://eth-backup.example"}
	if !slices.Equal(urls, want) {
		t.Fatalf("unexpected URLs: got %v, want %v", urls, want)
	}

	urls[0] = "https://mutated.example"
	if got := RPCURLs(vaa.ChainIDEthereum); !slices.Equal(got, want) {
		t.Fatalf("RPCURLs returned mutable backing storage: got %v, want %v", got, want)
	}
}
