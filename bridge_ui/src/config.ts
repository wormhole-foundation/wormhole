import { ChainId } from "@certusone/wormhole-sdk";

export type DisableTransfers = boolean | "to" | "from";

export interface WarningMessage {
  text: string;
  link?: {
    url: string;
    text: string;
  };
}

export interface ChainConfig {
  disableTransfers?: DisableTransfers;
  warningMessage?: WarningMessage;
}

export type ChainConfigMap = {
  [key in ChainId]?: ChainConfig;
};

export const CHAIN_CONFIG_MAP: ChainConfigMap = {
  // [CHAIN_ID_POLYGON]: {
  //   disableTransfers: true,
  //   warningMessage: {
  //     text: "Polygon is currently experiencing partial downtime. As a precautionary measure, Wormhole Network and Portal have paused Polygon support until the network has been fully restored.",
  //     link: {
  //       url: "https://twitter.com/0xPolygonDevs",
  //       text: "Follow @0xPolygonDevs for updates",
  //     },
  //   },
  // } as ChainConfig,
};
