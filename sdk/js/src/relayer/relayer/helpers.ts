import {
  ChainId,
  CHAIN_ID_TO_NAME,
  CHAINS,
  ChainName,
  Network,
  tryNativeToHexString,
  isChain,
  CONTRACTS,
} from "../../";
import { BigNumber, ContractReceipt, ethers } from "ethers";
import {
  getWormholeRelayer,
  RPCS_BY_CHAIN,
  RELAYER_CONTRACTS,
  getWormholeRelayerAddress,
  getCircleAPI,
  getWormscanAPI,
  getNameFromCCTPDomain,
} from "../consts";
import {
  parseWormholeRelayerPayloadType,
  parseOverrideInfoFromDeliveryEvent,
  RelayerPayloadId,
  parseWormholeRelayerSend,
  DeliveryInstruction,
  DeliveryStatus,
  RefundStatus,
  VaaKey,
  DeliveryOverrideArgs,
  parseRefundStatus,
  RedeliveryInstruction,
  parseWormholeRelayerResend,
  CCTPKey,
} from "../structs";
import { InfoRequestParams } from "./info";
import { Implementation__factory } from "../../ethers-contracts/";
import {
  DeliveryProvider,
  DeliveryProvider__factory,
} from "../../ethers-relayer-contracts/";
import { DeliveryEvent } from "../../ethers-relayer-contracts/WormholeRelayer";
import { VaaKeyStruct } from "../../ethers-relayer-contracts/IWormholeRelayer.sol/IWormholeRelayer";

export type DeliveryTargetInfo = {
  status: DeliveryStatus | string;
  transactionHash: string | null;
  vaaHash: string | null;
  sourceChain: ChainName | null;
  sourceVaaSequence: BigNumber | null;
  gasUsed: BigNumber;
  refundStatus: RefundStatus;
  timestamp?: number;
  revertString?: string; // Only defined if status is RECEIVER_FAILURE
  overrides?: DeliveryOverrideArgs;
};

export function parseWormholeLog(log: ethers.providers.Log): {
  type: RelayerPayloadId;
  parsed: DeliveryInstruction | RedeliveryInstruction | string;
} {
  const abi = [
    "event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel)",
  ];
  const iface = new ethers.utils.Interface(abi);
  const parsed = iface.parseLog(log);
  const payload = Buffer.from(parsed.args.payload.substring(2), "hex");
  const type = parseWormholeRelayerPayloadType(payload);
  if (type == RelayerPayloadId.Delivery) {
    return { type, parsed: parseWormholeRelayerSend(payload) };
  } else if (type == RelayerPayloadId.Redelivery) {
    return { type, parsed: parseWormholeRelayerResend(payload) };
  } else {
    throw Error("Invalid wormhole log");
  }
}

export function printChain(chainId: number) {
  if (!(chainId in CHAIN_ID_TO_NAME))
    throw Error(`Invalid Chain ID: ${chainId}`);
  return `${CHAIN_ID_TO_NAME[chainId as ChainId]} (Chain ${chainId})`;
}

export function printCCTPDomain(domain: number) {
  if (getNameFromCCTPDomain(domain) === undefined)
    throw Error(`Invalid cctp domain: ${domain}`);
  return `${getNameFromCCTPDomain(domain)} (Domain ${domain})`;
}

export const estimatedAttestationTimeInSeconds = (
  sourceChain: string,
  environment: Network
): number => {
  const testnetTime = sourceChain === "avalanche" ? 20 : 60;
  const mainnetTime = sourceChain === "avalanche" ? 20 : 60 * 13;
  return environment === "TESTNET" ? testnetTime : mainnetTime;
};

export function getDefaultProvider(
  network: Network,
  chain: ChainName,
  ci?: boolean
) {
  let rpc: string | undefined = "";
  if (ci) {
    if (chain == "ethereum") rpc = "http://eth-devnet:8545";
    else if (chain == "bsc") rpc = "http://eth-devnet2:8545";
    else throw Error(`This chain isn't in CI for relayers: ${chain}`);
  } else {
    rpc = RPCS_BY_CHAIN[network][chain];
  }
  if (!rpc) {
    throw Error(`No default RPC for chain ${chain} or network ${network}`);
  }
  return new ethers.providers.StaticJsonRpcProvider(rpc);
}

export function getDeliveryProvider(
  address: string,
  provider: ethers.providers.Provider
): DeliveryProvider {
  const contract = DeliveryProvider__factory.connect(address, provider);
  return contract;
}

export async function getWormholeRelayerInfoBySourceSequence(
  environment: Network,
  targetChain: ChainName,
  targetChainProvider: ethers.providers.Provider,
  sourceChain: ChainName | undefined,
  sourceVaaSequence: BigNumber | undefined,
  blockRange:
    | [ethers.providers.BlockTag, ethers.providers.BlockTag]
    | undefined,
  targetWormholeRelayerAddress: string
): Promise<DeliveryTargetInfo[]> {
  const deliveryEvents = await getWormholeRelayerDeliveryEventsBySourceSequence(
    environment,
    targetChain,
    targetChainProvider,
    sourceChain,
    sourceVaaSequence,
    blockRange,
    targetWormholeRelayerAddress
  );

  return deliveryEvents;
}

export async function getWormholeRelayerDeliveryEventsBySourceSequence(
  environment: Network,
  targetChain: ChainName,
  targetChainProvider: ethers.providers.Provider,
  sourceChain: ChainName | undefined,
  sourceVaaSequence: BigNumber | undefined,
  blockRange:
    | [ethers.providers.BlockTag, ethers.providers.BlockTag]
    | undefined,
  targetWormholeRelayerAddress: string
): Promise<DeliveryTargetInfo[]> {
  let sourceChainId = undefined;
  if (sourceChain) {
    sourceChainId = CHAINS[sourceChain];
    if (!sourceChainId) throw Error(`Invalid source chain: ${sourceChain}`);
  }

  const wormholeRelayer = getWormholeRelayer(
    targetChain,
    environment,
    targetChainProvider,
    targetWormholeRelayerAddress
  );

  const deliveryEventsFilter = wormholeRelayer.filters.Delivery(
    null,
    sourceChainId,
    sourceVaaSequence
  );

  const deliveryEvents: DeliveryEvent[] = await wormholeRelayer.queryFilter(
    deliveryEventsFilter,
    blockRange ? blockRange[0] : -2000,
    blockRange ? blockRange[1] : "latest"
  );

  const timestamps = await Promise.all(
    deliveryEvents.map(
      async (e) =>
        (await targetChainProvider.getBlock(e.blockNumber)).timestamp * 1000
    )
  );

  // There is a max limit on RPCs sometimes for how many blocks to query
  return await transformDeliveryEvents(deliveryEvents, timestamps);
}

export function deliveryStatus(status: number) {
  switch (status) {
    case 0:
      return DeliveryStatus.DeliverySuccess;
    case 1:
      return DeliveryStatus.ReceiverFailure;
    default:
      return DeliveryStatus.ThisShouldNeverHappen;
  }
}

export function transformDeliveryLog(
  log: {
    args: [
      string,
      number,
      BigNumber,
      string,
      number,
      BigNumber,
      number,
      string,
      string
    ];
    transactionHash: string;
  },
  timestamp: number
): DeliveryTargetInfo {
  const status = deliveryStatus(log.args[4]);
  if (!isChain(log.args[1]))
    throw Error(`Invalid source chain id: ${log.args[1]}`);
  const sourceChain = CHAIN_ID_TO_NAME[log.args[1] as ChainId];
  return {
    status,
    transactionHash: log.transactionHash,
    vaaHash: log.args[3],
    sourceVaaSequence: log.args[2],
    sourceChain,
    gasUsed: BigNumber.from(log.args[5]),
    refundStatus: parseRefundStatus(log.args[6]),
    revertString:
      status == DeliveryStatus.ReceiverFailure ? log.args[7] : undefined,
    timestamp,
    overrides:
      Buffer.from(log.args[8].substring(2), "hex").length > 0
        ? parseOverrideInfoFromDeliveryEvent(
            Buffer.from(log.args[8].substring(2), "hex")
          )
        : undefined,
  };
}

async function transformDeliveryEvents(
  events: DeliveryEvent[],
  timestamps: number[]
): Promise<DeliveryTargetInfo[]> {
  return events.map((x, i) => transformDeliveryLog(x, timestamps[i]));
}

export function getWormholeLog(
  receipt: ContractReceipt,
  bridgeAddress: string,
  emitterAddress: string,
  index: number,
  sequence?: number
): { log: ethers.providers.Log; sequence: string; payload: string } {
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
      payload: log.args[3],
      log: bridgeLog,
    };
  });

  const filtered = parsed.filter((x) => {
    return (
      x.emitterAddress == emitterAddress.toLowerCase() &&
      (sequence === undefined ? true : x.sequence + "" === sequence + "")
    );
  });

  if (filtered.length == 0) {
    throw Error(
      `No wormhole contract interactions found for this transaction, with emitter address ${emitterAddress} ${
        sequence === undefined ? "" : `and sequence ${sequence}`
      }`
    );
  }

  if (index >= filtered.length) {
    throw Error("Specified delivery index is out of range.");
  } else {
    return {
      log: filtered[index].log,
      sequence: filtered[index].sequence,
      payload: filtered[index].payload,
    };
  }
}

export function vaaKeyToVaaKeyStruct(vaaKey: VaaKey): VaaKeyStruct {
  return {
    chainId: vaaKey.chainId || 0,
    emitterAddress:
      vaaKey.emitterAddress ||
      "0x0000000000000000000000000000000000000000000000000000000000000000",
    sequence: vaaKey.sequence || 0,
  };
}

export async function getWormholeRelayerInfoByHash(
  deliveryHash: string,
  targetChain: ChainName,
  sourceChain: ChainName | undefined,
  sourceVaaSequence: number | undefined,
  infoRequest?: InfoRequestParams
): Promise<DeliveryTargetInfo[]> {
  const environment = infoRequest?.environment || "MAINNET";
  const targetChainProvider =
    infoRequest?.targetChainProviders?.get(targetChain) ||
    getDefaultProvider(environment, targetChain);

  if (!targetChainProvider) {
    throw Error(
      "No default RPC for this chain; pass in your own provider (as targetChainProvider)"
    );
  }
  const targetWormholeRelayerAddress =
    infoRequest?.wormholeRelayerAddresses?.get(targetChain) ||
    getWormholeRelayerAddress(targetChain, environment);
  const wormholeRelayer = getWormholeRelayer(
    targetChain,
    environment,
    targetChainProvider,
    targetWormholeRelayerAddress
  );

  const blockNumberSuccess = await wormholeRelayer.deliverySuccessBlock(
    deliveryHash
  );
  const blockNumberFailure = await wormholeRelayer.deliveryFailureBlock(
    deliveryHash
  );
  const blockNumber = blockNumberSuccess.gt(0)
    ? blockNumberSuccess
    : blockNumberFailure;

  if (blockNumber.toNumber() === 0) return [];

  // There is weirdness with arbitrum where if you call 'block.number', it gives you the L1 block number (the ethereum one) - and this is what is stored in the 'replay protection mapping' - so basically that value isn't useful in finding the delivery here
  const blockRange =
    infoRequest?.targetBlockRange ||
    (targetChain === "arbitrum"
      ? undefined
      : [blockNumber.toNumber(), blockNumber.toNumber()]);

  return await getWormholeRelayerInfoBySourceSequence(
    environment,
    targetChain,
    targetChainProvider,
    sourceChain,
    BigNumber.from(sourceVaaSequence),
    blockRange,
    targetWormholeRelayerAddress
  );
}

export function getDeliveryHashFromVaaFields(
  sourceChain: number,
  emitterAddress: string,
  sequence: number,
  timestamp: number,
  nonce: number,
  consistencyLevel: number,
  deliveryVaaPayload: string
): string {
  const body = ethers.utils.solidityPack(
    ["uint32", "uint32", "uint16", "bytes32", "uint64", "uint8", "bytes"],

    [
      timestamp,
      nonce,
      sourceChain,
      emitterAddress,
      sequence,
      consistencyLevel,
      deliveryVaaPayload,
    ]
  );
  const deliveryHash = ethers.utils.keccak256(ethers.utils.keccak256(body));
  return deliveryHash;
}

export async function getWormscanInfo(
  network: Network,
  sourceChain: ChainName,
  sequence: number,
  emitterAddress: string
) {
  const wormscanAPI = getWormscanAPI(network);
  const emitterAddressBytes32 = tryNativeToHexString(
    emitterAddress,
    sourceChain
  );
  const sourceChainId = CHAINS[sourceChain];
  const result = await fetch(
    `${wormscanAPI}api/v1/vaas/${sourceChainId}/${emitterAddressBytes32}/${sequence}`
  );
  return result;
}

export async function getWormscanRelayerInfo(
  sourceChain: ChainName,
  sequence: number,
  optionalParams?: {
    network?: Network;
    provider?: ethers.providers.Provider;
    wormholeRelayerAddress?: string;
  }
): Promise<Response> {
  const network = optionalParams?.network || "MAINNET";
  const wormholeRelayerAddress =
    optionalParams?.wormholeRelayerAddress ||
    getWormholeRelayerAddress(sourceChain, network);
  return getWormscanInfo(
    network,
    sourceChain,
    sequence,
    wormholeRelayerAddress
  );
}

export async function getRelayerTransactionHashFromWormscan(
  sourceChain: ChainName,
  sequence: number,
  optionalParams?: {
    network?: Network;
    provider?: ethers.providers.Provider;
    wormholeRelayerAddress?: string;
  }
): Promise<string> {
  const wormscanData = (
    await (
      await getWormscanRelayerInfo(sourceChain, sequence, optionalParams)
    ).json()
  ).data;
  return "0x" + wormscanData.txHash;
}

export async function getDeliveryHash(
  rx: ethers.ContractReceipt,
  sourceChain: ChainName,
  optionalParams?: {
    network?: Network;
    provider?: ethers.providers.Provider;
    index?: number;
    wormholeRelayerAddress?: string;
  }
): Promise<string> {
  const network: Network = optionalParams?.network || "MAINNET";
  const provider: ethers.providers.Provider =
    optionalParams?.provider || getDefaultProvider(network, sourceChain);
  const wormholeAddress = CONTRACTS[network][sourceChain].core;
  if (!wormholeAddress) {
    throw Error(`No wormhole contract on ${sourceChain} for ${network}`);
  }
  const wormholeRelayerAddress =
    optionalParams?.wormholeRelayerAddress ||
    RELAYER_CONTRACTS[network][sourceChain]?.wormholeRelayerAddress;
  if (!wormholeRelayerAddress) {
    throw Error(
      `No wormhole relayer contract on ${sourceChain} for ${network}`
    );
  }
  const logs = rx.logs.filter(
    (log) =>
      log.address.toLowerCase() === wormholeAddress.toLowerCase() &&
      log.topics[1].toLowerCase() ===
        "0x" +
          tryNativeToHexString(wormholeRelayerAddress, "ethereum").toLowerCase()
  );
  const index = optionalParams?.index || 0;
  if (logs.length === 0)
    throw Error(
      `No wormhole relayer log found${
        index > 0 ? ` (the ${index}-th wormhole relayer log was requested)` : ""
      }`
    );
  return getDeliveryHashFromLog(
    logs[index],
    CHAINS[sourceChain],
    provider,
    rx.blockHash
  );
}

export async function getDeliveryHashFromLog(
  wormholeLog: ethers.providers.Log,
  sourceChain: ChainId,
  provider: ethers.providers.Provider,
  blockHash: string
): Promise<string> {
  const wormholePublishedMessage =
    Implementation__factory.createInterface().parseLog(wormholeLog);

  const block = await provider.getBlock(blockHash);

  return getDeliveryHashFromVaaFields(
    sourceChain,
    wormholeLog.topics[1],
    wormholePublishedMessage.args["sequence"],
    block.timestamp,
    wormholePublishedMessage.args["nonce"],
    wormholePublishedMessage.args["consistencyLevel"],
    wormholePublishedMessage.args["payload"]
  );
}

export async function getCCTPMessageLogURL(
  cctpKey: CCTPKey,
  sourceChain: ChainName,
  receipt: ethers.providers.TransactionReceipt,
  environment: Network
) {
  let cctpLog;
  let messageSentLog;
  const DepositForBurnTopic = ethers.utils.keccak256(
    ethers.utils.toUtf8Bytes(
      "DepositForBurn(uint64,address,uint256,address,bytes32,uint32,bytes32,bytes32)"
    )
  );
  const MessageSentTopic = ethers.utils.keccak256(
    ethers.utils.toUtf8Bytes("MessageSent(bytes)")
  );
  try {
    if (getNameFromCCTPDomain(cctpKey.domain, environment) === sourceChain) {
      const cctpLogFilter = (log: ethers.providers.Log) => {
        return (
          log.topics[0] === DepositForBurnTopic &&
          parseInt(log.topics[1]) === cctpKey.nonce.toNumber()
        );
      };
      cctpLog = receipt.logs.find(cctpLogFilter);
      const index = receipt.logs.findIndex(cctpLogFilter);
      const messageSentLogs = receipt.logs.filter((log, i) => {
        return log.topics[0] === MessageSentTopic && i <= index;
      });
      messageSentLog = messageSentLogs[messageSentLogs.length - 1];
    }
  } catch (e) {
    console.log(e);
  }
  if (!cctpLog || !messageSentLog) return undefined;

  const message = new ethers.utils.Interface([
    "event MessageSent(bytes message)",
  ]).parseLog(messageSentLog).args.message;
  const msgHash = ethers.utils.keccak256(message);
  const url = getCircleAPI(environment) + msgHash;
  return { message, cctpLog, url };
}
