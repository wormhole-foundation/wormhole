use {
    near_sdk::{
        borsh::{
            self,
            BorshDeserialize,
            BorshSerialize,
        },
        collections::{
            LookupMap,
            UnorderedSet,
        },
        env,
        near_bindgen,
        require,
        AccountId,
        Balance,
        Gas,
        Promise,
        PromiseOrValue,
        PublicKey,
    },
    serde::Serialize,
};

pub mod byte_utils;

pub mod state;

use crate::byte_utils::{
    get_string_from_32,
    ByteUtils,
};

const CHAIN_ID_NEAR: u16 = 15;
const CHAIN_ID_SOL: u16 = 1;

#[derive(BorshDeserialize, BorshSerialize)]
pub struct GuardianAddress {
    pub bytes: Vec<u8>,
}

#[derive(BorshDeserialize, BorshSerialize)]
pub struct GuardianSetInfo {
    pub addresses:       Vec<GuardianAddress>,
    pub expiration_time: u64, // Guardian set expiration time
}

impl GuardianSetInfo {
    pub fn quorum(&self) -> usize {
        ((self.addresses.len() * 10 / 3) * 2) / 10 + 1
    }
}

#[must_use]
#[derive(Serialize, Debug, Clone)]
pub struct WormholeEvent {
    standard: String,
    event:    String,
    data:     String,
    nonce:    u32,
    emitter:  String,
    seq:      u64,
    block:    u64,
}

impl WormholeEvent {
    fn to_json_string(&self) -> String {
        // Events cannot fail to serialize so fine to panic on error
        #[allow(clippy::redundant_closure)]
        serde_json::to_string(self)
            .ok()
            .unwrap_or_else(|| env::abort())
    }

    fn to_json_event_string(&self) -> String {
        format!("EVENT_JSON:{}", self.to_json_string())
    }

    /// Logs the event to the host. This is required to ensure that the event is triggered
    /// and to consume the event.
    pub(crate) fn emit(self) {
        near_sdk::env::log_str(&self.to_json_event_string());
    }
}

#[near_bindgen]
#[derive(BorshDeserialize, BorshSerialize)]
pub struct OldWormhole {
    guardians:             LookupMap<u32, GuardianSetInfo>,
    dups:                  UnorderedSet<Vec<u8>>,
    emitters:              LookupMap<String, u64>,
    guardian_set_expirity: u64,
    guardian_set_index:    u32,
    owner_pk:              PublicKey,
    upgrade_hash:          Vec<u8>,
    message_fee:           u128,
    bank:                  u128,
}

#[near_bindgen]
#[derive(BorshDeserialize, BorshSerialize)]
pub struct Wormhole {
    guardians:             LookupMap<u32, GuardianSetInfo>,
    dups:                  UnorderedSet<Vec<u8>>,
    emitters:              LookupMap<String, u64>,
    guardian_set_expirity: u64,
    guardian_set_index:    u32,
    owner_pk:              PublicKey,
    upgrade_hash:          Vec<u8>,
    message_fee:           u128,
    bank:                  u128,
}

impl Default for Wormhole {
    fn default() -> Self {
        Self {
            guardians:             LookupMap::new(b"gs".to_vec()),
            dups:                  UnorderedSet::new(b"d".to_vec()),
            emitters:              LookupMap::new(b"e".to_vec()),
            guardian_set_index:    u32::MAX,
            guardian_set_expirity: 24 * 60 * 60 * 1_000_000_000, // 24 hours in nanoseconds
            owner_pk:              env::signer_account_pk(),
            upgrade_hash:          b"".to_vec(),
            message_fee:           0,
            bank:                  0,
        }
    }
}

// Nothing is mutable...
fn parse_and_verify_vaa(storage: &Wormhole, data: &[u8]) -> state::ParsedVAA {
    let vaa = state::ParsedVAA::parse(data);
    if vaa.version != 1 {
        env::panic_str("InvalidVersion");
    }
    let guardian_set = storage
        .guardians
        .get(&vaa.guardian_set_index)
        .expect("InvalidGuardianSetIndex");

    if guardian_set.expiration_time != 0 && guardian_set.expiration_time < env::block_timestamp() {
        env::panic_str("GuardianSetExpired");
    }

    if (vaa.len_signers as usize) < guardian_set.quorum() {
        env::panic_str("ContractError");
    }

    // Lets calculate the digest that we are comparing against
    let mut pos =
        state::ParsedVAA::HEADER_LEN + (vaa.len_signers * state::ParsedVAA::SIGNATURE_LEN); //  SIGNATURE_LEN: usize = 66;
    let p1 = env::keccak256(&data[pos..]);
    let digest = env::keccak256(&p1);

    // Verify guardian signatures
    let mut last_index: i32 = -1;
    pos = state::ParsedVAA::HEADER_LEN; // HEADER_LEN: usize = 6;

    for _ in 0..vaa.len_signers {
        // which guardian signature is this?
        let index = data.get_u8(pos) as i32;

        // We can't go backwards or use the same guardian over again
        if index <= last_index {
            env::panic_str("WrongGuardianIndexOrder");
        }
        last_index = index;

        pos += 1; // walk forward

        // Grab the whole signature
        let signature = &data[(pos)..(pos + state::ParsedVAA::SIG_DATA_LEN)]; // SIG_DATA_LEN: usize = 64;
        let key = guardian_set.addresses.get(index as usize).unwrap();

        pos += state::ParsedVAA::SIG_DATA_LEN; // SIG_DATA_LEN: usize = 64;
        let recovery = data.get_u8(pos);

        let v = env::ecrecover(&digest, signature, recovery, true).expect("cannot recover key");
        let k = &env::keccak256(&v)[12..32];
        if k != key.bytes {
            env::log_str(&format!(
                "wormhole/{}#{}: signature_error: {} != {}",
                file!(),
                line!(),
                hex::encode(&k),
                hex::encode(&key.bytes),
            ));

            env::panic_str("GuardianSignatureError");
        }
        pos += 1;
    }

    vaa
}

fn vaa_update_contract(
    storage: &mut Wormhole,
    _vaa: &state::ParsedVAA,
    data: &[u8],
    deposit: Balance,
    refund_to: AccountId,
) -> PromiseOrValue<bool> {
    let chain = data.get_u16(33);
    if chain != CHAIN_ID_NEAR {
        env::panic_str("InvalidContractUpgradeChain");
    }

    let uh = data.get_bytes32(0);
    env::log_str(&format!(
        "wormhole/{}#{}: vaa_update_contract: {}",
        file!(),
        line!(),
        hex::encode(&uh)
    ));
    storage.upgrade_hash = uh.to_vec();

    if deposit > 0 {
        PromiseOrValue::Promise(Promise::new(refund_to).transfer(deposit))
    } else {
        PromiseOrValue::Value(true)
    }
}

fn vaa_update_guardian_set(
    storage: &mut Wormhole,
    _vaa: &state::ParsedVAA,
    data: &[u8],
    mut deposit: Balance,
    refund_to: AccountId,
) -> PromiseOrValue<bool> {
    const ADDRESS_LEN: usize = 20;
    let new_guardian_set_index = data.get_u32(0);

    if storage.guardian_set_index + 1 != new_guardian_set_index {
        env::panic_str("InvalidGovernanceSetIndex");
    }

    let n_guardians = data.get_u8(4);

    let mut addresses = vec![];

    for i in 0..n_guardians {
        let pos = 5 + (i as usize) * ADDRESS_LEN;
        addresses.push(GuardianAddress {
            bytes: data[pos..pos + ADDRESS_LEN].to_vec(),
        });
    }

    let guardian_set = &mut storage
        .guardians
        .get(&storage.guardian_set_index)
        .expect("InvalidPreviousGuardianSetIndex");

    guardian_set.expiration_time = env::block_timestamp() + storage.guardian_set_expirity;

    storage
        .guardians
        .insert(&storage.guardian_set_index, &guardian_set);

    let g = GuardianSetInfo {
        addresses,
        expiration_time: 0,
    };

    let storage_used = env::storage_usage();

    storage.guardians.insert(&new_guardian_set_index, &g);
    storage.guardian_set_index = new_guardian_set_index;

    let required_cost =
        (Balance::from(env::storage_usage() - storage_used)) * env::storage_byte_cost();

    if required_cost > deposit {
        env::panic_str("DepositUnderflowForGuardianSet");
    }
    deposit -= required_cost;

    if deposit > 0 {
        PromiseOrValue::Promise(Promise::new(refund_to).transfer(deposit))
    } else {
        PromiseOrValue::Value(true)
    }
}

fn handle_set_fee(
    storage: &mut Wormhole,
    _vaa: &state::ParsedVAA,
    payload: &[u8],
    deposit: Balance,
    refund_to: AccountId,
) -> PromiseOrValue<bool> {
    let (_, amount) = payload.get_u256(0);

    storage.message_fee = amount as u128;

    if deposit > 0 {
        PromiseOrValue::Promise(Promise::new(refund_to).transfer(deposit))
    } else {
        PromiseOrValue::Value(true)
    }
}

fn handle_transfer_fee(
    storage: &mut Wormhole,
    _vaa: &state::ParsedVAA,
    payload: &[u8],
    deposit: Balance,
) -> PromiseOrValue<bool> {
    let (_, amount) = payload.get_u256(0);
    let destination = payload.get_bytes32(32).to_vec();

    if amount > storage.bank {
        env::panic_str("bankUnderFlow");
    }

    // We only support addresses 32 bytes or shorter...  No, we don't
    // support hash addresses in this governance message
    let d = AccountId::new_unchecked(get_string_from_32(&destination));

    if (deposit + amount) > 0 {
        storage.bank -= amount;
        PromiseOrValue::Promise(Promise::new(d).transfer(deposit + amount))
    } else {
        PromiseOrValue::Value(true)
    }
}

#[near_bindgen]
impl Wormhole {
    // I like passing the vaa's as strings around since it will show
    // up better in explorers... I'll let a near sensai talk me out
    // of this...
    pub fn verify_vaa(&self, vaa: String) -> u32 {
        let g1 = env::used_gas();
        let h = hex::decode(vaa).expect("invalidVaa");
        parse_and_verify_vaa(self, &h);
        let g2 = env::used_gas();

        env::log_str(&format!(
            "wormhole/{}#{}: vaa_verify: {}",
            file!(),
            line!(),
            serde_json::to_string(&(g2 - g1)).unwrap()
        ));

        self.guardian_set_index as u32
    }

    #[payable]
    pub fn register_emitter(&mut self, emitter: String) -> PromiseOrValue<bool> {
        let storage_used = env::storage_usage();

        self.emitters.insert(&emitter, &1);

        if env::storage_usage() < storage_used {
            env::panic_str("ImpossibleStorage");
        }

        let required_cost =
            (Balance::from(env::storage_usage() - storage_used)) * env::storage_byte_cost();
        let mut deposit = env::attached_deposit();
        if required_cost > deposit {
            env::panic_str("DepositUnderflowForToken2");
        }

        deposit -= required_cost;

        if deposit > 0 {
            PromiseOrValue::Promise(Promise::new(env::predecessor_account_id()).transfer(deposit))
        } else {
            PromiseOrValue::Value(false)
        }
    }

    #[payable]
    pub fn publish_message(&mut self, data: String, nonce: u32) -> u64 {
        require!(
            env::prepaid_gas() >= Gas(10_000_000_000_000),
            &format!(
                "wormhole/{}#{}: more gas is required {}",
                file!(),
                line!(),
                serde_json::to_string(&env::prepaid_gas()).unwrap()
            )
        );

        require!(
            env::attached_deposit() >= self.message_fee,
            "message_fee not provided"
        );
        self.bank += env::attached_deposit();

        let s = env::predecessor_account_id().to_string();

        if !self.emitters.contains_key(&s) {
            env::panic_str("EmitterNotRegistered");
        }

        let seq = self.emitters.get(&s).unwrap();

        self.emitters.insert(&s, &(seq + 1));

        WormholeEvent {
            standard: "wormhole".to_string(),
            event: "publish".to_string(),
            data,
            nonce,
            emitter: hex::encode(env::sha256(s.as_bytes())),
            seq,
            block: env::block_height(),
        }
        .emit();
        seq
    }

    #[payable]
    pub fn submit_vaa(&mut self, vaa: String) -> PromiseOrValue<bool> {
        let refund_to = env::predecessor_account_id();
        let mut deposit = env::attached_deposit();

        if env::attached_deposit() == 0 {
            env::panic_str("PayForStorage");
        }

        if env::prepaid_gas() < Gas(150_000_000_000_000) {
            env::panic_str("NotEnoughGas");
        }

        let h = hex::decode(vaa).expect("invalidVaa");
        let vaa = parse_and_verify_vaa(self, &h);

        // Check if VAA with this hash was already accepted
        if self.dups.contains(&vaa.hash) {
            env::panic_str("alreadyExecuted");
        }

        let storage_used = env::storage_usage();
        self.dups.insert(&vaa.hash);
        let required_cost =
            (Balance::from(env::storage_usage() - storage_used)) * env::storage_byte_cost();

        if required_cost > deposit {
            env::panic_str("DepositUnderflowForDupSuppression");
        }
        deposit -= required_cost;

        if (CHAIN_ID_SOL != vaa.emitter_chain)
            || (hex::decode("0000000000000000000000000000000000000000000000000000000000000004")
                .unwrap()
                != vaa.emitter_address)
        {
            env::panic_str("InvalidGovernanceEmitter");
        }

        // This is the core contract... it SHOULD only get governance packets and be on the latest

        if self.guardian_set_index != vaa.guardian_set_index {
            env::panic_str("InvalidGovernanceSet");
        }

        let data: &[u8] = &vaa.payload;

        if data[0..32]
            != hex::decode("00000000000000000000000000000000000000000000000000000000436f7265")
                .unwrap()
        {
            env::panic_str("InvalidGovernanceModule");
        }

        let chain = data.get_u16(33);
        let action = data.get_u8(32);

        if !((action == 2 && chain == 0) || chain == CHAIN_ID_NEAR) {
            env::panic_str("InvalidGovernanceChain");
        }

        let payload = &data[35..];

        match action {
            1u8 => vaa_update_contract(self, &vaa, payload, deposit, refund_to.clone()),
            2u8 => vaa_update_guardian_set(self, &vaa, payload, deposit, refund_to.clone()),
            3u8 => handle_set_fee(self, &vaa, payload, deposit, refund_to.clone()),
            4u8 => handle_transfer_fee(self, &vaa, payload, deposit),
            _ => env::panic_str("InvalidGovernanceAction"),
        }
    }

    pub fn message_fee(&self) -> u128 {
        self.message_fee
    }

    pub fn boot_wormhole(&mut self, gset: u32, addresses: Vec<String>) {
        if self.owner_pk != env::signer_account_pk() {
            env::panic_str("invalidSigner");
        }

        assert!(self.guardian_set_index == u32::MAX);

        let addr = addresses
            .iter()
            .map(|address| GuardianAddress {
                bytes: hex::decode(address).unwrap(),
            })
            .collect::<Vec<GuardianAddress>>();

        let g = GuardianSetInfo {
            addresses:       addr,
            expiration_time: 0,
        };
        self.guardians.insert(&gset, &g);
        self.guardian_set_index = gset;
        env::log_str(&format!("Booting guardian_set_index {}", gset));
    }

    #[private]
    pub fn update_contract_done(
        &mut self,
        refund_to: near_sdk::AccountId,
        storage_used: u64,
        attached_deposit: u128,
    ) {
        let delta = (env::storage_usage() as i128 - storage_used as i128)
            * env::storage_byte_cost() as i128;
        let refund = attached_deposit as i128 - delta;
        if refund > 0 {
            env::log_str(&format!(
                "wormhole/{}#{}: update_contract_done: refund {} to {}",
                file!(),
                line!(),
                refund,
                refund_to
            ));
            Promise::new(refund_to).transfer(refund as u128);
        }
    }

    #[private]
    fn update_contract_work(&mut self, v: Vec<u8>) -> Promise {
        if env::attached_deposit() == 0 {
            env::panic_str("attach some cash");
        }

        let s = env::sha256(&v);

        env::log_str(&format!(
            "wormhole/{}#{}: update_contract: {}",
            file!(),
            line!(),
            hex::encode(&s)
        ));

        if s.to_vec() != self.upgrade_hash {
            env::panic_str("invalidUpgradeContract");
        }

        Promise::new(env::current_account_id())
            .deploy_contract(v.to_vec())
            .then(Self::ext(env::current_account_id()).update_contract_done(
                env::predecessor_account_id(),
                env::storage_usage(),
                env::attached_deposit(),
            ))
    }

    #[payable]
    pub fn pass(&mut self) -> bool {
        env::log_str(&format!("wormhole::pass {} {}", file!(), line!()));

        return true;
    }

    #[init(ignore_state)]
    #[payable]
    pub fn migrate() -> Self {
        // call migrate on self
        if env::attached_deposit() != 1 {
            env::panic_str("Need money");
        }
        let old_state: OldWormhole = env::state_read().expect("failed");
        if env::signer_account_pk() != old_state.owner_pk {
            env::panic_str("CannotCallMigrate");
        }
        env::log_str(&format!("wormhole/{}#{}: migrate", file!(), line!(),));
        Self {
            guardians:             old_state.guardians,
            dups:                  old_state.dups,
            emitters:              old_state.emitters,
            guardian_set_expirity: old_state.guardian_set_expirity,
            guardian_set_index:    old_state.guardian_set_index,
            owner_pk:              old_state.owner_pk,
            upgrade_hash:          old_state.upgrade_hash,
            message_fee:           old_state.message_fee,
            bank:                  old_state.bank,
        }
    }
}

//  let result = await userAccount.functionCall({
//    contractId: config.wormholeAccount,
//    methodName: "update_contract",
//    args: await fs.readFileSync("...."),
//    attachedDeposit: "12500000000000000000000",
//    gas: 300000000000000,
//  });

#[no_mangle]
pub extern "C" fn update_contract() {
    env::setup_panic_hook();
    let mut contract: Wormhole = env::state_read().expect("Contract is not initialized");
    contract.update_contract_work(env::input().unwrap());
}
