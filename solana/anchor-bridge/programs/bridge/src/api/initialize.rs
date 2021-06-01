use crate::types::*;
use solitaire::*;

type Payer<'a> = Signer<Info<'a>>;
type GuardianSet<'a> = Derive<Data<'a, GuardianSetData, Uninitialized>, "GuardianSet">;
type Bridge<'a> = Derive<Data<'a, BridgeData, Uninitialized>, "Bridge">;

#[derive(FromAccounts, ToAccounts)]
pub struct Initialize<'b> {
    pub payer: Payer<'b>,
    pub guardian_set: GuardianSet<'b>,
    pub bridge: Bridge<'b>,
    pub transfer: Transfer<'b>,
}

impl<'b> InstructionContext<'b> for Initialize<'b> {
}

#[derive(FromAccounts, ToAccounts)]
pub struct Transfer<'b> {
    pub mint: Data<'b, Test, Initialized>,
    pub from: Data<'b, Test, Initialized>,
    pub to: Data<'b, Test, Initialized>,
}

impl<'b> InstructionContext<'b> for Transfer<'b> {
    fn verify(&self) -> Result<()> {
        return if self.mint.mint == self.from.mint {
            Ok(())
        } else {
            Err(SolitaireError::InvalidDerive(*self.mint.0.key).into())
        };
    }
}

#[derive(BorshDeserialize, BorshSerialize)]
pub struct Test {
    mint: Pubkey,
}

pub fn initialize(
    ctx: &ExecutionContext,
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
