import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm sui build-coin", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--decimals", alias: "-d" },
      { name: "--version-struct", alias: "-v" },
      { name: "--package-path", alias: "-p" },
      { name: "--wormhole-state", alias: "-w" },
      { name: "--token-bridge-state", alias: "-t" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("sui build-coin", flags);
  });
});
