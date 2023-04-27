import * as web3s from "@solana/web3.js";
import { NETWORKS } from "./networks";
import { impossible, Payload, VAA } from "./vaa";
import base58 from "bs58";
import { postVaaSolanaWithRetry } from "@certusone/wormhole-sdk/lib/cjs/solana";
import {
  ChainName,
  CHAINS,
  CONTRACTS,
  SolanaChainName,
} from "@certusone/wormhole-sdk/lib/cjs/utils/consts";
import { hexToUint8Array } from "@certusone/wormhole-sdk";
import {
  createUpgradeContractInstruction as createWormholeUpgradeContractInstruction,
  createUpgradeGuardianSetInstruction,
} from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
import {
  createCompleteTransferNativeInstruction,
  createCompleteTransferWrappedInstruction,
  createCreateWrappedInstruction,
  createRegisterChainInstruction as createTokenBridgeRegisterChainInstruction,
  createUpgradeContractInstruction as createTokenBridgeUpgradeContractInstruction,
  deriveEndpointKey,
  getEndpointRegistration,
} from "@certusone/wormhole-sdk/lib/cjs/solana/tokenBridge";
import {
  createRegisterChainInstruction as createNFTBridgeRegisterChainInstruction,
  createUpgradeContractInstruction as createNFTBridgeUpgradeContractInstruction,
} from "@certusone/wormhole-sdk/lib/cjs/solana/nftBridge";
import { getEmitterAddress } from "./emitter"

export async function execute_solana(
  v: VAA<Payload>,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET",
  chain: SolanaChainName
) {
  let ix: web3s.TransactionInstruction;
  const connection = setupConnection(NETWORKS[network][chain].rpc);
  const bridgeId = new web3s.PublicKey(CONTRACTS[network][chain].core);
  const tokenBridgeId =
    CONTRACTS[network][chain].token_bridge &&
    new web3s.PublicKey(CONTRACTS[network][chain].token_bridge);
  const nftBridgeId =
    CONTRACTS[network][chain].nft_bridge &&
    new web3s.PublicKey(CONTRACTS[network][chain].nft_bridge);

  const from = web3s.Keypair.fromSecretKey(
    base58.decode(NETWORKS[network][chain].key)
  );

  switch (v.payload.module) {
    case "Core":
      if (bridgeId === undefined) {
        throw Error("core bridge contract is undefined");
      }
      switch (v.payload.type) {
        case "GuardianSetUpgrade":
          console.log("Submitting new guardian set");
          ix = createUpgradeGuardianSetInstruction(
            bridgeId,
            from.publicKey,
            vaa
          );
          break;
        case "ContractUpgrade":
          console.log("Upgrading core contract");
          ix = createWormholeUpgradeContractInstruction(
            bridgeId,
            from.publicKey,
            vaa
          );
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on solana")
        default:
          ix = impossible(v.payload);
      }
      break;
    case "NFTBridge":
      if (nftBridgeId === undefined) {
        throw Error("nft bridge contract is undefined");
      }
      switch (v.payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          ix = createNFTBridgeUpgradeContractInstruction(
            nftBridgeId,
            bridgeId,
            from.publicKey,
            vaa
          );
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on solana")
        case "RegisterChain":
          console.log("Registering chain");
          ix = createNFTBridgeRegisterChainInstruction(
            nftBridgeId,
            bridgeId,
            from.publicKey,
            vaa
          );
          break;
        case "Transfer":
          throw Error("Can't redeem NFTs from CLI");
        // TODO: what's the authority account? just bail for now
        default:
          ix = impossible(v.payload);
      }
      break;
    case "TokenBridge":
      if (tokenBridgeId === undefined) {
        throw Error("token bridge contract is undefined");
      }
      const payload = v.payload;
      switch (payload.type) {
        case "ContractUpgrade":
          console.log("Upgrading contract");
          ix = createTokenBridgeUpgradeContractInstruction(
            tokenBridgeId,
            bridgeId,
            from.publicKey,
            vaa
          );
          break;
        case "RecoverChainId":
          throw new Error("RecoverChainId not supported on solana")
        case "RegisterChain":
          console.log("Registering chain");
          ix = createTokenBridgeRegisterChainInstruction(
            tokenBridgeId,
            bridgeId,
            from.publicKey,
            vaa
          );
          break;
        case "Transfer":
          console.log("Completing transfer");
          if (payload.tokenChain === CHAINS[chain]) {
            ix = createCompleteTransferNativeInstruction(
              tokenBridgeId,
              bridgeId,
              from.publicKey,
              vaa
            );
          } else {
            ix = createCompleteTransferWrappedInstruction(
              tokenBridgeId,
              bridgeId,
              from.publicKey,
              vaa
            );
          }
          break;
        case "AttestMeta":
          console.log("Creating wrapped token");
          ix = createCreateWrappedInstruction(
            tokenBridgeId,
            bridgeId,
            from.publicKey,
            vaa
          );
          break;
        case "TransferWithPayload":
          throw Error("Can't complete payload 3 transfer from CLI");
        default:
          impossible(payload);
          break;
      }
      break;
    default:
      ix = impossible(v.payload);
  }

  // First upload the VAA
  await postVaaSolanaWithRetry(
    connection,
    async (tx) => {
      tx.partialSign(from);
      return tx;
    },
    bridgeId,
    from.publicKey,
    vaa,
  );

  // Then do the actual thing
  const transaction = new web3s.Transaction().add(ix);

  const signature = await web3s.sendAndConfirmTransaction(
    connection,
    transaction,
    [from],
    {
      skipPreflight: true,
    }
  );
  console.log("SIGNATURE", signature);
}

function setupConnection(rpc: string): web3s.Connection {
  return new web3s.Connection(rpc, "confirmed");
}

// query_registrations_solana queries the bridge contract for chain registrations.
// Solana does not support querying to see "What address is chain X registered for?" Instead
// we have to ask "Is chain X registered with address Y?" Therefore, we loop through all of the
// chains and query to see if the latest address defined in the const file is registered.
export async function query_registrations_solana(
  network: "MAINNET" | "TESTNET" | "DEVNET",
  module: "Core" | "NFTBridge" | "TokenBridge",
): Promise<Object> {
  let chain = "solana";
  let n = NETWORKS[network][chain];
  let contracts = CONTRACTS[network][chain];

  let target_contract: string;

  switch (module) {
    case "TokenBridge":
      target_contract = contracts.token_bridge;
      break;
    case "NFTBridge":
      target_contract = contracts.nft_bridge;
      break;
    default:
      throw new Error(`Invalid module: ${module}`);
  }
  
  if (!target_contract) {
    throw new Error(`Contract for ${module} on ${network} does not exist`);
  }

  console.log(`Querying the ${module} on ${network} ${chain} for registered chains.`);

  const connection = setupConnection(NETWORKS[network][chain].rpc);
  const programId = new web3s.PublicKey(target_contract);

  // Query the bridge registration for all the chains in parallel.
  const registrationsPromise = Promise.all(
    Object.entries(CHAINS)
      .filter(([c_name, _]) => c_name !== chain && c_name !== "unset")
      .map(async ([c_name, c_id]) => [c_name, await (async () => {
          let addr: string;
          if (module === "TokenBridge") {
            if (CONTRACTS[network][c_name].token_bridge === undefined) {
              return null;
            }
            addr = CONTRACTS[network][c_name].token_bridge;
          } else {
            if (CONTRACTS[network][c_name].nft_bridge === undefined) {
              return null;
            }
            addr = CONTRACTS[network][c_name].nft_bridge;
          }
          let emitter_addr = await getEmitterAddress(c_name as ChainName, addr);

          const endpoint = deriveEndpointKey(
            programId,
            c_id,
            hexToUint8Array(emitter_addr)
          );

          let result = null;
          try {
            await getEndpointRegistration(
              connection,
              endpoint
            );
            result = `0x` + emitter_addr;
          } catch {
            // Not logging anything because a chain not registered returns an error.
          }
        
          return result;
        })()
      ])
  )

  const registrations = await registrationsPromise;

  let results = {}
  for (let [c_name, queryResponse] of registrations) {
    if (queryResponse) {
      results[c_name] = queryResponse;
    }
  }
  return results;
}
