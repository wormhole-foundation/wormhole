import { postVaaSolanaWithRetry } from "@certusone/wormhole-sdk/lib/esm/solana";
import {
  createRegisterChainInstruction as createNFTBridgeRegisterChainInstruction,
  createUpgradeContractInstruction as createNFTBridgeUpgradeContractInstruction,
} from "@certusone/wormhole-sdk/lib/esm/solana/nftBridge";
import {
  createCompleteTransferNativeInstruction,
  createCompleteTransferWrappedInstruction,
  createCreateWrappedInstruction,
  createRegisterChainInstruction as createTokenBridgeRegisterChainInstruction,
  createUpgradeContractInstruction as createTokenBridgeUpgradeContractInstruction,
  deriveEndpointKey,
  getEndpointRegistration,
} from "@certusone/wormhole-sdk/lib/esm/solana/tokenBridge";
import {
  createUpgradeGuardianSetInstruction,
  createUpgradeContractInstruction as createWormholeUpgradeContractInstruction,
} from "@certusone/wormhole-sdk/lib/esm/solana/wormhole";
import {
  CHAINS,
  CONTRACTS,
  ChainName,
  Network,
  SolanaChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import * as web3s from "@solana/web3.js";
import base58 from "bs58";
import { NETWORKS } from "./consts";
import { Payload, VAA, impossible } from "./vaa";
import { getEmitterAddress } from "./emitter";
import {
  transferFromSolana,
  transferNativeSol,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import {
  hexToUint8Array,
  tryNativeToUint8Array,
} from "@certusone/wormhole-sdk/lib/esm/utils";
import { PublicKey } from "@solana/web3.js";
import { getAssociatedTokenAddress } from "@solana/spl-token";

export async function execute_solana(
  v: VAA<Payload>,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET",
  chain: SolanaChainName
) {
  const { rpc, key } = NETWORKS[network][chain];
  if (!key) {
    throw Error(`No ${network} key defined for ${chain}`);
  }

  if (!rpc) {
    throw Error(`No ${network} rpc defined for ${chain}`);
  }

  const connection = setupConnection(rpc);
  const from = web3s.Keypair.fromSecretKey(base58.decode(key));

  const contracts = CONTRACTS[network][chain];
  if (!contracts.core) {
    throw new Error(`Core bridge address not defined for ${chain} ${network}`);
  }

  if (!contracts.nft_bridge) {
    throw new Error(`NFT bridge address not defined for ${chain} ${network}`);
  }

  if (!contracts.token_bridge) {
    throw new Error(`Token bridge address not defined for ${chain} ${network}`);
  }

  const bridgeId = new web3s.PublicKey(contracts.core);
  const tokenBridgeId = new web3s.PublicKey(contracts.token_bridge);
  const nftBridgeId = new web3s.PublicKey(contracts.nft_bridge);

  let ix: web3s.TransactionInstruction;
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
          throw new Error("RecoverChainId not supported on solana");
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
          throw new Error("RecoverChainId not supported on solana");
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
          throw new Error("RecoverChainId not supported on solana");
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
    case "WormholeRelayer":
      throw Error("Wormhole Relayer not supported on Solana");
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
    vaa
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

export async function transferSolana(
  srcChain: SolanaChainName,
  dstChain: ChainName,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const { key } = NETWORKS[network][srcChain];
  if (!key) {
    throw Error(`No ${network} key defined for ${srcChain}`);
  }

  const connection = setupConnection(rpc);
  const keypair = web3s.Keypair.fromSecretKey(base58.decode(key));

  const { core, token_bridge } = CONTRACTS[network][srcChain];
  if (!core) {
    throw new Error(
      `Core bridge address not defined for ${srcChain} ${network}`
    );
  }
  if (!token_bridge) {
    throw new Error(
      `Token bridge address not defined for ${srcChain} ${network}`
    );
  }

  const bridgeId = new web3s.PublicKey(core);
  const tokenBridgeId = new web3s.PublicKey(token_bridge);
  const payerAddress = keypair.publicKey.toString();

  let transaction;
  if (tokenAddress === "native") {
    transaction = await transferNativeSol(
      connection,
      bridgeId,
      tokenBridgeId,
      payerAddress,
      BigInt(amount),
      tryNativeToUint8Array(dstAddress, dstChain),
      dstChain
    );
  } else {
    // find the associated token account
    const fromAddress = (
      await getAssociatedTokenAddress(
        new PublicKey(tokenAddress),
        keypair.publicKey
      )
    ).toString();
    transaction = await transferFromSolana(
      connection,
      bridgeId,
      tokenBridgeId,
      payerAddress,
      fromAddress,
      tokenAddress, // mintAddress
      BigInt(amount),
      tryNativeToUint8Array(dstAddress, dstChain),
      dstChain
    );
  }

  // sign, send, and confirm transaction
  transaction.partialSign(keypair);
  const signature = await connection.sendRawTransaction(
    transaction.serialize()
  );
  await connection.confirmTransaction(signature);
  const info = await connection.getTransaction(signature);
  if (!info) {
    throw new Error("An error occurred while fetching the transaction info");
  }
  console.log("SIGNATURE", signature);
}

const setupConnection = (rpc: string): web3s.Connection =>
  new web3s.Connection(rpc, "confirmed");

// queryRegistrationsSolana queries the bridge contract for chain registrations.
// Solana does not support querying to see "What address is chain X registered for?" Instead
// we have to ask "Is chain X registered with address Y?" Therefore, we loop through all of the
// chains and query to see if the latest address defined in the const file is registered.
export async function queryRegistrationsSolana(
  network: Network,
  module: "Core" | "NFTBridge" | "TokenBridge"
): Promise<Object> {
  const chain = "solana" as ChainName;
  const n = NETWORKS[network][chain];
  const contracts = CONTRACTS[network][chain];

  let targetAddress: string | undefined;

  switch (module) {
    case "TokenBridge":
      targetAddress = contracts.token_bridge;
      break;
    case "NFTBridge":
      targetAddress = contracts.nft_bridge;
      break;
    default:
      throw new Error(`Invalid module: ${module}`);
  }

  if (!targetAddress) {
    throw new Error(`Contract for ${module} on ${network} does not exist`);
  }

  if (n === undefined || n.rpc === undefined) {
    throw new Error(`RPC for ${module} on ${network} does not exist`);
  }

  const connection = setupConnection(n.rpc);
  const programId = new web3s.PublicKey(targetAddress);

  // Query the bridge registration for all the chains in parallel.
  const registrations: (string | null)[][] = await Promise.all(
    Object.entries(CHAINS)
      .filter(([cname, _]) => cname !== chain && cname !== "unset")
      .map(async ([cstr, cid]) => [
        cstr,
        await (async () => {
          let cname = cstr as ChainName;
          let addr: string | undefined;
          if (module === "TokenBridge") {
            addr = CONTRACTS[network][cname].token_bridge;
          } else {
            addr = CONTRACTS[network][cname].nft_bridge;
          }
          if (addr === undefined) {
            return null;
          }
          let emitter_addr = await getEmitterAddress(cname as ChainName, addr);

          const endpoint = deriveEndpointKey(
            programId,
            cid,
            hexToUint8Array(emitter_addr)
          );

          let result: string | null = null;
          try {
            await getEndpointRegistration(connection, endpoint);
            result = emitter_addr;
          } catch {
            // Not logging anything because a chain not registered returns an error.
          }

          return result as string;
        })(),
      ])
  );

  const results: { [key: string]: string } = {};
  for (let [cname, queryResponse] of registrations) {
    if (cname && queryResponse) {
      results[cname] = queryResponse;
    }
  }
  return results;
}
