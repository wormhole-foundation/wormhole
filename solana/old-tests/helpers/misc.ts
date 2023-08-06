import * as fs from "fs";
import {
  CORE_BRIDGE_PROGRAM_ID,
  TEST_ROOT,
  TOKEN_BRIDGE_PROGRAM_ID,
} from "./consts";

export function tmpPath() {
  const tmp = `${TEST_ROOT}/.tmp`;
  if (!fs.existsSync(tmp)) {
    fs.mkdirSync(tmp);
  }

  return tmp;
}

export function removeTmpPath() {
  fs.rmSync(tmpPath(), { force: true, recursive: true });
}

export function artifactsPath() {
  const artifacts = `${TEST_ROOT}/../target/deploy`;
  if (!fs.existsSync(artifacts)) {
    throw new Error("Artifacts not found. Run `anchor build` first.");
  }

  return artifacts;
}

export function coreBridgeKeyPath() {
  return `${TEST_ROOT}/keys/${CORE_BRIDGE_PROGRAM_ID}.json`;
}

export function tokenBridgeKeyPath() {
  return `${TEST_ROOT}/keys/${TOKEN_BRIDGE_PROGRAM_ID}.json`;
}
