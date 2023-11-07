import { RoArray } from "../../utils";
import { ChainName } from "../chains";
import { Network } from "../networks";

export const gatewayContracts = [
  [
    "Mainnet",
    [["Wormchain", "wormhole14ejqjyq8um4p3xfqj74yld5waqljf88fz25yxnma0cngspxe3les00fpjx"]],
  ],
  [
    "Testnet",
    [["Wormchain", "wormhole1ctnjk7an90lz5wjfvr3cf6x984a8cjnv8dpmztmlpcq4xteaa2xs9pwmzk"]],
  ],
] as const satisfies RoArray<readonly [Network, RoArray<readonly [ChainName, string]>]>;

export const translatorContracts = [
  [
    "Mainnet",
    [
      ["Sei", ""], // TODO
    ],
  ],
  ["Testnet", [["Sei", "sei1dkdwdvknx0qav5cp5kw68mkn3r99m3svkyjfvkztwh97dv2lm0ksj6xrak"]]],
] as const satisfies RoArray<readonly [Network, RoArray<readonly [ChainName, string]>]>;
