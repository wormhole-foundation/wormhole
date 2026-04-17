mod helpers;

use cosmwasm_std::Binary;
use helpers::*;

use accountant::state::account;
use serde_wormhole::RawMessage;
use wormhole_bindings::fake::WormholeKeeper;
use wormhole_sdk::{
    vaa::{Body, Header},
    Address, Chain,
};

// --- Constants ---

const HUB_CHAIN: u16 = 2; // Ethereum
const HUB_ADDR: [u8; 32] = [0x42; 32];

const SPOKE_CHAIN_A: u16 = 23; // Arbitrum
const SPOKE_ADDR_A: [u8; 32] = [0x43; 32];

const SPOKE_CHAIN_B: u16 = 30; // Base
const SPOKE_ADDR_B: [u8; 32] = [0x44; 32];

const ROGUE_ADDR_X: [u8; 32] = [0xAA; 32]; // rogue on SPOKE_CHAIN_A

const DUMMY_MANAGER: [u8; 32] = [0x11; 32];
const DUMMY_TOKEN: [u8; 32] = [0x22; 32];

// --- Payload builders ---

fn hub_init_payload(mode: u8) -> Vec<u8> {
    let mut p = Vec::new();
    p.extend_from_slice(&[0x9c, 0x23, 0xbd, 0x3b]); // INFO_PREFIX
    p.extend_from_slice(&DUMMY_MANAGER); // manager_address
    p.push(mode); // 0 = Locking, 1 = Burning
    p.extend_from_slice(&DUMMY_TOKEN); // token_address
    p.push(8); // token_decimals
    p
}

fn registration_payload(chain_id: u16, addr: [u8; 32]) -> Vec<u8> {
    let mut p = Vec::new();
    p.extend_from_slice(&[0x18, 0xfc, 0x67, 0xc2]); // PEER_INFO_PREFIX
    p.extend_from_slice(&chain_id.to_be_bytes());
    p.extend_from_slice(&addr);
    p
}

fn transfer_payload(decimals: u8, amount: u64, to_chain: u16) -> Vec<u8> {
    // NativeTokenTransfer
    let mut ntt = Vec::new();
    ntt.extend_from_slice(&[0x99, 0x4E, 0x54, 0x54]); // NTT_PREFIX
    ntt.push(decimals);
    ntt.extend_from_slice(&amount.to_be_bytes());
    ntt.extend_from_slice(&[0u8; 32]); // source_token
    ntt.extend_from_slice(&[0u8; 32]); // to (recipient)
    ntt.extend_from_slice(&to_chain.to_be_bytes());

    // NttManagerMessage
    let mut mgr = Vec::new();
    mgr.extend_from_slice(&[0u8; 32]); // id
    mgr.extend_from_slice(&[0u8; 32]); // sender
    let ntt_len = ntt.len() as u16;
    mgr.extend_from_slice(&ntt_len.to_be_bytes()); // payload_len
    mgr.extend_from_slice(&ntt);

    // TransceiverMessage
    let mut p = Vec::new();
    p.extend_from_slice(&[0x99, 0x45, 0xFF, 0x10]); // WH_PREFIX
    p.extend_from_slice(&[0u8; 32]); // source_ntt_manager
    p.extend_from_slice(&[0u8; 32]); // recipient_ntt_manager
    let mgr_len = mgr.len() as u16;
    p.extend_from_slice(&mgr_len.to_be_bytes()); // ntt_manager_payload_len
    p.extend_from_slice(&mgr);
    p.extend_from_slice(&0u16.to_be_bytes()); // transceiver_payload_len = 0
    p
}

// --- VAA builder ---

fn build_signed_vaa(
    wh: &WormholeKeeper,
    emitter_chain: u16,
    emitter_address: [u8; 32],
    seq: u64,
    payload: &[u8],
) -> Binary {
    let body = Body {
        timestamp: 0,
        nonce: 0,
        emitter_chain: Chain::from(emitter_chain),
        emitter_address: Address(emitter_address),
        sequence: seq,
        consistency_level: 0,
        payload: RawMessage::new(payload),
    };

    let body_bytes = serde_wormhole::to_vec(&body).unwrap();
    let signatures = wh.sign(&body_bytes);

    let header = Header {
        version: 1,
        guardian_set_index: wh.guardian_set_index(),
        signatures,
    };

    let vaa = wormhole_sdk::Vaa {
        version: header.version,
        guardian_set_index: header.guardian_set_index,
        signatures: header.signatures,
        timestamp: body.timestamp,
        nonce: body.nonce,
        emitter_chain: body.emitter_chain,
        emitter_address: body.emitter_address,
        sequence: body.sequence,
        consistency_level: body.consistency_level,
        payload: RawMessage::new(payload),
    };

    serde_wormhole::to_vec(&vaa).map(Binary::from).unwrap()
}

// --- Sequence counter ---

struct Seq(u64);
impl Seq {
    fn new() -> Self {
        Self(0)
    }
    fn next(&mut self) -> u64 {
        let v = self.0;
        self.0 += 1;
        v
    }
}

// --- Helper to set up the legitimate hub + two spokes ---

fn setup_legitimate_network(wh: &WormholeKeeper, contract: &mut Contract, seq: &mut Seq) {
    // 1. Hub init (locking mode)
    let vaa = build_signed_vaa(wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();

    // 2. Hub pre-registers spoke A (hub can register peers that don't have a hub yet)
    let vaa = build_signed_vaa(
        wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_A, SPOKE_ADDR_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // 3. Spoke A registers hub → inherits hub (hub has pre-registered spoke A)
    let vaa = build_signed_vaa(
        wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // 4. Hub pre-registers spoke B
    let vaa = build_signed_vaa(
        wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_B, SPOKE_ADDR_B),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // 5. Spoke B registers hub → inherits hub (hub has pre-registered spoke B)
    let vaa = build_signed_vaa(
        wh,
        SPOKE_CHAIN_B,
        SPOKE_ADDR_B,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // 6. Cross-register spokes (both have hubs now, hubs match)
    let vaa = build_signed_vaa(
        wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_B, SPOKE_ADDR_B),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    let vaa = build_signed_vaa(
        wh,
        SPOKE_CHAIN_B,
        SPOKE_ADDR_B,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_A, SPOKE_ADDR_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();
}

fn account_key(chain: u16) -> account::Key {
    account::Key::new(
        chain,
        HUB_CHAIN,
        accountant::state::TokenAddress::new(HUB_ADDR),
    )
}

// ============================================================
// 1. Legitimate registration & transfer tests
// ============================================================

#[test]
fn legitimate_hub_init_locking() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();
}

#[test]
fn hub_init_burning_rejected() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(1));
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("burning hub should be rejected");
    assert!(
        err.root_cause()
            .to_string()
            .contains("ignoring non-locking NTT initialization"),
        "unexpected error: {err}"
    );
}

#[test]
fn duplicate_hub_init_rejected() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();

    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("duplicate hub init should fail");
    assert!(
        err.root_cause()
            .to_string()
            .contains("hub entry already exists"),
        "unexpected error: {err}"
    );
}

#[test]
fn spoke_registers_hub_and_inherits() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    // Init hub
    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();

    // Hub pre-registers spoke
    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_A, SPOKE_ADDR_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Spoke registers hub → should inherit
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    contract.submit_vaas(vec![vaa]).unwrap();
}

#[test]
fn spoke_registers_hub_without_preregistration_rejected() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    // Init hub
    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();

    // Spoke tries to register hub WITHOUT hub pre-registering the spoke first
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("should require hub pre-registration");
    assert!(
        err.root_cause()
            .to_string()
            .contains("hub has not registered this transceiver as a peer"),
        "unexpected error: {err}"
    );
}

#[test]
fn registration_to_peer_without_hub_rejected() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    // Try to register a peer when neither sender nor peer has a hub entry
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_B, SPOKE_ADDR_B),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("should fail without hub");
    assert!(
        err.root_cause().to_string().contains("no registered hub"),
        "unexpected error: {err}"
    );
}

#[test]
fn registration_to_non_hub_without_own_hub_rejected() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    // Init hub, pre-register spoke A, then spoke A registers hub
    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();

    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_A, SPOKE_ADDR_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Spoke B (no hub yet) tries to register spoke A (not a hub) → rejected
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_B,
        SPOKE_ADDR_B,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_A, SPOKE_ADDR_A),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("non-hub registration should fail");
    assert!(
        err.root_cause()
            .to_string()
            .contains("ignoring attempt to register peer before hub"),
        "unexpected error: {err}"
    );
}

#[test]
fn hub_mismatch_rejected() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    // Init two different hubs
    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();

    let other_hub: [u8; 32] = [0xFF; 32];
    let vaa = build_signed_vaa(&wh, 420, other_hub, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();

    // Hub pre-registers spoke A, then spoke A inherits first hub
    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_A, SPOKE_ADDR_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Spoke A tries to register peer from different hub → mismatch
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(420, other_hub),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("hub mismatch should fail");
    assert!(
        err.root_cause()
            .to_string()
            .contains("peer hub does not match"),
        "unexpected error: {err}"
    );
}

#[test]
fn duplicate_peer_registration_rejected() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();

    // Hub pre-registers spoke A
    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &registration_payload(SPOKE_CHAIN_A, SPOKE_ADDR_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Spoke A registers hub → inherits
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Try same registration again (different VAA sequence, but same peer slot)
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("duplicate should fail");
    assert!(
        err.root_cause()
            .to_string()
            .contains("peer entry for this chain already exists"),
        "unexpected error: {err}"
    );
}

#[test]
fn legitimate_hub_to_spoke_transfer() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();
    setup_legitimate_network(&wh, &mut contract, &mut seq);

    // Transfer 1000 from hub to spoke A
    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &transfer_payload(8, 1000, SPOKE_CHAIN_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Hub balance should be 1000 (locked)
    let bal = contract.query_balance(account_key(HUB_CHAIN)).unwrap();
    assert_eq!(*bal, cosmwasm_std::Uint256::from(1000u128));

    // Spoke A balance should be 1000 (minted)
    let bal = contract.query_balance(account_key(SPOKE_CHAIN_A)).unwrap();
    assert_eq!(*bal, cosmwasm_std::Uint256::from(1000u128));
}

#[test]
fn legitimate_spoke_to_hub_transfer() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();
    setup_legitimate_network(&wh, &mut contract, &mut seq);

    // Seed: hub → spoke A (1000)
    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &transfer_payload(8, 1000, SPOKE_CHAIN_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Spoke A → hub (400)
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &transfer_payload(8, 400, HUB_CHAIN),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Hub: 1000 - 400 = 600
    let bal = contract.query_balance(account_key(HUB_CHAIN)).unwrap();
    assert_eq!(*bal, cosmwasm_std::Uint256::from(600u128));

    // Spoke A: 1000 - 400 = 600
    let bal = contract.query_balance(account_key(SPOKE_CHAIN_A)).unwrap();
    assert_eq!(*bal, cosmwasm_std::Uint256::from(600u128));
}

#[test]
fn legitimate_spoke_to_spoke_transfer() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();
    setup_legitimate_network(&wh, &mut contract, &mut seq);

    // Seed: hub → spoke A (1000)
    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &transfer_payload(8, 1000, SPOKE_CHAIN_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Spoke A → spoke B (300)
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        SPOKE_ADDR_A,
        seq.next(),
        &transfer_payload(8, 300, SPOKE_CHAIN_B),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Spoke A: 1000 - 300 = 700
    let bal = contract.query_balance(account_key(SPOKE_CHAIN_A)).unwrap();
    assert_eq!(*bal, cosmwasm_std::Uint256::from(700u128));

    // Spoke B: 0 + 300 = 300
    let bal = contract.query_balance(account_key(SPOKE_CHAIN_B)).unwrap();
    assert_eq!(*bal, cosmwasm_std::Uint256::from(300u128));
}

#[test]
fn single_rogue_blocked_at_hub_inheritance() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();
    setup_legitimate_network(&wh, &mut contract, &mut seq);

    // Seed: hub → spoke A (1000)
    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &transfer_payload(8, 1000, SPOKE_CHAIN_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Rogue on spoke A's chain tries to register the hub → blocked at inheritance
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        ROGUE_ADDR_X,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("rogue should be blocked at inheritance");
    assert!(
        err.root_cause()
            .to_string()
            .contains("hub has not registered this transceiver as a peer"),
        "unexpected error: {err}"
    );
}

// ============================================================
// 2. Rogue emitter attack (Scenario A)
// ============================================================

#[test]
fn rogue_hub_inheritance_blocked_without_preregistration() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();
    setup_legitimate_network(&wh, &mut contract, &mut seq);

    // Rogue X (on spoke chain A) tries to register hub → blocked because hub
    // has not pre-registered the rogue
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        ROGUE_ADDR_X,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("rogue should not inherit hub");
    assert!(
        err.root_cause()
            .to_string()
            .contains("hub has not registered this transceiver as a peer"),
        "unexpected error: {err}"
    );
}

#[test]
fn rogue_pair_transfer_blocked() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();
    setup_legitimate_network(&wh, &mut contract, &mut seq);

    // Seed legitimate balance: hub → spoke A (1000)
    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        HUB_ADDR,
        seq.next(),
        &transfer_payload(8, 1000, SPOKE_CHAIN_A),
    );
    contract.submit_vaas(vec![vaa]).unwrap();

    // Verify initial state
    let bal = contract.query_balance(account_key(SPOKE_CHAIN_A)).unwrap();
    assert_eq!(*bal, cosmwasm_std::Uint256::from(1000u128));

    // Rogue X tries to inherit hub → blocked
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        ROGUE_ADDR_X,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("rogue should not inherit hub");
    assert!(
        err.root_cause()
            .to_string()
            .contains("hub has not registered this transceiver as a peer"),
        "unexpected error: {err}"
    );

    // Without hub inheritance, the rogue can't even attempt a transfer
    // (it has no hub registration, so the transfer would fail at hub load)
    let vaa = build_signed_vaa(
        &wh,
        SPOKE_CHAIN_A,
        ROGUE_ADDR_X,
        seq.next(),
        &transfer_payload(8, 400, SPOKE_CHAIN_B),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("rogue transfer should fail");
    assert!(
        err.root_cause().to_string().contains("no registered hub"),
        "unexpected error: {err}"
    );

    // Legitimate balance is untouched
    let bal = contract.query_balance(account_key(SPOKE_CHAIN_A)).unwrap();
    assert_eq!(*bal, cosmwasm_std::Uint256::from(1000u128));
}

// ============================================================
// 3. Scenario B is impossible (rogue on hub chain)
// ============================================================

#[test]
fn rogue_on_hub_chain_blocked_at_inheritance() {
    let (wh, mut contract) = proper_instantiate();
    let mut seq = Seq::new();

    // Init hub
    let vaa = build_signed_vaa(&wh, HUB_CHAIN, HUB_ADDR, seq.next(), &hub_init_payload(0));
    contract.submit_vaas(vec![vaa]).unwrap();

    let rogue_on_hub: [u8; 32] = [0xCC; 32];

    // Rogue on hub chain tries to register hub → blocked because hub hasn't pre-registered it
    let vaa = build_signed_vaa(
        &wh,
        HUB_CHAIN,
        rogue_on_hub,
        seq.next(),
        &registration_payload(HUB_CHAIN, HUB_ADDR),
    );
    let err = contract
        .submit_vaas(vec![vaa])
        .expect_err("rogue should not inherit hub");
    assert!(
        err.root_cause()
            .to_string()
            .contains("hub has not registered this transceiver as a peer"),
        "unexpected error: {err}"
    );
}
