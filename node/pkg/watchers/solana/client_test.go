package solana

import (
	"testing"

	"github.com/gagliardetto/solana-go"

	"github.com/stretchr/testify/assert"
)

func TestVerifyConstants(t *testing.T) {
	// If either of these ever change, message publication and reobservation may break.
	assert.Equal(t, SolanaAccountLen, solana.PublicKeyLength)
	assert.Equal(t, SolanaSignatureLen, len(solana.Signature{}))
}

var whLogPrefixForMainnet = createWhLogPrefix("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth")
var whLogPrefixForTestnet = createWhLogPrefix("3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5")

func TestIsPossibleWormholeMessageSuccess(t *testing.T) {
	// These are actual logs see in mainnet on 8/29/2024.
	logs := []string{
		"Program 3vxKRPwUTiEkeUVyoZ9MXFe1V71sRLbLqu1gRYaWmehQ invoke [1]",
		"Program log: Instruction: TransferWrappedTokensWithRelay",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [2]",
		"Program log: Instruction: InitializeAccount3",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA consumed 4214 of 189385 compute units",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [2]",
		"Program log: Instruction: Transfer",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA consumed 4645 of 162844 compute units",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [2]",
		"Program log: Instruction: Approve",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA consumed 2904 of 153570 compute units",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success",
		"Program wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb invoke [2]",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [3]",
		"Program log: Instruction: Burn",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA consumed 4790 of 91730 compute units",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success",
		"Program 11111111111111111111111111111111 invoke [3]",
		"Program 11111111111111111111111111111111 success",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [3]",
		"Program log: Sequence: 937184",
		"Program 11111111111111111111111111111111 invoke [4]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [4]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [4]",
		"Program 11111111111111111111111111111111 success",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 27141 of 74067 compute units",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
		"Program wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb consumed 87537 of 133116 compute units",
		"Program wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb success",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [2]",
		"Program log: Instruction: CloseAccount",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA consumed 3015 of 42346 compute units",
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success",
		"Program 3vxKRPwUTiEkeUVyoZ9MXFe1V71sRLbLqu1gRYaWmehQ consumed 186778 of 224134 compute units",
		"Program 3vxKRPwUTiEkeUVyoZ9MXFe1V71sRLbLqu1gRYaWmehQ success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
	}

	assert.True(t, isPossibleWormholeMessage(whLogPrefixForMainnet, logs))
}

func TestIsPossibleWormholeMessageFailNoLogs(t *testing.T) {
	// These are actual logs see in mainnet on 8/29/2024.
	logs := []string{}

	assert.False(t, isPossibleWormholeMessage(whLogPrefixForMainnet, logs))
}

func TestIsPossibleWormholeMessageFailNoWormhole(t *testing.T) {
	// These are actual logs see in mainnet on 8/29/2024.
	logs := []string{
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
	}

	assert.False(t, isPossibleWormholeMessage(whLogPrefixForMainnet, logs))
}

func TestIsPossibleWormholeMessageFailNoSequence(t *testing.T) {
	// These are actual logs see in mainnet on 8/29/2024.
	logs := []string{
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [1]",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 37058 of 500000 compute units",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
	}

	assert.False(t, isPossibleWormholeMessage(whLogPrefixForMainnet, logs))
}

func TestIsPossibleWormholeMessageFailAtEnd(t *testing.T) {
	// Note: I altered these logs to create this test. I don't know if this could ever happen.
	logs := []string{
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 37058 of 500000 compute units",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [1]",
	}

	assert.False(t, isPossibleWormholeMessage(whLogPrefixForMainnet, logs))
}

func TestIsPossibleWormholeMessageForSolanaRewriteSuccess(t *testing.T) {
	// These are actual logs see in testnet on 8/30/2024.
	logs := []string{
		"Program 4cmLyfxkgj2rGkPnXXJG8uroGvoSqsgigAsB3ACci2x7 invoke [1]",
		"Program log: Instruction: WithdrawNative",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program log: ctx.accounts.wormhole_program : 3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5",
		"Program log: ctx.accounts.wormhole_bridge : 6bi4JGDoRwUs9TYBuvoA7dUVyikTJDrJsJU1ew6KVLiu",
		"Program log: ctx.accounts.wormhole_message : DzKjuonZWoBjc9VcrtBBXwnEXA248Ma6fjUp1hgez9yg",
		"Program log: ctx.accounts.custom_wormhole_emitter : F7thxxU2rLuXBj8xxzA39K63oZALrFuYRkCPUpQVhRb3",
		"Program log: ctx.accounts.custom_wormhole_sequence : FEDW1vsFoGbQkaw9gbwuEZEsSwXLHcAKBVesXnmYNQnU",
		"Program log: ctx.accounts.depositor : 3X1PnDdoyMdzLEoEy2TAhw8i1zC1F4ipNsxJHotnzCKK",
		"Program log: ctx.accounts.wormhole_fee_collector : 7s3a1ycs16d6SNDumaRtjcoyMaTDZPavzgsmS3uUZYWX",
		"Program 3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 invoke [2]",
		"Program log: Instruction: LegacyPostMessage",
		"Program 11111111111111111111111111111111 invoke [3]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [3]",
		"Program 11111111111111111111111111111111 success",
		"Program log: Sequence: 26",
		"Program 3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 consumed 36772 of 68261 compute units",
		"Program 3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 success",
		"Program 4cmLyfxkgj2rGkPnXXJG8uroGvoSqsgigAsB3ACci2x7 consumed 169398 of 200000 compute units",
		"Program 4cmLyfxkgj2rGkPnXXJG8uroGvoSqsgigAsB3ACci2x7 success",
	}

	assert.True(t, isPossibleWormholeMessage(whLogPrefixForTestnet, logs))
}
