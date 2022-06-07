/** The default backend is relaying payload 1 token bridge messages only */
import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  hexToNativeString,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  parseTransferPayload,
} from "@certusone/wormhole-sdk";
import { ChainID } from "@certusone/wormhole-sdk/lib/cjs/proto/publicrpc/v1/publicrpc";

import { getListenerEnvironment, ListenerEnvironment } from "../../configureEnv";
import { getScopedLogger, ScopedLogger } from "../../helpers/logHelper";
import { ParsedVaa, parseVaaTyped } from "../../listener/validation";
import { TypedFilter, Listener, Relayer } from "../definitions";

// Copied from: spy_listen.ts
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
  env: ListenerEnvironment;
  parsedVaa?: ParsedVaa<Uint8Array>;
  parsedPayload?: any;

  constructor() {
    this.env = getListenerEnvironment();
    this.logger = getScopedLogger(["TokenBridgeListener"]);
  }

  /** Verify this payload is version 1. */
  verifyIsPayloadV1(): boolean {
    const isCorrectPayloadType =
      this.parsedVaa && this.parsedVaa.payload[0] === 1;

    if (!isCorrectPayloadType) {
      this.logger.debug("Specified vaa is not payload type 1.");
      return false;
    }
    return true;
  }

  /** Verify this payload has a fee specified for relaying. */
  verifyFeeSpecified(): boolean {
    /**
     * TODO: simulate gas fees / get notional from coingecko and ensure the fees cover the relay.
     *       We might just keep this check here but verify the notional is enough to pay the gas
     *       fees in the actual relayer. That way we can retry up to the max number of retries
     *       and if the gas fluctuates we might be able to make it still.
     */
    const sufficientFee =
      this.parsedPayload &&
      this.parsedPayload.fee &&
      this.parsedPayload.fee > 0;

    if (!sufficientFee) {
      this.logger.debug("Token transfer does not have a sufficient fee.");
      return false;
    }
    return true;
  }

  /** Verify the the token in this payload in the approved token list. */
  verifyIsApprovedToken(): boolean {
    const env = getListenerEnvironment();
    const originAddressNative = hexToNativeString(
      this.parsedPayload.originAddress,
      this.parsedPayload.originChain
    );

    const isApprovedToken = env.supportedTokens.find((token) => {
      return (
        originAddressNative &&
        token.address.toLowerCase() === originAddressNative.toLowerCase() &&
        token.chainId === this.parsedPayload.originChain
      );
    });

    if (!isApprovedToken) {
      this.logger.debug("Token transfer is not for an approved token.");
      return false;
    }

    return true;
  }

  /** Verify this is a VAA we want to relay. */
  public async shouldRelay(rawVaa: Uint8Array): Promise<boolean> {
    let parsedVaa: ParsedVaa<Uint8Array> | null = null;
    let parsedPayload: any = null;

    try {
      parsedVaa = await parseVaaTyped(rawVaa);
    } catch (e) {
      this.logger.error("Encountered error while parsing raw VAA " + e);
    }
    if (!parsedVaa) {
      return false;
    }
    this.parsedVaa = parsedVaa;

    parsedPayload = parseTransferPayload(Buffer.from(parsedVaa.payload));
    if (!parsedPayload) {
      this.logger.error("Failed to parse the transfer payload");
      return false;
    }

    // Verify this VAA should be relayed.
    if (!this.verifyIsPayloadV1()) {
      return false;
    } else if (!this.verifyIsApprovedToken()) {
      return false;
    } else if (!this.verifyFeeSpecified()) {
      return false;
    }
    // Great success!
    return true;
  }

  /** Get spy filters for all emitters we care about */
  async getEmitterFilters(): Promise<TypedFilter[]> {
    let filters: {
      emitterFilter: { chainId: ChainId; emitterAddress: string };
    }[] = [];
    for (let i = 0; i < this.env.spyServiceFilters.length; i++) {
      this.logger.info("Getting spyServiceFiltera " + i);
      const filter = this.env.spyServiceFilters[i];
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
}