import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm evm info", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--rpc", alias: undefined },
      { name: "--chain", alias: "-c" },
      { name: "--module", alias: "-m" },
      { name: "--contract-address", alias: "-a" },
      { name: "--implementation-only", alias: "-i" },
    ];

    test_command_flags("evm info", flags);
  });
});
