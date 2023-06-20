import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm generate upgrade", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--guardian-secret", alias: "-g" },
      { name: "--chain", alias: "-c" },
      { name: "--contract-address", alias: "-a" },
      { name: "--module", alias: "-m" },
    ];

    test_command_flags("generate upgrade", flags);
  });
});
