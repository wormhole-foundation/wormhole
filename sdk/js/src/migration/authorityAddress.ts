export default async function authorityAddress(program_id: string) {
  const { authority_address } = await import(
    "../solana/migration/wormhole_migration"
  );
  return authority_address(program_id);
}
