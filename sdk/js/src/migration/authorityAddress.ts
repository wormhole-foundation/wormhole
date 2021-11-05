import { importMigrationWasm } from "../solana/wasm";

export default async function authorityAddress(program_id: string) {
  const { authority_address } = await importMigrationWasm();
  return authority_address(program_id);
}
