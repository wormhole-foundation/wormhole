import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm aptos deploy", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["package-dir"];

    test_command_positional_args("aptos deploy", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--rpc", alias: "-r" },
      { name: "--named-addresses", alias: undefined },
    ];

    test_command_flags("aptos deploy", flags);
  });
});
