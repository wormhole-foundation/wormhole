import { describe, jest, test, expect } from "@jest/globals";
import {
  fromAccAddress,
  fromBase64,
  fromValAddress,
  getAddress,
  getOperatorWallet,
  getWallet,
  toAccAddress,
  toBase64,
  toValAddress,
} from "@wormhole-foundation/wormchain-sdk";
import {
  GUARDIAN_VALIDATOR_BASE64_VALADDR,
  GUARDIAN_VALIDATOR_VALADDR,
  TEST_WALLET_ADDRESS_1,
  TEST_WALLET_MNEMONIC_1,
} from "../consts";

jest.setTimeout(60000);

describe("SDK tests", () => {
  test("Address manipulation", (done) => {
    const accountAddress = TEST_WALLET_ADDRESS_1;
    const validatorAddress = GUARDIAN_VALIDATOR_VALADDR;
    const validatorAddr64 = GUARDIAN_VALIDATOR_BASE64_VALADDR;

    //checking invertibility
    expect(
      accountAddress === toAccAddress(fromAccAddress(accountAddress))
    ).toBe(true);
    expect(
      validatorAddress === toValAddress(fromValAddress(validatorAddress))
    ).toBe(true);
    expect(validatorAddr64 === toBase64(fromBase64(validatorAddr64))).toBe(
      true
    );

    //fromBase64
    expect(accountAddress === toAccAddress(fromBase64(validatorAddr64))).toBe(
      true
    );
    expect(validatorAddress === toValAddress(fromBase64(validatorAddr64))).toBe(
      true
    );

    //fromAcc
    //expect(something === toBase64(fromAccAddress(accountAddress))).toBe(true); //TODO don't have this string
    expect(
      validatorAddress === toValAddress(fromAccAddress(accountAddress))
    ).toBe(true);

    //fromValAddr
    expect(
      accountAddress === toAccAddress(fromValAddress(validatorAddress))
    ).toBe(true);
    expect(validatorAddr64 === toBase64(fromValAddress(validatorAddress))).toBe(
      true
    );

    //todo conversion tests
    done();
  });
  test("Wallet instantiation", (done) => {
    (async () => {
      const wallet = await getWallet(TEST_WALLET_MNEMONIC_1);
      const operWallet = await getOperatorWallet(TEST_WALLET_MNEMONIC_1);
      const address = await getAddress(wallet);
      const valAddr = await getAddress(operWallet);

      expect(address === TEST_WALLET_ADDRESS_1).toBe(true);
      expect(valAddr === GUARDIAN_VALIDATOR_VALADDR).toBe(true);

      done();
    })();
  });
});
