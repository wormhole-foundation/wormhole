use cw_storage_plus::Map;

pub const CHANNEL_CHAIN: Map<String, u16> = Map::new("channel_chain");
pub const VAA_ARCHIVE: Map<&[u8], bool> = Map::new("vaa_archive");
