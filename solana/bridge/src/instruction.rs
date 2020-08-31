#![allow(clippy::too_many_arguments)]
//! Instruction types

use std::io::{Cursor, Read, Write};
use std::mem::size_of;

use byteorder::{BigEndian, ReadBytesExt, WriteBytesExt};
use primitive_types::U256;
use solana_sdk::{
    instruction::{AccountMeta, Instruction},
    program_error::ProgramError,
    pubkey::Pubkey,
};

use crate::error::Error;
use crate::error::Error::VAATooLong;
use crate::instruction::BridgeInstruction::{Initialize, PokeProposal, PostVAA, TransferOut};
use crate::state::{AssetMeta, Bridge, BridgeConfig};
use crate::vaa::{VAABody, VAA};

/// chain id of this chain
pub const CHAIN_ID_SOLANA: u8 = 1;
/// maximum number of guardians
pub const MAX_LEN_GUARDIAN_KEYS: usize = 20;
/// maximum size of a posted VAA
pub const MAX_VAA_SIZE: usize = 1000;
/// size of a foreign address in bytes
const FOREIGN_ADDRESS_SIZE: usize = 32;

/// serialized VAA data
pub type VAAData = Vec<u8>;
/// X and Y point of P for guardians
pub type GuardianKey = [u8; 64];
/// address on a foreign chain
pub type ForeignAddress = [u8; FOREIGN_ADDRESS_SIZE];

#[repr(C)]
#[derive(Clone, Copy)]
pub struct InitializePayload {
    /// number of initial guardians
    pub len_guardians: u8,
    /// guardians that are allowed to sign mints
    pub initial_guardian: [[u8; 20]; MAX_LEN_GUARDIAN_KEYS],
    /// config for the bridge
    pub config: BridgeConfig,
}

#[repr(C)]
#[derive(Clone, Copy, Debug)]
pub struct TransferOutPayload {
    /// amount to transfer
    pub amount: U256,
    /// chain id to transfer to
    pub chain_id: u8,
    /// Information about the asset to be transferred
    pub asset: AssetMeta,
    /// address on the foreign chain to transfer to
    pub target: ForeignAddress,
    /// unique nonce of the transfer
    pub nonce: u32,
}

#[repr(C)]
#[derive(Clone, Copy, Debug)]
pub struct TransferOutPayloadRaw {
    /// amount to transfer
    pub amount: [u8; 32],
    /// chain id to transfer to
    pub chain_id: u8,
    /// Information about the asset to be transferred
    pub asset: AssetMeta,
    /// address on the foreign chain to transfer to
    pub target: ForeignAddress,
    /// unique nonce of the transfer
    pub nonce: u32,
}

/// Instructions supported by the SwapInfo program.
#[repr(C)]
pub enum BridgeInstruction {
    /// Initializes a new Bridge
    /// Accounts expected by this instruction:
    ///
    ///   0. `[writable, derived]`  The bridge to initialize.
    ///   1. `[]` The System program
    ///   2. `[]` The clock SysVar
    ///   3. `[writable, derived]` The initial guardian set account
    ///   4. `[signer]` The fee payer for new account creation
    Initialize(InitializePayload),

    /// Burns or locks a (wrapped) asset `token` from `sender` on the Solana chain.
    ///
    ///   Wrapped asset transfer out
    ///   0. `[writable]`  The from token account
    ///   1. `[]` The System program.
    ///   2. `[]` The spl token program.
    ///   3. `[]` The clock SysVar
    ///   4. `[derived]` The bridge config
    ///   5. `[writable, derived, empty]` The new transfer out tracking account
    ///   6. `[writable, derived]` The mint of the wrapped asset
    ///   7. ..7+M '[signer]' M signer accounts (from token authority)
    ///
    ///   Native token transfer out
    ///   0. `[writable]`  The from token account
    ///   1. `[]` The System program.
    ///   2. `[]` The spl token program.
    ///   3. `[]` The clock SysVar
    ///   4. `[derived]` The bridge config
    ///   5. `[writable, derived, empty]` The new transfer out tracking account
    ///   6. `[writable, derived]` The mint of the wrapped asset
    ///   7. `[writable, derived]` The custody token account of the bridge
    ///   8. ..8+M '[signer]' M signer accounts (from token authority)
    TransferOut(TransferOutPayload),

    /// Submits a VAA signed by `guardian` on a valid `proposal`.
    /// See docs for accounts
    PostVAA(VAAData),

    /// Deletes a `proposal` after the `VAA_EXPIRATION_TIME` is over to free up space on chain.
    /// This returns the rent to the sender.
    EvictTransferOut(),

    /// Deletes a `ExecutedVAA` after the `VAA_EXPIRATION_TIME` is over to free up space on chain.
    /// This returns the rent to the sender.
    EvictClaimedVAA(),

    /// Pokes a proposal with no valid VAAs attached so guardians reprocess it.
    PokeProposal(),
}

impl BridgeInstruction {
    /// Deserializes a byte buffer into a BridgeInstruction
    pub fn deserialize(input: &[u8]) -> Result<Self, ProgramError> {
        if input.len() < size_of::<u8>() {
            return Err(ProgramError::InvalidAccountData);
        }
        Ok(match input[0] {
            0 => {
                let payload: &InitializePayload = unpack(input)?;

                Initialize(*payload)
            }
            1 => {
                let payload: &TransferOutPayloadRaw = unpack(input)?;
                let amount = U256::from_big_endian(&payload.amount);

                TransferOut(TransferOutPayload {
                    amount,
                    chain_id: payload.chain_id,
                    asset: payload.asset,
                    target: payload.target,
                    nonce: payload.nonce,
                })
            }
            2 => {
                let payload: VAAData = input[1..].to_vec();
                PostVAA(payload)
            }
            5 => PokeProposal(),
            _ => return Err(ProgramError::InvalidInstructionData),
        })
    }

    /// Serializes a BridgeInstruction into a byte buffer.
    pub fn serialize(self: Self) -> Result<Vec<u8>, ProgramError> {
        let mut output = vec![0u8; size_of::<BridgeInstruction>()];

        match self {
            Self::Initialize(payload) => {
                output[0] = 0;
                #[allow(clippy::cast_ptr_alignment)]
                let value = unsafe {
                    &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut InitializePayload)
                };
                *value = payload;
            }
            Self::TransferOut(payload) => {
                output[0] = 1;
                #[allow(clippy::cast_ptr_alignment)]
                let value = unsafe {
                    &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut TransferOutPayloadRaw)
                };

                let mut amount_bytes = [0u8; 32];
                payload.amount.to_big_endian(&mut amount_bytes);

                *value = TransferOutPayloadRaw {
                    amount: amount_bytes,
                    chain_id: payload.chain_id,
                    asset: payload.asset,
                    target: payload.target,
                    nonce: payload.nonce,
                };
            }
            Self::PostVAA(payload) => {
                output[0] = 2;
                #[allow(clippy::cast_ptr_alignment)]
                let value =
                    unsafe { &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut VAAData) };
                *value = payload;
            }
            Self::EvictTransferOut() => {
                output[0] = 3;
            }
            Self::EvictClaimedVAA() => {
                output[0] = 4;
            }
            Self::PokeProposal() => {
                output[0] = 5;
            }
        }
        Ok(output)
    }
}

/// Creates an 'initialize' instruction.
#[cfg(not(target_arch = "bpf"))]
pub fn initialize(
    program_id: &Pubkey,
    sender: &Pubkey,
    initial_guardian: Vec<[u8; 20]>,
    config: &BridgeConfig,
) -> Result<Instruction, ProgramError> {
    if initial_guardian.len() > MAX_LEN_GUARDIAN_KEYS {
        return Err(ProgramError::InvalidArgument);
    }
    let mut initial_g = [[0u8; 20]; MAX_LEN_GUARDIAN_KEYS];
    for (i, key) in initial_guardian.iter().enumerate() {
        initial_g[i] = *key;
    }
    let data = BridgeInstruction::Initialize(InitializePayload {
        config: *config,
        len_guardians: initial_guardian.len() as u8,
        initial_guardian: initial_g,
    })
    .serialize()?;

    let bridge_key = Bridge::derive_bridge_id(program_id)?;
    let guardian_set_key = Bridge::derive_guardian_set_id(program_id, &bridge_key, 0)?;

    let accounts = vec![
        AccountMeta::new_readonly(solana_sdk::system_program::id(), false),
        AccountMeta::new_readonly(solana_sdk::sysvar::clock::id(), false),
        AccountMeta::new(bridge_key, false),
        AccountMeta::new(guardian_set_key, false),
        AccountMeta::new(*sender, true),
    ];

    Ok(Instruction {
        program_id: *program_id,
        accounts,
        data,
    })
}

/// Creates an 'TransferOut' instruction.
#[cfg(not(target_arch = "bpf"))]
pub fn transfer_out(
    program_id: &Pubkey,
    payer: &Pubkey,
    token_account: &Pubkey,
    token_mint: &Pubkey,
    t: &TransferOutPayload,
) -> Result<Instruction, ProgramError> {
    let data = BridgeInstruction::TransferOut(*t).serialize()?;

    let bridge_key = Bridge::derive_bridge_id(program_id)?;
    let transfer_key = Bridge::derive_transfer_id(
        program_id,
        &bridge_key,
        t.asset.chain,
        t.asset.address,
        t.chain_id,
        t.target,
        token_account.to_bytes(),
        t.nonce,
    )?;

    let mut accounts = vec![
        AccountMeta::new_readonly(*program_id, false),
        AccountMeta::new_readonly(solana_sdk::system_program::id(), false),
        AccountMeta::new_readonly(spl_token::id(), false),
        AccountMeta::new_readonly(solana_sdk::sysvar::clock::id(), false),
        AccountMeta::new(*token_account, false),
        AccountMeta::new(bridge_key, false),
        AccountMeta::new(transfer_key, false),
        AccountMeta::new(*token_mint, false),
        AccountMeta::new(*payer, true),
    ];

    // If the token is a native solana token add a custody account
    if t.asset.chain == CHAIN_ID_SOLANA {
        let custody_key = Bridge::derive_custody_id(program_id, &bridge_key, token_mint)?;
        accounts.push(AccountMeta::new(custody_key, false));
    }

    Ok(Instruction {
        program_id: *program_id,
        accounts,
        data,
    })
}

/// Creates a 'PostVAA' instruction.
#[cfg(not(target_arch = "bpf"))]
pub fn post_vaa(
    program_id: &Pubkey,
    payer: &Pubkey,
    v: VAAData,
) -> Result<Instruction, ProgramError> {
    let mut data = v.clone();
    data.insert(0, 2);

    // Parse VAA
    let vaa = VAA::deserialize(&v[..])?;

    let bridge_key = Bridge::derive_bridge_id(program_id)?;
    let guardian_set_key =
        Bridge::derive_guardian_set_id(program_id, &bridge_key, vaa.guardian_set_index)?;
    let claim_key = Bridge::derive_claim_id(program_id, &bridge_key, vaa.signature_body()?)?;

    let mut accounts = vec![
        AccountMeta::new_readonly(*program_id, false),
        AccountMeta::new_readonly(solana_sdk::system_program::id(), false),
        AccountMeta::new_readonly(solana_sdk::sysvar::clock::id(), false),
        AccountMeta::new(bridge_key, false),
        AccountMeta::new(guardian_set_key, false),
        AccountMeta::new(claim_key, false),
        AccountMeta::new(*payer, true),
    ];

    match vaa.payload.unwrap() {
        VAABody::UpdateGuardianSet(u) => {
            let guardian_set_key =
                Bridge::derive_guardian_set_id(program_id, &bridge_key, u.new_index)?;
            accounts.push(AccountMeta::new(guardian_set_key, false));
        }
        VAABody::Transfer(t) => {
            if t.source_chain == CHAIN_ID_SOLANA {
                // Solana (any) -> Ethereum (any)
                let transfer_key = Bridge::derive_transfer_id(
                    program_id,
                    &bridge_key,
                    t.asset.chain,
                    t.asset.address,
                    t.target_chain,
                    t.target_address,
                    t.source_address,
                    t.nonce,
                )?;
                accounts.push(AccountMeta::new(transfer_key, false))
            } else if t.asset.chain == CHAIN_ID_SOLANA {
                // Foreign (wrapped) -> Solana (native)
                let mint_key = Pubkey::new(&t.asset.address);
                let custody_key = Bridge::derive_custody_id(program_id, &bridge_key, &mint_key)?;
                accounts.push(AccountMeta::new_readonly(spl_token::id(), false));
                accounts.push(AccountMeta::new(mint_key, false));
                accounts.push(AccountMeta::new(Pubkey::new(&t.target_address), false));
                accounts.push(AccountMeta::new(custody_key, false));
            } else {
                // Foreign (native) -> Solana (wrapped)
                let wrapped_key = Bridge::derive_wrapped_asset_id(
                    program_id,
                    &bridge_key,
                    t.asset.chain,
                    t.asset.address,
                )?;
                let wrapped_meta_key =
                    Bridge::derive_wrapped_meta_id(program_id, &bridge_key, &wrapped_key)?;
                accounts.push(AccountMeta::new_readonly(spl_token::id(), false));
                accounts.push(AccountMeta::new(wrapped_key, false));
                accounts.push(AccountMeta::new(Pubkey::new(&t.target_address), false));
                accounts.push(AccountMeta::new(wrapped_meta_key, false));
            }
        }
    }

    Ok(Instruction {
        program_id: *program_id,
        accounts,
        data,
    })
}

/// Creates an 'PokeProposal' instruction.
#[cfg(not(target_arch = "bpf"))]
pub fn poke_proposal(
    program_id: &Pubkey,
    transfer_proposal: &Pubkey,
) -> Result<Instruction, ProgramError> {
    let data = BridgeInstruction::PokeProposal().serialize()?;

    let mut accounts = vec![AccountMeta::new(*transfer_proposal, false)];

    Ok(Instruction {
        program_id: *program_id,
        accounts,
        data,
    })
}

/// Unpacks a reference from a bytes buffer.
pub fn unpack<T>(input: &[u8]) -> Result<&T, ProgramError> {
    if input.len() < size_of::<u8>() + size_of::<T>() {
        return Err(ProgramError::InvalidInstructionData);
    }
    #[allow(clippy::cast_ptr_alignment)]
    let val: &T = unsafe { &*(&input[1] as *const u8 as *const T) };
    Ok(val)
}
