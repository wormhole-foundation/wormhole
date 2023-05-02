import {
  ChainId,
  CHAIN_ID_TO_NAME,
  Network,
  tryNativeToHexString,
} from "../../";
import { BigNumber, ContractReceipt, ethers } from "ethers";
import { getWormholeRelayer, RPCS_BY_CHAIN } from "../consts";
import {
  parseWormholeRelayerPayloadType,
  parseOverrideInfoFromDeliveryEvent,
  RelayerPayloadId,
  parseWormholeRelayerSend,
  DeliveryInstruction,
  DeliveryStatus,
  RefundStatus,
  VaaKey,
  DeliveryOverrideArgs
} from "../structs";
import { RelayProvider } from "../../ethers-contracts/RelayProvider";
import { RelayProvider__factory } from "../../ethers-contracts/factories/RelayProvider__factory";
import { Implementation__factory } from "../../ethers-contracts/factories/Implementation__factory";
import {
  DeliveryEvent,
  IWormholeRelayer,
} from "../../ethers-contracts/CoreRelayer";

type DeliveryTargetInfo = {
  status: DeliveryStatus | string;
  deliveryTxHash: string | null;
  vaaHash: string | null;
  sourceChain: number | null;
  sourceVaaSequence: BigNumber | null;
  gasUsed: number;
  refundStatus: RefundStatus;
  leftoverTransactionFee?: number; // Only defined if status is FORWARD_REQUEST_SUCCESS
  revertData?: string; // Only defined if status is RECEIVER_FAILURE or FORWARD_REQUEST_FAILURE
  overrides?: DeliveryOverrideArgs;
};

export function parseWormholeLog(log: ethers.providers.Log): {
  type: RelayerPayloadId;
  parsed: DeliveryInstruction | string;
} {
  const abi = [
    "event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel);",
  ];
  const iface = new ethers.utils.Interface(abi);
  const parsed = iface.parseLog(log);
  const payload = Buffer.from(parsed.args.payload.substring(2), "hex");
  const type = parseWormholeRelayerPayloadType(payload);
  if (type == RelayerPayloadId.Delivery) {
    return { type, parsed: parseWormholeRelayerSend(payload) };
  } else {
    throw Error("Invalid wormhole log");
  }
}

export function printChain(chainId: number) {
  return `${CHAIN_ID_TO_NAME[chainId as ChainId]} (Chain ${chainId})`;
}

export function getDefaultProvider(network: Network, chainId: ChainId) {
  return new ethers.providers.StaticJsonRpcProvider(
    RPCS_BY_CHAIN[network][CHAIN_ID_TO_NAME[chainId]]
  );
}

export function getRelayProvider(
  address: string,
  provider: ethers.providers.Provider
): RelayProvider {
  const contract = RelayProvider__factory.connect(address, provider);
  return contract;
}

export function getBlockRange(
  provider: ethers.providers.Provider,
  timestamp?: number
): [ethers.providers.BlockTag, ethers.providers.BlockTag] {
  return [-2040, "latest"];
}

export async function getWormholeRelayerInfoBySourceSequence(
  environment: Network,
  targetChain: ChainId,
  targetChainProvider: ethers.providers.Provider,
  sourceChain: number,
  sourceVaaSequence: BigNumber,
  blockStartNumber: ethers.providers.BlockTag,
  blockEndNumber: ethers.providers.BlockTag
): Promise<{chainId: ChainId, events: {status: string, transactionHash: string | null}[]}> {
  const deliveryEvents = await getWormholeRelayerDeliveryEventsBySourceSequence(
    environment,
    targetChain,
    targetChainProvider,
    sourceChain,
    sourceVaaSequence,
    blockStartNumber,
    blockEndNumber
  );
  if (deliveryEvents.length == 0) {
    let status = `Delivery didn't happen on ${printChain(
      targetChain
    )} within blocks ${blockStartNumber} to ${blockEndNumber}.`;
    try {
      const blockStart = await targetChainProvider.getBlock(blockStartNumber);
      const blockEnd = await targetChainProvider.getBlock(blockEndNumber);
      status = `Delivery didn't happen on ${printChain(
        targetChain
      )} within blocks ${blockStart.number} to ${
        blockEnd.number
      } (within times ${new Date(
        blockStart.timestamp * 1000
      ).toString()} to ${new Date(blockEnd.timestamp * 1000).toString()})`;
    } catch (e) {}
    deliveryEvents.push({
      status,
      deliveryTxHash: null,
      vaaHash: null,
      sourceChain: sourceChain,
      sourceVaaSequence,
      gasUsed: 0,
      refundStatus: RefundStatus.RefundFail
    });
  }
  const targetChainStatus = {
    chainId: targetChain,
    events: deliveryEvents.map((e) => ({
      status: e.status,
      transactionHash: e.deliveryTxHash,
    })),
  };

  return targetChainStatus;
}

export async function getWormholeRelayerDeliveryEventsBySourceSequence(
  environment: Network,
  targetChain: ChainId,
  targetChainProvider: ethers.providers.Provider,
  sourceChain: number,
  sourceVaaSequence: BigNumber,
  blockStartNumber: ethers.providers.BlockTag,
  blockEndNumber: ethers.providers.BlockTag
): Promise<DeliveryTargetInfo[]> {
  const coreRelayer = getWormholeRelayer(
    targetChain,
    environment,
    targetChainProvider
  );

  const deliveryEvents = coreRelayer.filters.Delivery(
    null,
    sourceChain,
    sourceVaaSequence
  );

  // There is a max limit on RPCs sometimes for how many blocks to query
  return await transformDeliveryEvents(
    await coreRelayer.queryFilter(
      deliveryEvents,
      blockStartNumber,
      blockEndNumber
    ),
    targetChainProvider
  );
}

export function deliveryStatus(status: number) {
  switch (status) {
    case 0:
      return DeliveryStatus.DeliverySuccess;
    case 1:
      return DeliveryStatus.ReceiverFailure;
    case 2:
      return DeliveryStatus.ForwardRequestFailure;
    case 3:
      return DeliveryStatus.ForwardRequestSuccess;
    default:
      return DeliveryStatus.ThisShouldNeverHappen;
  }
}

async function transformDeliveryEvents(
  events: DeliveryEvent[],
  targetProvider: ethers.providers.Provider
): Promise<DeliveryTargetInfo[]> {
  return Promise.all(
    events.map(async (x) => {
      console.log(x.args[8]);
      return {
        status: deliveryStatus(x.args[4]),
        deliveryTxHash: x.transactionHash,
        vaaHash: x.args[3],
        sourceVaaSequence: x.args[2],
        sourceChain: x.args[1],
        gasUsed: x.args[5],
        refundStatus: x.args[6],
        overridesInfo: (Buffer.from(x.args[8], "hex").length > 0) && parseOverrideInfoFromDeliveryEvent(Buffer.from(x.args[8], "hex"))
      };
    })
  );
}

export function getWormholeRelayerLog(
  receipt: ContractReceipt,
  bridgeAddress: string,
  emitterAddress: string,
  index: number
): { log: ethers.providers.Log; sequence: string } {
  const bridgeLogs = receipt.logs.filter((l) => {
    return l.address === bridgeAddress;
  });

  if (bridgeLogs.length == 0) {
    throw Error("No core contract interactions found for this transaction.");
  }

  const parsed = bridgeLogs.map((bridgeLog) => {
    const log = Implementation__factory.createInterface().parseLog(bridgeLog);
    return {
      sequence: log.args[1].toString(),
      nonce: log.args[2].toString(),
      emitterAddress: tryNativeToHexString(log.args[0].toString(), "ethereum"),
      log: bridgeLog,
    };
  });

  const filtered = parsed.filter(
    (x) => x.emitterAddress == emitterAddress.toLowerCase()
  );

  if (filtered.length == 0) {
    throw Error(
      "No CoreRelayer contract interactions found for this transaction."
    );
  }

  if (index >= filtered.length) {
    throw Error("Specified delivery index is out of range.");
  } else {
    return {
      log: filtered[index].log,
      sequence: filtered[index].sequence,
    };
  }
}

export function vaaKeyToVaaKeyStruct(
  vaaKey: VaaKey
): IWormholeRelayer.VaaKeyStruct {
  return {
    infoType: vaaKey.payloadType,
    chainId: vaaKey.chainId || 0,
    emitterAddress:
      vaaKey.emitterAddress ||
      "0x0000000000000000000000000000000000000000000000000000000000000000",
    sequence: vaaKey.sequence || 0,
    vaaHash:
      vaaKey.vaaHash ||
      "0x0000000000000000000000000000000000000000000000000000000000000000",
  };
}
