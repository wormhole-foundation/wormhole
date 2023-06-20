import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm aptos deploy-resource", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["seed", "package-dir"];

    test_command_positional_args("aptos deploy-resource", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--rpc", alias: "-r" },
      { name: "--named-addresses", alias: undefined },
    ];

    test_command_flags("aptos deploy-resource", flags);
  });
});
