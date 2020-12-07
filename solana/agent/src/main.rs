use std::{env, io::Write, mem::size_of, str::FromStr, fs};
use std::path::Path;
use libc;

use clap::{Arg, App, SubCommand};

use byteorder::{BigEndian, LittleEndian, ReadBytesExt, WriteBytesExt};
use futures::stream::TryStreamExt;
use solana_client::{
    client_error::ClientError, rpc_client::RpcClient, rpc_config::RpcSendTransactionConfig,
};
use solana_sdk::{
    commitment_config::{CommitmentConfig, CommitmentLevel},
    instruction::Instruction,
    pubkey::Pubkey,
    signature::{read_keypair_file, write_keypair_file, Keypair, Signature, Signer},
    transaction::Transaction,
};
use tokio::net::UnixListener;
use tokio::sync::mpsc;
use tonic::{transport::Server, Code, Request, Response, Status};

use service::{
    agent_server::{Agent, AgentServer},
    lockup_event::Event,
    Empty, LockupEvent, LockupEventNew, LockupEventVaaPosted, SubmitVaaRequest, SubmitVaaResponse,
    GetBalanceResponse, GetBalanceRequest,
    WatchLockupsRequest,
};
use spl_bridge::{
    instruction::{post_vaa, verify_signatures, VerifySigPayload, CHAIN_ID_SOLANA},
    state::{Bridge, GuardianSet, TransferOutProposal},
    vaa::VAA,
};

use crate::monitor::PubsubClient;

mod monitor;
mod socket;

pub mod service {
    include!(concat!(env!("OUT_DIR"), concat!("/", "agent.v1", ".rs")));
}

pub struct AgentImpl {
    url: String,
    bridge: Pubkey,

    rpc_url: String,
    key: Keypair,
}

pub struct SignatureItem {
    signature: [u8; 64 + 1],
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

            let mut vaa = match VAA::deserialize(&request.get_ref().vaa) {
                Ok(v) => v,
                Err(e) => {
                    return Err(Status::new(
                        Code::InvalidArgument,
                        format!("could not parse VAA: {}", e),
                    ));
                }
            };
            let verify_txs = pack_sig_verification_txs(&rpc, &bridge, &vaa, &key)?;

            // Strip signatures
            vaa.signatures = Vec::new();
            let ix = match post_vaa(&bridge, &key.pubkey(), vaa.serialize().unwrap()) {
                Ok(v) => v,
                Err(e) => {
                    return Err(Status::new(
                        Code::InvalidArgument,
                        format!("could not create post_vaa instruction: {}", e),
                    ));
                }
            };

            for mut tx in verify_txs {
                match sign_and_send(&rpc, &mut tx, vec![&key]) {
                    Ok(_) => (),
                    Err(e) => {
                        return Err(Status::new(
                            Code::Internal,
                            format!("tx sending failed: {}", e),
                        ));
                    }
                };
            }

            let mut transaction2 = Transaction::new_with_payer(&[ix], Some(&key.pubkey()));
            match sign_and_send(&rpc, &mut transaction2, vec![&key]) {
                Ok(s) => Ok(Response::new(SubmitVaaResponse {
                    signature: s.to_string(),
                })),
                Err(e) => Err(Status::new(
                    Code::Internal,
                    format!("tx sending failed: {}", e),
                )),
            }
        })
            .join()
            .unwrap()
    }

    async fn get_balance(
        &self,
        request: Request<GetBalanceRequest>,
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
                        format!("failed to fetch balance: {}", e),
                    ));
                }
            };

            Ok(Response::new(GetBalanceResponse {
                balance,
            }))
        })
            .join()
            .unwrap()
    }

    type WatchLockupsStream = mpsc::Receiver<Result<LockupEvent, Status>>;

    async fn watch_lockups(
        &self,
        _req: Request<WatchLockupsRequest>,
    ) -> Result<Response<Self::WatchLockupsStream>, Status> {
        let (mut tx, rx) = mpsc::channel(1);
        let url = self.url.clone();
        let bridge = self.bridge.clone();
        let rpc_url = self.rpc_url.clone();

        tokio::spawn(async move {
            let _rpc = RpcClient::new(rpc_url.to_string());
            let sub = PubsubClient::program_subscribe(&url, &bridge).unwrap();
            // looping and sending our response using stream
            loop {
                let item = sub.1.recv();
                match item {
                    Ok(v) => {
                        // We only want to track lockups
                        if v.value.account.data.len() != size_of::<TransferOutProposal>() {
                            continue;
                        }

                        println!("lockup changed in slot: {}", v.context.slot);

                        let b = match Bridge::unpack_immutable::<TransferOutProposal>(
                            v.value.account.data.as_slice(),
                        ) {
                            Ok(v) => v,
                            Err(e) => {
                                println!("failed to deserialize lockup: {}", e);
                                continue;
                            }
                        };

                        let mut amount_b: [u8; 32] = [0; 32];
                        b.amount.to_big_endian(&mut amount_b);

                        let event = if b.vaa_time == 0 {
                            // The Lockup was created
                            LockupEvent {
                                slot: v.context.slot,
                                lockup_address: v.value.pubkey.to_string(),
                                time: b.lockup_time as u64,
                                event: Some(Event::New(LockupEventNew {
                                    nonce: b.nonce,
                                    source_chain: CHAIN_ID_SOLANA as u32,
                                    target_chain: b.to_chain_id as u32,
                                    source_address: b.source_address.to_vec(),
                                    target_address: b.foreign_address.to_vec(),
                                    token_chain: b.asset.chain as u32,
                                    token_address: b.asset.address.to_vec(),
                                    token_decimals: b.asset.decimals as u32,
                                    amount: amount_b.to_vec(),
                                })),
                            }
                        } else {
                            // The VAA was submitted
                            LockupEvent {
                                slot: v.context.slot,
                                lockup_address: v.value.pubkey.to_string(),
                                time: b.lockup_time as u64,
                                event: Some(Event::VaaPosted(LockupEventVaaPosted {
                                    nonce: b.nonce,
                                    source_chain: CHAIN_ID_SOLANA as u32,
                                    target_chain: b.to_chain_id as u32,
                                    source_address: b.source_address.to_vec(),
                                    target_address: b.foreign_address.to_vec(),
                                    token_chain: b.asset.chain as u32,
                                    token_address: b.asset.address.to_vec(),
                                    token_decimals: b.asset.decimals as u32,
                                    amount: amount_b.to_vec(),
                                    vaa: b.vaa.to_vec(),
                                })),
                            }
                        };

                        let mut amount_b: [u8; 32] = [0; 32];
                        b.amount.to_big_endian(&mut amount_b);

                        if let Err(e) = tx.send(Ok(event)).await {
                            println!("sending event failed: {}", e);
                            return;
                        };
                        // We need to push a second message to flush the channel
                        // https://github.com/hyperium/tonic/issues/378
                        if let Err(e) = tx
                            .send(Ok(LockupEvent {
                                slot: 0,
                                time: 0,
                                lockup_address: String::from(""),
                                event: Some(Event::Empty(Empty {})),
                            }))
                            .await
                        {
                            println!("sending event failed: {}", e);
                            return;
                        };
                    }
                    Err(e) => {
                        println!("watcher died: {}", e);
                        tx.send(Err(Status::new(Code::Aborted, "watcher died")))
                            .await;
                        return;
                    }
                };
            }
        });

        Ok(Response::new(rx))
    }
}

fn pack_sig_verification_txs<'a>(
    rpc: &RpcClient,
    bridge: &Pubkey,
    vaa: &VAA,
    sender_keypair: &'a Keypair,
) -> Result<Vec<Transaction>, Status> {
    // Load guardian set
    let bridge_key = Bridge::derive_bridge_id(bridge).unwrap();
    let guardian_key =
        Bridge::derive_guardian_set_id(bridge, &bridge_key, vaa.guardian_set_index).unwrap();
    let guardian_account = rpc
        .get_account_with_commitment(
            &guardian_key,
            CommitmentConfig {
                commitment: CommitmentLevel::Single,
            },
        )
        .unwrap()
        .value
        .unwrap_or_default();
    let data = guardian_account.data;
    let guardian_set: &GuardianSet = Bridge::unpack_immutable(data.as_slice()).unwrap();

    // Map signatures to guardian set
    let mut signature_items: Vec<SignatureItem> = Vec::new();
    for s in vaa.signatures.iter() {
        let mut item = SignatureItem {
            signature: [0; 64 + 1],
            key: [0; 20],
            index: s.index,
        };

        item.signature[0..32].copy_from_slice(&s.r);
        item.signature[32..64].copy_from_slice(&s.s);
        item.signature[64] = s.v;
        item.key = guardian_set.keys[s.index as usize];

        signature_items.push(item);
    }

    let vaa_hash = match vaa.body_hash() {
        Ok(v) => v,
        Err(e) => {
            return Err(Status::new(
                Code::InvalidArgument,
                format!("could get vaa body hash: {}", e),
            ));
        }
    };
    let vaa_body = match vaa.signature_body() {
        Ok(v) => v,
        Err(e) => {
            return Err(Status::new(
                Code::InvalidArgument,
                format!("could get vaa body: {}", e),
            ));
        }
    };

    let signature_acc =
        Bridge::derive_signature_id(&bridge, &bridge_key, &vaa_hash, guardian_set.index).unwrap();

    let mut verify_txs: Vec<Transaction> = Vec::new();
    for (tx_index, chunk) in signature_items.chunks(6).enumerate() {
        let mut secp_payload = Vec::new();
        let mut signature_status = [-1i8; 20];

        let data_offset = 1 + chunk.len() * 11;
        let message_offset = data_offset + chunk.len() * 85;

        // 1 number of signatures
        secp_payload.write_u8(chunk.len() as u8);

        // Secp signature info description (11 bytes * n)
        for (i, s) in chunk.iter().enumerate() {
            secp_payload.write_u16::<LittleEndian>((data_offset + 85 * i) as u16);
            secp_payload.write_u8(0);
            secp_payload.write_u16::<LittleEndian>((data_offset + 85 * i + 65) as u16);
            secp_payload.write_u8(0);
            secp_payload.write_u16::<LittleEndian>(message_offset as u16);
            secp_payload.write_u16::<LittleEndian>(vaa_body.len() as u16);
            secp_payload.write_u8(0);
            signature_status[s.index as usize] = i as i8;
        }

        // Write signatures and addresses
        for s in chunk.iter() {
            secp_payload.write(&s.signature);
            secp_payload.write(&s.key);
        }

        // Write body
        secp_payload.write(&vaa_body);

        let secp_ix = Instruction {
            program_id: solana_sdk::secp256k1_program::id(),
            data: secp_payload,
            accounts: vec![],
        };

        let payload = VerifySigPayload {
            signers: signature_status,
            hash: vaa_hash,
            initial_creation: tx_index == 0,
        };

        let verify_ix = match verify_signatures(
            &bridge,
            &signature_acc,
            &sender_keypair.pubkey(),
            vaa.guardian_set_index,
            &payload,
        ) {
            Ok(v) => v,
            Err(e) => {
                return Err(Status::new(
                    Code::InvalidArgument,
                    format!("could not create verify instruction: {}", e),
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
) -> Result<Signature, ClientError> {
    let (recent_blockhash, _fee_calculator) = rpc.get_recent_blockhash()?;

    tx.sign(&keys, recent_blockhash);

    rpc.send_and_confirm_transaction_with_spinner_and_config(
        &tx,
        CommitmentConfig {
            commitment: CommitmentLevel::Single,
        },
        RpcSendTransactionConfig {
            skip_preflight: false,
            preflight_commitment: Some(CommitmentLevel::SingleGossip),
            encoding: None,
        },
    )
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let matches = App::new("Wormhole Solana agent")
        .arg(Arg::with_name("bridge")
            .long("bridge")
            .value_name("ADDRESS")
            .help("Bridge address")
            .required(true)
            .takes_value(true))
        .arg(Arg::with_name("ws")
            .long("ws")
            .value_name("URI")
            .help("PubSub Websocket URI (ws[s]://)")
            .required(true)
            .takes_value(true))
        .arg(Arg::with_name("rpc")
            .long("rpc")
            .value_name("URI")
            .help("RPC URI (http[s]://)")
            .required(true)
            .takes_value(true))
        .arg(Arg::with_name("socket")
            .long("socket")
            .value_name("FILE")
            .help("Path to agent socket")
            .required(true)
            .takes_value(true))
        .arg(Arg::with_name("keypair")
            .long("keypair")
            .value_name("FILE")
            .help("Fee payer account key ")
            .required(true)
            .takes_value(true))
        .get_matches();

    let bridge = matches.value_of("bridge").unwrap();
    let ws_url = matches.value_of("ws").unwrap();
    let rpc_url = matches.value_of("rpc").unwrap();
    let socket_path = matches.value_of("socket").unwrap();
    let keypair = read_keypair_file(matches.value_of("keypair").unwrap()).unwrap();

    println!("Agent using account: {}", keypair.pubkey());

    let agent = AgentImpl {
        url: ws_url.to_string(),
        rpc_url: rpc_url.to_string(),
        bridge: Pubkey::from_str(bridge).unwrap(),
        key: keypair,
    };

    // Setting a umask appears to be the only way of safely creating a UNIX socket using
    // UnixListener::bind without introducing a TOCTOU race condition.
    unsafe { libc::umask(0o0077) };

    // Delete existing socket file and recreate it with restrictive permissions.
    let mut path = Path::new(socket_path);
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
