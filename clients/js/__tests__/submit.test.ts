import yargs from "yargs";
import { describe, expect, it } from "@jest/globals";
import { test_command_flags, test_command_positional_args } from "./utils/cli";

describe("worm submit", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["vaa"];

    test_command_positional_args("submit", args);
  });

  describe("check flags", () => {
    const flags = ["chain", "network", "contract-address", "rpc", "all-chains"];

    test_command_flags("submit", flags);
  });
});
