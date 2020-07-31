#![allow(clippy::too_many_arguments)]

//! Instruction types

use std::mem::size_of;

use solana_sdk::{
    instruction::{AccountMeta, Instruction},
    program_error::ProgramError,
    pubkey::Pubkey,
};

use crate::error::Error;
use crate::state::BridgeConfig;

/// size of a VAA in bytes
const VAA_SIZE: usize = 32;

/// chain id of this chain
const CHAIN_ID_SOLANA: usize = 1;
/// size of a foreign address in bytes
const FOREIGN_ADDRESS_SIZE: usize = 32;

/// validator payment approval
pub type VAA = [u8; VAA_SIZE];
/// address on a foreign chain
pub type ForeignAddress = [u8; FOREIGN_ADDRESS_SIZE];

/// Instructions supported by the SwapInfo program.
#[repr(C)]
#[derive(Clone, Debug, PartialEq)]
pub enum BridgeInstruction {
    /// Initializes a new Bridge
    Initialize {
        /// guardians that are allowed to sign mints
        initial_guardian: Pubkey,
        /// config for the bridge
        config: BridgeConfig,
    },

    /// Burns a wrapped asset `token` from `sender` on the Solana chain.
    Lock {
        /// amount to transfer
        amount: u64,
        /// chain id to transfer to
        chain_id: u8,
        /// address on the foreign chain to transfer to
        foreign_address: ForeignAddress,
    },

    /// Locks a Solana native token (spl-token) `token` from `sender` on the Solana chain by
    /// transferring it to the `custody_account`.
    LockNative {
        /// amount to transfer
        amount: u64,
        /// chain id to transfer to
        chain_id: u8,
        /// address on the foreign chain to transfer to
        foreign_address: ForeignAddress,
    },

    /// Submits a VAA signed by `guardian` on a valid `proposal`.
    PostVAA(VAA),

    /// Deletes a `proposal` after the `VAA_EXPIRATION_TIME` is over to free up space on chain.
    /// This returns the rent to the sender.
    EvictTransferOut(),

    /// Deletes a `ExecutedVAA` after the `VAA_EXPIRATION_TIME` is over to free up space on chain.
    /// This returns the rent to the sender.
    EvictExecutedVAA(),
}

impl BridgeInstruction {
    /// Deserializes a byte buffer into an [SwapInstruction](enum.SwapInstruction.html).
    pub fn deserialize(input: &[u8]) -> Result<Self, ProgramError> {
        if input.len() < size_of::<u8>() {
            return Err(ProgramError::InvalidAccountData);
        }
        Ok(match input[0] {
            _ => return Err(ProgramError::InvalidInstructionData),
        })
    }

    /// Serializes an [SwapInstruction](enum.SwapInstruction.html) into a byte buffer.
    pub fn serialize(self: &Self) -> Result<Vec<u8>, ProgramError> {
        let mut output = vec![0u8; size_of::<BridgeInstruction>()];
        match self {
            Self::Initialize { initial_guardian, config } => {
                output[0] = 0;
                #[allow(clippy::cast_ptr_alignment)]
                    let value = unsafe { &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut Pubkey) };
                *value = *initial_guardian;
                let value =
                    unsafe { &mut *(&mut output[size_of::<u8>() + size_of::<Pubkey>()] as *mut u8 as *mut BridgeConfig) };
                *value = *config;
            }
            _ => {
                panic!("")
            }
        }
        Ok(output)
    }
}

/// Creates an 'initialize' instruction.
pub fn initialize(
    program_id: &Pubkey,
    token_program_id: &Pubkey,
    swap_pubkey: &Pubkey,
    authority_pubkey: &Pubkey,
    token_a_pubkey: &Pubkey,
    token_b_pubkey: &Pubkey,
    pool_pubkey: &Pubkey,
    user_output_pubkey: &Pubkey,
) -> Result<Instruction, ProgramError> {
    let data = BridgeInstruction::Initialize {
        config: BridgeConfig {
            vaa_expiration_time: 21
        },
        initial_guardian: Pubkey::default(),
    }.serialize()?;

    let accounts = vec![
        AccountMeta::new(*swap_pubkey, true),
        AccountMeta::new(*authority_pubkey, false),
        AccountMeta::new(*token_a_pubkey, false),
        AccountMeta::new(*token_b_pubkey, false),
        AccountMeta::new(*pool_pubkey, false),
        AccountMeta::new(*user_output_pubkey, false),
        AccountMeta::new(*token_program_id, false),
    ];

    Ok(Instruction {
        program_id: *program_id,
        accounts,
        data,
    })
}

/// Unpacks a reference from a bytes buffer.
pub fn unpack<T>(input: &[u8]) -> Result<&T, ProgramError> {
    if input.len() < size_of::<u8>() + size_of::<T>() {
        return Err(ProgramError::InvalidAccountData);
    }
    #[allow(clippy::cast_ptr_alignment)]
        let val: &T = unsafe { &*(&input[1] as *const u8 as *const T) };
    Ok(val)
}
