import {
  approveEth,
  attestFromEth,
  attestFromSolana,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  createWrappedOnEth,
  createWrappedOnSolana,
  createWrappedOnTerra,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getForeignAssetSolana,
  getIsTransferCompletedEth,
  getIsTransferCompletedSolana,
  getIsTransferCompletedTerra,
  hexToUint8Array,
  nativeToHexString,
  postVaaSolana,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  transferFromEth,
  transferFromSolana,
  CHAIN_ID_TERRA2,
} from "@certusone/wormhole-sdk";

import getSignedVAAWithRetry from "@certusone/wormhole-sdk/lib/cjs/rpc/getSignedVAAWithRetry";
import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";

import { parseUnits } from "@ethersproject/units";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { describe, expect, jest, test } from "@jest/globals";

import { ethers } from "ethers";

import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { LCDClient, MnemonicKey } from "@terra-money/terra.js";
import axios from "axios";
import {
  ETH_CORE_BRIDGE_ADDRESS,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_CORE_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOLANA_TOKEN_BRIDGE_ADDRESS,
  SPY_RELAY_URL,
  TERRA2_GAS_PRICES_URL,
  TERRA2_NODE_URL,
  TERRA2_TOKEN_BRIDGE_ADDRESS,
  TERRA_CHAIN_ID,
  TERRA_GAS_PRICES_URL,
  TERRA_NODE_URL,
  TERRA_PRIVATE_KEY,
  TERRA_TOKEN_BRIDGE_ADDRESS,
  TEST_ERC20,
  TEST_SOLANA_TOKEN,
  WORMHOLE_RPC_HOSTS,
} from "./consts";

import { sleep } from "../helpers/utils";

setDefaultWasm("node");

jest.setTimeout(60000);

test("Verify Spy Relay is running", (done) => {
  (async () => {
    try {
      console.log(
        "Sending query to spy relay to see if it's running, query: [%s]",
        SPY_RELAY_URL
      );

      const result = await axios.get(SPY_RELAY_URL);

      expect(result).toHaveProperty("status");
      expect(result.status).toBe(200);

      done();
    } catch (e) {
      console.error("Spy Relay does not appear to be running!");
      console.error(e);
      done("Spy Relay does not appear to be running!");
    }
  })();
});

let sequence: string;
let emitterAddress: string;
let transferSignedVAA: Uint8Array;

describe("Solana to Ethereum", () => {
  test("Attest Solana SPL to Ethereum", (done) => {
    (async () => {
      console.log("Attest Solana SPL to Ethereum");
      try {
        // create a keypair for Solana
        const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
        const payerAddress = keypair.publicKey.toString();
        // attest the test token
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const transaction = await attestFromSolana(
          connection,
          SOLANA_CORE_BRIDGE_ADDRESS,
          SOLANA_TOKEN_BRIDGE_ADDRESS,
          payerAddress,
          TEST_SOLANA_TOKEN
        );
        // sign, send, and confirm transaction
        transaction.partialSign(keypair);
        const txid = await connection.sendRawTransaction(
          transaction.serialize()
        );
        await connection.confirmTransaction(txid);
        const info = await connection.getTransaction(txid);
        if (!info) {
          throw new Error(
            "An error occurred while fetching the transaction info"
          );
        }
        // get the sequence from the logs (needed to fetch the vaa)
        const sequence = parseSequenceFromLogSolana(info);
        emitterAddress = await getEmitterAddressSolana(
          SOLANA_TOKEN_BRIDGE_ADDRESS
        );
        // poll until the guardian(s) witness and sign the vaa
        const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_SOLANA,
          emitterAddress,
          sequence,
          {
            transport: NodeHttpTransport(),
          }
        );
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        try {
          await createWrappedOnEth(ETH_TOKEN_BRIDGE_ADDRESS, signer, signedVAA);
        } catch (e) {
          // this could fail because the token is already attested (in an unclean env)
        }
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done(
          "An error occurred while trying to attest from Solana to Ethereum"
        );
      }
    })();
  });
  // TODO: it is attested
  test("Send Solana SPL to Ethereum", (done) => {
    (async () => {
      console.log("Send Solana SPL to Ethereum");
      try {
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        const targetAddress = await signer.getAddress();
        // create a keypair for Solana
        const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
        const payerAddress = keypair.publicKey.toString();
        // find the associated token account
        const fromAddress = (
          await Token.getAssociatedTokenAddress(
            ASSOCIATED_TOKEN_PROGRAM_ID,
            TOKEN_PROGRAM_ID,
            new PublicKey(TEST_SOLANA_TOKEN),
            keypair.publicKey
          )
        ).toString();
        // transfer the test token
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const amount = parseUnits("1", 9).toBigInt();
        const fee = parseUnits("1", 3).toBigInt();
        const transaction = await transferFromSolana(
          connection,
          SOLANA_CORE_BRIDGE_ADDRESS,
          SOLANA_TOKEN_BRIDGE_ADDRESS,
          payerAddress,
          fromAddress,
          TEST_SOLANA_TOKEN,
          amount + fee,
          hexToUint8Array(nativeToHexString(targetAddress, CHAIN_ID_ETH) || ""),
          CHAIN_ID_ETH,
          Buffer.from(TEST_SOLANA_TOKEN),
          CHAIN_ID_SOLANA,
          undefined,
          fee
        );
        // sign, send, and confirm transaction
        console.log("Sending transaction.");
        transaction.partialSign(keypair);
        const txid = await connection.sendRawTransaction(
          transaction.serialize()
        );
        console.log("Confirming transaction.");
        await connection.confirmTransaction(txid);
        const info = await connection.getTransaction(txid);
        if (!info) {
          throw new Error(
            "An error occurred while fetching the transaction info"
          );
        }
        // get the sequence from the logs (needed to fetch the vaa)
        console.log("Parsing sequence number from log.");
        sequence = parseSequenceFromLogSolana(info);
        const emitterAddress = await getEmitterAddressSolana(
          SOLANA_TOKEN_BRIDGE_ADDRESS
        );
        // poll until the guardian(s) witness and sign the vaa
        console.log("Waiting on signed vaa, sequence %d", sequence);
        const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_SOLANA,
          emitterAddress,
          sequence,
          {
            transport: NodeHttpTransport(),
          }
        );
        console.log("Got signed vaa: ", signedVAA);
        transferSignedVAA = signedVAA;
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to send from Solana to Ethereum");
      }
    })();
  });
  test("Spy Relay redeemed on Eth", (done) => {
    (async () => {
      try {
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        let success: boolean = false;
        for (let count = 0; count < 5 && !success; ++count) {
          console.log(
            "sleeping before querying spy relay",
            new Date().toLocaleString()
          );
          await sleep(5000);
          success = await getIsTransferCompletedEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            provider,
            transferSignedVAA
          );
          console.log(
            "getIsTransferCompletedEth returned %d, count is %d",
            success,
            count
          );
        }
        expect(success).toBe(true);
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to redeem on Eth");
      }
    })();
  });
});

describe("Ethereum to Solana", () => {
  test("Attest Ethereum ERC-20 to Solana", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        // attest the test token
        const receipt = await attestFromEth(
          ETH_TOKEN_BRIDGE_ADDRESS,
          signer,
          TEST_ERC20
        );
        // get the sequence from the logs (needed to fetch the vaa)
        const sequence = parseSequenceFromLogEth(
          receipt,
          ETH_CORE_BRIDGE_ADDRESS
        );
        const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
        // create a keypair for Solana
        const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
        const payerAddress = keypair.publicKey.toString();
        // post vaa to Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        await postVaaSolana(
          connection,
          async (transaction) => {
            transaction.partialSign(keypair);
            return transaction;
          },
          SOLANA_CORE_BRIDGE_ADDRESS,
          payerAddress,
          Buffer.from(signedVAA)
        );
        // create wormhole wrapped token (mint and metadata) on solana
        const transaction = await createWrappedOnSolana(
          connection,
          SOLANA_CORE_BRIDGE_ADDRESS,
          SOLANA_TOKEN_BRIDGE_ADDRESS,
          payerAddress,
          signedVAA
        );
        // sign, send, and confirm transaction
        try {
          transaction.partialSign(keypair);
          const txid = await connection.sendRawTransaction(
            transaction.serialize()
          );
          await connection.confirmTransaction(txid);
        } catch (e) {
          // this could fail because the token is already attested (in an unclean env)
        }
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done(
          "An error occurred while trying to attest from Ethereum to Solana"
        );
      }
    })();
  });
  // TODO: it is attested
  test("Send Ethereum ERC-20 to Solana", (done) => {
    (async () => {
      try {
        // create a keypair for Solana
        const connection = new Connection(SOLANA_HOST, "confirmed");
        const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
        // determine destination address - an associated token account
        const solanaMintKey = new PublicKey(
          (await getForeignAssetSolana(
            connection,
            SOLANA_TOKEN_BRIDGE_ADDRESS,
            CHAIN_ID_ETH,
            hexToUint8Array(nativeToHexString(TEST_ERC20, CHAIN_ID_ETH) || "")
          )) || ""
        );
        const recipient = await Token.getAssociatedTokenAddress(
          ASSOCIATED_TOKEN_PROGRAM_ID,
          TOKEN_PROGRAM_ID,
          solanaMintKey,
          keypair.publicKey
        );
        // create the associated token account if it doesn't exist
        const associatedAddressInfo = await connection.getAccountInfo(
          recipient
        );
        if (!associatedAddressInfo) {
          const transaction = new Transaction().add(
            await Token.createAssociatedTokenAccountInstruction(
              ASSOCIATED_TOKEN_PROGRAM_ID,
              TOKEN_PROGRAM_ID,
              solanaMintKey,
              recipient,
              keypair.publicKey, // owner
              keypair.publicKey // payer
            )
          );
          const { blockhash } = await connection.getRecentBlockhash();
          transaction.recentBlockhash = blockhash;
          transaction.feePayer = keypair.publicKey;
          // sign, send, and confirm transaction
          transaction.partialSign(keypair);
          const txid = await connection.sendRawTransaction(
            transaction.serialize()
          );
          await connection.confirmTransaction(txid);
        }
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        const amount = parseUnits("1", 18);
        const fee = parseUnits("1", 12);
        const transferAmount = amount.add(fee);
        // approve the bridge to spend tokens
        await approveEth(
          ETH_TOKEN_BRIDGE_ADDRESS,
          TEST_ERC20,
          signer,
          transferAmount
        );
        // transfer tokens
        const receipt = await transferFromEth(
          ETH_TOKEN_BRIDGE_ADDRESS,
          signer,
          TEST_ERC20,
          transferAmount,
          CHAIN_ID_SOLANA,
          hexToUint8Array(
            nativeToHexString(recipient.toString(), CHAIN_ID_SOLANA) || ""
          ),
          fee
        );
        // get the sequence from the logs (needed to fetch the vaa)
        sequence = parseSequenceFromLogEth(receipt, ETH_CORE_BRIDGE_ADDRESS);
        emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
        console.log("Got signed vaa: ", signedVAA);
        transferSignedVAA = signedVAA;
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to send from Ethereum to Solana");
      }
    })();
  });
  test("Spy Relay redeemed on Sol", (done) => {
    (async () => {
      try {
        const connection = new Connection(SOLANA_HOST, "confirmed");
        let success: boolean = false;
        for (let count = 0; count < 5 && !success; ++count) {
          console.log(
            "sleeping before querying spy relay",
            new Date().toLocaleString()
          );
          await sleep(5000);
          success = await getIsTransferCompletedSolana(
            SOLANA_TOKEN_BRIDGE_ADDRESS,
            transferSignedVAA,
            connection
          );
          console.log(
            "getIsTransferCompletedSolana returned %d, count is %d",
            success,
            count
          );
        }
        expect(success).toBe(true);
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to redeem on Sol");
      }
    })();
  });
});

describe("Ethereum to Terra Classic", () => {
  test("Attest Ethereum ERC-20 to Terra Classic", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        // attest the test token
        const receipt = await attestFromEth(
          ETH_TOKEN_BRIDGE_ADDRESS,
          signer,
          TEST_ERC20
        );
        // get the sequence from the logs (needed to fetch the vaa)
        const sequence = parseSequenceFromLogEth(
          receipt,
          ETH_CORE_BRIDGE_ADDRESS
        );
        const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
        const lcd = new LCDClient({
          URL: TERRA_NODE_URL,
          chainID: TERRA_CHAIN_ID,
          isClassic: true,
        });
        const mk = new MnemonicKey({
          mnemonic: TERRA_PRIVATE_KEY,
        });
        const wallet = lcd.wallet(mk);
        const msg = await createWrappedOnTerra(
          TERRA_TOKEN_BRIDGE_ADDRESS,
          wallet.key.accAddress,
          signedVAA
        );
        const gasPrices = await axios
          .get(TERRA_GAS_PRICES_URL)
          .then((result) => result.data);
        const account = await lcd.auth.accountInfo(wallet.key.accAddress);
        const feeEstimate = await lcd.tx.estimateFee(
          [
            {
              sequenceNumber: account.getSequenceNumber(),
              publicKey: account.getPublicKey(),
            },
          ],
          {
            msgs: [msg],
            feeDenoms: ["uluna"],
            gasPrices,
          }
        );
        const tx = await wallet.createAndSignTx({
          msgs: [msg],
          memo: "test",
          feeDenoms: ["uluna"],
          gasPrices,
          fee: feeEstimate,
        });
        await lcd.tx.broadcast(tx);
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done(
          "An error occurred while trying to attest from Ethereum to Terra Classic"
        );
      }
    })();
  });
  // TODO: it is attested
  test("Send Ethereum ERC-20 to Terra Classic", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        const amount = parseUnits("1", 18);
        const fee = parseUnits("1", 12);
        const transferAmount = amount.add(fee);
        // approve the bridge to spend tokens
        await approveEth(
          ETH_TOKEN_BRIDGE_ADDRESS,
          TEST_ERC20,
          signer,
          transferAmount
        );
        const lcd = new LCDClient({
          URL: TERRA_NODE_URL,
          chainID: TERRA_CHAIN_ID,
          isClassic: true,
        });
        const mk = new MnemonicKey({
          mnemonic: TERRA_PRIVATE_KEY,
        });
        const wallet = lcd.wallet(mk);
        // transfer tokens
        const receipt = await transferFromEth(
          ETH_TOKEN_BRIDGE_ADDRESS,
          signer,
          TEST_ERC20,
          transferAmount,
          CHAIN_ID_TERRA,
          hexToUint8Array(
            nativeToHexString(wallet.key.accAddress, CHAIN_ID_TERRA) || ""
          ),
          fee
        );
        // get the sequence from the logs (needed to fetch the vaa)
        sequence = parseSequenceFromLogEth(receipt, ETH_CORE_BRIDGE_ADDRESS);
        emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
        console.log("Got signed vaa: ", signedVAA);
        transferSignedVAA = signedVAA;
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done(
          "An error occurred while trying to send from Ethereum to Terra Classic"
        );
      }
    })();
  });

  test("Spy Relay redeemed on Terra Classic", (done) => {
    (async () => {
      try {
        const lcd = new LCDClient({
          URL: TERRA_NODE_URL,
          chainID: TERRA_CHAIN_ID,
          isClassic: true,
        });
        var success: boolean = false;
        for (let count = 0; count < 5 && !success; ++count) {
          console.log(
            "sleeping before querying spy relay",
            new Date().toLocaleString()
          );
          await sleep(5000);
          success = await await getIsTransferCompletedTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            transferSignedVAA,
            lcd,
            TERRA_GAS_PRICES_URL
          );
          console.log(
            "getIsTransferCompletedTerra returned %d, count is %d",
            success,
            count
          );
        }
        expect(success).toBe(true);
        done();
      } catch (e) {
        console.error(e);
        done(
          "An error occurred while checking to see if redeem on Terra Classic was successful"
        );
      }
    })();
  });
});

describe("Ethereum to Terra", () => {
  test("Attest Ethereum ERC-20 to Terra", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        // attest the test token
        const receipt = await attestFromEth(
          ETH_TOKEN_BRIDGE_ADDRESS,
          signer,
          TEST_ERC20
        );
        // get the sequence from the logs (needed to fetch the vaa)
        const sequence = parseSequenceFromLogEth(
          receipt,
          ETH_CORE_BRIDGE_ADDRESS
        );
        const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
        const lcd = new LCDClient({
          URL: TERRA2_NODE_URL,
          chainID: TERRA_CHAIN_ID,
          isClassic: false,
        });
        const mk = new MnemonicKey({
          mnemonic: TERRA_PRIVATE_KEY,
        });
        const wallet = lcd.wallet(mk);
        const msg = await createWrappedOnTerra(
          TERRA2_TOKEN_BRIDGE_ADDRESS,
          wallet.key.accAddress,
          signedVAA
        );
        const gasPrices = await axios
          .get(TERRA2_GAS_PRICES_URL)
          .then((result) => result.data);
        const account = await lcd.auth.accountInfo(wallet.key.accAddress);
        const feeEstimate = await lcd.tx.estimateFee(
          [
            {
              sequenceNumber: account.getSequenceNumber(),
              publicKey: account.getPublicKey(),
            },
          ],
          {
            msgs: [msg],
            feeDenoms: ["uluna"],
            gasPrices,
          }
        );
        const tx = await wallet.createAndSignTx({
          msgs: [msg],
          memo: "test",
          feeDenoms: ["uluna"],
          gasPrices,
          fee: feeEstimate,
        });
        await lcd.tx.broadcast(tx);
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to attest from Ethereum to Terra");
      }
    })();
  });
  // TODO: it is attested
  test("Send Ethereum ERC-20 to Terra", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
        const amount = parseUnits("1", 18);
        const fee = parseUnits("1", 12);
        const transferAmount = amount.add(fee);
        // approve the bridge to spend tokens
        await approveEth(
          ETH_TOKEN_BRIDGE_ADDRESS,
          TEST_ERC20,
          signer,
          transferAmount
        );
        const lcd = new LCDClient({
          URL: TERRA2_NODE_URL,
          chainID: TERRA_CHAIN_ID,
          isClassic: false,
        });
        const mk = new MnemonicKey({
          mnemonic: TERRA_PRIVATE_KEY,
        });
        const wallet = lcd.wallet(mk);
        // transfer tokens
        const receipt = await transferFromEth(
          ETH_TOKEN_BRIDGE_ADDRESS,
          signer,
          TEST_ERC20,
          transferAmount,
          CHAIN_ID_TERRA2,
          hexToUint8Array(
            nativeToHexString(wallet.key.accAddress, CHAIN_ID_TERRA2) || ""
          ),
          fee
        );
        // get the sequence from the logs (needed to fetch the vaa)
        sequence = parseSequenceFromLogEth(receipt, ETH_CORE_BRIDGE_ADDRESS);
        emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
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
        console.log("Got signed vaa: ", signedVAA);
        transferSignedVAA = signedVAA;
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to send from Ethereum to Terra");
      }
    })();
  });

  test("Spy Relay redeemed on Terra", (done) => {
    (async () => {
      try {
        const lcd = new LCDClient({
          URL: TERRA2_NODE_URL,
          chainID: TERRA_CHAIN_ID,
          isClassic: false,
        });
        var success: boolean = false;
        for (let count = 0; count < 5 && !success; ++count) {
          console.log(
            "sleeping before querying spy relay",
            new Date().toLocaleString()
          );
          await sleep(5000);
          success = await await getIsTransferCompletedTerra(
            TERRA2_TOKEN_BRIDGE_ADDRESS,
            transferSignedVAA,
            lcd,
            TERRA2_GAS_PRICES_URL
          );
          console.log(
            "getIsTransferCompletedTerra returned %d, count is %d",
            success,
            count
          );
        }
        expect(success).toBe(true);
        done();
      } catch (e) {
        console.error(e);
        done(
          "An error occurred while checking to see if redeem on Terra was successful"
        );
      }
    })();
  });
});
