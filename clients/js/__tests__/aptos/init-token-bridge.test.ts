import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm aptos init-token-bridge", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("aptos init-token-bridge", flags);
  });
});
