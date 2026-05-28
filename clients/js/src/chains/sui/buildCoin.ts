import fs from "fs";
import { buildPackage } from "./publish";
import { SuiBuildOutput } from "./types";
import { Network } from "@wormhole-foundation/sdk";

export const buildCoin = async (
  network: Network,
  packagePath: string,
  version: string,
  decimals: number
): Promise<SuiBuildOutput> => {
  try {
    setupCoin(packagePath, version, decimals);
    return buildPackage(`${packagePath}/wrapped_coin`, network);
  } finally {
    cleanupCoin(`${packagePath}/wrapped_coin`);
  }
};

const setupCoin = (
  packagePath: string,
  version: string,
  decimals: number
): void => {
  // Check to see if the given version string is valid. We don't include the
  // end boundary in the regex to accomodate versions such as V__0_1_0_patch,
  // in the off chance we need such a naming scheme.
  if (!/^V__[0-9]+_[0-9]+_[0-9]+/.test(version)) {
    throw new Error(`Invalid version ${version}`);
  }

  // Assemble package directory
  fs.rmSync(`${packagePath}/wrapped_coin`, { recursive: true, force: true });
  fs.mkdirSync(`${packagePath}/wrapped_coin/sources`, { recursive: true });

  // Replace template variables in coin.move
  const coinTemplate = fs
    .readFileSync(
      `${packagePath}/templates/wrapped_coin/sources/coin.move`,
      "utf8"
    )
    .toString();
  const coin = coinTemplate
    .replace(/{{DECIMALS}}/, decimals.toString())
    .replace(/{{VERSION}}/g, version);
  fs.writeFileSync(
    `${packagePath}/wrapped_coin/sources/coin.move`,
    coin,
    "utf8"
  );

  // Copy Move.toml from template
  const templateToml = fs.readFileSync(
    `${packagePath}/templates/wrapped_coin/Move.toml`,
    "utf8"
  );
  fs.writeFileSync(
    `${packagePath}/wrapped_coin/Move.toml`,
    templateToml,
    "utf8"
  );

  // Note: In Sui v1.63+, the package system automatically resolves dependencies
  // using Pub.localnet.toml for ephemeral publications. No manual Move.toml
  // modification is needed.
};

const cleanupCoin = (packagePath: string) => {
  // Clean up the generated wrapped_coin directory
  fs.rmSync(packagePath, { recursive: true, force: true });
};
