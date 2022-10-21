import { expect, jest, test } from "@jest/globals";
import { getAddress, getWallet } from "@wormhole-foundation/wormchain-sdk";
import { TEST_WALLET_MNEMONIC_1, TEST_WALLET_MNEMONIC_2 } from "../consts";
import { getBalance, sendTokens } from "../utils/walletHelpers";

jest.setTimeout(60000);

test("basicTransfer", async () => {
  try {
    const DENOM = "utest";
    const wallet1 = await getWallet(TEST_WALLET_MNEMONIC_1);
    console.log("wallet 1", wallet1);
    const wallet2 = await getWallet(TEST_WALLET_MNEMONIC_2);
    console.log("wallet 2", wallet2);
    const wallet1Address = await getAddress(wallet1);
    console.log("wallet 1 address", wallet1Address);
    const wallet2Address = await getAddress(wallet2);
    console.log("wallet 2 address", wallet2Address);
    const wallet1InitialBalance = parseInt(
      await getBalance(wallet1Address, DENOM)
    );
    console.log("wallet 1 init", wallet1InitialBalance);
    const wallet2InitialBalance = parseInt(
      await getBalance(wallet2Address, DENOM)
    );
    console.log("wallet 2 init", wallet2InitialBalance);

    await sendTokens(wallet1, wallet2Address, "100", DENOM);

    const wallet1BalanceAfterTransfer = parseInt(
      await getBalance(wallet1Address, DENOM)
    );
    console.log("wallet 1 afer", wallet1BalanceAfterTransfer);
    const wallet2BalanceAfterTransfer = parseInt(
      await getBalance(wallet2Address, DENOM)
    );
    console.log("wallet 2 after", wallet2BalanceAfterTransfer);

    expect(wallet1InitialBalance - wallet1BalanceAfterTransfer).toBe(100);
    expect(wallet2BalanceAfterTransfer - wallet2InitialBalance).toBe(100);
  } catch (e) {
    console.error(e);
    expect(true).toBe(false);
  }
});
