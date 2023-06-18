import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "./utils/cli";

describe("worm submit", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["vaa"];

    test_command_positional_args("submit", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--chain", alias: "-c" },
      { name: "--network", alias: "-n" },
      { name: "--contract-address", alias: "-a" },
      { name: "--rpc", alias: undefined },
      { name: "--all-chains", alias: "--ac" },
    ];

    test_command_flags("submit", flags);
  });
});
