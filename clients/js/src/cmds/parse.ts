import yargs from "yargs";
import { parse, vaaDigest } from "../vaa";

export const command = "parse <vaa>";
export const desc = "Parse a VAA (can be in either hex or base64 format)";
export const builder = (y: typeof yargs) => {
  return y.positional("vaa", {
    describe: "vaa",
    type: "string",
    demandOption: true,
  });
};
export const handler = (argv: Awaited<ReturnType<typeof builder>["argv"]>) => {
  let buf: Buffer;
  try {
    buf = Buffer.from(String(argv.vaa), "hex");
    if (buf.length == 0) {
      throw Error("Couldn't parse VAA as hex");
    }
  } catch (e) {
    buf = Buffer.from(String(argv.vaa), "base64");
    if (buf.length == 0) {
      throw Error("Couldn't parse VAA as base64 or hex");
    }
  }

  const parsedVaa = parse(buf);
  console.log(
    JSON.stringify({ ...parsedVaa, digest: vaaDigest(parsedVaa) }, null, 2)
  );
};
