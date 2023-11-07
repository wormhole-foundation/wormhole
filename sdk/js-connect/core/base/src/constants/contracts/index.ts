import * as core from "./core";
import * as tb from "./tokenBridge";
import * as tbr from "./tokenBridgeRelayer";
import * as nb from "./nftBridge";
import * as r from "./relayer";
import * as circle from "./circle";
import * as g from "./cosmos";

import { constMap } from "../../utils";

export const coreBridge = constMap(core.coreBridgeContracts);
export const tokenBridge = constMap(tb.tokenBridgeContracts);
export const tokenBridgeRelayer = constMap(tbr.tokenBridgeRelayerContracts);
export const nftBridge = constMap(nb.nftBridgeContracts);
export const relayer = constMap(r.relayerContracts);
export const gateway = constMap(g.gatewayContracts);
export const translator = constMap(g.translatorContracts);

export { CircleContracts } from "./circle";
export const circleContracts = constMap(circle.circleContracts);
