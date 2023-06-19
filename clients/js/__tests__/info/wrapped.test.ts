import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm info wrapped", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["origin-chain", "origin-address", "target-chain"];

    test_command_positional_args("info wrapped", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("info wrapped", flags);
  });
});
