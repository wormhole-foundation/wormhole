import {
  ChainId,
  hexToNativeString,
  parseTransferPayload,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import { importCoreWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { getListenerEnvironment } from "../configureEnv";
import { getLogger } from "../helpers/logHelper";
import {
  connectToRedis,
  getBackupQueue,
  getKey,
  RedisTables,
} from "../helpers/redisHelper";

const logger = getLogger();

export function validateInit(): boolean {
  const env = getListenerEnvironment();

  logger.info(
    "supported target chains: [" + env.spyServiceFilters.toString() + "]"
  );
  if (env.spyServiceFilters.length) {
    env.spyServiceFilters.forEach((allowedContract) => {
      logger.info(
        "adding allowed contract: chainId: [" +
          allowedContract.chainId +
          "] => address: [" +
          allowedContract.emitterAddress +
          "]"
      );
    });
  } else {
    logger.info("There are no white listed contracts provisioned.");
  }

  logger.info("supported tokens : [" + env.supportedTokens.toString() + "]");
  if (env.supportedTokens.length) {
    env.supportedTokens.forEach((supportedToken) => {
      logger.info(
        "adding allowed contract: chainId: [" +
          supportedToken.chainId +
          "] => address: [" +
          supportedToken.address +
          "]" +
          " key: " +
          getKey(supportedToken.chainId, supportedToken.address)
      );
    });
  } else {
    logger.info("There are no white listed contracts provisioned.");
  }

  return true;
}

export async function parseAndValidateVaa(
  rawVaa: Uint8Array
): Promise<string | ParsedVaa<ParsedTransferPayload>> {
  logger.debug("About to validate: " + uint8ArrayToHex(rawVaa));
  let parsedVaa: ParsedVaa<Uint8Array> | null = null;
  try {
    parsedVaa = await parseVaaTyped(rawVaa);
  } catch (e) {
    logger.error("Encountered error while parsing raw VAA " + e);
  }
  if (!parsedVaa) {
    return "Unable to parse the specified VAA.";
  }
  const env = getListenerEnvironment();

  //You have to derive all the emitter addresses from the native addresses, because emitter addresses cannot be mapped backwards to native.
  //This is especially important because they are only uninvertible on Solana, and if you convert the emitter addresses to native,
  //It will work for all chains except Solana.

  //TODO calc emitter addresses, and compare against those, rather than getting the natives from the emitter

  // const nativeAddress = hexToNativeString(
  //   uint8ArrayToHex(parsedVaa.emitterAddress),
  //   parsedVaa.emitterChain
  // );

  // logger.info("nativeAddress format for emitter address in validator:" + nativeAddress);

  // const isApprovedAddress = env.spyServiceFilters.find((allowedContract) => {
  //   console.log(
  //     parsedVaa,
  //     nativeAddress,
  //     allowedContract.emitterAddress,
  //     "in approved address"
  //   );
  //   return (
  //     parsedVaa &&
  //     nativeAddress &&
  //     allowedContract.chainId === parsedVaa.emitterChain &&
  //     allowedContract.emitterAddress.toLowerCase() ===
  //       nativeAddress.toLowerCase()
  //   );
  // });

  // if (!isApprovedAddress) {
  //   logger.debug("Specified vaa is not from an approved address.");
  //   return "VAA is not from a monitored contract.";
  // }

  const isCorrectPayloadType = parsedVaa.payload[0] === 1;

  if (!isCorrectPayloadType) {
    logger.debug("Specified vaa is not payload type 1.");
    return "Specified vaa is not payload type 1..";
  }

  let parsedPayload: any = null;
  try {
    parsedPayload = parseTransferPayload(Buffer.from(parsedVaa.payload));
  } catch (e) {
    logger.error("Encountered error while parsing vaa payload" + e);
  }

  if (!parsedPayload) {
    logger.debug("Failed to parse the transfer payload.");
    return "Could not parse the transfer payload.";
  }

  const originAddressNative = hexToNativeString(
    parsedPayload.originAddress,
    parsedPayload.originChain
  );

  const isApprovedToken = env.supportedTokens.find((token) => {
    return (
      originAddressNative &&
      token.address.toLowerCase() === originAddressNative.toLowerCase() &&
      token.chainId === parsedPayload.originChain
    );
  });

  if (!isApprovedToken) {
    logger.debug("Token transfer is not for an approved token.");
    return "Token transfer is not for an approved token.";
  }

  //TODO configurable
  const sufficientFee = parsedPayload.fee && parsedPayload.fee > 0;

  if (!sufficientFee) {
    logger.debug("Token transfer does not have a sufficient fee.");
    return "Token transfer does not have a sufficient fee.";
  }

  const key = getKey(parsedPayload.originChain, originAddressNative as string); //was null checked above

  const isQueued = await checkQueue(key);
  if (isQueued) {
    return isQueued;
  }
  //TODO maybe an is redeemed check?

  const fullyTyped = { ...parsedVaa, payload: parsedPayload };
  return fullyTyped;
}

async function checkQueue(key: string): Promise<string | null> {
  try {
    const backupQueue = getBackupQueue();
    const queuedRecord = backupQueue.find((record) => {
      record[0] === key;
    });

    if (queuedRecord) {
      logger.debug("VAA was already in the listener queue");
      return "VAA was already in the listener queue";
    }

    const rClient = await connectToRedis();
    if (!rClient) {
      logger.error("Failed to connect to redis");
      return null;
    }
    await rClient.select(RedisTables.INCOMING);
    const record1 = await rClient.get(key);

    if (record1) {
      logger.debug("VAA was already in INCOMING table");
      rClient.quit();
      return "VAA was already in INCOMING table";
    }

    await rClient.select(RedisTables.WORKING);
    const record2 = await rClient.get(key);
    if (record2) {
      logger.debug("VAA was already in WORKING table");
      rClient.quit();
      return "VAA was already in WORKING table";
    }
    rClient.quit();
  } catch (e) {
    logger.error("Failed to connect to redis");
  }

  return null;
}

//TODO move these to the official SDK
export async function parseVaaTyped(signedVAA: Uint8Array) {
  const { parse_vaa } = await importCoreWasm();
  const parsedVAA = parse_vaa(signedVAA);
  return {
    timestamp: parseInt(parsedVAA.timestamp),
    nonce: parseInt(parsedVAA.nonce),
    emitterChain: parseInt(parsedVAA.emitter_chain) as ChainId,
    emitterAddress: parsedVAA.emitter_address, //This will be in wormhole HEX format
    sequence: parseInt(parsedVAA.sequence),
    consistencyLevel: parseInt(parsedVAA.consistency_level),
    payload: parsedVAA.payload,
  };
}

export type ParsedVaa<T> = {
  timestamp: number;
  nonce: number;
  emitterChain: ChainId;
  emitterAddress: Uint8Array;
  sequence: number;
  consistencyLevel: number;
  payload: T;
};

export type ParsedTransferPayload = {
  amount: BigInt;
  originAddress: Uint8Array; //hex
  originChain: ChainId;
  targetAddress: Uint8Array; //hex
  targetChain: ChainId;
  fee?: BigInt;
};
