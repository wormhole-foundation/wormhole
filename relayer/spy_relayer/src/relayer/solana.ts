import {
  getIsTransferCompletedSolana,
  hexToUint8Array,
} from "@certusone/wormhole-sdk";
import * as web3 from "@solana/web3.js";
import { ChainConfigInfo } from "../configureEnv";
import { getScopedLogger, ScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import { chainConfigToEvmProviderAndSigner } from "./evm";
import * as xApp from "../xRaydium/scripts/lib";
import * as relay from "../xRaydium/scripts/relay";
import * as ethers from "ethers";
import * as raydiumSdk from "@raydium-io/raydium-sdk";

const MAX_VAA_UPLOAD_RETRIES_SOLANA = 5;

function loggableChainConfig({
  walletPrivateKey,
  solanaPrivateKey,
  ...o
}: ChainConfigInfo): Omit<
  Omit<ChainConfigInfo, "solanaPrivateKey">,
  "walletPrivateKey"
> {
  return o;
}

export async function relaySolana(
  chainConfigInfo: ChainConfigInfo,
  emitterChainConfigInfo: ChainConfigInfo,
  signedVAAString: string,
  checkOnly: boolean,
  walletPrivateKey: Uint8Array,
  relayLogger: ScopedLogger,
  metrics: PromHelper
) {
  console.log("signedVAAString: ", signedVAAString);
  const logger = getScopedLogger(["solana"], relayLogger);
  console.log(
    "relaySolana chainConfigInfo: ",
    loggableChainConfig(chainConfigInfo)
  );
  //TODO native transfer & create associated token account
  //TODO close connection
  const signedVaaArray = hexToUint8Array(signedVAAString);
  const connection = new web3.Connection(chainConfigInfo.nodeUrl, "confirmed");
  if (!chainConfigInfo.bridgeAddress) {
    // This should never be the case, as enforced by createSolanaChainConfig
    return { redeemed: false, result: null };
  }

  console.log("==============in relaySolana.ts==============");
  console.log(
    "chainConfigInfo.tokenBridgeAddress: ",
    chainConfigInfo.tokenBridgeAddress
  );
  const alreadyRedeemed = await getIsTransferCompletedSolana(
    chainConfigInfo.tokenBridgeAddress,
    signedVaaArray,
    connection
  );
  //@ts-ignore
  const { transfer, baseVAA } = await xApp.parseTransferTokenWithPayload(
    signedVaaArray
  );

  const { signer, provider } = await chainConfigToEvmProviderAndSigner(
    emitterChainConfigInfo
  );
  const addrs = await xApp.loadAddrs();
  let ctx: xApp.Context;
  if (process.env.ENV_TYPE === "DEV_NET") {
    ctx = xApp.getDevNetCtx(
      signer,
      emitterChainConfigInfo.chainId,
      xApp._undef(
        emitterChainConfigInfo.walletPrivateKey,
        "expected emitter chain to have wallet private key"
      )[0],
      addrs.fuji.XRaydiumBridge,
      provider
    );
  } else {
    const evmPrivateKey = xApp._undef(
      emitterChainConfigInfo.walletPrivateKey
    )[0];
    ctx = xApp.getAvaxMainnetCtx(
      emitterChainConfigInfo.xRaydiumAddress,
      evmPrivateKey,
      walletPrivateKey,
      chainConfigInfo.xRaydiumAddress
    );
  }
  console.log(ctx.sol.payer.publicKey.toBase58());

  const header = await xApp.parseHeaderFromPayload3(transfer.payload3, true);
  const escrowState = await xApp.tryFetchEscrowState(
    ctx.sol,
    transfer,
    header,
    {
      silent: true,
      retries: 2,
    }
  );
  if (
    alreadyRedeemed &&
    escrowState &&
    escrowState.marker.kind === "Completed" &&
    escrowState.inputTokens.every(
      (t) => t.hasReturned.kind !== "NotReturned"
    ) &&
    escrowState.outputTokens.every((t) => t.hasReturned.kind !== "NotReturned")
  ) {
    logger.info("VAA has already been redeemed!");
    return { redeemed: true, result: "already redeemed" };
  }
  if (checkOnly) {
    return { redeemed: false, result: "not redeemed" };
  }

  await relay.relayToSolana(ctx, signedVaaArray, baseVAA, transfer);

  logger.info("\n\n============= Done relaying to solana ============\n\n");

  return { redeemed: true, result: "redeemed" };
}

export const mainnetSolanaRPC =
  "https://raydium.rpcpool.com/c642b692bb7f2de7ddf65b4d3b16";

export const basePIDS = {
  LIQUIDITY_PROGRAM_ID_V4: raydiumSdk.LIQUIDITY_PROGRAM_ID_V4,
  SERUM_PROGRAM_ID_V3: raydiumSdk.SERUM_PROGRAM_ID_V3,
  wormholeRPC: "https://wormhole-v2-mainnet-api.certus.one",
  tokenBridgeSolana: new web3.PublicKey(
    "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb"
  ),
  coreBridgeSolana: new web3.PublicKey(
    "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
  ),
};

export const AvaxPIDS: Omit<
  Omit<xApp.PIDS, "xRaydiumEvmAddr">,
  "solanaProxy"
> = {
  ...basePIDS,
  coreBridgeEvm: "0x54a8e5f9c4CbA08F9943965859F6c34eAF03E26c",
  tokenBridgeEvm: "0x0e082F06FF657D94310cB8cE8B0D9a04541d8052",
};

export const EthereumPIDS = {
  coreBridgeEvm: "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B",
  tokenBridgeEvm: "0x3ee18B2214AFF97000D974cf647E7C347E8fa585",
};
