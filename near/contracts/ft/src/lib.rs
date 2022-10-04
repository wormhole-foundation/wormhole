use {
    near_contract_standards::fungible_token::{
        metadata::{
            FungibleTokenMetadata,
            FungibleTokenMetadataProvider,
        },
        FungibleToken,
    },
    near_sdk::{
        borsh::{
            self,
            BorshDeserialize,
            BorshSerialize,
        },
        collections::LazyOption,
        env,
        json_types::U128,
        near_bindgen,
        utils::assert_one_yocto,
        AccountId,
        Balance,
        PanicOnDefault,
        Promise,
        PromiseOrValue,
        StorageUsage,
    },
};

const CHAIN_ID_NEAR: u16 = 15;

#[derive(BorshDeserialize, BorshSerialize, PanicOnDefault)]
pub struct FTContractMeta {
    metadata: FungibleTokenMetadata,
    vaa:      Vec<u8>,
    sequence: u64,
}

#[near_bindgen]
#[derive(BorshDeserialize, BorshSerialize, PanicOnDefault)]
pub struct FTContract {
    token:      FungibleToken,
    meta:       LazyOption<FTContractMeta>,
    controller: AccountId,
    hash:       Vec<u8>,
}

pub fn get_string_from_32(v: &[u8]) -> String {
    let s = String::from_utf8_lossy(v);
    s.chars().filter(|c| c != &'\0').collect()
}

#[near_bindgen]
impl FTContract {
    fn on_account_closed(&mut self, account_id: AccountId, balance: Balance) {
        env::log_str(&format!("Closed @{} with {}", account_id, balance));
    }

    fn on_tokens_burned(&mut self, account_id: AccountId, amount: Balance) {
        env::log_str(&format!("Account @{} burned {}", account_id, amount));
    }

    #[init]
    pub fn new(metadata: FungibleTokenMetadata, asset_meta: Vec<u8>, seq_number: u64) -> Self {
        assert!(!env::state_exists(), "Already initialized");

        metadata.assert_valid();

        let meta = FTContractMeta {
            metadata,
            vaa: asset_meta,
            sequence: seq_number,
        };

        let acct = env::current_account_id();
        let astr = acct.to_string();

        Self {
            token:      FungibleToken::new(b"ft".to_vec()),
            meta:       LazyOption::new(b"md".to_vec(), Some(&meta)),
            controller: env::predecessor_account_id(),
            hash:       env::sha256(astr.as_bytes()),
        }
    }

    pub fn update_ft(
        &mut self,
        metadata: FungibleTokenMetadata,
        asset_meta: Vec<u8>,
        seq_number: u64,
    ) {
        if env::predecessor_account_id() != self.controller {
            env::panic_str("CrossContractInvalidCaller");
        }

        if seq_number <= self.meta.get().unwrap().sequence {
            env::panic_str("AssetMetaDataRollback");
        }

        let meta = FTContractMeta {
            metadata,
            vaa: asset_meta,
            sequence: seq_number,
        };

        self.meta.replace(&meta);
    }

    #[payable]
    pub fn wh_burn(&mut self, from: AccountId, amount: u128) {
        assert_one_yocto();

        if env::predecessor_account_id() != self.controller {
            env::panic_str("CrossContractInvalidCaller");
        }

        self.token.internal_withdraw(&from, amount);

        near_contract_standards::fungible_token::events::FtBurn {
            owner_id: &from,
            amount:   &U128::from(amount),
            memo:     Some("Wormhole burn"),
        }
        .emit();
    }

    #[payable]
    pub fn wh_mint(
        &mut self,
        account_id: AccountId,
        refund_to: AccountId,
        amount: u128,
    ) -> Promise {
        if env::predecessor_account_id() != self.controller {
            env::panic_str("CrossContractInvalidCaller");
        }

        let mut deposit: Balance = env::attached_deposit();

        if deposit == 0 {
            env::panic_str("ZeroDepositNotAllowed");
        }

        if !self.token.accounts.contains_key(&account_id) {
            let min_balance = self.storage_balance_bounds().min.0;
            if deposit < min_balance {
                env::panic_str("The attached deposit is less than the minimum storage balance");
            }

            self.token.internal_register_account(&account_id);

            deposit -= min_balance;
        }

        self.token.internal_deposit(&account_id, amount);

        near_contract_standards::fungible_token::events::FtMint {
            owner_id: &account_id,
            amount:   &U128::from(amount),
            memo:     Some("wormhole minted tokens"),
        }
        .emit();

        Promise::new(refund_to).transfer(deposit)
    }

    #[payable]
    pub fn wh_update(&mut self, v: Vec<u8>) -> Promise {
        assert_one_yocto();

        if env::predecessor_account_id() != self.controller {
            env::panic_str("CrossContractInvalidCaller");
        }

        Promise::new(env::current_account_id())
            .deploy_contract(v.to_vec())
    }

    #[payable]
    pub fn vaa_withdraw(
        &mut self,
        from: AccountId,
        amount: u128,
        receiver: String,
        chain: u16,
        fee: u128,
        payload: String,
    ) -> String {
        assert_one_yocto();

        if env::predecessor_account_id() != self.controller {
            env::panic_str("CrossContractInvalidCaller");
        }

        let vaa = self.meta.get().unwrap().vaa;

        if amount > (u64::MAX as u128) || fee > (u64::MAX as u128) {
            env::panic_str("transfer exceeds max bridged token amount");
        }

        if fee >= amount {
            env::panic_str("fee exceeds amount");
        }

        let mut p = [
            // PayloadID uint8 = 1
            (if payload.is_empty() { 1 } else { 3 } as u8)
                .to_be_bytes()
                .to_vec(),
            // Amount uint256
            vec![0; 24],
            (amount as u64).to_be_bytes().to_vec(),
            //TokenAddress bytes32
            vaa[0..32].to_vec(),
            // TokenChain uint16
            vaa[32..34].to_vec(),
            // To bytes32
            vec![0; (64 - receiver.len()) / 2],
            hex::decode(receiver).unwrap(),
            // ToChain uint16
            (chain as u16).to_be_bytes().to_vec(),
        ]
        .concat();

        if payload.is_empty() {
            p = [p, vec![0; 24], (fee as u64).to_be_bytes().to_vec()].concat();
            if p.len() != 133 {
                env::panic_str(&format!("payload1 formatting error  len = {}", p.len()));
            }
        } else {
            if fee != 0 {
                env::panic_str("Payload3 does not support fees");
            }

            let account_hash = env::sha256(from.as_bytes());

            p = [p, account_hash, hex::decode(&payload).unwrap()].concat();
            if p.len() != (133 + (payload.len() / 2)) {
                env::panic_str(&format!("payload3 formatting error  len = {}", p.len()));
            }
        }

        self.token.internal_withdraw(&from, amount);

        near_contract_standards::fungible_token::events::FtBurn {
            owner_id: &from,
            amount:   &U128::from(amount),
            memo:     Some("Wormhole burn"),
        }
        .emit();

        hex::encode(p)
    }

    #[payable]
    pub fn vaa_transfer(
        &mut self,
        amount: u128,
        account_id: AccountId,
        recipient_chain: u16,
        fee: u128,
        refund_to: AccountId,
    ) -> Promise {
        if env::predecessor_account_id() != self.controller {
            env::panic_str("CrossContractInvalidCaller");
        }

        if recipient_chain != CHAIN_ID_NEAR {
            env::panic_str("InvalidRecipientChain");
        }

        if amount == 0 {
            env::panic_str("ZeroAmountWastesGas");
        }

        if amount <= fee {
            env::panic_str("amount <= fee");
        }

        let mut deposit: Balance = env::attached_deposit();

        if deposit == 0 {
            env::panic_str("ZeroDepositNotAllowed");
        }

        if !self.token.accounts.contains_key(&account_id) {
            let min_balance = self.storage_balance_bounds().min.0;
            if deposit < min_balance {
                env::panic_str("The attached deposit is less than the minimum storage balance");
            }

            self.token.internal_register_account(&account_id);

            deposit -= min_balance;
        }

        self.token.internal_deposit(&account_id, amount - fee);

        near_contract_standards::fungible_token::events::FtMint {
            owner_id: &account_id,
            amount:   &U128::from(amount - fee),
            memo:     Some("wormhole minted tokens"),
        }
        .emit();

        if fee != 0 {
            self.token.internal_deposit(&env::signer_account_id(), fee);

            near_contract_standards::fungible_token::events::FtMint {
                owner_id: &env::signer_account_id(),
                amount:   &U128::from(fee),
                memo:     Some("wormhole minted tokens"),
            }
            .emit();
        }

        env::log_str("vaa_transfer called in ft");

        Promise::new(refund_to).transfer(deposit)
    }

    pub fn account_storage_usage(&self) -> StorageUsage {
        self.token.account_storage_usage
    }

    /// Return true if the caller is either controller or self
    pub fn controller_or_self(&self) -> bool {
        let caller = env::predecessor_account_id();
        caller == self.controller || caller == env::current_account_id()
    }
}

near_contract_standards::impl_fungible_token_core!(FTContract, token, on_tokens_burned);
near_contract_standards::impl_fungible_token_storage!(FTContract, token, on_account_closed);

#[near_bindgen]
impl FungibleTokenMetadataProvider for FTContract {
    fn ft_metadata(&self) -> FungibleTokenMetadata {
        self.meta.get().unwrap().metadata
    }
}
