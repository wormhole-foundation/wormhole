import { parseVaa } from "@certusone/wormhole-sdk";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { execSync } from "child_process";
import * as fs from "fs";
import {
  GUARDIAN_KEYS,
  expectIxErr,
  expectIxOk,
  invokeVerifySignaturesAndPostVaa,
  loadProgramBpf,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Init Message V1", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    it.skip("Cannot Invoke `process_message_v1` with Different Emitter Authority", async () => {
      // TODO
    });

    it.skip("Cannot Invoke `process_message_v1` to Close Draft Message without `close_account_destination`", async () => {
      // TODO
    });

    it.skip("Cannot Invoke `process_message_v1` with Nonsensical Index", async () => {
      // TODO
    });
  });

  describe("Ok", () => {
    it.skip("Invoke `process_message_v1` to Write Part of Large Message", async () => {
      // TODO
    });

    it.skip("Invoke `process_message_v1` to Write More of Large Message", async () => {
      // TODO
    });

    it.skip("Invoke `process_message_v1` to Close Draft Message", async () => {
      // TODO
    });
  });
});
