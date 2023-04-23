import yargs from "yargs";
import axios from "axios";
import { Other } from "@certusone/wormhole-sdk";
import { parse, Payload, serialiseVAA, sign, Signature, VAA } from "../vaa";

exports.command = "edit-vaa";
exports.desc = "Edits or generates a VAA";
exports.builder = (y: typeof yargs) => {
  return y.option("vaa", {
    alias: "v",
    describe: "vaa in hex format",
    type: "string",
  })
  .option("guardian-set-index", {
    alias: "gsi",
    describe: "Guardian set index",
    type: "number",
  })
  .option("signatures", {
    alias: "sigs",
    describe: "Comma separated list of signatures",
    type: "string",
  })
  .option("sigurl", {
    alias: "su",
    describe: "url to json containing the vaa data including signatures",
    type: "string",
  }) 
  .option("sigfile", {
    alias: "sf",
    describe: "json file containing the vaa data including signatures",
    type: "string",
  })  
  .option("guardian-secret", {
    alias: "gs",
    describe: "Guardian's secret key",
    type: "string",
  })
  ;
};
exports.handler = async(argv) => {
  let numSigs = 0;
  if (argv["signatures"]) { numSigs += 1; }
  if (argv["sigfile"]) { numSigs += 1; }
  if (argv["sigurl"]) { numSigs += 1; }
  if (argv["guardian-secret"]) { numSigs += 1; }
  if (numSigs > 1) {
    throw new Error(`may only specify one of "--signatures", "--sigfile", "--sigurl" or "--guardian-secret"`);
  }

  let vaa: VAA<Payload | Other>;
  if (argv["vaa"]) {
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
    vaa = parse(buf);
  } else {
    vaa = {
        version: 1,
        guardianSetIndex: 0,
        signatures: [],
        timestamp: 0,
        nonce: 0,
        emitterChain: 0,
        emitterAddress: "0x0",
        sequence: BigInt(Math.floor(Math.random() * 100000000)),
        consistencyLevel: 0,
        payload: {
          type: "Other",
          hex: `00`,
      },
    };
  }

  if (argv["signatures"]) {
    vaa.signatures = argv["signatures"].split(",");
  } else if (argv["sigfile"]) {
    const vaaData = require(argv["sigfile"]);
    let sigs: Signature[] = [];
    for (let gsi in vaaData) {
      let sig: Signature = {
        guardianSetIndex: Number(gsi),
        signature: Buffer.from(vaaData[gsi].signature, 'base64').toString('hex'),
       };

      sigs.push(sig);
    }
    vaa.signatures = sigs;
  } else if (argv["sigurl"]) {
    let vaaData = await axios.get(argv["sigurl"]);
    let sigs: Signature[] = [];
    for (let gsi in vaaData.data) {
      let sig: Signature = {
        guardianSetIndex: Number(gsi),
        signature: Buffer.from(vaaData.data[gsi].signature, 'base64').toString('hex'),
       };

      sigs.push(sig);
    }
    vaa.signatures = sigs;
  } else if (argv["guardian-secret"]) {
    vaa.guardianSetIndex = 0;
    vaa.signatures = sign([argv["guardian-secret"]], vaa as VAA<Payload>);
  }

  console.log(serialiseVAA(vaa as unknown as VAA<Payload>))
};
