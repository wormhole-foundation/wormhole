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
    pubkey::Pubkey,
    sysvar::clock::Clock,
};
use solitaire::{
    processors::seeded::invoke_seeded,
    *,
};

use super::{
    verify_and_execute_native_transfers,
    verify_and_execute_wrapped_transfers,
};

////////////////////////////////////////////////////////////////////////////////
// Sender

#[repr(transparent)]
pub struct SenderAccount<'b>(pub MaybeMut<Signer<Info<'b>>>);

impl<'a, 'b: 'a> Peel<'a, 'b> for SenderAccount<'b> {
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self>
    where
        Self: Sized,
    {
        Ok(SenderAccount(MaybeMut::peel(ctx)?))
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        MaybeMut::persist(&self.0, program_id)
    }
}

// May or may not be a PDA, so we don't use [`Derive`], instead implement
// [`Seeded`] directly.
impl<'b> Seeded<()> for SenderAccount<'b> {
    fn seeds(_accs: ()) -> Vec<Vec<u8>> {
        vec![String::from("sender").as_bytes().to_vec()]
    }
}

impl<'a, 'b: 'a> Keyed<'a, 'b> for SenderAccount<'b> {
    fn info(&'a self) -> &Info<'b> {
        &self.0
    }
}

impl<'b> SenderAccount<'b> {
    /// Transfers with payload also include the address of the account or contract
    /// that sent the transfer. Semantically this is identical to "msg.sender" on
    /// EVM chains, i.e. it is the address of the immediate caller of the token
    /// bridge transaction.
    /// Since on Solana, a transaction can have multiple different signers, getting
    /// this information is not so straightforward.
    /// The strategy we use to figure out the sender of the transaction is to
    /// require an additional signer ([`SenderAccount`]) for the transaction.
    /// If the transaction was sent by a user wallet directly, then this may just be
    /// the wallet's pubkey. If, however, the transaction was initiated by a
    /// program, then we require this to be a PDA derived from the sender program's
    /// id and the string "sender". In this case, the sender program must also
    /// attach its program id to the instruction data. If the PDA verification
    /// succeeds (thereby proving that [[`cpi_program_id`]] indeed signed the
    /// transaction), then the program's id is attached to the VAA as the sender,
    /// otherwise the transaction is rejected.
    ///
    /// Note that a program may opt to forego the PDA derivation and instead just
    /// pass on the original wallet as the wallet account (or any other signer, as
    /// long as they don't provide their program_id in the instruction data). The
    /// sender address is provided as a means for protocols to verify on the
    /// receiving end that the message was emitted by a contract they trust, so
    /// foregoing this check is not advised. If the receiving contract needs to know
    /// the sender wallet's address too, then that information can be included in
    /// the additional payload, along with any other data that the protocol needs to
    /// send across. The legitimacy of the attached data can be verified by checking
    /// that the sender contract is a trusted one.
    ///
    /// Also note that attaching the correct PDA as [[`SenderAccount`]] but missing the
    /// [[`cpi_program_id`]] field will result in a successful transaction, but in
    /// that case the PDA's address will directly be encoded into the payload
    /// instead of the sender program's id.
    fn derive_sender_address(&self, cpi_program_id: &Option<Pubkey>) -> Result<Address> {
        match cpi_program_id {
            Some(cpi_program_id) => {
                self.verify_derivation(cpi_program_id, ())?;
                Ok(cpi_program_id.to_bytes())
            }
            None => Ok(self.info().key.to_bytes()),
        }
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

    /// See [`derive_sender_address`]
    pub sender: SenderAccount<'b>,
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
    /// See [`derive_sender_address`]
    pub cpi_program_id: Option<Pubkey>,
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
        from_address: accs.sender.derive_sender_address(&data.cpi_program_id)?,
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

    /// See [`derive_sender_address`]
    pub sender: SenderAccount<'b>,
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
    /// See [`derive_sender_address`]
    pub cpi_program_id: Option<Pubkey>,
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
        from_address: accs.sender.derive_sender_address(&data.cpi_program_id)?,
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
