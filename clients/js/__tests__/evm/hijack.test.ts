import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm evm hijack", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--rpc", alias: undefined },
      { name: "--core-contract-address", alias: "-a" },
      { name: "--guardian-address", alias: "-g" },
      { name: "--guardian-set-index", alias: "-i" },
    ];

    test_command_flags("evm hijack", flags);
  });
});
