import {ethers} from "ethers";
import {describe, expect, test} from "@jest/globals";
import {CHAIN_ID_ETH, CHAIN_ID_BSC} from "../..";
import {
  ETH_NODE_URL,
  BSC_NODE_URL,
  ETH_PRIVATE_KEY,
  MOCK_BATCH_VAA_SENDER_ABI,
  MOCK_BATCH_VAA_SENDER_ADDRESS,
  MOCK_BATCH_VAA_SENDER_ADDRESS_BYTES,
} from "./consts";
import {parseWormholeEventsFromReceipt, getSignedBatchVaaFromReceiptOnEth, getSignedVaaFromReceiptOnEth} from "./utils";

describe("Batch VAAs", () => {
  // ETH VAAs
  let encodedBatchVAAFromEth: ethers.BytesLike;
  let batchVAAPayloads: ethers.BytesLike[] = [];

  // BSC VAAs
  let singleVAAPaylod: ethers.BytesLike;
  let encodedBatchVAAFromBsc: ethers.BytesLike;
  let legacyVAAFromBSC: Uint8Array;

  test("Should Generate a Batch VAA From a Contract on Ethereum", (done) => {
    (async () => {
      try {
        // create a signer for ETH
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);

        // create a contract instance for the mock batch VAA sender contract
        const mockBatchSenderContractOnEth = new ethers.Contract(
          MOCK_BATCH_VAA_SENDER_ADDRESS,
          MOCK_BATCH_VAA_SENDER_ABI,
          provider
        );
        const contractWithSigner = mockBatchSenderContractOnEth.connect(signer);

        // generate parameters for the batch VAA generation
        const nonce: number = 42000;
        const consistencyLevels: number[] = [15, 10, 2, 15];
        batchVAAPayloads = [
          ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff0")),
          ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff1")),
          ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff2")),
          ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff3")),
        ];

        // call mock contract and generate a batch VAA
        const tx = await contractWithSigner.sendMultipleMessages(nonce, batchVAAPayloads, consistencyLevels);
        const receipt: ethers.ContractReceipt = await tx.wait();

        // grab message events from the transaction logs
        const messageEvents = await parseWormholeEventsFromReceipt(receipt);

        // verify the payload in each VAA from the message events
        for (let i = 0; i < messageEvents.length; i++) {
          expect(messageEvents[i].name).toEqual("LogMessagePublished");
          expect(messageEvents[i].args.nonce).toEqual(nonce);
          expect(messageEvents[i].args.payload).toEqual(batchVAAPayloads[i]);
          expect(messageEvents[i].args.consistencyLevel).toEqual(consistencyLevels[i]);
        }

        // REVIEW: this will be replaced with a call to fetch the real batch VAA
        // simulate fetching the batch VAA
        encodedBatchVAAFromEth = await getSignedBatchVaaFromReceiptOnEth(
          receipt,
          CHAIN_ID_ETH,
          MOCK_BATCH_VAA_SENDER_ADDRESS_BYTES,
          0 // guardianSetIndex
        );

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to generate a batch VAA on ethereum");
      }
    })();
  });

  test("Should Verify a Batch VAA From a Contract on BSC", (done) => {
    (async () => {
      try {
        // create a signer for BSC
        const provider = new ethers.providers.WebSocketProvider(BSC_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);

        // create a contract instance for the mock batch VAA sender contract
        const mockBatchSenderContractOnBsc = new ethers.Contract(
          MOCK_BATCH_VAA_SENDER_ADDRESS,
          MOCK_BATCH_VAA_SENDER_ABI,
          provider
        );
        const contractWithSigner = mockBatchSenderContractOnBsc.connect(signer);

        // Invoke the BSC contract to parse the encoded batch VAA and
        // confirm that the VAA has the expected number of hashes
        const parsedBatchVAAFromEth = await contractWithSigner.parseBatchVAA(encodedBatchVAAFromEth);
        expect(parsedBatchVAAFromEth.header.hashes.length).toEqual(batchVAAPayloads.length);

        // Invoke the BSC contract that consumes the batch VAA.
        // This function call will parse and verify the batch,
        // and save each verified message's payload in a map.
        await contractWithSigner.consumeBatchVAA(encodedBatchVAAFromEth);

        // query the mock contract and confirm that each payload was saved in the contract
        for (let i = 0; i < batchVAAPayloads.length; i++) {
          const payloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromEth.header.hashes[i]);
          expect(payloadFromContract).toEqual(batchVAAPayloads[i]);
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

  test("Should Generate a Single VAA (a Batch and Legacy VAA) From a Contract on BSC", (done) => {
    (async () => {
      try {
        // create a signer for BSC
        const provider = new ethers.providers.WebSocketProvider(BSC_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);

        // create a contract instance for the mock batch VAA sender contract
        const mockBatchSenderContractOnBsc = new ethers.Contract(
          MOCK_BATCH_VAA_SENDER_ADDRESS,
          MOCK_BATCH_VAA_SENDER_ABI,
          provider
        );
        const contractWithSigner = mockBatchSenderContractOnBsc.connect(signer);

        // create a batch with a single observation
        // generate parameters for the batch VAA generation
        const nonce: number = 42000;
        const consistencyLevel: number = 2;
        singleVAAPaylod = ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff4"));

        // call mock contract and generate a batch VAA
        const tx = await contractWithSigner.sendMultipleMessages(nonce, [singleVAAPaylod], [consistencyLevel]);
        const receipt: ethers.ContractReceipt = await tx.wait();

        // grab message events from the transaction logs
        const messageEvents = await parseWormholeEventsFromReceipt(receipt);

        // verify the VAA payload info from the receipt logs
        expect(messageEvents[0].name).toEqual("LogMessagePublished");
        expect(messageEvents[0].args.nonce).toEqual(nonce);
        expect(messageEvents[0].args.payload).toEqual(singleVAAPaylod);
        expect(messageEvents[0].args.consistencyLevel).toEqual(consistencyLevel);

        // REVIEW: this will be replaced with a call to fetch the real batch VAA
        // simulate fetching the batch VAA
        encodedBatchVAAFromBsc = await getSignedBatchVaaFromReceiptOnEth(
          receipt,
          CHAIN_ID_BSC,
          MOCK_BATCH_VAA_SENDER_ADDRESS_BYTES,
          0 // guardianSetIndex
        );

        // fetch the legacy VAA for the observation in the batch
        legacyVAAFromBSC = await getSignedVaaFromReceiptOnEth(receipt, CHAIN_ID_BSC, MOCK_BATCH_VAA_SENDER_ADDRESS);

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to generate a batch VAA on ethereum");
      }
    })();
  });

  test("Should Verify a Batch VAA (With a Single Observation) From a Contract on Ethereum", (done) => {
    (async () => {
      try {
        // create a signer for ETH
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);

        // create a contract instance for the mock batch VAA sender contract
        const mockBatchSenderContractOnEth = new ethers.Contract(
          MOCK_BATCH_VAA_SENDER_ADDRESS,
          MOCK_BATCH_VAA_SENDER_ABI,
          provider
        );
        const contractWithSigner = mockBatchSenderContractOnEth.connect(signer);

        // Invoke the ETH contract to parse the encoded batch VAA and
        // confirm that the VAA has the expected number of hashes
        const parsedBatchVAAFromBsc = await contractWithSigner.parseBatchVAA(encodedBatchVAAFromBsc);
        expect(parsedBatchVAAFromBsc.header.hashes.length).toEqual(1); // there's only one VAA in the batch

        // Invoke the ETH contract that consumes the batch VAA.
        // This function call will parse and verify the batch,
        // and save the verified message's payload in a map.
        await contractWithSigner.consumeBatchVAA(encodedBatchVAAFromBsc);

        // query the mock contract and confirm that the payload was saved in the contract
        const payloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromBsc.header.hashes[0]);
        expect(payloadFromContract).toEqual(singleVAAPaylod);

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to generate a batch VAA on ethereum");
      }
    })();
  });

  test("Should Verify a Legacy VAA From a Contract on Ethereum Using parseAndVerifyVAA", (done) => {
    (async () => {
      try {
        // create a signer for ETH
        const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);

        // create a contract instance for the mock batch VAA sender contract
        const mockBatchSenderContractOnEth = new ethers.Contract(
          MOCK_BATCH_VAA_SENDER_ADDRESS,
          MOCK_BATCH_VAA_SENDER_ABI,
          provider
        );
        const contractWithSigner = mockBatchSenderContractOnEth.connect(signer);

        // Invoke the ETH contract to parse the encoded batch VAA. Grab the hash
        // and clear the verifiedPayloads map.
        const parsedBatchVAAFromBsc = await contractWithSigner.parseBatchVAA(encodedBatchVAAFromBsc);
        await contractWithSigner.clearPayload(parsedBatchVAAFromBsc.header.hashes[0]);

        // confirm that the payload was removed from the verifiedPayloads map
        const emptyPayloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromBsc.header.hashes[0]);
        expect(emptyPayloadFromContract).toEqual("0x");

        // invoke the ETH contract to parse the legacy VAA and save the hash
        const parsedLegacyVAA = await contractWithSigner.parseLegacyVAA(legacyVAAFromBSC);
        const legacyVAAHash = parsedLegacyVAA.hash;

        // verify the legacy VAA and confirm that the payload was saved in the contract
        await contractWithSigner.consumeSingleVAA(legacyVAAFromBSC);
        const payloadFromContract = await contractWithSigner.getPayload(legacyVAAHash);
        expect(payloadFromContract).toEqual(parsedLegacyVAA.payload);

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
