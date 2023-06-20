import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm evm chains", () => {
  describe("check flags", () => {
    const flags: Flag[] = [{ name: "--rpc", alias: undefined }];

    test_command_flags("evm chains", flags);
  });
});
