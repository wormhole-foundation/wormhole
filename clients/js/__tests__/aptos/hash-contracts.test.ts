import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm aptos hash-contracts", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["package-dir"];

    test_command_positional_args("aptos hash-contracts", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [{ name: "--named-addresses", alias: undefined }];

    test_command_flags("aptos hash-contracts", flags);
  });
});
