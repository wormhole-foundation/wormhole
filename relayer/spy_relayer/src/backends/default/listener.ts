/** The default backend is relaying payload 1 token bridge messages only */
import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  uint8ArrayToHex,
  tryHexToNativeString,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  parseTransferPayload,
} from "@certusone/wormhole-sdk";
import {
  getListenerEnvironment,
  ListenerEnvironment,
} from "../../configureEnv";
import { getScopedLogger, ScopedLogger } from "../../helpers/logHelper";
import {
  ParsedVaa,
  ParsedTransferPayload,
  parseVaaTyped,
} from "../../listener/validation";
import { TypedFilter, Listener } from "../definitions";
import {
  getKey,
  initPayloadWithVAA,
  storeInRedis,
  checkQueue,
  StoreKey,
  storeKeyFromParsedVAA,
  storeKeyToJson,
  StorePayload,
  storePayloadToJson,
} from "../../helpers/redisHelper";

async function encodeEmitterAddress(
  myChainId: ChainId,
  emitterAddressStr: string
): Promise<string> {
  if (myChainId === CHAIN_ID_SOLANA) {
    return await getEmitterAddressSolana(emitterAddressStr);
  }

  if (myChainId === CHAIN_ID_TERRA) {
    return await getEmitterAddressTerra(emitterAddressStr);
  }

  return getEmitterAddressEth(emitterAddressStr);
}

/** Listener for payload 1 token bridge messages only */
export class TokenBridgeListener implements Listener {
  logger: ScopedLogger;

  /**
   * @throws - when the listener environment setup fails
   */
  constructor() {
    this.logger = getScopedLogger(["TokenBridgeListener"]);
  }

  /** Verify this payload is version 1. */
  verifyIsPayloadV1(parsedVaa: ParsedVaa<Uint8Array>): boolean {
    const isCorrectPayloadType = parsedVaa.payload[0] === 1;

    if (!isCorrectPayloadType) {
      this.logger.debug("Specified vaa is not payload type 1.");
      return false;
    }
    return true;
  }

  /** Verify this payload has a fee specified for relaying. */
  verifyFeeSpecified(payload: ParsedTransferPayload): boolean {
    /**
     * TODO: simulate gas fees / get notional from coingecko and ensure the fees cover the relay.
     *       We might just keep this check here but verify the notional is enough to pay the gas
     *       fees in the actual relayer. That way we can retry up to the max number of retries
     *       and if the gas fluctuates we might be able to make it still.
     */

    /** Is the specified fee sufficient to relay? */
    const sufficientFee = payload.fee && payload.fee > BigInt(0);

    if (!sufficientFee) {
      this.logger.debug("Token transfer does not have a sufficient fee.");
      return false;
    }
    return true;
  }

  /** Verify the the token in this payload in the approved token list. */
  verifyIsApprovedToken(payload: ParsedTransferPayload): boolean {
    let originAddressNative: string;
    let env = getListenerEnvironment();
    try {
      originAddressNative = tryHexToNativeString(
        payload.originAddress,
        payload.originChain
      );
    } catch (e: any) {
      return false;
    }

    // Token is in the SUPPORTED_TOKENS env var config
    const isApprovedToken = env.supportedTokens.find((token) => {
      return (
        originAddressNative &&
        token.address.toLowerCase() === originAddressNative.toLowerCase() &&
        token.chainId === payload.originChain
      );
    });

    if (!isApprovedToken) {
      this.logger.debug("Token transfer is not for an approved token.");
      return false;
    }

    return true;
  }

  /** Parses a raw VAA byte array
   *
   * @throws when unable to parse the VAA
   */
  public async parseVaa(rawVaa: Uint8Array): Promise<ParsedVaa<Uint8Array>> {
    let parsedVaa: ParsedVaa<Uint8Array> | null = null;

    try {
      parsedVaa = await parseVaaTyped(rawVaa);
    } catch (e) {
      this.logger.error("Encountered error while parsing raw VAA " + e);
    }
    if (!parsedVaa) {
      throw new Error("Unable to parse the specified VAA.");
    }

    return parsedVaa;
  }

  /** Parse the VAA and return the payload nicely typed */
  public async parsePayload(
    rawPayload: Uint8Array
  ): Promise<ParsedTransferPayload> {
    let parsedPayload: any;
    try {
      parsedPayload = parseTransferPayload(Buffer.from(rawPayload));
    } catch (e) {
      this.logger.error("Encountered error while parsing vaa payload" + e);
    }

    if (!parsedPayload) {
      this.logger.debug("Failed to parse the transfer payload.");
      throw new Error("Could not parse the transfer payload.");
    }
    return parsedPayload;
  }

  /** Verify this is a VAA we want to relay. */
  public async validate(
    rawVaa: Uint8Array
  ): Promise<ParsedVaa<ParsedTransferPayload> | string> {
    let parsedVaa = await this.parseVaa(rawVaa);
    let parsedPayload: ParsedTransferPayload;

    // Verify this is actual a token bridge transfer payload
    if (!this.verifyIsPayloadV1(parsedVaa)) {
      return "Wrong payload type";
    }
    try {
      parsedPayload = await this.parsePayload(parsedVaa.payload);
    } catch (e: any) {
      return "Payload parsing failure";
    }

    // Verify we want to relay this request
    if (
      !this.verifyIsApprovedToken(parsedPayload) ||
      !this.verifyFeeSpecified(parsedPayload)
    ) {
      return "Validation failed";
    }

    // Great success!
    return { ...parsedVaa, payload: parsedPayload };
  }

  /** Get spy filters for all emitters we care about */
  public async getEmitterFilters(): Promise<TypedFilter[]> {
    let env = getListenerEnvironment();
    let filters: {
      emitterFilter: { chainId: ChainId; emitterAddress: string };
    }[] = [];
    for (let i = 0; i < env.spyServiceFilters.length; i++) {
      this.logger.info("Getting spyServiceFiltera " + i);
      const filter = env.spyServiceFilters[i];
      this.logger.info(
        "Getting spyServiceFilter[" +
          i +
          "]: chainId = " +
          filter.chainId +
          ", emmitterAddress = [" +
          filter.emitterAddress +
          "]"
      );
      const typedFilter = {
        emitterFilter: {
          chainId: filter.chainId as ChainId,
          emitterAddress: await encodeEmitterAddress(
            filter.chainId,
            filter.emitterAddress
          ),
        },
      };
      this.logger.info("Getting spyServiceFilterc " + i);
      this.logger.info(
        "adding filter: chainId: [" +
          typedFilter.emitterFilter.chainId +
          "], emitterAddress: [" +
          typedFilter.emitterFilter.emitterAddress +
          "]"
      );
      this.logger.info("Getting spyServiceFilterd " + i);
      filters.push(typedFilter);
      this.logger.info("Getting spyServiceFiltere " + i);
    }
    return filters;
  }

  /** Process and validate incoming VAAs from the spy. */
  public async process(rawVaa: Uint8Array): Promise<void> {
    // TODO: Use a type guard function to verify the ParsedVaa type too?
    const validationResults: ParsedVaa<ParsedTransferPayload> | string =
      await this.validate(rawVaa);

    if (typeof validationResults === "string") {
      this.logger.debug(`Skipping spied request: ${validationResults}`);
      return;
    }
    const parsedVaa: ParsedVaa<ParsedTransferPayload> = validationResults;

    const originChain = parsedVaa.payload.originChain;
    const originAddress = parsedVaa.payload.originAddress;

    let originAddressNative: string;
    try {
      originAddressNative = tryHexToNativeString(originAddress, originChain);
    } catch (e: any) {
      this.logger.error(
        `Failure to convert address "${originAddress}" on chain "${originChain}" to the native address`
      );
      return;
    }

    const key = getKey(parsedVaa.payload.originChain, originAddressNative);

    const isQueued = await checkQueue(key);
    if (isQueued) {
      this.logger.error(`Not storing in redis: ${isQueued}`);
      return;
    }

    this.logger.info(
      "forwarding vaa to relayer: emitter: [" +
        parsedVaa.emitterChain +
        ":" +
        uint8ArrayToHex(parsedVaa.emitterAddress) +
        "], seqNum: " +
        parsedVaa.sequence +
        ", payload: origin: [" +
        parsedVaa.payload.originAddress +
        ":" +
        parsedVaa.payload.originAddress +
        "], target: [" +
        parsedVaa.payload.targetChain +
        ":" +
        parsedVaa.payload.targetAddress +
        "],  amount: " +
        parsedVaa.payload.amount +
        "],  fee: " +
        parsedVaa.payload.fee +
        ", "
    );

    const redisKey: StoreKey = storeKeyFromParsedVAA(parsedVaa);
    const redisPayload: StorePayload = initPayloadWithVAA(
      uint8ArrayToHex(rawVaa)
    );

    await this.store(redisKey, redisPayload);
  }

  public async store(key: StoreKey, payload: StorePayload): Promise<void> {
    let serializedKey = storeKeyToJson(key);
    let serializedPayload = storePayloadToJson(payload);

    this.logger.debug(
      `storing: key: [${key.chain_id}/${key.emitter_address}/${key.sequence}], payload: [${serializedPayload}]`
    );

    return await storeInRedis(serializedKey, serializedPayload);
  }
}
