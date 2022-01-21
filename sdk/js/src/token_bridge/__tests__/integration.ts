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
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  postVaaSolana,
  redeemOnEth,
  redeemOnSolana,
  redeemOnTerra,
  transferFromEth,
  transferFromSolana,
} from "../..";
import getSignedVAAWithRetry from "../../rpc/getSignedVAAWithRetry";
import { postVaaWithRetry } from "../../solana/postVaa";
import { setDefaultWasm } from "../../solana/wasm";
import {
  ETH_CORE_BRIDGE_ADDRESS,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_CORE_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOLANA_TOKEN_BRIDGE_ADDRESS,
  TERRA_CHAIN_ID,
  TERRA_GAS_PRICES_URL,
  TERRA_NODE_URL,
  TERRA_PRIVATE_KEY,
  TERRA_TOKEN_BRIDGE_ADDRESS,
  TEST_ERC20,
  TEST_SOLANA_TOKEN,
  WORMHOLE_RPC_HOSTS,
} from "./consts";
import { transferFromEthToSolana } from "./helpers";

setDefaultWasm("node");

jest.setTimeout(60000);

// TODO: setup keypair and provider/signer before, destroy provider after
// TODO: make the repeatable (can't attest an already attested token)

describe("Integration Tests", () => {
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
          const payerAddress = keypair.publicKey.toString();
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
          // approve the bridge to spend tokens
          await approveEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            TEST_ERC20,
            signer,
            amount
          );
          // transfer tokens
          const receipt = await transferFromEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            signer,
            TEST_ERC20,
            amount,
            CHAIN_ID_SOLANA,
            hexToUint8Array(
              nativeToHexString(recipient.toString(), CHAIN_ID_SOLANA) || ""
            )
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
          // post vaa to Solana
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
          expect(
            await getIsTransferCompletedSolana(
              SOLANA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              connection
            )
          ).toBe(false);
          // redeem tokens on solana
          const transaction = await redeemOnSolana(
            connection,
            SOLANA_CORE_BRIDGE_ADDRESS,
            SOLANA_TOKEN_BRIDGE_ADDRESS,
            payerAddress,
            signedVAA
          );
          // sign, send, and confirm transaction
          transaction.partialSign(keypair);
          const txid = await connection.sendRawTransaction(
            transaction.serialize()
          );
          await connection.confirmTransaction(txid);
          expect(
            await getIsTransferCompletedSolana(
              SOLANA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              connection
            )
          ).toBe(true);
          provider.destroy();
          done();
        } catch (e) {
          console.error(e);
          done(
            "An error occurred while trying to send from Ethereum to Solana"
          );
        }
      })();
    });
    // TODO: it has increased balance
  });
  describe("Solana to Ethereum", () => {
    test("Attest Solana SPL to Ethereum", (done) => {
      (async () => {
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
          const emitterAddress = await getEmitterAddressSolana(
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
            await createWrappedOnEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              signer,
              signedVAA
            );
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
          const transaction = await transferFromSolana(
            connection,
            SOLANA_CORE_BRIDGE_ADDRESS,
            SOLANA_TOKEN_BRIDGE_ADDRESS,
            payerAddress,
            fromAddress,
            TEST_SOLANA_TOKEN,
            amount,
            hexToUint8Array(
              nativeToHexString(targetAddress, CHAIN_ID_ETH) || ""
            ),
            CHAIN_ID_ETH
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
          const emitterAddress = await getEmitterAddressSolana(
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
          expect(
            await getIsTransferCompletedEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              provider,
              signedVAA
            )
          ).toBe(false);
          await redeemOnEth(ETH_TOKEN_BRIDGE_ADDRESS, signer, signedVAA);
          expect(
            await getIsTransferCompletedEth(
              ETH_TOKEN_BRIDGE_ADDRESS,
              provider,
              signedVAA
            )
          ).toBe(true);
          provider.destroy();
          done();
        } catch (e) {
          console.error(e);
          done(
            "An error occurred while trying to send from Solana to Ethereum"
          );
        }
      })();
    });
    // TODO: it has increased balance
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
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
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
          const feeEstimate = await lcd.tx.estimateFee(
            wallet.key.accAddress,
            [msg],
            {
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
            "An error occurred while trying to attest from Ethereum to Terra"
          );
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
          // approve the bridge to spend tokens
          await approveEth(
            ETH_TOKEN_BRIDGE_ADDRESS,
            TEST_ERC20,
            signer,
            amount
          );
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
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
            amount,
            CHAIN_ID_TERRA,
            hexToUint8Array(
              nativeToHexString(wallet.key.accAddress, CHAIN_ID_TERRA) || ""
            )
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
          expect(
            await getIsTransferCompletedTerra(
              TERRA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              wallet.key.accAddress,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(false);
          const msg = await redeemOnTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
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
              TERRA_TOKEN_BRIDGE_ADDRESS,
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
          done("An error occurred while trying to send from Ethereum to Terra");
        }
      })();
    });
    // TODO: it has increased balance
  });
  describe("Terra deposit and transfer tokens", () => {
    test("Tokens transferred can't exceed tokens deposited", (done) => {
      (async () => {
        try {
          const lcd = new LCDClient({
            URL: TERRA_NODE_URL,
            chainID: TERRA_CHAIN_ID,
          });
          const mk = new MnemonicKey({
            mnemonic: TERRA_PRIVATE_KEY,
          });
          const wallet = lcd.wallet(mk);
          const gasPrices = await axios
            .get(TERRA_GAS_PRICES_URL)
            .then((result) => result.data);
          // deposit some tokens (separate transactions)
          for (let i = 0; i < 3; i++) {
            const deposit = new MsgExecuteContract(
              wallet.key.accAddress,
              TERRA_TOKEN_BRIDGE_ADDRESS,
              {
                deposit_tokens: {},
              },
              { uusd: "900000087654321" }
            );
            const feeEstimate = await lcd.tx.estimateFee(
              wallet.key.accAddress,
              [deposit],
              {
                memo: "localhost",
                feeDenoms: ["uluna"],
                gasPrices,
              }
            );
            const tx = await wallet.createAndSignTx({
              msgs: [deposit],
              memo: "localhost",
              feeDenoms: ["uluna"],
              gasPrices,
              fee: feeEstimate,
            });
            await lcd.tx.broadcast(tx);
          }
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          // attempt to transfer more than we've deposited
          const transfer = new MsgExecuteContract(
            wallet.key.accAddress,
            TERRA_TOKEN_BRIDGE_ADDRESS,
            {
              initiate_transfer: {
                asset: {
                  amount: "5900000087654321",
                  info: {
                    native_token: {
                      denom: "uusd",
                    },
                  },
                },
                recipient_chain: CHAIN_ID_ETH,
                recipient: Buffer.from(signer.publicKey).toString("base64"),
                fee: "0",
                nonce: Math.round(Math.round(Math.random() * 100000)),
              },
            },
            {}
          );
          let error = false;
          try {
            await lcd.tx.estimateFee(wallet.key.accAddress, [transfer], {
              memo: "localhost",
              feeDenoms: ["uluna"],
              gasPrices,
            });
          } catch (e) {
            error = e.response.data.error.includes("Overflow: Cannot Sub");
          }
          expect(error).toEqual(true);
          // withdraw the tokens we deposited
          const withdraw = new MsgExecuteContract(
            wallet.key.accAddress,
            TERRA_TOKEN_BRIDGE_ADDRESS,
            {
              withdraw_tokens: {
                asset: {
                  native_token: {
                    denom: "uusd",
                  },
                },
              },
            },
            {}
          );
          const feeEstimate = await lcd.tx.estimateFee(
            wallet.key.accAddress,
            [withdraw],
            {
              memo: "localhost",
              feeDenoms: ["uluna"],
              gasPrices,
            }
          );
          const tx = await wallet.createAndSignTx({
            msgs: [withdraw],
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
            "An error occurred while testing deposits to and transfers from Terra"
          );
        }
      })();
    });
  });
  describe("Post VAA with retry", () => {
    test("postVAA with retry, no failures", (done) => {
      (async () => {
        try {
          // create a keypair for Solana
          const connection = new Connection(SOLANA_HOST, "confirmed");
          const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
          const payerAddress = keypair.publicKey.toString();
          const sequence = await transferFromEthToSolana();
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
          let maxFailures = 0;
          // post vaa to Solana

          const postPromise = postVaaWithRetry(
            connection,
            async (transaction) => {
              await new Promise(function (resolve) {
                //We delay here so the connection has time to get wrecked
                setTimeout(function () {
                  resolve(500);
                });
              });
              transaction.partialSign(keypair);
              return transaction;
            },
            SOLANA_CORE_BRIDGE_ADDRESS,
            payerAddress,
            Buffer.from(signedVAA),
            maxFailures
          );

          await postPromise;
          // redeem tokens on solana
          const transaction = await redeemOnSolana(
            connection,
            SOLANA_CORE_BRIDGE_ADDRESS,
            SOLANA_TOKEN_BRIDGE_ADDRESS,
            payerAddress,
            signedVAA
          );
          // sign, send, and confirm transaction
          transaction.partialSign(keypair);
          const txid = await connection.sendRawTransaction(
            transaction.serialize()
          );
          await connection.confirmTransaction(txid);
          expect(
            await getIsTransferCompletedSolana(
              SOLANA_TOKEN_BRIDGE_ADDRESS,
              signedVAA,
              connection
            )
          ).toBe(true);
          done();
        } catch (e) {
          console.error(e);
          done(
            "An error occurred while happy-path testing post VAA with retry."
          );
        }
      })();
    });
    test("Reject on signature failure", (done) => {
      (async () => {
        try {
          // create a keypair for Solana
          const connection = new Connection(SOLANA_HOST, "confirmed");
          const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
          const payerAddress = keypair.publicKey.toString();
          const sequence = await transferFromEthToSolana();
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
          let maxFailures = 5;
          // post vaa to Solana

          let error = false;
          try {
            const postPromise = postVaaWithRetry(
              connection,
              async (transaction) => {
                return Promise.reject();
              },
              SOLANA_CORE_BRIDGE_ADDRESS,
              payerAddress,
              Buffer.from(signedVAA),
              maxFailures
            );

            await postPromise;
          } catch (e) {
            error = true;
          }
          expect(error).toBe(true);
          done();
        } catch (e) {
          console.error(e);
          done(
            "An error occurred while trying to send from Ethereum to Solana"
          );
        }
      })();
    });
  });
});
