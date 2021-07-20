use libc;
use std::{
    fs,
    io::Write,
    path::Path,
    str::FromStr,
};

use clap::{
    App,
    Arg,
};

use byteorder::{
    LittleEndian,
    WriteBytesExt,
};
use futures::stream::TryStreamExt;
use solana_client::{
    client_error::ClientError,
    rpc_client::RpcClient,
    rpc_config::RpcSendTransactionConfig,
};
use solana_sdk::{
    commitment_config::{
        CommitmentConfig,
        CommitmentLevel,
    },
    instruction::Instruction,
    pubkey::Pubkey,
    signature::{
        read_keypair_file,
        Keypair,
        Signature,
        Signer,
    },
    transaction::Transaction,
};
use tokio::net::UnixListener;

use tonic::{
    transport::Server,
    Code,
    Request,
    Response,
    Status,
};

use borsh::BorshDeserialize;
use bridge::{
    accounts::{
        GuardianSet,
        GuardianSetDerivationData,
    },
    instructions::{
        hash_vaa,
        post_vaa,
        serialize_vaa,
        verify_signatures,
    },
    types::GuardianSetData,
    PostVAAData,
    VerifySignaturesData,
};
use service::{
    agent_server::{
        Agent,
        AgentServer,
    },
    GetBalanceRequest,
    GetBalanceResponse,
    SubmitVaaRequest,
    SubmitVaaResponse,
};
use sha3::Digest;
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};

mod socket;

pub mod service {
    include!(concat!(env!("OUT_DIR"), concat!("/", "agent.v1", ".rs")));
}

pub struct AgentImpl {
    bridge: Pubkey,

    rpc_url: String,
    key: Keypair,
}

pub struct SignatureItem {
    signature: Vec<u8>,
    key: [u8; 20],
    index: u8,
}

#[tonic::async_trait]
impl Agent for AgentImpl {
    async fn submit_vaa(
        &self,
        request: Request<SubmitVaaRequest>,
    ) -> Result<Response<SubmitVaaResponse>, Status> {
        // Hack to clone keypair
        let b = self.key.to_bytes();
        let key = Keypair::from_bytes(&b).unwrap();
        let bridge = self.bridge.clone();

        let rpc_url = self.rpc_url.clone();

        // we need to spawn an extra thread because tokio does not allow nested runtimes
        std::thread::spawn(move || {
            let rpc = RpcClient::new(rpc_url);

            let vaa = &request.get_ref().vaa.as_ref().unwrap();

            let mut emitter_address = [0u8; 32];
            emitter_address.copy_from_slice(vaa.emitter_address.as_slice());
            let post_data = PostVAAData {
                version: vaa.version as u8,
                guardian_set_index: vaa.guardian_set_index,
                timestamp: vaa.timestamp.as_ref().unwrap().seconds as u32,
                nonce: vaa.nonce,
                emitter_chain: vaa.emitter_chain as u16,
                emitter_address: emitter_address,
                sequence: vaa.sequence,
                consistency_level: vaa.consistency_level as u8,
                payload: vaa.payload.clone(),
            };

            let verify_txs =
                pack_sig_verification_txs(&rpc, &bridge, &post_data, &vaa.signatures, &key)?;

            // Strip signatures
            let ix = post_vaa(bridge, key.pubkey(), post_data);

            for mut tx in verify_txs {
                match sign_and_send(&rpc, &mut tx, vec![&key], request.get_ref().skip_preflight) {
                    Ok(_) => (),
                    Err(e) => {
                        return Err(Status::new(
                            Code::Internal,
                            format!("tx sending failed: {:?}", e),
                        ));
                    }
                };
            }

            let mut transaction2 = Transaction::new_with_payer(&[ix], Some(&key.pubkey()));
            match sign_and_send(
                &rpc,
                &mut transaction2,
                vec![&key],
                request.into_inner().skip_preflight,
            ) {
                Ok(s) => Ok(Response::new(SubmitVaaResponse {
                    signature: s.to_string(),
                })),
                Err(e) => Err(Status::new(
                    Code::Internal,
                    format!("tx sending failed: {:?}", e),
                )),
            }
        })
        .join()
        .unwrap()
    }

    async fn get_balance(
        &self,
        _request: Request<GetBalanceRequest>,
    ) -> Result<Response<GetBalanceResponse>, Status> {
        // Hack to clone keypair
        let b = self.key.pubkey();

        let rpc_url = self.rpc_url.clone();

        // we need to spawn an extra thread because tokio does not allow nested runtimes
        std::thread::spawn(move || {
            let rpc = RpcClient::new(rpc_url);

            let balance = match rpc.get_balance(&b) {
                Ok(v) => v,
                Err(e) => {
                    return Err(Status::new(
                        Code::Internal,
                        format!("failed to fetch balance: {:?}", e),
                    ));
                }
            };

            Ok(Response::new(GetBalanceResponse { balance }))
        })
        .join()
        .unwrap()
    }
}

fn pack_sig_verification_txs<'a>(
    rpc: &RpcClient,
    bridge: &Pubkey,
    vaa: &PostVAAData,
    signatures: &Vec<service::Signature>,
    sender_keypair: &'a Keypair,
) -> Result<Vec<Transaction>, Status> {
    // Load guardian set
    let guardian_key = GuardianSet::<'_, { AccountState::Initialized }>::key(
        &GuardianSetDerivationData {
            index: vaa.guardian_set_index,
        },
        bridge,
    );
    let guardian_account = rpc
        .get_account_with_commitment(
            &guardian_key,
            CommitmentConfig {
                commitment: CommitmentLevel::Processed,
            },
        )
        .unwrap()
        .value
        .unwrap_or_default();
    let data = guardian_account.data;
    let guardian_set: GuardianSetData = GuardianSetData::try_from_slice(data.as_slice()).unwrap();

    // Map signatures to guardian set
    let mut signature_items: Vec<SignatureItem> = Vec::new();
    for s in signatures.iter() {
        let mut item = SignatureItem {
            signature: s.signature.clone(),
            key: [0; 20],
            index: s.guardian_index as u8,
        };
        item.key = guardian_set.keys[s.guardian_index as usize];

        signature_items.push(item);
    }

    let vaa_body = serialize_vaa(vaa);
    let body_hash: [u8; 32] = {
        let mut h = sha3::Keccak256::default();
        h.write(vaa_body.as_slice())?;
        h.finalize().into()
    };

    let mut verify_txs: Vec<Transaction> = Vec::new();
    for (_tx_index, chunk) in signature_items.chunks(6).enumerate() {
        let mut secp_payload = Vec::new();
        let mut signature_status = [-1i8; 19];

        let data_offset = 1 + chunk.len() * 11;
        let message_offset = data_offset + chunk.len() * 85;

        // 1 number of signatures
        secp_payload.write_u8(chunk.len() as u8)?;

        // Secp signature info description (11 bytes * n)
        for (i, s) in chunk.iter().enumerate() {
            secp_payload.write_u16::<LittleEndian>((data_offset + 85 * i) as u16)?;
            secp_payload.write_u8(0)?;
            secp_payload.write_u16::<LittleEndian>((data_offset + 85 * i + 65) as u16)?;
            secp_payload.write_u8(0)?;
            secp_payload.write_u16::<LittleEndian>(message_offset as u16)?;
            secp_payload.write_u16::<LittleEndian>(body_hash.len() as u16)?;
            secp_payload.write_u8(0)?;
            signature_status[s.index as usize] = i as i8;
        }

        // Write signatures and addresses
        for s in chunk.iter() {
            secp_payload.write(&s.signature)?;
            secp_payload.write(&s.key)?;
        }

        // Write body
        secp_payload.write(&body_hash)?;

        let secp_ix = Instruction {
            program_id: solana_sdk::secp256k1_program::id(),
            data: secp_payload,
            accounts: vec![],
        };

        let body_hash: [u8; 32] = hash_vaa(vaa);

        let payload = VerifySignaturesData {
            signers: signature_status,
            initial_creation: false,
        };

        let verify_ix = match verify_signatures(
            *bridge,
            sender_keypair.pubkey(),
            vaa.guardian_set_index,
            body_hash,
            payload,
        ) {
            Ok(v) => v,
            Err(e) => {
                return Err(Status::new(
                    Code::InvalidArgument,
                    format!("could not create verify instruction: {:?}", e),
                ));
            }
        };

        verify_txs.push(Transaction::new_with_payer(
            &[secp_ix, verify_ix],
            Some(&sender_keypair.pubkey()),
        ))
    }

    Ok(verify_txs)
}

fn sign_and_send(
    rpc: &RpcClient,
    tx: &mut Transaction,
    keys: Vec<&Keypair>,
    skip_preflight: bool,
) -> Result<Signature, ClientError> {
    let (recent_blockhash, _fee_calculator) = rpc.get_recent_blockhash()?;

    tx.sign(&keys, recent_blockhash);

    rpc.send_and_confirm_transaction_with_spinner_and_config(
        &tx,
        CommitmentConfig {
            commitment: CommitmentLevel::Processed,
        },
        RpcSendTransactionConfig {
            skip_preflight,
            preflight_commitment: Some(CommitmentLevel::Processed),
            encoding: None,
        },
    )
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let matches = App::new("Wormhole Solana agent")
        .arg(
            Arg::with_name("bridge")
                .long("bridge")
                .value_name("ADDRESS")
                .help("Bridge address")
                .required(true)
                .takes_value(true),
        )
        .arg(
            Arg::with_name("ws")
                .long("ws")
                .value_name("URI")
                .help("PubSub Websocket URI (ws[s]://)")
                .required(true)
                .takes_value(true),
        )
        .arg(
            Arg::with_name("rpc")
                .long("rpc")
                .value_name("URI")
                .help("RPC URI (http[s]://)")
                .required(true)
                .takes_value(true),
        )
        .arg(
            Arg::with_name("socket")
                .long("socket")
                .value_name("FILE")
                .help("Path to agent socket")
                .required(true)
                .takes_value(true),
        )
        .arg(
            Arg::with_name("keypair")
                .long("keypair")
                .value_name("FILE")
                .help("Fee payer account key ")
                .required(true)
                .takes_value(true),
        )
        .get_matches();

    let bridge = matches.value_of("bridge").unwrap();
    let rpc_url = matches.value_of("rpc").unwrap();
    let socket_path = matches.value_of("socket").unwrap();
    let keypair = read_keypair_file(matches.value_of("keypair").unwrap()).unwrap();

    println!("Agent using account: {}", keypair.pubkey());

    let agent = AgentImpl {
        rpc_url: rpc_url.to_string(),
        bridge: Pubkey::from_str(bridge).unwrap(),
        key: keypair,
    };

    // Setting a umask appears to be the only way of safely creating a UNIX socket using
    // UnixListener::bind without introducing a TOCTOU race condition.
    unsafe { libc::umask(0o0077) };

    // Delete existing socket file and recreate it with restrictive permissions.
    let path = Path::new(socket_path);
    if path.exists() {
        fs::remove_file(path)?;
    }

    let mut listener = UnixListener::bind(socket_path)?;
    println!("Agent listening on {}", socket_path);

    Server::builder()
        .add_service(AgentServer::new(agent))
        .serve_with_incoming(listener.incoming().map_ok(socket::UnixStream))
        .await?;

    Ok(())
}
