import { describe, expect, it } from "@jest/globals";
import {
  Flag,
  test_command_flags,
  test_command_positional_args,
} from "../utils/tests";

describe("worm aptos upgrade", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["package-dir"];

    //TODO: unskip these tests by removing 'true' param once runtime error is resolved
    // Issue source: https://github.com/wormhole-foundation/wormhole/issues/3109
    test_command_positional_args("aptos upgrade", args, true);
  });

  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--contract-address", alias: "-a" },
      { name: "--network", alias: "-n" },
      { name: "--rpc", alias: "-r" },
      { name: "--named-addresses", alias: undefined },
    ];

    //TODO: unskip these tests by removing 'true' param once runtime error is resolved
    // Issue source: https://github.com/wormhole-foundation/wormhole/issues/3109
    test_command_flags("aptos upgrade", flags, true);
  });

  //NOTE: At least one test must exist to avoid runtime errors in jest
  //TODO: delete empty test once runtime error is resolved
  // Issue source: https://github.com/wormhole-foundation/wormhole/issues/3109
  it(`empty test`, async () => {
    expect(true).toBe(true);
  });
});
