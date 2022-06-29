import * as web3s from '@solana/web3.js'
import { NETWORKS } from "./networks";
import { impossible, Payload, VAA } from "./vaa";
import base58 from "bs58";
import { CHAIN_ID_SOLANA, importCoreWasm, importNftWasm, importTokenWasm, ixFromRust } from "@certusone/wormhole-sdk";
import { CONTRACTS } from "@certusone/wormhole-sdk"
import { postVaaSolanaWithRetry } from "@certusone/wormhole-sdk"

export async function execute_solana(
  v: VAA<Payload>,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET"
) {
  let ix: web3s.TransactionInstruction
  let connection = setupConnection(NETWORKS[network].solana.rpc)
  let bridge_id = new web3s.PublicKey(CONTRACTS[network].solana.core)
  let token_bridge_id = new web3s.PublicKey(CONTRACTS[network].solana.token_bridge)
  let nft_bridge_id = new web3s.PublicKey(CONTRACTS[network].solana.nft_bridge)

  let from = web3s.Keypair.fromSecretKey(base58.decode(NETWORKS[network].solana.key))

  switch (v.payload.module) {
    case "Core":
      const bridge = await importCoreWasm()
      switch (v.payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set")
          ix = bridge.update_guardian_set_ix(bridge_id.toString(), from.publicKey.toString(), vaa);
          break
        case "ContractUpgrade":
          console.log("Upgrading core contract")
          ix = bridge.upgrade_contract_ix(bridge_id.toString(), from.publicKey.toString(), from.publicKey.toString(), vaa);
          break
        default:
          ix = impossible(v.payload)
      }
      break
    case "NFTBridge":
      const nft_bridge = await importNftWasm()
      switch (v.payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract")
          ix = nft_bridge.upgrade_contract_ix(nft_bridge_id.toString(), bridge_id.toString(), from.publicKey.toString(), from.publicKey.toString(), vaa);
          break
        case "RegisterChain":
          console.log("Registering chain")
          ix = nft_bridge.register_chain_ix(nft_bridge_id.toString(), bridge_id.toString(), from.publicKey.toString(), vaa);
          break
        case "Transfer":
          throw Error("Can't redeem NFTs from CLI")
        // TODO: what's the authority account? just bail for now
        default:
          ix = impossible(v.payload)
      }
      break
    case "TokenBridge":
      const token_bridge = await importTokenWasm()
      const payload = v.payload;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract")
          ix = token_bridge.upgrade_contract_ix(token_bridge_id.toString(), bridge_id.toString(), from.publicKey.toString(), from.publicKey.toString(), vaa)
          break
        case "RegisterChain":
          console.log("Registering chain")
          ix = token_bridge.register_chain_ix(token_bridge_id.toString(), bridge_id.toString(), from.publicKey.toString(), vaa)
          break
        case "Transfer":
          console.log("Completing transfer")
          if (payload.tokenChain === CHAIN_ID_SOLANA) {
            ix = token_bridge.complete_transfer_native_ix(token_bridge_id.toString(), bridge_id.toString(), from.publicKey.toString(), vaa)
          } else {
            ix = token_bridge.complete_transfer_wrapped_ix(token_bridge_id.toString(), bridge_id.toString(), from.publicKey.toString(), vaa)
          }
          break
        case "AttestMeta":
          console.log("Creating wrapped token")
          ix = token_bridge.create_wrapped_ix(token_bridge_id.toString(), bridge_id.toString(), from.publicKey.toString(), vaa)
          break
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI")
        default:
          impossible(payload)
          break

      }
      break
    default:
      ix = impossible(v.payload)
  }

  // First upload the VAA
  await postVaaSolanaWithRetry(connection,
    async (tx) => {
      tx.partialSign(from)
      return tx
    },
    bridge_id.toString(), from.publicKey.toString(), vaa, 5)

  // Then do the actual thing
  let transaction = new web3s.Transaction().add(ixFromRust(ix))

  let signature = await web3s.sendAndConfirmTransaction(
    connection,
    transaction,
    [from],
    {
      skipPreflight: true
    }
  )
  console.log('SIGNATURE', signature)
}

function setupConnection(rpc: string): web3s.Connection {
  return new web3s.Connection(
    rpc,
    'confirmed',
  )
}
