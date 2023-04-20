import yargs from "yargs";
import { parse, Payload, serialiseVAA, sign, VAA, vaaDigest } from "../vaa";

exports.command = "resign <vaa> <guardian-secret>";
exports.desc = "Resigns a VAA (devnet and testnet only, can be in either hex or base64 format) using the specified guardian secret";
exports.builder = (y: typeof yargs) => {
  return y.positional("vaa", {
    describe: "vaa",
    type: "string",
  })
  .positional("guardian-secret", {
    describe: "Guardian's secret key",
    type: "string",
  })
  ;
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

  if (parsed_vaa.signatures.length !== 1) {
    throw Error("Only able to resign VAAs with a single signature, this vaa has " + parsed_vaa.signatures.length.toString());
  }

  parsed_vaa.guardianSetIndex = 0;
  parsed_vaa.signatures = sign([argv["guardian-secret"]], parsed_vaa as VAA<Payload>);
  console.log(serialiseVAA(parsed_vaa as VAA<Payload>));
};
