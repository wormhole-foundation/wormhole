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

#[derive(Debug, Default)]
pub struct AdditionalTestInputs {
    pub payer_is_signer: Option<bool>,
}

#[allow(clippy::too_many_arguments)]
pub fn set_up_transaction(
    tx_payer_signer: &Keypair,
    guardian_set_index: u32,
    total_signatures: u8,
    guardian_signatures: &[[u8; GUARDIAN_SIGNATURE_LENGTH]],
    payer_signer: &Keypair,
    guardian_signatures_signer: &Keypair,
    recent_blockhash: Hash,
    additional_inputs: Option<AdditionalTestInputs>,
) -> VersionedTransaction {
    let tx_payer = tx_payer_signer.pubkey();
    let payer = payer_signer.pubkey();

    let mut post_signatures_ix = verify_vaa::PostSignatures {
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

    let AdditionalTestInputs { payer_is_signer } = additional_inputs.unwrap_or_default();

    let needs_additional_signer = match payer_is_signer {
        Some(is_signer) => {
            post_signatures_ix.accounts[0].is_signer = is_signer;
            is_signer
        }
        None => tx_payer != payer,
    };

    // Adding compute budget instructions to ensure all instructions fit into
    // one transaction.
    //
    // NOTE: Invoking the compute budget costs in total 300 CU.
    let message = Message::try_compile(
        &tx_payer,
        &[
            post_signatures_ix,
            ComputeBudgetInstruction::set_compute_unit_price(69),
            // NOTE: CU limit is higher than needed to resolve errors in test.
            ComputeBudgetInstruction::set_compute_unit_limit(25_000),
        ],
        &[],
        recent_blockhash,
    )
    .unwrap();

    if needs_additional_signer {
        VersionedTransaction::try_new(
            VersionedMessage::V0(message),
            &[tx_payer_signer, payer_signer, guardian_signatures_signer],
        )
        .unwrap()
    } else {
        VersionedTransaction::try_new(
            VersionedMessage::V0(message),
            &[tx_payer_signer, guardian_signatures_signer],
        )
        .unwrap()
    }
}
