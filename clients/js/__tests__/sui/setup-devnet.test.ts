import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm sui setup-devnet", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--private-key", alias: "-k" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("sui setup-devnet", flags);
  });
});
