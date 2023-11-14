#![allow(unused_mut)]
//#![allow(unused_imports)]
//#![allow(unused_variables)]
//#![allow(dead_code)]

use {
    near_contract_standards::non_fungible_token::{
        metadata::{
            NFTContractMetadata,
            TokenMetadata,
            NFT_METADATA_SPEC,
        },
        Token,
        TokenId,
    },
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
        ext_contract,
        json_types::{
            Base64VecU8,
        },
        near_bindgen,
        utils::{
            assert_one_yocto,
            is_promise_success,
        },
        AccountId,
        Balance,
        Gas,
        PanicOnDefault,
        Promise,
        PromiseError,
        PromiseOrValue,
        PublicKey,
    },
    std::str,
};

pub mod byte_utils;
pub mod state;

use std::cmp::max;

use crate::byte_utils::{
    get_string_from_32,
    ByteUtils,
};

const CHAIN_ID_NEAR: u16 = 15;
const CHAIN_ID_SOL: u16 = 1;

const BRIDGE_NFT_BINARY: &[u8] =
    include_bytes!("../../nft-wrapped/target/wasm32-unknown-unknown/release/near_nft.wasm");

/// Initial balance for the BridgeToken contract to cover storage and related.
const TRANSFER_BUFFER: u128 = 2000;

#[ext_contract(ext_nft_contract)]
pub trait NFTContract {
    fn new(owner_id: AccountId, metadata: NFTContractMetadata, seq_number: u64) -> Self;
    fn nft_transfer(
        &mut self,
        receiver_id: AccountId,
        token_id: TokenId,
        approval_id: Option<u64>,
        memo: Option<String>,
    );

    fn update_ft(&mut self, owner_id: AccountId, metadata: NFTContractMetadata, seq_number: u64);
    fn nft_token(&self, token_id: TokenId) -> Option<Token>;
    fn nft_mint(
        &mut self,
        token_id: TokenId,
        token_owner_id: AccountId,
        token_metadata: TokenMetadata,
        refund_to: AccountId,
    ) -> Token;
    fn nft_burn(&mut self, token_id: TokenId, from: AccountId, refund_to: AccountId) -> Promise;
}

#[ext_contract(ext_token_bridge)]
pub trait ExtTokenBridge {
    fn finish_deploy(&self, token: String, token_id: String);
}

#[ext_contract(ext_worm_hole)]
pub trait Wormhole {
    fn verify_vaa(&self, vaa: String) -> u32;
    fn publish_message(&self, data: String, nonce: u32) -> u64;
}

#[derive(BorshDeserialize, BorshSerialize, PanicOnDefault)]
pub struct TokenData {
    meta:    String,
    address: String,
    chain:   u16,
}

#[near_bindgen]
#[derive(BorshDeserialize, BorshSerialize)]
pub struct NFTBridgeOld {
    booted:               bool,
    core:                 AccountId,
    dups:                 UnorderedSet<Vec<u8>>,
    owner_pk:             PublicKey,
    emitter_registration: LookupMap<u16, Vec<u8>>,
    last_asset:           u32,
    upgrade_hash:         Vec<u8>,

    tokens:    LookupMap<AccountId, TokenData>,
    key_map:   LookupMap<Vec<u8>, AccountId>,
    hash_map:  LookupMap<Vec<u8>, AccountId>,
    token_map: LookupMap<Vec<u8>, String>,
}

#[near_bindgen]
#[derive(BorshDeserialize, BorshSerialize)]
pub struct NFTBridge {
    booted:               bool,
    core:                 AccountId,
    gov_idx:              u32,
    dups:                 LookupMap<Vec<u8>, bool>,
    owner_pk:             PublicKey,
    emitter_registration: LookupMap<u16, Vec<u8>>,
    last_asset:           u32,
    upgrade_hash:         Vec<u8>,

    tokens:    LookupMap<AccountId, TokenData>,
    key_map:   LookupMap<Vec<u8>, AccountId>,
    hash_map:  LookupMap<Vec<u8>, AccountId>,
    token_map: LookupMap<Vec<u8>, String>,

    bank:      LookupMap<AccountId, Balance>,
}

impl Default for NFTBridge {
    fn default() -> Self {
        Self {
            booted:               false,
            core:                 AccountId::new_unchecked("".to_string()),
            gov_idx:              0,
            dups:                 LookupMap::new(b"d".to_vec()),
            owner_pk:             env::signer_account_pk(),
            emitter_registration: LookupMap::new(b"c".to_vec()),
            last_asset:           0,
            upgrade_hash:         b"".to_vec(),

            tokens:    LookupMap::new(b"ta".to_vec()),
            key_map:   LookupMap::new(b"k".to_vec()),
            hash_map:  LookupMap::new(b"a".to_vec()),
            token_map: LookupMap::new(b"tm".to_vec()),

            bank: LookupMap::new(b"b".to_vec()),
        }
    }
}

fn vaa_register_chain(
    storage: &mut NFTBridge,
    vaa: &state::ParsedVAA,
    mut deposit: Balance,
    refund_to: &AccountId,
) -> Balance {
    let data: &[u8] = &vaa.payload;
    let target_chain = data.get_u16(33);
    let chain = data.get_u16(35);

    if (target_chain != CHAIN_ID_NEAR) && (target_chain != 0) {
        refund_and_panic("InvalidREegisterChainChain", refund_to);
    }

    if storage.emitter_registration.contains_key(&chain) {
        refund_and_panic("DuplicateChainRegistration", refund_to);
    }

    let storage_used = env::storage_usage();
    storage
        .emitter_registration
        .insert(&chain, &data[37..69].to_vec());
    let required_cost = (Balance::from(env::storage_usage()) - Balance::from(storage_used))
        * env::storage_byte_cost();

    if required_cost > deposit {
        refund_and_panic("DepositUnderflowForRegistration", refund_to);
    }
    deposit -= required_cost;

    env::log_str(&format!(
        "register chain {} to {}",
        chain,
        hex::encode(&data[37..69])
    ));

    deposit
}

fn vaa_upgrade_contract(
    storage: &mut NFTBridge,
    vaa: &state::ParsedVAA,
    deposit: Balance,
    refund_to: &AccountId,
) -> Balance {
    let data: &[u8] = &vaa.payload;
    let chain = data.get_u16(33);
    if chain != CHAIN_ID_NEAR {
        refund_and_panic("InvalidContractUpgradeChain", refund_to);
    }

    let uh = data.get_bytes32(0);
    env::log_str(&format!(
        "nft-bridge/{}#{}: vaa_update_contract: {}",
        file!(),
        line!(),
        hex::encode(&uh)
    ));
    storage.upgrade_hash = uh.to_vec(); // Too lazy to do proper accounting here...
    deposit
}

fn vaa_governance(
    storage: &mut NFTBridge,
    vaa: &state::ParsedVAA,
    gov_idx: u32,
    deposit: Balance,
    refund_to: &AccountId,
) -> Balance {
    if gov_idx != vaa.guardian_set_index {
        refund_and_panic("InvalidGovernanceSet", refund_to);
    }

    if (CHAIN_ID_SOL != vaa.emitter_chain)
        || (hex::decode("0000000000000000000000000000000000000000000000000000000000000004")
            .unwrap()
            != vaa.emitter_address)
    {
        refund_and_panic("InvalidGovernanceEmitter", refund_to);
    }

    let data: &[u8] = &vaa.payload;
    let action = data.get_u8(32);

    match action {
        1u8 => vaa_register_chain(storage, vaa, deposit, refund_to),
        2u8 => vaa_upgrade_contract(storage, vaa, deposit, refund_to),
        _ => refund_and_panic("InvalidGovernanceAction", refund_to),
    }
}

fn vaa_transfer(
    storage: &mut NFTBridge,
    vaa: &state::ParsedVAA,
    _action: u8,
    mut deposit: Balance,
    refund_to: AccountId,
) -> PromiseOrValue<bool> {
    let data: &[u8] = &vaa.payload;

    let mut offset: usize = 1; // offset into data in bytes
    let nft_address = data.get_const_bytes::<32>(offset).to_vec();
    offset += 32;
    let nft_chain = data.get_u16(offset);
    offset += 2;
    let symbol = data.get_const_bytes::<32>(offset);
    offset += 32;
    let name = data.get_const_bytes::<32>(offset);
    offset += 32;

    let token_id_vec = data.get_const_bytes::<32>(offset).to_vec();
    let token_id = hex::encode(token_id_vec.clone());

    offset += 32;
    let uri_length: usize = data.get_u8(offset).into();
    offset += 1;
    let uri = data.get_bytes(offset, uri_length).to_vec();
    offset += uri_length;
    let recipient = data.get_const_bytes::<32>(offset).to_vec();
    offset += 32;
    let recipient_chain = data.get_u16(offset);

    if recipient_chain != CHAIN_ID_NEAR {
        refund_and_panic("Not directed at this chain", &refund_to);
    }

    if !storage.hash_map.contains_key(&recipient) {
        refund_and_panic("ReceipientNotRegistered", &refund_to);
    }

    let recipient_account = storage.hash_map.get(&recipient).unwrap();

    env::log_str(&format!(
        "nft-bridge/{}#{}: {}",
        file!(),
        line!(),
        hex::encode(recipient)
    ));

    let bridge_token_account;

    let mut prom = if nft_chain == CHAIN_ID_NEAR {
        if !storage.hash_map.contains_key(&nft_address) {
            refund_and_panic("ReceipientNotRegistered", &refund_to);
        }

        deposit -= 1;

        let bridge_token_account_id = storage.hash_map.get(&nft_address).unwrap();
        bridge_token_account = bridge_token_account_id.to_string();

        ext_nft_contract::ext(bridge_token_account_id)
            .with_attached_deposit(1)
            .nft_transfer(recipient_account, token_id.clone(), None, None)
    } else {
        // The land of Wormhole assets
        let tkey = nft_key(nft_address.clone(), nft_chain);

        let base_uri = String::from_utf8(uri).unwrap();

        let reference = hex::encode(&vaa.payload);

        let storage_used = env::storage_usage();
        let token_key = [tkey.clone(), token_id_vec].concat();
        storage.token_map.insert(&token_key, &reference);

        let storage_used_now = env::storage_usage();
        let delta = (storage_used_now - storage_used) as u128 * env::storage_byte_cost();

        if delta > deposit {
            env::log_str(&format!(
                "nft-bridge/{}#{}: vaa_trnsfer: delta: {} bytes: {}  needed: {}",
                file!(),
                line!(),
                delta,
                storage_used_now - storage_used,
                ((deposit - delta) / env::storage_byte_cost())
            ));
            refund_and_panic("PrecheckFailedDepositUnderFlow", &refund_to);
        }
        deposit -= delta;

        let n = get_string_from_32(&name);
        let s = get_string_from_32(&symbol);

        let md = TokenMetadata {
            title:          Some(n.clone()),
            description:    Some(s.clone()),
            media:          Some(base_uri.clone()),
            media_hash:     Some(Base64VecU8::from(env::sha256(base_uri.as_bytes()))),
            copies:         Some(1u64),
            issued_at:      None,
            expires_at:     None,
            starts_at:      None,
            updated_at:     None,
            extra:          None,
            reference:      None,
            reference_hash: None,
        };

        if storage.key_map.contains_key(&tkey) {
            let dep = deposit;
            deposit = 0;

            let bridge_token_account_id = storage.key_map.get(&tkey).unwrap();
            bridge_token_account = bridge_token_account_id.to_string();

            ext_nft_contract::ext(bridge_token_account_id)
                .with_attached_deposit(dep)
                .nft_mint(token_id.clone(), recipient_account, md, refund_to.clone())
        } else {
            let ft = NFTContractMetadata {
                spec:           NFT_METADATA_SPEC.to_string(),
                name:           n.clone() + " (wormhole)",
                symbol:         s.clone(),
                icon:           None,
                base_uri:       None,
                reference:      None,
                reference_hash: None,
            };

            let storage_used = env::storage_usage();
            storage.last_asset += 1;
            let asset_id = storage.last_asset;
            bridge_token_account = format!("{}.{}", asset_id, env::current_account_id());
            let bridge_token_account_id: AccountId =
                AccountId::new_unchecked(bridge_token_account.clone());

            let d = TokenData {
                meta:    reference,
                address: hex::encode(nft_address),
                chain:   nft_chain,
            };

            storage.tokens.insert(&bridge_token_account_id, &d);
            storage.key_map.insert(&tkey, &bridge_token_account_id);
            storage.hash_map.insert(
                &env::sha256(bridge_token_account.as_bytes()),
                &bridge_token_account_id,
            );

            let storage_used_now = env::storage_usage();

            let delta = (storage_used_now - storage_used) as u128 * env::storage_byte_cost();

            let cost = ((TRANSFER_BUFFER * 2) + BRIDGE_NFT_BINARY.len() as u128)
                * env::storage_byte_cost();
            if cost + delta > deposit {
                env::log_str(&format!(
                    "nft-bridge/{}#{}: vaa_trnsfer: cost: {} delta: {} bytes: {}  needed: {}",
                    file!(),
                    line!(),
                    cost,
                    delta,
                    storage_used_now - storage_used,
                    ((deposit - (cost + delta)) / env::storage_byte_cost())
                ));
                refund_and_panic("PrecheckFailedDepositUnderFlow", &refund_to);
            }

            deposit -= cost + delta;

            let dep = deposit;
            deposit = 0;

            Promise::new(bridge_token_account_id.clone())
                .create_account()
                .transfer(cost)
                .add_full_access_key(storage.owner_pk.clone())
                .deploy_contract(BRIDGE_NFT_BINARY.to_vec())
                // Lets initialize it with useful stuff
                .then(ext_nft_contract::ext(bridge_token_account_id.clone()).new(
                    env::current_account_id(),
                    ft,
                    vaa.sequence,
                ))
                .then(
                    ext_nft_contract::ext(bridge_token_account_id)
                        .with_attached_deposit(dep)
                        .nft_mint(token_id.clone(), recipient_account, md, refund_to.clone()),
                )
        }
    };

    if deposit > 0 {
        env::log_str(&format!(
            "nft-bridge/{}#{}: refund {} to {}",
            file!(),
            line!(),
            deposit,
            env::predecessor_account_id()
        ));
        prom = prom.then(Promise::new(refund_to).transfer(deposit));
    }

    PromiseOrValue::Promise(prom.then(
        ext_token_bridge::ext(env::current_account_id()).finish_deploy(bridge_token_account, token_id),
    ))
}

fn refund_and_panic(s: &str, refund_to: &AccountId) -> ! {
    if env::attached_deposit() > 0 {
        env::log_str(&format!(
            "nft-bridge/{}#{}: refund {} to {}",
            file!(),
            line!(),
            env::attached_deposit(),
            refund_to
        ));
        Promise::new(refund_to.clone()).transfer(env::attached_deposit());
    }
    env::panic_str(s);
}

fn nft_key(address: Vec<u8>, chain: u16) -> Vec<u8> {
    [address, chain.to_be_bytes().to_vec()].concat()
}

#[near_bindgen]
impl NFTBridge {
    pub fn emitter(&self) -> (String, String) {
        let acct = env::current_account_id();
        let astr = acct.to_string();

        (astr.clone(), hex::encode(env::sha256(astr.as_bytes())))
    }

    pub fn is_wormhole(&self, token: &String) -> bool {
        let astr = format!(".{}", env::current_account_id().as_str());
        token.ends_with(&astr)
    }

    pub fn deposit_estimates(&self) -> (String, String) {
        // This is a worst case if we have to store a lot of data as well as create a new account
        let cost =
            ((TRANSFER_BUFFER * 5) + BRIDGE_NFT_BINARY.len() as u128) * env::storage_byte_cost();

        (env::storage_byte_cost().to_string(), cost.to_string())
    }

    pub fn get_original_asset(&self, token: String) -> (String, u16) {
        let account = AccountId::new_unchecked(token);

        if !self.tokens.contains_key(&account) {
            env::panic_str("UnknownAssetId");
        }

        let t = self.tokens.get(&account).unwrap();
        (t.address, t.chain)
    }

    pub fn get_foreign_asset(&self, address: String, chain: u16) -> String {
        let p = nft_key(hex::decode(address).unwrap(), chain);

        if self.key_map.contains_key(&p) {
            return self.key_map.get(&p).unwrap().to_string();
        }

        "".to_string()
    }

    #[payable]
    pub fn register_account(&mut self, account: String) -> String {
        let old_storage_cost = env::storage_usage() as u128 * env::storage_byte_cost() as u128;

        let account_hash = env::sha256(account.as_bytes());
        let ret = hex::encode(&account_hash);

        if self.hash_map.contains_key(&account_hash) {
            Promise::new(env::predecessor_account_id()).transfer(env::attached_deposit());
            return ret;
        }
        let a = AccountId::new_unchecked(account);
        self.hash_map.insert(&account_hash, &a);

        let new_storage_cost = env::storage_usage() as u128 * env::storage_byte_cost() as u128;
        if new_storage_cost <= old_storage_cost {
            refund_and_panic("ImpossibleStorageCost", &env::predecessor_account_id());
        }

        if (new_storage_cost - old_storage_cost) > env::attached_deposit() {
            refund_and_panic("InvalidStorageDeposit", &env::predecessor_account_id());
        }

        let refund = env::attached_deposit() - (new_storage_cost - old_storage_cost);
        if refund > 0 {
            Promise::new(env::predecessor_account_id()).transfer(refund);
        }

        ret
    }

    pub fn hash_account(&self, account: String) -> (bool, String) {
        // Yes, you could hash it yourself but then you wouldn't know
        // if it was already registered...
        let account_hash = env::sha256(account.as_bytes());
        let ret = hex::encode(&account_hash);
        (self.hash_map.contains_key(&account_hash), ret)
    }

    pub fn hash_lookup(&self, hash: String) -> (bool, String) {
        let account_hash = hex::decode(&hash).unwrap();
        if self.hash_map.contains_key(&account_hash) {
            (true, self.hash_map.get(&account_hash).unwrap().to_string())
        } else {
            (false, "".to_string())
        }
    }

    pub fn is_transfer_completed(&self, vaa: String) -> (bool, bool) {
        let h = hex::decode(vaa).expect("invalidVaa");
        let pvaa = state::ParsedVAA::parse(&h);

        if self.dups.contains_key(&pvaa.hash) {
            (true, self.dups.get(&pvaa.hash).unwrap())
        } else {
            (false, false)
        }
    }

    #[payable]
    pub fn submit_vaa(
        &mut self,
        vaa: String,
        mut refund_to: Option<AccountId>,
    ) -> PromiseOrValue<bool> {
        if refund_to == None {
            refund_to = Some(env::predecessor_account_id());
        }

        if env::prepaid_gas() < Gas(300_000_000_000_000) {
            env::panic_str("NotEnoughGas");
        }

        if env::attached_deposit() < (TRANSFER_BUFFER * env::storage_byte_cost()) {
            env::panic_str("StorageDepositUnderflow");
        }

        let h = hex::decode(&vaa).unwrap();
        let pvaa = state::ParsedVAA::parse(&h);

        if pvaa.version != 1 {
            env::panic_str("invalidVersion");
        }

        // Check if VAA with this hash was already accepted
        if self.dups.contains_key(&pvaa.hash) {
            let e = self.dups.get(&pvaa.hash).unwrap();
            if e {
                env::panic_str("alreadyExecuted");
            } else {
                self.dups.insert(&pvaa.hash, &true);
                self.submit_vaa_work(&pvaa, refund_to.unwrap())
            }
        } else {
            let r = refund_to.unwrap();
            PromiseOrValue::Promise(
                ext_worm_hole::ext(self.core.clone())
                    .verify_vaa(vaa.clone())
                    .then(
                        Self::ext(env::current_account_id())
                            .with_unused_gas_weight(10)
                            .with_attached_deposit(env::attached_deposit())
                            .verify_vaa_callback(pvaa.hash, r.clone()),
                    )
                    .then(
                        Self::ext(env::current_account_id()).refunder(r, env::attached_deposit()),
                    ),
            )
        }
    }

    #[private]
    pub fn refunder(&mut self, refund_to: AccountId, amt: Balance) {
        if !is_promise_success() {
            env::log_str(&format!(
                "nft-bridge/{}#{}: refunding {} to {}?",
                file!(),
                line!(),
                amt,
                refund_to
            ));
            Promise::new(refund_to).transfer(amt);
        }
    }


    #[private] // So, all of wormhole security rests in this one statement?
    #[payable]
    pub fn verify_vaa_callback(
        &mut self,
        hash: Vec<u8>,
        refund_to: AccountId,
        #[callback_result] gov_idx: Result<u32, PromiseError>,
    ) -> Promise {
        if gov_idx.is_err() {
            env::panic_str("vaaVerifyFail");
        }
        self.gov_idx = gov_idx.unwrap();

        // Check if VAA with this hash was already accepted
        if self.dups.contains_key(&hash) {
            env::panic_str("alreadyExecuted2");
        }

        let storage_used = env::storage_usage();
        let mut deposit = env::attached_deposit();

        self.dups.insert(&hash, &false);

        let required_cost =
            (Balance::from(env::storage_usage() - storage_used)) * env::storage_byte_cost();
        if required_cost > deposit {
            env::panic_str("DepositUnderflowForHash");
        }
        deposit -= required_cost;

        env::log_str(&format!(
            "nft-bridge/{}#{}: refunding {} to {}?",
            file!(),
            line!(),
            deposit,
            refund_to
        ));
        Promise::new(refund_to).transfer(deposit)
    }

    #[private] // So, all of wormhole security rests in this one statement?
    #[payable]
    fn submit_vaa_work(
        &mut self,
        pvaa: &state::ParsedVAA,
        refund_to: AccountId,
    ) -> PromiseOrValue<bool> {
        env::log_str(&format!(
            "nft-bridge/{}#{}: submit_vaa_callback: {}  {} used: {}  prepaid: {}",
            file!(),
            line!(),
            env::attached_deposit(),
            env::predecessor_account_id(),
            serde_json::to_string(&env::used_gas()).unwrap(),
            serde_json::to_string(&env::prepaid_gas()).unwrap()
        ));

        if pvaa.version != 1 {
            env::panic_str("invalidVersion");
        }

        let data: &[u8] = &pvaa.payload;

        let governance = data[0..32]
            == hex::decode("00000000000000000000000000000000000000000000004e4654427269646765")
                .unwrap();
        let action = data.get_u8(0);

        let deposit = env::attached_deposit();

        if governance {
            let bal = vaa_governance(self, pvaa, self.gov_idx, deposit, &refund_to);
            if bal > 0 {
                env::log_str(&format!(
                    "nft-bridge/{}#{}: refunding {} to {}",
                    file!(),
                    line!(),
                    bal,
                    refund_to
                ));

                return PromiseOrValue::Promise(Promise::new(refund_to).transfer(bal));
            }
            return PromiseOrValue::Value(true);
        }

        if !self.emitter_registration.contains_key(&pvaa.emitter_chain) {
            env::log_str(&format!(
                "nft-bridge/{}#{}: Chain Not Registered: {}",
                file!(),
                line!(),
                pvaa.emitter_chain
            ));

            refund_and_panic("ChainNotRegistered", &refund_to);
        }

        let ce = self.emitter_registration.get(&pvaa.emitter_chain).unwrap();
        if ce != pvaa.emitter_address {
            refund_and_panic("InvalidRegistration", &refund_to);
        }

        if action == 1u8 {
            vaa_transfer(self, &pvaa, action, deposit, refund_to)
        } else {
            refund_and_panic("invalidPortAction", &refund_to);
        }
    }

    #[private]
    pub fn finish_deploy(&mut self, token: String, token_id: String) -> (String, String) {
        if is_promise_success() {
            (token, token_id)
        } else {
            env::panic_str("bad deploy");
        }
    }

    pub fn boot_portal(&mut self, core: String) {
        if self.owner_pk != env::signer_account_pk() {
            env::panic_str("invalidSigner");
        }

        if self.booted {
            env::panic_str("NoDonut");
        }
        self.booted = true;
        self.core = AccountId::try_from(core).unwrap();

        let account_hash = env::sha256(env::current_account_id().to_string().as_bytes());
        env::log_str(&format!("nft emitter: {}", hex::encode(account_hash)));
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
        let delta = max(0, delta);

        let refund = attached_deposit as i128 - delta;
        if refund > 0 {
            env::log_str(&format!(
                "nft-bridge/{}#{}: update_contract_done: refund {} to {}",
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
        let s = env::sha256(&v);

        env::log_str(&format!(
            "nft-bridge/{}#{}: update_contract: {}",
            file!(),
            line!(),
            hex::encode(&s)
        ));

        if s.to_vec() != self.upgrade_hash {
            env::panic_str("invalidUpgradeContract");
        }

        let storage_cost = ((v.len() + 32) as Balance) * env::storage_byte_cost();
        assert!(
            env::attached_deposit() >= storage_cost,
            "DepositUnderFlow:{}",
            storage_cost
        );

        Promise::new(env::current_account_id())
            .deploy_contract(v.to_vec())
            .then(Self::ext(env::current_account_id()).update_contract_done(
                env::predecessor_account_id(),
                env::storage_usage(),
                env::attached_deposit(),
            ))
    }

    //#[allow(clippy::too_many_arguments)]
    #[payable]
    pub fn initiate_transfer(
        &mut self,
        asset: AccountId,
        token_id: TokenId,
        recipient_chain: u16,
        recipient: String,
        nonce: u32,
    ) -> Promise {
        assert_one_yocto();

        if env::prepaid_gas() < Gas(300_000_000_000_000) {
            refund_and_panic("NotEnoughGas", &env::predecessor_account_id());
        }

        if !self.tokens.contains_key(&asset) {
            refund_and_panic("UnknownWormholeAsset", &env::predecessor_account_id());
        }

        let td = self.tokens.get(&asset).unwrap();

        let token_key = [
            hex::decode(td.address).unwrap(),
            td.chain.to_be_bytes().to_vec(),
            hex::decode(&token_id).unwrap(),
        ]
        .concat();
        if !self.token_map.contains_key(&token_key) {
            refund_and_panic("CannotFindMetaDataForToken", &env::predecessor_account_id());
        }

        let meta = self.token_map.get(&token_key).unwrap();

        let astr = format!(".{}", env::current_account_id().as_str());
        if asset.to_string().ends_with(&astr) {
            ext_nft_contract::ext(asset.clone())
                .nft_burn(
                    token_id.clone(),
                    env::predecessor_account_id(),
                    env::predecessor_account_id(),
                )
                .then(
                    Self::ext(env::current_account_id())
                        .with_unused_gas_weight(10)
                        .initiate_transfer_wormhole(
                            asset,
                            token_id,
                            recipient_chain,
                            recipient,
                            nonce,
                            meta,
                            env::predecessor_account_id(),
                        ),
                )
        } else {
            refund_and_panic("NativeNFTsUseDifferentAPI", &env::predecessor_account_id());
        }
    }

    #[private]
    pub fn initiate_transfer_wormhole(
        &mut self,
        _asset: AccountId,
        _token_id: TokenId,
        recipient_chain: u16,
        recipient: String,
        _nonce: u32,
        meta: String,
        _caller: AccountId,
    ) -> Promise {
        if !is_promise_success() {
            env::panic_str("Failed to burn NFT");
        }

        let old = hex::decode(meta).unwrap();

        let p = [
            old[0..(old.len() - 34)].to_vec(),
            vec![0; (64 - recipient.len()) / 2],
            hex::decode(recipient).unwrap(),
            (recipient_chain as u16).to_be_bytes().to_vec(),
        ]
        .concat();

        if old.len() != p.len() {
            refund_and_panic("formatting error", &env::predecessor_account_id());
        }

        ext_worm_hole::ext(self.core.clone())
            .publish_message(hex::encode(p), env::block_height() as u32)
    }
}

//  let result = await userAccount.functionCall({
//    contractId: config.tokenAccount,
//    methodName: "update_contract",
//    args: wormholeContract,
//    attachedDeposit: "12500000000000000000000",
//    gas: 300000000000000,
//  });

#[no_mangle]
pub extern "C" fn update_contract() {
    env::setup_panic_hook();
    let mut contract: NFTBridge = env::state_read().expect("Contract is not initialized");
    contract.update_contract_work(env::input().unwrap());
}
