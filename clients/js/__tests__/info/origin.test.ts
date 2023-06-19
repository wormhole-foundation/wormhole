import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/cli";

describe("worm info origin", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["chain", "address"];

    test_command_positional_args("info origin", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--network", alias: "-n" },
      { name: "--rpc", alias: "-r" },
    ];

    test_command_flags("info origin", flags);
  });
});
