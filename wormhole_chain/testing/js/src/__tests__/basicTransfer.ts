import { describe, jest, test, expect, it } from "@jest/globals";
import {
  HOLE_DENOM,
  TEST_WALLET_MNEMONIC_1,
  TEST_WALLET_MNEMONIC_2,
} from "../consts";
import {
  faucet,
  getAddress,
  getBalance,
  getWallet,
  sendTokens,
} from "../core/walletHelpers";

jest.setTimeout(60000);

test("faucet => wallet 1 => wallet 2", async () => {
  try {
    const DENOM = "utest";

    const wallet1 = await getWallet(TEST_WALLET_MNEMONIC_1);
    const wallet2 = await getWallet(TEST_WALLET_MNEMONIC_2);
    const wallet1Address = await getAddress(wallet1);
    const wallet2Address = await getAddress(wallet2);
    const wallet1InitialBalance = await getBalance(DENOM, wallet1Address);
    const wallet2InitialBalance = await getBalance(DENOM, wallet2Address);

    // console.log("calling faucet");

    // await faucet(HOLE_DENOM, "100", wallet1Address);

    // const wallet1BalanceAfterFaucet = await getBalance(
    //   'utest',
    //   wallet1Address
    // );

    // console.log("wallet 1 after faucet", wallet1BalanceAfterFaucet);

    //expect(wallet1BalanceAfterFaucet - wallet1InitialBalance).toBe(100);

    await sendTokens(wallet1, DENOM, BigInt(100), wallet2Address);

    const wallet1BalanceAfterTransfer = await getBalance(DENOM, wallet1Address);
    const wallet2BalanceAfterTransfer = await getBalance(DENOM, wallet2Address);

    expect(wallet1InitialBalance === wallet1BalanceAfterTransfer).toBe(true);
    expect(wallet2BalanceAfterTransfer - wallet2InitialBalance).toBe(100);
  } catch (e) {
    console.error(e);
    expect(true).toBe(false);
  }
});
