use solana_sdk::{
    compute_budget::ComputeBudgetInstruction,
    hash::Hash,
    message::{v0::Message, VersionedMessage},
    signature::Keypair,
    signer::Signer,
    transaction::VersionedTransaction,
};
use wormhole_svm_definitions::{GUARDIAN_SIGNATURE_LENGTH, VERIFY_VAA_SHIM_PROGRAM_ID};
use wormhole_svm_shim::verify_vaa;

pub fn set_up_transaction(
    guardian_set_index: u32,
    total_signatures: u8,
    guardian_signatures: &[[u8; GUARDIAN_SIGNATURE_LENGTH]],
    payer_signer: &Keypair,
    guardian_signatures_signer: &Keypair,
    recent_blockhash: Hash,
) -> VersionedTransaction {
    let payer = payer_signer.pubkey();

    let post_signatures_ix = verify_vaa::PostSignatures {
        program_id: &VERIFY_VAA_SHIM_PROGRAM_ID,
        accounts: verify_vaa::PostSignaturesAccounts {
            payer: &payer,
            guardian_signatures: &guardian_signatures_signer.pubkey(),
        },
        data: verify_vaa::PostSignaturesData::new(
            guardian_set_index,
            total_signatures,
            guardian_signatures,
        ),
    }
    .instruction();

    // Adding compute budget instructions to ensure all instructions fit into
    // one transaction.
    //
    // NOTE: Invoking the compute budget costs in total 300 CU.
    let message = Message::try_compile(
        &payer,
        &[
            post_signatures_ix,
            ComputeBudgetInstruction::set_compute_unit_price(69),
            ComputeBudgetInstruction::set_compute_unit_limit(8_000),
        ],
        &[],
        recent_blockhash,
    )
    .unwrap();

    VersionedTransaction::try_new(
        VersionedMessage::V0(message),
        &[&payer_signer, &guardian_signatures_signer],
    )
    .unwrap()
}
