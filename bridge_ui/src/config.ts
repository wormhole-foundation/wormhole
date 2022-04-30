import { ChainId, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";

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
  [CHAIN_ID_SOLANA]: {
    disableTransfers: false,
    warningMessage: {
      text: "Solana has halted. Transactions may not be able to be completed.",
      link: {
        url: "https://twitter.com/SolanaStatus/status/1520508697100926977",
        text: "Follow @SolanaStatus for updates",
      },
    },
  } as ChainConfig,
};
