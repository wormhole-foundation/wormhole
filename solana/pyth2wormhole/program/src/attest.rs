use crate::{
    config::P2WConfigAccount,
    types::{PriceAttestation, batch_serialize},
};
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solana_program::{
    clock::Clock,
    instruction::{
        AccountMeta,
        Instruction,
    },
    program::{
        invoke,
        invoke_signed,
    },
    program_error::ProgramError,
    pubkey::Pubkey,
    rent::Rent,
};

use bridge::{
    accounts::BridgeData,
    types::ConsistencyLevel,
    PostMessageData,
};

use solitaire::{
    invoke_seeded,
    trace,
    AccountState,
    Derive,
    ExecutionContext,
    FromAccounts,
    Info,
    InstructionContext,
    Keyed,
    Mut,
    Peel,
    Result as SoliResult,
    Seeded,
    Signer,
    SolitaireError,
    Sysvar,
    ToInstruction,
};

pub type P2WEmitter<'b> = Derive<Info<'b>, "p2w-emitter">;

/// Important: must be manually maintained until native Solitaire
/// variable len vector support.
///
/// The number must reflect how many pyth product/price pairs are
/// expected in the Attest struct below. The constant itself is only
/// used in the on-chain config in order for attesters to learn the
/// correct value dynamically.
pub const P2W_MAX_BATCH_SIZE: u16 = 5;

#[derive(FromAccounts, ToInstruction)]
pub struct Attest<'b> {
    // Payer also used for wormhole
    pub payer: Mut<Signer<Info<'b>>>,
    pub system_program: Info<'b>,
    pub config: P2WConfigAccount<'b, { AccountState::Initialized }>,

    // Hardcoded product/price pairs, bypassing Solitaire's variable-length limitations
    // Any change to the number of accounts must include an appropriate change to P2W_MAX_BATCH_SIZE
    pub pyth_product: Info<'b>,
    pub pyth_price: Info<'b>,

    pub pyth_product2: Option<Info<'b>>,
    pub pyth_price2: Option<Info<'b>>,

    pub pyth_product3: Option<Info<'b>>,
    pub pyth_price3: Option<Info<'b>>,

    pub pyth_product4: Option<Info<'b>>,
    pub pyth_price4: Option<Info<'b>>,

    pub pyth_product5: Option<Info<'b>>,
    pub pyth_price5: Option<Info<'b>>,

    // Did you read the comment near `pyth_product`?
    // pub pyth_product6: Option<Info<'b>>,
    // pub pyth_price6: Option<Info<'b>>,

    // pub pyth_product7: Option<Info<'b>>,
    // pub pyth_price7: Option<Info<'b>>,

    // pub pyth_product8: Option<Info<'b>>,
    // pub pyth_price8: Option<Info<'b>>,

    // pub pyth_product9: Option<Info<'b>>,
    // pub pyth_price9: Option<Info<'b>>,

    // pub pyth_product10: Option<Info<'b>>,
    // pub pyth_price10: Option<Info<'b>>,

    pub clock: Sysvar<'b, Clock>,

    /// Wormhole program address - must match the config value
    pub wh_prog: Info<'b>,

    // wormhole's post_message accounts
    //
    // This contract makes no attempt to exhaustively validate
    // Wormhole's account inputs. Only the wormhole contract address
    // is validated (see above).

    /// Bridge config needed for fee calculation
    pub wh_bridge: Mut<Info<'b>>,

    /// Account to store the posted message
    pub wh_message: Signer<Mut<Info<'b>>>,

    /// Emitter of the VAA
    pub wh_emitter: P2WEmitter<'b>,

    /// Tracker for the emitter sequence
    pub wh_sequence: Mut<Info<'b>>,

    // We reuse our payer
    // pub wh_payer: Mut<Signer<Info<'b>>>,
    /// Account to collect tx fee
    pub wh_fee_collector: Mut<Info<'b>>,

    pub wh_rent: Sysvar<'b, Rent>,
}

#[derive(BorshDeserialize, BorshSerialize)]
pub struct AttestData {
    pub consistency_level: ConsistencyLevel,
}

impl<'b> InstructionContext<'b> for Attest<'b> {
    fn deps(&self) -> Vec<Pubkey> {
        vec![solana_program::system_program::id()]
    }
}

pub fn attest(ctx: &ExecutionContext, accs: &mut Attest, data: AttestData) -> SoliResult<()> {
    accs.config.verify_derivation(ctx.program_id, None)?;

    if accs.config.wh_prog != *accs.wh_prog.key {
        trace!(&format!(
            "Wormhole program account mismatch (expected {:?}, got {:?})",
            accs.config.wh_prog, accs.wh_prog.key
        ));
    }

    // Make the specified prices iterable
    let price_pair_opts = [
        Some(&accs.pyth_product),
        Some(&accs.pyth_price),
        accs.pyth_product2.as_ref(),
        accs.pyth_price2.as_ref(),
        accs.pyth_product3.as_ref(),
        accs.pyth_price3.as_ref(),
        accs.pyth_product4.as_ref(),
        accs.pyth_price4.as_ref(),
        accs.pyth_product5.as_ref(),
        accs.pyth_price5.as_ref(),

	// Did you read the comment near `pyth_product`?
	// accs.pyth_product6.as_ref(),
        // accs.pyth_price6.as_ref(),
        // accs.pyth_product7.as_ref(),
        // accs.pyth_price7.as_ref(),
        // accs.pyth_product8.as_ref(),
        // accs.pyth_price8.as_ref(),
        // accs.pyth_product9.as_ref(),
        // accs.pyth_price9.as_ref(),
        // accs.pyth_product10.as_ref(),
        // accs.pyth_price10.as_ref(),
    ];

    let price_pairs: Vec<_> = price_pair_opts.into_iter().filter_map(|acc| *acc).collect();

    if price_pairs.len() % 2 != 0 {
        trace!(&format!(
            "Uneven product/price count detected: {}",
            price_pairs.len()
        ));
        return Err(ProgramError::InvalidAccountData.into());
    }

    trace!("{} Pyth symbols received", price_pairs.len() / 2);

    // Collect the validated symbols for batch serialization
    let mut attestations = Vec::with_capacity(price_pairs.len() / 2);

    for pair in price_pairs.as_slice().chunks_exact(2) {

	let product = pair[0];
	let price = pair[1];

        if accs.config.pyth_owner != *price.owner
            || accs.config.pyth_owner != *product.owner
        {
            trace!(&format!(
            "Pair {:?} - {:?}: pyth_owner pubkey mismatch (expected {:?}, got product owner {:?} and price owner {:?}",
		
		product, price, 
            accs.config.pyth_owner, product.owner, price.owner
        ));
            return Err(SolitaireError::InvalidOwner(accs.pyth_price.owner.clone()).into());
        }

	let attestation = PriceAttestation::from_pyth_price_bytes(
	    price.key.clone(),
	    accs.clock.unix_timestamp,
	    &*price.try_borrow_data()?,
	)?;

	// The following check is crucial against poorly ordered
	// account inputs, e.g. [Some(prod1), Some(price1),
	// Some(prod2), None, None, Some(price)], interpreted by
	// earlier logic as [(prod1, price1), (prod2, price3)].
	//
	// Failing to verify the product/price relationship could lead
	// to mismatched product/price metadata, which would result in
	// a false attestation.
	if &attestation.product_id != product.key {
	    trace!(&format!(
		"Price's product_id does not match the pased account (points at {:?} instead)",
		attestation.product_id
	    ));
	    return Err(ProgramError::InvalidAccountData.into());
	}

	attestations.push(attestation);
    }

    trace!("Attestations successfully created");

    let bridge_config = BridgeData::try_from_slice(&accs.wh_bridge.try_borrow_mut_data()?)?.config;

    // Pay wormhole fee
    let transfer_ix = solana_program::system_instruction::transfer(
        accs.payer.key,
        accs.wh_fee_collector.info().key,
        bridge_config.fee,
    );
    solana_program::program::invoke(&transfer_ix, ctx.accounts)?;

    // Send payload
    let post_message_data = (
        bridge::instruction::Instruction::PostMessage,
        PostMessageData {
	    nonce: 0, // Superseded by the sequence number
            payload: batch_serialize(attestations.as_slice().iter()).map_err(|e| {
		trace!(e.to_string());
		ProgramError::InvalidAccountData
	    })?,
	    consistency_level: data.consistency_level,
        },
    );

    let ix = Instruction::new_with_bytes(
        accs.config.wh_prog,
        post_message_data.try_to_vec()?.as_slice(),
        vec![
            AccountMeta::new(*accs.wh_bridge.key, false),
            AccountMeta::new(*accs.wh_message.key, true),
            AccountMeta::new_readonly(*accs.wh_emitter.key, true),
            AccountMeta::new(*accs.wh_sequence.key, false),
            AccountMeta::new(*accs.payer.key, true),
            AccountMeta::new(*accs.wh_fee_collector.key, false),
            AccountMeta::new_readonly(*accs.clock.info().key, false),
            AccountMeta::new_readonly(solana_program::sysvar::rent::ID, false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
    );

    trace!("Before cross-call");

    invoke_seeded(&ix, ctx, &accs.wh_emitter, None)?;

    Ok(())
}
