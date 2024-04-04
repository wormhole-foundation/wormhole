import { beforeAll, expect, jest, test } from "@jest/globals";
import { ethers } from "ethers";
import { parseUnits } from "ethers/lib/utils";
import { Account, KeyPair, Near, connect, keyStores } from "near-api-js";
import {
  FinalExecutionOutcome,
  Provider,
  getTransactionLastResult,
} from "near-api-js/lib/providers";
import { parseNearAmount } from "near-api-js/lib/utils/format";
import {
  createWrappedOnEth,
  createWrappedOnNear,
  getEmitterAddressEth,
  getEmitterAddressNear,
  getIsTransferCompletedEth,
  getIsTransferCompletedNear,
  hashAccount,
  hexToUint8Array,
  parseSequenceFromLogEth,
  parseSequenceFromLogNear,
  redeemOnEth,
  redeemOnNear,
  registerAccount,
  tryNativeToUint8Array,
  updateWrappedOnEth,
} from "../..";
import { CHAIN_ID_ETH, CHAIN_ID_NEAR, CONTRACTS } from "../../utils/consts";
import { attestFromEth, attestNearFromNear } from "../attest";
import { approveEth, transferFromEth, transferNearFromNear } from "../transfer";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY5,
  NEAR_NODE_URL,
  TEST_ERC20,
} from "./utils/consts";
import { getSignedVAABySequence } from "./utils/helpers";

let near: Near;
let nearProvider: Provider;
let account: Account;
const networkId = "sandbox";
const accountId = "devnet.test.near";
const PRIVATE_KEY =
  "ed25519:nCW2EsTn91b7ettRqQX6ti8ZBNwo7tbMsenBu9nmSVG9aDhNB7hgw7S9w5M9CZu1bF23FbvhKZPfDmh2Gbs45Fs";

const ethProvider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
const signer = new ethers.Wallet(ETH_PRIVATE_KEY5, ethProvider);
const ethEmitterAddress = getEmitterAddressEth(
  CONTRACTS.DEVNET.ethereum.token_bridge
);
const nearEmitterAddress = getEmitterAddressNear(
  CONTRACTS.DEVNET.near.token_bridge
);
const ethTransferAmount = parseUnits("1", 18);
let ethWalletAddress: string;

beforeAll(async () => {
  const keyStore = new keyStores.InMemoryKeyStore();
  await keyStore.setKey(networkId, accountId, KeyPair.fromString(PRIVATE_KEY));
  const config = {
    keyStore,
    networkId,
    nodeUrl: NEAR_NODE_URL,
    headers: {},
  };
  near = await connect(config);
  nearProvider = near.connection.provider;
  account = await near.account(accountId);
  ethWalletAddress = await signer.getAddress();
});

const nearParseLogAndGetSignedVaa = async (outcome: FinalExecutionOutcome) => {
  const sequence = parseSequenceFromLogNear(outcome);
  if (sequence === null) {
    throw new Error("sequence is null");
  }
  return await getSignedVAABySequence(
    CHAIN_ID_NEAR,
    sequence,
    nearEmitterAddress
  );
};

const ethParseLogAndGetSignedVaa = async (receipt: ethers.ContractReceipt) => {
  const sequence = parseSequenceFromLogEth(
    receipt,
    CONTRACTS.DEVNET.ethereum.core
  );
  return await getSignedVAABySequence(
    CHAIN_ID_ETH,
    sequence,
    ethEmitterAddress
  );
};

test("Attest and transfer Near token from Near to Ethereum", async () => {
  // Attest
  const attestMsg = await attestNearFromNear(
    account.connection.provider,
    CONTRACTS.DEVNET.near.core,
    CONTRACTS.DEVNET.near.token_bridge
  );
  const attestOutcome = await account.functionCall(attestMsg);
  const attestSignedVaa = await nearParseLogAndGetSignedVaa(attestOutcome);
  try {
    await createWrappedOnEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      signer,
      attestSignedVaa
    );
  } catch {
    await updateWrappedOnEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      signer,
      attestSignedVaa
    );
  }
  // Transfer
  const transferMsg = await transferNearFromNear(
    account.connection.provider,
    CONTRACTS.DEVNET.near.core,
    CONTRACTS.DEVNET.near.token_bridge,
    BigInt(parseNearAmount("1") || ""),
    tryNativeToUint8Array(ethWalletAddress, "ethereum"),
    "ethereum",
    BigInt(0)
  );
  const transferOutcome = await account.functionCall(transferMsg);
  const transferSignedVAA = await nearParseLogAndGetSignedVaa(transferOutcome);
  await redeemOnEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signer,
    transferSignedVAA
  );
  expect(
    await getIsTransferCompletedEth(
      CONTRACTS.DEVNET.ethereum.token_bridge,
      ethProvider,
      transferSignedVAA
    )
  ).toBe(true);
});

test("Attest and transfer token from Ethereum to Near", async () => {
  // Attest
  const attestReceipt = await attestFromEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signer,
    TEST_ERC20
  );
  await ethProvider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
  const attestSignedVaa = await ethParseLogAndGetSignedVaa(attestReceipt);
  const createWrappedMsgs = await createWrappedOnNear(
    nearProvider,
    CONTRACTS.DEVNET.near.token_bridge,
    attestSignedVaa
  );
  for (const msg of createWrappedMsgs) {
    await account.functionCall(msg);
  }
  // Transfer
  await approveEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    TEST_ERC20,
    signer,
    ethTransferAmount
  );
  let { isRegistered, accountHash } = await hashAccount(
    nearProvider,
    CONTRACTS.DEVNET.near.token_bridge,
    account.accountId
  );
  if (!isRegistered) {
    const registerAccountMsg = registerAccount(
      account.accountId,
      CONTRACTS.DEVNET.near.token_bridge
    );
    accountHash = getTransactionLastResult(
      await account.functionCall(registerAccountMsg)
    );
  }
  const transferReceipt = await transferFromEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signer,
    TEST_ERC20,
    ethTransferAmount,
    "near",
    hexToUint8Array(accountHash)
  );
  await ethProvider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
  const transferSignedVaa = await ethParseLogAndGetSignedVaa(transferReceipt);
  const redeemMsgs = await redeemOnNear(
    nearProvider,
    account.accountId,
    CONTRACTS.DEVNET.near.token_bridge,
    transferSignedVaa
  );
  expect(
    await getIsTransferCompletedNear(
      nearProvider,
      CONTRACTS.DEVNET.near.token_bridge,
      transferSignedVaa
    )
  ).toBe(false);
  for (const msg of redeemMsgs) {
    await account.functionCall(msg);
  }
  expect(
    await getIsTransferCompletedNear(
      nearProvider,
      CONTRACTS.DEVNET.near.token_bridge,
      transferSignedVaa
    )
  ).toBe(true);
});
