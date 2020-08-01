#![allow(clippy::too_many_arguments)]

//! Instruction types

use std::io::Write;
use std::mem::size_of;

use solana_sdk::{
    instruction::{AccountMeta, Instruction},
    program_error::ProgramError,
    pubkey::Pubkey,
};
use zerocopy::{AsBytes, FromBytes};

use crate::error::Error;
use crate::instruction::BridgeInstruction::Initialize;
use crate::state::{AssetMeta, BridgeConfig};

/// chain id of this chain
pub const CHAIN_ID_SOLANA: u8 = 1;

/// size of a VAA in bytes
const VAA_SIZE: usize = 32;

/// size of a foreign address in bytes
const FOREIGN_ADDRESS_SIZE: usize = 32;

/// validator payment approval
pub type VAA = [u8; VAA_SIZE];
/// X and Y point of P for guardians
pub type GuardianKey = [u8; 64];
/// address on a foreign chain
pub type ForeignAddress = [u8; FOREIGN_ADDRESS_SIZE];

#[repr(C)]
#[derive(Clone, Copy)]
pub struct InitializePayload {
    /// guardians that are allowed to sign mints
    pub initial_guardian: GuardianKey,
    /// config for the bridge
    pub config: BridgeConfig,
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct TransferOutPayload {
    /// amount to transfer
    pub amount: u64,
    /// chain id to transfer to
    pub chain_id: u8,
    /// Information about the asset to be transferred
    pub asset: AssetMeta,
    /// address on the foreign chain to transfer to
    pub target: ForeignAddress,
}

/// Instructions supported by the SwapInfo program.
#[repr(C)]
pub enum BridgeInstruction {
    /// Initializes a new Bridge
    Initialize(InitializePayload),

    /// Burns or locks a (wrapped) asset `token` from `sender` on the Solana chain.
    TransferOut(TransferOutPayload),

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
                    let value = unsafe { &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut InitializePayload) };
                *value = payload;
            }
            Self::TransferOut(payload) => {
                output[0] = 1;
                #[allow(clippy::cast_ptr_alignment)]
                    let value = unsafe { &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut TransferOutPayload) };
                *value = payload;
            }
            Self::PostVAA(payload) => {
                output[0] = 2;
                #[allow(clippy::cast_ptr_alignment)]
                    let value = unsafe { &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut VAA) };
                *value = payload;
            }
            Self::EvictTransferOut() => {
                output[0] = 3;
            }
            Self::EvictExecutedVAA() => {
                output[0] = 4;
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
    sender: &Pubkey,
    bridge: &Pubkey,
    initial_guardian: GuardianKey,
    config: &BridgeConfig,
) -> Result<Instruction, ProgramError> {
    let data = BridgeInstruction::Initialize(InitializePayload {
        config: *config,
        initial_guardian,
    }).serialize()?;

    let accounts = vec![
        AccountMeta::new(*sender, true),
        AccountMeta::new(*bridge, false),
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
