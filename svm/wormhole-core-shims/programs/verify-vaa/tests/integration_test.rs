use solana_program_test::tokio;
use solana_sdk::{
    compute_budget::ComputeBudgetInstruction,
    message::{v0::Message, VersionedMessage},
    pubkey::Pubkey,
    signature::Keypair,
    signer::Signer,
    transaction::VersionedTransaction,
};
use wormhole_svm_definitions::{
    borsh::{deserialize_with_discriminator, GuardianSignatures},
    compute_keccak_digest, VERIFY_VAA_SHIM_PROGRAM_ID,
};
use wormhole_svm_shim::verify_vaa;

mod common;

const VAA: &str = "AQAAAAQNAL1qji7v9KnngyX0VxK+3fCMVscWTLoYX8L48NWquq2WGrcHd4H0wYc0KF4ZOWjLD2okXoBjGQIDJzx4qIrbSzQBAQq69h+neXGb58VfhZgraPVCxJmnTj8JIDq5jqi3Qav1e+IW51mIJlOhSAdCRbEyQLzf6Z3C19WJJqSyt/z1XF0AAvFgDHkseyMZTE5vQjflu4tc5OLPJe2VYCxTJT15LA02YPrWgOM6HhfUhXDhFoG5AI/s2ApjK8jaqi7LGJILAUMBA6cp4vfko8hYyRvogqQWsdk9e20g0O6s60h4ewweapXCQHerQpoJYdDxlCehN4fuYnuudEhW+6FaXLjwNJBdqsoABDg9qXjXB47nBVCZAGns2eosVqpjkyDaCfo/p1x8AEjBA80CyC1/QlbG9L4zlnnDIfZWylsf3keJqx28+fZNC5oABi6XegfozgE8JKqvZLvd7apDhrJ6Qv+fMiynaXASkafeVJOqgFOFbCMXdMKehD38JXvz3JrlnZ92E+I5xOJaDVgABzDSte4mxUMBMJB9UUgJBeAVsokFvK4DOfvh6G3CVqqDJplLwmjUqFB7fAgRfGcA8PWNStRc+YDZiG66YxPnptwACe84S31Kh9voz2xRk1THMpqHQ4fqE7DizXPNWz6Z6ebEXGcd7UP9PBXoNNvjkLWZJZOdbkZyZqztaIiAo4dgWUABCobiuQP92WjTxOZz0KhfWVJ3YBVfsXUwaVQH4/p6khX0HCEVHR9VHmjvrAAGDMdJGWW+zu8mFQc4gPU6m4PZ6swADO7voA5GWZZPiztz22pftwxKINGvOjCPlLpM1Y2+Vq6AQuez/mlUAmaL0NKgs+5VYcM1SGBz0TL3ABRhKQAhUEMADWmiMo0J1Qaj8gElb+9711ZjvAY663GIyG/E6EdPW+nPKJI9iZE180sLct+krHj0J7PlC9BjDiO2y149oCOJ6FgAEcaVkYK43EpN7XqxrdpanX6R6TaqECgZTjvtN3L6AP2ceQr8mJJraYq+qY8pTfFvPKEqmW9CBYvnA5gIMpX59WsAEjIL9Hdnx+zFY0qSPB1hB9AhqWeBP/QfJjqzqafsczaeCN/rWUf6iNBgXI050ywtEp8JQ36rCn8w6dRhUusn+MEAZ32XyAAAAAAAFczO6yk0j3G90i/+9DoqGcH1teF8XMpUEVKRIBgmcq3lAAAAAAAC/1wAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAC6Q7dAAAAAAAAAAAAAAAAAAoLhpkcYhizbB0Z1KLp6wzjYG60gAAgAAAAAAAAAAAAAAAInNTEvk5b/1WVF+JawF1smtAdicABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==";

// Post signatures.

#[tokio::test]
async fn test_post_signatures_13_at_once() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let guardian_signatures_signer = Keypair::new();
    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &decoded_vaa.guardian_signatures,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(
        out.simulation_details.unwrap().units_consumed,
        // 13_355
        3_343
    );

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
        deserialize_with_discriminator::<GuardianSignatures>(&account_data[..]).unwrap(),
        expected_guardian_signatures_data
    );
}

#[tokio::test]
async fn test_post_signatures_lamports_already_in_guardian_signatures() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let guardian_signatures_signer = Keypair::new();

    // Send lamports to the guardian signatures account.
    let recent_blockhash = common::transfer_lamports(
        &banks_client,
        recent_blockhash,
        &payer_signer,
        &guardian_signatures_signer.pubkey(),
        6_960 * 128,
    )
    .await;

    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &decoded_vaa.guardian_signatures,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(
        out.simulation_details.unwrap().units_consumed,
        // 17_267
        5_654
    );

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
        deserialize_with_discriminator::<GuardianSignatures>(&account_data[..]).unwrap(),
        expected_guardian_signatures_data
    );
}

#[tokio::test]
async fn test_post_signatures_separate_transactions() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    // Split up signatures
    let guardian_signatures_1 = &decoded_vaa.guardian_signatures[..12];
    let guardian_signatures_2 = &decoded_vaa.guardian_signatures[12..];
    assert_eq!(guardian_signatures_2.len(), 1);

    let guardian_signatures_signer = Keypair::new();

    // First transaction.

    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_1,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(
        out.simulation_details.unwrap().units_consumed,
        // 12_828
        3_343
    );

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
        deserialize_with_discriminator::<GuardianSignatures>(&guardian_signatures_info_data[..])
            .unwrap(),
        expected_guardian_signatures_data
    );

    // Second transaction.
    let recent_blockhash = banks_client.get_latest_blockhash().await.unwrap();

    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_2,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(
        out.simulation_details.unwrap().units_consumed,
        // 7_628
        1_007
    );

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
        deserialize_with_discriminator::<GuardianSignatures>(&guardian_signatures_info_data[..])
            .unwrap(),
        expected_guardian_signatures_data
    );
}

#[tokio::test]
async fn test_cannot_post_signatures_refund_recipient_mismatch() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    // Split up signatures
    let guardian_signatures_1 = &decoded_vaa.guardian_signatures[..12];
    let guardian_signatures_2 = &decoded_vaa.guardian_signatures[12..];
    assert_eq!(guardian_signatures_2.len(), 1);

    let guardian_signatures_signer = Keypair::new();

    // First transaction.

    let (_, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_1,
        recent_blockhash,
        Some(&guardian_signatures_signer),
    )
    .await;

    let another_payer_signer = Keypair::new();

    // Send some lamports to the another payer.
    let recent_blockhash = common::transfer_lamports(
        &banks_client,
        recent_blockhash,
        &payer_signer,
        &another_payer_signer.pubkey(),
        2_000_000_000,
    )
    .await;

    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_2,
        &another_payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError thrown in programs/verify-vaa/src/instructions/post_signatures.rs:42. Error Code: WriteAuthorityMismatch. Error Number: 6001. Error Message: WriteAuthorityMismatch.";
    let err_msg = "Program log: Payer (account #1) must match refund recipient";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_post_signatures_total_signatures_mismatch() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    // Split up signatures
    let guardian_signatures_1 = &decoded_vaa.guardian_signatures[..12];
    let guardian_signatures_2 = &decoded_vaa.guardian_signatures[12..];
    assert_eq!(guardian_signatures_2.len(), 1);

    let guardian_signatures_signer = Keypair::new();

    // First transaction.

    let (_, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_1,
        recent_blockhash,
        Some(&guardian_signatures_signer),
    )
    .await;

    // Second transaction.

    let another_payer_signer = Keypair::new();

    // Send some lamports to the another payer.
    let recent_blockhash = common::transfer_lamports(
        &banks_client,
        recent_blockhash,
        &payer_signer,
        &another_payer_signer.pubkey(),
        2_000_000_000,
    )
    .await;

    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures - 1,
        guardian_signatures_2,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    let err_msg = "Program log: Total signatures mismatch";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_post_signatures_zero_signatures() {
    let (banks_client, payer_signer, recent_blockhash, _) = common::start_test(VAA).await;

    let guardian_signatures_signer = Keypair::new();
    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        0,   // guardian set index
        1,   // total signatures
        &[], // guardian signatures
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError thrown in programs/verify-vaa/src/instructions/post_signatures.rs:24. Error Code: EmptyGuardianSignatures. Error Number: 6000. Error Message: EmptyGuardianSignatures.";
    let err_msg = "Program log: Guardian signatures must not be empty";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()));
}

#[tokio::test]
async fn test_cannot_post_signatures_total_signatures_too_large() {
    let (banks_client, payer_signer, recent_blockhash, _) = common::start_test(VAA).await;

    let guardian_signatures_signer = Keypair::new();
    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        0,          // guardian set index
        155,        // total signatures
        &[[0; 66]], // guardian signatures
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    let err_msg = "Account data size realloc limited to 10240 in inner instructions";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_post_signatures_payer_is_guardian_signatures() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let guardian_signatures_signer = Keypair::new();

    // First transfer lamports to guardian signatures signer.
    let recent_blockhash = common::transfer_lamports(
        &banks_client,
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
            ComputeBudgetInstruction::set_compute_unit_limit(8_000),
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
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError thrown in programs/verify-vaa/src/instructions/post_signatures.rs:4. Error Code: TryingToInitPayerAsProgramAccount. Error Number: 4101. Error Message: You cannot/should not initialize the payer account as a program account.";
    let err_msg =
        "Program log: Guardian signatures (account #2) cannot be initialized as payer (account #1)";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_post_signatures_more_signatures_than_total() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let guardian_signatures_signer = Keypair::new();
    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        1,
        decoded_vaa.total_signatures - 1,
        &decoded_vaa.guardian_signatures,
        &payer_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError caused by account: guardian_signatures. Error Code: AccountDidNotSerialize. Error Number: 3004. Error Message: Failed to serialize the account.";
    let err_msg = "Program log: Too many input guardian signatures";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_post_signatures_refund_recipient_is_not_signer() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    // Split up signatures
    let guardian_signatures_1 = &decoded_vaa.guardian_signatures[..12];
    let guardian_signatures_2 = &decoded_vaa.guardian_signatures[12..];
    assert_eq!(guardian_signatures_2.len(), 1);

    let guardian_signatures_signer = Keypair::new();

    // First transaction.

    let (_, refund_recipient_signer, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_1,
        recent_blockhash,
        Some(&guardian_signatures_signer),
    )
    .await;

    let transaction = common::post_signatures::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        guardian_signatures_2,
        &refund_recipient_signer,
        &guardian_signatures_signer,
        recent_blockhash,
        Some(common::post_signatures::AdditionalTestInputs {
            payer_is_signer: Some(false),
        }),
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    let err_msg = "Program log: Payer (account #1) must be a signer";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

// Close signatures.

#[tokio::test]
async fn test_close_signatures() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let (guardian_signatures, refund_recipient_signer, recent_blockhash) =
        common::send_post_signatures_transaction(
            &banks_client,
            &payer_signer,
            decoded_vaa.guardian_set_index,
            decoded_vaa.total_signatures,
            &decoded_vaa.guardian_signatures,
            recent_blockhash,
            None, // guardian_signatures_signer
        )
        .await;

    let transaction = common::close_signatures::set_up_transaction(
        &payer_signer,
        &refund_recipient_signer,
        &guardian_signatures,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(
        out.simulation_details.unwrap().units_consumed,
        // 5_165
        715
    );

    banks_client.process_transaction(transaction).await.unwrap();

    // Check that the guardian signatures account does not exist anymore.
    //
    // NOTE: Because there are only two accounts in this instruction, we can
    // assume the refund recipient received all of the lamports from the
    // guardian signatures account.
    assert!(banks_client
        .get_account(guardian_signatures)
        .await
        .unwrap()
        .is_none());
}

#[tokio::test]
async fn test_cannot_close_signatures_refund_recipient_mismatch() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &decoded_vaa.guardian_signatures,
        recent_blockhash,
        None, // guardian_signatures_signer
    )
    .await;

    let another_refund_recipient_signer = Keypair::new();

    // Send some lamports to the another payer.
    let recent_blockhash = common::transfer_lamports(
        &banks_client,
        recent_blockhash,
        &payer_signer,
        &another_refund_recipient_signer.pubkey(),
        2_000_000_000,
    )
    .await;

    let transaction = common::close_signatures::set_up_transaction(
        &payer_signer,
        &another_refund_recipient_signer,
        &guardian_signatures,
        recent_blockhash,
        None, // additional inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError caused by account: guardian_signatures. Error Code: ConstraintHasOne. Error Number: 2001. Error Message: A has one constraint was violated.";
    let err_msg = "Program log: Refund recipient (account #2) mismatch";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()));
}

#[tokio::test]
async fn test_cannot_close_signatures_refund_recipient_is_not_signer() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let (guardian_signatures, refund_recipient_signer, recent_blockhash) =
        common::send_post_signatures_transaction(
            &banks_client,
            &payer_signer,
            decoded_vaa.guardian_set_index,
            decoded_vaa.total_signatures,
            &decoded_vaa.guardian_signatures,
            recent_blockhash,
            None, // guardian_signatures_signer
        )
        .await;

    let transaction = common::close_signatures::set_up_transaction(
        &payer_signer,
        &refund_recipient_signer,
        &guardian_signatures,
        recent_blockhash,
        Some(common::close_signatures::AdditionalTestInputs {
            refund_recipient_is_signer: Some(false),
        }),
    );

    let out = banks_client
        .simulate_transaction(transaction.clone())
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    let err_msg = "Program log: Refund recipient (account #2) must be a signer";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

// Verify hash.

#[tokio::test]
async fn test_verify_hash() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &decoded_vaa.guardian_signatures,
        recent_blockhash,
        None, // guardian_signatures_signer
    )
    .await;

    let message_hash = solana_sdk::keccak::hash(&decoded_vaa.body);
    let digest = compute_keccak_digest(message_hash, None);

    let (transaction, bump_costs) = common::verify_hash::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        &guardian_signatures,
        digest,
        recent_blockhash,
        None, // additional_inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_ok());
    assert_eq!(
        out.simulation_details.unwrap().units_consumed - bump_costs.guardian_set,
        // 342_276
        334_558
    );
}

#[tokio::test]
async fn test_cannot_verify_hash_invalid_guardian_set() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &decoded_vaa.guardian_signatures,
        recent_blockhash,
        None, // guardian_signatures_signer
    )
    .await;

    let message_hash = solana_sdk::keccak::hash(&decoded_vaa.body);
    let digest = compute_keccak_digest(message_hash, None);

    let (transaction, _) = common::verify_hash::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        &guardian_signatures,
        digest,
        recent_blockhash,
        Some(common::verify_hash::AdditionalTestInputs {
            invalid_guardian_set: Some(Pubkey::new_unique()),
        }),
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError caused by account: guardian_set. Error Code: AccountNotInitialized. Error Number: 3012. Error Message: The program expected this account to be already initialized.";
    let err_msg = "Program log: Guardian set (account #1) seeds constraint violated";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_verify_hash_invalid_guardian_set_index() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &decoded_vaa.guardian_signatures,
        recent_blockhash,
        None, // guardian_signatures_signer
    )
    .await;

    let message_hash = solana_sdk::keccak::hash(&decoded_vaa.body);
    let digest = compute_keccak_digest(message_hash, None);

    let (transaction, _) = common::verify_hash::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index - 1,
        &guardian_signatures,
        digest,
        recent_blockhash,
        None, // additional_inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError caused by account: guardian_set. Error Code: ConstraintSeeds. Error Number: 2006. Error Message: A seeds constraint was violated.";
    let err_msg = "Program log: Guardian set (account #1) address creation failed";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_verify_hash_expired_guardian_set() {
    const VAA_EXPIRED_SET: &str = "AQAAAAMNAnTlXVVK+fjEIYfgQggRENn0/f++V7dCtNCHnrSj05X8ctnm0x1Fzn8hODqvC44/eWTUso+tUPQpHjCgEP0g1xMBA/S7K8O34D/AkEkYLrQbgKFDq6W3uDycJ90B75GmLaniGrmPxDBX1gog6ISTqrDIB9OBL1e3fVGqaMOTCrrPS6gABFrsiVNLrIsU2hPJHWF0c/CT2+DWS+TAq05lfSVvVExfLjv6WejfY3PdN+dLbCKE0JpD14BiNbcYeRXGJxPE76sBBs8nYy7KLd5McKmEGvhN2zBdlHcNByoJf8ZK1WXifnUeWhHnnWDr05wWigqw657ZjRytNVzgFA+MruATReXsxM0BB+NIIlTBvvqF0WZ5FNpT53nzmcpypZYDIU6041CcY2FqU5ntQdEsjZDiu1Q4W0Sjjcfg3xfApMURec+5Q3IP0isBCTjCgO4J/+tgfxazGf8WC2kMRn21Dh1E1u6QG52wul7wIhCQTejShVbWG1U4f8y5mk3wA/X4qHVK3tizoozeJHsACjGD+X3upTjNNQZFfqMxYrnDxRLK6UCGSTb++1AlWHwNQGBfbPf/upVZBO1qIHAiEGQllkyTg4aEoinp38bSN9oADDcEbPbb9R+g1VTWOg8VS8ucHEX4ojahNghH/n6r9tOKfMwfprBtnasT5y1NjSDO7uvt9WTDMTgUCtDdtluz4ToBDQd4w/uKG0pQziUsWMZZeeNTgrdTP1LBJ/6eTWMmMV7QGHep0XwCsEhXOmIVgrDcny6+g8GOpS7bV5Y9d5WQBpYAD2xbXI9zaXN/mRqsk6dF87dzpYHqf4lGPXkv7sKACUZyI71qwyTfjua0XAuNpsUPWLT1pXEa5iIBpgzR8A81M04AEHOKvkaYNNmg6WXsq8noXD4iA3Q8ibC7r4mOkPsOhqPmYX1SzR4g3jLRSM/Ck4BTGo7hVXKv37zMcrAddEErqiIBESYgyM+I7XUdNNWGZ4lWxSnc5WF65DiHH1U8nRsZ5RiPJFic1xp8Zg/5uil3sRUKcXti6M3coO4N9x4W++PxV2MBEqUxI7EX5evuk6uHyoh8VVYYvVAo1XWt8Yx8sDlDqNLOWJaxpGzq0WbB8EUPpRlDzG8YgZVyXl49ZEFj/vMxgW0BZhTIZwAAAAAAAgAAAAAAAAAAAAAAAO4MzphZqQ+m4rdRggW+9MeZtxbXAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAlQL5AASAAAAAAAAAAAAAAAAtTNofvd0WQkzaMQ+lfjfHCtaH3oAAAAAAAAAAAAAAACVPJV2dXAP9gZNOrN+ppSrFdP9wgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAC0W8AD2fGpZ2cDa08dvhobF0SoEEHFqAoL7PN5HTYzDEM0lCf6DptD8OMI950PhoAt0aoIpOjLblmX6I3DGC27gAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB";

    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) =
        common::start_test(VAA_EXPIRED_SET).await;
    assert_eq!(decoded_vaa.total_signatures, 13);
    assert_eq!(decoded_vaa.guardian_set_index, 3);

    let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &decoded_vaa.guardian_signatures,
        recent_blockhash,
        None, // guardian_signatures_signer
    )
    .await;

    let message_hash = solana_sdk::keccak::hash(&decoded_vaa.body);
    let digest = compute_keccak_digest(message_hash, None);

    let (transaction, _) = common::verify_hash::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        &guardian_signatures,
        digest,
        recent_blockhash,
        None, // additional_inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError thrown in programs/verify-vaa/src/instructions/verify_hash.rs:44. Error Code: GuardianSetExpired. Error Number: 6002. Error Message: GuardianSetExpired.";
    let err_msg = "Program log: Guardian set (account #1) is expired";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_verify_hash_no_quorum() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let mut insufficient_guardian_signatures = decoded_vaa.guardian_signatures.clone();
    insufficient_guardian_signatures.pop();

    let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &insufficient_guardian_signatures,
        recent_blockhash,
        None, // guardian_signatures_signer
    )
    .await;

    let message_hash = solana_sdk::keccak::hash(&decoded_vaa.body);
    let digest = compute_keccak_digest(message_hash, None);

    let (transaction, _) = common::verify_hash::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        &guardian_signatures,
        digest,
        recent_blockhash,
        None, // additional_inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError thrown in programs/verify-vaa/src/instructions/verify_hash.rs:56. Error Code: NoQuorum. Error Number: 6003. Error Message: NoQuorum.";
    let err_msg = "Program log: Guardian signatures (account #2) fails to meet quorum";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_verify_hash_non_increasing_guardian_index() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let mut non_increasing_guardian_signatures = decoded_vaa.guardian_signatures.clone();
    non_increasing_guardian_signatures.rotate_right(1);

    let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &non_increasing_guardian_signatures,
        recent_blockhash,
        None, // guardian_signatures_signer
    )
    .await;

    let message_hash = solana_sdk::keccak::hash(&decoded_vaa.body);
    let digest = compute_keccak_digest(message_hash, None);

    let (transaction, _) = common::verify_hash::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        &guardian_signatures,
        digest,
        recent_blockhash,
        None, // additional_inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError thrown in programs/verify-vaa/src/instructions/verify_hash.rs:69. Error Code: InvalidGuardianIndexNonIncreasing. Error Number: 6005. Error Message: InvalidGuardianIndexNonIncreasing.";
    let err_msg = "Program log: Guardian signatures (account #2) has non-increasing guardian index";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_verify_hash_guardian_index_out_of_range() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let mut out_of_range_guardian_signatures = decoded_vaa.guardian_signatures.clone();
    out_of_range_guardian_signatures[0][0] = 19;

    let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &out_of_range_guardian_signatures,
        recent_blockhash,
        None, // guardian_signatures_signer
    )
    .await;

    let message_hash = solana_sdk::keccak::hash(&decoded_vaa.body);
    let digest = compute_keccak_digest(message_hash, None);

    let (transaction, _) = common::verify_hash::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        &guardian_signatures,
        digest,
        recent_blockhash,
        None, // additional_inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError thrown in programs/verify-vaa/src/instructions/verify_hash.rs:78. Error Code: InvalidGuardianIndexOutOfRange. Error Number: 6006. Error Message: InvalidGuardianIndexOutOfRange.";
    let err_msg = "Program log: Guardian signatures (account #2) guardian index out of range";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}

#[tokio::test]
async fn test_cannot_verify_hash_invalid_signature() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    for i in 0..13 {
        let mut out_of_range_guardian_signatures = decoded_vaa.guardian_signatures.clone();
        out_of_range_guardian_signatures[i][65] = 255;

        let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
            &banks_client,
            &payer_signer,
            decoded_vaa.guardian_set_index,
            decoded_vaa.total_signatures,
            &out_of_range_guardian_signatures,
            recent_blockhash,
            None, // guardian_signatures_signer
        )
        .await;

        let message_hash = solana_sdk::keccak::hash(&decoded_vaa.body);
        let digest = compute_keccak_digest(message_hash, None);

        let (transaction, _) = common::verify_hash::set_up_transaction(
            &payer_signer,
            decoded_vaa.guardian_set_index,
            &guardian_signatures,
            digest,
            recent_blockhash,
            None, // additional_inputs
        );

        let out = banks_client
            .simulate_transaction(transaction)
            .await
            .unwrap();
        assert!(
            out.result.unwrap().is_err(),
            "Unexpected success at index {}",
            i
        );

        // let err_msg = "Program log: AnchorError occurred. Error Code: InvalidSignature. Error Number: 6004. Error Message: InvalidSignature.";
        let err_msg = format!("Program log: Guardian signature index {} is invalid", i);
        assert!(
            out.simulation_details.unwrap().logs.contains(&err_msg),
            "Unexpected error message at index {}",
            i
        );
    }
}

#[tokio::test]
async fn test_cannot_verify_hash_invalid_guardian_recovery() {
    let (banks_client, payer_signer, recent_blockhash, decoded_vaa) = common::start_test(VAA).await;
    assert_eq!(decoded_vaa.total_signatures, 13);

    let mut out_of_range_guardian_signatures = decoded_vaa.guardian_signatures.clone();

    let mismatched_signature: [u8; 65] = out_of_range_guardian_signatures[11][1..]
        .try_into()
        .unwrap();
    out_of_range_guardian_signatures[12][1..].copy_from_slice(&mismatched_signature);

    let (guardian_signatures, _, recent_blockhash) = common::send_post_signatures_transaction(
        &banks_client,
        &payer_signer,
        decoded_vaa.guardian_set_index,
        decoded_vaa.total_signatures,
        &out_of_range_guardian_signatures,
        recent_blockhash,
        None, // guardian_signatures_signer
    )
    .await;

    let message_hash = solana_sdk::keccak::hash(&decoded_vaa.body);
    let digest = compute_keccak_digest(message_hash, None);

    let (transaction, _) = common::verify_hash::set_up_transaction(
        &payer_signer,
        decoded_vaa.guardian_set_index,
        &guardian_signatures,
        digest,
        recent_blockhash,
        None, // additional_inputs
    );

    let out = banks_client
        .simulate_transaction(transaction)
        .await
        .unwrap();
    assert!(out.result.unwrap().is_err());

    // let err_msg = "Program log: AnchorError thrown in programs/verify-vaa/src/instructions/verify_hash.rs:127. Error Code: InvalidGuardianKeyRecovery. Error Number: 6007. Error Message: InvalidGuardianKeyRecovery.";
    let err_msg = "Program log: Guardian signature index 12 does not recover guardian 18 pubkey";
    assert!(out
        .simulation_details
        .unwrap()
        .logs
        .contains(&err_msg.to_string()))
}
