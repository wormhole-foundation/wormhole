import {
  Bridge__factory,
  CHAIN_ID_CELO,
  CHAIN_ID_FANTOM,
  CHAIN_ID_KLAYTN,
  CHAIN_ID_POLYGON,
  getIsTransferCompletedEth,
  hexToUint8Array,
  redeemOnEth,
  redeemOnEthNative,
  importCoreWasm,
  parseTransferPayload
} from "@certusone/wormhole-sdk";
import { BigNumber, ContractReceipt, Contract } from "ethers";
//"/Users/leo/Developer/wormhole/relayer/spy_relayer/src/xRaydium/node_modules/hardhat/internal/lib/hardhat-lib"
import { ChainConfigInfo } from "../configureEnv";
import { getScopedLogger, ScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import { CeloProvider, CeloWallet } from "@celo-tools/celo-ethers-wrapper";
import * as lib from "../lib/lib";
import fs from "fs"
import * as types from "../xRaydium/solana-proxy/generated_client/types";
import "@nomiclabs/hardhat-ethers";
import {ethers} from "hardhat";
import xRaydium_abi from "../utils/xRaydium_abi.json";
//import * as addrs from "../xRaydium/addrs";

//import ethers from "hardhat";
//import {ethers} from "../xRaydium/node_modules/hardhat/internal/lib/hardhat-lib"
//xRaydium/node_modules/hardhat/internal/lib/hardhat-lib

export function newProvider(
  url: string,
  batch: boolean = false
//@ts-ignore
): ethers.providers.JsonRpcProvider | ethers.providers.JsonRpcBatchProvider {
  // only support http(s), not ws(s) as the websocket constructor can blow up the entire process
  // it uses a nasty setTimeout(()=>{},0) so we are unable to cleanly catch its errors
  if (url.startsWith("http")) {
    if (batch) {
      return new ethers.providers.JsonRpcBatchProvider(url);
    }
    return new ethers.providers.JsonRpcProvider(url);
  }
  throw new Error("url does not start with http/https!");
}

export async function relayEVM(
  chainConfigInfo: ChainConfigInfo,
  signedVAA: string,
  unwrapNative: boolean,
  checkOnly: boolean,
  walletPrivateKey: string,
  relayLogger: ScopedLogger,
  metrics: PromHelper
) {
  const logger = getScopedLogger(
    ["evm", chainConfigInfo.chainName],
    relayLogger
  );
  const signedVaaArray = hexToUint8Array(signedVAA);
  let provider;
  let signer;
  if (chainConfigInfo.chainId === CHAIN_ID_CELO) {
    provider = new CeloProvider(chainConfigInfo.nodeUrl);
    await provider.ready;
    signer = new CeloWallet(walletPrivateKey, provider);
  } else {
    provider = newProvider(chainConfigInfo.nodeUrl);
    signer = new ethers.Wallet(walletPrivateKey, provider);
  }

  logger.debug("Checking to see if vaa has already been redeemed.");
  const alreadyRedeemed = await getIsTransferCompletedEth(
    chainConfigInfo.tokenBridgeAddress,
    provider,
    signedVaaArray
  );

  if (alreadyRedeemed) {
    logger.info("VAA has already been redeemed!");
    return { redeemed: true, result: "already redeemed" };
  }
  if (checkOnly) {
    return { redeemed: false, result: "not redeemed" };
  }
  if (unwrapNative) {
    logger.info(
      "Will redeem and unwrap using pubkey: %s",
      await signer.getAddress()
    );
  } else {
    logger.info("Will redeem using pubkey: %s", await signer.getAddress());
  }
  const { parse_vaa } = await importCoreWasm();
  const parsed = parse_vaa(signedVaaArray);
  let transferPayload: any = parseTransferPayload(
    Buffer.from(parsed["payload"]),
  );
  transferPayload["payload3"] = Buffer.from(parsed["payload"].slice(133));
  logger.info(parsed, "Parsed VAA");

  // TODO: check sender of payload 3 is solana proxy via sender field 

  // const addrs = JSON.parse(String(fs.readFile("../xRaydium/addrs")));

  //const addrs = JSON.parse(String(await fs.promises.readFile("../xRaydium" + "/addrs")));

  //const XRaydiumBridge = await ethers.getContractFactory(xRaydium_abi.abi);

  //const contract = await XRaydiumBridge.attach("0xD768Ffbc3904F89f53Af2A640e3b6C640D85D6B9");

  const contract = new ethers.Contract('0xD768Ffbc3904F89f53Af2A640e3b6C640D85D6B9', xRaydium_abi.abi, provider);

  const response = types.Response.layout().decode(transferPayload.payload3);
  logger.info(response, "Response");

  const receipt = await (await contract.connect(signer).receiveResponse(signedVaaArray)).wait();
  
  logger.info("=============done redeem responses to EVM!!!...!!!")

  metrics.incSuccesses(chainConfigInfo.chainId);
  return { redeemed: true, result: "redeemed"};
}


