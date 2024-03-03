const ci = !!process.env.CI;

// see devnet.md
export const ETH_NODE_URL = ci
  ? "http://eth-devnet:8545"
  : "http://localhost:8545";
export const ETH_PRIVATE_KEY =
  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"; // account 0
// account 1 used by NFT tests
export const ETH_PRIVATE_KEY2 =
  "0x6370fd033278c143179d81c5526140625662b8daa446c22ee2d73db3707e620c"; // account 2 - terra2 tests
export const ETH_PRIVATE_KEY3 =
  "0x646f1ce2fdad0e6deeeb5c7e8e5543bdde65e86029e2fd9fc169899c440a7913"; // account 3 - solana tests
export const ETH_PRIVATE_KEY4 =
  "0xadd53f9a7e588d003326d1cbf9e4a43c061aadd9bc938c843a79e7b4fd2ad743"; // account 4 - terrac tests
export const ETH_PRIVATE_KEY5 =
  "0x395df67f0c2d2d9fe1ad08d1bc8b6627011959b79c53d7dd6a3536a33ab8a4fd"; // account 5 - near tests
export const ETH_PRIVATE_KEY6 =
  "0xe485d098507f54e7733a205420dfddbe58db035fa577fc294ebd14db90767a52"; // account 6 - aptos tests
export const ETH_PRIVATE_KEY7 =
  "0xa453611d9419d0e56f499079478fd72c37b251a94bfde4d19872c44cf65386e3"; // account 7 - algorand tests
export const ETH_PRIVATE_KEY9 =
  "0xb0057716d5917badaf911b193b12b910811c1497b5bada8d7711f758981c3773"; // account 9 - accountant tests
export const ETH_PRIVATE_KEY10 =
  "0x77c5495fbb039eed474fc940f29955ed0531693cc9212911efd35dff0373153f"; // account 10 - sui tests
export const ETH_PRIVATE_KEY11 =
  "0xd99b5b29e6da2528bf458b26237a6cf8655a3e3276c1cdc0de1f98cefee81c01"; // account 11 - ntt-accountant tests
export const ETH_PRIVATE_KEY12 =
  "0x9b9c613a36396172eab2d34d72331c8ca83a358781883a535d2941f66db07b24"; // account 12 - ntt-accountant tests
export const SOLANA_HOST = ci
  ? "http://solana-devnet:8899"
  : "http://localhost:8899";
export const SOLANA_PRIVATE_KEY = new Uint8Array([
  14, 173, 153, 4, 176, 224, 201, 111, 32, 237, 183, 185, 159, 247, 22, 161, 89,
  84, 215, 209, 212, 137, 10, 92, 157, 49, 29, 192, 101, 164, 152, 70, 87, 65,
  8, 174, 214, 157, 175, 126, 98, 90, 54, 24, 100, 177, 247, 77, 19, 112, 47,
  44, 165, 109, 233, 102, 14, 86, 109, 29, 134, 145, 132, 141,
]);
export const TERRA_NODE_URL = ci
  ? "http://terra-terrad:1317"
  : "http://localhost:1317";
export const TERRA2_NODE_URL = ci
  ? "http://terra2-terrad:1317"
  : "http://localhost:1318";
export const TERRA_CHAIN_ID = "localterra";
// NOTE: test1 is used by getIsTransferCompletedTerra, so avoid using it in the integration tests
// Accounts from https://github.com/terra-money/LocalTerra/blob/main/README.md#accounts
export const TERRA_PUBLIC_KEY = "terra17tv2hvwpg0ukqgd2y5ct2w54fyan7z0zxrm2f9"; // test7
export const TERRA_PRIVATE_KEY =
  "noble width taxi input there patrol clown public spell aunt wish punch moment will misery eight excess arena pen turtle minimum grain vague inmate"; // test7
export const TERRA_PRIVATE_KEY2 =
  "quality vacuum heart guard buzz spike sight swarm shove special gym robust assume sudden deposit grid alcohol choice devote leader tilt noodle tide penalty"; // test2
export const TERRA2_PRIVATE_KEY =
  "symbol force gallery make bulk round subway violin worry mixture penalty kingdom boring survey tool fringe patrol sausage hard admit remember broken alien absorb"; // test3
export const TEST_ERC20 = "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A";
export const TEST_SOLANA_TOKEN = "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ";
export const WORMHOLE_RPC_HOSTS = ci
  ? ["http://guardian:7071"]
  : ["http://localhost:7071"];

export type Environment = "devnet" | "testnet" | "mainnet";
export const CLUSTER: Environment = "devnet" as Environment; //This is the currently selected environment.

export const TERRA_HOST =
  CLUSTER === "mainnet"
    ? {
        URL: "https://lcd.terra.dev",
        chainID: "columbus-5",
        name: "mainnet",
        isClassic: true,
      }
    : CLUSTER === "testnet"
    ? {
        URL: "https://bombay-lcd.terra.dev",
        chainID: "bombay-12",
        name: "testnet",
        isClassic: true,
      }
    : {
        URL: TERRA_NODE_URL,
        chainID: "columbus-5",
        name: "localterra",
        isClassic: true,
      };

export const NEAR_NODE_URL = ci ? "http://near:3030" : "http://localhost:3030";

export const APTOS_NODE_URL = ci
  ? "http://aptos:8080/v1"
  : "http://0.0.0.0:8080/v1";
export const APTOS_FAUCET_URL = ci
  ? "http://aptos:8081"
  : "http://0.0.0.0:8081";
export const APTOS_PRIVATE_KEY =
  "537c1f91e56891445b491068f519b705f8c0f1a1e66111816dd5d4aa85b8113d";

export const SUI_NODE_URL = ci ? "http://sui:9000" : "http://localhost:9000";
