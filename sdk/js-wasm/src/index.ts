const coreWasms = {
  bundler: async () => await import("./core/bridge"),
  node: async () => await import("./core-node/bridge"),
};
const migrationWasms = {
  bundler: async () => await import("./migration/wormhole_migration"),
  node: async () => await import("./migration-node/wormhole_migration"),
};
const nftWasms = {
  bundler: async () => await import("./nft/nft_bridge"),
  node: async () => await import("./nft-node/nft_bridge"),
};
const tokenWasms = {
  bundler: async () => await import("./token/token_bridge"),
  node: async () => await import("./token-node/token_bridge"),
};
let importDefaultCoreWasm = coreWasms.bundler;
let importDefaultMigrationWasm = migrationWasms.bundler;
let importDefaultNftWasm = nftWasms.bundler;
let importDefaultTokenWasm = tokenWasms.bundler;
export function setDefaultWasm(type: "bundler" | "node") {
  importDefaultCoreWasm = coreWasms[type];
  importDefaultMigrationWasm = migrationWasms[type];
  importDefaultNftWasm = nftWasms[type];
  importDefaultTokenWasm = tokenWasms[type];
}
export async function importCoreWasm() {
  return await importDefaultCoreWasm();
}
export async function importMigrationWasm() {
  return await importDefaultMigrationWasm();
}
export async function importNftWasm() {
  return await importDefaultNftWasm();
}
export async function importTokenWasm() {
  return await importDefaultTokenWasm();
}
