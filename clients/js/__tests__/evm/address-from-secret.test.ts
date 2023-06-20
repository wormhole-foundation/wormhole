import { describe } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm evm address-from-secret", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["secret"];

    test_command_positional_args("evm address-from-secret", args);
  });

  describe("check flags", () => {
    const flags: Flag[] = [{ name: "--rpc", alias: undefined }];

    test_command_flags("evm address-from-secret", flags);
  });
});
