import { importMigrationWasm } from "../solana/wasm";

export default async function parsePool(data: Uint8Array) {
  const { parse_pool } = await importMigrationWasm();
  return parse_pool(data);
}
