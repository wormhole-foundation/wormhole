use crate::{
    accounts::{
        Bridge,
        GuardianSet,
        GuardianSetDerivationData,
    },
    types::*,
};
use solitaire::{
    CreationLamports::Exempt,
    *,
};

type Payer<'a> = Signer<Info<'a>>;

#[derive(FromAccounts, ToInstruction)]
pub struct Initialize<'b> {
    pub bridge: Bridge<'b, { AccountState::Uninitialized }>,
    pub guardian_set: GuardianSet<'b, { AccountState::Uninitialized }>,
    pub payer: Payer<'b>,
}

impl<'b> InstructionContext<'b> for Initialize<'b> {
}

pub fn initialize(
    ctx: &ExecutionContext,
    accs: &mut Initialize,
    config: BridgeConfig,
) -> Result<()> {
    accs.guardian_set.index = 0;
    accs.guardian_set.creation_time = 0;
    accs.guardian_set.keys = Vec::with_capacity(20);
    accs.guardian_set.keys.push([
        0x1d, 0x72, 0x87, 0x7e, 0xb2, 0xd8, 0x98, 0x73, 0x8a, 0xfe, 0x94, 0xc6, 0x10, 0x11, 0x52,
        0xed, 0xe0, 0x43, 0x5d, 0xe9,
    ]);

    // Initialize Guardian Set
    accs.guardian_set.create(
        &GuardianSetDerivationData { index: 0 },
        ctx,
        accs.payer.key,
        Exempt,
    )?;

    // Initialize the Bridge state for the first time.
    accs.bridge.create(ctx, accs.payer.key, Exempt)?;
    accs.bridge.guardian_set_index = 0;
    accs.bridge.config = config;

    Ok(())
}
