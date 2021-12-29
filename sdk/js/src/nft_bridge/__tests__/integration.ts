import { parseUnits } from "@ethersproject/units";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { describe, expect, jest, test } from "@jest/globals";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import {
  Coins,
  LCDClient,
  MnemonicKey,
  Msg,
  MsgExecuteContract,
  StdFee,
  TxInfo,
  Wallet,
} from "@terra-money/terra.js";
import axios from "axios";
import { ethers } from "ethers";
import {
  approveEth,
  attestFromEth,
  attestFromSolana,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getEmitterAddressEth,
  getEmitterAddressTerra,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  parseSequenceFromLogTerra,
  postVaaSolana,
} from "../..";
import {
  redeemOnEth,
  redeemOnSolana,
  redeemOnTerra,
  transferFromEth,
  transferFromTerra,
  transferFromSolana,
  getIsTransferCompletedEth,
  getIsTransferCompletedSolana,
  getIsTransferCompletedTerra,
} from "../";
import getSignedVAAWithRetry from "../../rpc/getSignedVAAWithRetry";
import { setDefaultWasm } from "../../solana/wasm";
import {
  ETH_CORE_BRIDGE_ADDRESS,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  ETH_NFT_BRIDGE_ADDRESS,
  SOLANA_CORE_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOLANA_NFT_BRIDGE_ADDRESS,
  TERRA_CHAIN_ID,
  TERRA_GAS_PRICES_URL,
  TERRA_NODE_URL,
  TERRA_PRIVATE_KEY,
  TERRA_NFT_BRIDGE_ADDRESS,
  TEST_ERC721,
  TEST_CW721,
  TEST_SOLANA_TOKEN,
  WORMHOLE_RPC_HOSTS,
} from "./consts";
import { ExtensionNetworkOnlyWalletProvider, TxResult } from "@terra-money/wallet-provider";

setDefaultWasm("node");

jest.setTimeout(60000);

const lcd = new LCDClient({
  URL: TERRA_NODE_URL,
  chainID: TERRA_CHAIN_ID,
});
const terraWallet: Wallet = lcd.wallet(new MnemonicKey({
  mnemonic: TERRA_PRIVATE_KEY,
}));

async function getGasPrices() {
  return axios
    .get(TERRA_GAS_PRICES_URL)
    .then((result) => result.data);
}

async function waitForTerraExecution(txHash: string): Promise<TxInfo> {
  let info: TxInfo | undefined = undefined;
  while (!info) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      info = await lcd.tx.txInfo(txHash);
    } catch (e) {
      console.error(e);
    }
  }
  if (info.code !== undefined) {
    // error code
    throw new Error(
      `Tx ${txHash}: error code ${info.code}: ${info.raw_log}`
    );
  }
  return info;
}

async function estimateTerraFee(gasPrices: Coins.Input, msgs: Msg[]): Promise<StdFee> {
  const feeEstimate = await lcd.tx.estimateFee(
    terraWallet.key.accAddress,
    msgs,
    {
      memo: "localhost",
      feeDenoms: ["uluna"],
      gasPrices,
    }
  );
  return feeEstimate;
}

describe("Integration Tests", () => {
  describe("Ethereum to Terra", () => {
    test("Send Ethereum ERC-721 to Terra", (done) => {
      (async () => {
        try {
          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          // transfer NFT
          const receipt = await transferFromEth(
            ETH_NFT_BRIDGE_ADDRESS,
            signer,
            TEST_ERC721,
            0,
            CHAIN_ID_TERRA,
            hexToUint8Array(
              nativeToHexString(terraWallet.key.accAddress, CHAIN_ID_TERRA) || ""
            ));
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_NFT_BRIDGE_ADDRESS);
          // poll until the guardian(s) witness and sign the vaa
          const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ETH,
            emitterAddress,
            sequence,
            {
              transport: NodeHttpTransport(),
            }
          );
          expect(
            await getIsTransferCompletedTerra(
              TERRA_NFT_BRIDGE_ADDRESS,
              signedVAA,
              terraWallet.key.accAddress,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(false);
          const msg = await redeemOnTerra(
            TERRA_NFT_BRIDGE_ADDRESS,
            terraWallet.key.accAddress,
            signedVAA
          );
          const gasPrices = await getGasPrices();
          const tx = await terraWallet.createAndSignTx({
            msgs: [msg],
            memo: "localhost",
            feeDenoms: ["uluna"],
            gasPrices,
            fee: await estimateTerraFee(gasPrices, [msg]),
          });
          await lcd.tx.broadcast(tx);
          expect(
            await getIsTransferCompletedTerra(
              TERRA_NFT_BRIDGE_ADDRESS,
              signedVAA,
              terraWallet.key.accAddress,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(true);
          provider.destroy();
          done();
        } catch (e) {
          console.error(e);
          done(`An error occured while trying to transfer from Ethereum to Solana: ${e}`);
        }
      })();
    });
    test("Send Terra CW721 to Ethereum", (done) => {
      (async () => {
        try {
          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          // transfer NFT
          const msgs = await transferFromTerra(
            terraWallet.key.accAddress,
            TERRA_NFT_BRIDGE_ADDRESS,
            TEST_CW721,
            "0",
            CHAIN_ID_ETH,
            hexToUint8Array(
              nativeToHexString(await signer.getAddress(), CHAIN_ID_ETH) || ""
            ));
          const gasPrices = await getGasPrices();
          let tx = await terraWallet.createAndSignTx({
            msgs: msgs,
            memo: "test",
            feeDenoms: ["uluna"],
            gasPrices,
            fee: await estimateTerraFee(gasPrices, msgs)
          });
          const txresult = await lcd.tx.broadcast(tx);
          // get the sequence from the logs (needed to fetch the vaa)
          const info = await waitForTerraExecution(txresult.txhash);
          const sequence = parseSequenceFromLogTerra(info);
          const emitterAddress = await getEmitterAddressTerra(TERRA_NFT_BRIDGE_ADDRESS);
          // poll until the guardian(s) witness and sign the vaa
          const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_TERRA,
            emitterAddress,
            sequence,
            {
              transport: NodeHttpTransport(),
            }
          );
          expect(
            await getIsTransferCompletedEth(
              ETH_NFT_BRIDGE_ADDRESS,
              provider,
              signedVAA,
            )
          ).toBe(false);
          await redeemOnEth(
            ETH_NFT_BRIDGE_ADDRESS,
            signer,
            signedVAA
          );
          expect(
            await getIsTransferCompletedEth(
              ETH_NFT_BRIDGE_ADDRESS,
              provider,
              signedVAA,
            )
          ).toBe(true);
          provider.destroy();
          done();
        } catch (e) {
          console.error(e);
          done(`An error occured while trying to transfer from Ethereum to Solana: ${e}`);
        }
      })();
    });
  })
});
