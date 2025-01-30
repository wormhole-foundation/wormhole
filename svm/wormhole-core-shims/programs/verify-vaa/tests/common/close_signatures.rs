use solana_sdk::{
    compute_budget::ComputeBudgetInstruction,
    hash::Hash,
    message::{v0::Message, VersionedMessage},
    pubkey::Pubkey,
    signature::Keypair,
    signer::Signer,
    transaction::VersionedTransaction,
};
use wormhole_svm_definitions::VERIFY_VAA_SHIM_PROGRAM_ID;
use wormhole_svm_shim::verify_vaa;

pub fn set_up_transaction(
    refund_recipient_signer: &Keypair,
    guardian_signatures: &Pubkey,
    recent_blockhash: Hash,
) -> VersionedTransaction {
    let refund_recipient = refund_recipient_signer.pubkey();

    let post_signatures_ix = verify_vaa::CloseSignatures {
        program_id: &VERIFY_VAA_SHIM_PROGRAM_ID,
        accounts: verify_vaa::CloseSignaturesAccounts {
            guardian_signatures,
            refund_recipient: &refund_recipient,
        },
    }
    .instruction();

    // Adding compute budget instructions to ensure all instructions fit into
    // one transaction.
    //
    // NOTE: Invoking the compute budget costs in total 300 CU.
    let message = Message::try_compile(
        &refund_recipient,
        &[
            post_signatures_ix,
            ComputeBudgetInstruction::set_compute_unit_price(69),
            ComputeBudgetInstruction::set_compute_unit_limit(1_100),
        ],
        &[],
        recent_blockhash,
    )
    .unwrap();

    VersionedTransaction::try_new(VersionedMessage::V0(message), &[&refund_recipient_signer])
        .unwrap()
}
