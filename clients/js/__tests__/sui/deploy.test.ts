import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm sui deploy", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["package-dir"];

    test_command_positional_args("sui deploy", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--debug", alias: "-d" },
      { name: "--private-key", alias: "-k" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("sui deploy", flags);
  });
});
