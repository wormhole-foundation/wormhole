import { coins } from "@cosmjs/proto-signing";
import { DeliverTxResponse, StdFee } from "@cosmjs/stargate";
import axios from "axios";
import pkg from "protobufjs";
const { Field, Type } = pkg;
import * as sdk from "@wormhole-foundation/wormchain-sdk";
import {
  fromAccAddress,
  fromValAddress,
  toBase64,
  toValAddress,
} from "@wormhole-foundation/wormchain-sdk";
import {
  DEVNET_GUARDIAN2_PRIVATE_KEY,
  DEVNET_GUARDIAN2_PUBLIC_KEY,
  GUARDIAN_VALIDATOR_VALADDR,
  WORM_DENOM,
  NODE_URL,
  TENDERMINT_URL,
  TEST_TRANSFER_VAA_1,
  TEST_WALLET_ADDRESS_2,
  TEST_WALLET_MNEMONIC_2,
  TILTNET_GUARDIAN_PUBKEY,
  UPGRADE_GUARDIAN_SET_VAA,
  VALIDATOR2_TENDERMINT_KEY,
} from "./consts.js";
import { signValidatorAddress } from "./utils/walletHelpers.js";

import fs from "fs";

const {
  getAddress,
  getWallet,
  getWormchainSigningClient,
  getWormholeQueryClient,
} = sdk;

let err: string | null = null;

//This test is split out into a global script file because it is not composeable with the other tests.

//This test requires a fresh tilt environment and cannot be repeated. It requires both the first & second validator to be running in tilt.

//TODO string encodings are all wrong, convert everything to base64
//TODO get a guardian set upgrade VAA for tiltnet that hands off to a new guardian key
//TODO figure out the valAddr of test wallet 2
async function fullBootstrapProcess() {
  try {
    console.log("Starting wormchain bootstrap test process");

    //construct the clients we will use for the test
    const queryClient = getWormholeQueryClient(NODE_URL, true);
    const wallet2Signer = await getWallet(TEST_WALLET_MNEMONIC_2);
    const wallet2Address = await getAddress(wallet2Signer);
    const signingClient = await getWormchainSigningClient(
      TENDERMINT_URL,
      wallet2Signer
    );

    //verify that guardian 1 is the only bonded validator
    const validators = await queryClient.staking.queryValidators({});
    expectEqual(
      "Initial bonded validators",
      validators.data.validators?.map((x) => x.operator_address),
      [GUARDIAN_VALIDATOR_VALADDR]
    );

    const Guardian1ValidatorAddress: string = getValidatorAddressBase64(
      "../../validators/first_validator/config/priv_validator_key.json"
    );
    const Guardian2ValidatorAddress: string = getValidatorAddressBase64(
      "../../validators/second_validator/config/priv_validator_key.json"
    );

    //verify that guardian 1 is producing blocks
    let latestBlock = await getLatestBlock();
    let validatorSet = latestBlock.block.last_commit.signatures;

    expectEqual(
      "Signers on first block",
      validatorSet.map((sig: any) => sig.validator_address),
      [Guardian1ValidatorAddress]
    );

    //verify that guardian 1 is registered to test wallet 1.
    let response = await queryClient.core.queryGuardianValidatorAll();
    const guardianValidators = response.data.guardianValidator || [];
    const tiltnetGuardian = {
      guardianKey: TILTNET_GUARDIAN_PUBKEY,
      validatorAddr: toBase64(fromValAddress(GUARDIAN_VALIDATOR_VALADDR)),
    };
    expectEqual(
      "Initial guardian validators",
      guardianValidators.map((x) => ({
        guardianKey: x.guardianKey,
        validatorAddr: x.validatorAddr,
      })),
      [tiltnetGuardian]
    );

    //verify that the latest guardian set is 1
    const response2 = await queryClient.core.queryLatestGuardianSetIndex();
    let index = response2.data.latestGuardianSetIndex;
    expectEqual('Initial "latest" guardian set', index, 0);

    //verify that the consensus guardian set is 1
    const response3 = await queryClient.core.queryConsensusGuardianSetIndex();
    index = response3.data.ConsensusGuardianSetIndex?.index;
    expectEqual("Initial consensus guardian set", index, 0);

    //verify that the only guardian public key is guardian public key 1.
    const response4 = await queryClient.core.queryGuardianSet(0);
    const guardianSet = response4.data || null;
    expectEqual("Guardian set 0", guardianSet.GuardianSet?.keys, [
      TILTNET_GUARDIAN_PUBKEY,
    ]);

    //process upgrade VAA
    const msg = signingClient.core.msgExecuteGovernanceVAA({
      signer: await getAddress(wallet2Signer),
      vaa: fromHex(UPGRADE_GUARDIAN_SET_VAA),
    });
    const receipt = await signingClient.signAndBroadcast(
      wallet2Address,
      [msg],
      getZeroFee()
    );
    expectTxSuccess("guardian set upgrade VAA", receipt);

    const guardianKey2base64 = Buffer.from(
      DEVNET_GUARDIAN2_PUBLIC_KEY,
      "hex"
    ).toString("base64");

    //verify only guardian 2 is in guardian set 1.
    const response7 = await queryClient.core.queryGuardianSet(1);
    const guardianSet7 = response7.data || null;
    expectEqual("Guardian set 1", guardianSet7.GuardianSet?.keys, [
      guardianKey2base64,
    ]);

    //verify latest guardian set is 1
    const response5 = await queryClient.core.queryLatestGuardianSetIndex();
    let index5 = response5.data.latestGuardianSetIndex || null;
    expectEqual("Latest guardian set after upgrade", index5, 1);

    //verify consensus guardian set is 0
    const response6 = await queryClient.core.queryConsensusGuardianSetIndex();
    let index6 = response6.data.ConsensusGuardianSetIndex?.index;
    expectEqual("Consensus guardian set after upgrade", index6, 0);

    //verify guardian 1 is still producing blocks
    let latestBlock2 = await getLatestBlock();
    let validatorSet2 = latestBlock2.block.last_commit.signatures;
    expectEqual(
      "Validators after upgrade",
      validatorSet2.map((sig: any) => sig.validator_address),
      [Guardian1ValidatorAddress]
    );

    //TODO attempt to register guardian2 to validator2, exception because validator2 is not bonded.

    //protobuf quackery
    //We should technically load the cosmos crypto ed15519 proto file here, but I'm going to spoof a type with the same field because our TS SDK doesn't have the proto files
    let AwesomeMessage = new Type("AwesomeMessage").add(
      new Field("key", 1, "bytes")
    );
    const pubkey = AwesomeMessage.encode({
      key: Buffer.from(VALIDATOR2_TENDERMINT_KEY, "base64"),
    }).finish();

    //bond validator2
    const bondMsg = signingClient.staking.msgCreateValidator({
      commission: { rate: "0", max_change_rate: "0", max_rate: "0" },
      description: {
        moniker: "secondValidator",
        details: "details",
        identity: "identity",
        security_contact: "contact",
        website: "https://.com",
      },
      delegator_address: TEST_WALLET_ADDRESS_2,
      min_self_delegation: "0",
      pubkey: {
        type_url: "/cosmos.crypto.ed25519.PubKey",
        value: pubkey,
      },
      validator_address: toValAddress(fromAccAddress(TEST_WALLET_ADDRESS_2)),
      value: { denom: "uworm", amount: "0" },
    });
    const createValidatorReceipt = await signingClient.signAndBroadcast(
      wallet2Address,
      [bondMsg],
      getZeroFee()
    );
    expectTxSuccess("second validator registration", createValidatorReceipt);

    //confirm validator2 is bonded
    const validators2 = await queryClient.staking.queryValidators({});
    expectEqual(
      "Second bonded validators",
      validators2.data.validators?.map((x) => x.operator_address).sort(),
      [
        GUARDIAN_VALIDATOR_VALADDR,
        toValAddress(fromAccAddress(TEST_WALLET_ADDRESS_2)),
      ].sort()
    );

    let latestBlock3 = await getLatestBlock();
    let validatorSet3 = latestBlock3.block.last_commit.signatures;
    expectEqual(
      "Signers after second validator bonded",
      validatorSet3.map((sig: any) => sig.validator_address),
      [Guardian1ValidatorAddress]
    );

    //attempt to register guardian2 to validator2
    //TODO what encoding for the guardian key & how to sign the validator address?
    const registerMsg = signingClient.core.msgRegisterAccountAsGuardian({
      guardianPubkey: { key: Buffer.from(DEVNET_GUARDIAN2_PUBLIC_KEY, "hex") },
      signer: TEST_WALLET_ADDRESS_2,
      signature: signValidatorAddress(
        toValAddress(fromAccAddress(TEST_WALLET_ADDRESS_2)),
        DEVNET_GUARDIAN2_PRIVATE_KEY
      ),
    });
    const registerMsgReceipe = await signingClient.signAndBroadcast(
      TEST_WALLET_ADDRESS_2,
      [registerMsg],
      getZeroFee()
    );
    expectTxSuccess("second guardian registration", registerMsgReceipe);

    //confirm validator2 is also now registered as a guardian validator
    let guardianValResponse =
      await queryClient.core.queryGuardianValidatorAll();
    const guardianValidators2 =
      guardianValResponse.data.guardianValidator || [];
    const secondGuardian = {
      guardianKey: Buffer.from(DEVNET_GUARDIAN2_PUBLIC_KEY, "hex").toString(
        "base64"
      ),
      validatorAddr: toBase64(fromAccAddress(TEST_WALLET_ADDRESS_2)),
    };
    expectEqual(
      "Updated guardian validators",
      guardianValidators2
        .map((x) => ({
          guardianKey: x.guardianKey,
          validatorAddr: x.validatorAddr,
        }))
        .sort(),
      [secondGuardian, tiltnetGuardian].sort()
    );

    //confirm consensus guardian set is now 2
    const conResponse = await queryClient.core.queryConsensusGuardianSetIndex();
    index = conResponse.data.ConsensusGuardianSetIndex?.index;
    expectEqual("Updated consensus guardian set", index, 1);

    //confirm blocks are only signed by validator2
    console.log("Waiting 4 seconds for latest block...");
    await new Promise((resolve) => setTimeout(resolve, 4000));
    latestBlock = await getLatestBlock();
    validatorSet = latestBlock.block.last_commit.signatures;
    expectEqual(
      "Signing validators on final block",
      validatorSet.map((sig: any) => sig.validator_address),
      [Guardian2ValidatorAddress]
    );

    console.log("Successfully completed bootstrap process.");
  } catch (e) {
    if (!err) {
      // if err is set, it means we ejected, so it's a test failure, not an ordinary exception
      console.error(e);
      console.log("Hit a critical error, process will terminate.");
    }
  } finally {
    if (err) {
      console.log(red("ERROR: ") + err);
    }
  }
}

//TODO figure out how to best move these stock cosmos queries into the SDK
async function getLatestBlock() {
  return await (
    await axios.get(NODE_URL + "/cosmos/base/tendermint/v1beta1/blocks/latest")
  ).data;
}

function eject(error: string) {
  err = error;
  throw new Error();
}

function fromHex(hexString: string) {
  return Buffer.from(hexString, "hex");
}

export function getZeroFee(): StdFee {
  return {
    amount: coins(0, WORM_DENOM),
    gas: "180000", // 180k",
  };
}

const wait = async () => {
  await fullBootstrapProcess();
};
wait();

function getValidatorAddressBase64(file: string): string {
  const validator_key_file = fs.readFileSync(file);
  return Buffer.from(
    JSON.parse(validator_key_file.toString()).address,
    "hex"
  ).toString("base64");
}

function equal<T>(actual: T, expected: T): boolean {
  if (Array.isArray(actual) && Array.isArray(expected)) {
    return (
      actual.length === expected.length &&
      actual.every((val, index) => equal(val, expected[index]))
    );
  } else if (typeof actual === "object" && typeof expected === "object") {
    return JSON.stringify(actual) === JSON.stringify(expected);
  } else {
    return actual === expected;
  }
}

function expectEqual<T>(msg: string, actual: T, expected: T): void {
  if (!equal(actual, expected)) {
    eject(
      msg +
        ":\nExpected: " +
        green(stringify(expected)) +
        ", got: " +
        red(stringify(actual))
    );
  } else {
    console.log(msg + ": " + green("PASS"));
  }
}

function expectTxSuccess(msg: string, receipt: DeliverTxResponse): void {
  if (receipt.code !== 0) {
    eject(
      "Transaction " +
        msg +
        " failed. Transaction hash: " +
        red(receipt.transactionHash)
    );
  }

  console.log(
    "Transaction " +
      msg +
      ": " +
      green("PASS") +
      " (" +
      receipt.transactionHash +
      ")"
  );
}

function stringify<T>(x: T): string {
  if (Array.isArray(x)) {
    return "[" + x.map((x) => stringify(x)) + "]";
  } else if (typeof x === "object") {
    return JSON.stringify(x);
  } else {
    return "" + x;
  }
}

function red(str: string): string {
  return "\x1b[31m" + str + "\x1b[0m";
}

function green(str: string): string {
  return "\x1b[32m" + str + "\x1b[0m";
}

export default {};
