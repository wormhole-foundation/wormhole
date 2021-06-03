use crate::types::*;
use solitaire::*;

type Payer<'a> = Signer<Info<'a>>;
type GuardianSet<'a> = Derive<Data<'a, GuardianSetData, Uninitialized>, "GuardianSet">;
type Bridge<'a> = Derive<Data<'a, BridgeData, Uninitialized>, "Bridge">;

#[derive(FromAccounts, ToAccounts)]
pub struct Initialize<'b> {
    pub bridge: Bridge<'b>,
    pub guardian_set: GuardianSet<'b>,
    pub payer: Payer<'b>,
}

impl<'b> InstructionContext<'b> for Initialize<'b> {
}

pub fn initialize(
    _ctx: &ExecutionContext,
    accs: &mut Initialize,
    config: BridgeConfig,
) -> Result<()> {
    // Initialize the Guardian Set for the first time.
    let index = Index::new(0);

    // Initialize Guardian Set
    accs.guardian_set.index = index;
    accs.guardian_set.creation_time = 0;
    accs.guardian_set.keys = Vec::with_capacity(20);

    // Initialize the Bridge state for the first time.
    accs.bridge.guardian_set_index = index;
    accs.bridge.config = config;

    Ok(())
}
