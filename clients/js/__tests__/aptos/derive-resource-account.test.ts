import { describe } from "@jest/globals";
import { test_command_positional_args } from "../utils/tests";

describe("worm aptos derive-resource-account", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["account", "seed"];

    test_command_positional_args("aptos derive-resource-account", args);
  });
});
