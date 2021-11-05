import { importMigrationWasm } from "../solana/wasm";

export default async function fromCustodyAddress(
  program_id: string,
  pool: string
) {
  const { from_custody_address } = await importMigrationWasm();
  return from_custody_address(program_id, pool);
}
