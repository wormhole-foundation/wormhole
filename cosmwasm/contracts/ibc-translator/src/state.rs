use cw_storage_plus::{Item, Map};
use cw_token_bridge::msg::TransferInfoResponse;

pub const TOKEN_BRIDGE_CONTRACT: Item<String> = Item::new("token_bridge_contract");

// Holds temp state for the wormhole message that the contract is currently processing
pub const CURRENT_TRANSFER: Item<TransferInfoResponse> = Item::new("current_transfer");

// Maps cw20 address -> bank token denom
pub const CW_DENOMS: Map<String, String> = Map::new("cw_denoms");

pub const CHAIN_TO_CHANNEL_MAP: Map<u16, String> = Map::new("chain_to_channel_map");

pub const VAA_ARCHIVE: Map<&[u8], bool> = Map::new("vaa_archive");
