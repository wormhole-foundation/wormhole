import yargs from "yargs";

describe("worm chain-id", () => {
  const FIRST_POSITIONAL_ARGUMENT = "chain";

  it("should has <chain> as first positional argument", async () => {
    const commandArgvs = await yargs.command(require("../cmds/chainId")).argv;
    const argvList = commandArgvs._;

    expect(argvList[0]).toEqual(FIRST_POSITIONAL_ARGUMENT);
  });
});
