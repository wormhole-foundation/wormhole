import { expect, jest, test } from "@jest/globals";
import {
  fromAccAddress,
  getWallet,
  getWormchainSigningClient,
  toValAddress,
} from "@wormhole-foundation/wormchain-sdk";
import { getZeroFee } from "../bootstrap";
import {
  DEVNET_GUARDIAN_PRIVATE_KEY,
  DEVNET_GUARDIAN_PUBLIC_KEY,
  TENDERMINT_URL,
  TEST_WALLET_ADDRESS_2,
  TEST_WALLET_MNEMONIC_2,
} from "../consts";
import { signValidatorAddress } from "../utils/walletHelpers";

jest.setTimeout(60000);

/*
This file tests to make sure that the network can start from genesis, and then change out the guardian set.

Prerequesites: Have two nodes running - the tilt guardian validator, and the 'second' wormhole chain node.

This test will register the the public ket of the second node, and then process a governance VAA to switch over the network.

*/

test("Verify guardian validator", async () => {});

test("Process guardian set upgrade", async () => {});

test("RegisterGuardianValidator", async () => {
  const wallet = await getWallet(TEST_WALLET_MNEMONIC_2);
  const signingClient = await getWormchainSigningClient(TENDERMINT_URL, wallet);
  const registerMsg = signingClient.core.msgRegisterAccountAsGuardian({
    guardianPubkey: { key: Buffer.from(DEVNET_GUARDIAN_PUBLIC_KEY, "hex") },
    signer: TEST_WALLET_ADDRESS_2,
    signature: signValidatorAddress(
      toValAddress(fromAccAddress(TEST_WALLET_ADDRESS_2)),
      DEVNET_GUARDIAN_PRIVATE_KEY
    ),
  });
  const registerMsgReceipe = await signingClient.signAndBroadcast(
    TEST_WALLET_ADDRESS_2,
    [registerMsg],
    getZeroFee()
  );
  console.log("transaction hash: " + registerMsgReceipe.transactionHash);
  expect(registerMsgReceipe.code === 0).toBe(true);
});
