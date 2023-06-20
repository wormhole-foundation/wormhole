import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm sui publish-example-message", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--package-id", alias: "-p" },
      { name: "--state", alias: "-s" },
      { name: "--wormhole-state", alias: "-w" },
      { name: "--message", alias: "-m" },
      { name: "--private-key", alias: "-k" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("sui publish-example-message", flags);
  });
});
