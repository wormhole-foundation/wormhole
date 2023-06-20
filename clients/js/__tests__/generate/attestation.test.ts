import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm generate attestation", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--guardian-secret", alias: "-g" },
      { name: "--emitter-chain", alias: "-e" },
      { name: "--emitter-address", alias: "-f" },
      { name: "--chain", alias: "-c" },
      { name: "--token-address", alias: "-a" },
      { name: "--decimals", alias: "-d" },
      { name: "--symbol", alias: "-s" },
      { name: "--name", alias: "-n" },
    ];

    test_command_flags("generate attestation", flags);
  });
});
