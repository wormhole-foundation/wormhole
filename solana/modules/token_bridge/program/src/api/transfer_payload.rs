use crate::{
    accounts::{
        AuthoritySigner,
        ConfigAccount,
        CoreBridge,
        CustodyAccount,
        CustodyAccountDerivationData,
        CustodySigner,
        EmitterAccount,
        WrappedDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    messages::PayloadTransferWithPayload,
    types::*,
    TokenBridgeError::InvalidChain,
};
use bridge::{
    api::PostMessageData,
    types::ConsistencyLevel,
    vaa::SerializePayload,
    CHAIN_ID_SOLANA,
};
use primitive_types::U256;
use solana_program::{
    account_info::AccountInfo,
    instruction::{
        AccountMeta,
        Instruction,
    },
    sysvar::clock::Clock,
};
use solitaire::{
    processors::seeded::invoke_seeded,
    *,
};
use solana_program::pubkey::Pubkey;

use super::{
    verify_and_execute_native_transfers,
    verify_and_execute_wrapped_transfers,
};

////////////////////////////////////////////////////////////////////////////////
// Sender

#[repr(transparent)]
pub struct Sender<'b>(MaybeMut<Signer<Info<'b>>>);

impl<'a, 'b: 'a, 'c> Peel<'a, 'b, 'c> for Sender<'b> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self>
    where
        Self: Sized,
    {
        Ok(Sender(MaybeMut::peel(ctx)?))
    }

    fn deps() -> Vec<Pubkey> {
        MaybeMut::<Signer<Info<'b>>>::deps()
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        MaybeMut::persist(&self.0, program_id)
    }
}

// May or may not be a PDA, so we don't use [`Derive`], instead implement
// [`Seeded`] directly.
impl<'b> Seeded<()> for Sender<'b> {
    fn seeds(_accs: ()) -> Vec<Vec<u8>> {
        vec![String::from("sender").as_bytes().to_vec()]
    }
}

impl<'a, 'b: 'a> Keyed<'a, 'b> for Sender<'b> {
    fn info(&'a self) -> &Info<'b> {
        &self.0
    }
}

/// TODO(csongor): document
fn derive_sender_address(sender: &Sender, cpi_program_id: &AccountInfo) -> Result<Address> {
    if cpi_program_id.key.to_bytes() == [0; 32] {
        return Ok(sender.info().key.to_bytes());
    } else {
        sender.verify_derivation(cpi_program_id.key, ())?;
        Ok(cpi_program_id.key.to_bytes())
    }
}

////////////////////////////////////////////////////////////////////////////////
// Transfer wrapped with payload

#[derive(FromAccounts)]
pub struct TransferNativeWithPayload<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub from: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub mint: Mut<Data<'b, SplMint, { AccountState::Initialized }>>,
    pub custody: Mut<CustodyAccount<'b, { AccountState::MaybeInitialized }>>,

    // This could allow someone to race someone else's tx if they do the approval in a separate tx.
    // Therefore the approval must be set in the same tx.
    pub authority_signer: AuthoritySigner<'b>,

    pub custody_signer: CustodySigner<'b>,

    /// CPI Context
    pub bridge: Mut<CoreBridge<'b, { AccountState::Initialized }>>,

    /// Account to store the posted message
    pub message: Signer<Mut<Info<'b>>>,

    /// Emitter of the VAA
    pub emitter: EmitterAccount<'b>,

    /// Tracker for the emitter sequence
    pub sequence: Mut<Info<'b>>,

    /// Account to collect tx fee
    pub fee_collector: Mut<Info<'b>>,

    pub clock: Sysvar<'b, Clock>,

    pub sender: Sender<'b>,
    pub cpi_program_id: Info<'b>,
}

impl<'a> From<&TransferNativeWithPayload<'a>> for CustodyAccountDerivationData {
    fn from(accs: &TransferNativeWithPayload<'a>) -> Self {
        CustodyAccountDerivationData {
            mint: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct TransferNativeWithPayloadData {
    pub nonce: u32,
    pub amount: u64,
    pub target_address: Address,
    pub target_chain: ChainID,
    pub payload: Vec<u8>,
}

pub fn transfer_native_with_payload(
    ctx: &ExecutionContext,
    accs: &mut TransferNativeWithPayload,
    data: TransferNativeWithPayloadData,
) -> Result<()> {
    // Prevent transferring to the same chain.
    if data.target_chain == CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }

    let derivation_data: CustodyAccountDerivationData = (&*accs).into();
    let (amount, _fee) = verify_and_execute_native_transfers(
        ctx,
        &derivation_data,
        &accs.payer,
        &accs.from,
        &accs.mint,
        &accs.custody,
        &accs.authority_signer,
        &accs.custody_signer,
        &accs.bridge,
        &accs.fee_collector,
        data.amount,
        0,
    )?;

    // Post message
    let payload = PayloadTransferWithPayload {
        amount: U256::from(amount),
        token_address: accs.mint.info().key.to_bytes(),
        token_chain: CHAIN_ID_SOLANA,
        to: data.target_address,
        to_chain: data.target_chain,
        from_address: derive_sender_address(&accs.sender, &accs.cpi_program_id)?,
        payload: data.payload,
    };
    let params = (
        bridge::instruction::Instruction::PostMessage,
        PostMessageData {
            nonce: data.nonce,
            payload: payload.try_to_vec()?,
            consistency_level: ConsistencyLevel::Finalized,
        },
    );

    let ix = Instruction::new_with_bytes(
        accs.config.wormhole_bridge,
        params.try_to_vec()?.as_slice(),
        vec![
            AccountMeta::new(*accs.bridge.info().key, false),
            AccountMeta::new(*accs.message.key, true),
            AccountMeta::new_readonly(*accs.emitter.key, true),
            AccountMeta::new(*accs.sequence.key, false),
            AccountMeta::new(*accs.payer.key, true),
            AccountMeta::new(*accs.fee_collector.key, false),
            AccountMeta::new_readonly(*accs.clock.info().key, false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            AccountMeta::new_readonly(solana_program::sysvar::rent::ID, false),
        ],
    );
    invoke_seeded(&ix, ctx, &accs.emitter, None)?;

    Ok(())
}

////////////////////////////////////////////////////////////////////////////////
// Transfer wrapped with payload

#[derive(FromAccounts)]
pub struct TransferWrappedWithPayload<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub from: Mut<Data<'b, SplAccount, { AccountState::Initialized }>>,
    pub from_owner: MaybeMut<Signer<Info<'b>>>,
    pub mint: Mut<WrappedMint<'b, { AccountState::Initialized }>>,
    pub wrapped_meta: WrappedTokenMeta<'b, { AccountState::Initialized }>,

    pub authority_signer: AuthoritySigner<'b>,

    /// CPI Context
    pub bridge: Mut<CoreBridge<'b, { AccountState::Initialized }>>,

    /// Account to store the posted message
    pub message: Signer<Mut<Info<'b>>>,

    /// Emitter of the VAA
    pub emitter: EmitterAccount<'b>,

    /// Tracker for the emitter sequence
    pub sequence: Mut<Info<'b>>,

    /// Account to collect tx fee
    pub fee_collector: Mut<Info<'b>>,

    pub clock: Sysvar<'b, Clock>,

    pub sender: Sender<'b>,
    pub cpi_program_id: Info<'b>,
}

impl<'a> From<&TransferWrappedWithPayload<'a>> for WrappedDerivationData {
    fn from(accs: &TransferWrappedWithPayload<'a>) -> Self {
        WrappedDerivationData {
            token_chain: 1,
            token_address: accs.mint.info().key.to_bytes(),
        }
    }
}

impl<'a> From<&TransferWrappedWithPayload<'a>> for WrappedMetaDerivationData {
    fn from(accs: &TransferWrappedWithPayload<'a>) -> Self {
        WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct TransferWrappedWithPayloadData {
    pub nonce: u32,
    pub amount: u64,
    pub target_address: Address,
    pub target_chain: ChainID,
    pub payload: Vec<u8>,
}

pub fn transfer_wrapped_with_payload(
    ctx: &ExecutionContext,
    accs: &mut TransferWrappedWithPayload,
    data: TransferWrappedWithPayloadData,
) -> Result<()> {
    // Prevent transferring to the same chain.
    if data.target_chain == CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }

    let derivation_data: WrappedMetaDerivationData = (&*accs).into();
    verify_and_execute_wrapped_transfers(
        ctx,
        &derivation_data,
        &accs.payer,
        &accs.from,
        &accs.from_owner,
        &accs.mint,
        &accs.wrapped_meta,
        &accs.authority_signer,
        &accs.bridge,
        &accs.fee_collector,
        data.amount,
        0,
    )?;

    // Post message
    let payload = PayloadTransferWithPayload {
        amount: U256::from(data.amount),
        token_address: accs.wrapped_meta.token_address,
        token_chain: accs.wrapped_meta.chain,
        to: data.target_address,
        to_chain: data.target_chain,
        from_address: derive_sender_address(&accs.sender, &accs.cpi_program_id)?,
        payload: data.payload,
    };
    let params = (
        bridge::instruction::Instruction::PostMessage,
        PostMessageData {
            nonce: data.nonce,
            payload: payload.try_to_vec()?,
            consistency_level: ConsistencyLevel::Finalized,
        },
    );

    let ix = Instruction::new_with_bytes(
        accs.config.wormhole_bridge,
        params.try_to_vec()?.as_slice(),
        vec![
            AccountMeta::new(*accs.bridge.info().key, false),
            AccountMeta::new(*accs.message.key, true),
            AccountMeta::new_readonly(*accs.emitter.key, true),
            AccountMeta::new(*accs.sequence.key, false),
            AccountMeta::new(*accs.payer.key, true),
            AccountMeta::new(*accs.fee_collector.key, false),
            AccountMeta::new_readonly(*accs.clock.info().key, false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            AccountMeta::new_readonly(solana_program::sysvar::rent::ID, false),
        ],
    );
    invoke_seeded(&ix, ctx, &accs.emitter, None)?;

    Ok(())
}

