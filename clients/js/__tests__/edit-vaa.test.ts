import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "./utils/cli";

describe("worm edit-vaa", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--vaa", alias: "-v" },
      { name: "--network", alias: "-n" },
      { name: "--guardian-set-index", alias: "--gsi" },
      { name: "--signatures", alias: "--sigs" },
      { name: "--wormscanurl", alias: "--wsu" },
      { name: "--wormscanfile", alias: "--wsf" },
      { name: "--emitter-chain-id", alias: "--ec" },
      { name: "--emitter-address", alias: "--ea" },
      { name: "--nonce", alias: "--no" },
      { name: "--sequence", alias: "--seq" },
      { name: "--consistency-level", alias: "--cl" },
      { name: "--timestamp", alias: "--ts" },
      { name: "--payload", alias: "-p" },
      { name: "--guardian-secret", alias: "--gs" },
    ];

    test_command_flags("edit-vaa", flags);
  });
});
