import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm info registrations", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["network", "chain", "module"];

    test_command_positional_args("info registrations", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [{ name: "--verify", alias: "-v" }];

    test_command_flags("info registrations", flags);
  });
});
