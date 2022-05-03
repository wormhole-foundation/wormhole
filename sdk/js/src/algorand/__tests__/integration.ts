import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { describe, expect, jest, test } from "@jest/globals";
import { LCDClient, MnemonicKey } from "@terra-money/terra.js";
import algosdk, {
  Account,
  decodeAddress,
  getApplicationAddress,
  makeApplicationCallTxnFromObject,
  OnApplicationComplete,
  waitForConfirmation,
} from "algosdk";
import { ethers } from "ethers";
import { parseUnits } from "ethers/lib/utils";
import { textToUint8Array } from "../..";
import { parseSequenceFromLogAlgorand } from "../../";
import {
  getEmitterAddressAlgorand,
  getEmitterAddressEth,
  getEmitterAddressTerra,
} from "../../bridge";
import {
  parseSequenceFromLogEth,
  parseSequenceFromLogTerra,
} from "../../bridge/parseSequenceFromLog";
import { TokenImplementation__factory } from "../../ethers-contracts";
import getSignedVAAWithRetry from "../../rpc/getSignedVAAWithRetry";
import { setDefaultWasm } from "../../solana/wasm";
import {
  approveEth,
  attestFromAlgorand,
  attestFromEth,
  attestFromTerra,
  createWrappedOnAlgorand,
  createWrappedOnEth,
  getForeignAssetEth,
  getIsTransferCompletedAlgorand,
  getIsTransferCompletedEth,
  getIsTransferCompletedTerra,
  getOriginalAssetAlgorand,
  redeemOnAlgorand,
  redeemOnEth,
  redeemOnTerra,
  transferFromAlgorand,
  transferFromEth,
  transferFromTerra,
  updateWrappedOnEth,
  WormholeWrappedInfo,
} from "../../token_bridge";
import {
  ETH_CORE_BRIDGE_ADDRESS,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  ETH_TOKEN_BRIDGE_ADDRESS,
  TERRA_CHAIN_ID,
  TERRA_GAS_PRICES_URL,
  TERRA_NODE_URL,
  TERRA_PRIVATE_KEY,
  TERRA_TOKEN_BRIDGE_ADDRESS,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "../../token_bridge/__tests__/consts";
import {
  getSignedVAABySequence,
  queryBalanceOnTerra,
  waitForTerraExecution,
} from "../../token_bridge/__tests__/helpers";
import {
  CHAIN_ID_ALGORAND,
  CHAIN_ID_ETH,
  CHAIN_ID_TERRA,
  hexToUint8Array,
  nativeToHexString,
  uint8ArrayToHex,
} from "../../utils";
import { safeBigIntToNumber } from "../../utils/bigint";
import { _parseVAAAlgorand } from "../Algorand";
import {
  createAsset,
  getAlgoClient,
  getBalance,
  getBalances,
  getForeignAssetFromVaaAlgorand,
  getTempAccounts,
  signSendAndConfirmAlgorand,
} from "./testHelpers";

const CORE_ID = BigInt(4);
const TOKEN_BRIDGE_ID = BigInt(6);

setDefaultWasm("node");

jest.setTimeout(120000);

describe("Integration Tests", () => {
  describe("Algorand tests", () => {
    test("Algorand transfer native ALGO to Eth and back again", (done) => {
      (async () => {
        try {
          console.log("Starting attestation...");
          const client: algosdk.Algodv2 = getAlgoClient();
          const tempAccts: Account[] = await getTempAccounts();
          const numAccts: number = tempAccts.length;
          expect(numAccts).toBeGreaterThan(0);
          const wallet: Account = tempAccts[0];

          let accountInfo = await client.accountInformation(wallet.addr).do();
          console.log("Account balance: %d microAlgos", accountInfo.amount);

          // Asset Index of native ALGO is 0
          const AlgoIndex = BigInt(0);
          console.log("Testing attestFromAlgorand...");
          const b = await getBalances(client, wallet.addr);
          console.log("balances", b);
          const txs = await attestFromAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            wallet.addr,
            AlgoIndex
          );
          const result = await signSendAndConfirmAlgorand(client, txs, wallet);
          const sn = parseSequenceFromLogAlgorand(result);

          // Now, try to send a NOP
          console.log("Start of NOP...");
          const suggParams: algosdk.SuggestedParams = await client
            .getTransactionParams()
            .do();
          const nopTxn = makeApplicationCallTxnFromObject({
            from: wallet.addr,
            appIndex: safeBigIntToNumber(TOKEN_BRIDGE_ID),
            onComplete: OnApplicationComplete.NoOpOC,
            appArgs: [textToUint8Array("nop")],
            suggestedParams: suggParams,
          });
          const resp = await client
            .sendRawTransaction(nopTxn.signTxn(wallet.sk))
            .do();
          console.log("resp", resp);
          const response = await waitForConfirmation(client, resp.txId, 1);
          console.log("End of NOP");
          // End of NOP

          console.log("Getting emitter address...");
          const emitterAddr = getEmitterAddressAlgorand(TOKEN_BRIDGE_ID);
          const { vaaBytes } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ALGORAND,
            emitterAddr,
            sn,
            { transport: NodeHttpTransport() }
          );
          const pvaa = _parseVAAAlgorand(vaaBytes);
          console.log("VAA for createWrappedOnEth:", pvaa);
          const provider = new ethers.providers.WebSocketProvider(
            ETH_NODE_URL
          ) as any;
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          let success: boolean = true;
          try {
            const cr = await createWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer,
              vaaBytes
            );
          } catch (e) {
            console.log(
              "createWrappedOnEth() failed.  Trying updateWrappedOnEth()..."
            );
            success = false;
          }
          if (!success) {
            console.log("using updateWrappedOnEth...");
            try {
              const cr = await updateWrappedOnEth(
                ETH_TOKEN_BRIDGE_ADDRESS,
                signer,
                vaaBytes
              );
              success = true;
            } catch (e) {
              console.error("failed to updateWrappedOnEth", e);
            }
          }
          console.log("Attestation is complete...");
          console.log("Starting transfer to Eth...");
          // Check wallet
          const a = parseInt(AlgoIndex.toString());
          const originAssetHex = (
            "0000000000000000000000000000000000000000000000000000000000000000" +
            a.toString(16)
          ).slice(-64);
          console.log(
            "calling getForeignAssetEth",
            ETH_TOKEN_BRIDGE_ADDRESS,
            CHAIN_ID_ALGORAND,
            originAssetHex
          );
          const foreignAsset = await getForeignAssetEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            provider,
            CHAIN_ID_ALGORAND,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          console.log("foreignAsset", foreignAsset);
          let token = TokenImplementation__factory.connect(
            foreignAsset,
            signer
          );

          // Get initial balance on ethereum
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const initialBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const initialBalOnEthInt = parseInt(initialBalOnEth._hex);
          console.log("Balance on Eth before transfer = ", initialBalOnEthInt);

          // Get initial balance on Algorand
          let algoWalletBals: Map<number, number> = await getBalances(
            client,
            wallet.addr
          );
          console.log("algoWalletBals:", algoWalletBals);
          const startingAlgoBal = algoWalletBals.get(
            safeBigIntToNumber(AlgoIndex)
          );
          if (!startingAlgoBal) {
            throw new Error("startingAlgoBal is undefined");
          }
          console.log("startingAlgoBal", startingAlgoBal);

          // Start transfer from Algorand to Ethereum
          const hexStr = nativeToHexString(
            ETH_TEST_WALLET_PUBLIC_KEY,
            CHAIN_ID_ETH
          );
          if (!hexStr) {
            throw new Error("Failed to convert to hexStr");
          }
          console.log("hexStr", hexStr);
          const AmountToTransfer: number = 12300;
          const Fee: number = 0;
          console.log("Calling transferFromAlgorand...");
          const transferTxs = await transferFromAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            wallet.addr,
            AlgoIndex,
            BigInt(AmountToTransfer),
            hexStr,
            CHAIN_ID_ETH,
            BigInt(Fee)
          );
          const transferResult = await signSendAndConfirmAlgorand(
            client,
            transferTxs,
            wallet
          );
          const txSid = parseSequenceFromLogAlgorand(transferResult);
          console.log("Getting signed VAA...");
          const signedVaa = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ALGORAND,
            emitterAddr,
            txSid,
            { transport: NodeHttpTransport() }
          );
          console.log("About to redeemOnEth...");
          console.log("vaa", uint8ArrayToHex(signedVaa.vaaBytes));
          const pv = _parseVAAAlgorand(signedVaa.vaaBytes);
          console.log(pv);
          const roe = await redeemOnEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            signedVaa.vaaBytes
          );
          console.log("Check if transfer is complete...");
          expect(
            await getIsTransferCompletedEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              provider,
              signedVaa.vaaBytes
            )
          ).toBe(true);
          // Test finished.  Check wallet balances
          const balOnEthAfter = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const balOnEthAfterInt = parseInt(balOnEthAfter._hex);
          console.log("Balance on Eth after transfer = ", balOnEthAfterInt);
          expect(balOnEthAfterInt - initialBalOnEthInt).toEqual(
            AmountToTransfer
          );

          // Get final balance on Algorand
          algoWalletBals = await getBalances(client, wallet.addr);
          console.log("algoWalletBals after:", algoWalletBals);
          const finalAlgoBal = algoWalletBals.get(
            safeBigIntToNumber(AlgoIndex)
          );
          if (!finalAlgoBal) {
            throw new Error("finalAlgoBal is undefined");
          }
          console.log(
            "startingAlgoBal",
            startingAlgoBal,
            "finalAlgoBal",
            finalAlgoBal,
            "AmountToTransfer",
            AmountToTransfer
          );
          // expect(startingAlgoBal - finalAlgoBal).toBe(AmountToTransfer);

          // Attempt to transfer from Eth back to Algorand
          const Amount: string = "100";

          // approve the bridge to spend tokens
          console.log("About to approveEth...");
          await approveEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            foreignAsset,
            signer,
            Amount
          );
          console.log(
            "About to transferFromEth...",
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            foreignAsset,
            Amount,
            CHAIN_ID_ALGORAND,
            wallet.addr,
            decodeAddress(wallet.addr)
          );
          const receipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            foreignAsset,
            Amount,
            CHAIN_ID_ALGORAND,
            decodeAddress(wallet.addr).publicKey
          );
          console.log("receipt", receipt);
          // get the sequence from the logs (needed to fetch the vaa)
          console.log("About to parseSeq...");
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);

          // poll until the guardian(s) witness and sign the vaa
          console.log("About to getSignedVAA...");
          const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ETH,
            emitterAddress,
            sequence,
            {
              transport: NodeHttpTransport(),
            }
          );
          algoWalletBals = await getBalances(client, wallet.addr);
          console.log("algoWallet2Bals before:", algoWalletBals);
          console.log("About to redeemOnAlgorand...");
          const redeemTxs = await redeemOnAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            signedVAA,
            wallet.addr
          );
          await signSendAndConfirmAlgorand(client, redeemTxs, wallet);
          console.log("About to getIsTransferComplete...");
          const completed: boolean = await getIsTransferCompletedAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            signedVAA
          );
          console.log("Checking wallets...");
          const newBal = await token.balanceOf(ETH_TEST_WALLET_PUBLIC_KEY);
          const newBalInt = parseInt(newBal._hex);
          console.log(
            "newBalInt",
            newBalInt,
            "AmountToTransfer",
            AmountToTransfer,
            "Amount",
            Amount
          );
          // expect(newBalInt).toBe(AmountToTransfer - parseInt(Amount));

          // Get second final balance on Algorand
          algoWalletBals = await getBalances(client, wallet.addr);
          console.log("algoWalletBals after after:", algoWalletBals);
          const secondFinalAlgoBal = algoWalletBals.get(
            safeBigIntToNumber(AlgoIndex)
          );
          if (!secondFinalAlgoBal) {
            throw new Error("secondFinalAlgoBal is undefined");
          }
          console.log(
            "secondFinalAlgoBal",
            secondFinalAlgoBal,
            "finalAlgoBal",
            finalAlgoBal,
            "Amount",
            Amount
          );
          // expect(secondFinalAlgoBal - finalAlgoBal).toBe(
          //   parseInt(Amount) * 100
          // );
          console.log("algoWallet2Bals after:", algoWalletBals);
          provider.destroy();
        } catch (e) {
          console.error("Algorand ALGO transfer error:", e);
          done("Algorand ALGO transfer error");
          return;
        }
        done();
      })();
    });
    test("Algorand create chuckNorium, transfer to Eth and back again", (done) => {
      (async () => {
        try {
          console.log("Starting attestation...");
          const client: algosdk.Algodv2 = getAlgoClient();
          const tempAccts: Account[] = await getTempAccounts();
          const numAccts: number = tempAccts.length;
          expect(numAccts).toBeGreaterThan(0);
          const wallet: Account = tempAccts[0];

          let accountInfo = await client.accountInformation(wallet.addr).do();
          console.log("Account balance: %d microAlgos", accountInfo.amount);

          console.log("Creating fake asset...");
          const assetIndex: number = await createAsset(wallet);
          console.log("Newly created asset index =", assetIndex);
          console.log("Testing attestFromAlgorand...");
          const attestTxs = await attestFromAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            wallet.addr,
            BigInt(assetIndex)
          );
          const attestResult = await signSendAndConfirmAlgorand(
            client,
            attestTxs,
            wallet
          );
          const attestSn = parseSequenceFromLogAlgorand(attestResult);
          console.log("attestSn", attestSn);

          // Now, try to send a NOP
          console.log("Start of NOP...");
          const suggParams: algosdk.SuggestedParams = await client
            .getTransactionParams()
            .do();
          const nopTxn = makeApplicationCallTxnFromObject({
            from: wallet.addr,
            appIndex: safeBigIntToNumber(TOKEN_BRIDGE_ID),
            onComplete: OnApplicationComplete.NoOpOC,
            appArgs: [textToUint8Array("nop")],
            suggestedParams: suggParams,
          });
          const resp = await client
            .sendRawTransaction(nopTxn.signTxn(wallet.sk))
            .do();
          console.log("resp", resp);
          const response = await waitForConfirmation(client, resp.txId, 1);
          console.log("End of NOP");
          // End of NOP

          console.log("Getting emitter address...");
          const emitterAddr = getEmitterAddressAlgorand(TOKEN_BRIDGE_ID);
          const { vaaBytes } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ALGORAND,
            emitterAddr,
            attestSn,
            { transport: NodeHttpTransport() }
          );
          const provider = new ethers.providers.WebSocketProvider(
            ETH_NODE_URL
          ) as any;
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          let success: boolean = true;
          try {
            const cr = await createWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer,
              vaaBytes
            );
          } catch (e) {
            console.log(
              "createWrappedOnEth() failed.  Trying updateWrappedOnEth()..."
            );
            success = false;
          }
          if (!success) {
            console.log("using updateWrappedOnEth...");
            try {
              const cr = await updateWrappedOnEth(
                ETH_TOKEN_BRIDGE_ADDRESS,
                signer,
                vaaBytes
              );
              success = true;
            } catch (e) {
              console.error("failed to updateWrappedOnEth", e);
              done("failed to update attestation on Eth");
              return;
            }
          }
          console.log("Attestation is complete...");
          console.log("Starting transfer to Eth...");
          // Check wallet
          const a = parseInt(assetIndex.toString());
          const originAssetHex = (
            "0000000000000000000000000000000000000000000000000000000000000000" +
            a.toString(16)
          ).slice(-64);
          console.log(
            "assetIndex: ",
            assetIndex,
            ", originAssetHex:",
            originAssetHex
          );
          console.log(
            "calling getForeignAssetEth",
            ETH_TOKEN_BRIDGE_ADDRESS,
            CHAIN_ID_ALGORAND,
            originAssetHex
          );
          const foreignAsset = await getForeignAssetEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            provider,
            CHAIN_ID_ALGORAND,
            hexToUint8Array(originAssetHex)
          );
          if (!foreignAsset) {
            throw new Error("foreignAsset is null");
          }
          console.log("foreignAsset", foreignAsset);
          let token = TokenImplementation__factory.connect(
            foreignAsset,
            signer
          );

          // Get initial balance on ethereum
          const ETH_TEST_WALLET_PUBLIC_KEY =
            "0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1";
          const initialBalOnEth = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const initialBalOnEthInt = parseInt(initialBalOnEth._hex);
          console.log("Balance on Eth before transfer = ", initialBalOnEthInt);

          // Get initial balance on Algorand
          let algoWalletBals: Map<number, number> = await getBalances(
            client,
            wallet.addr
          );
          console.log("algoWalletBals:", algoWalletBals);
          const startingAlgoBal = algoWalletBals.get(assetIndex);
          if (!startingAlgoBal) {
            throw new Error("startingAlgoBal is undefined");
          }
          console.log("startingAlgoBal", startingAlgoBal);

          // Start transfer from Algorand to Ethereum
          const hexStr = nativeToHexString(
            ETH_TEST_WALLET_PUBLIC_KEY,
            CHAIN_ID_ETH
          );
          if (!hexStr) {
            throw new Error("Failed to convert to hexStr");
          }
          console.log("hexStr", hexStr);
          const AmountToTransfer: number = 12300;
          const Fee: number = 0;
          console.log("Calling transferFromAlgorand...");
          const transferTxs = await transferFromAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            wallet.addr,
            BigInt(assetIndex),
            BigInt(AmountToTransfer),
            hexStr,
            CHAIN_ID_ETH,
            BigInt(Fee)
          );
          const transferResult = await signSendAndConfirmAlgorand(
            client,
            transferTxs,
            wallet
          );
          const txSid = parseSequenceFromLogAlgorand(transferResult);
          console.log("Getting signed VAA...");
          const signedVaa = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ALGORAND,
            emitterAddr,
            txSid,
            { transport: NodeHttpTransport() }
          );
          console.log("About to redeemOnEth...");
          console.log("vaa", uint8ArrayToHex(signedVaa.vaaBytes));
          const roe = await redeemOnEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            signedVaa.vaaBytes
          );
          console.log("Check if transfer is complete...");
          expect(
            await getIsTransferCompletedEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              provider,
              signedVaa.vaaBytes
            )
          ).toBe(true);
          // Test finished.  Check wallet balances
          const balOnEthAfter = await token.balanceOf(
            ETH_TEST_WALLET_PUBLIC_KEY
          );
          const balOnEthAfterInt = parseInt(balOnEthAfter._hex);
          console.log("Balance on Eth after transfer = ", balOnEthAfterInt);
          const FinalAmt: number = AmountToTransfer / 100;
          expect(balOnEthAfterInt).toEqual(FinalAmt);

          // Get final balance on Algorand
          algoWalletBals = await getBalances(client, wallet.addr);
          console.log("algoWalletBals after:", algoWalletBals);
          const finalAlgoBal = algoWalletBals.get(assetIndex);
          if (!finalAlgoBal) {
            throw new Error("finalAlgoBal is undefined");
          }
          console.log("finalAlgoBal", finalAlgoBal);
          expect(startingAlgoBal - finalAlgoBal).toBe(AmountToTransfer);

          // Attempt to transfer from Eth back to Algorand
          const Amount: string = "100";

          // approve the bridge to spend tokens
          console.log("About to approveEth...");
          await approveEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            foreignAsset,
            signer,
            Amount
          );
          console.log(
            "About to transferFromEth...",
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            foreignAsset,
            Amount,
            CHAIN_ID_ALGORAND,
            wallet.addr,
            decodeAddress(wallet.addr)
          );
          const receipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            foreignAsset,
            Amount,
            CHAIN_ID_ALGORAND,
            decodeAddress(wallet.addr).publicKey
          );
          console.log("receipt", receipt);
          // get the sequence from the logs (needed to fetch the vaa)
          console.log("About to parseSeq...");
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);

          // poll until the guardian(s) witness and sign the vaa
          console.log("About to getSignedVAA...");
          const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ETH,
            emitterAddress,
            sequence,
            {
              transport: NodeHttpTransport(),
            }
          );
          console.log("About to redeemOnAlgorand...");
          const redeemTxs = await redeemOnAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            signedVAA,
            wallet.addr
          );
          await signSendAndConfirmAlgorand(client, redeemTxs, wallet);
          console.log("About to getIsTransferComplete...");
          const completed: boolean = await getIsTransferCompletedAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            signedVAA
          );
          console.log("Checking wallets...");
          const newBal = await token.balanceOf(ETH_TEST_WALLET_PUBLIC_KEY);
          const newBalInt = parseInt(newBal._hex);
          console.log("newBalInt", newBalInt);
          expect(newBalInt).toBe(FinalAmt - parseInt(Amount));

          // Get second final balance on Algorand
          algoWalletBals = await getBalances(client, wallet.addr);
          console.log("algoWalletBals after after:", algoWalletBals);
          const secondFinalAlgoBal = algoWalletBals.get(assetIndex);
          if (!secondFinalAlgoBal) {
            throw new Error("secondFinalAlgoBal is undefined");
          }
          console.log("secondFinalAlgoBal", secondFinalAlgoBal);
          expect(secondFinalAlgoBal - finalAlgoBal).toBe(
            parseInt(Amount) * 100
          );
          provider.destroy();
        } catch (e) {
          console.error("Algorand chuckNorium transfer error:", e);
          done("Algorand chuckNorium transfer error");
          return;
        }
        done();
      })();
    });
    test("Transfer wrapped Luna from Terra to Algorand and back again", (done) => {
      (async () => {
        try {
          console.log("Starting attestation...");
          const tbAddr: string = getApplicationAddress(TOKEN_BRIDGE_ID);
          const decTbAddr: Uint8Array = decodeAddress(tbAddr).publicKey;
          const aa: string = uint8ArrayToHex(decTbAddr);
          const client: algosdk.Algodv2 = getAlgoClient();
          const tempAccts: Account[] = await getTempAccounts();
          const numAccts: number = tempAccts.length;
          expect(numAccts).toBeGreaterThan(0);
          const algoWallet: Account = tempAccts[0];
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const terraWallet = lcd.wallet(mk);
          const Asset: string = "uluna";
          // const Asset: string = "uusd";
          const FeeAsset: string = "uusd";
          const Amount: string = "1000000";
          const TerraWalletAddress: string =
            "terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v";
          const msg = await attestFromTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            TerraWalletAddress,
            Asset
          );
          const gasPrices = lcd.config.gasPrices;
          let feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await terraWallet.sequence(),
                publicKey: terraWallet.key.publicKey,
              },
            ],
            {
              msgs: [msg],
              memo: "localhost",
              feeDenoms: [FeeAsset],
              gasPrices,
            }
          );
          const executeAttest = await terraWallet.createAndSignTx({
            msgs: [msg],
            memo: "Testing...",
            feeDenoms: [FeeAsset],
            gasPrices,
            fee: feeEstimate,
          });
          const attestResult = await lcd.tx.broadcast(executeAttest);
          const attestInfo = await waitForTerraExecution(attestResult.txhash);
          if (!attestInfo) {
            throw new Error("info not found");
          }
          const attestSn = parseSequenceFromLogTerra(attestInfo);
          if (!attestSn) {
            throw new Error("Sequence not found");
          }
          const emitterAddress = await getEmitterAddressTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS
          );
          const attestSignedVaa = await getSignedVAABySequence(
            CHAIN_ID_TERRA,
            attestSn,
            emitterAddress
          );
          console.log("About to createWrappedOnAlgorand...", attestSignedVaa);
          const createWrappedTxs = await createWrappedOnAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            algoWallet.addr,
            attestSignedVaa
          );
          await signSendAndConfirmAlgorand(
            client,
            createWrappedTxs,
            algoWallet
          );

          let assetIdCreated = await getForeignAssetFromVaaAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            attestSignedVaa
          );
          console.log("assetIdCreated", assetIdCreated);
          if (!assetIdCreated) {
            throw new Error("Failed to create asset");
          }

          // Start of transfer from Terra to Algorand
          // Get initial balance of luna on Terra
          const initialTerraBalance: number = await queryBalanceOnTerra(Asset);
          console.log("Initial Terra balance of", Asset, initialTerraBalance);

          // Get initial balance of uusd on Terra
          const initialFeeBalance: number = await queryBalanceOnTerra(FeeAsset);
          console.log("Initial Terra balance of", FeeAsset, initialFeeBalance);

          // Get initial balance of wrapped luna on Algorand
          const originAssetHex = nativeToHexString(Asset, CHAIN_ID_TERRA);
          if (!originAssetHex) {
            throw new Error("originAssetHex is null");
          }
          // TODO:  Get wallet balance on Algorand

          // Get Balances
          console.log("aa", tbAddr, decTbAddr, aa);
          const tbBals: Map<number, number> = await getBalances(
            client,
            algoWallet.addr
            // "TPFKQBOR7RJ475XW6XMOZMSMBCZH6WNGFQNT7CM7NL2UMBCMBIU5PVBGPM"
          );
          console.log("bals:", tbBals, assetIdCreated);
          let assetIdCreatedBegBal: number = 0;
          const tempBal = tbBals.get(safeBigIntToNumber(assetIdCreated));
          if (tempBal) {
            assetIdCreatedBegBal = tempBal;
          }
          console.log("assetIdCreatedBegBal", assetIdCreatedBegBal);

          // Start transfer from Terra to Algorand
          const txMsgs = await transferFromTerra(
            terraWallet.key.accAddress,
            TERRA_TOKEN_BRIDGE_ADDRESS,
            Asset,
            Amount,
            CHAIN_ID_ALGORAND,
            decodeAddress(algoWallet.addr).publicKey // This needs to be Algorand wallet
          );
          const executeTx = await terraWallet.createAndSignTx({
            msgs: txMsgs,
            memo: "Testing transfer...",
            feeDenoms: [FeeAsset],
            gasPrices,
            fee: feeEstimate,
          });
          const txResult = await lcd.tx.broadcast(executeTx);
          console.log("Transfer gas used: ", txResult.gas_used);
          const txInfo = await waitForTerraExecution(txResult.txhash);
          if (!txInfo) {
            throw new Error("info not found");
          }

          // Get VAA in order to do redemption step
          const txSn = parseSequenceFromLogTerra(txInfo);
          if (!txSn) {
            throw new Error("Sequence not found");
          }
          const txSignedVaa = await getSignedVAABySequence(
            CHAIN_ID_TERRA,
            txSn,
            emitterAddress
          );
          const redeemTxs = await redeemOnAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            txSignedVaa,
            algoWallet.addr
          );
          await signSendAndConfirmAlgorand(client, redeemTxs, algoWallet);
          expect(
            await getIsTransferCompletedAlgorand(
              client,
              TOKEN_BRIDGE_ID,
              txSignedVaa
            )
          ).toBe(true);

          // Test finished.  Check wallet balances
          // Get Balances
          const bals: Map<number, number> = await getBalances(
            client,
            algoWallet.addr
          );
          console.log("algoWallet bals:", bals);
          let assetIdCreatedEndBal: number = 0;
          const tmpBal = bals.get(safeBigIntToNumber(assetIdCreated));
          if (tmpBal) {
            assetIdCreatedEndBal = tmpBal;
          }
          console.log("assetIdCreatedEndBal", assetIdCreatedEndBal);
          expect(assetIdCreatedEndBal - assetIdCreatedBegBal).toBe(
            parseInt(Amount)
          );

          // Get final balance of uluna on Terra
          const finalTerraBalance = await queryBalanceOnTerra(Asset);
          console.log("Final Terra balance of", Asset, finalTerraBalance);

          // Get final balance of uusd on Terra
          const finalFeeBalance: number = await queryBalanceOnTerra(FeeAsset);
          console.log("Final Terra balance of", FeeAsset, finalFeeBalance);
          expect(initialTerraBalance - 1e6 === finalTerraBalance).toBe(true);

          // Start of transfer back to Terra
          const TransferBackAmount: number = 100000;

          // transfer wrapped luna from Algorand to Terra
          const terraHexStr = nativeToHexString(
            terraWallet.key.accAddress,
            CHAIN_ID_TERRA
          );
          if (!terraHexStr) {
            throw new Error("Failed to convert to hexStr");
          }
          const Fee: number = 0;
          console.log("Calling transferFromAlgorand...");
          const transferTxs = await transferFromAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            algoWallet.addr,
            assetIdCreated,
            BigInt(TransferBackAmount),
            terraHexStr,
            CHAIN_ID_TERRA,
            BigInt(Fee)
          );
          const transferResult = await signSendAndConfirmAlgorand(
            client,
            transferTxs,
            algoWallet
          );
          const txSid = parseSequenceFromLogAlgorand(transferResult);
          console.log("Getting signed VAA...");
          const signedVaa = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ALGORAND,
            aa,
            txSid,
            { transport: NodeHttpTransport() }
          );
          console.log("vaa", uint8ArrayToHex(signedVaa.vaaBytes));
          console.log("About to redeemOnTerra...");

          const redeemMsg = await redeemOnTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            terraWallet.key.accAddress,
            signedVaa.vaaBytes
          );
          feeEstimate = await lcd.tx.estimateFee(
            [
              {
                sequenceNumber: await terraWallet.sequence(),
                publicKey: terraWallet.key.publicKey,
              },
            ],
            {
              msgs: [redeemMsg],
              memo: "localhost",
              feeDenoms: [FeeAsset],
              gasPrices,
            }
          );
          const tx = await terraWallet.createAndSignTx({
            msgs: [redeemMsg],
            memo: "localhost",
            feeDenoms: ["uusd"],
            gasPrices,
            fee: feeEstimate,
          });
          await lcd.tx.broadcast(tx);
          expect(
            await getIsTransferCompletedTerra(
              TERRA_TOKEN_BRIDGE_ADDRESS,
              signedVaa.vaaBytes,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(true);

          // Check wallet balances after
          console.log("Checking wallet balances after transfer...");
          const finalLunaOnTerraBalance = await queryBalanceOnTerra(Asset);
          console.log(
            "Terra balance after transfer = ",
            finalLunaOnTerraBalance,
            finalTerraBalance
          );
          expect(finalLunaOnTerraBalance - finalTerraBalance).toBe(
            TransferBackAmount
          );
          const retBals: Map<number, number> = await getBalances(
            client,
            algoWallet.addr
          );
          console.log("algoWallet bals:", retBals);
          let assetIdCreatedFinBal: number = 0;
          const tBal = retBals.get(safeBigIntToNumber(assetIdCreated));
          if (tBal) {
            assetIdCreatedFinBal = tBal;
          }
          console.log(
            "assetIdCreatedFinBal",
            assetIdCreatedFinBal,
            assetIdCreatedEndBal
          );
          expect(assetIdCreatedEndBal - assetIdCreatedFinBal).toBe(
            TransferBackAmount
          );
          console.log("TESTING getOriginalAssetAlgorand....");
          const info: WormholeWrappedInfo = await getOriginalAssetAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            assetIdCreated
          );
          expect(info.chainId).toBe(CHAIN_ID_TERRA);
          expect(info.isWrapped).toBe(true);
        } catch (e) {
          console.error("Terra <=> Algorand error:", e);
          done("Terra <=> Algorand error");
        }
        done();
      })();
    });
    test("Testing relay type redeem", (done) => {
      (async () => {
        try {
          console.log("Starting new test of transferring ETH to algorand.");
          const client: algosdk.Algodv2 = getAlgoClient();
          const tempAccts: Account[] = await getTempAccounts();
          const numAccts: number = tempAccts.length;
          expect(numAccts).toBeGreaterThan(0);
          const algoWallet: Account = tempAccts[0];
          const algoWalletBalance = await getBalance(
            client,
            algoWallet.addr,
            BigInt(0)
          );
          expect(algoWalletBalance).toBeGreaterThan(0);
          const relayerWallet: Account = tempAccts[1];
          const relayerWalletBalance = await getBalance(
            client,
            relayerWallet.addr,
            BigInt(0)
          );
          expect(relayerWalletBalance).toBeGreaterThan(0);
          console.log("algoWallet", algoWallet.addr, algoWalletBalance);
          console.log(
            "relayerWallet",
            relayerWallet.addr,
            relayerWalletBalance
          );
          // ETH setup to transfer LUNA to Algorand

          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          console.log("here");
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          console.log("here");
          // attest the test token
          const receipt = await attestFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20
          );
          console.log("here");
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          console.log("here");
          const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
          console.log("here");
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
          console.log("About to createWrappedOnAlgorand...");
          const createWrappedTxs = await createWrappedOnAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            algoWallet.addr,
            signedVAA
          );
          await signSendAndConfirmAlgorand(
            client,
            createWrappedTxs,
            algoWallet
          );

          let assetIdCreated = await getForeignAssetFromVaaAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            signedVAA
          );
          if (!assetIdCreated) {
            throw new Error("Failed to create asset");
          }
          console.log("assetIdCreated", assetIdCreated);
          console.log(
            "algoWallet balance:",
            await getBalance(client, algoWallet.addr, assetIdCreated)
          );
          console.log(
            "relayerWallet balance:",
            await getBalance(client, relayerWallet.addr, assetIdCreated)
          );

          // Start of transfer from ETH to Algorand
          // approve the bridge to spend tokens
          const amount = parseUnits("2", 18);
          const halfAmount = parseUnits("1", 18);
          await approveEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            TEST_ERC20,
            signer,
            amount
          );
          // transfer half the tokens directly
          const firstHalfReceipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20,
            halfAmount,
            CHAIN_ID_ALGORAND,
            decodeAddress(algoWallet.addr).publicKey // This needs to be Algorand wallet
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const firstHalfSn = parseSequenceFromLogEth(
            firstHalfReceipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const ethEmitterAddress = getEmitterAddressEth(
            ETH_TOKEN_BRIDGE_ADDRESS
          );
          // poll until the guardian(s) witness and sign the vaa
          const { vaaBytes: firstHalfVaa } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ETH,
            ethEmitterAddress,
            firstHalfSn,
            {
              transport: NodeHttpTransport(),
            }
          );

          console.log("about to redeemOnAlgorand...");
          // Redeem half the amount on Algorand
          const firstHalfRedeemTxs = await redeemOnAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            firstHalfVaa,
            algoWallet.addr
          );
          await signSendAndConfirmAlgorand(
            client,
            firstHalfRedeemTxs,
            algoWallet
          );
          expect(
            await getIsTransferCompletedAlgorand(
              client,
              TOKEN_BRIDGE_ID,
              firstHalfVaa
            )
          ).toBe(true);
          // transfer second half of tokens via relayer
          const secondHalfReceipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20,
            halfAmount,
            CHAIN_ID_ALGORAND,
            decodeAddress(algoWallet.addr).publicKey // This needs to be Algorand wallet
          );
          // get the sequence from the logs (needed to fetch the vaa)
          const secondHalfSn = parseSequenceFromLogEth(
            secondHalfReceipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          // poll until the guardian(s) witness and sign the vaa
          const { vaaBytes: secondHalfVaa } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ETH,
            ethEmitterAddress,
            secondHalfSn,
            {
              transport: NodeHttpTransport(),
            }
          );

          console.log("about to redeemOnAlgorand second half...");
          // Redeem second half the amount on Algorand
          const redeemTxs = await redeemOnAlgorand(
            client,
            TOKEN_BRIDGE_ID,
            CORE_ID,
            secondHalfVaa,
            relayerWallet.addr
          );
          await signSendAndConfirmAlgorand(client, redeemTxs, relayerWallet);
          expect(
            await getIsTransferCompletedAlgorand(
              client,
              TOKEN_BRIDGE_ID,
              secondHalfVaa
            )
          ).toBe(true);
          console.log("Destroying the provider...");
          provider.destroy();
        } catch (e) {
          console.error("new test error:", e);
          done("new test error");
          return;
        }
        done();
      })();
    });
  });
});
