//! Program instruction processing logic

use std::io::Write;
use std::mem::size_of;
use std::slice::Iter;
use std::str;

use num_traits::AsPrimitive;
use sha3::Digest;
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
use solana_sdk::{
    account_info::next_account_info, account_info::AccountInfo, entrypoint::ProgramResult, info,
    program_error::ProgramError, pubkey::bs58, pubkey::Pubkey,
};
use spl_token::state::Mint;

use crate::instruction::BridgeInstruction::*;
use crate::instruction::{
    BridgeInstruction, ForeignAddress, GuardianKey, TransferOutPayload, CHAIN_ID_SOLANA, VAA_BODY,
};
use crate::state::*;
use crate::syscalls::{sol_verify_schnorr, RawKey, SchnorrifyInput};
use crate::vaa::{BodyTransfer, BodyUpdateGuardianSet, VAABody, VAA};
use crate::{error::Error, instruction::unpack};

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

                let mut k = sha3::Keccak256::default();
                if let Err(_) = k.write(vaa_data) {
                    return Err(Error::ParseFailed.into());
                };
                let hash = k.finalize();

                Self::process_vaa(program_id, accounts, &vaa_body, &vaa, hash.as_ref())
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

        let expected_guardian_set_key =
            Bridge::derive_guardian_set_id(program_id, new_account_info.key, 0)?;
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
        let expected_mint_address = Bridge::derive_wrapped_asset_id(
            program_id,
            bridge_info.key,
            t.asset.chain,
            t.asset.address,
        )?;
        if expected_mint_address != *mint_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Check that the transfer account was derived correctly
        let expected_transfer_id = Bridge::derive_transfer_id(
            program_id,
            bridge_info.key,
            t.asset.chain,
            t.asset.address,
            t.chain_id,
            t.target,
            sender.owner.to_bytes(),
            clock.slot.as_(),
        )?;
        if expected_transfer_id != *proposal_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Load proposal account
        let mut proposal_data = proposal_info.data.borrow_mut();
        let proposal: &mut TransferOutProposal = Bridge::unpack_unchecked(&mut proposal_data)?;
        if proposal.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

        // Burn tokens
        Bridge::wrapped_burn(
            accounts,
            &bridge.config.token_program,
            sender_info.key,
            sender_account_info.key,
            t.amount,
        )?;

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
        let expected_transfer_id = Bridge::derive_transfer_id(
            program_id,
            bridge_info.key,
            t.asset.chain,
            t.asset.address,
            t.chain_id,
            t.target,
            sender.owner.to_bytes(),
            clock.slot.as_(),
        )?;
        if expected_transfer_id != *proposal_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Load proposal account
        let mut proposal_data = proposal_info.data.borrow_mut();
        let proposal: &mut TransferOutProposal = Bridge::unpack_unchecked(&mut proposal_data)?;
        if proposal.is_initialized {
            return Err(Error::AlreadyExists.into());
        }

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
                sender_info.key,
            )?;
        }

        // Check that the custody token account is owned by the derived key
        let custody = Self::token_account_deserialize(custody_info)?;
        if custody.owner != *bridge_info.key {
            return Err(Error::WrongTokenAccountOwner.into());
        }

        // Transfer tokens to custody
        Bridge::token_transfer_caller(
            accounts,
            &bridge.config.token_program,
            sender_account_info.key,
            custody_info.key,
            sender_info.key,
            t.amount,
        )?;

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

    /// Processes a VAA
    pub fn process_vaa(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        vaa_data: &[u8; 100],
        vaa: &VAA,
        hash: &[u8; 32],
    ) -> ProgramResult {
        let account_info_iter = &mut accounts.iter();
        let bridge_info = next_account_info(account_info_iter)?;
        let clock_info = next_account_info(account_info_iter)?;
        let guardian_set_info = next_account_info(account_info_iter)?;
        let claim_info = next_account_info(account_info_iter)?;

        let mut bridge = Bridge::bridge_deserialize(bridge_info)?;
        let clock = Clock::from_account_info(clock_info)?;
        let mut guardian_set = Bridge::guardian_set_deserialize(guardian_set_info)?;

        // Check that the guardian set is valid
        let expected_guardian_set =
            Bridge::derive_guardian_set_id(program_id, bridge_info.key, vaa.guardian_set_index)?;
        if expected_guardian_set != *guardian_set_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Check that the claim is valid
        let expected_claim = Bridge::derive_claim_id(program_id, bridge_info.key, hash)?;
        if expected_claim != *claim_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        // Check that the guardian set is still active
        if (guardian_set.expiration_time as i64) < clock.unix_timestamp {
            return Err(Error::GuardianSetExpired.into());
        }

        // Check that the VAA is still valid
        if (guardian_set.expiration_time as i64) + (bridge.config.vaa_expiration_time as i64)
            < clock.unix_timestamp
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
                account_info_iter,
                &clock,
                bridge_info,
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

        // Load proposal account
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
        account_info_iter: &mut Iter<AccountInfo>,
        clock: &Clock,
        bridge_info: &AccountInfo,
        bridge: &mut Bridge,
        old_guardian_set: &mut GuardianSet,
        b: &BodyUpdateGuardianSet,
    ) -> ProgramResult {
        let guardian_set_new_info = next_account_info(account_info_iter)?;

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
        let expected_guardian_set =
            Bridge::derive_guardian_set_id(program_id, bridge_info.key, b.new_index)?;
        if expected_guardian_set != *guardian_set_new_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        let mut guardian_set_new_data = guardian_set_new_info.data.borrow_mut();
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
        bridge: &mut Bridge,
        b: &BodyTransfer,
    ) -> ProgramResult {
        let mint_info = next_account_info(account_info_iter)?;
        let destination_info = next_account_info(account_info_iter)?;

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
                let sender_info = next_account_info(account_info_iter)?;
                Self::create_wrapped_mint(
                    program_id,
                    accounts,
                    &bridge.config.token_program,
                    mint_info.key,
                    bridge_info.key,
                    sender_info.key,
                    &b.asset,
                )?;
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
        vaa_data: &[u8; 100],
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
            b.ref_block,
        )?;
        if expected_proposal != *proposal_info.key {
            return Err(Error::InvalidDerivedAccount.into());
        }

        let mut proposal = Self::transfer_out_proposal_deserialize(proposal_info)?;
        if !proposal.matches_vaa(b) {
            return Err(Error::VAAProposalMismatch.into());
        }

        // Set vaa
        proposal.vaa = *vaa_data;
        proposal.vaa_time = vaa.timestamp;

        Ok(())
    }
}

// Test program id for the swap program.
#[cfg(not(target_arch = "bpf"))]
const WORMHOLE_PROGRAM_ID: Pubkey = Pubkey::new_from_array([2u8; 32]);
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
        TOKEN_PROGRAM_ID => spl_token::state::State::process(
            &instruction.program_id,
            &new_account_infos,
            &instruction.data,
        ),
        _ => panic!(),
    }
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

    fn create_bridge(
        program_id: &Pubkey,
        config: &BridgeConfig,
        initial_guardians: &RawKey,
    ) -> (Pubkey, Account) {
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

        return ((account_key, account_account));
    }
}
