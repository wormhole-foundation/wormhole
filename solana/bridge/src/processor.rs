//! Program instruction processing logic
#![cfg(feature = "program")]

use std::mem::size_of;
use std::slice::Iter;

use num_traits::AsPrimitive;
use primitive_types::U256;
use solana_sdk::clock::Clock;
#[cfg(not(target_arch = "bpf"))]
use solana_sdk::instruction::Instruction;
#[cfg(target_arch = "bpf")]
use solana_sdk::program::invoke_signed;
use solana_sdk::rent::Rent;
use solana_sdk::system_instruction::create_account;
use solana_sdk::sysvar::Sysvar;
use solana_sdk::{
    account_info::next_account_info, account_info::AccountInfo, entrypoint::ProgramResult, info,
    program_error::ProgramError, pubkey::Pubkey,
};
use spl_token::state::Mint;

use crate::error::Error;
use crate::instruction::BridgeInstruction::*;
use crate::instruction::{BridgeInstruction, TransferOutPayload, VAAData, CHAIN_ID_SOLANA};
use crate::state::*;
use crate::syscalls::RawKey;
use crate::vaa::{BodyTransfer, BodyUpdateGuardianSet, VAABody, VAA};

/// Instruction processing logic
impl Bridge {
    /// Processes an [Instruction](enum.Instruction.html).
    pub fn process(program_id: &Pubkey, accounts: &[AccountInfo], input: &[u8]) -> ProgramResult {
        let instruction = BridgeInstruction::deserialize(input)?;
        match instruction {
            Initialize(payload) => {
                info!("Instruction: Initialize");
                Self::process_initialize(
                    program_id,
                    accounts,
                    payload.initial_guardian,
                    payload.config,
                )
            }
            TransferOut(p) => {
                info!("Instruction: TransferOut");

                if p.asset.chain == CHAIN_ID_SOLANA {
                    Self::process_transfer_native_out(program_id, accounts, &p)
                } else {
                    Self::process_transfer_out(program_id, accounts, &p)
                }
            }
            PostVAA(vaa_body) => {
                info!("Instruction: PostVAA");
                let len = vaa_body[0] as usize;
                let vaa_data = &vaa_body[..len];
                let vaa = VAA::deserialize(vaa_data)?;

                let hash = vaa.body_hash()?;

                Self::process_vaa(program_id, accounts, vaa_body, &vaa, &hash)
            }
            _ => panic!(""),
        }
    }

    /// Unpacks a token state from a bytes buffer while assuring that the state is initialized.
    pub fn process_initialize(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        initial_guardian_key: RawKey,
        config: BridgeConfig,
    ) -> ProgramResult {
        let account_info_iter = &mut accounts.iter();
        next_account_info(account_info_iter)?; // System program
        let clock_info = next_account_info(account_info_iter)?;
        let new_bridge_info = next_account_info(account_info_iter)?;
        let new_guardian_info = next_account_info(account_info_iter)?;
        let payer_info = next_account_info(account_info_iter)?;

        let clock = Clock::from_account_info(clock_info)?;

        // Create bridge account
        let bridge_seed = Bridge::derive_bridge_seeds();
        Bridge::check_and_create_account::<BridgeConfig>(
            program_id,
            accounts,
            new_bridge_info.key,
            payer_info.key,
            &bridge_seed,
        )?;

        let mut new_account_data = new_bridge_info.data.borrow_mut();
        let mut bridge: &mut Bridge = Self::unpack_unchecked(&mut new_account_data)?;
        if bridge.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

        // Create guardian set account
        let guardian_seed = Bridge::derive_guardian_set_seeds(new_bridge_info.key, 0);
        Bridge::check_and_create_account::<GuardianSet>(
            program_id,
            accounts,
            new_guardian_info.key,
            payer_info.key,
            &guardian_seed,
        )?;

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
        info!("wrapped transfer out");
        let account_info_iter = &mut accounts.iter();
        next_account_info(account_info_iter)?; // System program
        next_account_info(account_info_iter)?; // Token program
        let sender_account_info = next_account_info(account_info_iter)?;
        let bridge_info = next_account_info(account_info_iter)?;
        let transfer_info = next_account_info(account_info_iter)?;
        let mint_info = next_account_info(account_info_iter)?;
        let payer_info = next_account_info(account_info_iter)?;
        let authority_info = next_account_info(account_info_iter)?;

        let sender = Bridge::token_account_deserialize(sender_account_info)?;
        let bridge = Bridge::bridge_deserialize(bridge_info)?;

        // Does the token belong to the mint
        if sender.mint != *mint_info.key {
            return Err(Error::TokenMintMismatch.into());
        }

        // Check that the mint is actually a wrapped asset belonging to *this* bridge instance
        let expected_mint_address = Bridge::derive_wrapped_asset_id(
            program_id,
            bridge_info.key,
            t.asset.chain,
            t.asset.address,
        )?;
        if expected_mint_address != *mint_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Create transfer account
        let transfer_seed = Bridge::derive_transfer_id_seeds(
            bridge_info.key,
            t.asset.chain,
            t.asset.address,
            t.chain_id,
            t.target,
            sender.owner.to_bytes(),
            t.nonce,
        );
        Bridge::check_and_create_account::<TransferOutProposal>(
            program_id,
            accounts,
            transfer_info.key,
            payer_info.key,
            &transfer_seed,
        )?;

        // Load transfer account
        let mut transfer_data = transfer_info.data.borrow_mut();
        let transfer: &mut TransferOutProposal = Bridge::unpack_unchecked(&mut transfer_data)?;
        if transfer.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

        // Burn tokens
        Bridge::wrapped_burn(
            accounts,
            &bridge.config.token_program,
            authority_info.key,
            sender_account_info.key,
            t.amount,
        )?;

        // Initialize transfer
        transfer.is_initialized = true;
        transfer.foreign_address = t.target;
        transfer.amount = t.amount;
        transfer.to_chain_id = t.chain_id;
        transfer.asset = t.asset;

        Ok(())
    }

    /// Transfers a native token to a foreign chain
    pub fn process_transfer_native_out(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        t: &TransferOutPayload,
    ) -> ProgramResult {
        info!("native transfer out");
        let account_info_iter = &mut accounts.iter();
        next_account_info(account_info_iter)?; // System program
        next_account_info(account_info_iter)?; // Token program
        let sender_account_info = next_account_info(account_info_iter)?;
        let bridge_info = next_account_info(account_info_iter)?;
        let transfer_info = next_account_info(account_info_iter)?;
        let mint_info = next_account_info(account_info_iter)?;
        let payer_info = next_account_info(account_info_iter)?;
        let custody_info = next_account_info(account_info_iter)?;

        let sender = Bridge::token_account_deserialize(sender_account_info)?;
        let bridge = Bridge::bridge_deserialize(bridge_info)?;

        // Does the token belong to the mint
        if sender.mint != *mint_info.key {
            return Err(Error::TokenMintMismatch.into());
        }

        // Create transfer account
        let transfer_seed = Bridge::derive_transfer_id_seeds(
            bridge_info.key,
            t.asset.chain,
            t.asset.address,
            t.chain_id,
            t.target,
            sender_account_info.key.to_bytes(),
            t.nonce,
        );
        Bridge::check_and_create_account::<TransferOutProposal>(
            program_id,
            accounts,
            transfer_info.key,
            payer_info.key,
            &transfer_seed,
        )?;

        // Load transfer account
        let mut transfer_data = transfer_info.data.borrow_mut();
        let transfer: &mut TransferOutProposal = Bridge::unpack_unchecked(&mut transfer_data)?;
        if transfer.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

        // Check that custody account was derived correctly
        let expected_custody_id =
            Bridge::derive_custody_id(program_id, bridge_info.key, mint_info.key)?;
        if expected_custody_id != *custody_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Create the account if it does not exist
        if custody_info.data_is_empty() {
            Bridge::create_custody_account(
                program_id,
                accounts,
                &bridge.config.token_program,
                bridge_info.key,
                custody_info.key,
                mint_info.key,
                payer_info.key,
            )?;
        }

        // Check that the custody token account is owned by the derived key
        let custody = Self::token_account_deserialize(custody_info)?;
        if custody.owner != *bridge_info.key {
            return Err(Error::WrongTokenAccountOwner.into());
        }

        // Transfer tokens to custody - This also checks that custody mint = mint
        Bridge::token_transfer_caller(
            accounts,
            &bridge.config.token_program,
            sender_account_info.key,
            custody_info.key,
            bridge_info.key,
            t.amount,
        )?;

        // Initialize proposal
        transfer.is_initialized = true;
        transfer.foreign_address = t.target;
        transfer.amount = t.amount;
        transfer.to_chain_id = t.chain_id;

        // Don't use the user-given data as we don't check mint = AssetMeta.address
        transfer.asset = AssetMeta {
            chain: CHAIN_ID_SOLANA,
            address: mint_info.key.to_bytes(),
        };

        Ok(())
    }

    /// Processes a VAA
    pub fn process_vaa(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        vaa_data: VAAData,
        vaa: &VAA,
        hash: &[u8; 32],
    ) -> ProgramResult {
        let account_info_iter = &mut accounts.iter();

        // Load VAA processing default accounts
        next_account_info(account_info_iter)?; // System program
        let clock_info = next_account_info(account_info_iter)?;
        let bridge_info = next_account_info(account_info_iter)?;
        let guardian_set_info = next_account_info(account_info_iter)?;
        let claim_info = next_account_info(account_info_iter)?;
        let payer_info = next_account_info(account_info_iter)?;

        let mut bridge = Bridge::bridge_deserialize(bridge_info)?;
        let clock = Clock::from_account_info(clock_info)?;
        let mut guardian_set = Bridge::guardian_set_deserialize(guardian_set_info)?;

        // Check that the guardian set is valid
        let expected_guardian_set =
            Bridge::derive_guardian_set_id(program_id, bridge_info.key, vaa.guardian_set_index)?;
        if expected_guardian_set != *guardian_set_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Check and create claim
        let claim_seeds = Bridge::derive_claim_seeds(bridge_info.key, hash);
        Bridge::check_and_create_account::<ClaimedVAA>(
            program_id,
            accounts,
            claim_info.key,
            payer_info.key,
            &claim_seeds,
        )?;

        // Check that the guardian set is still active
        if (guardian_set.expiration_time as i64) > clock.unix_timestamp {
            return Err(Error::GuardianSetExpired.into());
        }

        // Check that the VAA is still valid
        if (guardian_set.expiration_time as i64) + (bridge.config.vaa_expiration_time as i64)
            > clock.unix_timestamp
        {
            return Err(Error::VAAExpired.into());
        }

        // Verify VAA signature
        if !vaa.verify(&guardian_set.pubkey) {
            return Err(Error::InvalidVAASignature.into());
        }

        let payload = vaa.payload.ok_or(Error::InvalidVAAAction)?;
        match payload {
            VAABody::UpdateGuardianSet(v) => Self::process_vaa_set_update(
                program_id,
                accounts,
                account_info_iter,
                &clock,
                bridge_info,
                payer_info,
                &mut bridge,
                &mut guardian_set,
                &v,
            ),
            VAABody::Transfer(v) => {
                if v.source_chain == CHAIN_ID_SOLANA {
                    Self::process_vaa_transfer(
                        program_id,
                        accounts,
                        account_info_iter,
                        bridge_info,
                        payer_info,
                        &mut bridge,
                        &v,
                    )
                } else {
                    Self::process_vaa_transfer_post(
                        program_id,
                        account_info_iter,
                        bridge_info,
                        vaa,
                        &v,
                        vaa_data,
                    )
                }
            }
        }?;

        // Load claim account
        let mut claim_data = claim_info.data.borrow_mut();
        let claim: &mut ClaimedVAA = Bridge::unpack_unchecked(&mut claim_data)?;
        if claim.is_initialized {
            return Err(Error::VAAClaimed.into());
        }

        // Set claimed
        claim.is_initialized = true;
        claim.vaa_time = clock.unix_timestamp as u32;

        Ok(())
    }

    /// Processes a Guardian set update
    pub fn process_vaa_set_update(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        account_info_iter: &mut Iter<AccountInfo>,
        clock: &Clock,
        bridge_info: &AccountInfo,
        payer_info: &AccountInfo,
        bridge: &mut Bridge,
        old_guardian_set: &mut GuardianSet,
        b: &BodyUpdateGuardianSet,
    ) -> ProgramResult {
        let new_guardian_info = next_account_info(account_info_iter)?;

        // TODO this could deadlock the bridge if an update is performed with an invalid key
        // The new guardian set must be signed by the current one
        if bridge.guardian_set_index != old_guardian_set.index {
            return Err(Error::OldGuardianSet.into());
        }

        // The new guardian set must have an index > current
        // We don't check +1 because we trust the set to not set something close to max(u32)
        if bridge.guardian_set_index >= b.new_index {
            return Err(Error::GuardianIndexNotIncreasing.into());
        }

        // Set the exirity on the old guardian set
        // The guardian set will expire once all currently issues vaas have expired
        old_guardian_set.expiration_time =
            (clock.unix_timestamp as u32) + bridge.config.vaa_expiration_time;

        // Check whether the new guardian set was derived correctly
        let guardian_seed = Bridge::derive_guardian_set_seeds(bridge_info.key, b.new_index);
        Bridge::check_and_create_account::<GuardianSet>(
            program_id,
            accounts,
            new_guardian_info.key,
            payer_info.key,
            &guardian_seed,
        )?;

        let mut guardian_set_new_data = new_guardian_info.data.borrow_mut();
        let guardian_set_new: &mut GuardianSet =
            Bridge::unpack_unchecked(&mut guardian_set_new_data)?;

        // The new guardian set must not exist
        if guardian_set_new.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

        // Set values on the new guardian set
        guardian_set_new.is_initialized = true;
        guardian_set_new.index = b.new_index;
        guardian_set_new.pubkey = b.new_key;
        guardian_set_new.creation_time = clock.unix_timestamp as u32;

        // Update the bridge guardian set id
        bridge.guardian_set_index = b.new_index;

        Ok(())
    }

    /// Processes a VAA transfer in
    pub fn process_vaa_transfer(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        account_info_iter: &mut Iter<AccountInfo>,
        bridge_info: &AccountInfo,
        payer_info: &AccountInfo,
        bridge: &mut Bridge,
        b: &BodyTransfer,
    ) -> ProgramResult {
        next_account_info(account_info_iter)?; // Token program
        let mint_info = next_account_info(account_info_iter)?;
        let destination_info = next_account_info(account_info_iter)?;
        let wrapped_meta_info = next_account_info(account_info_iter)?;

        let destination = Self::token_account_deserialize(destination_info)?;
        if destination.mint != *mint_info.key {
            return Err(Error::TokenMintMismatch.into());
        }

        if b.asset.chain == CHAIN_ID_SOLANA {
            let custody_info = next_account_info(account_info_iter)?;
            let expected_custody_id =
                Bridge::derive_custody_id(program_id, bridge_info.key, mint_info.key)?;
            if expected_custody_id != *custody_info.key {
                return Err(Error::InvalidDerivedAccount.into());
            }

            // Native Solana asset, transfer from custody
            Bridge::token_transfer_custody(
                accounts,
                &bridge.config.token_program,
                bridge_info.key,
                custody_info.key,
                destination_info.key,
                b.amount,
            )?;
        } else {
            // Foreign chain asset, mint wrapped asset
            let expected_mint_address = Bridge::derive_wrapped_asset_id(
                program_id,
                bridge_info.key,
                b.asset.chain,
                b.asset.address,
            )?;
            if expected_mint_address != *mint_info.key {
                return Err(Error::InvalidDerivedAccount.into());
            }

            // If wrapped mint does not exist, create it
            if mint_info.data_is_empty() {
                Self::create_wrapped_mint(
                    program_id,
                    accounts,
                    &bridge.config.token_program,
                    mint_info.key,
                    bridge_info.key,
                    payer_info.key,
                    &b.asset,
                )?;

                // Check and create wrapped asset meta if it is unset
                let wrapped_meta_seeds =
                    Bridge::derive_wrapped_meta_seeds(bridge_info.key, mint_info.key);
                Bridge::check_and_create_account::<WrappedAssetMeta>(
                    program_id,
                    accounts,
                    wrapped_meta_info.key,
                    payer_info.key,
                    &wrapped_meta_seeds,
                )?;

                let mut wrapped_meta_data = wrapped_meta_info.data.borrow_mut();
                let wrapped_meta: &mut WrappedAssetMeta =
                    Bridge::unpack_unchecked(&mut wrapped_meta_data)?;

                wrapped_meta.is_initialized = true;
                wrapped_meta.address = b.asset.address;
                wrapped_meta.chain = b.asset.chain;
            }

            Bridge::wrapped_mint_to(
                accounts,
                &bridge.config.token_program,
                mint_info.key,
                destination_info.key,
                bridge_info.key,
                b.amount,
            )?;
        }

        Ok(())
    }

    /// Processes a VAA post for data availability (for Solana -> foreign transfers)
    pub fn process_vaa_transfer_post(
        program_id: &Pubkey,
        account_info_iter: &mut Iter<AccountInfo>,
        bridge_info: &AccountInfo,
        vaa: &VAA,
        b: &BodyTransfer,
        vaa_data: VAAData,
    ) -> ProgramResult {
        let proposal_info = next_account_info(account_info_iter)?;

        // Check whether the proposal was derived correctly
        let expected_proposal = Bridge::derive_transfer_id(
            program_id,
            bridge_info.key,
            b.asset.chain,
            b.asset.address,
            b.target_chain,
            b.target_address,
            b.source_address,
            b.nonce,
        )?;
        if expected_proposal != *proposal_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        let mut proposal = Self::transfer_out_proposal_deserialize(proposal_info)?;
        if !proposal.matches_vaa(b) {
            return Err(Error::VAAProposalMismatch.into());
        }

        // Set vaa
        proposal.vaa = vaa_data;
        proposal.vaa_time = vaa.timestamp;

        Ok(())
    }
}

// Test program id for the swap program.
#[cfg(not(target_arch = "bpf"))]
const WORMHOLE_PROGRAM_ID: Pubkey = Pubkey::new_from_array([2u8; 32]);
#[cfg(not(target_arch = "bpf"))]
const TOKEN_PROGRAM_ID: Pubkey = Pubkey::new_from_array([2u8; 32]);

/// Routes invokes to the token program, used for testing.
#[cfg(not(target_arch = "bpf"))]
pub fn invoke_signed<'a>(
    instruction: &Instruction,
    account_infos: &[AccountInfo<'a>],
    signers_seeds: &[&[&[u8]]],
) -> ProgramResult {
    let mut new_account_infos = vec![];
    for meta in instruction.accounts.iter() {
        for account_info in account_infos.iter() {
            if meta.pubkey == *account_info.key {
                let mut new_account_info = account_info.clone();
                for seeds in signers_seeds.iter() {
                    let signer =
                        Pubkey::create_program_address(seeds, &WORMHOLE_PROGRAM_ID).unwrap();
                    if *account_info.key == signer {
                        new_account_info.is_signer = true;
                    }
                }
                new_account_infos.push(new_account_info);
            }
        }
    }

    match instruction.program_id {
        TOKEN_PROGRAM_ID => spl_token::processor::Processor::process(
            &instruction.program_id,
            &new_account_infos,
            &instruction.data,
        ),
        _ => panic!(),
    }
}

/// Implementation of actions
impl Bridge {
    /// Burn a wrapped asset from account
    pub fn wrapped_burn(
        accounts: &[AccountInfo],
        token_program_id: &Pubkey,
        authority: &Pubkey,
        token_account: &Pubkey,
        amount: U256,
    ) -> Result<(), ProgramError> {
        let all_signers: Vec<&Pubkey> = accounts
            .iter()
            .filter_map(|item| if item.is_signer { Some(item.key) } else { None })
            .collect();
        let ix = spl_token::instruction::burn(
            token_program_id,
            token_account,
            authority,
            all_signers.as_slice(),
            amount,
        )?;
        invoke_signed(&ix, accounts, &[&["bridge".as_bytes()]])
    }

    /// Mint a wrapped asset to account
    pub fn wrapped_mint_to(
        accounts: &[AccountInfo],
        token_program_id: &Pubkey,
        mint: &Pubkey,
        destination: &Pubkey,
        bridge: &Pubkey,
        amount: U256,
    ) -> Result<(), ProgramError> {
        let ix = spl_token::instruction::mint_to(
            token_program_id,
            mint,
            destination,
            bridge,
            &[],
            amount,
        )?;
        invoke_signed(&ix, accounts, &[&["bridge".as_bytes()]])
    }

    /// Transfer tokens from a caller
    pub fn token_transfer_caller(
        accounts: &[AccountInfo],
        token_program_id: &Pubkey,
        source: &Pubkey,
        destination: &Pubkey,
        authority: &Pubkey,
        amount: U256,
    ) -> Result<(), ProgramError> {
        let ix = spl_token::instruction::transfer(
            token_program_id,
            source,
            destination,
            authority,
            &[],
            amount,
        )?;
        invoke_signed(&ix, accounts, &[&["bridge".as_bytes()]])
    }

    /// Transfer tokens from a custody account
    pub fn token_transfer_custody(
        accounts: &[AccountInfo],
        token_program_id: &Pubkey,
        bridge: &Pubkey,
        source: &Pubkey,
        destination: &Pubkey,
        amount: U256,
    ) -> Result<(), ProgramError> {
        let ix = spl_token::instruction::transfer(
            token_program_id,
            source,
            destination,
            bridge,
            &[],
            amount,
        )?;
        invoke_signed(&ix, accounts, &[&["bridge".as_bytes()]])
    }

    /// Create a new account
    pub fn create_custody_account(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        token_program: &Pubkey,
        bridge: &Pubkey,
        account: &Pubkey,
        mint: &Pubkey,
        payer: &Pubkey,
    ) -> Result<(), ProgramError> {
        Self::create_account::<Mint>(
            token_program,
            accounts,
            mint,
            payer,
            &Self::derive_custody_seeds(bridge, mint),
        )?;
        let ix = spl_token::instruction::initialize_account(token_program, account, mint, bridge)?;
        invoke_signed(&ix, accounts, &[&["bridge".as_bytes()]])
    }

    /// Create a mint for a wrapped asset
    pub fn create_wrapped_mint(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        token_program: &Pubkey,
        mint: &Pubkey,
        bridge: &Pubkey,
        payer: &Pubkey,
        asset: &AssetMeta,
    ) -> Result<(), ProgramError> {
        Self::create_account::<Mint>(
            token_program,
            accounts,
            mint,
            payer,
            &Self::derive_wrapped_asset_seeds(bridge, asset.chain, asset.address),
        )?;
        let ix = spl_token::instruction::initialize_mint(
            token_program,
            mint,
            None,
            Some(bridge),
            U256::from(0),
            8,
        )?;
        invoke_signed(&ix, accounts, &[&["bridge".as_bytes()]])
    }

    /// Check that a key was derived correctly and create account
    pub fn check_and_create_account<T: Sized>(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        new_account: &Pubkey,
        payer: &Pubkey,
        seeds: &Vec<Vec<u8>>,
    ) -> Result<(), ProgramError> {
        let expected_key = Bridge::derive_key(program_id, seeds)?;
        if expected_key != *new_account {
            return Err(Error::InvalidDerivedAccount.into());
        }

        Self::create_account::<T>(program_id, accounts, new_account, payer, seeds)
    }

    /// Create a new account
    pub fn create_account<T: Sized>(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        new_account: &Pubkey,
        payer: &Pubkey,
        seeds: &Vec<Vec<u8>>,
    ) -> Result<(), ProgramError> {
        let size = size_of::<T>();
        let ix = create_account(
            payer,
            new_account,
            Rent::default().minimum_balance(size as usize),
            size as u64,
            program_id,
        );
        let s: Vec<_> = seeds.iter().map(|item| item.as_slice()).collect();
        //invoke_signed(&ix, accounts, &[s.as_slice()])
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use solana_sdk::{
        account::Account, account_info::create_is_signer_account_infos, instruction::Instruction,
    };
    use spl_token::{
        instruction::{initialize_account, initialize_mint},
        processor::Processor,
        state::{Account as SplAccount, Mint as SplMint},
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
            Processor::process(&instruction.program_id, &account_infos, &instruction.data)
        }
    }
}
