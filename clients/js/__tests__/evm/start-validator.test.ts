import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm evm start-validator", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--rpc", alias: undefined },
      { name: "--validator-args", alias: "-a" },
    ];

    test_command_flags("evm start-validator", flags);
  });
});
