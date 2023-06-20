import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm generate recover-chain-id", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--guardian-secret", alias: "-g" },
      { name: "--module", alias: "-m" },
      { name: "--evm-chain-id", alias: "-e" },
      { name: "--new-chain-id", alias: "-c" },
    ];

    test_command_flags("generate recover-chain-id", flags);
  });
});
