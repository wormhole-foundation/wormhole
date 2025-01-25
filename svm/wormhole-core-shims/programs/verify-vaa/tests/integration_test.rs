use solana_program_test::tokio;
use solana_sdk::{
    compute_budget::ComputeBudgetInstruction,
    message::{v0::Message, VersionedMessage},
    signature::Keypair,
    signer::Signer,
    transaction::VersionedTransaction,
};
use wormhole_svm_definitions::{
    borsh::{deserialize_account_data, GuardianSignatures},
    VERIFY_VAA_SHIM_PROGRAM_ID,
};
use wormhole_svm_shim::verify_vaa;

mod common;

const VAA: &str = "AQAAAAQNAL1qji7v9KnngyX0VxK+3fCMVscWTLoYX8L48NWquq2WGrcHd4H0wYc0KF4ZOWjLD2okXoBjGQIDJzx4qIrbSzQBAQq69h+neXGb58VfhZgraPVCxJmnTj8JIDq5jqi3Qav1e+IW51mIJlOhSAdCRbEyQLzf6Z3C19WJJqSyt/z1XF0AAvFgDHkseyMZTE5vQjflu4tc5OLPJe2VYCxTJT15LA02YPrWgOM6HhfUhXDhFoG5AI/s2ApjK8jaqi7LGJILAUMBA6cp4vfko8hYyRvogqQWsdk9e20g0O6s60h4ewweapXCQHerQpoJYdDxlCehN4fuYnuudEhW+6FaXLjwNJBdqsoABDg9qXjXB47nBVCZAGns2eosVqpjkyDaCfo/p1x8AEjBA80CyC1/QlbG9L4zlnnDIfZWylsf3keJqx28+fZNC5oABi6XegfozgE8JKqvZLvd7apDhrJ6Qv+fMiynaXASkafeVJOqgFOFbCMXdMKehD38JXvz3JrlnZ92E+I5xOJaDVgABzDSte4mxUMBMJB9UUgJBeAVsokFvK4DOfvh6G3CVqqDJplLwmjUqFB7fAgRfGcA8PWNStRc+YDZiG66YxPnptwACe84S31Kh9voz2xRk1THMpqHQ4fqE7DizXPNWz6Z6ebEXGcd7UP9PBXoNNvjkLWZJZOdbkZyZqztaIiAo4dgWUABCobiuQP92WjTxOZz0KhfWVJ3YBVfsXUwaVQH4/p6khX0HCEVHR9VHmjvrAAGDMdJGWW+zu8mFQc4gPU6m4PZ6swADO7voA5GWZZPiztz22pftwxKINGvOjCPlLpM1Y2+Vq6AQuez/mlUAmaL0NKgs+5VYcM1SGBz0TL3ABRhKQAhUEMADWmiMo0J1Qaj8gElb+9711ZjvAY663GIyG/E6EdPW+nPKJI9iZE180sLct+krHj0J7PlC9BjDiO2y149oCOJ6FgAEcaVkYK43EpN7XqxrdpanX6R6TaqECgZTjvtN3L6AP2ceQr8mJJraYq+qY8pTfFvPKEqmW9CBYvnA5gIMpX59WsAEjIL9Hdnx+zFY0qSPB1hB9AhqWeBP/QfJjqzqafsczaeCN/rWUf6iNBgXI050ywtEp8JQ36rCn8w6dRhUusn+MEAZ32XyAAAAAAAFczO6yk0j3G90i/+9DoqGcH1teF8XMpUEVKRIBgmcq3lAAAAAAAC/1wAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAC6Q7dAAAAAAAAAAAAAAAAAAoLhpkcYhizbB0Z1KLp6wzjYG60gAAgAAAAAAAAAAAAAAAInNTEvk5b/1WVF+JawF1smtAdicABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==";

// Post signatures.

#[tokio::test]
async fn test_post_signatures_13_at_once() {
    let (mut banks_client, payer_signer, recent_blockhash, decoded_vaa) =
        common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let guardian_signatures_signer = Keypair::new();
    let transaction = common::post_signatures::set_up_transaction(
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &decoded_vaa.guardian_signatures,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(out.simulation_details.unwrap().units_consumed, 13_355);

    banks_client.process_transaction(transaction).await.unwrap();

    // Check guardian signatures account after processing the transaction.
    let guardian_signatures_info = banks_client
        .get_account(guardian_signatures_signer.pubkey())
        .await
        .unwrap()
        .unwrap();

    let account_data = &guardian_signatures_info.data;
    let (expected_length, expected_guardian_signatures_data) =
        common::generate_expected_guardian_signatures_info(
            &payer_signer.pubkey(),
            decoded_vaa.total_signatures,
            decoded_vaa.guardian_set_index,
            decoded_vaa.guardian_signatures,
        );
    assert_eq!(account_data.len(), expected_length);
    assert_eq!(
        deserialize_account_data::<GuardianSignatures>(&account_data[..]).unwrap(),
        expected_guardian_signatures_data
    );
}

#[tokio::test]
async fn test_post_signatures_lamports_already_in_guardian_signatures() {
    let (mut banks_client, payer_signer, recent_blockhash, decoded_vaa) =
        common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let guardian_signatures_signer = Keypair::new();

    // Send lamports to the guardian signatures account.
    let recent_blockhash = common::transfer_lamports(
        &mut banks_client,
        recent_blockhash,
        &payer_signer,
        &guardian_signatures_signer.pubkey(),
        6960 * 128,
    )
    .await;

    let transaction = common::post_signatures::set_up_transaction(
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &decoded_vaa.guardian_signatures,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(out.simulation_details.unwrap().units_consumed, 17_267);

    banks_client.process_transaction(transaction).await.unwrap();

    // Check guardian signatures account after processing the transaction.
    let guardian_signatures_info = banks_client
        .get_account(guardian_signatures_signer.pubkey())
        .await
        .unwrap()
        .unwrap();

    let account_data = &guardian_signatures_info.data;
    let (expected_length, expected_guardian_signatures_data) =
        common::generate_expected_guardian_signatures_info(
            &payer_signer.pubkey(),
            decoded_vaa.total_signatures,
            decoded_vaa.guardian_set_index,
            decoded_vaa.guardian_signatures,
        );
    assert_eq!(account_data.len(), expected_length);
    assert_eq!(
        deserialize_account_data::<GuardianSignatures>(&account_data[..]).unwrap(),
        expected_guardian_signatures_data
    );
}

#[tokio::test]
async fn test_post_signatures_separate_transactions() {
    let (mut banks_client, payer_signer, recent_blockhash, decoded_vaa) =
        common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    // Split up signatures
    let guardian_signatures_1 = &decoded_vaa.guardian_signatures[..12];
    let guardian_signatures_2 = &decoded_vaa.guardian_signatures[12..];
    assert_eq!(guardian_signatures_2.len(), 1);

    let guardian_signatures_signer = Keypair::new();

    // First transaction.

    let transaction = common::post_signatures::set_up_transaction(
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_1,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(out.simulation_details.unwrap().units_consumed, 12_828);

    banks_client.process_transaction(transaction).await.unwrap();

    let guardian_signatures_info_data = banks_client
        .get_account(guardian_signatures_signer.pubkey())
        .await
        .unwrap()
        .unwrap()
        .data;

    let mut expected_guardian_signatures = guardian_signatures_1.to_vec();

    // Check guardian signatures account after processing the transaction.
    let (expected_length, expected_guardian_signatures_data) =
        common::generate_expected_guardian_signatures_info(
            &payer_signer.pubkey(),
            decoded_vaa.total_signatures,
            decoded_vaa.guardian_set_index,
            expected_guardian_signatures.clone(),
        );
    assert_eq!(guardian_signatures_info_data.len(), expected_length);
    assert_eq!(
        deserialize_account_data::<GuardianSignatures>(&guardian_signatures_info_data[..]).unwrap(),
        expected_guardian_signatures_data
    );

    // Second transaction.
    let recent_blockhash = banks_client.get_latest_blockhash().await.unwrap();

    let transaction = common::post_signatures::set_up_transaction(
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_2,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(out.simulation_details.unwrap().units_consumed, 7_628);

    banks_client.process_transaction(transaction).await.unwrap();

    let guardian_signatures_info_data = banks_client
        .get_account(guardian_signatures_signer.pubkey())
        .await
        .unwrap()
        .unwrap()
        .data;

    expected_guardian_signatures.extend_from_slice(guardian_signatures_2);

    // Check guardian signatures account after processing the transaction.
    let (expected_length, expected_guardian_signatures_data) =
        common::generate_expected_guardian_signatures_info(
            &payer_signer.pubkey(),
            decoded_vaa.total_signatures,
            decoded_vaa.guardian_set_index,
            expected_guardian_signatures.clone(),
        );
    assert_eq!(guardian_signatures_info_data.len(), expected_length);
    assert_eq!(
        deserialize_account_data::<GuardianSignatures>(&guardian_signatures_info_data[..]).unwrap(),
        expected_guardian_signatures_data
    );
}

#[tokio::test]
async fn test_cannot_post_signatures_refund_recipient_mismatch() {
    let (mut banks_client, payer_signer, recent_blockhash, decoded_vaa) =
        common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    // Split up signatures
    let guardian_signatures_1 = &decoded_vaa.guardian_signatures[..12];
    let guardian_signatures_2 = &decoded_vaa.guardian_signatures[12..];
    assert_eq!(guardian_signatures_2.len(), 1);

    let guardian_signatures_signer = Keypair::new();

    // First transaction.

    let transaction = common::post_signatures::set_up_transaction(
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_1,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
    );
    banks_client.process_transaction(transaction).await.unwrap();

    // Second transaction.
    let recent_blockhash = banks_client.get_latest_blockhash().await.unwrap();

    let another_payer_signer = Keypair::new();

    // Send some lamports to the another payer.
    let recent_blockhash = common::transfer_lamports(
        &mut banks_client,
        recent_blockhash,
        &payer_signer,
        &another_payer_signer.pubkey(),
        2_000_000_000,
    )
    .await;

    let transaction = common::post_signatures::set_up_transaction(
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_2,
        &another_payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&"Program log: AnchorError thrown in programs/verify-vaa/src/instructions/post_signatures.rs:42. Error Code: WriteAuthorityMismatch. Error Number: 6001. Error Message: WriteAuthorityMismatch.".to_string()))
}

#[tokio::test]
async fn test_cannot_post_signatures_zero_signatures() {
    let (mut banks_client, payer_signer, recent_blockhash, _) = common::start_test(VAA).await;

    let guardian_signatures_signer = Keypair::new();
    let transaction = common::post_signatures::set_up_transaction(
        0,   // guardian set index
        1,   // total signatures
        &[], // guardian signatures
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&"Program log: AnchorError thrown in programs/verify-vaa/src/instructions/post_signatures.rs:24. Error Code: EmptyGuardianSignatures. Error Number: 6000. Error Message: EmptyGuardianSignatures.".to_string()))
}

#[tokio::test]
async fn test_cannot_post_signatures_total_signatures_too_large() {
    let (mut banks_client, payer_signer, recent_blockhash, _) = common::start_test(VAA).await;

    let guardian_signatures_signer = Keypair::new();
    let transaction = common::post_signatures::set_up_transaction(
        0,          // guardian set index
        155,        // total signatures
        &[[0; 66]], // guardian signatures
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&"Account data size realloc limited to 10240 in inner instructions".to_string()))
}

#[tokio::test]
async fn test_cannot_post_signatures_payer_is_guardian_signatures() {
    let (mut banks_client, payer_signer, recent_blockhash, decoded_vaa) =
        common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let guardian_signatures_signer = Keypair::new();

    // First transfer lamports to guardian signatures signer.
    let recent_blockhash = common::transfer_lamports(
        &mut banks_client,
        recent_blockhash,
        &payer_signer,
        &guardian_signatures_signer.pubkey(),
        2_000_000_000,
    )
    .await;

    let guardian_signatures = guardian_signatures_signer.pubkey();

    let post_signatures_ix = verify_vaa::PostSignatures {
        program_id: &VERIFY_VAA_SHIM_PROGRAM_ID,
        accounts: verify_vaa::PostSignaturesAccounts {
            payer: &guardian_signatures,
            guardian_signatures: &guardian_signatures,
        },
        data: verify_vaa::PostSignaturesData::new(
            decoded_vaa.guardian_set_index,
            decoded_vaa.total_signatures,
            &decoded_vaa.guardian_signatures,
        ),
    }
    .instruction();

    // Adding compute budget instructions to ensure all instructions fit into
    // one transaction.
    //
    // NOTE: Invoking the compute budget costs in total 300 CU.
    let message = Message::try_compile(
        &guardian_signatures,
        &[
            post_signatures_ix,
            ComputeBudgetInstruction::set_compute_unit_price(69),
            ComputeBudgetInstruction::set_compute_unit_limit(420_000),
        ],
        &[],
        recent_blockhash,
    )
    .unwrap();

    let transaction = VersionedTransaction::try_new(
        VersionedMessage::V0(message),
        &[&guardian_signatures_signer],
    )
    .unwrap();

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());
    assert!(out.simulation_details.unwrap().logs.contains(&"Program log: AnchorError thrown in programs/verify-vaa/src/instructions/post_signatures.rs:4. Error Code: TryingToInitPayerAsProgramAccount. Error Number: 4101. Error Message: You cannot/should not initialize the payer account as a program account.".to_string()))
}

#[tokio::test]
async fn test_cannot_post_signatures_more_signatures_than_total() {
    let (mut banks_client, payer_signer, recent_blockhash, decoded_vaa) =
        common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let guardian_signatures_signer = Keypair::new();
    let transaction = common::post_signatures::set_up_transaction(
        1,
        decoded_vaa.total_signatures - 1,
        &decoded_vaa.guardian_signatures,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&"Program log: AnchorError caused by account: guardian_signatures. Error Code: AccountDidNotSerialize. Error Number: 3004. Error Message: Failed to serialize the account.".to_string()))
}
