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

export function setUpWormholeDirectory(
  srcWormholePath: string,
  dstWormholePath: string
) {
  fs.cpSync(srcWormholePath, dstWormholePath, { recursive: true });

  // Remove irrelevant files. This part is not necessary, but is helpful
  // for debugging a clean package directory.
  const removeThese = [
    "Move.devnet.toml",
    "Move.lock",
    "Makefile",
    "README.md",
    "build",
  ];
  for (const basename of removeThese) {
    fs.rmSync(`${dstWormholePath}/${basename}`, {
      recursive: true,
      force: true,
    });
  }

  // Fix Move.toml file.
  const moveTomlPath = `${dstWormholePath}/Move.toml`;
  const moveToml = fs.readFileSync(moveTomlPath, UTF8);
  fs.writeFileSync(
    moveTomlPath,
    moveToml.replace(`wormhole = "_"`, `wormhole = "0x0"`),
    UTF8
  );
}

export function cleanUpPackageDirectory(packagePath: string) {
  fs.rmSync(packagePath, { recursive: true, force: true });
}
