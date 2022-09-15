#![deny(warnings)]

// Re-Export Contract API/Types.
pub use bridge::{
    solitaire as bridge_entrypoint,
    types::ConsistencyLevel,
    BridgeConfig,
    BridgeData,
    MessageData,
    PostVAAData,
    PostedVAAData,
    VerifySignaturesData,
};

mod accounts;
mod message;

// Export Thin Solana API for Wormhole data.
pub use {
    accounts::{
        config,
        emitter,
        fee_collector,
        read_config,
        read_vaa,
        sequence,
    },
    message::{
        post_message,
        Message,
    },
};
