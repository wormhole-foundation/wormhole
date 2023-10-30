import * as fs from "fs";

const ROOT = `${__dirname}/..`;
const CORE_BRIDGE_IDL = `${ROOT}/target/idl/wormhole_core_bridge_solana.json`;
const CORE_BRIDGE_TYPES = `${ROOT}/target/types/wormhole_core_bridge_solana.ts`;
const TOKEN_BRIDGE_IDL = `${ROOT}/target/idl/wormhole_token_bridge_solana.json`;
const TOKEN_BRIDGE_TYPES = `${ROOT}/target/types/wormhole_token_bridge_solana.ts`;

const IGNORE_TYPES = [
  '"name": "MessageAccount"',
  '"name": "VaaAccount"',
  '"name": "VaaVersion"',
  '"name": "CoreBridge"',
  '"name": "TokenBridge"',
  '"name": "AccountVariant"',
];

main();

function main() {
  if (!fs.existsSync(CORE_BRIDGE_IDL)) {
    throw new Error("Core Bridge IDL non-existent");
  }
  if (!fs.existsSync(CORE_BRIDGE_TYPES)) {
    throw new Error("Core Bridge types non-existent");
  }
  if (!fs.existsSync(TOKEN_BRIDGE_IDL)) {
    throw new Error("Token Bridge IDL non-existent");
  }
  if (!fs.existsSync(TOKEN_BRIDGE_TYPES)) {
    throw new Error("Token Bridge types non-existent");
  }

  // Core Bridge.
  {
    const idl = fs.readFileSync(CORE_BRIDGE_IDL, "utf8").split("\n");
    const types = fs.readFileSync(CORE_BRIDGE_TYPES, "utf8").split("\n");
    for (const matchStr of IGNORE_TYPES) {
      while (spliceType(idl, matchStr));
      while (spliceType(types, matchStr));
    }
    fs.writeFileSync(CORE_BRIDGE_IDL, idl.join("\n"), "utf8");
    fs.writeFileSync(CORE_BRIDGE_TYPES, types.join("\n"), "utf8");
  }

  // Token Bridge.
  {
    const idl = fs.readFileSync(TOKEN_BRIDGE_IDL, "utf8").split("\n");
    const types = fs.readFileSync(TOKEN_BRIDGE_TYPES, "utf8").split("\n");
    for (const matchStr of IGNORE_TYPES) {
      while (spliceType(idl, matchStr));
      while (spliceType(types, matchStr));
    }
    fs.writeFileSync(TOKEN_BRIDGE_IDL, idl.join("\n"), "utf8");
    fs.writeFileSync(TOKEN_BRIDGE_TYPES, types.join("\n"), "utf8");
  }
}

function spliceType(lines: string[], matchStr: string) {
  let lineNumber = 0;
  let start = -1;
  let spaces = -1;
  for (const line of lines) {
    if (line.includes(matchStr)) {
      start = lineNumber - 1;
      spaces = line.indexOf('"') - 2;
    } else if (start > -1) {
      if (line == "}".padStart(spaces + 1, " ")) {
        lines[start - 1] = lines[start - 1].replace("},", "}");
        lines.splice(start, lineNumber - start + 1);
        return true;
      } else if (line == "},".padStart(spaces + 2, " ")) {
        lines.splice(start, lineNumber - start + 1);
        return true;
      }
    }
    ++lineNumber;
  }

  return false;
}
