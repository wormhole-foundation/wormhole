import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm sui init-token-bridge", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--package-id", alias: "-p" },
      { name: "--wormhole-state", alias: "-w" },
      { name: "--governance-chain-id", alias: "-c" },
      { name: "--governance-address", alias: "-a" },
      { name: "--private-key", alias: "-k" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("sui init-token-bridge", flags);
  });
});
