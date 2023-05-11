import * as wh from "@certusone/wormhole-sdk";
import { Next, ParsedVaaWithBytes, sleep } from "relayer-engine";
import {
  VaaKeyType,
  RelayerPayloadId,
  parseWormholeRelayerPayloadType,
  parseWormholeRelayerSend,
  deliveryInstructionsPrintable,
  vaaKeyPrintable,
  parseWormholeRelayerResend,
  RedeliveryInstruction,
  DeliveryInstruction,
  packOverrides,
  DeliveryOverrideArgs,
} from "@certusone/wormhole-sdk/lib/cjs/relayer";
import { EVMChainId } from "@certusone/wormhole-sdk";
import { GRContext } from "./app";
import { BigNumber, ethers } from "ethers";
import {
  CoreRelayer__factory,
  IDelivery,
} from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts";

export async function processProviderPriceUpdate(ctx: GRContext, next: Next) {
  next();
}
