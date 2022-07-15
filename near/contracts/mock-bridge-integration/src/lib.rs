#![allow(unused_variables)]
#![allow(unused_imports)]

use near_contract_standards::non_fungible_token::metadata::{
    NFTContractMetadata, TokenMetadata, NFT_METADATA_SPEC,
};
use near_contract_standards::non_fungible_token::{Token, TokenId};

use near_sdk::borsh::{self, BorshDeserialize, BorshSerialize};

use near_sdk::json_types::Base64VecU8;

use near_sdk::json_types::U128;
use near_sdk::{env, ext_contract, near_bindgen, AccountId, Balance, Promise, PromiseOrValue};

use near_sdk::utils::is_promise_success;

const BRIDGE_TOKEN_BINARY: &[u8] = include_bytes!(
    "../../mock-bridge-token/target/wasm32-unknown-unknown/release/mock_bridge_token.wasm"
);

const BRIDGE_NFT_BINARY: &[u8] =
    include_bytes!("../../nft-wrapped/target/wasm32-unknown-unknown/release/nft.wasm");

/// Initial balance for the BridgeToken contract to cover storage and related.
const BRIDGE_TOKEN_INIT_BALANCE: Balance = 5_860_000_000_000_000_000_000;

#[ext_contract(ext_ft_contract)]
pub trait MockFtContract {
    fn new() -> Self;
    fn airdrop(&self, a: AccountId, amount: u128);
}

#[ext_contract(ext_wormhole)]
pub trait MockWormhole {
    fn pass(&self) -> bool;
}

#[ext_contract(ext_nft_contract)]
pub trait MockNftContract {
    fn new(owner_id: AccountId, metadata: NFTContractMetadata, seq_number: u64) -> Self;
    fn nft_mint(
        &mut self,
        token_id: TokenId,
        token_owner_id: AccountId,
        token_metadata: TokenMetadata,
        refund_to: AccountId,
    ) -> Token;
}

#[near_bindgen]
#[derive(BorshDeserialize, BorshSerialize)]
pub struct PortalTest {
    cnt: u32,
}

impl Default for PortalTest {
    fn default() -> Self {
        Self { cnt: 0 }
    }
}

#[near_bindgen]
impl PortalTest {
    #[payable]
    pub fn deploy_ft(&mut self, account: String) -> Promise {
        let a = AccountId::try_from(account).unwrap();

        let name = format!("b{}", env::block_height());

        let bridge_token_account = format!("{}.{}", name, env::current_account_id());

        let bridge_token_account_id: AccountId =
            AccountId::new_unchecked(bridge_token_account.clone());

        let v = BRIDGE_TOKEN_BINARY.to_vec();

        Promise::new(bridge_token_account_id.clone())
            .create_account()
            .transfer(BRIDGE_TOKEN_INIT_BALANCE + (v.len() as u128 * env::storage_byte_cost()))
            .add_full_access_key(env::signer_account_pk())
            .deploy_contract(v)
            // Lets initialize it with useful stuff
            .then(ext_ft_contract::ext(bridge_token_account_id.clone()).new())
            .then(
                ext_ft_contract::ext(bridge_token_account_id)
                    .with_attached_deposit(BRIDGE_TOKEN_INIT_BALANCE)
                    .airdrop(a, BRIDGE_TOKEN_INIT_BALANCE),
            )
            // And then lets tell us we are done!
            .then(Self::ext(env::current_account_id()).finish_deploy(bridge_token_account))
    }

    #[payable]
    pub fn chunker(&mut self, s: String) -> Promise {
        self.cnt += 1;

        env::log_str(&format!(
            "mock-bridge-integration/{}#{}: amount: {}  cnt: {}",
            file!(),
            line!(),
            env::attached_deposit(),
            self.cnt
        ));

        Self::ext(env::current_account_id())
            .with_attached_deposit(env::attached_deposit())
            .chunks(s)
            .then(
                Self::ext(env::current_account_id())
                    .refunder(env::predecessor_account_id(), env::attached_deposit()),
            )
    }

    #[private]
    pub fn refunder(&mut self, refund_to: AccountId, amt: Balance) {
        if !is_promise_success() {
            env::log_str(&format!(
                "mock-bridge-integration/{}#{}: refunding {} to {}",
                file!(),
                line!(),
                amt,
                refund_to
            ));
            Promise::new(refund_to).transfer(amt);
        }
    }

    #[payable]
    pub fn chunks(&mut self, s: String) -> Promise {
        self.cnt += 1;

        env::log_str(&format!(
            "mock-bridge-integration/{}#{}: amount: {}  cnt: {}",
            file!(),
            line!(),
            env::attached_deposit(),
            self.cnt
        ));

        ext_wormhole::ext(AccountId::new_unchecked("wormhole.test.near".to_string()))
            .with_attached_deposit(env::attached_deposit())
            .pass()
            .then(Self::ext(env::current_account_id()).thrower(s))
    }

    pub fn thrower(&mut self, s: String) -> Promise {
        self.cnt += 1;

        env::log_str(&format!(
            "mock-bridge-integration/{}#{}: amount: {}  cnt: {}",
            file!(),
            line!(),
            env::attached_deposit(),
            self.cnt
        ));
        env::panic_str(&s);
    }

    #[payable]
    pub fn deploy_nft(&mut self, account: String) -> Promise {
        let a = AccountId::try_from(account).unwrap();

        let bridge_nft_account = format!("b{}.{}", env::block_height(), env::current_account_id());
        let bridge_nft_account_id: AccountId = AccountId::new_unchecked(bridge_nft_account.clone());

        let v = BRIDGE_NFT_BINARY.to_vec();

        let md = NFTContractMetadata {
            spec: NFT_METADATA_SPEC.to_string(),
            name: "RandomNFT".to_string(),
            symbol: "RNFT".to_string(),
            icon: None,
            base_uri: None,
            reference: None,
            reference_hash: None,
        };

        Promise::new(bridge_nft_account_id.clone())
            .create_account()
            .transfer(env::attached_deposit())
            .add_full_access_key(env::signer_account_pk())
            .deploy_contract(v)
            // Lets initialize it with useful stuff
            .then(
                ext_nft_contract::ext(bridge_nft_account_id.clone())
                    .with_unused_gas_weight(3)
                    .new(env::current_account_id(), md, 0),
            )
            .then(Self::ext(env::current_account_id()).finish_deploy(bridge_nft_account))
    }

    #[payable]
    pub fn mint_nft(
        &mut self,
        nft: AccountId,
        token_id: String,
        media: String,
        give_to: AccountId,
    ) -> Promise {
        let md = TokenMetadata {
            title: Some("Phil ".to_string() + &token_id),
            description: Some("George ".to_string() + &token_id),
            media: Some(media.clone()),
            media_hash: Some(Base64VecU8::from(env::sha256(media.as_bytes()))),
            copies: Some(1u64),
            issued_at: None,
            expires_at: None,
            starts_at: None,
            updated_at: None,
            extra: None,
            reference: None,
            reference_hash: None,
        };

        ext_nft_contract::ext(nft)
            .with_attached_deposit(BRIDGE_TOKEN_INIT_BALANCE)
            .nft_mint(token_id, give_to, md, env::current_account_id())
    }

    pub fn ft_on_transfer(
        &mut self,
        sender_id: AccountId,
        amount: U128,
        msg: String,
    ) -> PromiseOrValue<U128> {
        env::log_str(&msg);
        env::panic_str("ft_on_transfer");
    }

    #[private]
    pub fn finish_deploy(&mut self, ret: String) -> String {
        if is_promise_success() {
            ret
        } else {
            env::panic_str("bad deploy");
        }
    }

    pub fn payload3(
        amount: u128,
        token_address: Vec<u8>,
        token_chain: u16,
        fee: u128,
        vaa: String,
    ) {
        env::log_str(&format!(
            "mock-bridge-integration/{}#{}: amount: {}  vaa: {}",
            file!(),
            line!(),
            amount,
            vaa
        ));
    }
}
