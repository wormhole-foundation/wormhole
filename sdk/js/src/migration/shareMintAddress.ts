export default async function shareMintAddress(
  program_id: string,
  pool: string
) {
  const { share_mint_address } = await import(
    "../solana/migration/wormhole_migration"
  );
  return share_mint_address(program_id, pool);
}
