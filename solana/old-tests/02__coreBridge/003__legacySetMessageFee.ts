import { BN, web3 } from "@coral-xyz/anchor";
import {
  CORE_BRIDGE_PROGRAM_ID,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  LOCALHOST,
  airdrop,
  expectIxErr,
  expectIxOk,
  invokeVerifySignaturesAndPostVaa,
} from "../helpers";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as coreBridgeSDK from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";

const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 2_003_000;

describe("Core Bridge: Legacy Set Message Fee (Governance)", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const governance = new GovernanceEmitter(
    GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
    GOVERNANCE_SEQUENCE
  );
  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  const localVariables = new Map<string, any>();

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Ok", async () => {
    it("Invoke `set_message_fee`", async () => {
      const amount = new BN(6969);

      const timestamp = 12345678;
      const chain = 1;
      const published = governance.publishWormholeSetMessageFee(
        timestamp,
        chain,
        BigInt(amount.toString())
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Verify and Post
      await invokeVerifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

      // Set message fee.
      await expectIxOk(
        connection,
        [coreBridgeSDK.createSetFeesInstruction(CORE_BRIDGE_PROGRAM_ID, payer, signedVaa)],
        [payerSigner]
      );

      // TODO: Check bridge program data to see if message fee was set correctly.

      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `set_message_fee` with Same VAA", async () => {
      const signedVaa: Buffer = localVariables.get("signedVaa")!;

      await expectIxErr(
        connection,
        [coreBridgeSDK.createSetFeesInstruction(CORE_BRIDGE_PROGRAM_ID, payer, signedVaa)],
        [payerSigner],
        "AlreadyInitialized"
      );
    });
  });
});
