import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm evm storage-update", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--rpc", alias: undefined },
      { name: "--contract-address", alias: "-a" },
      { name: "--storage-slot", alias: "-k" },
      { name: "--value", alias: "-v" },
    ];

    test_command_flags("evm storage-update", flags);
  });
});
