use crate::{
    accounts::{
        AuthoritySigner,
        ConfigAccount,
        CustodyAccount,
        CustodyAccountDerivationData,
        CustodySigner,
        EmitterAccount,
        Endpoint,
        EndpointDerivationData,
        MintSigner,
        SplTokenMeta,
        SplTokenMetaDerivationData,
        WrappedDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    api::{
        complete_transfer::{
            CompleteNativeData,
            CompleteWrappedData,
        },
        AttestTokenData,
        CreateWrappedData,
        RegisterChainData,
        SenderAccount,
        TransferNativeData,
        TransferWrappedData,
        UpgradeContractData,
    },
    messages::{
        PayloadAssetMeta,
        PayloadGovernanceRegisterChain,
        PayloadTransfer,
        PayloadTransferWithPayload,
    },
    CompleteNativeWithPayloadData,
    CompleteWrappedWithPayloadData,
    TransferNativeWithPayloadData,
    TransferWrappedWithPayloadData,
};
use borsh::BorshSerialize;
use bridge::{
    accounts::{
        Bridge,
        Claim,
        ClaimDerivationData,
        FeeCollector,
        Sequence,
        SequenceDerivationData,
    },
    api::ForeignAddress,
    PostVAAData,
    CHAIN_ID_SOLANA,
};
use solana_program::{
    instruction::{
        AccountMeta,
        Instruction,
    },
    pubkey::Pubkey,
};
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};

pub fn initialize(
    program_id: Pubkey,
    payer: Pubkey,
    bridge: Pubkey,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(config_key, false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
        ],
        data: (crate::instruction::Instruction::Initialize, bridge).try_to_vec()?,
    })
}

#[allow(clippy::too_many_arguments)]
pub fn complete_native(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    to: Pubkey,
    fee_recipient: Option<Pubkey>,
    mint: Pubkey,
    data: CompleteNativeData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(program_id, message_key, vaa.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: vaa.emitter_chain,
            emitter_address: vaa.emitter_address,
        },
        &program_id,
    );
    let custody_key = CustodyAccount::<'_, { AccountState::Initialized }>::key(
        &CustodyAccountDerivationData { mint },
        &program_id,
    );
    let custody_signer_key = CustodySigner::key(None, &program_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            message_acc,
            claim_acc,
            AccountMeta::new_readonly(endpoint, false),
            AccountMeta::new(to, false),
            if let Some(fee_r) = fee_recipient {
                AccountMeta::new(fee_r, false)
            } else {
                AccountMeta::new(to, false)
            },
            AccountMeta::new(custody_key, false),
            AccountMeta::new_readonly(mint, false),
            AccountMeta::new_readonly(custody_signer_key, false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (crate::instruction::Instruction::CompleteNative, data).try_to_vec()?,
    })
}

#[allow(clippy::too_many_arguments)]
pub fn complete_native_with_payload(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    to: Pubkey,
    to_owner: Pubkey,
    fee_recipient: Option<Pubkey>,
    mint: Pubkey,
    data: CompleteNativeWithPayloadData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(program_id, message_key, vaa.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: vaa.emitter_chain,
            emitter_address: vaa.emitter_address,
        },
        &program_id,
    );
    let custody_key = CustodyAccount::<'_, { AccountState::Initialized }>::key(
        &CustodyAccountDerivationData { mint },
        &program_id,
    );
    let custody_signer_key = CustodySigner::key(None, &program_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            message_acc,
            claim_acc,
            AccountMeta::new_readonly(endpoint, false),
            AccountMeta::new(to, false),
            AccountMeta::new_readonly(to_owner, true),
            if let Some(fee_r) = fee_recipient {
                AccountMeta::new(fee_r, false)
            } else {
                AccountMeta::new(to, false)
            },
            AccountMeta::new(custody_key, false),
            AccountMeta::new_readonly(mint, false),
            AccountMeta::new_readonly(custody_signer_key, false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (
            crate::instruction::Instruction::CompleteNativeWithPayload,
            data,
        )
            .try_to_vec()?,
    })
}

#[allow(clippy::too_many_arguments)]
pub fn complete_wrapped(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    payload: PayloadTransfer,
    to: Pubkey,
    fee_recipient: Option<Pubkey>,
    data: CompleteWrappedData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(program_id, message_key, vaa.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: vaa.emitter_chain,
            emitter_address: vaa.emitter_address,
        },
        &program_id,
    );
    let mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain: payload.token_chain,
            token_address: payload.token_address,
        },
        &program_id,
    );
    let meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedMetaDerivationData { mint_key },
        &program_id,
    );
    let mint_authority_key = MintSigner::key(None, &program_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            message_acc,
            claim_acc,
            AccountMeta::new_readonly(endpoint, false),
            AccountMeta::new(to, false),
            if let Some(fee_r) = fee_recipient {
                AccountMeta::new(fee_r, false)
            } else {
                AccountMeta::new(to, false)
            },
            AccountMeta::new(mint_key, false),
            AccountMeta::new_readonly(meta_key, false),
            AccountMeta::new_readonly(mint_authority_key, false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (crate::instruction::Instruction::CompleteWrapped, data).try_to_vec()?,
    })
}

#[allow(clippy::too_many_arguments)]
pub fn complete_wrapped_with_payload(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    payload: PayloadTransferWithPayload,
    to: Pubkey,
    to_owner: Pubkey,
    fee_recipient: Option<Pubkey>,
    data: CompleteWrappedWithPayloadData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(program_id, message_key, vaa.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: vaa.emitter_chain,
            emitter_address: vaa.emitter_address,
        },
        &program_id,
    );
    let mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain: payload.token_chain,
            token_address: payload.token_address,
        },
        &program_id,
    );
    let meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedMetaDerivationData { mint_key },
        &program_id,
    );
    let mint_authority_key = MintSigner::key(None, &program_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            message_acc,
            claim_acc,
            AccountMeta::new_readonly(endpoint, false),
            AccountMeta::new(to, false),
            AccountMeta::new_readonly(to_owner, true),
            if let Some(fee_r) = fee_recipient {
                AccountMeta::new(fee_r, false)
            } else {
                AccountMeta::new(to, false)
            },
            AccountMeta::new(mint_key, false),
            AccountMeta::new_readonly(meta_key, false),
            AccountMeta::new_readonly(mint_authority_key, false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (
            crate::instruction::Instruction::CompleteWrappedWithPayload,
            data,
        )
            .try_to_vec()?,
    })
}

pub fn create_wrapped(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    payload: PayloadAssetMeta,
    data: CreateWrappedData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(program_id, message_key, vaa.clone());
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: vaa.emitter_chain,
            emitter_address: vaa.emitter_address,
        },
        &program_id,
    );
    let mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain: payload.token_chain,
            token_address: payload.token_address,
        },
        &program_id,
    );
    let mint_meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedMetaDerivationData { mint_key },
        &program_id,
    );
    let mint_authority_key = MintSigner::key(None, &program_id);
    // SPL Metadata
    let spl_metadata = SplTokenMeta::key(
        &SplTokenMetaDerivationData { mint: mint_key },
        &spl_token_metadata::id(),
    );

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            AccountMeta::new_readonly(endpoint, false),
            message_acc,
            claim_acc,
            AccountMeta::new(mint_key, false),
            AccountMeta::new(mint_meta_key, false),
            AccountMeta::new(spl_metadata, false),
            AccountMeta::new_readonly(mint_authority_key, false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
            AccountMeta::new_readonly(spl_token_metadata::id(), false),
        ],
        data: (crate::instruction::Instruction::CreateWrapped, data).try_to_vec()?,
    })
}

pub fn register_chain(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
    payload: PayloadGovernanceRegisterChain,
    data: RegisterChainData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let (message_acc, claim_acc) = claimable_vaa(program_id, message_key, vaa);
    let endpoint = Endpoint::<'_, { AccountState::Initialized }>::key(
        &EndpointDerivationData {
            emitter_chain: payload.chain,
            emitter_address: payload.endpoint_address,
        },
        &program_id,
    );

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            AccountMeta::new(endpoint, false),
            message_acc,
            claim_acc,
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
        ],
        data: (crate::instruction::Instruction::RegisterChain, data).try_to_vec()?,
    })
}

fn claimable_vaa(
    bridge_id: Pubkey,
    message_key: Pubkey,
    vaa: PostVAAData,
) -> (AccountMeta, AccountMeta) {
    let claim_key = Claim::<'_>::key(
        &ClaimDerivationData {
            emitter_address: vaa.emitter_address,
            emitter_chain: vaa.emitter_chain,
            sequence: vaa.sequence,
        },
        &bridge_id,
    );

    (
        AccountMeta::new_readonly(message_key, false),
        AccountMeta::new(claim_key, false),
    )
}

/// Required accounts
///
/// | name             | account                                                           | signer |
/// |------------------+-------------------------------------------------------------------+--------|
/// | payer            | Pubkey                                                            | true   |
/// | config           | PDA(program_id, \["config"\])                                     | false  |
/// | from             | Pubkey                                                            | false  |
/// | mint             | Pubkey                                                            | false  |
/// | custody          | PDA(program_id, \[mint\])                                         | false  |
/// | authority_signer | PDA(program_id, \["authority_signer"\])                           | false  |
/// | custody_signer   | PDA(program_id, \["custody_signer"\])                             | false  |
/// | bridge_config    | PDA(bridge_id,  \["Bridge"\])                                     | false  |
/// | message          | Pubkey                                                            | true   |
/// | emitter          | PDA(program_id, \["emitter"\])                                    | false  |
/// | sequence         | PDA(bridge_id,  \["Sequence", emitter\])                          | false  |
/// | fee_collector    | PDA(bridge_id,  \["fee_collector"\])                              | false  |
/// | rent             | rent sysvar                                                       | false  |
/// | system_program   | system program                                                    | false  |
/// | bridge_id        | bridge_id program                                                 | false  |
/// | spl_token        | spl_token program                                                 | false  |
pub fn transfer_native(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    from: Pubkey,
    mint: Pubkey,
    data: TransferNativeData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let custody_key = CustodyAccount::<'_, { AccountState::Initialized }>::key(
        &CustodyAccountDerivationData { mint },
        &program_id,
    );

    let authority_signer_key = AuthoritySigner::key(None, &program_id);
    let custody_signer_key = CustodySigner::key(None, &program_id);
    let emitter_key = EmitterAccount::key(None, &program_id);

    // Bridge keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let sequence_key = Sequence::key(
        &SequenceDerivationData {
            emitter_key: &emitter_key,
        },
        &bridge_id,
    );
    let fee_collector_key = FeeCollector::key(None, &bridge_id);

    let instruction = crate::instruction::Instruction::TransferNative;

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            AccountMeta::new(from, false),
            AccountMeta::new(mint, false),
            AccountMeta::new(custody_key, false),
            AccountMeta::new_readonly(authority_signer_key, false),
            AccountMeta::new_readonly(custody_signer_key, false),
            AccountMeta::new(bridge_config, false),
            AccountMeta::new(message_key, true),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (instruction, data).try_to_vec()?,
    })
}

/// Required accounts
///
/// | name             | account                                                                | signer |
/// |------------------+------------------------------------------------------------------------+--------|
/// | payer            | Pubkey                                                                 | true   |
/// | config           | PDA(program_id, \["config"\])                                          | false  |
/// | from             | Pubkey                                                                 | false  |
/// | mint             | Pubkey                                                                 | false  |
/// | custody          | PDA(program_id, \[mint\])                                              | false  |
/// | authority_signer | PDA(program_id, \["authority_signer"\])                                | false  |
/// | custody_signer   | PDA(program_id, \["custody_signer"\])                                  | false  |
/// | bridge_config    | PDA(bridge_id,  \["Bridge"\])                                          | false  |
/// | message          | Pubkey                                                                 | true   |
/// | emitter          | PDA(program_id, \["emitter"\])                                         | false  |
/// | sequence         | PDA(bridge_id,  \["Sequence", emitter\])                               | false  |
/// | fee_collector    | PDA(bridge_id,  \["fee_collector"\])                                   | false  |
/// | clock            | clock sysvar                                                           | false  |
/// | sender           | if Some(p) = data.cpi_program_id then PDA(p, \["sender"\]) else payer  | true   |
/// | rent             | rent sysvar                                                            | false  |
/// | system_program   | system program                                                         | false  |
/// | bridge_id        | bridge_id program                                                      | false  |
/// | spl_token        | spl_token program                                                      | false  |
pub fn transfer_native_with_payload(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    from: Pubkey,
    mint: Pubkey,
    data: TransferNativeWithPayloadData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let custody_key = CustodyAccount::<'_, { AccountState::Initialized }>::key(
        &CustodyAccountDerivationData { mint },
        &program_id,
    );

    let authority_signer_key = AuthoritySigner::key(None, &program_id);
    let custody_signer_key = CustodySigner::key(None, &program_id);
    let emitter_key = EmitterAccount::key(None, &program_id);

    // Bridge keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let sequence_key = Sequence::key(
        &SequenceDerivationData {
            emitter_key: &emitter_key,
        },
        &bridge_id,
    );
    let fee_collector_key = FeeCollector::key(None, &bridge_id);

    let sender = match data.cpi_program_id {
        Some(cpi_program_id) => SenderAccount::key((), &cpi_program_id),
        None => payer,
    };

    let instruction = crate::instruction::Instruction::TransferNativeWithPayload;

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            AccountMeta::new(from, false),
            AccountMeta::new(mint, false),
            AccountMeta::new(custody_key, false),
            AccountMeta::new_readonly(authority_signer_key, false),
            AccountMeta::new_readonly(custody_signer_key, false),
            AccountMeta::new(bridge_config, false),
            AccountMeta::new(message_key, true),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            AccountMeta::new(sender, true),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (instruction, data).try_to_vec()?,
    })
}

/// Required accounts
///
/// | name             | account                                                                | signer |
/// |------------------+------------------------------------------------------------------------+--------|
/// | payer            | Pubkey                                                                 | true   |
/// | config           | PDA(program_id, \["config"\])                                          | false  |
/// | from             | Pubkey                                                                 | false  |
/// | from_owner       | Pubkey                                                                 | true   |
/// | wrapped_mint     | PDA(program_id, \["wrapped", token_chain, token_address\])             | false  |
/// | wrapped_meta     | PDA(program_id, \["meta", wrapped_mint\])                              | false  |
/// | authority_signer | PDA(program_id, \["authority_signer"\])                                | false  |
/// | bridge_config    | PDA(bridge_id,  \["Bridge"\])                                          | false  |
/// | message          | Pubkey                                                                 | true   |
/// | emitter          | PDA(program_id, \["emitter"\])                                         | false  |
/// | sequence         | PDA(bridge_id,  \["Sequence", emitter\])                               | false  |
/// | fee_collector    | PDA(bridge_id,  \["fee_collector"\])                                   | false  |
/// | clock            | clock sysvar                                                           | false  |
/// | rent             | rent sysvar                                                            | false  |
/// | system_program   | system program                                                         | false  |
/// | bridge_id        | bridge_id program                                                      | false  |
/// | spl_token        | spl_token program                                                      | false  |
#[allow(clippy::too_many_arguments)]
pub fn transfer_wrapped(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    from: Pubkey,
    from_owner: Pubkey,
    token_chain: u16,
    token_address: ForeignAddress,
    data: TransferWrappedData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);

    let wrapped_mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain,
            token_address,
        },
        &program_id,
    );
    let wrapped_meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedMetaDerivationData {
            mint_key: wrapped_mint_key,
        },
        &program_id,
    );

    let authority_signer = AuthoritySigner::key(None, &program_id);
    let emitter_key = EmitterAccount::key(None, &program_id);

    // Bridge keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let sequence_key = Sequence::key(
        &SequenceDerivationData {
            emitter_key: &emitter_key,
        },
        &bridge_id,
    );
    let fee_collector_key = FeeCollector::key(None, &bridge_id);

    let instruction = crate::instruction::Instruction::TransferWrapped;

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            AccountMeta::new(from, false),
            AccountMeta::new_readonly(from_owner, true),
            AccountMeta::new(wrapped_mint_key, false),
            AccountMeta::new_readonly(wrapped_meta_key, false),
            AccountMeta::new_readonly(authority_signer, false),
            AccountMeta::new(bridge_config, false),
            AccountMeta::new(message_key, true),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (instruction, data).try_to_vec()?,
    })
}

/// Required accounts
///
/// | name             | account                                                                | signer |
/// |------------------+------------------------------------------------------------------------+--------|
/// | payer            | Pubkey                                                                 | true   |
/// | config           | PDA(program_id, \["config"\])                                          | false  |
/// | from             | Pubkey                                                                 | false  |
/// | from_owner       | Pubkey                                                                 | true   |
/// | wrapped_mint     | PDA(program_id, \["wrapped", token_chain, token_address\])             | false  |
/// | wrapped_meta     | PDA(program_id, \["meta", wrapped_mint\])                              | false  |
/// | authority_signer | PDA(program_id, \["authority_signer"\])                                | false  |
/// | bridge_config    | PDA(bridge_id,  \["Bridge"\])                                          | false  |
/// | message          | Pubkey                                                                 | true   |
/// | emitter          | PDA(program_id, \["emitter"\])                                         | false  |
/// | sequence         | PDA(bridge_id,  \["Sequence", emitter\])                               | false  |
/// | fee_collector    | PDA(bridge_id,  \["fee_collector"\])                                   | false  |
/// | clock            | clock sysvar                                                           | false  |
/// | sender           | if Some(p) = data.cpi_program_id then PDA(p, \["sender"\]) else payer  | true   |
/// | rent             | rent sysvar                                                            | false  |
/// | system_program   | system program                                                         | false  |
/// | bridge_id        | bridge_id program                                                      | false  |
/// | spl_token        | spl_token program                                                      | false  |
#[allow(clippy::too_many_arguments)]
pub fn transfer_wrapped_with_payload(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    from: Pubkey,
    from_owner: Pubkey,
    token_chain: u16,
    token_address: ForeignAddress,
    data: TransferWrappedWithPayloadData,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);

    let wrapped_mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain,
            token_address,
        },
        &program_id,
    );
    let wrapped_meta_key = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedMetaDerivationData {
            mint_key: wrapped_mint_key,
        },
        &program_id,
    );

    let authority_signer = AuthoritySigner::key(None, &program_id);
    let emitter_key = EmitterAccount::key(None, &program_id);

    // Bridge keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let sequence_key = Sequence::key(
        &SequenceDerivationData {
            emitter_key: &emitter_key,
        },
        &bridge_id,
    );
    let fee_collector_key = FeeCollector::key(None, &bridge_id);

    let sender = match data.cpi_program_id {
        Some(cpi_program_id) => SenderAccount::key((), &cpi_program_id),
        None => payer,
    };

    let instruction = crate::instruction::Instruction::TransferWrappedWithPayload;

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(config_key, false),
            AccountMeta::new(from, false),
            AccountMeta::new_readonly(from_owner, true),
            AccountMeta::new(wrapped_mint_key, false),
            AccountMeta::new_readonly(wrapped_meta_key, false),
            AccountMeta::new_readonly(authority_signer, false),
            AccountMeta::new(bridge_config, false),
            AccountMeta::new(message_key, true),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            AccountMeta::new(sender, true),
            // Dependencies
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
            AccountMeta::new_readonly(spl_token::id(), false),
        ],
        data: (instruction, data).try_to_vec()?,
    })
}

pub fn attest(
    program_id: Pubkey,
    bridge_id: Pubkey,
    payer: Pubkey,
    message_key: Pubkey,
    mint: Pubkey,
    nonce: u32,
) -> solitaire::Result<Instruction> {
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &program_id);
    let emitter_key = EmitterAccount::key(None, &program_id);

    // SPL Metadata
    let spl_metadata = SplTokenMeta::key(
        &SplTokenMetaDerivationData { mint },
        &spl_token_metadata::id(),
    );

    // Mint Metadata
    let mint_meta = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &WrappedMetaDerivationData { mint_key: mint },
        &program_id,
    );

    // Bridge Keys
    let bridge_config = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &bridge_id);
    let sequence_key = Sequence::key(
        &SequenceDerivationData {
            emitter_key: &emitter_key,
        },
        &bridge_id,
    );
    let fee_collector_key = FeeCollector::key(None, &bridge_id);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(config_key, false),
            AccountMeta::new_readonly(mint, false),
            AccountMeta::new_readonly(mint_meta, false),
            AccountMeta::new_readonly(spl_metadata, false),
            // Bridge accounts
            AccountMeta::new(bridge_config, false),
            AccountMeta::new(message_key, true),
            AccountMeta::new_readonly(emitter_key, false),
            AccountMeta::new(sequence_key, false),
            AccountMeta::new(fee_collector_key, false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            // Dependencies
            AccountMeta::new(solana_program::sysvar::rent::id(), false),
            AccountMeta::new(solana_program::system_program::id(), false),
            // Program
            AccountMeta::new_readonly(bridge_id, false),
        ],
        data: (
            crate::instruction::Instruction::AttestToken,
            AttestTokenData { nonce },
        )
            .try_to_vec()?,
    })
}

pub fn upgrade_contract(
    program_id: Pubkey,
    payer: Pubkey,
    payload_message: Pubkey,
    emitter: Pubkey,
    new_contract: Pubkey,
    spill: Pubkey,
    sequence: u64,
) -> Instruction {
    let claim = Claim::<'_>::key(
        &ClaimDerivationData {
            emitter_address: emitter.to_bytes(),
            emitter_chain: CHAIN_ID_SOLANA,
            sequence,
        },
        &program_id,
    );

    let (upgrade_authority, _) = Pubkey::find_program_address(&["upgrade".as_bytes()], &program_id);

    let (program_data, _) = Pubkey::find_program_address(
        &[program_id.as_ref()],
        &solana_program::bpf_loader_upgradeable::id(),
    );

    Instruction {
        program_id,

        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new_readonly(payload_message, false),
            AccountMeta::new(claim, false),
            AccountMeta::new_readonly(upgrade_authority, false),
            AccountMeta::new(spill, false),
            AccountMeta::new(new_contract, false),
            AccountMeta::new(program_data, false),
            AccountMeta::new(program_id, false),
            AccountMeta::new_readonly(solana_program::sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::sysvar::clock::id(), false),
            AccountMeta::new_readonly(solana_program::bpf_loader_upgradeable::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],

        data: (
            crate::instruction::Instruction::UpgradeContract,
            UpgradeContractData {},
        )
            .try_to_vec()
            .unwrap(),
    }
}
