import { ethers } from "ethers";
import { describe, expect, test } from "@jest/globals";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { CHAIN_ID_ETH, CHAIN_ID_BSC, getSignedBatchVAAWithRetry } from "../..";
import {
  ETH_NODE_URL,
  BSC_NODE_URL,
  ETH_PRIVATE_KEY,
  MOCK_BATCH_VAA_SENDER_ABI,
  MOCK_BATCH_VAA_SENDER_ADDRESS,
  WORMHOLE_RPC_HOSTS,
} from "./consts";
import { parseWormholeEventsFromReceipt, getSignedVaaFromReceiptOnEth } from "./utils";

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

  // BSC VAAs
  let singleVAAPayload: ethers.BytesLike;
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

        // generate parameters for the batch VAA
        const nonce: number = 42000;

        // call mock contract and generate a batch VAA
        const receipt: ethers.ContractReceipt = await contractWithSigner
          .sendMultipleMessages(nonce, batchVAAPayloads, batchVAAConsistencyLevels)
          .then((tx: ethers.ContractTransaction) => tx.wait());

        // grab message events from the transaction logs
        const messageEvents = await parseWormholeEventsFromReceipt(receipt);

        // verify the payload in each VAA from the message events
        for (let i = 0; i < messageEvents.length; i++) {
          expect(messageEvents[i].name).toEqual("LogMessagePublished");
          expect(messageEvents[i].args.nonce).toEqual(nonce);
          expect(messageEvents[i].args.payload).toEqual(batchVAAPayloads[i]);
          expect(messageEvents[i].args.consistencyLevel).toEqual(batchVAAConsistencyLevels[i]);
        }

        // convert the hex string transactionHash to a byte array
        const transactionBytes = ethers.utils.arrayify(receipt.transactionHash)

        // fetch the batch VAA from the guardian
        const batchVaaRes = await getSignedBatchVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_ETH,
          transactionBytes,
          nonce,
          {
            transport: NodeHttpTransport(),
          }
        );
        // the Proto response type allows an empty response, so check for it
        if (!batchVaaRes.signedBatchVaa || !batchVaaRes.signedBatchVaa.batchVaa) {
          const err = new Error("received empty response from guardian for ETH batch.")
          console.error(err)
          throw err
        }
        encodedBatchVAAFromEth = batchVaaRes.signedBatchVaa.batchVaa;

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to generate a batch VAA on ethereum");
      }
    })();
  });

  test("Should Verify a Batch VAA From a Contract, on BSC", (done) => {
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
        const receipt: ethers.ContractReceipt = await contractWithSigner
          .consumeBatchVAA(encodedBatchVAAFromEth)
          .then((tx: ethers.ContractTransaction) => tx.wait());

        // query the mock contract and confirm that each payload was saved in the contract
        for (let i = 0; i < batchVAAPayloads.length; i++) {
          const payloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromEth.hashes[i]);
          expect(payloadFromContract).toEqual(batchVAAPayloads[i]);

          // clear the payload from the contract after verifying it
          const receipt: ethers.ContractReceipt = await contractWithSigner
            .clearPayload(parsedBatchVAAFromEth.hashes[i])
            .then((tx: ethers.ContractTransaction) => tx.wait());
          const emptyPayloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromEth.hashes[i]);
          expect(emptyPayloadFromContract).toEqual("0x");
        }

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to verify a batch VAA on BSC");
      }
    })();
  });

  test("Should Generate a VAAs (a Batch and Legacy VAA) on BSC", (done) => {
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

        // call mock contract and generate a batch VAA with one payload
        const receipt: ethers.ContractReceipt = await contractWithSigner
          .sendMultipleMessages(nonce, [singleVAAPayload], [consistencyLevel])
          .then((tx: ethers.ContractTransaction) => tx.wait());

        // grab message events from the transaction logs
        const messageEvents = await parseWormholeEventsFromReceipt(receipt);

        // verify the VAA payload info from the receipt logs
        expect(messageEvents[0].name).toEqual("LogMessagePublished");
        expect(messageEvents[0].args.nonce).toEqual(nonce);
        expect(messageEvents[0].args.payload).toEqual(singleVAAPayload);
        expect(messageEvents[0].args.consistencyLevel).toEqual(consistencyLevel);

        // convert the hex string transactionHash to a byte array
        const transactionBytes = ethers.utils.arrayify(receipt.transactionHash)

        // fetch the batch VAA from the guardian
        const batchVaaRes = await getSignedBatchVAAWithRetry(
          WORMHOLE_RPC_HOSTS,
          CHAIN_ID_BSC,
          transactionBytes,
          nonce,
          {
            transport: NodeHttpTransport(),
          }
        );
        // the Proto response type allows an empty response, so check for it
        if (!batchVaaRes.signedBatchVaa || !batchVaaRes.signedBatchVaa.batchVaa) {
          const err = new Error("received empty response from guardian for BSC batch.")
          console.error(err)
          throw err
        }
        encodedBatchVAAFromBsc = batchVaaRes.signedBatchVaa.batchVaa;

        // fetch the legacy VAA for the observation in the batch
        legacyVAAFromBSC = await getSignedVaaFromReceiptOnEth(receipt, CHAIN_ID_BSC, MOCK_BATCH_VAA_SENDER_ADDRESS);

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to generate a legacy VAA & batch VAA on BSC");
      }
    })();
  });

  test("Should Verify a Batch VAA (With a Single Observation) From a Contract, on Ethereum", (done) => {
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
        const receipt: ethers.ContractReceipt = await contractWithSigner
          .consumeBatchVAA(encodedBatchVAAFromBsc)
          .then((tx: ethers.ContractTransaction) => tx.wait());

        // query the mock contract and confirm that the payload was saved in the contract
        const payloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromBsc.hashes[0]);
        expect(payloadFromContract).toEqual(singleVAAPayload);

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to verify a batch VAA with single observation on ethereum");
      }
    })();
  });

  test("Should Verify a Legacy VAA From a Contract, on Ethereum", (done) => {
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

        const receipt: ethers.ContractReceipt = await contractWithSigner
          .clearPayload(parsedBatchVAAFromBsc.hashes[0])
          .then((tx: ethers.ContractTransaction) => tx.wait());

        // confirm that the payload was removed from the verifiedPayloads map
        const emptyPayloadFromContract = await contractWithSigner.getPayload(parsedBatchVAAFromBsc.hashes[0]);
        expect(emptyPayloadFromContract).toEqual("0x");

        // invoke the ETH contract to parse the legacy VAA and save the hash
        const parsedLegacyVAA = await contractWithSigner.parseVM(legacyVAAFromBSC);
        const legacyVAAHash = parsedLegacyVAA.hash;

        // verify the legacy VAA and confirm that the payload was saved in the contract
        const receipt2: ethers.ContractReceipt = await contractWithSigner
          .consumeSingleVAA(legacyVAAFromBSC)
          .then((tx: ethers.ContractTransaction) => tx.wait());
        const payloadFromContract = await contractWithSigner.getPayload(legacyVAAHash);
        expect(payloadFromContract).toEqual(parsedLegacyVAA.payload);

        // confirm that the payload was removed from the verifiedPayloads map
        const receipt3: ethers.ContractReceipt = await contractWithSigner
          .clearPayload(legacyVAAHash)
          .then((tx: ethers.ContractTransaction) => tx.wait());
        const emptyLegacyPayloadFromContract = await contractWithSigner.getPayload(legacyVAAHash);
        expect(emptyLegacyPayloadFromContract).toEqual("0x");

        // destory the provider and end the test
        provider.destroy();
        done();
      } catch (e) {
        console.error(e);
        done("An error occurred while trying to verify a legacy VAA on ethereum");
      }
    })();
  });
});
