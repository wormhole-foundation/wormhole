import * as fs from "fs";

const IDL_ROOT = `${__dirname}/../../target/idl`;
const TYPES_ROOT = `${__dirname}/../../target/types`;

const IDL_OUT = `${__dirname}/../src/idl`;
const TYPES_OUT = `${__dirname}/../src/types`;

const CORE_BRIDGE_IDL = "solana_wormhole_core_bridge.json";
const TOKEN_BRIDGE_IDL = "solana_wormhole_token_bridge.json";
const CORE_BRIDGE_TYPE = "SolanaWormholeCoreBridge";

main();

async function main() {
  // Make directories if they don't exist.
  if (!fs.existsSync(IDL_OUT)) {
    fs.mkdirSync(IDL_OUT);
  }
  if (!fs.existsSync(TYPES_OUT)) {
    fs.mkdirSync(TYPES_OUT);
  }

  // Now copy IDL and Typescript types.
  for (const idlFilename of [CORE_BRIDGE_IDL, TOKEN_BRIDGE_IDL]) {
    const idl = fs.readFileSync(`${IDL_ROOT}/${idlFilename}`, "utf8");

    fs.writeFileSync(`${IDL_OUT}/${idlFilename}`, idl);
    fs.writeFileSync(
      `${TYPES_OUT}/${idlFilename.replace(".json", ".ts")}`,
      fs.readFileSync(
        `${TYPES_ROOT}/${idlFilename.replace(".json", ".ts")}`,
        "utf8"
      )
    );
  }

  // const coreBridgeTypes: any[] = JSON.parse(idl).types;

  // const otherIdlFilenames = [
  //   ["solana_wormhole_token_bridge.json", "SolanaWormholeTokenBridge"],
  // ];
  // for (const [fn, idlType] of otherIdlFilenames) {
  //   // First fix IDL.
  //   const idl = fs.readFileSync(`${IDL_ROOT}/${fn}`, "utf8");
  //   const fixedIdl = JSON.parse(idl);

  //   const types: any[] = fixedIdl.types;
  //   // for (const coreBridgeTypeName of SPECIFIC_CORE_BRIDGE_TYPES) {
  //   //   // Add if non-existent. Throw if exists (no name clashing please).
  //   //   if (types.find((t) => t.name == coreBridgeTypeName) === undefined) {
  //   //     const coreBridgeType = coreBridgeTypes.find(
  //   //       (t) => t.name == coreBridgeTypeName
  //   //     );
  //   //     fixedIdl.types.push(coreBridgeType);
  //   //   } else {
  //   //     throw new Error(`name clash: ${coreBridgeTypeName}`);
  //   //   }
  //   // }

  //   fixedIdl.types = types.sort((left, right) =>
  //     left.name < right.name ? -1 : 1
  //   );

  //   fs.writeFileSync(`${IDL_OUT}/${fn}`, JSON.stringify(fixedIdl, null, 2));

  //   // Now fix Typescript type.
  //   const compiledTypes = fs.readFileSync(
  //     `${TYPES_ROOT}/${fn.replace(".json", ".ts")}`,
  //     "utf8"
  //   );

  //   // First the type.
  //   const constIdlIndex = compiledTypes.indexOf("export const IDL: ");

  //   const fixedType = JSON.parse(
  //     compiledTypes.substring(
  //       compiledTypes.indexOf(idlType) + idlType.length + 3,
  //       constIdlIndex - 3
  //     )
  //   );
  //   fixedType.types = fixedIdl.types;

  //   // Now const IDL.
  //   const fixedConstIdl = JSON.parse(
  //     compiledTypes.substring(
  //       constIdlIndex + idlType.length + 21,
  //       compiledTypes.length - 2
  //     )
  //   );
  //   fixedConstIdl.types = fixedIdl.types;

  //   fs.writeFileSync(
  //     `${TYPES_OUT}/${fn.replace(".json", ".ts")}`,
  //     `export type SolanaWormholeTokenBridge = ${JSON.stringify(
  //       fixedType,
  //       null,
  //       2
  //     )};\n\nexport const IDL: SolanaWormholeTokenBridge = ${JSON.stringify(
  //       fixedConstIdl,
  //       null,
  //       2
  //     )};\n`
  //   );
  // }
}
