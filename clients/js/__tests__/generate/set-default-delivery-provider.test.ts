import { describe } from "@jest/globals";
import { Flag, test_command_flags } from "../utils/tests";

describe("worm generate set-default-delivery-provider", () => {
  describe("check flags", () => {
    const flags: Flag[] = [
      { name: "--guardian-secret", alias: "-g" },
      { name: "--chain", alias: "-c" },
      { name: "--delivery-provider-address", alias: "-p" },
    ];

    test_command_flags("generate set-default-delivery-provider", flags);
  });
});
