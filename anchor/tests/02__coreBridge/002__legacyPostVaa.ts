import {
  MockEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import { web3 } from "@coral-xyz/anchor";
import {
  GUARDIAN_KEYS,
  LOCALHOST,
  airdrop,
  verifySignaturesAndPostVaa,
} from "../helpers";

const GUARDIAN_SET_INDEX = 0;

describe("Core Bridge: Legacy Verify Signatures and Post VAA", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  const foreignEmitter = new MockEmitter(
    "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
    2,
    2_002_000
  );

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  const localVariables = new Map<string, any>();

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Ok", async () => {
    it("Invoke `legacy_verify_signatures` and `legacy_post_vaa`", async () => {
      const timestamp = 12345678;
      const nonce = 420;
      const payload = Buffer.from("Someone set us up the bomb.");
      const consistencyLevel = 1;
      const published = foreignEmitter.publishMessage(
        nonce,
        payload,
        consistencyLevel,
        timestamp
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Verify and Post
      await verifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

      // TODO: Check Posted VAA account.
    });
  });
});
