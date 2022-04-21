import axios from "axios";
import { OperationCanceledException } from "typescript";
import * as sdk from "wormhole-chain-sdk";
import {
  NODE_URL,
  TENDERMINT_URL,
  TEST_WALLET_MNEMONIC_2,
  TILTNET_GUARDIAN_PRIVATE_KEY,
  TILTNET_GUARDIAN_PUBKEY,
  GUARDIAN2_UPGRADE_VAA,
  TEST_TRANSFER_VAA_1,
  DEVNET_GUARDIAN_PUBLIC_KEY,
  DEVNET_GUARDIAN2_PUBLIC_KEY,
  GUARDIAN_VALIDATOR2_VALADDR,
  VALIDATOR2_TENDERMINT_KEY,
  TEST_WALLET_ADDRESS_2,
  GUARDIAN_VALIDATOR_VALADDR,
  DEVNET_GUARDIAN2_PRIVATE_KEY,
  HOLE_DENOM,
  TEST_WALLET_MNEMONIC_1,
} from "./consts.js";
import { StdFee } from "@cosmjs/stargate";
import { coins } from "@cosmjs/proto-signing";

const {
  getAddress,
  getStargateQueryClient,
  getWallet,
  getWormchainSigningClient,
  getWormholeQueryClient,
} = sdk;
//@ts-ignore
import * as elliptic from "elliptic";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";

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
    console.log("instantiating stargate client.");
    const stargateClient = await getStargateQueryClient(TENDERMINT_URL);

    //verify that guardian 1 is the only bonded validator
    console.log("Logging initial bonded validators:");
    const validators = await stargateClient.staking.validators(
      "BOND_STATUS_BONDED"
    );
    validators.validators.forEach((item) => {
      console.log("Validator: " + item.operatorAddress);
    });
    if (!(validators.validators.length === 1)) {
      eject(
        "Unexpected amount of initial validators: " +
          validators.validators.length
      );
    }
    if (
      !validators.validators.find(
        (x) => x.operatorAddress === GUARDIAN_VALIDATOR_VALADDR
      )
    ) {
      eject(
        "Failed to find first_validator in the initial set of bonded validators."
      );
    }
    newline();

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
        (sig: any) => sig.validator_address === GUARDIAN_VALIDATOR_VALADDR
      )
    ) {
      eject(
        "Failed to find first_validator in the signature set of the initial block"
      );
    }
    newline();

    //verify that guardian 1 is registered to test wallet 1.
    console.log("Logging initial guardian validators: ");
    let response = await queryClient.coreClient.queryGuardianValidatorAll();
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
          x.validatorAddr === GUARDIAN_VALIDATOR_VALADDR
      )
    ) {
      eject(
        "Failed to find first_validator registered against the tilt guardian pubkey in the initial guardian validator set."
      );
    }
    newline();

    //verify that the latest guardian set is 1
    console.log("Pulling initial latest guardian set");
    const response2 =
      await queryClient.coreClient.queryLatestGuardianSetIndex();
    let index = response2.data.latestGuardianSetIndex || null;
    console.log("Current guardian set index: " + index);
    if (index !== 0) {
      eject("Latest Guardian set index was not 0 at initialization.");
    }
    newline();

    //verify that the consensus guardian set is 1
    console.log("Pulling initial consensus guardian set");
    const response3 =
      await queryClient.coreClient.queryConsensusGuardianSetIndex();
    index = response3.data.ConsensusGuardianSetIndex?.index || null;
    console.log("Current guardian set index: " + index);
    if (index !== 0) {
      eject("Latest Guardian set index was not 0 at initialization.");
    }
    newline();

    //verify that the only guardian public key is guardian public key 1.
    console.log("Pulling Guardian Set 0");
    const response4 = await queryClient.coreClient.queryGuardianSet(0);
    const guardianSet = response4.data || null;
    console.log("Guardian set obj: " + guardianSet);
    if (guardianSet.GuardianSet?.keys?.length !== 1) {
      eject("Unexpected length of guardian set 1.");
    }
    if (
      !guardianSet.GuardianSet?.keys?.find((x) => {
        x === TILTNET_GUARDIAN_PUBKEY;
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
      vaa: Buffer.from(GUARDIAN2_UPGRADE_VAA, "hex"),
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

    //verify only guardian 2 is in guardian set 1.
    console.log("Pulling Guardian Set 1");
    const response7 = await queryClient.coreClient.queryGuardianSet(1);
    const guardianSet7 = response7.data || null;
    console.log("Guardian set obj: " + guardianSet7);
    if (guardianSet7.GuardianSet?.keys?.length !== 1) {
      eject("Unexpected length of guardian set 1.");
    }
    if (
      !guardianSet7.GuardianSet?.keys?.find((x) => {
        x === DEVNET_GUARDIAN2_PUBLIC_KEY;
      })
    ) {
      eject("Failed to find the tiltnet guardian 2 in guardian set 1.");
    }
    newline();

    //verify latest guardian set is 1
    console.log("Pulling latest guardian set after upgrade");
    const response5 =
      await queryClient.coreClient.queryLatestGuardianSetIndex();
    let index5 = response5.data.latestGuardianSetIndex || null;
    console.log("Current latest guardian set index: " + index5);
    if (index5 !== 1) {
      eject("Latest Guardian set index was not 1 after upgrade.");
    }
    newline();

    //verify consensus guardian set is 0
    console.log("Pulling consensus guardian set after upgrade");
    const response6 =
      await queryClient.coreClient.queryConsensusGuardianSetIndex();
    let index6 = response6.data.ConsensusGuardianSetIndex?.index || null;
    console.log("Current consensus guardian set index: " + index6);
    if (index5 !== 1) {
      eject("Latest Guardian set index was not 1 after upgrade.");
    }
    newline();

    //verify guardian 1 is still producing blocks
    console.log("Logging block signatures after upgrade: ");
    let latestBlock2 = await getLatestBlock();
    let validatorSet2 = latestBlock.block.last_commit.signatures;
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
        (sig: any) => sig.validator_address === GUARDIAN_VALIDATOR2_VALADDR
      )
    ) {
      eject(
        "Failed to find first_validator in the signature set after the guardian upgrade"
      );
    }
    newline();

    //TODO attempt to register guardian2 to validator2, exception because validator2 is not bonded.

    //bond validator2
    const bondMsg = signingClient.staking.msgCreateValidator({
      commission: { rate: "0", maxChangeRate: "0", maxRate: "0" },
      description: {
        moniker: "secondValidator",
        details: "details",
        identity: "identity",
        securityContact: "contact",
        website: "https://.com",
      },
      delegatorAddress: TEST_WALLET_ADDRESS_2,
      minSelfDelegation: "0",
      pubkey: {
        typeUrl: "not sure",
        value: Buffer.from(VALIDATOR2_TENDERMINT_KEY, "base64"), //TODO this is wrong
      },
      validatorAddress: GUARDIAN_VALIDATOR2_VALADDR,
      value: { denom: "uhole", amount: "0" },
    });
    const createValidatorReceipt = await signingClient.signAndBroadcast(
      wallet2Address,
      [bondMsg],
      getZeroFee()
    );
    console.log("transaction hash: " + transferreceipt.transactionHash);
    if (createValidatorReceipt.code !== 0) {
      eject("Registering second validator failed.");
    }
    newline();

    //confirm validator2 is bonded
    console.log("Logging validators after second bond");
    const validators2 = await stargateClient.staking.validators(
      "BOND_STATUS_BONDED"
    );
    validators2.validators.forEach((item) => {
      console.log("Validator: " + item.operatorAddress);
    });
    if (!(validators2.validators.length === 2)) {
      eject(
        "Unexpected amount of second validators: " +
          validators2.validators.length
      );
    }
    if (
      !validators2.validators.find(
        (x) => x.operatorAddress === GUARDIAN_VALIDATOR_VALADDR
      )
    ) {
      eject(
        "Failed to find first_validator in the second set of bonded validators."
      );
    }
    if (
      !validators2.validators.find(
        (x) => x.operatorAddress === GUARDIAN_VALIDATOR2_VALADDR
      )
    ) {
      eject(
        "Failed to find first_validator in the second set of bonded validators."
      );
    }
    newline();

    //attempt to register guardian2 to validator2
    //TODO what encoding for the guardian key & how to sign the validator address?
    const registerMsg = signingClient.core.msgRegisterAccountAsGuardian({
      addressBech32: TEST_WALLET_ADDRESS_2,
      guardianPubkey: { key: Buffer.from(DEVNET_GUARDIAN2_PUBLIC_KEY, "hex") },
      signer: TEST_WALLET_ADDRESS_2,
      signature: signValidatorAddress(
        GUARDIAN_VALIDATOR2_VALADDR,
        DEVNET_GUARDIAN2_PRIVATE_KEY
      ),
    });
    const registerMsgReceipe = await signingClient.signAndBroadcast(
      wallet2Address,
      [registerMsg],
      getZeroFee()
    );
    console.log("transaction hash: " + transferreceipt.transactionHash);
    if (registerMsgReceipe.code !== 0) {
      eject("Registering second validator as guardian validator failed.");
    }
    newline();

    //confirm validator2 is also now registered as a guardian validator
    console.log("Logging guardian validators after the second register: ");
    let guardianValResponse =
      await queryClient.coreClient.queryGuardianValidatorAll();
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
    const conResponse =
      await queryClient.coreClient.queryConsensusGuardianSetIndex();
    index = conResponse.data.ConsensusGuardianSetIndex?.index || null;
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
        (sig: any) => sig.validator_address === GUARDIAN_VALIDATOR2_VALADDR
      )
    ) {
      eject(
        "Failed to find second_validator in the signature set of the final block"
      );
    }
    newline();
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

fullBootstrapProcess();
while (!done) {}

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
  return Buffer.from(GUARDIAN2_UPGRADE_VAA, "hex");
}

export function signValidatorAddress(valAddr: string, privKey: string) {
  const ec = new elliptic.ec("secp256k1");
  const key = ec.keyFromPrivate(privKey);
  const signature = key.sign(valAddr, { canonical: true });
  return signature as Uint8Array; //TODO determine if this is correct
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

export default {};
