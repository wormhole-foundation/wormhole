import { importMigrationWasm } from "../solana/wasm";

export default async function shareMintAddress(
  program_id: string,
  pool: string
) {
  const { share_mint_address } = await importMigrationWasm();
  return share_mint_address(program_id, pool);
}
