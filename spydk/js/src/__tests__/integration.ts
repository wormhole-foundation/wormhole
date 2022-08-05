import { attestFromSolana } from "@certusone/wormhole-sdk";
import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { jest, test } from "@jest/globals";
import { Connection, Keypair } from "@solana/web3.js";
import { createSpyRPCServiceClient, subscribeSignedVAA } from "..";

setDefaultWasm("node");

jest.setTimeout(60000);
const ci = !!process.env.CI;
export const SOLANA_HOST = ci
  ? "http://solana-devnet:8899"
  : "http://localhost:8899";
const SOLANA_PRIVATE_KEY = new Uint8Array([
  14, 173, 153, 4, 176, 224, 201, 111, 32, 237, 183, 185, 159, 247, 22, 161, 89,
  84, 215, 209, 212, 137, 10, 92, 157, 49, 29, 192, 101, 164, 152, 70, 87, 65,
  8, 174, 214, 157, 175, 126, 98, 90, 54, 24, 100, 177, 247, 77, 19, 112, 47,
  44, 165, 109, 233, 102, 14, 86, 109, 29, 134, 145, 132, 141,
]);
const SOLANA_CORE_BRIDGE_ADDRESS =
  "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";
const SOLANA_TOKEN_BRIDGE_ADDRESS =
  "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE";
const SPYMASTER = ci ? "spy:7072" : "localhost:7072";
const TEST_SOLANA_TOKEN = "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ";

test("Can spy on messages", (done) => {
  (async () => {
    // set up the spy client
    const client = createSpyRPCServiceClient(SPYMASTER);
    // subscribe to the stream of signedVAAs
    const stream = await subscribeSignedVAA(client, {});
    // register error callback to avoid crashing on .cancel()
    stream.addListener("error", (error: any) => {
      if (error.code === 1) {
        // Cancelled on client
        done();
      } else {
        done(error);
      }
    });
    // register data callback
    stream.addListener("data", ({}: { vaaBytes: any }) => {
      // cancel the stream to end the test
      stream.cancel();
    });
    // make a transaction which posts a message
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
    const txid = await connection.sendRawTransaction(transaction.serialize());
    await connection.confirmTransaction(txid);
  })();
});
