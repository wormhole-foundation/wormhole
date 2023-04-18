import { execSync } from "child_process";
import fs from "fs";
import path from "path";
import { Network } from "../utils";
import { MoveToml } from "./MoveToml";
import { GithubTreeResponse, SuiBuildOutput } from "./types";
import { isValidSuiAddress } from "./utils";

// TODO(aki): Remove this when this branch is merged in
const TEMPORARY_SUI_BRANCH = "sui/integration_v2";

export const getCoinBuildOutputManual = async (
  network: Network,
  coreBridgeAddress: string,
  tokenBridgeAddress: string,
  vaa: string
): Promise<SuiBuildOutput> => {
  await cloneDependencies();
  setupMainToml(
    `${__dirname}/dependencies/wormhole`,
    network,
    coreBridgeAddress
  );
  setupMainToml(
    `${__dirname}/dependencies/token_bridge`,
    network,
    tokenBridgeAddress
  );
  setupCoin(coreBridgeAddress, tokenBridgeAddress, vaa);
  const buildOutput = buildPackage(`${__dirname}/wrapped_coin`);
  cleanupTempToml(`${__dirname}/dependencies/wormhole`);
  cleanupTempToml(`${__dirname}/dependencies/token_bridge`);
  return buildOutput;
};

const buildPackage = (packagePath: string): SuiBuildOutput => {
  return JSON.parse(
    execSync(
      `sui move build --dump-bytecode-as-base64 --path ${packagePath} 2> /dev/null`,
      {
        encoding: "utf-8",
      }
    )
  );
};

const cleanupTempToml = (packagePath: string): void => {
  const defaultTomlPath = getDefaultTomlPath(packagePath);
  const tempTomlPath = getTempTomlPath(packagePath);
  if (fs.existsSync(tempTomlPath)) {
    fs.renameSync(tempTomlPath, defaultTomlPath);
  }
};

const cloneDependencies = async (): Promise<void> => {
  const { tree, sha: latestSha } = await fetchWormholeTree(
    TEMPORARY_SUI_BRANCH
  );
  const suiSha = tree.find((n) => n.path === "sui")?.sha;
  if (!suiSha) {
    throw new Error("Failed to fetch url");
  }

  const promises: Promise<void>[] = [];
  const { tree: suiTree } = await fetchWormholeTree(suiSha, true);
  promises.push(fetchWormholeFilesFromTree(suiTree, latestSha, "wormhole"));
  promises.push(fetchWormholeFilesFromTree(suiTree, latestSha, "token_bridge"));
  await Promise.all(promises);
};

const fetchWormholeFile = async (
  sha: string,
  path: string
): Promise<string> => {
  const res = await fetch(
    `https://raw.githubusercontent.com/wormhole-foundation/wormhole/${sha}/sui/${path}`
  );
  return res.text();
};

// TODO(aki): we can further optimize subsequent runs by caching every dir/blob
const fetchWormholeFilesFromTree = async (
  tree: GithubTreeResponse["tree"],
  sha: string,
  subdirName: string
): Promise<void> => {
  if (!tree || !tree.length) {
    throw new Error("Received empty tree");
  }

  const latestSha = tree.find((n) => n.path === subdirName)?.sha;
  if (!latestSha) {
    throw new Error(
      `Invalid response, couldn't find node with path ${subdirName}`
    );
  }

  const shaPath = `${__dirname}/dependencies/${subdirName}/.sha`;
  if (
    fs.existsSync(shaPath) &&
    fs.readFileSync(shaPath, "utf-8") === latestSha
  ) {
    return;
  } else {
    fs.rmSync(`${__dirname}/dependencies/${subdirName}`, {
      recursive: true,
      force: true,
    });
  }

  const wormhole = tree.filter(
    (n) => n.path.startsWith(subdirName) && n.type === "blob"
  );
  const promises = [];
  for (const node of wormhole) {
    const localPath = `${__dirname}/dependencies/${node.path}`;
    promises.push(
      fs.promises
        .mkdir(path.dirname(localPath), { recursive: true })
        .then(() => fetchWormholeFile(sha, node.path))
        .then((file) => fs.promises.writeFile(localPath, file))
    );
  }

  await Promise.all(promises);
  fs.writeFileSync(shaPath, latestSha);
};

const fetchWormholeTree = async (
  sha: string,
  recursive: boolean = false
): Promise<GithubTreeResponse> => {
  const res = await fetch(
    `https://api.github.com/repos/wormhole-foundation/wormhole/git/trees/${sha}?recursive=${recursive}`
  );
  return res.json();
};

const getDefaultTomlPath = (packagePath: string): string =>
  `${packagePath}/Move.toml`;

const getTempTomlPath = (packagePath: string): string =>
  `${packagePath}/Move.temp.toml`;

const getTomlPathByNetwork = (packagePath: string, network: Network): string =>
  `${packagePath}/Move.${network.toLowerCase()}.toml`;

const getPackageNameFromPath = (packagePath: string): string =>
  packagePath.split("/").pop() || "";

// TODO(aki): parallelize
const setupCoin = (
  coreBridgeAddress: string,
  tokenBridgeAddress: string,
  vaa: string
): void => {
  fs.rmSync(`${__dirname}/wrapped_coin`, { recursive: true, force: true });
  fs.mkdirSync(`${__dirname}/wrapped_coin/sources`, { recursive: true });

  const coin = fs
    .readFileSync(`${__dirname}/templates/wrapped_coin/coin.move`, "utf8")
    .toString();
  fs.writeFileSync(
    `${__dirname}/wrapped_coin/sources/coin.move`,
    coin.replace(`{{VAA_BYTES}}`, vaa),
    "utf8"
  );

  const toml = new MoveToml(`${__dirname}/templates/wrapped_coin/Move.toml`)
    .updateRow("addresses", "wormhole", coreBridgeAddress)
    .updateRow("addresses", "token_bridge", tokenBridgeAddress)
    .serialize();
  fs.writeFileSync(`${__dirname}/wrapped_coin/Move.toml`, toml, "utf8");
};

const setupMainToml = (
  packagePath: string,
  network: Network,
  publishedAddress: string,
  isDependency: boolean = false
): void => {
  if (!isValidSuiAddress(publishedAddress)) {
    throw new Error(
      `Invalid address ${publishedAddress} for package ${packagePath}`
    );
  }

  const defaultTomlPath = getDefaultTomlPath(packagePath);
  const tempTomlPath = getTempTomlPath(packagePath);
  const networkTomlPath = getTomlPathByNetwork(packagePath, network);

  if (fs.existsSync(tempTomlPath)) {
    // It's possible that this dependency has been set up by another package
    if (isDependency) {
      return;
    }

    cleanupTempToml(packagePath);
  }

  // Save default Move.toml
  if (!fs.existsSync(defaultTomlPath)) {
    throw new Error(
      `Invalid package layout. Move.toml not found at ${defaultTomlPath}`
    );
  }

  fs.renameSync(defaultTomlPath, tempTomlPath);

  // Set Move.toml from appropriate network
  if (!fs.existsSync(networkTomlPath)) {
    throw new Error(`Move.toml for ${network} not found at ${networkTomlPath}`);
  }

  fs.copyFileSync(networkTomlPath, defaultTomlPath);

  // Replace undefined addresses in base Move.toml and ensure dependencies are
  // published
  const tomlStr = fs.readFileSync(defaultTomlPath, "utf8").toString();
  const toml = new MoveToml(tomlStr);
  const packageName = getPackageNameFromPath(packagePath);
  if (!isDependency) {
    if (toml.isPublished()) {
      throw new Error(`Package ${packageName} is already published.`);
    }

    fs.writeFileSync(
      defaultTomlPath,
      toml
        .addRow("package", "published-at", publishedAddress)
        .updateRow("addresses", packageName, publishedAddress)
        .serialize()
    );
  }
};
