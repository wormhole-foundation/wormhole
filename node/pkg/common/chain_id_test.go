package common

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type ChainMap map[vaa.ChainID]string

// The purpose of this test is to verify that the `ChainID` definitions in the protobuf are in sync with the ones in vaa.structs.
// This test parses the generated protobuf file (thereby also verifying that generated files have been updated) and compares the
// list of chain IDs with the ones returned by `vaa.GetAllNetworkIDs`, making sure that everything matches and there are no duplicates.
func TestVerifyProtoBufChainIDs(t *testing.T) {
	// Get all of the chain IDs from the SDK and put them in a map keyed by chain ID.
	vaaChainList := vaa.GetAllNetworkIDs()
	require.NotEmpty(t, vaaChainList)
	assert.NotEmpty(t, vaaChainList)

	vaaChains := ChainMap{}
	for _, chainId := range vaaChainList {
		vaaChains[chainId] = chainId.String()
	}

	// GetAllNetworkIDs intentionally does not include "Unset". Add it in.
	vaaChains[vaa.ChainIDUnset] = "unset"

	// Get all of the chain IDs from the protobuf and put them in a map keyed by chain ID.
	protoChainList, err := parseProtoFile()
	require.NoError(t, err)
	require.NotEmpty(t, protoChainList)

	protoChains := ChainMap{}
	for _, entry := range protoChainList {
		chainId, err := vaa.ChainIDFromNumber(entry.chainID)
		label := "protoChains-invalid-value/" + entry.chainName + fmt.Sprintf("(%d)", entry.chainID)
		t.Run(label, func(t *testing.T) {
			require.NoError(t, err)
		})

		// This should never fail because the protobuf definition does not have `option allow_alias = true`, but we'll check it anyway.
		label = "protoChains-unique-value/" + entry.chainName + fmt.Sprintf("(%d)", entry.chainID)
		t.Run(label, func(t *testing.T) {
			_, exists := protoChains[chainId]
			require.False(t, exists)
		})

		protoChains[chainId] = entry.chainName
	}

	// Make sure everything in vaaChains is in protoChains.
	// This failure looks like this:
	// --- FAIL: TestVerifyProtoBufChainIDs/vaaChains-in-protoChains/junk(56) (0.00s)
	for chainId, chainName := range vaaChains {
		label := "vaaChains-in-protoChains/" + chainName + fmt.Sprintf("(%d)", uint16(chainId))
		t.Run(label, func(t *testing.T) {
			_, exists := protoChains[chainId]
			assert.True(t, exists)
		})
	}

	// Make sure everything in protoChains is in the vaaChains.
	// This failure looks like this:
	// --- FAIL: TestVerifyProtoBufChainIDs/protoChains-in-vaaChains/ChainID_CHAIN_ID_JUNK(56) (0.00s)
	for chainId, chainName := range protoChains {
		label := "protoChains-in-vaaChains/" + chainName + fmt.Sprintf("(%d)", uint16(chainId))
		t.Run(label, func(t *testing.T) {
			_, exists := vaaChains[chainId]
			assert.True(t, exists)
		})
	}
}

type ProtoEntry struct {
	chainID   int
	chainName string
}

// parseProtoFile parses the generated protobuf file and returns a list of all of the chain IDs defined there.
func parseProtoFile() ([]ProtoEntry, error) {
	// Parse the source file to extract ChainID constants from the generated protobuf file.
	// This reads the Go source code and builds an Abstract Syntax Tree (AST)
	// to programmatically find all ChainID constant declarations
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "../proto/publicrpc/v1/publicrpc.pb.go", nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Stores each ChainID declared as a constant in publicrpc.pb.go
	// Each ChainInfo contains the chain name and numeric value.
	chains := []ProtoEntry{}

	// Walk the AST to find ChainID constants
	// This traverses the parsed Go code looking for constant declarations
	// that have the type "ChainID"
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.CONST {
				for _, spec := range x.Specs {
					if vspec, ok := spec.(*ast.ValueSpec); ok {
						// Check if this is a ChainID constant by examining the type
						if vspec.Type != nil {
							if ident, ok := vspec.Type.(*ast.Ident); ok && ident.Name == "ChainID" {
								for i, name := range vspec.Names {
									// Extract the numeric value from the constant declaration
									if len(vspec.Values) > i {
										if basic, ok := vspec.Values[i].(*ast.BasicLit); ok {
											if value, err := strconv.Atoi(basic.Value); err == nil {
												chains = append(chains, ProtoEntry{value, name.Name})
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	return chains, nil
}
