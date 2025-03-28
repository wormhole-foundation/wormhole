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

#[derive(Debug, Default)]
pub struct AdditionalTestInputs {
    pub refund_recipient_is_signer: Option<bool>,
}

pub fn set_up_transaction(
    tx_payer_signer: &Keypair,
    refund_recipient_signer: &Keypair,
    guardian_signatures: &Pubkey,
    recent_blockhash: Hash,
    additional_inputs: Option<AdditionalTestInputs>,
) -> VersionedTransaction {
    let tx_payer = tx_payer_signer.pubkey();
    let refund_recipient = refund_recipient_signer.pubkey();

    let mut close_signatures_ix = verify_vaa::CloseSignatures {
        program_id: &VERIFY_VAA_SHIM_PROGRAM_ID,
        accounts: verify_vaa::CloseSignaturesAccounts {
            guardian_signatures,
            refund_recipient: &refund_recipient,
        },
    }
    .instruction();

    let AdditionalTestInputs {
        refund_recipient_is_signer,
    } = additional_inputs.unwrap_or_default();

    let needs_additional_signer = match refund_recipient_is_signer {
        Some(is_signer) => {
            close_signatures_ix.accounts[1].is_signer = is_signer;
            is_signer
        }
        None => tx_payer != refund_recipient,
    };

    // Adding compute budget instructions to ensure all instructions fit into
    // one transaction.
    //
    // NOTE: Invoking the compute budget costs in total 300 CU.
    let message = Message::try_compile(
        &tx_payer,
        &[
            close_signatures_ix,
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
            &[tx_payer_signer, refund_recipient_signer],
        )
        .unwrap()
    } else {
        VersionedTransaction::try_new(VersionedMessage::V0(message), &[tx_payer_signer]).unwrap()
    }
}
