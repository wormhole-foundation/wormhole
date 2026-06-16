//! Shared mollusk fixtures for the in-process real-CPI suite.

pub mod mollusk_fixtures;

use std::path::PathBuf;

/// Resolve `solana_noreplay.so`: the hash-pinned `tests/fixtures/` copy by
/// default, or `GA_NOREPLAY_SO` for local iteration (override skips the check).
pub fn noreplay_so_path() -> PathBuf {
    if let Ok(p) = std::env::var("GA_NOREPLAY_SO") {
        return PathBuf::from(p);
    }
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("tests/fixtures/solana_noreplay.so")
}
