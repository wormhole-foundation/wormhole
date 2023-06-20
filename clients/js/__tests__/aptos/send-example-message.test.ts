import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm aptos send-example-message", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["message"];

    test_command_positional_args("aptos send-example-message", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [{ name: "--network", alias: "-n" }];

    test_command_flags("aptos send-example-message", flags);
  });
});
