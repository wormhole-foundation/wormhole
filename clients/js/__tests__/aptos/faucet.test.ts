import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm aptos faucet", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--rpc", alias: "-r" },
      { name: "--faucet", alias: "-f" },
      { name: "--amount", alias: "-m" },
      { name: "--account", alias: "-a" },
    ];

    test_command_flags("aptos faucet", flags);
  });
});
