import { describe, expect, it } from "@jest/globals";
import { run_worm_command } from "./utils/cli";

describe("worm recover", () => {
  describe("check functionality", () => {
    //mocks
    const digest =
      "0x99656f88302bda18573212d4812daeea7d39f8af695db1fbc4d99fd94f552606";
    const signature =
      "6da03c5e56cb15aeeceadc1e17a45753ab4dc0ec7bf6a75ca03143ed4a294f6f61bc3f478a457833e43084ecd7c985bf2f55a55f168aac0e030fc49e845e497101";

    //expected output
    const expectedAddress = "0x6FbEBc898F403E4773E95feB15E80C9A99c8348d";

    it(`should return correct address from digest and signature`, async () => {
      const output = run_worm_command(`recover ${digest} ${signature}`);

      expect(output).toContain(expectedAddress);
    });
  });
});
