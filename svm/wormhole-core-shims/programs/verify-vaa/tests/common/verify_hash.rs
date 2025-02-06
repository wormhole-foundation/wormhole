use solana_sdk::{
    compute_budget::ComputeBudgetInstruction,
    hash::Hash,
    keccak,
    message::{v0::Message, VersionedMessage},
    pubkey::Pubkey,
    signature::Keypair,
    signer::Signer,
    transaction::VersionedTransaction,
};
use wormhole_svm_definitions::{CORE_BRIDGE_PROGRAM_ID, VERIFY_VAA_SHIM_PROGRAM_ID};
use wormhole_svm_shim::verify_vaa::{self, VerifyHashData};

use super::bump_cu_cost;

#[derive(Debug, Default)]
pub struct AdditionalTestInputs {
    pub invalid_guardian_set: Option<Pubkey>,
}

pub struct BumpCosts {
    pub guardian_set: u64,
}

pub fn set_up_transaction(
    tx_payer_signer: &Keypair,
    guardian_set_index: u32,
    guardian_signatures: &Pubkey,
    digest: keccak::Hash,
    recent_blockhash: Hash,
    additional_inputs: Option<AdditionalTestInputs>,
) -> (VersionedTransaction, BumpCosts) {
    let tx_payer = tx_payer_signer.pubkey();

    let AdditionalTestInputs {
        invalid_guardian_set,
    } = additional_inputs.unwrap_or_default();

    let (guardian_set, guardian_set_bump) = invalid_guardian_set.map(|key| (key, 0)).unwrap_or(
        wormhole_svm_definitions::find_guardian_set_address(
            guardian_set_index.to_be_bytes(),
            &CORE_BRIDGE_PROGRAM_ID,
        ),
    );

    let verify_hash_ix = verify_vaa::VerifyHash {
        program_id: &VERIFY_VAA_SHIM_PROGRAM_ID,
        accounts: verify_vaa::VerifyHashAccounts {
            guardian_set: &guardian_set,
            guardian_signatures,
        },
        data: VerifyHashData::new(guardian_set_bump, digest),
    }
    .instruction();

    // Adding compute budget instructions to ensure all instructions fit into
    // one transaction.
    //
    // NOTE: Invoking the compute budget costs in total 300 CU.
    let message = Message::try_compile(
        &tx_payer,
        &[
            verify_hash_ix,
            ComputeBudgetInstruction::set_compute_unit_price(69),
            ComputeBudgetInstruction::set_compute_unit_limit(340_000),
        ],
        &[],
        recent_blockhash,
    )
    .unwrap();

    (
        VersionedTransaction::try_new(VersionedMessage::V0(message), &[tx_payer_signer]).unwrap(),
        BumpCosts {
            guardian_set: bump_cu_cost(guardian_set_bump),
        },
    )
}
