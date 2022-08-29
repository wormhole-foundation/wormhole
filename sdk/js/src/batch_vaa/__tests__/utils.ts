import {ethers} from "ethers";
import {describe, it} from "@jest/globals";
import {ChainId, getSignedVAAWithRetry, getEmitterAddressEth} from "../..";
import {WORMHOLE_MESSAGE_EVENT_ABI, SIGNER_PRIVATE_KEY, WORMHOLE_RPC_HOSTS} from "./consts";
const elliptic = require("elliptic");
import {NodeHttpTransport} from "@improbable-eng/grpc-web-node-http-transport";

export async function parseWormholeEventsFromReceipt(
  receipt: ethers.ContractReceipt
): Promise<ethers.utils.LogDescription[]> {
  // create the wormhole message interface
  const wormholeMessageInterface = new ethers.utils.Interface(WORMHOLE_MESSAGE_EVENT_ABI);

  // loop through the logs and parse the events that were emitted
  const logDescriptions: ethers.utils.LogDescription[] = await Promise.all(
    receipt.logs.map(async (log) => {
      return wormholeMessageInterface.parseLog(log);
    })
  );

  return logDescriptions;
}

export function doubleKeccak256(body: ethers.BytesLike) {
  return ethers.utils.keccak256(ethers.utils.keccak256(body));
}

function zeroPadBytes(value: string, length: number): string {
  while (value.length < 2 * length) {
    value = "0" + value;
  }
  return value;
}

export async function getSignedBatchVaaFromReceiptOnEth(
  receipt: ethers.ContractReceipt,
  emitterChainId: ChainId,
  emitterAddress: ethers.BytesLike,
  guardianSetIndex: number
): Promise<ethers.BytesLike> {
  // grab each message from the transaction logs
  const messageEvents = await parseWormholeEventsFromReceipt(receipt);

  // create a timestamp for the
  const timestamp = Math.floor(+new Date() / 1000);

  let observationHashes = "";
  let encodedObservationsWithLengthPrefix = "";
  for (let i = 0; i < messageEvents.length; i++) {
    const event = messageEvents[i];

    // encode the observation
    const encodedObservation = ethers.utils.solidityPack(
      ["uint32", "uint32", "uint16", "bytes32", "uint64", "uint8", "bytes"],
      [
        timestamp,
        event.args.nonce,
        emitterChainId,
        emitterAddress,
        event.args.sequence,
        event.args.consistencyLevel,
        event.args.payload,
      ]
    );

    // compute the hash of the observation
    const hash = doubleKeccak256(encodedObservation);
    observationHashes += hash.substring(2);

    // grab the index, and length of the observation and add them to the observation bytestring
    // divide observationBytes by two to convert string representation length to bytes
    const observationElements = [
      ethers.utils.solidityPack(["uint8"], [i]).substring(2),
      ethers.utils.solidityPack(["uint32"], [encodedObservation.substring(2).length / 2]).substring(2),
      encodedObservation.substring(2),
    ];
    encodedObservationsWithLengthPrefix += observationElements.join("");
  }

  // compute the has of batch hashes - hash(hash(VAA1), hash(VAA2), ...)
  const batchHash = doubleKeccak256("0x" + observationHashes);

  // sign the batchHash
  const ec = new elliptic.ec("secp256k1");
  const key = ec.keyFromPrivate(SIGNER_PRIVATE_KEY);
  const signature = key.sign(batchHash.substring(2), {canonical: true});

  // create the signature
  const packSig = [
    ethers.utils.solidityPack(["uint8"], [0]).substring(2),
    zeroPadBytes(signature.r.toString(16), 32),
    zeroPadBytes(signature.s.toString(16), 32),
    ethers.utils.solidityPack(["uint8"], [signature.recoveryParam]).substring(2),
  ];
  const signatures = packSig.join("");

  const vm = [
    // this is a type 2 VAA since it's a batch
    ethers.utils.solidityPack(["uint8"], [2]).substring(2),
    ethers.utils.solidityPack(["uint32"], [guardianSetIndex]).substring(2), // guardianSetIndex
    ethers.utils.solidityPack(["uint8"], [1]).substring(2), // number of signers
    signatures,
    ethers.utils.solidityPack(["uint8"], [messageEvents.length]).substring(2),
    observationHashes,
    ethers.utils.solidityPack(["uint8"], [messageEvents.length]).substring(2),
    encodedObservationsWithLengthPrefix,
  ].join("");

  return "0x" + vm;
}

export async function getSignedVaaFromReceiptOnEth(
  receipt: ethers.ContractReceipt,
  emitterChainId: ChainId,
  contractAddress: ethers.BytesLike
): Promise<Uint8Array> {
  const messageEvents = await parseWormholeEventsFromReceipt(receipt);

  // grab the sequence from the parsed message log
  if (messageEvents.length !== 1) {
    throw Error("more than one message found in log");
  }
  const sequence = messageEvents[0].args.sequence;

  // fetch the signed VAA
  const result = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    emitterChainId,
    getEmitterAddressEth(contractAddress),
    sequence.toString(),
    {
      transport: NodeHttpTransport(),
    }
  );
  return result.vaaBytes;
}

export function removeObservationFromBatch(indexToRemove: number, encodedVM: ethers.BytesLike): ethers.BytesLike {
  // index of the signature count (number of signers for the VM)
  let index: number = 5;

  // grab the signature count
  const sigCount: number = parseInt(ethers.utils.hexDataSlice(encodedVM, index, index + 1));
  index += 1;

  // skip the signatures
  index += 66 * sigCount;

  // hash count
  const hashCount: number = parseInt(ethers.utils.hexDataSlice(encodedVM, index, index + 1));
  index += 1;

  // skip the hashes
  index += 32 * hashCount;

  // observation count
  const observationCount: number = parseInt(ethers.utils.hexDataSlice(encodedVM, index, index + 1));
  const observationCountIndex: number = index; // save the index
  index += 1;

  // find the index of the observation that will be removed
  let bytesRangeToRemove: number[] = [0, 0];
  for (let i = 0; i < observationCount; i++) {
    const observationStartIndex = index;

    // parse the observation index and the observation length
    const observationIndex: number = parseInt(ethers.utils.hexDataSlice(encodedVM, index, index + 1));
    index += 1;

    const observationLen: number = parseInt(ethers.utils.hexDataSlice(encodedVM, index, index + 4));
    index += 4;

    // save the index of the observation we want to remove
    if (observationIndex == indexToRemove) {
      bytesRangeToRemove[0] = observationStartIndex;
      bytesRangeToRemove[1] = observationStartIndex + 5 + observationLen;
    }
    index += observationLen;
  }

  // remove the observation by slicing the original byte array
  const newEncodedVMByteArray: ethers.BytesLike[] = [
    ethers.utils.hexDataSlice(encodedVM, 0, observationCountIndex),
    ethers.utils.hexlify([observationCount - 1]),
    ethers.utils.hexDataSlice(encodedVM, observationCountIndex + 1, bytesRangeToRemove[0]),
    ethers.utils.hexDataSlice(encodedVM, bytesRangeToRemove[1], encodedVM.length),
  ];
  return ethers.utils.hexConcat(newEncodedVMByteArray);
}

describe("Utils Should Exist", () => {
  it("Dummy Test", () => {
    return;
  });
});
