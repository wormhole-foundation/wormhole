import { MapLevel } from "../../utils";
import { Network } from "../networks";
import { Chain } from "../chains";

export const tokenBridgeRelayerContracts = [
  [
    "Mainnet",
    [
      ["Ethereum", "0xcafd2f0a35a4459fa40c0517e17e6fa2939441ca"],
      ["Bsc", "0xcafd2f0a35a4459fa40c0517e17e6fa2939441ca"],
      ["Polygon", "0xcafd2f0a35a4459fa40c0517e17e6fa2939441ca"],
      ["Avalanche", "0xcafd2f0a35a4459fa40c0517e17e6fa2939441ca"],
      ["Fantom", "0xcafd2f0a35a4459fa40c0517e17e6fa2939441ca"],
      ["Celo", "0xcafd2f0a35a4459fa40c0517e17e6fa2939441ca"],
      ["Sui", "0x57f4e0ba41a7045e29d435bc66cc4175f381eb700e6ec16d4fdfe92e5a4dff9f"],
      ["Solana", "3vxKRPwUTiEkeUVyoZ9MXFe1V71sRLbLqu1gRYaWmehQ"],
      ["Base", "0xaE8dc4a7438801Ec4edC0B035EcCCcF3807F4CC1"],
      ["Moonbeam", "0xcafd2f0a35a4459fa40c0517e17e6fa2939441ca"],
    ],
  ],
  [
    "Testnet",
    [
      ["Ethereum", "0x9563a59c15842a6f322b10f69d1dd88b41f2e97b"],
      ["Bsc", "0x9563a59c15842a6f322b10f69d1dd88b41f2e97b"],
      ["Polygon", "0x9563a59c15842a6f322b10f69d1dd88b41f2e97b"],
      ["Avalanche", "0x9563a59c15842a6f322b10f69d1dd88b41f2e97b"],
      ["Fantom", "0x9563a59c15842a6f322b10f69d1dd88b41f2e97b"],
      ["Celo", "0x9563a59c15842a6f322b10f69d1dd88b41f2e97b"],
      ["Sui", "0xb30040e5120f8cb853b691cb6d45981ae884b1d68521a9dc7c3ae881c0031923"],
      ["Base", "0xae8dc4a7438801ec4edc0b035eccccf3807f4cc1"],
      ["Moonbeam", "0x9563a59c15842a6f322b10f69d1dd88b41f2e97b"],
      ["Solana", "3bPRWXqtSfUaCw3S4wdgvypQtsSzcmvDeaqSqPDkncrg"],
    ],
  ],
] as const satisfies MapLevel<Network, MapLevel<Chain, string>>;
