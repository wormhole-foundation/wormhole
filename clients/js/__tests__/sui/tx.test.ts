import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm sui tx", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["transaction-digest"];

    test_command_positional_args("sui tx", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("sui tx", flags);
  });
});
