import { beforeAll, afterAll, expect, test } from "@jest/globals";
import {
  isTxError,
  LCDClient,
  MnemonicKey,
  Msg,
  Wallet,
} from "@terra-money/terra.js";
import { ethers } from "ethers";
import { parseUnits } from "ethers/lib/utils";
import {
  createWrappedOnEth,
  createWrappedOnTerra,
  getEmitterAddressEth,
  getEmitterAddressTerra,
  getIsTransferCompletedTerra,
  parseSequenceFromLogEth,
  parseSequenceFromLogTerra,
  redeemOnEth,
  redeemOnTerra,
  updateWrappedOnEth,
} from "../..";
import { tryNativeToUint8Array } from "../../utils";
import { CHAIN_ID_ETH, CHAIN_ID_TERRA2, CONTRACTS } from "../../utils/consts";
import { attestFromEth, attestFromTerra } from "../attest";
import { approveEth, transferFromEth, transferFromTerra } from "../transfer";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY2,
  TERRA2_GAS_PRICES_URL,
  TERRA2_PRIVATE_KEY,
  TEST_ERC20,
} from "./consts";
import { getSignedVAABySequence, waitForTerraExecution } from "./helpers";

const lcd = new LCDClient({
  URL: !!process.env.CI ? "http://terra2-terrad:1317" : "http://localhost:1318",
  chainID: "localterra",
});
const terraWallet = lcd.wallet(
  new MnemonicKey({ mnemonic: TERRA2_PRIVATE_KEY })
);
const terraWalletAddress = terraWallet.key.accAddress;

const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
const signer = new ethers.Wallet(ETH_PRIVATE_KEY2, provider);
const ethEmitterAddress = getEmitterAddressEth(
  CONTRACTS.DEVNET.ethereum.token_bridge
);
const ethTransferAmount = parseUnits("1", 18);

let ethWalletAddress: string;
let terraEmitterAddress: string;

beforeAll(async () => {
  ethWalletAddress = await signer.getAddress();
  terraEmitterAddress = await getEmitterAddressTerra(
    CONTRACTS.DEVNET.terra2.token_bridge
  );
});

afterAll(async () => {
  provider.destroy();
});

const terraBroadcastAndWaitForExecution = async (
  msgs: Msg[],
  wallet: Wallet
) => {
  const tx = await wallet.createAndSignTx({
    msgs,
  });
  const txResult = await lcd.tx.broadcast(tx);
  if (isTxError(txResult)) {
    throw new Error("tx error");
  }
  const txInfo = await waitForTerraExecution(txResult.txhash, lcd);
  if (!txInfo) {
    throw new Error("tx info not found");
  }
  return txInfo;
};

const terraBroadcastTxAndGetSignedVaa = async (msgs: Msg[], wallet: Wallet) => {
  const txInfo = await terraBroadcastAndWaitForExecution(msgs, wallet);
  const txSequence = parseSequenceFromLogTerra(txInfo);
  if (!txSequence) {
    throw new Error("tx sequence not found");
  }
  return await getSignedVAABySequence(
    CHAIN_ID_TERRA2,
    txSequence,
    terraEmitterAddress
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

test("Attest and transfer token from Terra2 to Ethereum", async () => {
  // Attest
  const attestMsg = await attestFromTerra(
    CONTRACTS.DEVNET.terra2.token_bridge,
    terraWalletAddress,
    "uluna"
  );
  const attestSignedVaa = await terraBroadcastTxAndGetSignedVaa(
    [attestMsg],
    terraWallet
  );
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
  const transferMsgs = await transferFromTerra(
    terraWalletAddress,
    CONTRACTS.DEVNET.terra2.token_bridge,
    "uluna",
    "1000000",
    CHAIN_ID_ETH,
    tryNativeToUint8Array(ethWalletAddress, CHAIN_ID_ETH)
  );
  const transferSignedVaa = await terraBroadcastTxAndGetSignedVaa(
    transferMsgs,
    terraWallet
  );
  await redeemOnEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signer,
    transferSignedVaa
  );
});

test("Attest and transfer token from Ethereum to Terra2", async () => {
  // Attest
  const attestReceipt = await attestFromEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signer,
    TEST_ERC20
  );
  const attestSignedVaa = await ethParseLogAndGetSignedVaa(attestReceipt);
  const createWrappedMsg = await createWrappedOnTerra(
    CONTRACTS.DEVNET.terra2.token_bridge,
    terraWalletAddress,
    attestSignedVaa
  );
  await terraBroadcastAndWaitForExecution([createWrappedMsg], terraWallet);
  // Transfer
  await approveEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    TEST_ERC20,
    signer,
    ethTransferAmount
  );
  const transferReceipt = await transferFromEth(
    CONTRACTS.DEVNET.ethereum.token_bridge,
    signer,
    TEST_ERC20,
    ethTransferAmount,
    CHAIN_ID_TERRA2,
    tryNativeToUint8Array(terraWalletAddress, CHAIN_ID_TERRA2)
  );
  const transferSignedVaa = await ethParseLogAndGetSignedVaa(transferReceipt);
  const redeemMsg = await redeemOnTerra(
    CONTRACTS.DEVNET.terra2.token_bridge,
    terraWalletAddress,
    transferSignedVaa
  );
  await terraBroadcastAndWaitForExecution([redeemMsg], terraWallet);
  expect(
    await getIsTransferCompletedTerra(
      CONTRACTS.DEVNET.terra2.token_bridge,
      transferSignedVaa,
      lcd,
      TERRA2_GAS_PRICES_URL
    )
  ).toBe(true);
});
