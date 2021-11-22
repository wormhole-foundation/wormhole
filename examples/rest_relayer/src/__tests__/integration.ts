import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  getEmitterAddressSolana,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogSolana,
  transferFromSolana,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import getSignedVAAWithRetry from "@certusone/wormhole-sdk/lib/cjs/rpc/getSignedVAAWithRetry";
import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { parseUnits } from "@ethersproject/units";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { jest, test } from "@jest/globals";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, Keypair, PublicKey } from "@solana/web3.js";
import axios from "axios";
import {
  ETH_PUBLIC_KEY,
  SOLANA_CORE_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOLANA_TOKEN_BRIDGE_ADDRESS,
  TEST_SOLANA_TOKEN,
  WORMHOLE_RPC_HOSTS,
} from "./consts";

const RELAYER_URL = "http://localhost:3001/relay";

setDefaultWasm("node");

jest.setTimeout(60000);

test("Send Solana SPL to Ethereum", (done) => {
  (async () => {
    try {
      const targetAddress = ETH_PUBLIC_KEY;
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
        hexToUint8Array(nativeToHexString(targetAddress, CHAIN_ID_ETH) || ""),
        CHAIN_ID_ETH
      );
      // sign, send, and confirm transaction
      transaction.partialSign(keypair);
      const txid = await connection.sendRawTransaction(transaction.serialize());
      console.log("Solana transaction:", txid);
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
      const result = await axios.post(RELAYER_URL, {
        chainId: CHAIN_ID_ETH,
        signedVAA: uint8ArrayToHex(signedVAA),
      });
      console.log(result);
      done();
    } catch (e) {
      console.error(e);
      done("An error occurred while trying to send from Solana to Ethereum");
    }
  })();
});
