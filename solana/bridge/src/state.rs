//! Bridge transition types

use std::mem::size_of;
use std::str;

use num_traits::AsPrimitive;
use solana_sdk::{
    account_info::AccountInfo, account_info::next_account_info, entrypoint::ProgramResult, info,
    program_error::ProgramError, pubkey::bs58, pubkey::Pubkey,
};
use solana_sdk::clock::Clock;
use solana_sdk::hash::hash;
#[cfg(not(target_arch = "bpf"))]
use solana_sdk::instruction::Instruction;
use solana_sdk::log::sol_log;
#[cfg(target_arch = "bpf")]
use solana_sdk::program::invoke_signed;
use solana_sdk::rent::Rent;
use solana_sdk::system_instruction::{create_account, SystemInstruction};
use solana_sdk::sysvar::Sysvar;
use spl_token::state::Mint;

use crate::{
    error::Error,
    instruction::unpack,
};
use crate::instruction::{BridgeInstruction, CHAIN_ID_SOLANA, ForeignAddress, GuardianKey, TransferOutPayload, VAA};
use crate::instruction::BridgeInstruction::*;
use crate::syscalls::{SchnorrifyInput, sol_verify_schnorr};

/// fee rate as a ratio
#[repr(C)]
#[derive(Clone, Copy)]
pub struct Fee {
    /// denominator of the fee ratio
    pub denominator: u64,
    /// numerator of the fee ratio
    pub numerator: u64,
}

/// guardian set
#[repr(C)]
#[derive(Clone, Copy)]
pub struct GuardianSet {
    /// index of the set
    pub index: u32,
    /// public key of the threshold schnorr set
    pub pubkey: GuardianKey,
    /// creation time
    pub creation_time: u32,
    /// expiration time when VAAs issued by this set are no longer valid
    pub expiration_time: u32,

    /// Is `true` if this structure has been initialized.
    pub is_initialized: bool,
}

impl IsInitialized for GuardianSet {
    fn is_initialized(&self) -> bool {
        self.is_initialized
    }
}

/// proposal to transfer tokens to a foreign chain
#[repr(C)]
#[derive(Clone, Copy)]
pub struct TransferOutProposal {
    /// amount to transfer
    pub amount: u64,
    /// chain id to transfer to
    pub to_chain_id: u8,
    /// address on the foreign chain to transfer to
    pub foreign_address: ForeignAddress,
    /// asset that is being transferred
    pub asset: AssetMeta,
    /// vaa to unlock the tokens on the foreign chain
    pub vaa: VAA,
    /// time the vaa was submitted
    pub vaa_time: u32,

    /// Is `true` if this structure has been initialized.
    pub is_initialized: bool,
}

impl IsInitialized for TransferOutProposal {
    fn is_initialized(&self) -> bool {
        self.is_initialized
    }
}

/// record of a claimed VAA
#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct ClaimedVAA {
    /// hash of the vaa
    pub hash: [u8; 32],
    /// time the vaa was submitted
    pub vaa_time: u32,

    /// Is `true` if this structure has been initialized.
    pub is_initialized: bool,
}

impl IsInitialized for ClaimedVAA {
    fn is_initialized(&self) -> bool {
        self.is_initialized
    }
}

/// Metadata about an asset
#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct AssetMeta {
    /// Address of the token
    pub address: ForeignAddress,

    /// Chain of the token
    pub chain: u8,
}

/// Config for a bridge.
#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct BridgeConfig {
    /// Period for how long a VAA is valid. This is also the period after a valid VAA has been
    /// published to a `TransferOutProposal` or `ClaimedVAA` after which the account can be evicted.
    /// This exists to guarantee data availability and prevent replays.
    pub vaa_expiration_time: u32,

    /// Token program that is used for this bridge
    pub token_program: Pubkey,
}

/// Bridge state.
#[repr(C)]
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Bridge {
    /// the currently active guardian set
    pub guardian_set_index: u32,

    /// read-only config parameters for a bridge instance.
    pub config: BridgeConfig,

    /// Is `true` if this structure has been initialized.
    pub is_initialized: bool,
}

impl IsInitialized for Bridge {
    fn is_initialized(&self) -> bool {
        self.is_initialized
    }
}

/// Instruction processing logic
impl Bridge {
    /// Processes an [Instruction](enum.Instruction.html).
    pub fn process(program_id: &Pubkey, accounts: &[AccountInfo], input: &[u8]) -> ProgramResult {
        let instruction = BridgeInstruction::deserialize(input)?;
        match instruction {
            Initialize(payload) => {
                info!("Instruction: Initialize");
                Self::process_initialize(program_id, accounts, payload.initial_guardian, payload.config)
            }
            TransferOut(p) => {
                info!("Instruction: TransferOut");

                if p.asset.chain == CHAIN_ID_SOLANA {
                    Self::process_transfer_native_out(program_id, accounts, &p)
                } else {
                    Self::process_transfer_out(program_id, accounts, &p)
                }
            }
            _ => {
                panic!("")
            }
        }
    }

    /// Unpacks a token state from a bytes buffer while assuring that the state is initialized.
    pub fn process_initialize(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        initial_guardian_key: GuardianKey,
        config: BridgeConfig,
    ) -> ProgramResult {
        let account_info_iter = &mut accounts.iter();
        let new_account_info = next_account_info(account_info_iter)?;
        let new_guardian_info = next_account_info(account_info_iter)?;
        let clock_info = next_account_info(account_info_iter)?;
        let clock = Clock::from_account_info(clock_info)?;

        let mut new_account_data = new_account_info.data.borrow_mut();
        let mut bridge: &mut Bridge = Self::unpack_unchecked(&mut new_account_data)?;
        if bridge.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

        let expected_bridge_key = Bridge::derive_bridge_id(program_id)?;
        if expected_bridge_key != *new_account_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        let expected_guardian_set_key = Bridge::derive_guardian_set_id(program_id, new_account_info.key, 0)?;
        if expected_guardian_set_key != *new_guardian_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        let mut new_guardian_data = new_guardian_info.data.borrow_mut();
        let mut guardian_info: &mut GuardianSet = Self::unpack_unchecked(&mut new_guardian_data)?;
        if guardian_info.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

        // Initialize bridge params
        bridge.is_initialized = true;
        bridge.guardian_set_index = 0;
        bridge.config = config;

        // Initialize the initial guardian set
        guardian_info.is_initialized = true;
        guardian_info.index = 0;
        guardian_info.creation_time = clock.unix_timestamp.as_();
        guardian_info.pubkey = initial_guardian_key;

        Ok(())
    }

    /// Transfers a wrapped asset out
    pub fn process_transfer_out(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        t: &TransferOutPayload,
    ) -> ProgramResult {
        let account_info_iter = &mut accounts.iter();
        let sender_account_info = next_account_info(account_info_iter)?;
        let clock_info = next_account_info(account_info_iter)?;
        let bridge_info = next_account_info(account_info_iter)?;
        let proposal_info = next_account_info(account_info_iter)?;
        let mint_info = next_account_info(account_info_iter)?;
        let sender_info = next_account_info(account_info_iter)?;

        let clock = Clock::from_account_info(clock_info)?;
        let sender = Bridge::token_account_deserialize(sender_account_info)?;
        let bridge = Bridge::bridge_deserialize(bridge_info)?;
        let mint = Bridge::mint_deserialize(mint_info)?;

        // Does the token belong to the mint
        if sender.mint != *mint_info.key {
            return Err(Error::TokenMintMismatch.into());
        }

        // Is the mint owned by the program
        if mint.owner.unwrap() != *program_id {
            return Err(Error::WrongMintOwner.into());
        }

        // Check that the mint is actually a wrapped asset belonging to *this* bridge instance
        let expected_mint_address =
            Bridge::derive_wrapped_asset_id(
                program_id, bridge_info.key, t.asset.chain,
                t.asset.address)?;
        if expected_mint_address != *mint_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Check that the transfer account was derived correctly
        let expected_transfer_id =
            Bridge::derive_transfer_id(
                program_id, bridge_info.key, t.asset.chain,
                t.asset.address, t.chain_id, t.target,
                sender.owner, clock.slot.as_())?;
        if expected_transfer_id != *proposal_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Load proposal account
        let mut proposal_data = proposal_info.data.borrow_mut();
        let proposal: &mut TransferOutProposal =
            Bridge::unpack_unchecked(&mut proposal_data)?;
        if proposal.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

        // Burn tokens
        Bridge::wrapped_burn(accounts, &bridge.config.token_program,
                             sender_info.key, sender_account_info.key, t.amount)?;

        // Initialize proposal
        proposal.is_initialized = true;
        proposal.foreign_address = t.target;
        proposal.amount = t.amount;
        proposal.to_chain_id = t.chain_id;
        proposal.asset = t.asset;

        Ok(())
    }

    /// Transfers a native token to a foreign chain
    pub fn process_transfer_native_out(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        t: &TransferOutPayload,
    ) -> ProgramResult {
        let account_info_iter = &mut accounts.iter();
        let sender_account_info = next_account_info(account_info_iter)?;
        let clock_info = next_account_info(account_info_iter)?;
        let bridge_info = next_account_info(account_info_iter)?;
        let proposal_info = next_account_info(account_info_iter)?;
        let mint_info = next_account_info(account_info_iter)?;
        let custody_info = next_account_info(account_info_iter)?;
        let sender_info = next_account_info(account_info_iter)?;

        let clock = Clock::from_account_info(clock_info)?;
        let sender = Bridge::token_account_deserialize(sender_account_info)?;
        let bridge = Bridge::bridge_deserialize(bridge_info)?;
        let mint = Bridge::mint_deserialize(mint_info)?;

        // Does the token belong to the mint
        if sender.mint != *mint_info.key {
            return Err(Error::TokenMintMismatch.into());
        }

        // If the mint is owned by the program, it's a wrapped asset
        if mint.owner.unwrap() == *program_id {
            return Err(Error::WrongMintOwner.into());
        }

        // Check that the transfer account was derived correctly
        let expected_transfer_id =
            Bridge::derive_transfer_id(
                program_id, bridge_info.key, t.asset.chain,
                t.asset.address, t.chain_id, t.target,
                sender.owner, clock.slot.as_())?;
        if expected_transfer_id != *proposal_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Load proposal account
        let mut proposal_data = proposal_info.data.borrow_mut();
        let proposal: &mut TransferOutProposal =
            Bridge::unpack_unchecked(&mut proposal_data)?;
        if proposal.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

        let custody_addr = Bridge::derive_custody(program_id, bridge_info.key, mint_info.key)?;
        if expected_transfer_id != *custody_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Create the account if it does not exist
        if custody_info.data_len() == 0 {
            Bridge::create_account(accounts, sender_info.key, &custody_addr,
                                   size_of::<spl_token::state::Mint>(), program_id)?;
        } else if custody_info.owner != program_id {
            return Err(Error::WrongTokenAccountOwner.into());
        }

        // Transfer tokens to custody
        Bridge::token_transfer_caller(accounts, &bridge.config.token_program, sender_account_info.key,
                                      &custody_addr, sender_info.key, t.amount)?;

        // Initialize proposal
        proposal.is_initialized = true;
        proposal.foreign_address = t.target;
        proposal.amount = t.amount;
        proposal.to_chain_id = t.chain_id;

        // Don't use the user-given data as we don't check mint = AssetMeta.address
        proposal.asset = AssetMeta {
            chain: CHAIN_ID_SOLANA,
            address: mint_info.key.to_bytes(),
        };

        Ok(())
    }
}

/// Implementation of serialization functions
impl Bridge {
    /// Deserializes a spl_token `Account`.
    pub fn token_account_deserialize(
        info: &AccountInfo,
    ) -> Result<spl_token::state::Account, Error> {
        Ok(
            *spl_token::state::State::unpack(&mut info.data.borrow_mut())
                .map_err(|_| Error::ExpectedAccount)?,
        )
    }

    /// Deserializes a spl_token `Mint`.
    pub fn mint_deserialize(info: &AccountInfo) -> Result<spl_token::state::Mint, Error> {
        Ok(
            *spl_token::state::State::unpack(&mut info.data.borrow_mut())
                .map_err(|_| Error::ExpectedToken)?,
        )
    }

    /// Deserializes a `Bridge`.
    pub fn bridge_deserialize(info: &AccountInfo) -> Result<Bridge, Error> {
        Ok(
            *Bridge::unpack(&mut info.data.borrow_mut())
                .map_err(|_| Error::ExpectedBridge)?,
        )
    }
    /// Unpacks a token state from a bytes buffer while assuring that the state is initialized.
    pub fn unpack<T: IsInitialized>(input: &mut [u8]) -> Result<&mut T, ProgramError> {
        let mut_ref: &mut T = Self::unpack_unchecked(input)?;
        if !mut_ref.is_initialized() {
            return Err(Error::UninitializedState.into());
        }
        Ok(mut_ref)
    }
    /// Unpacks a token state from a bytes buffer without checking that the state is initialized.
    pub fn unpack_unchecked<T: IsInitialized>(input: &mut [u8]) -> Result<&mut T, ProgramError> {
        if input.len() != size_of::<T>() {
            return Err(ProgramError::InvalidAccountData);
        }
        #[allow(clippy::cast_ptr_alignment)]
            Ok(unsafe { &mut *(&mut input[0] as *mut u8 as *mut T) })
    }
}


/// Implementation of actions and derivation
impl Bridge {
    /// Calculates a derived address for this program
    pub fn derive_bridge_id(program_id: &Pubkey) -> Result<Pubkey, Error> {
        Pubkey::create_program_address(&[program_id.to_string().as_str()], program_id)
            .or(Err(Error::InvalidProgramAddress))
    }

    /// Calculates a derived address for a custody account
    pub fn derive_custody(program_id: &Pubkey, bridge: &Pubkey, mint: &Pubkey) -> Result<Pubkey, Error> {
        Pubkey::create_program_address(&["custody", bridge.to_string().as_str(), mint.to_string().as_str()], program_id)
            .or(Err(Error::InvalidProgramAddress))
    }

    /// Calculates a derived address for this program
    pub fn derive_guardian_set_id(program_id: &Pubkey, bridge_key: &Pubkey, guardian_set_index: u32) -> Result<Pubkey, Error> {
        Pubkey::create_program_address(&[
            bridge_key.to_string().as_str(),
            guardian_set_index.to_string().as_str()
        ], program_id)
            .or(Err(Error::InvalidProgramAddress))
    }

    /// Calculates a derived address for this program
    pub fn derive_wrapped_asset_id(program_id: &Pubkey, bridge_key: &Pubkey, asset_chain: u8, asset: ForeignAddress) -> Result<Pubkey, Error> {
        Pubkey::create_program_address(&[
            &"wrapped",
            bridge_key.to_string().as_str(),
            asset_chain.to_string().as_str(),
            bs58::encode(asset).into_string().as_str()
        ], program_id)
            .or(Err(Error::InvalidProgramAddress))
    }

    /// Calculates a derived address for this program
    pub fn derive_transfer_id(program_id: &Pubkey, bridge_key: &Pubkey,
                              asset_chain: u8, asset: ForeignAddress,
                              target_chain: u8, target_address: ForeignAddress,
                              user: Pubkey, slot: u64) -> Result<Pubkey, Error> {
        Pubkey::create_program_address(&[
            &"transfer",
            bridge_key.to_string().as_str(),
            asset_chain.to_string().as_str(),
            bs58::encode(asset).into_string().as_str(),
            target_chain.to_string().as_str(),
            bs58::encode(target_address).into_string().as_str(),
            user.to_string().as_str(),
            slot.to_string().as_str(),
        ], program_id)
            .or(Err(Error::InvalidProgramAddress))
    }

    /// Issue a spl_token `Burn` instruction.
    pub fn wrapped_burn(
        accounts: &[AccountInfo],
        token_program_id: &Pubkey,
        authority: &Pubkey,
        token_account: &Pubkey,
        amount: u64,
    ) -> Result<(), ProgramError> {
        let all_signers: Vec<&Pubkey> = accounts.iter()
            .filter_map(|item| if item.is_signer { Some(item.key) } else { None })
            .collect();
        let ix =
            spl_token::instruction::burn(
                token_program_id,
                token_account,
                authority,
                all_signers.as_slice(),
                amount,
            )?;
        invoke_signed(&ix, accounts, &[])
    }

    /// Issue a spl_token `MintTo` instruction.
    pub fn wrapped_mint_to(
        accounts: &[AccountInfo],
        token_program_id: &Pubkey,
        chain_id: u8,
        asset: [u8; 32],
        mint: &Pubkey,
        destination: &Pubkey,
        authority: &Pubkey,
        amount: u64,
    ) -> Result<(), ProgramError> {
        let chain_str = chain_id.to_string();
        let asset_str = bs58::encode(asset).into_string();
        let signers = &[&["wrapped", chain_str.as_str(), asset_str.as_str()][..]];
        let ix = spl_token::instruction::mint_to(
            token_program_id,
            mint,
            destination,
            authority,
            &[],
            amount,
        )?;
        invoke_signed(&ix, accounts, signers)
    }

    /// Issue a spl_token `Transfer` instruction.
    pub fn token_transfer_caller(
        accounts: &[AccountInfo],
        token_program_id: &Pubkey,
        source: &Pubkey,
        destination: &Pubkey,
        authority: &Pubkey,
        amount: u64,
    ) -> Result<(), ProgramError> {
        let all_signers: Vec<&Pubkey> = accounts.iter()
            .filter_map(|item| if item.is_signer { Some(item.key) } else { None })
            .collect();
        let ix = spl_token::instruction::transfer(
            token_program_id,
            source,
            destination,
            authority,
            all_signers.as_slice(),
            amount,
        )?;
        invoke_signed(&ix, accounts, &[])
    }

    /// Issue a spl_token `Transfer` instruction.
    pub fn token_transfer(
        accounts: &[AccountInfo],
        token_program_id: &Pubkey,
        bridge: &Pubkey,
        source: &Pubkey,
        destination: &Pubkey,
        authority: &Pubkey,
        amount: u64,
    ) -> Result<(), ProgramError> {
        let signers = &[&["wrapped", "kot"][..]];
        let ix = spl_token::instruction::transfer(
            token_program_id,
            source,
            destination,
            authority,
            &[],
            amount,
        )?;
        invoke_signed(&ix, accounts, signers)
    }

    /// Create a new account
    pub fn create_account(
        accounts: &[AccountInfo],
        payer_account: &Pubkey,
        new_address: &Pubkey,
        size: usize,
        owner: &Pubkey,
    ) -> Result<(), ProgramError> {
        let ix = create_account(payer_account, new_address,
                                Rent::default().minimum_balance(size), size.as_(), owner);
        invoke_signed(&ix, accounts, &[])
    }

    /// Create a new account
    pub fn create_token_account(
        accounts: &[AccountInfo],
        token_program: &Pubkey,
        account: &Pubkey,
        mint: &Pubkey,
        owner: &Pubkey,
    ) -> Result<(), ProgramError> {
        let ix = spl_token::instruction::initialize_account(token_program,
                                                            account, mint, owner)?;
        invoke_signed(&ix, accounts, &[])
    }
}

/// Check is a token state is initialized
pub trait IsInitialized {
    /// Is initialized
    fn is_initialized(&self) -> bool;
}

// Test program id for the swap program.
#[cfg(not(target_arch = "bpf"))]
const WORMHOLE_PROGRAM_ID: Pubkey = Pubkey::new_from_array([2u8; 32]);

/// Routes invokes to the token program, used for testing.
#[cfg(not(target_arch = "bpf"))]
pub fn invoke_signed<'a>(
    instruction: &Instruction,
    account_infos: &[AccountInfo<'a>],
    signers_seeds: &[&[&str]],
) -> ProgramResult {
    let mut new_account_infos = vec![];
    for meta in instruction.accounts.iter() {
        for account_info in account_infos.iter() {
            if meta.pubkey == *account_info.key {
                let mut new_account_info = account_info.clone();
                for seeds in signers_seeds.iter() {
                    let signer = Pubkey::create_program_address(seeds, &WORMHOLE_PROGRAM_ID).unwrap();
                    if *account_info.key == signer {
                        new_account_info.is_signer = true;
                    }
                }
                new_account_infos.push(new_account_info);
            }
        }
    }
    spl_token::state::State::process(
        &instruction.program_id,
        &new_account_infos,
        &instruction.data,
    )
}

#[cfg(test)]
mod tests {
    use solana_sdk::{
        account::Account, account_info::create_is_signer_account_infos, instruction::Instruction,
    };
    use spl_token::{
        instruction::{initialize_account, initialize_mint},
        state::{Account as SplAccount, Mint as SplMint, State as SplState},
    };

    use crate::instruction::initialize;

    use super::*;

    const TOKEN_PROGRAM_ID: Pubkey = Pubkey::new_from_array([1u8; 32]);

    // Pulls in the stubs required for `info!()`
    #[cfg(not(target_arch = "bpf"))]
    solana_sdk::program_stubs!();

    fn pubkey_rand() -> Pubkey {
        Pubkey::new(&rand::random::<[u8; 32]>())
    }

    fn do_process_instruction(
        instruction: Instruction,
        accounts: Vec<&mut Account>,
    ) -> ProgramResult {
        let mut meta = instruction
            .accounts
            .iter()
            .zip(accounts)
            .map(|(account_meta, account)| (&account_meta.pubkey, account_meta.is_signer, account))
            .collect::<Vec<_>>();

        let account_infos = create_is_signer_account_infos(&mut meta);
        if instruction.program_id == WORMHOLE_PROGRAM_ID {
            Bridge::process(&instruction.program_id, &account_infos, &instruction.data)
        } else {
            SplState::process(&instruction.program_id, &account_infos, &instruction.data)
        }
    }

    fn mint_token(
        program_id: &Pubkey,
        authority_key: &Pubkey,
        amount: u64,
    ) -> ((Pubkey, Account), (Pubkey, Account)) {
        let token_key = pubkey_rand();
        let mut token_account = Account::new(0, size_of::<SplMint>(), &program_id);
        let account_key = pubkey_rand();
        let mut account_account = Account::new(0, size_of::<SplAccount>(), &program_id);

        // create pool and pool account
        do_process_instruction(
            initialize_account(&program_id, &account_key, &token_key, &authority_key).unwrap(),
            vec![
                &mut account_account,
                &mut Account::default(),
                &mut token_account,
            ],
        )
            .unwrap();
        let mut authority_account = Account::default();
        do_process_instruction(
            initialize_mint(
                &program_id,
                &token_key,
                Some(&account_key),
                Some(&authority_key),
                amount,
                2,
            )
                .unwrap(),
            if amount == 0 {
                vec![&mut token_account, &mut authority_account]
            } else {
                vec![
                    &mut token_account,
                    &mut account_account,
                    &mut authority_account,
                ]
            },
        )
            .unwrap();

        return ((token_key, token_account), (account_key, account_account));
    }
}
