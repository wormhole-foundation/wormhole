export default async function fromCustodyAddress(
  program_id: string,
  pool: string
) {
  const { from_custody_address } = await import(
    "../solana/migration/wormhole_migration"
  );
  return from_custody_address(program_id, pool);
}
