import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm near deploy", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["file"];

    test_command_positional_args("near deploy", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--module", alias: "-m" },
      { name: "--network", alias: "-n" },
      { name: "--account", alias: undefined },
      { name: "--attach", alias: undefined },
      { name: "--target", alias: undefined },
      { name: "--mnemonic", alias: undefined },
      { name: "--key", alias: undefined },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("near deploy", flags);
  });
});
