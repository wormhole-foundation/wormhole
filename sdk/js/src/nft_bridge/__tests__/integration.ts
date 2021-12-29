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
  LCDClient,
  MnemonicKey,
  MsgExecuteContract,
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
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  postVaaSolana,
} from "../..";
import {
  redeemOnEth,
  redeemOnSolana,
  redeemOnTerra,
  transferFromEth,
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
import { ExtensionNetworkOnlyWalletProvider } from "@terra-money/wallet-provider";

setDefaultWasm("node");

jest.setTimeout(60000);

describe("Integration Tests", () => {
  describe("Ethereum to Solana", () => {
    test("Send Ethereum ERC-721 to Terra", (done) => {
      (async () => {
        try {
          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const wallet = lcd.wallet(mk);
          // transfer NFT
          const receipt = await transferFromEth(
            ETH_NFT_BRIDGE_ADDRESS,
            signer,
            TEST_ERC721,
            0,
            CHAIN_ID_TERRA,
            hexToUint8Array(
              nativeToHexString(wallet.key.accAddress, CHAIN_ID_TERRA) || ""
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
              wallet.key.accAddress,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(false);
          const msg = await redeemOnTerra(
            TERRA_NFT_BRIDGE_ADDRESS,
            wallet.key.accAddress,
            signedVAA
          );
          const gasPrices = await axios
            .get(TERRA_GAS_PRICES_URL)
            .then((result) => result.data);
          const feeEstimate = await lcd.tx.estimateFee(
            wallet.key.accAddress,
            [msg],
            {
              memo: "localhost",
              feeDenoms: ["uluna"],
              gasPrices,
            }
          );
          const tx = await wallet.createAndSignTx({
            msgs: [msg],
            memo: "localhost",
            feeDenoms: ["uluna"],
            gasPrices,
            fee: feeEstimate,
          });
          await lcd.tx.broadcast(tx);
          expect(
            await getIsTransferCompletedTerra(
              TERRA_NFT_BRIDGE_ADDRESS,
              signedVAA,
              wallet.key.accAddress,
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
    })
  })
});
