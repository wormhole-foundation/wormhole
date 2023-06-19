import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "./utils/cli";

describe("worm verify-vaa", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--vaa", alias: "-v" },
      { name: "--network", alias: "-n" },
    ];

    test_command_flags("verify-vaa", flags);
  });
});
