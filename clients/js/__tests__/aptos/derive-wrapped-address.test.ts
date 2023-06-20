import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm aptos derive-wrapped-address", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["chain", "origin-address"];

    test_command_positional_args("aptos derive-wrapped-address", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [{ name: "--network", alias: "-n" }];

    test_command_flags("aptos derive-wrapped-address", flags);
  });
});
