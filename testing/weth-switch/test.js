// This test is intended to be run on devnet without an active eth miner
// see https://github.com/trufflesuite/ganache-cli-archive#custom-methods

const {
  NodeHttpTransport,
} = require("@improbable-eng/grpc-web-node-http-transport");
const { ethers } = require("ethers");
const { parseUnits } = require("ethers/lib/utils");
const {
  approveEth,
  transferFromEth,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  getEmitterAddressEth,
  getSignedVAA,
  BridgeImplementation__factory,
} = require("@certusone/wormhole-sdk");

const BSC_NODE_URL = "ws://localhost:8546";
const ETH_NODE_URL = "ws://localhost:8545";
const ETH_PRIVATE_KEY =
  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d";
const ETH_TOKEN_BRIDGE_ADDRESS = "0x0290FB167208Af455bB137780163b7B7a9a10C16";
// see https://eips.ethereum.org/EIPS/eip-1967#logic-contract-address
const LOGIC_CONTRACT_STORAGE =
  "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc";
const SIGNED_VAA =
  "010000000001003fd2219eed5b1a433120cf3edc37ab6c28f86222aa552e294e175e7e42ff32301c5c936b16ae6470bdfedbe9d7cdb50e664aac1e19da8489a7369caa7e1b5439010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000002a0518100000000000000000000000000000000000000000000546f6b656e427269646765020002000000000000000000000000daa71fbba28c946258dd3d5fcc9001401f72270f";

(async () => {
  // create a signer for Eth
  const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
  const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
  const tokenBridge = BridgeImplementation__factory.connect(
    ETH_TOKEN_BRIDGE_ADDRESS,
    signer
  );
  console.log(
    "OLD IMPL",
    await provider.getStorageAt(
      ETH_TOKEN_BRIDGE_ADDRESS,
      LOGIC_CONTRACT_STORAGE
    )
  );
  console.log("OLD WETH", await tokenBridge.WETH());
  console.log("UPGRADING...");
  await tokenBridge.upgrade("0x" + SIGNED_VAA);
  console.log("SUCCESS!");
  console.log(
    "NEW IMPL",
    await provider.getStorageAt(
      ETH_TOKEN_BRIDGE_ADDRESS,
      LOGIC_CONTRACT_STORAGE
    )
  );
  console.log("NEW WETH", await tokenBridge.WETH());

  provider.destroy();
})();
