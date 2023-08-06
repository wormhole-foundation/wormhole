import {
  GovernanceEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import { createPostSignedVaaTransactions } from "@certusone/wormhole-sdk/lib/cjs/solana/sendAndConfirmPostVaa";
import * as coreBridgeSDK from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
import { web3 } from "@coral-xyz/anchor";
import { coreBridge } from "wormhole-solana-sdk";
import {
  CORE_BRIDGE_PROGRAM_ID,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  LOCALHOST,
  airdrop,
  artifactsPath,
  expectIxOk,
  loadProgramBpf,
  verifySignaturesAndPostVaa,
} from "../helpers";

const GUARDIAN_SET_INDEX = 3;

describe("Core Bridge: New Implementation", () => {
  const connection = new web3.Connection(LOCALHOST, "processed");

  const governance = new GovernanceEmitter(
    GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
    2_100_000
  );
  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  const payerSigner = web3.Keypair.generate();
  const payer = payerSigner.publicKey;

  const localVariables = new Map<string, any>();

  before("Airdrop Payer", async () => {
    await airdrop(connection, payer);
  });

  describe("Ok", async () => {
    it("Load New Implementation", async () => {
      const artifactPath = `${artifactsPath()}/solana_wormhole_core_bridge.so`;
      const bufferAuthority = coreBridge.upgradeAuthority(
        CORE_BRIDGE_PROGRAM_ID
      );
      const implementation = loadProgramBpf(
        payerSigner,
        artifactPath,
        bufferAuthority
      );

      localVariables.set("implementation", implementation);
    });

    it("Invoke `upgrade_contract`", async () => {
      const newContract: web3.PublicKey = localVariables.get("implementation")!;

      const timestamp = 12345678;
      const chain = 1;
      const published = governance.publishWormholeUpgradeContract(
        timestamp,
        chain,
        newContract.toString()
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Verify and Post
      await verifySignaturesAndPostVaa(connection, payerSigner, signedVaa);

      // Upgrade.
      await expectIxOk(
        connection,
        [
          coreBridgeSDK.createUpgradeContractInstruction(
            CORE_BRIDGE_PROGRAM_ID,
            payer,
            signedVaa
          ),
        ],
        [payerSigner]
      );
    });
  });
});
