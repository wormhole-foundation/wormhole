import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm aptos init-wormhole", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--rpc", alias: "-r" },
      { name: "--chain-id", alias: undefined },
      { name: "--governance-chain-id", alias: undefined },
      { name: "--governance-address", alias: undefined },
      { name: "--guardian-address", alias: "-g" },
    ];

    test_command_flags("aptos init-wormhole", flags);
  });
});
