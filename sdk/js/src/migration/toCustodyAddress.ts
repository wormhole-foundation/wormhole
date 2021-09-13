export default async function toCustodyAddress(
  program_id: string,
  pool: string
) {
  const { to_custody_address } = await import(
    "../solana/migration/wormhole_migration"
  );
  return to_custody_address(program_id, pool);
}
