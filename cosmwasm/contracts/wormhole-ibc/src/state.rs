use cw_storage_plus::{Item, Map};

pub const WORMCHAIN_CHANNEL_ID: Item<String> = Item::new("wormchain_channel_id");
pub const VAA_ARCHIVE: Map<&[u8], bool> = Map::new("vaa_archive");
