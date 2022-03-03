
import { describe, jest, test, expect } from "@jest/globals";
import { HOLE_DENOM, TEST_TRANSFER_VAA, TEST_WALLET_MNEMONIC_1, TEST_WALLET_MNEMONIC_2 } from "../consts";
import { redeemVaaMsg } from "../core/tokenBridge";
import { faucet, getAddress, getBalance, getWallet, sendTokens, signSendAndConfirm } from "../core/walletHelpers";

jest.setTimeout(60000);

describe("Token bridge tests", () => {
    test("simple VAA redeem", (done) => {
      (async () => {
          try {
            const wallet1 = await getWallet(TEST_WALLET_MNEMONIC_1);
            const wallet1Address = await getAddress(wallet1)
            const wallet1InitialBalance = await getBalance(HOLE_DENOM, wallet1Address);

            //VAA for 100 hole to this specific wallet
            const msg = redeemVaaMsg(TEST_TRANSFER_VAA);
            await signSendAndConfirm(wallet1, [msg]);

            const wallet1BalanceAfterTransfer = await getBalance(HOLE_DENOM, wallet1Address);

            expect(wallet1BalanceAfterTransfer - wallet1InitialBalance).toBe(100000000); //100 hole = 100000000 uhole

            done();
          } catch (e){
              expect(true).toBe(false);
              console.error(e);
              done();
          }
      })();
    });
});
