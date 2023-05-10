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
} from "@certusone/wormhole-sdk/lib/esm/solana/tokenBridge";
import {
  createUpgradeGuardianSetInstruction,
  createUpgradeContractInstruction as createWormholeUpgradeContractInstruction,
} from "@certusone/wormhole-sdk/lib/esm/solana/wormhole";
import {
  CHAINS,
  CONTRACTS,
  SolanaChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import * as web3s from "@solana/web3.js";
import base58 from "bs58";
import { NETWORKS } from "./consts";
import { Payload, VAA, impossible } from "./vaa";

export async function execute_solana(
  v: VAA<Payload>,
  vaa: Buffer,
  network: "MAINNET" | "TESTNET" | "DEVNET",
  chain: SolanaChainName
) {
  const { rpc, key } = NETWORKS[network][chain];
  if (!key) {
    throw Error(`No ${network} key defined for NEAR`);
  }

  if (!rpc) {
    throw Error(`No ${network} rpc defined for NEAR`);
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

const setupConnection = (rpc: string): web3s.Connection =>
  new web3s.Connection(rpc, "confirmed");
