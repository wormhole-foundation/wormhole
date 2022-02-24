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
} = require("@certusone/wormhole-sdk");
const { default: axios } = require("axios");

const ETH_NODE_URL = "ws://localhost:8545";
const ETH_PRIVATE_KEY =
  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d";
const ETH_CORE_BRIDGE_ADDRESS = "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550";
const ETH_TOKEN_BRIDGE_ADDRESS = "0x0290FB167208Af455bB137780163b7B7a9a10C16";
const TEST_ERC20 = "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A";
const WORMHOLE_RPC_HOST = "http://localhost:7071";

(async () => {
  // create a signer for Eth
  const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
  const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
  console.log(`Height ${await provider.getBlockNumber()}`);

  // TEST 1 - send with 20 blocks conf
  const amount = parseUnits("1", 18);
  // approve the bridge to spend tokens
  await approveEth(ETH_TOKEN_BRIDGE_ADDRESS, TEST_ERC20, signer, amount);
  // transfer tokens
  const receipt = await transferFromEth(
    ETH_TOKEN_BRIDGE_ADDRESS,
    signer,
    TEST_ERC20,
    amount,
    CHAIN_ID_BSC,
    hexToUint8Array(
      nativeToHexString(await signer.getAddress(), CHAIN_ID_ETH) || ""
    )
  );
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = parseSequenceFromLogEth(receipt, ETH_CORE_BRIDGE_ADDRESS);
  const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
  console.log(`FIRST TX, SEQ ${sequence}`);
  for (let i = 1; i <= 20; i++) {
    console.log(`Attempt ${i}`);
    await axios.post("http://localhost:8545", {
      id: 1337,
      jsonrpc: "2.0",
      method: "evm_mine",
      params: [Date.now()],
    });
    console.log(`Height ${await provider.getBlockNumber()}`);
    try {
      const { vaaBytes: signedVAA } = await getSignedVAA(
        WORMHOLE_RPC_HOST,
        CHAIN_ID_ETH,
        emitterAddress,
        sequence,
        {
          transport: NodeHttpTransport(),
        }
      );
      console.log(!!signedVAA);
    } catch (e) {
      console.error(e.message);
    }
  }

  // TEST 2 - send with 1 conf, rollback before tx, 30 confs
  const {
    data: { result: snapshotId },
  } = await axios.post("http://localhost:8545", {
    id: 1337,
    jsonrpc: "2.0",
    method: "evm_snapshot",
    params: [],
  });
  console.log(`SNAPSHOT ${snapshotId}`);
  console.log(`Height ${await provider.getBlockNumber()}`);
  const amount2 = parseUnits("2", 18);
  // approve the bridge to spend tokens
  await approveEth(ETH_TOKEN_BRIDGE_ADDRESS, TEST_ERC20, signer, amount2);
  // transfer tokens
  const receipt2 = await transferFromEth(
    ETH_TOKEN_BRIDGE_ADDRESS,
    signer,
    TEST_ERC20,
    amount2,
    CHAIN_ID_BSC,
    hexToUint8Array(
      nativeToHexString(await signer.getAddress(), CHAIN_ID_ETH) || ""
    )
  );
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence2 = parseSequenceFromLogEth(receipt2, ETH_CORE_BRIDGE_ADDRESS);
  console.log(`SECOND TX, SEQ ${sequence2}`);
  console.log(`Attempt 1`);
  await axios.post("http://localhost:8545", {
    id: 1337,
    jsonrpc: "2.0",
    method: "evm_mine",
    params: [Date.now()],
  });
  console.log(`Height ${await provider.getBlockNumber()}`);
  try {
    const { vaaBytes: signedVAA } = await getSignedVAA(
      WORMHOLE_RPC_HOST,
      CHAIN_ID_ETH,
      emitterAddress,
      sequence2,
      {
        transport: NodeHttpTransport(),
      }
    );
    console.log(!!signedVAA);
  } catch (e) {
    console.error(e.message);
  }
  console.log(`Rollback 1`);
  await axios.post("http://localhost:8545", {
    id: 1337,
    jsonrpc: "2.0",
    method: "evm_revert",
    params: [snapshotId],
  });
  console.log(`Height ${await provider.getBlockNumber()}`);
  try {
    const { vaaBytes: signedVAA } = await getSignedVAA(
      WORMHOLE_RPC_HOST,
      CHAIN_ID_ETH,
      emitterAddress,
      sequence2,
      {
        transport: NodeHttpTransport(),
      }
    );
    console.log(!!signedVAA);
  } catch (e) {
    console.error(e.message);
  }
  for (let i = 1; i <= 30; i++) {
    console.log(`Attempt ${i}`);
    await axios.post("http://localhost:8545", {
      id: 1337,
      jsonrpc: "2.0",
      method: "evm_mine",
      params: [Date.now()],
    });
    console.log(`Height ${await provider.getBlockNumber()}`);
    try {
      const { vaaBytes: signedVAA } = await getSignedVAA(
        WORMHOLE_RPC_HOST,
        CHAIN_ID_ETH,
        emitterAddress,
        sequence2,
        {
          transport: NodeHttpTransport(),
        }
      );
      console.log(!!signedVAA);
    } catch (e) {
      console.error(e.message);
    }
  }

  // TEST 3 - repeat test 1 and verify test 2 is still not found
  const amount3 = parseUnits("3", 18);
  // approve the bridge to spend tokens
  await approveEth(ETH_TOKEN_BRIDGE_ADDRESS, TEST_ERC20, signer, amount3);
  // transfer tokens
  const receipt3 = await transferFromEth(
    ETH_TOKEN_BRIDGE_ADDRESS,
    signer,
    TEST_ERC20,
    amount3,
    CHAIN_ID_BSC,
    hexToUint8Array(
      nativeToHexString(await signer.getAddress(), CHAIN_ID_ETH) || ""
    )
  );
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence3 = parseSequenceFromLogEth(receipt3, ETH_CORE_BRIDGE_ADDRESS);
  console.log(`FIRST TX, SEQ ${sequence3}`);
  for (let i = 1; i <= 20; i++) {
    console.log(`Attempt ${i}`);
    await axios.post("http://localhost:8545", {
      id: 1337,
      jsonrpc: "2.0",
      method: "evm_mine",
      params: [Date.now()],
    });
    console.log(`Height ${await provider.getBlockNumber()}`);
    try {
      const { vaaBytes: signedVAA } = await getSignedVAA(
        WORMHOLE_RPC_HOST,
        CHAIN_ID_ETH,
        emitterAddress,
        sequence3,
        {
          transport: NodeHttpTransport(),
        }
      );
      console.log(!!signedVAA);
    } catch (e) {
      console.error(e.message);
    }
  }
  console.log("Checking SEQ from test 2");
  try {
    const { vaaBytes: signedVAA } = await getSignedVAA(
      WORMHOLE_RPC_HOST,
      CHAIN_ID_ETH,
      emitterAddress,
      sequence2,
      {
        transport: NodeHttpTransport(),
      }
    );
    console.log(!!signedVAA);
  } catch (e) {
    console.error(e.message);
  }

  provider.destroy();
})();
