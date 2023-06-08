import {
  ChainId,
  CHAIN_ID_TO_NAME,
  CHAINS,
  ChainName,
  Network,
  tryNativeToHexString,
  isChain,
  CONTRACTS
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
  DeliveryOverrideArgs,
  parseForwardFailureError
} from "../structs";
import { DeliveryProvider, DeliveryProvider__factory, Implementation__factory, IWormholeRelayerDelivery__factory } from "../../ethers-contracts/";
import {DeliveryEvent} from "../../ethers-contracts/WormholeRelayer"
import { VaaKeyStruct } from "../../ethers-contracts/IWormholeRelayer.sol/IWormholeRelayer";

export type DeliveryTargetInfo = {
  status: DeliveryStatus | string;
  transactionHash: string | null;
  vaaHash: string | null;
  sourceChain: ChainName;
  sourceVaaSequence: BigNumber | null;
  gasUsed: BigNumber;
  refundStatus: RefundStatus;
  revertString?: string; // Only defined if status is RECEIVER_FAILURE or FORWARD_REQUEST_FAILURE
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
  if(!(chainId in CHAIN_ID_TO_NAME)) throw Error(`Invalid Chain ID: ${chainId}`);
  return `${CHAIN_ID_TO_NAME[chainId as ChainId]} (Chain ${chainId})`;
}

export function getDefaultProvider(network: Network, chain: ChainName, ci?: boolean) {
  let rpc: string | undefined = "";
  if(ci) {
    if(chain == "ethereum") rpc = "http://eth-devnet:8545";
    else if(chain == "bsc") rpc = "http://eth-devnet2:8545";
    else throw Error(`This chain isn't in CI for relayers: ${chain}`)
  } else {
    rpc = RPCS_BY_CHAIN[network][chain];
  }
  if(!rpc) {
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

export function getBlockRange(
  provider: ethers.providers.Provider,
  timestamp?: number
): [ethers.providers.BlockTag, ethers.providers.BlockTag] {
  return [-2040, "latest"];
}

export async function getWormholeRelayerInfoBySourceSequence(
  environment: Network,
  targetChain: ChainName,
  targetChainProvider: ethers.providers.Provider,
  sourceChain: ChainName,
  sourceVaaSequence: BigNumber,
  blockStartNumber: ethers.providers.BlockTag,
  blockEndNumber: ethers.providers.BlockTag,
  targetWormholeRelayerAddress: string
): Promise<{chain: ChainName, events: DeliveryTargetInfo[]}> {
  const deliveryEvents = await getWormholeRelayerDeliveryEventsBySourceSequence(
    environment,
    targetChain,
    targetChainProvider,
    sourceChain,
    sourceVaaSequence,
    blockStartNumber,
    blockEndNumber,
    targetWormholeRelayerAddress
  );
  if (deliveryEvents.length == 0) {
    let status = `Delivery didn't happen on ${targetChain} within blocks ${blockStartNumber} to ${blockEndNumber}.`;
    try {
      const blockStart = await targetChainProvider.getBlock(blockStartNumber);
      const blockEnd = await targetChainProvider.getBlock(blockEndNumber);
      status = `Delivery didn't happen on ${targetChain} within blocks ${blockStart.number} to ${
        blockEnd.number
      } (within times ${new Date(
        blockStart.timestamp * 1000
      ).toString()} to ${new Date(blockEnd.timestamp * 1000).toString()})`;
    } catch (e) {}
    deliveryEvents.push({
      status,
      transactionHash: null,
      vaaHash: null,
      sourceChain: sourceChain,
      sourceVaaSequence,
      gasUsed: BigNumber.from(0),
      refundStatus: RefundStatus.RefundFail
    });
  }
  const targetChainStatus = {
    chain: targetChain,
    events: deliveryEvents
  };

  return targetChainStatus;
}

export async function getWormholeRelayerDeliveryEventsBySourceSequence(
  environment: Network,
  targetChain: ChainName,
  targetChainProvider: ethers.providers.Provider,
  sourceChain: ChainName,
  sourceVaaSequence: BigNumber,
  blockStartNumber: ethers.providers.BlockTag,
  blockEndNumber: ethers.providers.BlockTag,
  targetWormholeRelayerAddress: string
): Promise<DeliveryTargetInfo[]> {
  const sourceChainId = CHAINS[sourceChain];
  if(!sourceChainId) throw Error(`Invalid source chain: ${sourceChain}`)
  const wormholeRelayer = getWormholeRelayer(
    targetChain,
    environment,
    targetChainProvider,
    targetWormholeRelayerAddress
  );

  const deliveryEvents = wormholeRelayer.filters.Delivery(
    null,
    sourceChainId,
    sourceVaaSequence
  );

  const deliveryEventsPreFilter: DeliveryEvent[] = await wormholeRelayer.queryFilter(
    deliveryEvents,
    blockStartNumber,
    blockEndNumber
  );

  const isValid: boolean[] = await Promise.all(deliveryEventsPreFilter.map((deliveryEvent) => areSignaturesValid(deliveryEvent.getTransaction(), targetChain, targetChainProvider, environment)));

  // There is a max limit on RPCs sometimes for how many blocks to query
  return await transformDeliveryEvents(
    deliveryEventsPreFilter.filter((deliveryEvent, i) => isValid[i]),
    targetChainProvider
  );
}

async function areSignaturesValid(transaction: Promise<ethers.Transaction>, targetChain: ChainName, targetChainProvider: ethers.providers.Provider, environment: Network) {
  const coreAddress = CONTRACTS[environment][targetChain].core;
  if(!coreAddress) throw Error(`No Wormhole Address for chain ${targetChain}, network ${environment}`);

  const wormhole = Implementation__factory.connect(coreAddress, targetChainProvider);
  const decodedData = IWormholeRelayerDelivery__factory.createInterface().parseTransaction(await transaction);

  const vaaIsValid = async (vaa: ethers.utils.BytesLike): Promise<boolean> => {
    const [,result,reason] = await wormhole.parseAndVerifyVM(vaa);
    if(!result) console.log(`Invalid vaa! Reason: ${reason}`);
    return result;
  }

  const vaas = decodedData.args[0];
  for(let i=0; i<vaas.length; i++) {
    if(!(await vaaIsValid(vaas[i]))) {
      return false;
    }
  }

  return true;
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
      const status = deliveryStatus(x.args[4]);
      if(!isChain(x.args[1])) throw Error(`Invalid source chain id: ${x.args[1]}`);
      const sourceChain = CHAIN_ID_TO_NAME[x.args[1] as ChainId];
      return {
        status,
        transactionHash: x.transactionHash,
        vaaHash: x.args[3],
        sourceVaaSequence: x.args[2],
        sourceChain,
        gasUsed: BigNumber.from(x.args[5]),
        refundStatus: x.args[6],
        revertString: (status == DeliveryStatus.ReceiverFailure) ? x.args[7] : (status == DeliveryStatus.ForwardRequestFailure ? parseForwardFailureError(Buffer.from(x.args[7].substring(2), "hex")): undefined),
        overridesInfo: (Buffer.from(x.args[8].substring(2), "hex").length > 0) && parseOverrideInfoFromDeliveryEvent(Buffer.from(x.args[8].substring(2), "hex"))
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
      "No WormholeRelayer contract interactions found for this transaction."
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
): VaaKeyStruct {
  return {
    chainId: vaaKey.chainId || 0,
    emitterAddress:
      vaaKey.emitterAddress ||
      "0x0000000000000000000000000000000000000000000000000000000000000000",
    sequence: vaaKey.sequence || 0,
  };
}
