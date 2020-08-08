use std::mem::size_of;
use std::rc::Rc;
use std::str::FromStr;
use std::sync::mpsc::RecvError;
use std::thread::sleep;

use solana_sdk::program_error::ProgramError;
use solana_sdk::pubkey::Pubkey;
use spl_token::state::Account;
use tokio::stream::Stream;
use tokio::sync::mpsc;
use tokio::time::Duration;
use tonic::{transport::Server, Code, Request, Response, Status};

use service::agent_server::{Agent, AgentServer};
use service::{
    lockup_event::Event, Empty, LockupEvent, LockupEventNew, LockupEventVaaPosted,
    SubmitVaaRequest, SubmitVaaResponse, VaaPostedEvent, WatchLockupsRequest, WatchVaaRequest,
};
use spl_bridge::instruction::CHAIN_ID_SOLANA;
use spl_bridge::state::{Bridge, TransferOutProposal};

use crate::monitor::{ProgramNotificationMessage, PubsubClient};

mod monitor;

pub mod service {
    include!(concat!(env!("OUT_DIR"), concat!("/", "service", ".rs")));
}

#[derive(Default)]
pub struct AgentImpl {
    url: String,
}

#[tonic::async_trait]
impl Agent for AgentImpl {
    async fn submit_vaa(
        &self,
        request: Request<SubmitVaaRequest>,
    ) -> Result<Response<SubmitVaaResponse>, Status> {
        println!("Got a request from {:?}", request.remote_addr());

        let reply = SubmitVaaResponse {};
        Ok(Response::new(reply))
    }

    type WatchLockupsStream = mpsc::Receiver<Result<LockupEvent, Status>>;

    async fn watch_lockups(
        &self,
        _: Request<WatchLockupsRequest>,
    ) -> Result<Response<Self::WatchLockupsStream>, Status> {
        let (mut tx, rx) = mpsc::channel(1);
        let mut tx1 = tx.clone();
        let url = self.url.clone();
        // creating a new task
        tokio::spawn(async move {
            // looping and sending our response using stream
            let sub =
                PubsubClient::program_subscribe(&url, &Pubkey::from_str("").unwrap()).unwrap();
            loop {
                let item = sub.1.recv();
                match item {
                    Ok(v) => {
                        //
                        let b = match Bridge::unpack_immutable::<TransferOutProposal>(
                            v.value.account.data.as_slice(),
                        ) {
                            Ok(v) => v,
                            Err(_) => continue,
                        };

                        let mut amount_b: [u8; 32] = [0; 32];
                        b.amount.to_big_endian(&mut amount_b);

                        let event = if b.vaa_time == 0 {
                            // The Lockup was created
                            LockupEvent {
                                event: Some(Event::New(LockupEventNew {
                                    nonce: b.nonce,
                                    source_chain: CHAIN_ID_SOLANA as u32,
                                    target_chain: b.to_chain_id as u32,
                                    source_address: b.source_address.to_vec(),
                                    target_address: b.foreign_address.to_vec(),
                                    token_chain: b.asset.chain as u32,
                                    token_address: b.asset.address.to_vec(),
                                    amount: amount_b.to_vec(),
                                })),
                            }
                        } else {
                            // The VAA was submitted
                            LockupEvent {
                                event: Some(Event::VaaPosted(LockupEventVaaPosted {
                                    nonce: b.nonce,
                                    source_chain: CHAIN_ID_SOLANA as u32,
                                    target_chain: b.to_chain_id as u32,
                                    source_address: b.source_address.to_vec(),
                                    target_address: b.foreign_address.to_vec(),
                                    token_chain: b.asset.chain as u32,
                                    token_address: b.asset.address.to_vec(),
                                    amount: amount_b.to_vec(),
                                    vaa: b.vaa.to_vec(),
                                })),
                            }
                        };

                        let mut amount_b: [u8; 32] = [0; 32];
                        b.amount.to_big_endian(&mut amount_b);

                        if let Err(_) = tx.send(Ok(event)).await {
                            return;
                        };
                    }
                    Err(_) => {
                        tx.send(Err(Status::new(Code::Aborted, "watcher died")))
                            .await;
                        return;
                    }
                };
            }
        });
        tokio::spawn(async move {
            // We need to keep the channel alive https://github.com/hyperium/tonic/issues/378
            loop {
                tx1.send(Ok(LockupEvent {
                    event: Some(Event::Empty(Empty {})),
                }))
                .await;
                sleep(Duration::new(1, 0))
            }
        });
        Ok(Response::new(rx))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = "[::1]:50051".parse().unwrap();
    let agent = AgentImpl {
        url: String::from("ws://localhost:8900"),
    };

    println!("Agent listening on {}", addr);

    Server::builder()
        .add_service(AgentServer::new(agent))
        .serve(addr)
        .await?;

    Ok(())
}
