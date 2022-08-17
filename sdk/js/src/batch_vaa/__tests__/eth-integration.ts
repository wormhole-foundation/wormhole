import {ethers} from "ethers";
import {describe, expect, test} from "@jest/globals";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  MOCK_BATCH_VAA_SENDER_ABI,
  MOCK_BATCH_VAA_SENDER_ADDRESS,
  WORMHOLE_MESSAGE_EVENT_ABI,
} from "./consts";
import {parseWormholeEventsFromReceipt} from "./utils";

describe("Batch VAAs", () => {
  test("Should Generate a Batch VAA From a Contract", (done) => {
    (async () => {
      try {
        // create a signer for Eth
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);

        // create a contract instance for the mock batch VAA sender contract
        const mockBatchSenderContract = new ethers.Contract(
          MOCK_BATCH_VAA_SENDER_ADDRESS,
          MOCK_BATCH_VAA_SENDER_ABI,
          provider
        );
        const contractWithSigner = mockBatchSenderContract.connect(signer);

        // generate parameters for the batch VAA generation
        const nonce: number = 42000;
        const payload: ethers.BytesLike = ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff"));
        const consistencyLevel: number = 15;

        // call mock contract and generate a batch VAA
        const tx = await contractWithSigner.sendMultipleMessages(nonce, payload, consistencyLevel);
        const receipt: ethers.ContractReceipt = await tx.wait();

        // grab message events from the transaction logs
        const messageEvents = await parseWormholeEventsFromReceipt(receipt);

        // verify the payload in each VAA from the message events
        for (let i = 0; i < messageEvents.length; i++) {
          expect(messageEvents[i].name).toEqual("LogMessagePublished");
          expect(messageEvents[i].args.nonce).toEqual(nonce);
          expect(messageEvents[i].args.payload).toEqual(payload);
          expect(messageEvents[i].args.consistencyLevel).toEqual(consistencyLevel);
        }

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to generate a batch VAA on ethereum");
      }
    })();
  });
});
