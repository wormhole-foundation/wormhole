import {
  approveEth,
  attestFromEth,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CONTRACTS,
  createWrappedOnEth,
  getEmitterAddressEth,
  getForeignAssetEth,
  getSignedVAAWithRetry,
  hexToUint8Array,
  parseSequenceFromLogEth,
  redeemOnEth,
  transferFromEth,
  tryNativeToUint8Array,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import { ethers } from "ethers";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { parseUnits } from "ethers/lib/utils";

// see devnet.md
const ci = false;
const ETH_NODE_URL = ci ? "ws://eth-devnet:8545" : "ws://localhost:8545";
const EVM2_NODE_URL = ci ? "ws://eth-devnet:8546" : "ws://localhost:8546";
const ETH_PRIVATE_KEY =
  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"; // account 0
const TEST_ERC20 = "0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A";
const WORMHOLE_RPC_HOSTS = ci
  ? ["http://guardian:7071"]
  : ["http://localhost:7071"];

(async () => {
  /* Test 1
   *
   * A legitimate transfer of source=origin tokens
   *
   */
  console.log("Attesting Token");
  // create a signer for Eth
  const providerETH = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
  const signerETH = new ethers.Wallet(ETH_PRIVATE_KEY, providerETH);
  // attest the test token
  const receipt = await attestFromEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signerETH,
    TEST_ERC20
  );
  console.log(receipt.transactionHash);
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = parseSequenceFromLogEth(
    receipt,
    CONTRACTS.DEVNET.ethereum.core
  );
  const emitterAddress = getEmitterAddressEth(
    CONTRACTS.DEVNET.ethereum.token_bridge
  );
  console.log(`waiting for ${CHAIN_ID_ETH}/${emitterAddress}/${sequence}`);
  // poll until the guardian(s) witness and sign the vaa
  const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    CHAIN_ID_ETH,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(),
    }
  );
  console.log("fetched vaa", uint8ArrayToHex(signedVAA));
  // create a signer for Eth
  const providerBSC = new ethers.providers.WebSocketProvider(EVM2_NODE_URL);
  const signerBSC = new ethers.Wallet(ETH_PRIVATE_KEY, providerBSC);
  try {
    await createWrappedOnEth(
      CONTRACTS.DEVNET.bsc.token_bridge,
      signerBSC,
      signedVAA
    );
    console.log("Created!");
  } catch (e) {
    console.error(e);
    // this could fail because the token is already attested (in an unclean env)
  }
  const DECIMALS = 18;
  const recipientAddress = await signerBSC.getAddress();
  console.log("sending to", recipientAddress);
  const amount = parseUnits("1", DECIMALS);

  // approve the bridge to spend tokens
  await approveEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    TEST_ERC20,
    signerETH,
    amount
  );
  console.log("approved");
  // transfer tokens
  const receipt2 = await transferFromEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signerETH,
    TEST_ERC20,
    amount,
    CHAIN_ID_BSC,
    tryNativeToUint8Array(recipientAddress.toString(), CHAIN_ID_BSC)
  );
  console.log("transfer tx", receipt2.transactionHash);
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence2 = parseSequenceFromLogEth(
    receipt2,
    CONTRACTS.DEVNET.ethereum.core
  );
  console.log(`waiting for ${CHAIN_ID_ETH}/${emitterAddress}/${sequence2}`);
  // poll until the guardian(s) witness and sign the vaa
  const { vaaBytes: signedVAA2 } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    CHAIN_ID_ETH,
    emitterAddress,
    sequence2,
    {
      transport: NodeHttpTransport(),
    }
  );
  console.log("redeeming", uint8ArrayToHex(signedVAA2));
  await redeemOnEth(CONTRACTS.DEVNET.bsc.token_bridge, signerBSC, signedVAA2);
  console.log("redeemed");
  /* Test 2
   *
   * A legitimate transfer of source=foreign tokens
   *
   */
  const testErc20OnBsc = await getForeignAssetEth(
    CONTRACTS.DEVNET.bsc.token_bridge,
    providerBSC,
    "ethereum",
    tryNativeToUint8Array(TEST_ERC20, "ethereum")
  );
  if (!testErc20OnBsc) {
    throw new Error("wrapped test erc not found");
  }
  console.log("sending", testErc20OnBsc, "from BSC to Eth");

  // approve the bridge to spend tokens
  await approveEth(
    CONTRACTS.DEVNET.bsc.token_bridge,
    testErc20OnBsc,
    signerBSC,
    amount
  );
  console.log("approved");
  // transfer tokens
  const receipt3 = await transferFromEth(
    CONTRACTS.DEVNET.bsc.token_bridge,
    signerBSC,
    testErc20OnBsc,
    amount,
    CHAIN_ID_ETH,
    tryNativeToUint8Array(recipientAddress.toString(), CHAIN_ID_BSC)
  );
  console.log("transfer tx", receipt3.transactionHash);
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence3 = parseSequenceFromLogEth(
    receipt3,
    CONTRACTS.DEVNET.ethereum.core
  );
  console.log(`waiting for ${CHAIN_ID_BSC}/${emitterAddress}/${sequence3}`);
  // poll until the guardian(s) witness and sign the vaa
  const { vaaBytes: signedVAA3 } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    CHAIN_ID_BSC,
    emitterAddress,
    sequence3,
    {
      transport: NodeHttpTransport(),
    }
  );
  console.log("redeeming", uint8ArrayToHex(signedVAA3));
  await redeemOnEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signerETH,
    signedVAA3
  );
  console.log("redeemed");
  /* Test 2
   *
   * A bad transfer of source=foreign tokens
   *
   */
  // {
  //   version: 1,
  //   guardianSetIndex: 0,
  //   signatures: [
  //     {
  //       guardianSetIndex: 0,
  //       signature: '5b3213e0dda29902c018d084769125227c758f6935358e814a2cbb26bc167798329e0ba9618926b7e1d7bf9301f711d7d7e34230c3e053be85e95141af1dd03b01'
  //     }
  //   ],
  //   timestamp: 1,
  //   nonce: 1,
  //   emitterChain: 2,
  //   emitterAddress: '0x0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16',
  //   sequence: 2n,
  //   consistencyLevel: 0,
  //   payload: {
  //     module: 'TokenBridge',
  //     type: 'Transfer',
  //     amount: 100000000n,
  //     tokenAddress: '0x0000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a',
  //     tokenChain: 2,
  //     toAddress: '0x00000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1',
  //     chain: 4,
  //     fee: 0n
  //   },
  //   digest: '0xe2bd712489209293b71b4ae29c9aad85df6cf50332f29a2b4d21e60f9e25d3b4'
  // }
  const SPOOFED =
    "010000000001005b3213e0dda29902c018d084769125227c758f6935358e814a2cbb26bc167798329e0ba9618926b7e1d7bf9301f711d7d7e34230c3e053be85e95141af1dd03b01000000010000000100020000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C16000000000000000200010000000000000000000000000000000000000000000000000000000005f5e1000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a000200000000000000000000000090F8bf6A479f320ead074411a4B0e7944Ea8c9C100040000000000000000000000000000000000000000000000000000000000000000";
  console.log("spoofed VAA", SPOOFED);
  await redeemOnEth(
    CONTRACTS.DEVNET.bsc.token_bridge,
    signerBSC,
    hexToUint8Array(SPOOFED)
  );
  console.log("sending", testErc20OnBsc, "from BSC to Eth");
  // approve the bridge to spend tokens
  await approveEth(
    CONTRACTS.DEVNET.bsc.token_bridge,
    testErc20OnBsc,
    signerBSC,
    amount
  );
  console.log("approved");
  // transfer tokens
  const receipt4 = await transferFromEth(
    CONTRACTS.DEVNET.bsc.token_bridge,
    signerBSC,
    testErc20OnBsc,
    amount,
    CHAIN_ID_ETH,
    tryNativeToUint8Array(recipientAddress.toString(), CHAIN_ID_BSC)
  );
  console.log("transfer tx", receipt4.transactionHash);
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence4 = parseSequenceFromLogEth(
    receipt4,
    CONTRACTS.DEVNET.ethereum.core
  );
  console.log(
    `CHECK GUARDIAN "ACCT" LOGS FOR ${CHAIN_ID_BSC}/${emitterAddress}/${sequence4}`
  );
  providerETH.destroy();
  providerBSC.destroy();
})();
