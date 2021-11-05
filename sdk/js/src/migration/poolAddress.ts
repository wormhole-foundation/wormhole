import { importMigrationWasm } from "../solana/wasm";

export default async function poolAddress(
  program_id: string,
  from_mint: string,
  to_mint: string
) {
  const { pool_address } = await importMigrationWasm();
  return pool_address(program_id, from_mint, to_mint);
}
