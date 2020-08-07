#![allow(clippy::too_many_arguments)]
//! Instruction types

use std::mem::size_of;

use primitive_types::U256;
use solana_sdk::{
    instruction::{AccountMeta, Instruction},
    program_error::ProgramError,
    pubkey::Pubkey,
};

use crate::instruction::BridgeInstruction::Initialize;
use crate::state::{AssetMeta, BridgeConfig};
use crate::syscalls::RawKey;

/// chain id of this chain
pub const CHAIN_ID_SOLANA: u8 = 1;

/// size of a VAA in bytes
const VAA_SIZE: usize = 100;

/// size of a foreign address in bytes
const FOREIGN_ADDRESS_SIZE: usize = 32;

/// validator payment approval
pub type VAA_BODY = [u8; VAA_SIZE];
/// X and Y point of P for guardians
pub type GuardianKey = [u8; 64];
/// address on a foreign chain
pub type ForeignAddress = [u8; FOREIGN_ADDRESS_SIZE];

#[repr(C)]
#[derive(Clone, Copy)]
pub struct InitializePayload {
    /// guardians that are allowed to sign mints
    pub initial_guardian: RawKey,
    /// config for the bridge
    pub config: BridgeConfig,
}

#[repr(C)]
#[derive(Clone, Copy)]
pub struct TransferOutPayload {
    /// amount to transfer
    pub amount: U256,
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
    PostVAA(VAA_BODY),

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
                let value = unsafe {
                    &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut InitializePayload)
                };
                *value = payload;
            }
            Self::TransferOut(payload) => {
                output[0] = 1;
                #[allow(clippy::cast_ptr_alignment)]
                let value = unsafe {
                    &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut TransferOutPayload)
                };
                *value = payload;
            }
            Self::PostVAA(payload) => {
                output[0] = 2;
                #[allow(clippy::cast_ptr_alignment)]
                let value =
                    unsafe { &mut *(&mut output[size_of::<u8>()] as *mut u8 as *mut VAA_BODY) };
                *value = payload;
            }
            Self::EvictTransferOut() => {
                output[0] = 3;
            }
            Self::EvictExecutedVAA() => {
                output[0] = 4;
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
    initial_guardian: RawKey,
    config: &BridgeConfig,
) -> Result<Instruction, ProgramError> {
    let data = BridgeInstruction::Initialize(InitializePayload {
        config: *config,
        initial_guardian,
    })
    .serialize()?;

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
