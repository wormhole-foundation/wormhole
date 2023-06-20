import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm aptos start-validator", () => {
  describe("check flags", () => {
    const flags: Flag[] = [{ name: "--validator-args", alias: "-a" }];

    test_command_flags("aptos start-validator", flags);
  });
});
