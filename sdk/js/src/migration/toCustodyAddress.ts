import { importMigrationWasm } from "../solana/wasm";

export default async function toCustodyAddress(
  program_id: string,
  pool: string
) {
  const { to_custody_address } = await importMigrationWasm();
  return to_custody_address(program_id, pool);
}
