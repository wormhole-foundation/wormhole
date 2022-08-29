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
import {
  parseWormholeEventsFromReceipt,
  getSignedBatchVaaFromReceiptOnEth,
  getSignedVaaFromReceiptOnEth,
  removeObservationFromBatch,
} from "./utils";

describe("Batch VAAs", () => {
  // The following tests rely on the payloads in batchVAAPayloads.
  // Be cautious when making changes to the payloads in the batchVAAPayloads array.
  // Adding or removing payloads will impact the results the of tests.
  const batchVAAPayloads: ethers.BytesLike[] = [
    ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff0")),
    ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff1")),
    ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff2")),
    ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff3")),
  ];
  const batchVAAConsistencyLevels: number[] = [15, 10, 2, 15];

  // ETH VAAs
  let encodedBatchVAAFromEth: ethers.BytesLike;
  let encodedPartialBatchVAAForEth: ethers.BytesLike;

  // BSC VAAs
  let singleVAAPayload: ethers.BytesLike;
  let encodedBatchVAAFromBsc: ethers.BytesLike;
  let legacyVAAFromBSC: Uint8Array;
  let encodedPartialBatchVAAForBSC: ethers.BytesLike;

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

        // generate parameters for the batch VAA
        const nonce: number = 42000;

        // call mock contract and generate a batch VAA
        const tx = await contractWithSigner.sendMultipleMessages(nonce, batchVAAPayloads, batchVAAConsistencyLevels);
        const receipt: ethers.ContractReceipt = await tx.wait();

        // grab message events from the transaction logs
        const messageEvents = await parseWormholeEventsFromReceipt(receipt);

        // verify the payload in each VAA from the message events
        for (let i = 0; i < messageEvents.length; i++) {
          expect(messageEvents[i].name).toEqual("LogMessagePublished");
          expect(messageEvents[i].args.nonce).toEqual(nonce);
          expect(messageEvents[i].args.payload).toEqual(batchVAAPayloads[i]);
          expect(messageEvents[i].args.consistencyLevel).toEqual(batchVAAConsistencyLevels[i]);
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
        // confirm that the VAA has the expected number of hashes.
        const parsedBatchVAAFromEth = await contractWithSigner.parseBatchVM(encodedBatchVAAFromEth);
        expect(parsedBatchVAAFromEth.hashes.length).toEqual(batchVAAPayloads.length);

        // Invoke the BSC contract that consumes the batch VAA.
        // This function call will parse and verify the batch,
        // and save each verified message's payload in a map.
        await contractWithSigner.consumeBatchVAA(encodedBatchVAAFromEth);

        // query the mock contract and confirm that each payload was saved in the contract
        for (let i = 0; i < batchVAAPayloads.length; i++) {
          const payloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromEth.hashes[i]);
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
        singleVAAPayload = ethers.utils.hexlify(ethers.utils.toUtf8Bytes("SuperCoolCrossChainStuff4"));

        // call mock contract and generate a batch VAA
        const tx = await contractWithSigner.sendMultipleMessages(nonce, [singleVAAPayload], [consistencyLevel]);
        const receipt: ethers.ContractReceipt = await tx.wait();

        // grab message events from the transaction logs
        const messageEvents = await parseWormholeEventsFromReceipt(receipt);

        // verify the VAA payload info from the receipt logs
        expect(messageEvents[0].name).toEqual("LogMessagePublished");
        expect(messageEvents[0].args.nonce).toEqual(nonce);
        expect(messageEvents[0].args.payload).toEqual(singleVAAPayload);
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
        // confirm that the VAA has the expected number of hashes.
        const parsedBatchVAAFromBsc = await contractWithSigner.parseBatchVM(encodedBatchVAAFromBsc);
        expect(parsedBatchVAAFromBsc.hashes.length).toEqual(1); // there's only one VAA in the batch

        // Invoke the ETH contract that consumes the batch VAA.
        // This function call will parse and verify the batch,
        // and save the verified message's payload in a map.
        await contractWithSigner.consumeBatchVAA(encodedBatchVAAFromBsc);

        // query the mock contract and confirm that the payload was saved in the contract
        const payloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromBsc.hashes[0]);
        expect(payloadFromContract).toEqual(singleVAAPayload);

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to generate a batch VAA on ethereum");
      }
    })();
  });

  test("Should Verify a Legacy VAA From a Contract on Ethereum", (done) => {
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
        const parsedBatchVAAFromBsc = await contractWithSigner.parseBatchVM(encodedBatchVAAFromBsc);
        await contractWithSigner.clearPayload(parsedBatchVAAFromBsc.hashes[0]);

        // confirm that the payload was removed from the verifiedPayloads map
        const emptyPayloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromBsc.hashes[0]);
        expect(emptyPayloadFromContract).toEqual("0x");

        // invoke the ETH contract to parse the legacy VAA and save the hash
        const parsedLegacyVAA = await contractWithSigner.parseVM(legacyVAAFromBSC);
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

  test("Should Convert a Batch VAA From Ethereum Into Two Partial Batch VAAs", (done) => {
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

        // parse the original batch VAA from Eth to confirm values in the partial batches
        const parsedBatchVAAFromEth = await contractWithSigner.parseBatchVM(encodedBatchVAAFromEth);

        // Create a partial batch intended to be submitted on BSC by
        // removing the last observation from the original batch VAA
        // created on Eth. Index 3 is the last observation, since
        // there are only four VAAs in the batch.
        const removedIndex: number = 3;
        encodedPartialBatchVAAForBSC = removeObservationFromBatch(removedIndex, encodedBatchVAAFromEth);

        // parse the partial batch VAA and sanity check the values
        const parsedPartialBatchVAAForBSC = await contractWithSigner.parseBatchVM(encodedPartialBatchVAAForBSC);
        expect(parsedPartialBatchVAAForBSC.indexedObservations.length).toEqual(
          parsedBatchVAAFromEth.indexedObservations.length - 1
        );
        expect(parsedPartialBatchVAAForBSC.hashes.length).toEqual(parsedBatchVAAFromEth.hashes.length);

        for (let i = 0; i < parsedPartialBatchVAAForBSC.indexedObservations.length; i++) {
          expect(parsedPartialBatchVAAForBSC.indexedObservations[i].observation).toEqual(
            parsedBatchVAAFromEth.indexedObservations[i].observation
          );
        }

        // Create a partial batch intended to be submitted on Ethereum by
        // removing the first three observations from the original batch VAA
        // created on BSC.
        const numObservationsToRemove: number = 3;
        encodedPartialBatchVAAForEth = encodedBatchVAAFromEth;
        for (let i = 0; i < numObservationsToRemove; i++) {
          const removedIndex: number = i;
          encodedPartialBatchVAAForEth = removeObservationFromBatch(removedIndex, encodedPartialBatchVAAForEth);
        }

        // parse the partial batch VAA and sanity check the values
        const parsedPartialBatchVAAForEth = await contractWithSigner.parseBatchVM(encodedPartialBatchVAAForEth);
        expect(parsedPartialBatchVAAForEth.indexedObservations.length).toEqual(
          parsedBatchVAAFromEth.indexedObservations.length - numObservationsToRemove
        );
        expect(parsedPartialBatchVAAForEth.hashes.length).toEqual(parsedBatchVAAFromEth.hashes.length);

        // Check to see if the remaining observation is the same as the
        // last observation in the orignal batch VAA.
        expect(parsedPartialBatchVAAForEth.indexedObservations[0].observation).toEqual(
          parsedBatchVAAFromEth.indexedObservations[parsedBatchVAAFromEth.indexedObservations.length - 1].observation
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

  test("Should Verify a Partial Batch VAA From a Contract on BSC", (done) => {
    (async () => {
      try {
        // create a signer for BSC
        const provider = new ethers.providers.WebSocketProvider(BSC_NODE_URL);
        const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);

        // create a contract instance for the mock batch VAA sender contract
        const mockBatchSenderContractOnEth = new ethers.Contract(
          MOCK_BATCH_VAA_SENDER_ADDRESS,
          MOCK_BATCH_VAA_SENDER_ABI,
          provider
        );
        const contractWithSigner = mockBatchSenderContractOnEth.connect(signer);

        // parse the partial batch VAA to save the relevant hashes (hashes of remaining indexedObservations)
        const parsedPartialBatchVAAForBSC = await contractWithSigner.parseBatchVM(encodedPartialBatchVAAForBSC);
        const partialBatchHashes: string[] = [];

        for (let i = 0; i < parsedPartialBatchVAAForBSC.indexedObservations.length; i++) {
          const relevantHash: string =
            parsedPartialBatchVAAForBSC.hashes[parsedPartialBatchVAAForBSC.indexedObservations[i].index];
          partialBatchHashes.push(relevantHash);
        }
        expect(partialBatchHashes.length).toEqual(3);

        // Clear the relevant payloads from the contract before consuming the partial batch VAA,
        // since they were saved in earlier tests.
        for (const hash of partialBatchHashes) {
          await contractWithSigner.clearPayload(hash);

          // confirm the payload was cleared
          const payloadFromContract = await contractWithSigner.getPayload(hash);
          expect(payloadFromContract).toEqual("0x");
        }

        // Invoke the BSC contract that consumes the partial batch VAA.
        // This function call will parse and verify the batch,
        // and save each verified message's payload in a map.
        await contractWithSigner.consumeBatchVAA(encodedPartialBatchVAAForBSC);

        // query the mock contract and confirm that each payload was saved in the contract
        for (let i = 0; i < partialBatchHashes.length; i++) {
          const payloadFromContract = await contractWithSigner.getPayload(partialBatchHashes[i]);
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

  test("Should Verify a Partial Batch VAA From a Contract on Ethereum", (done) => {
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

        // parse the partial batch VAA to save the relevant hashes (hashes of remaining indexedObservations)
        const parsedPartialBatchVAAForEth = await contractWithSigner.parseBatchVM(encodedPartialBatchVAAForEth);
        const partialBatchHashes: string[] = [];

        for (let i = 0; i < parsedPartialBatchVAAForEth.indexedObservations.length; i++) {
          const relevantHash: string =
            parsedPartialBatchVAAForEth.hashes[parsedPartialBatchVAAForEth.indexedObservations[i].index];
          partialBatchHashes.push(relevantHash);
        }
        expect(partialBatchHashes.length).toEqual(1);

        // Clear the relevant payloads from the contract before consuming the partial batch VAA,
        // since they were saved in earlier tests.
        for (const hash of partialBatchHashes) {
          await contractWithSigner.clearPayload(hash);

          // confirm the payload was cleared
          const payloadFromContract = await contractWithSigner.getPayload(hash);
          expect(payloadFromContract).toEqual("0x");
        }

        // Invoke the Eth contract that consumes the partial batch VAA.
        // This function call will parse and verify the batch,
        // and save each verified message's payload in a map.
        await contractWithSigner.consumeBatchVAA(encodedPartialBatchVAAForEth);

        // Query the mock contract and confirm that the payload was saved in the contract.
        // There should only be one payload saved for this partial batch VAA. It should
        // be the last payload in batchVAAPayloads, since the first three observations
        // were removed from the batch VAA.
        const payloadFromContract = await contractWithSigner.getPayload(partialBatchHashes[0]);
        expect(payloadFromContract).toEqual(batchVAAPayloads[batchVAAPayloads.length - 1]);

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
