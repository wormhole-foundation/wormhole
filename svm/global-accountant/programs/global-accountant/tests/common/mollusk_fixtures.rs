//! Builds a `Mollusk` with the global-accountant `.so` plus the real
//! `solana_noreplay.so` at its canonical ID, so tests exercise the production
//! CPI path with no mock branch. The sibling `.so` lives in `tests/fixtures/`
//! and is verified against a pinned SHA-256 digest before loading.
//!
//! ## Regenerating a fixture
//!
//! 1. `cd <sibling-repo> && cargo build-sbf`.
//! 2. Copy the `.so` over `tests/fixtures/<name>.so`.
//! 3. `shasum -a 256 tests/fixtures/<name>.so` and update the matching
//!    `*_SO_SHA256` constant below.
//!
//! Local iteration: `GA_NOREPLAY_SO=<path>` redirects the resolver and skips
//! the SHA-256 check.

use {
    global_accountant_definitions::NOREPLAY_PROGRAM_ID,
    mollusk_svm::{
        program::{create_program_account_loader_v3, loader_keys::LOADER_V3},
        Mollusk,
    },
    sha2::{Digest, Sha256},
    solana_account::Account,
    solana_pubkey::Pubkey,
    std::{fs, path::Path},
};

/// SHA-256 of the pinned `solana_noreplay.so`.
/// Reproduce: `shasum -a 256 programs/global-accountant/tests/fixtures/solana_noreplay.so`.
const NOREPLAY_SO_SHA256: [u8; 32] = [
    0x33, 0xbe, 0x38, 0x6b, 0xac, 0xf5, 0x6b, 0x98, 0x98, 0xfb, 0xa7, 0x5b, 0x10, 0x48, 0xcb, 0xe1,
    0x17, 0x90, 0xf2, 0x42, 0xe9, 0x75, 0x07, 0xb4, 0x52, 0xeb, 0x7f, 0x08, 0xd0, 0x86, 0x9a, 0xc7,
];

/// Build a `Mollusk` with the global-accountant program at `program_id` and the
/// noreplay fixture preloaded at its canonical ID. `program_name` is the `.so`
/// stem (typically `"global_accountant"`).
pub fn mollusk_with_fixtures(program_id: &Pubkey, program_name: &str) -> Mollusk {
    let mut mollusk = Mollusk::new(program_id, program_name);

    let noreplay_elf = read_so(
        &super::noreplay_so_path(),
        "solana_noreplay",
        "GA_NOREPLAY_SO",
        &NOREPLAY_SO_SHA256,
    );

    mollusk.add_program_with_loader_and_elf(
        &Pubkey::new_from_array(NOREPLAY_PROGRAM_ID),
        &LOADER_V3,
        &noreplay_elf,
    );

    mollusk
}

/// Loader-V3 executable `(program_id, Account)` for the noreplay program,
/// required because `process_instruction` consumes the account list verbatim;
/// a system-owned stand-in would fail as `UnsupportedProgramId` at CPI time.
pub fn keyed_account_for_noreplay_program() -> (Pubkey, Account) {
    let id = Pubkey::new_from_array(NOREPLAY_PROGRAM_ID);
    let account = create_program_account_loader_v3(&id);
    (id, account)
}

/// Read a fixture `.so` and verify its SHA-256, panicking with the regen recipe
/// on a missing file or hash drift. Skips the check when the path came from
/// `env_override`.
fn read_so(path: &Path, label: &str, env_override: &str, expected_sha256: &[u8; 32]) -> Vec<u8> {
    let bytes = fs::read(path).unwrap_or_else(|e| {
        panic!(
            "missing fixture program `{label}` at {}: {e}\n  hint: the canonical \
             fixture is checked in at tests/fixtures/{label}.so; if you intentionally \
             removed it, set ${env_override}=/path/to/your.so to redirect",
            path.display()
        )
    });
    if std::env::var(env_override).is_ok() {
        // Override is the documented escape hatch; skip the hash check.
        return bytes;
    }
    let actual = Sha256::digest(&bytes);
    if &actual[..] != expected_sha256 {
        panic!(
            "fixture `{label}` SHA-256 drift\n  expected: {}\n  actual:   {}\n  recompute: \
             shasum -a 256 {}\n  then update the corresponding *_SO_SHA256 constant in \
             tests/common/mollusk_fixtures.rs",
            hex_lower(expected_sha256),
            hex_lower(&actual[..]),
            path.display(),
        );
    }
    bytes
}

/// Lowercase hex-encode a byte slice (for the drift panic message).
fn hex_lower(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push(HEX[(b >> 4) as usize] as char);
        out.push(HEX[(b & 0x0f) as usize] as char);
    }
    out
}
