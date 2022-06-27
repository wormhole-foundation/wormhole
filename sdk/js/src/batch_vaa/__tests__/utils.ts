import {ethers} from "ethers";
import {describe, it} from "@jest/globals";
import {ChainId, getSignedVAAWithRetry, getEmitterAddressEth} from "../..";
import {WORMHOLE_MESSAGE_EVENT_ABI, WORMHOLE_RPC_HOSTS} from "./consts";
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
