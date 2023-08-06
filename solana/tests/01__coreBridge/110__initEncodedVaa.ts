import * as anchor from "@coral-xyz/anchor";
import { createAccountIx, expectDeepEqual, expectIxErr, expectIxOk } from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Init Encoded VAA", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    it("Cannot Invoke `init_encoded_vaa` without Created Account", async () => {
      const encodedVaa = anchor.web3.Keypair.generate();

      const initIx = await coreBridge.initEncodedVaaIx(program, {
        writeAuthority: payer.publicKey,
        encodedVaa: encodedVaa.publicKey,
      });
      await expectIxErr(connection, [initIx], [payer], "ConstraintOwner");
    });

    it("Cannot Invoke `init_encoded_vaa` with Nonsensical Account Size", async () => {
      const encodedVaa = anchor.web3.Keypair.generate();

      const createIx = await createAccountIx(
        program.provider.connection,
        program.programId,
        payer,
        encodedVaa,
        45 // one less than the minimum
      );

      const initIx = await coreBridge.initEncodedVaaIx(program, {
        writeAuthority: payer.publicKey,
        encodedVaa: encodedVaa.publicKey,
      });
      await expectIxErr(
        connection,
        [createIx, initIx],
        [payer, encodedVaa],
        "InvalidCreatedAccountSize"
      );
    });

    it("Cannot Invoke `init_encoded_vaa` with Expected VAA Size == 0", async () => {
      const { encodedVaa, instructions } = await createIxs(program, payer, 0);

      await expectIxErr(connection, instructions, [payer, encodedVaa], "InvalidCreatedAccountSize");
    });
  });

  describe("Ok", () => {
    const vaaSizes = [1, 10 * 1_024, 10 * 1_024_000];

    for (const vaaSize of vaaSizes) {
      it(`Invoke \`init_encoded_vaa\` with VAA Size == ${vaaSize}`, async () => {
        const { encodedVaa, instructions } = await createIxs(program, payer, vaaSize);

        await expectIxOk(connection, instructions, [payer, encodedVaa]);

        const encodedVaaData = await coreBridge.EncodedVaa.fetch(program, encodedVaa.publicKey);
        expectDeepEqual(encodedVaaData, {
          status: coreBridge.ProcessingStatus.Writing,
          writeAuthority: payer.publicKey,
          version: coreBridge.VaaVersion.Unset,
          buf: Buffer.alloc(vaaSize),
        });

        // Only pick one for the next test.
        if (vaaSize == 1) {
          localVariables.set("encodedVaa", encodedVaa);
        }
      });
    }

    it("Cannot Invoke `init_encoded_vaa` with Same Encoded VAA", async () => {
      const encodedVaa = localVariables.get("encodedVaa") as anchor.web3.Keypair;

      const initIx = await coreBridge.initEncodedVaaIx(program, {
        writeAuthority: payer.publicKey,
        encodedVaa: encodedVaa.publicKey,
      });
      await expectIxErr(connection, [initIx], [payer], "AccountNotZeroed");
    });
  });
});

async function prepareEncodedVaa(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  vaaSize: number
) {
  const encodedVaa = anchor.web3.Keypair.generate();
  const createIx = await createAccountIx(
    program.provider.connection,
    program.programId,
    payer,
    encodedVaa,
    46 + vaaSize
  );

  return {
    encodedVaa,
    createIx,
  };
}

async function createIxs(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  vaaSize: number
) {
  const { encodedVaa, createIx } = await prepareEncodedVaa(program, payer, vaaSize);

  const initIx = await coreBridge.initEncodedVaaIx(program, {
    writeAuthority: payer.publicKey,
    encodedVaa: encodedVaa.publicKey,
  });

  return { encodedVaa, instructions: [createIx, initIx] };
}
