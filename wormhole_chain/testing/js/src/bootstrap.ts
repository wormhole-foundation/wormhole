import { coins } from "@cosmjs/proto-signing";
import { StdFee } from "@cosmjs/stargate";
import axios from "axios";
//@ts-ignore
import pkg from "protobufjs";
const { Field, Type } = pkg;
import * as sdk from "wormhole-chain-sdk";
import {
  fromAccAddress,
  fromValAddress,
  toBase64,
  toValAddress,
} from "wormhole-chain-sdk";
import {
  DEVNET_GUARDIAN2_PRIVATE_KEY,
  DEVNET_GUARDIAN2_PUBLIC_KEY,
  GUARDIAN_VALIDATOR2_VALADDR,
  GUARDIAN_VALIDATOR_VALADDR,
  HOLE_DENOM,
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

const {
  getAddress,
  getWallet,
  getWormchainSigningClient,
  getWormholeQueryClient,
} = sdk;

let done = false;
let err: string | null = null;

//This test is split out into a global script file because it is not composeable with the other tests.

//This test requires a fresh tilt environment and cannot be repeated. It requires both the first & second validator to be running in tilt.

//TODO string encodings are all wrong, convert everything to base64
//TODO get a guardian set upgrade VAA for tiltnet that hands off to a new guardian key
//TODO figure out the valAddr of test wallet 2
async function fullBootstrapProcess() {
  try {
    console.log("Starting wormchain bootstrap test process");
    newline();

    //construct the clients we will use for the test
    console.log("instantiating query client.");
    const queryClient = getWormholeQueryClient(NODE_URL, true);
    console.log("instantiating wallet2.");
    const wallet2Signer = await getWallet(TEST_WALLET_MNEMONIC_2);
    console.log("get wallet address.");
    const wallet2Address = await getAddress(wallet2Signer);
    console.log("instantiating signing client.");
    const signingClient = await getWormchainSigningClient(
      TENDERMINT_URL,
      wallet2Signer
    );

    //verify that guardian 1 is the only bonded validator
    console.log("Logging initial bonded validators:");
    const validators = await queryClient.staking.queryValidators({});
    validators.data.validators?.forEach((item) => {
      console.log("Validator: " + item.operator_address);
    });
    if (!(validators.data.validators?.length === 1)) {
      eject(
        "Unexpected amount of initial validators: " +
          validators.data.validators?.length
      );
    }
    if (
      !validators.data.validators?.find(
        (x) => x.operator_address === GUARDIAN_VALIDATOR_VALADDR
      )
    ) {
      eject(
        "Failed to find first_validator in the initial set of bonded validators."
      );
    }
    newline();

    //TODO figure out how to calculate this and why the evidence field on the latest block is always an empty array.
    const Guardian1ValidatorAddress = "h/2mNkBThr9oY0FEKsyf3s+aI5Y=";

    //verify that guardian 1 is producing blocks
    console.log("Logging initial block signatures: ");
    let latestBlock = await getLatestBlock();
    let validatorSet = latestBlock.block.last_commit.signatures;
    validatorSet.forEach((sig: any) => {
      console.log("Signature: " + sig.validator_address);
    });
    if (!(validatorSet && validatorSet.length === 1)) {
      eject(
        "Unexpected length of signing validators on initial block: " +
          validatorSet?.length
      );
    }
    if (
      !validatorSet.find(
        (sig: any) => sig.validator_address === Guardian1ValidatorAddress
      )
    ) {
      eject(
        "Failed to find first_validator in the signature set of the initial block"
      );
    }
    newline();

    //verify that guardian 1 is registered to test wallet 1.
    console.log("Logging initial guardian validators: ");
    let response = await queryClient.core.queryGuardianValidatorAll();
    const guardianValidators = response.data.guardianValidator || [];
    guardianValidators.forEach((item) => {
      console.log(
        "guardianKey: " + item.guardianKey + ", valAddr: " + item.validatorAddr
      );
    });
    if (!(guardianValidators.length === 1)) {
      eject(
        "Unexpected length of initial guardian validators: " +
          guardianValidators.length
      );
    }
    if (
      !guardianValidators.find(
        (x) =>
          x.guardianKey === TILTNET_GUARDIAN_PUBKEY &&
          x.validatorAddr ===
            toBase64(fromValAddress(GUARDIAN_VALIDATOR_VALADDR))
      )
    ) {
      eject(
        "Failed to find first_validator registered against the tilt guardian pubkey in the initial guardian validator set."
      );
    }
    newline();

    //verify that the latest guardian set is 1
    console.log("Pulling initial latest guardian set");
    const response2 = await queryClient.core.queryLatestGuardianSetIndex();
    let index = response2.data.latestGuardianSetIndex;
    console.log("Current guardian set index: " + index);
    if (index !== 0) {
      eject("Latest Guardian set index was not 0 at initialization.");
    }
    newline();

    //verify that the consensus guardian set is 1
    console.log("Pulling initial consensus guardian set");
    const response3 = await queryClient.core.queryConsensusGuardianSetIndex();
    index = response3.data.ConsensusGuardianSetIndex?.index;
    console.log("Current guardian set index: " + index);
    if (index !== 0) {
      eject("Latest Guardian set index was not 0 at initialization.");
    }
    newline();

    //verify that the only guardian public key is guardian public key 1.
    console.log("Pulling Guardian Set 0");
    const response4 = await queryClient.core.queryGuardianSet(0);
    const guardianSet = response4.data || null;
    console.log("Guardian set obj: " + JSON.stringify(guardianSet));
    if (guardianSet.GuardianSet?.keys?.length !== 1) {
      eject("Unexpected length of guardian set 1.");
    }
    if (
      !guardianSet.GuardianSet?.keys?.find((x) => {
        return x === TILTNET_GUARDIAN_PUBKEY;
      })
    ) {
      eject("Failed to find the tiltnet guardian in guardian set 0.");
    }
    newline();

    //bridge in uhole tokens to wallet 2
    console.log("Bridging in tokens to wallet 2");
    const transferMsg1 = signingClient.tokenbridge.msgExecuteVAA({
      creator: wallet2Address,
      vaa: fromHex(TEST_TRANSFER_VAA_1),
    });
    const transferreceipt = await signingClient.signAndBroadcast(
      wallet2Address,
      [transferMsg1],
      getZeroFee()
    );
    console.log("transaction hash: " + transferreceipt.transactionHash);
    if (transferreceipt.code !== 0) {
      eject("Initial bridge redeem token VAA transaction failed.");
    }
    newline();

    //process upgrade VAA
    console.log("Submitting upgrade guardian set VAA");
    const msg = signingClient.core.msgExecuteGovernanceVAA({
      signer: await getAddress(wallet2Signer),
      vaa: fromHex(UPGRADE_GUARDIAN_SET_VAA),
    });
    const receipt = await signingClient.signAndBroadcast(
      wallet2Address,
      [msg],
      getZeroFee()
    );
    console.log("transaction hash: " + receipt.transactionHash);
    if (receipt.code !== 0) {
      eject("Failed to upgrade the guardian set.");
    }
    newline();

    const guardianKey2base64 = Buffer.from(
      DEVNET_GUARDIAN2_PUBLIC_KEY,
      "hex"
    ).toString("base64");
    console.log("guardian 2 base 64", guardianKey2base64);

    //verify only guardian 2 is in guardian set 1.
    console.log("Pulling Guardian Set 1");
    const response7 = await queryClient.core.queryGuardianSet(1);
    const guardianSet7 = response7.data || null;
    console.log("Guardian set obj: " + JSON.stringify(guardianSet7));
    if (guardianSet7.GuardianSet?.keys?.length !== 1) {
      eject("Unexpected length of guardian set 1.");
    }
    if (
      !guardianSet7.GuardianSet?.keys?.find((x) => {
        return x === guardianKey2base64;
      })
    ) {
      eject("Failed to find the tiltnet guardian 2 in guardian set 1.");
    }
    newline();

    //verify latest guardian set is 1
    console.log("Pulling latest guardian set after upgrade");
    const response5 = await queryClient.core.queryLatestGuardianSetIndex();
    let index5 = response5.data.latestGuardianSetIndex || null;
    console.log("Current latest guardian set index: " + index5);
    if (index5 !== 1) {
      eject("Latest Guardian set index was not 1 after upgrade.");
    }
    newline();

    //verify consensus guardian set is 0
    console.log("Pulling consensus guardian set after upgrade");
    const response6 = await queryClient.core.queryConsensusGuardianSetIndex();
    let index6 = response6.data.ConsensusGuardianSetIndex?.index;
    console.log("Current consensus guardian set index: " + index6);
    if (index6 !== 0) {
      eject("Consensus Guardian set index was not 0 after upgrade.");
    }
    newline();

    //verify guardian 1 is still producing blocks
    console.log("Logging block signatures after upgrade: ");
    let latestBlock2 = await getLatestBlock();
    let validatorSet2 = latestBlock2.block.last_commit.signatures;
    validatorSet2.forEach((sig: any) => {
      console.log("Signature: " + sig.validator_address);
    });
    if (!(validatorSet2 && validatorSet2.length === 1)) {
      eject(
        "Unexpected length of signing validators on initial block: " +
          validatorSet2?.length
      );
    }
    if (
      !validatorSet2.find(
        (sig: any) => sig.validator_address === Guardian1ValidatorAddress
      )
    ) {
      eject(
        "Failed to find first_validator in the signature set after the guardian upgrade"
      );
    }
    newline();

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
    console.log("Attempting to bond the second validator.");
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
      value: { denom: "uhole", amount: "0" },
    });
    const createValidatorReceipt = await signingClient.signAndBroadcast(
      wallet2Address,
      [bondMsg],
      getZeroFee()
    );
    console.log("transaction hash: " + createValidatorReceipt.transactionHash);
    if (createValidatorReceipt.code !== 0) {
      eject("Registering second validator failed.");
    }
    newline();

    //confirm validator2 is bonded
    console.log("Logging validators after second bond");
    const validators2 = await queryClient.staking.queryValidators({});
    validators2.data.validators?.forEach((item) => {
      console.log("Validator: " + item.operator_address);
    });
    if (!(validators2.data.validators?.length === 2)) {
      eject(
        "Unexpected amount of second validators: " +
          validators2.data.validators?.length
      );
    }
    if (
      !validators2.data.validators?.find((x) => {
        return x.operator_address === GUARDIAN_VALIDATOR_VALADDR;
      })
    ) {
      eject(
        "Failed to find first_validator in the second set of bonded validators."
      );
    }
    if (
      !validators2.data.validators?.find(
        (x) =>
          x.operator_address ===
          toValAddress(fromAccAddress(TEST_WALLET_ADDRESS_2))
      )
    ) {
      eject(
        "Failed to find second_validator in the second set of bonded validators."
      );
    }
    newline();

    console.log("Logging block signatures after second validator was bonded: ");
    let latestBlock3 = await getLatestBlock();
    let validatorSet3 = latestBlock3.block.last_commit.signatures;
    validatorSet3.forEach((sig: any) => {
      console.log("Signature: " + sig.validator_address);
    });
    if (!(validatorSet3 && validatorSet3.length === 1)) {
      eject(
        "Unexpected length of signing validators on initial block: " +
          validatorSet3?.length
      );
    }
    if (
      !validatorSet3.find(
        (sig: any) => sig.validator_address === Guardian1ValidatorAddress
      )
    ) {
      eject(
        "Failed to find first_validator in the signature set after the guardian upgrade"
      );
    }
    newline();

    //attempt to register guardian2 to validator2
    //TODO what encoding for the guardian key & how to sign the validator address?
    console.log(
      "Attempting to register the second validator as a guardian validator."
    );
    const registerMsg = signingClient.core.msgRegisterAccountAsGuardian({
      addressBech32: toValAddress(fromAccAddress(TEST_WALLET_ADDRESS_2)),
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
    console.log("transaction hash: " + registerMsgReceipe.transactionHash);
    if (registerMsgReceipe.code !== 0) {
      eject("Registering second validator as guardian validator failed.");
    }
    newline();

    //confirm validator2 is also now registered as a guardian validator
    console.log("Logging guardian validators after the second register: ");
    let guardianValResponse =
      await queryClient.core.queryGuardianValidatorAll();
    const guardianValidators2 =
      guardianValResponse.data.guardianValidator || [];
    guardianValidators2.forEach((item) => {
      console.log(
        "guardianKey: " + item.guardianKey + ", valAddr: " + item.validatorAddr
      );
    });
    if (!(guardianValidators2.length === 2)) {
      eject(
        "Unexpected length of updated guardian validators: " +
          guardianValidators2.length
      );
    }
    if (
      !guardianValidators2.find(
        (x) =>
          x.guardianKey === TILTNET_GUARDIAN_PUBKEY &&
          x.validatorAddr === GUARDIAN_VALIDATOR_VALADDR
      )
    ) {
      eject(
        "Failed to find first_validator registered against the tilt guardian pubkey in the updated guardian validator set."
      );
    }
    if (
      !guardianValidators2.find(
        (x) =>
          x.guardianKey === DEVNET_GUARDIAN2_PUBLIC_KEY &&
          x.validatorAddr === GUARDIAN_VALIDATOR2_VALADDR
      )
    ) {
      eject(
        "Failed to find second_validator registered against the tilt guardian pubkey in the updated guardian validator set."
      );
    }
    newline();

    //confirm consensus guardian set is now 2
    console.log("Pulling updated consensus guardian set");
    const conResponse = await queryClient.core.queryConsensusGuardianSetIndex();
    index = conResponse.data.ConsensusGuardianSetIndex?.index;
    console.log("Current guardian set index: " + index);
    if (index !== 0) {
      eject("Latest Guardian set index was not 1 after update.");
    }
    newline();

    //confirm blocks are only signed by validator2
    console.log("Logging final block signatures: ");
    latestBlock = await getLatestBlock();
    validatorSet = latestBlock.block.last_commit.signatures;
    validatorSet.forEach((sig: any) => {
      console.log("Signature: " + sig.validator_address);
    });
    if (!(validatorSet && validatorSet.length === 1)) {
      eject(
        "Unexpected length of signing validators on final block: " +
          validatorSet?.length
      );
    }
    if (
      !validatorSet.find(
        (sig: any) =>
          sig.validator_address ===
          toBase64(fromValAddress(GUARDIAN_VALIDATOR2_VALADDR))
      )
    ) {
      eject(
        "Failed to find second_validator in the signature set of the final block"
      );
    }
    newline();

    console.log("Successfully completed bootstrap process.");
  } catch (e) {
    console.error(e);
    console.log("Hit a critical error, process will terminate.");
    done = true;
  } finally {
    done = true;
    if (err) {
      newline();
      console.log("ERROR: " + err);
    }
  }
}

//TODO figure out how to best move these stock cosmos queries into the SDK
async function getLatestBlock() {
  return await (
    await axios.get(NODE_URL + "/cosmos/base/tendermint/v1beta1/blocks/latest")
  ).data;
}

function newline() {
  console.log("");
}

function eject(error: string) {
  done = true;
  err = error;
  throw new Error();
}

function fromHex(hexString: string) {
  return Buffer.from(hexString, "hex");
}

export function getZeroFee(): StdFee {
  return {
    amount: coins(0, HOLE_DENOM),
    gas: "180000", // 180k",
  };
}

const wait = async () => {
  await fullBootstrapProcess();
};
wait();

const stringToBase64 = (item: string) => Buffer.from(item).toString("base64");

export default {};
