import yargs from "yargs";
import { parse, vaaDigest } from "../vaa";

exports.command = "parse <vaa>";
exports.desc = "Parse a VAA (can be in either hex or base64 format)";
exports.builder = (y: typeof yargs) => {
  return y.positional("vaa", {
    describe: "vaa",
    type: "string",
  });
};
exports.handler = (argv) => {
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
  const parsed_vaa = parse(buf);
  let parsed_vaa_with_digest = parsed_vaa;
  parsed_vaa_with_digest["digest"] = vaaDigest(parsed_vaa);
  console.log(parsed_vaa_with_digest);
};
