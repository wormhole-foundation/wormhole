import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm sui init-wormhole", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--package-id", alias: "-p" },
      { name: "--initial-guardian", alias: "-i" },
      { name: "--debug", alias: "-d" },
      { name: "--governance-chain-id", alias: "-c" },
      { name: "--guardian-set-index", alias: "-s" },
      { name: "--governance-address", alias: "-a" },
      { name: "--private-key", alias: "-k" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("sui init-wormhole", flags);
  });
});
