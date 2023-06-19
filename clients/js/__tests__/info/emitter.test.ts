import { describe } from "@jest/globals";
import { test_command_positional_args } from "../utils/tests";

describe("worm info emitter", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["chain", "address"];

    test_command_positional_args("info emitter", args);
  });
});
