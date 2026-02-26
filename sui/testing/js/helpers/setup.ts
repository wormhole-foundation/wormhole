import * as fs from "fs";
import * as mock from "@certusone/wormhole-sdk/lib/cjs/mock";
import { GUARDIAN_PRIVATE_KEY, UTF8 } from "./consts";

export function generateVaaFromDigest(
  digest: Buffer,
  governance: mock.GovernanceEmitter
) {
  const timestamp = 12345678;
  const published = governance.publishWormholeUpgradeContract(
    timestamp,
    2,
    "0x" + digest.toString("hex")
  );

  // Sui is not supported yet by the SDK, so we need to adjust the payload.
  published.writeUInt16BE(21, published.length - 34);

  // We will use the signed VAA when we execute the upgrade.
  const guardians = new mock.MockGuardians(0, [GUARDIAN_PRIVATE_KEY]);
  return guardians.addSignatures(published, [0]);
}

export function modifyHardCodedVersionControl(
  packagePath: string,
  currentVersion: number,
  newVersion: number
) {
  const versionControlDotMove = `${packagePath}/sources/version_control.move`;

  const contents = fs.readFileSync(versionControlDotMove, UTF8);
  const src = `const CURRENT_BUILD_VERSION: u64 = ${currentVersion}`;
  if (contents.indexOf(src) < 0) {
    throw new Error("current version not found");
  }

  const dst = `const CURRENT_BUILD_VERSION: u64 = ${newVersion}`;
  fs.writeFileSync(versionControlDotMove, contents.replace(src, dst), UTF8);
}
