//nolint:unparam
package adminrpc

type adminCommandTestEntry struct {
	label     string
	errText   string // empty string means success
	prototext string
}

var adminCommandTest = []adminCommandTestEntry{
	// build/bin/guardiand template guardian-set-update --num=2 --idx=4
	{
		label:   "GuardianSetUpdate success",
		errText: "",
		prototext: `
			current_set_index: 4
			messages: {
				sequence: 13675600082943268828
				nonce: 1875482155
				guardian_set: {
					guardians: {
						pubkey: "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
						name: "Example validator 0"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
				}
			}`,
	},
	{
		label:   "GuardianSetUpdate no guardian keys",
		errText: "empty guardian set specified",
		prototext: `
			current_set_index: 4
			messages: {
				sequence: 13675600082943268828
				nonce: 1875482155
				guardian_set: {
				}
			}`,
	},
	{
		label:   "GuardianSetUpdate too many guardian keys",
		errText: "too many guardians",
		prototext: `
			current_set_index: 4
			messages: {
				sequence: 13675600082943268828
				nonce: 1875482155
				guardian_set: {
					guardians: {
						pubkey: "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
						name: "Example validator 0"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
					guardians: {
						pubkey: "0x88D7D8B32a9105d228100E72dFFe2Fae0705D31c"
						name: "Example validator 1"
					}
				}
			}`,
	},
	{
		label:   "GuardianSetUpdate invalid guardian key",
		errText: "invalid pubkey format",
		prototext: `
			current_set_index: 4
			messages: {
				sequence: 13675600082943268828
				nonce: 1875482155
				guardian_set: {
					guardians: {
						pubkey: "Hello, World!"
						name: "Example validator 0"
					}
				}
			}`,
	},
	{
		label:   "GuardianSetUpdate duplicate guardian key",
		errText: "duplicate pubkey at index 1",
		prototext: `
			current_set_index: 4
			messages: {
				sequence: 13675600082943268828
				nonce: 1875482155
				guardian_set: {
					guardians: {
						pubkey: "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
						name: "Example validator 0"
					}
					guardians: {
						pubkey: "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
						name: "Example validator 0"
					}
				}
			}`,
	},

	// build/bin/guardiand template contract-upgrade --chain-id "ethereum" --new-address 0xC89Ce4735882C9F0f0FE26686c53074E09B0D550
	{
		label:   "ContractUpgrade success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 12970493895929703138
					nonce: 700990237
					contract_upgrade: {
						chain_id: 2
						new_contract: "000000000000000000000000c89ce4735882c9f0f0fe26686c53074e09b0d550"
					}
				}`,
	},
	{
		label:   "ContractUpgrade invalid contract address",
		errText: "invalid new contract address encoding (expected hex)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 12970493895929703138
					nonce: 700990237
					contract_upgrade: {
						chain_id: 2
						new_contract: "Hello, World!"
					}
				}`,
	},
	{
		label:   "ContractUpgrade contract address wrong length",
		errText: "invalid new_contract address",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 12970493895929703138
					nonce: 700990237
					contract_upgrade: {
						chain_id: 2
						new_contract: "c89ce4735882c9f0f0fe26686c53074e09b0d550"
					}
				}`,
	},
	{
		label:   "ContractUpgrade chain id too large",
		errText: "invalid chain_id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 12970493895929703138
					nonce: 700990237
					contract_upgrade: {
						chain_id: 65536
						new_contract: "000000000000000000000000c89ce4735882c9f0f0fe26686c53074e09b0d550"
					}
				}`,
	},

	// build/bin/guardiand template token-bridge-register-chain --chain-id ethereum --module TokenBridge --new-address 0x0290FB167208Af455bB137780163b7B7a9a10C16
	{
		label:   "BridgeRegisterChain success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13048739474937843907
					nonce: 277897432
					bridge_register_chain: {
						module: "TokenBridge"
						chain_id: 2
						emitter_address: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
					}
				}`,
	},
	{
		label:   "BridgeRegisterChain invalid chain id",
		errText: "invalid chain_id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13048739474937843907
					nonce: 277897432
					bridge_register_chain: {
						module: "TokenBridge"
						chain_id: 65536
						emitter_address: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
					}
				}`,
	},
	{
		label:   "BridgeRegisterChain invalid emitter address",
		errText: "invalid emitter address encoding (expected hex)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13048739474937843907
					nonce: 277897432
					bridge_register_chain: {
						module: "TokenBridge"
						chain_id: 2
						emitter_address: "Hello, World!"
					}
				}`,
	},
	{
		label:   "BridgeRegisterChain invalid emitter address length",
		errText: "invalid emitter address (expected 32 bytes)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13048739474937843907
					nonce: 277897432
					bridge_register_chain: {
						module: "TokenBridge"
						chain_id: 2
						emitter_address: "0290fb167208af455bb137780163b7b7a9a10c16"
					}
				}`,
	},

	// build/bin/guardiand template recover-chain-id --module TokenBridge --evm-chain-id 42 --new-chain-id 43
	{
		label:   "RecoverChainId success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 4474356899438211298
					nonce: 4040780926
					recover_chain_id: {
						module: "TokenBridge"
						evm_chain_id: "42"
						new_chain_id: 43
					}
				}`,
	},
	{
		label:   "RecoverChainId invalid evm chain id",
		errText: "invalid evm_chain_id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 4474356899438211298
					nonce: 4040780926
					recover_chain_id: {
						module: "TokenBridge"
						evm_chain_id: "Hello, World!"
						new_chain_id: 43
					}
				}`,
	},
	{
		label:   "RecoverChainId evm chain id too large",
		errText: "evm_chain_id overflow",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 4474356899438211298
					nonce: 4040780926
					recover_chain_id: {
						module: "TokenBridge"
						evm_chain_id: "115792089237316195423570985008687907853269984665640564039457584007913129639936"
						new_chain_id: 43
					}
				}`,
	},
	{
		label:   "RecoverChainId invalid new chain id",
		errText: "invalid new_chain_id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 4474356899438211298
					nonce: 4040780926
					recover_chain_id: {
						module: "TokenBridge"
						evm_chain_id: "42"
						new_chain_id: 65536
					}
				}`,
	},

	// build/bin/guardiand template accountant-modify-balance --target-chain-id 3104 --sequence 3 --chain-id 1 --token-chain-id 2 --token-address 0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2 --action 1 --amount 12000000000000 --reason "fix bad value"
	{
		label:   "AccountantModifyBalance add success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 3104
						sequence: 3
						chain_id: 1
						token_chain: 2
						token_address: "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
						kind:  MODIFICATION_KIND_ADD
						amount: "12000000000000"
						reason: "fix bad value"
					}
				}`,
	},
	{
		label:   "AccountantModifyBalance subtract success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 3104
						sequence: 3
						chain_id: 1
						token_chain: 2
						token_address: "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
						kind:  MODIFICATION_KIND_SUBTRACT
						amount: "12000000000"
						reason: "fix bad value"
					}
				}`,
	},
	{
		label:   "AccountantModifyBalance invalid target chain id",
		errText: "invalid target_chain_id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 65536
						sequence: 3
						chain_id: 1
						token_chain: 2
						token_address: "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
						kind:  MODIFICATION_KIND_ADD
						amount: "12000000000000"
						reason: "fix bad value"
					}
				}`,
	},
	{
		label:   "AccountantModifyBalance invalid chain id",
		errText: "invalid chain_id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 3104
						sequence: 3
						chain_id: 65536
						token_chain: 2
						token_address: "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
						kind:  MODIFICATION_KIND_ADD
						amount: "12000000000000"
						reason: "fix bad value"
					}
				}`,
	},
	{
		label:   "AccountantModifyBalance invalid token chain id",
		errText: "invalid token_chain",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 3104
						sequence: 3
						chain_id: 1
						token_chain: 65536
						token_address: "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
						kind:  MODIFICATION_KIND_ADD
						amount: "12000000000000"
						reason: "fix bad value"
					}
				}`,
	},
	{
		label:   "AccountantModifyBalance invalid token address",
		errText: "invalid token address (expected hex)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 3104
						sequence: 3
						chain_id: 1
						token_chain: 2
						token_address: "Hello, World!"
						kind:  MODIFICATION_KIND_ADD
						amount: "12000000000000"
						reason: "fix bad value"
					}
				}`,
	},
	{
		label:   "AccountantModifyBalance token address wrong length",
		errText: "invalid new token address (expected 32 bytes)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 3104
						sequence: 3
						chain_id: 1
						token_chain: 2
						token_address: "0000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
						kind:  MODIFICATION_KIND_ADD
						amount: "12000000000000"
						reason: "fix bad value"
					}
				}`,
	},
	{
		label:   "AccountantModifyBalance reason too long",
		errText: "the reason should not be larger than 32 bytes",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 3104
						sequence: 3
						chain_id: 1
						token_chain: 2
						token_address: "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
						kind:  MODIFICATION_KIND_ADD
						amount: "12000000000000"
						reason: "reason is too long!!!!!!!!!!!!!!!"
					}
				}`,
	},
	{
		label:   "AccountantModifyBalance invalid amount",
		errText: "invalid amount",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 3104
						sequence: 3
						chain_id: 1
						token_chain: 2
						token_address: "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
						kind:  MODIFICATION_KIND_ADD
						amount: "Hello, World!"
						reason: "fix bad value"
					}
				}`,
	},
	{
		label:   "AccountantModifyBalance amount overflow",
		errText: "amount overflow",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					accountant_modify_balance: {
						module:  "GlobalAccountant"
						target_chain_id: 3104
						sequence: 3
						chain_id: 1
						token_chain: 2
						token_address: "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
						kind:  MODIFICATION_KIND_ADD
						amount: "115792089237316195423570985008687907853269984665640564039457584007913129639936"
						reason: "fix bad value"
					}
				}`,
	},

	// build/bin/guardiand template token-bridge-upgrade-contract --chain-id ethereum --module TokenBridge --new-address 0x0290FB167208Af455bB137780163b7B7a9a10C16
	{
		label:   "BridgeUpgradeContract success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 2474806415844516465
					nonce: 1137535017
					bridge_contract_upgrade: {
						module: "TokenBridge"
						target_chain_id: 2
						new_contract: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
					}
				}`,
	},
	{
		label:   "BridgeUpgradeContract invalid target chain id",
		errText: "invalid target_chain_id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 2474806415844516465
					nonce: 1137535017
					bridge_contract_upgrade: {
						module: "TokenBridge"
						target_chain_id: 65536
						new_contract: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
					}
				}`,
	},
	{
		label:   "BridgeUpgradeContract invalid new contract address",
		errText: "invalid new contract address (expected hex)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 2474806415844516465
					nonce: 1137535017
					bridge_contract_upgrade: {
						module: "TokenBridge"
						target_chain_id: 2
						new_contract: "Hello, World!"
					}
				}`,
	},
	{
		label:   "BridgeUpgradeContract contract address too long",
		errText: "invalid new contract address (expected 32 bytes)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 2474806415844516465
					nonce: 1137535017
					bridge_contract_upgrade: {
						module: "TokenBridge"
						target_chain_id: 2
						new_contract: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c1600"
					}
				}`,
	},

	// build/bin/guardiand template wormchain-store-code --wasm-hash 0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16
	{
		label:   "WormchainStoreCode success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 1985924966232230622
					nonce: 1162627577
					wormchain_store_code: {
						wasm_hash: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
					}
				}`,
	},
	{
		label:   "WormchainStoreCode invalid wasm hash",
		errText: "invalid cosmwasm bytecode hash (expected hex)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 1985924966232230622
					nonce: 1162627577
					wormchain_store_code: {
						wasm_hash: "Hello, World!"
					}
				}`,
	},
	{
		label:   "WormchainStoreCode invalid wasm hash length",
		errText: "invalid cosmwasm bytecode hash (expected 32 bytes but received 33 bytes)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 1985924966232230622
					nonce: 1162627577
					wormchain_store_code: {
						wasm_hash: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c1600"
					}
				}`,
	},

	// build/bin/guardiand template wormchain-instantiate-contract --code-id 12345678 --label "Hi, Mom!" --instantiation-msg "Some random junk"
	{
		label:   "WormchainInstantiateContract success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 3355941316502237895
					nonce: 749992725
					wormchain_instantiate_contract: {
						code_id: 12345678
						label: "Hi, Mom!"
						instantiation_msg: "Some random junk"
					}
				}`,
	},

	// build/bin/guardiand template wormchain-migrate-contract --code-id 12345678 --contract-address wormhole1ghd753shjuwexxywmgs4xz7x2q732vcnkm6h2pyv9s6ah3hylvrqtm7t3h --instantiation-msg "Some junk"
	{
		label:   "WormchainMigrateContract success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13069018493465262425
					nonce: 222803138
					wormchain_migrate_contract: {
						code_id: 12345678
						contract: "wormhole1ghd753shjuwexxywmgs4xz7x2q732vcnkm6h2pyv9s6ah3hylvrqtm7t3h"
						instantiation_msg: "Some junk"
					}
				}`,
	},

	// build/bin/guardiand template wormchain-add-wasm-instantiate-allowlist --code-id 12345678 --contract-address wormchain-add-wasm-instantiate-allowlist
	{
		label:   "WormchainWasmInstantiateAllowlist success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 14341603504021037676
					nonce: 1682273406
					wormchain_wasm_instantiate_allowlist: {
						code_id: 12345678
						contract: "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh"
						action: WORMCHAIN_WASM_INSTANTIATE_ALLOWLIST_ACTION_ADD
					}
				}`,
	},
	{
		label:   "WormchainWasmInstantiateAllowlist invalid bech32 contract address",
		errText: "decoding bech32 failed",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 14341603504021037676
					nonce: 1682273406
					wormchain_wasm_instantiate_allowlist: {
						code_id: 12345678
						contract: "Hi, Mom!"
						action: WORMCHAIN_WASM_INSTANTIATE_ALLOWLIST_ACTION_ADD
					}
				}`,
	},
	{
		label:   "WormchainWasmInstantiateAllowlist unexpected action",
		errText: "unrecognized wasm instantiate allowlist action",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 14341603504021037676
					nonce: 1682273406
					wormchain_wasm_instantiate_allowlist: {
						code_id: 12345678
						contract: "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh"
						action: WORMCHAIN_WASM_INSTANTIATE_ALLOWLIST_ACTION_UNSPECIFIED
					}
				}`,
	},

	// build/bin/guardiand template circle-integration-update-wormhole-finality --chain-id ethereum --finality 42
	{
		label:   "CircleIntegrationUpdateWormholeFinality success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 6415677735604800159
					nonce: 1410643993
					circle_integration_update_wormhole_finality: {
						finality: 42
						target_chain_id: 2
					}
				}`,
	},

	// build/bin/guardiand template circle-integration-register-emitter-and-domain --chain-id ethereum --circle-domain 42 --foreign-emitter-chain-id bsc --foreign-emitter-address 0x0290FB167208Af455bB137780163b7B7a9a10C16
	{
		label:   "CircleIntegrationRegisterEmitterAndDomain success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 4922548351928530675
					nonce: 1868545932
					circle_integration_register_emitter_and_domain: {
						foreign_emitter_chain_id: 4
						foreign_emitter_address: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
						circle_domain: 42
						target_chain_id: 2
					}
				}`,
	},
	{
		label:   "CircleIntegrationRegisterEmitterAndDomain invalid target chain id",
		errText: "invalid target chain id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 4922548351928530675
					nonce: 1868545932
					circle_integration_register_emitter_and_domain: {
						foreign_emitter_chain_id: 4
						foreign_emitter_address: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
						circle_domain: 42
						target_chain_id: 65536
					}
				}`,
	},
	{
		label:   "CircleIntegrationRegisterEmitterAndDomain invalid foreign chain id",
		errText: "invalid foreign emitter chain id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 4922548351928530675
					nonce: 1868545932
					circle_integration_register_emitter_and_domain: {
						foreign_emitter_chain_id: 65536
						foreign_emitter_address: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
						circle_domain: 42
						target_chain_id: 2
					}
				}`,
	},
	{
		label:   "CircleIntegrationRegisterEmitterAndDomain invalid foreign emitter address",
		errText: "invalid foreign emitter address encoding",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 4922548351928530675
					nonce: 1868545932
					circle_integration_register_emitter_and_domain: {
						foreign_emitter_chain_id: 4
						foreign_emitter_address: "Hello, World!"
						circle_domain: 42
						target_chain_id: 2
					}
				}`,
	},
	{
		label:   "CircleIntegrationRegisterEmitterAndDomain foreign emitter address too short",
		errText: "invalid foreign emitter address (expected 32 bytes)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 4922548351928530675
					nonce: 1868545932
					circle_integration_register_emitter_and_domain: {
						foreign_emitter_chain_id: 4
						foreign_emitter_address: "00000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
						circle_domain: 42
						target_chain_id: 2
					}
				}`,
	},

	// build/bin/guardiand template circle-integration-upgrade-contract-implementation --chain-id ethereum --new-implementation-address 0x0290FB167208Af455bB137780163b7B7a9a10C16
	{
		label:   "CircleIntegrationUpgradeContractImplementation success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 17801246082911495918
					nonce: 3226303109
					circle_integration_upgrade_contract_implementation: {
						new_implementation_address: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
						target_chain_id: 2
					}
				}`,
	},
	{
		label:   "CircleIntegrationUpgradeContractImplementation invalid target chain id",
		errText: "invalid target chain id, must be <= 65535",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 17801246082911495918
					nonce: 3226303109
					circle_integration_upgrade_contract_implementation: {
						new_implementation_address: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
						target_chain_id: 65536
					}
				}`,
	},
	{
		label:   "CircleIntegrationUpgradeContractImplementation invalid implementation address",
		errText: "invalid new implementation address encoding (expected hex)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 17801246082911495918
					nonce: 3226303109
					circle_integration_upgrade_contract_implementation: {
						new_implementation_address: "Hello, World!"
						target_chain_id: 2
					}
				}`,
	},
	{
		label:   "CircleIntegrationUpgradeContractImplementation invalid implementation address length",
		errText: "invalid new implementation address (expected 32 bytes)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 17801246082911495918
					nonce: 3226303109
					circle_integration_upgrade_contract_implementation: {
						new_implementation_address: "00000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
						target_chain_id: 2
					}
				}`,
	},

	// build/bin/guardiand template ibc-receiver-update-channel-chain --target-chain-id 3104 --channel-id "channel-3" --chain-id 20
	{
		label:   "IbcUpdateChannelChain success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 17350963017355124444
					nonce: 45403294
					ibc_update_channel_chain: {
						target_chain_id: 3104
						channel_id: "channel-3"
						chain_id: 20
						module: IBC_UPDATE_CHANNEL_CHAIN_MODULE_RECEIVER
					}
				}`,
	},
	{
		label:   "IbcUpdateChannelChain invalid target chain id",
		errText: "invalid target chain id, must be <= 65535",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 17350963017355124444
					nonce: 45403294
					ibc_update_channel_chain: {
						target_chain_id: 65536
						channel_id: "channel-3"
						chain_id: 20
						module: IBC_UPDATE_CHANNEL_CHAIN_MODULE_RECEIVER
					}
				}`,
	},
	{
		label:   "IbcUpdateChannelChain invalid chain id",
		errText: "invalid chain id, must be <= 65535",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 17350963017355124444
					nonce: 45403294
					ibc_update_channel_chain: {
						target_chain_id: 3104
						channel_id: "channel-3"
						chain_id: 65536
						module: IBC_UPDATE_CHANNEL_CHAIN_MODULE_RECEIVER
					}
				}`,
	},
	{
		label:   "IbcUpdateChannelChain invalid chain id",
		errText: "invalid channel ID length, must be <= 64",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 17350963017355124444
					nonce: 45403294
					ibc_update_channel_chain: {
						target_chain_id: 3104
						channel_id: "channel-that-is-too-long!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
						chain_id: 20
						module: IBC_UPDATE_CHANNEL_CHAIN_MODULE_RECEIVER
					}
				}`,
	},
	{
		label:   "IbcUpdateChannelChain invalid module",
		errText: "unrecognized ibc update channel chain module",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 17350963017355124444
					nonce: 45403294
					ibc_update_channel_chain: {
						target_chain_id: 3104
						channel_id: "channel-3"
						chain_id: 20
						module: IBC_UPDATE_CHANNEL_CHAIN_MODULE_UNSPECIFIED
					}
				}`,
	},

	// build/bin/guardiand template wormhole-relayer-set-default-delivery-provider --chain-id ethereum --new-address 0x0290fb167208af455bb137780163b7b7a9a10c16
	{
		label:   "WormholeRelayerSetDefaultDeliveryProvider success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13862516333551023628
					nonce: 2155214233
					wormhole_relayer_set_default_delivery_provider: {
						chain_id: 2
						new_default_delivery_provider_address: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
					}
				}`,
	},
	{
		label:   "WormholeRelayerSetDefaultDeliveryProvider invalid target chain id",
		errText: "invalid target_chain_id",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13862516333551023628
					nonce: 2155214233
					wormhole_relayer_set_default_delivery_provider: {
						chain_id: 65536
						new_default_delivery_provider_address: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
					}
				}`,
	},
	{
		label:   "WormholeRelayerSetDefaultDeliveryProvider invalid new address",
		errText: "invalid new default delivery provider address (expected hex)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13862516333551023628
					nonce: 2155214233
					wormhole_relayer_set_default_delivery_provider: {
						chain_id: 2
						new_default_delivery_provider_address: "Hello, World!"
					}
				}`,
	},
	{
		label:   "WormholeRelayerSetDefaultDeliveryProvider new address wrong length",
		errText: "invalid new default delivery provider address (expected 32 bytes)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13862516333551023628
					nonce: 2155214233
					wormhole_relayer_set_default_delivery_provider: {
						chain_id: 2
						new_default_delivery_provider_address: "00000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
					}
				}`,
	},

	// build/bin/guardiand template governance-evm-call --chain-id ethereum --target-address 0x0290fb167208af455bb137780163b7b7a9a10c16 --call-data 00010304 --governance-contract 0x0000000000000000000000000000000000000004
	{
		label:   "EvmCall success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13339105063374311672
					nonce: 2195389763
					evm_call: {
						chain_id: 2
						governance_contract: "0x0000000000000000000000000000000000000004"
						target_contract: "0x0290FB167208Af455bB137780163b7B7a9a10C16"
						abi_encoded_call: "00010304"
					}
				}`,
	},
	{
		label:   "EvmCall invalid call data (shouldn't start with 0x)",
		errText: "failed to decode ABI encoded call",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 13339105063374311672
					nonce: 2195389763
					evm_call: {
						chain_id: 2
						governance_contract: "0x0000000000000000000000000000000000000004"
						target_contract: "0x0290FB167208Af455bB137780163b7B7a9a10C16"
						abi_encoded_call: "0x00010304"
					}
				}`,
	},

	// build/bin/guardiand template governance-solana-call --chain-id solana --call-data 00010304 --governance-contract B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE
	{
		label:   "SolanaCall success",
		errText: "",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					solana_call: {
						chain_id: 1
						governance_contract: "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE"
						encoded_instruction: "00010304"
					}
				}`,
	},
	{
		label:   "SolanaCall invalid governance contract",
		errText: "failed to decode base58 governance contract address",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					solana_call: {
						chain_id: 1
						governance_contract: "0x0290FB167208Af455bB137780163b7B7a9a10C16"
						encoded_instruction: "00010304"
					}
				}`,
	},
	{
		label:   "SolanaCall invalid governance contract length",
		errText: "invalid governance contract address length (expected 32 bytes)",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					solana_call: {
						chain_id: 1
						governance_contract: "3HdmTyMzgniBchBeicyFxGxWaCLdq72XjUZ8YdxqUef"
						encoded_instruction: "00010304"
					}
				}`,
	},
	{
		label:   "SolanaCall invalid instruction",
		errText: "failed to decode instruction",
		prototext: `
				current_set_index: 4
				messages: {
					sequence: 315027427769585223
					nonce: 2920957782
					solana_call: {
						chain_id: 1
						governance_contract: "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE"
						encoded_instruction: "Hello, World!"
					}
				}`,
	},
}
