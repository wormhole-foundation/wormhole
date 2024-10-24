import { beforeAll, describe, expect, test } from "@jest/globals";
import {
  LCDClient,
  MnemonicKey,
  Msg,
  Wallet,
  isTxError,
} from "@terra-money/terra.js";
import { ethers } from "ethers";
import { parseUnits } from "ethers/lib/utils";
import {
  createWrappedOnEth,
  createWrappedOnTerra,
  getEmitterAddressEth,
  getEmitterAddressTerra,
  getIsTransferCompletedTerra,
  getIsTransferCompletedTerra2,
  parseSequenceFromLogEth,
  parseSequenceFromLogTerra,
  redeemOnEth,
  redeemOnTerra,
  updateWrappedOnEth,
} from "../..";
import { tryNativeToUint8Array } from "../../utils";
import {
  CHAIN_ID_ETH,
  CHAIN_ID_TERRA,
  CHAIN_ID_TERRA2,
  CONTRACTS,
} from "../../utils/consts";
import { attestFromEth, attestFromTerra } from "../attest";
import { approveEth, transferFromEth, transferFromTerra } from "../transfer";
import {
  ETH_NODE_URL,
  ETH_PRIVATE_KEY2,
  TERRA2_NODE_URL,
  TERRA2_PRIVATE_KEY,
  TERRA_CHAIN_ID,
  TERRA_NODE_URL,
  TERRA_PRIVATE_KEY2,
  TEST_ERC20,
} from "./utils/consts";
import { getSignedVAABySequence, waitForTerraExecution } from "./utils/helpers";

const lcd = new LCDClient({
  URL: TERRA2_NODE_URL,
  chainID: TERRA_CHAIN_ID,
});
const terraWallet = lcd.wallet(
  new MnemonicKey({ mnemonic: TERRA2_PRIVATE_KEY })
);
const terraWalletAddress = terraWallet.key.accAddress;

const lcdClassic = new LCDClient({
  URL: TERRA_NODE_URL,
  chainID: TERRA_CHAIN_ID,
  isClassic: true,
});
const terraClassicWallet = lcdClassic.wallet(
  new MnemonicKey({ mnemonic: TERRA_PRIVATE_KEY2 })
);
const terraClassicWalletAddress = terraClassicWallet.key.accAddress;

const provider = new ethers.providers.JsonRpcProvider(ETH_NODE_URL);
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

const terraBroadcastAndWaitForExecution = async (
  msgs: Msg[],
  wallet: Wallet,
  isClassic = false
) => {
  const tx = await wallet.createAndSignTx({
    msgs,
  });
  const _lcd = isClassic ? lcdClassic : lcd;
  const txResult = await _lcd.tx.broadcast(tx);
  if (isTxError(txResult)) {
    throw new Error("tx error");
  }
  const txInfo = await waitForTerraExecution(txResult.txhash, _lcd);
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

describe("Terra Integration Tests", () => {
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
    await provider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
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
    await provider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`
    const transferSignedVaa = await ethParseLogAndGetSignedVaa(transferReceipt);
    const redeemMsg = await redeemOnTerra(
      CONTRACTS.DEVNET.terra2.token_bridge,
      terraWalletAddress,
      transferSignedVaa
    );
    expect(
      await getIsTransferCompletedTerra2(
        CONTRACTS.DEVNET.terra2.token_bridge,
        transferSignedVaa,
        lcd
      )
    ).toBe(false);
    await terraBroadcastAndWaitForExecution([redeemMsg], terraWallet);
    expect(
      await getIsTransferCompletedTerra2(
        CONTRACTS.DEVNET.terra2.token_bridge,
        transferSignedVaa,
        lcd
      )
    ).toBe(true);
  });

  test("Attest and transfer Terra2 native token to Terra Classic", async () => {
    const attestMsg = await attestFromTerra(
      CONTRACTS.DEVNET.terra2.token_bridge,
      terraWalletAddress,
      "uluna"
    );
    const attestSignedVaa = await terraBroadcastTxAndGetSignedVaa(
      [attestMsg],
      terraWallet
    );
    const createWrappedMsg = await createWrappedOnTerra(
      CONTRACTS.DEVNET.terra.token_bridge,
      terraClassicWalletAddress,
      attestSignedVaa
    );
    await terraBroadcastAndWaitForExecution(
      [createWrappedMsg],
      terraClassicWallet,
      true
    );
    // Transfer
    const transferMsgs = await transferFromTerra(
      terraWalletAddress,
      CONTRACTS.DEVNET.terra2.token_bridge,
      "uluna",
      "1000000",
      CHAIN_ID_TERRA,
      tryNativeToUint8Array(terraClassicWalletAddress, CHAIN_ID_TERRA)
    );
    const transferSignedVaa = await terraBroadcastTxAndGetSignedVaa(
      transferMsgs,
      terraWallet
    );
    const redeemMsg = await redeemOnTerra(
      CONTRACTS.DEVNET.terra.token_bridge,
      terraClassicWalletAddress,
      transferSignedVaa
    );
    await terraBroadcastAndWaitForExecution(
      [redeemMsg],
      terraClassicWallet,
      true
    );
    expect(
      await getIsTransferCompletedTerra(
        CONTRACTS.DEVNET.terra.token_bridge,
        transferSignedVaa,
        lcdClassic
      )
    ).toBe(true);
  });
});
