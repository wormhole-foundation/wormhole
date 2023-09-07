import { CHAIN_ID_ETH, tryNativeToUint8Array } from "@certusone/wormhole-sdk";
import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
} from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";

describe("Token Bridge -- Instruction: Secure Registered Emitter", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  // Test variables.
  const localVariables = new Map<string, any>();

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    it("Invoke `secure_registered_emitter`", async () => {
      const foreignChain = CHAIN_ID_ETH;
      const foreignEmitter = Array.from(
        tryNativeToUint8Array(ETHEREUM_TOKEN_BRIDGE_ADDRESS, CHAIN_ID_ETH)
      );
      const legacyRegistered = tokenBridge.RegisteredEmitter.address(
        program.programId,
        foreignChain,
        foreignEmitter
      );

      const ix = await tokenBridge.secureRegisteredEmitterIx(program, {
        payer: payer.publicKey,
        legacyRegisteredEmitter: legacyRegistered,
      });
      await expectIxOk(connection, [ix], [payer]);

      const registered = tokenBridge.RegisteredEmitter.address(program.programId, CHAIN_ID_ETH);
      expect(registered.toString()).not.equal(legacyRegistered.toString());

      const registeredData = await tokenBridge.RegisteredEmitter.fromAccountAddress(
        connection,
        registered
      );
      const legacyRegisteredData = await tokenBridge.RegisteredEmitter.fromAccountAddress(
        connection,
        legacyRegistered
      );
      expectDeepEqual(registeredData, legacyRegisteredData);

      // Save for later.
      localVariables.set("legacyRegistered", legacyRegistered);
    });

    it("Cannot Invoke `secure_registered_emitter` with Same Registered Emitter", async () => {
      const legacyRegistered = localVariables.get("legacyRegistered") as anchor.web3.PublicKey;

      const ix = await tokenBridge.secureRegisteredEmitterIx(program, {
        payer: payer.publicKey,
        legacyRegisteredEmitter: legacyRegistered,
      });
      await expectIxErr(connection, [ix], [payer], "already in use");
    });
  });
});
