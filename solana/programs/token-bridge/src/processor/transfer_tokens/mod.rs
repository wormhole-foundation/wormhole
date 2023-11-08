mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

pub use crate::legacy::instruction::TransferTokensArgs;

use crate::error::TokenBridgeError;
use anchor_lang::prelude::*;

pub(self) fn require_valid_relayer_fee(args: &TransferTokensArgs) -> Result<()> {
    // Cannot configure a fee greater than the total transfer amount.
    require!(
        args.relayer_fee <= args.amount,
        TokenBridgeError::InvalidRelayerFee
    );

    // Done.
    Ok(())
}
