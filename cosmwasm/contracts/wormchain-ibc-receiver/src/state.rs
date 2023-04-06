use cw_storage_plus::Map;

pub const CHAIN_CONNECTIONS: Map<String, u16> = Map::new("chain_connections");
