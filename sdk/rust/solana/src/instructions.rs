//! Functions for creating instructions for CPI calls.

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};

/// A Copy of the `Instruction` enum from the Solitaire Solana program.
#[repr(u8)]
#[derive(BorshSerialize, BorshDeserialize)]
pub enum Instruction {
    Initialize,
    PostMessage,
    PostVAA,
    SetFees,
    TransferFees,
    UpgradeContract,
    UpgradeGuardianSet,
    VerifySignatures,
    PostMessageUnreliable,
}

mod initialize;
mod post_message;
mod post_vaa;
mod set_fees;
mod verify_signatures;

pub use {
    initialize::initialize,
    post_message::{
        post_message,
        post_message_unreliable,
    },
    post_vaa::post_vaa,
    post_vaa::PostVAAData,
    set_fees::set_fees,
    verify_signatures::{
        verify_signatures,
        verify_signatures_txs,
    },
};

#[cfg(test)]
mod tests {
    use {
        super::Instruction as A,
        bridge::instruction::Instruction as B,
    };

    // This checks that the `Instruction` enum defined in this package has the same variants as the
    // `Instruction` coming from the Solitaire Solana program, the unnecessary appearing match is
    // there to force an exhaustiveness check.
    #[test]
    #[rustfmt::skip]
    fn test_variants() {
        // Macro makes it hard to screw up typo-ing the variant names.
        macro_rules! compare {
            ($variant:ident) => {
                assert_eq!(A::$variant as u8, B::$variant as u8)
            };
        }

        match B::Initialize {
            B::Initialize            => compare!(Initialize),
            B::PostMessage           => compare!(PostMessage),
            B::PostVAA               => compare!(PostVAA),
            B::SetFees               => compare!(SetFees),
            B::TransferFees          => compare!(TransferFees),
            B::UpgradeContract       => compare!(UpgradeContract),
            B::UpgradeGuardianSet    => compare!(UpgradeGuardianSet),
            B::VerifySignatures      => compare!(VerifySignatures),
            B::PostMessageUnreliable => compare!(PostMessageUnreliable),
        }
    }
}
