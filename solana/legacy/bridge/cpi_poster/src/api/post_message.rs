use bridge::types::ConsistencyLevel;
use solana_program::program::invoke;
use solitaire::{
    trace,
    *,
};

#[derive(FromAccounts)]
pub struct PostMessage<'b> {
    /// Bridge config needed for fee calculation.
    pub bridge: Mut<Info<'b>>,

    /// Account to store the posted message
    pub message: Signer<Mut<Info<'b>>>,

    /// Emitter of the VAA
    pub emitter: MaybeMut<Info<'b>>,

    /// Tracker for the emitter sequence
    pub sequence: Mut<Info<'b>>,

    /// Payer for account creation
    pub payer: Mut<Info<'b>>,

    /// Account to collect tx fee
    pub fee_collector: Mut<Info<'b>>,

    pub clock: Info<'b>,

    pub bridge_program: Info<'b>,
}

#[derive(BorshDeserialize, BorshSerialize)]
pub struct PostMessageData {
    /// Unique nonce for this message
    pub nonce: u32,

    /// Message payload
    pub payload: Vec<u8>,

    /// Commitment Level required for an attestation to be produced
    pub consistency_level: ConsistencyLevel,
}

pub fn post_message(
    ctx: &ExecutionContext,
    accs: &mut PostMessage,
    data: PostMessageData,
) -> Result<()> {
    let ix = bridge::instructions::post_message(
        *accs.bridge_program.key,
        *accs.payer.key,
        *accs.emitter.key,
        *accs.message.key,
        data.nonce,
        data.payload,
        data.consistency_level,
    )
    .unwrap();
    invoke(&ix, ctx.accounts)?;

    Ok(())
}
