use crate::types::*;
use solitaire::{
    CreationLamports::Exempt,
    *,
};

type Payer<'a> = Signer<Info<'a>>;
type GuardianSet<'a> =
    Derive<Data<'a, GuardianSetData, { AccountState::Uninitialized }>, "GuardianSet">;
type Bridge<'a> = Derive<Data<'a, BridgeData, { AccountState::Uninitialized }>, "Bridge">;

#[derive(FromAccounts, ToInstruction)]
pub struct Initialize<'b> {
    pub bridge: Bridge<'b>,
    pub guardian_set: GuardianSet<'b>,
    pub payer: Payer<'b>,
}

impl<'b> InstructionContext<'b> for Initialize<'b> {
}

pub fn initialize(
    ctx: &ExecutionContext,
    accs: &mut Initialize,
    config: BridgeConfig,
) -> Result<()> {
    // Initialize Guardian Set
    accs.guardian_set.index = 0;
    accs.guardian_set.creation_time = 0;
    accs.guardian_set.keys = Vec::with_capacity(20);

    accs.bridge.create(ctx, accs.payer.key, Exempt)?;

    // Initialize the Bridge state for the first time.
    accs.bridge.guardian_set_index = 0;
    accs.bridge.config = config;

    Ok(())
}
