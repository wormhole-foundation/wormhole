export default async function parsePool(data: Uint8Array) {
  const { parse_pool } = await import("../solana/migration/wormhole_migration");
  return parse_pool(data);
}
