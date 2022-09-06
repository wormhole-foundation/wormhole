use near_contract_standards::fungible_token::metadata::{
    FungibleTokenMetadata, FungibleTokenMetadataProvider, FT_METADATA_SPEC
};

use near_contract_standards::fungible_token::FungibleToken;
use near_sdk::collections::LazyOption;
use near_sdk::json_types::{U128};

use near_sdk::borsh::{self, BorshDeserialize, BorshSerialize};
use near_sdk::{
    env, near_bindgen, AccountId, PanicOnDefault,
    PromiseOrValue, StorageUsage,
};

#[near_bindgen]
#[derive(BorshDeserialize, BorshSerialize, PanicOnDefault)]
pub struct MockFTContract {
    token: FungibleToken,
    meta: LazyOption<FungibleTokenMetadata>,
    controller: AccountId
}

#[near_bindgen]
impl MockFTContract {
    #[init]
    pub fn new() -> Self {
        assert!(!env::state_exists(), "Already initialized");

        let name = "MockFT".to_string();

        let metadata = FungibleTokenMetadata {
            spec: FT_METADATA_SPEC.to_string(),
            name: name.clone(),
            symbol: name,
            icon: Some("".to_string()), // Is there ANY way to supply this?
            reference: None,
            reference_hash: None,
            decimals: 18,
        };

        Self {
            token: FungibleToken::new(b"ft".to_vec()),
            meta: LazyOption::new(b"md".to_vec(), Some(&metadata)),
            controller: env::predecessor_account_id(),
        }
    }

    #[payable]
    pub fn airdrop(&mut self, a: AccountId, amount: u128) {
        self.storage_deposit(Some(a.clone()), None);
        self.token.internal_deposit(&a, amount);

        near_contract_standards::fungible_token::events::FtMint {
            owner_id: &a,
            amount: &U128::from(amount),
            memo: Some("wormhole mock minted tokens"),
        }
        .emit();
    }

    pub fn account_storage_usage(&self) -> StorageUsage {
        self.token.account_storage_usage
    }
}

near_contract_standards::impl_fungible_token_core!(MockFTContract, token);
near_contract_standards::impl_fungible_token_storage!(MockFTContract, token);

#[near_bindgen]
impl FungibleTokenMetadataProvider for MockFTContract {
    fn ft_metadata(&self) -> FungibleTokenMetadata {
        self.meta.get().unwrap()
    }
}
