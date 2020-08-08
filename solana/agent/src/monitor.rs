use std::str::FromStr;
use std::{
    marker::PhantomData,
    sync::{
        atomic::{AtomicBool, Ordering},
        mpsc::{channel, Receiver},
        Arc, RwLock,
    },
    thread::JoinHandle,
};

use bs58;
use log::*;
use serde::{de::DeserializeOwned, de::Error, Deserialize, Deserializer, Serialize};
use serde_json::{
    json,
    value::Value::{Number, Object},
    Map, Value,
};
use solana_sdk::account::Account;
use solana_sdk::account_info::AccountInfo;
use solana_sdk::pubkey::Pubkey;
use thiserror::Error;
use tungstenite::{client::AutoStream, connect, Message, WebSocket};
use url::{ParseError, Url};

#[derive(Debug, Error)]
pub enum PubsubClientError {
    #[error("url parse error")]
    UrlParseError(#[from] ParseError),

    #[error("unable to connect to server")]
    ConnectionError(#[from] tungstenite::Error),

    #[error("json parse error")]
    JsonParseError(#[from] serde_json::error::Error),

    #[error("unexpected message format")]
    UnexpectedMessageError,
}

#[derive(Serialize, Deserialize, PartialEq, Clone, Debug)]
pub struct ProgramUpdate {
    #[serde(deserialize_with = "from_bs58")]
    pub pubkey: Pubkey,
    pub account: ProgramAccount,
}

#[derive(Serialize, Deserialize, PartialEq, Clone, Debug)]
#[serde(rename_all = "camelCase")]
pub struct ProgramAccount {
    /// lamports in the account
    pub lamports: u64,
    /// data held in this account
    #[serde(deserialize_with = "bytes_from_bs58")]
    pub data: Vec<u8>,
    /// the program that owns this account. If executable, the program that loads this account.
    #[serde(deserialize_with = "from_bs58")]
    pub owner: Pubkey,
    /// this account's data contains a loaded program (and is now read-only)
    pub executable: bool,
    /// the epoch at which this account will next owe rent
    pub rent_epoch: u64,
}

#[derive(Serialize, Deserialize, PartialEq, Clone, Debug)]
pub struct ProgramUpdateContext {
    pub slot: u64,
}

#[derive(Serialize, Deserialize, PartialEq, Clone, Debug)]
pub struct ProgramNotificationMessage {
    pub value: ProgramUpdate,
    pub context: ProgramUpdateContext,
}

pub struct PubsubClientSubscription<T>
where
    T: DeserializeOwned,
{
    message_type: PhantomData<T>,
    operation: &'static str,
    socket: Arc<RwLock<WebSocket<AutoStream>>>,
    subscription_id: u64,
    t_cleanup: Option<JoinHandle<()>>,
    exit: Arc<AtomicBool>,
}

impl<T> Drop for PubsubClientSubscription<T>
where
    T: DeserializeOwned,
{
    fn drop(&mut self) {
        self.send_unsubscribe()
            .unwrap_or_else(|_| warn!("unable to unsubscribe from websocket"));
        self.socket
            .write()
            .unwrap()
            .close(None)
            .unwrap_or_else(|_| warn!("unable to close websocket"));
    }
}

impl<T> PubsubClientSubscription<T>
where
    T: DeserializeOwned,
{
    fn send_subscribe(
        writable_socket: &Arc<RwLock<WebSocket<AutoStream>>>,
        operation: &str,
        program: &Pubkey,
    ) -> Result<u64, PubsubClientError> {
        let method = format!("{}Subscribe", operation);
        writable_socket
            .write()
            .unwrap()
            .write_message(Message::Text(
                json!({
                "jsonrpc":"2.0","id":1,"method":method,"params":[program.to_string(),{"encoding":"binary"}]
                })
                    .to_string(),
            ))?;
        let message = writable_socket.write().unwrap().read_message()?;
        Self::extract_subscription_id(message)
    }

    fn extract_subscription_id(message: Message) -> Result<u64, PubsubClientError> {
        let message_text = &message.into_text()?;
        let json_msg: Map<String, Value> = serde_json::from_str(message_text)?;

        if let Some(Number(x)) = json_msg.get("result") {
            if let Some(x) = x.as_u64() {
                return Ok(x);
            }
        }

        Err(PubsubClientError::UnexpectedMessageError)
    }

    pub fn send_unsubscribe(&self) -> Result<(), PubsubClientError> {
        let method = format!("{}Unubscribe", self.operation);
        self.socket
            .write()
            .unwrap()
            .write_message(Message::Text(
                json!({
                "jsonrpc":"2.0","id":1,"method":method,"params":[self.subscription_id]
                })
                .to_string(),
            ))
            .map_err(|err| err.into())
    }

    fn read_message(
        writable_socket: &Arc<RwLock<WebSocket<AutoStream>>>,
    ) -> Result<T, PubsubClientError> {
        let message = writable_socket.write().unwrap().read_message()?;
        let message_text = &message.into_text().unwrap();
        let json_msg: Map<String, Value> = serde_json::from_str(message_text)?;

        if let Some(Object(value_1)) = json_msg.get("params") {
            if let Some(value_2) = value_1.get("result") {
                let x: T = serde_json::from_value::<T>(value_2.clone()).unwrap();
                return Ok(x);
            }
        }

        Err(PubsubClientError::UnexpectedMessageError)
    }

    pub fn shutdown(&mut self) -> std::thread::Result<()> {
        if self.t_cleanup.is_some() {
            info!("websocket thread - shutting down");
            self.exit.store(true, Ordering::Relaxed);
            let x = self.t_cleanup.take().unwrap().join();
            info!("websocket thread - shut down.");
            x
        } else {
            warn!("websocket thread - already shut down.");
            Ok(())
        }
    }
}

const SLOT_OPERATION: &str = "program";

pub struct PubsubClient {}

impl PubsubClient {
    pub fn program_subscribe(
        url: &str,
        program: &Pubkey,
    ) -> Result<
        (
            PubsubClientSubscription<ProgramNotificationMessage>,
            Receiver<ProgramNotificationMessage>,
        ),
        PubsubClientError,
    > {
        let url = Url::parse(url)?;
        let (socket, _response) = connect(url)?;
        let (sender, receiver) = channel::<ProgramNotificationMessage>();

        let socket = Arc::new(RwLock::new(socket));
        let socket_clone = socket.clone();
        let exit = Arc::new(AtomicBool::new(false));
        let exit_clone = exit.clone();
        let subscription_id =
            PubsubClientSubscription::<ProgramNotificationMessage>::send_subscribe(
                &socket_clone,
                SLOT_OPERATION,
                program,
            )
            .unwrap();

        let t_cleanup = std::thread::spawn(move || {
            loop {
                if exit_clone.load(Ordering::Relaxed) {
                    break;
                }

                let message: Result<ProgramNotificationMessage, PubsubClientError> =
                    PubsubClientSubscription::read_message(&socket_clone);

                if let Ok(msg) = message {
                    match sender.send(msg.clone()) {
                        Ok(_) => (),
                        Err(err) => {
                            info!("receive error: {:?}", err);
                            break;
                        }
                    }
                } else {
                    info!("receive error: {:?}", message);
                    break;
                }
            }

            info!("websocket - exited receive loop");
        });

        let result: PubsubClientSubscription<ProgramNotificationMessage> =
            PubsubClientSubscription {
                message_type: PhantomData,
                operation: SLOT_OPERATION,
                socket,
                subscription_id,
                t_cleanup: Some(t_cleanup),
                exit,
            };

        Ok((result, receiver))
    }
}

fn from_bs58<'de, D>(deserializer: D) -> Result<Pubkey, D::Error>
where
    D: Deserializer<'de>,
{
    let s: String = Deserialize::deserialize(deserializer)?;
    Pubkey::from_str(s.as_str()).map_err(D::Error::custom)
}

fn bytes_from_bs58<'de, D>(deserializer: D) -> Result<Vec<u8>, D::Error>
where
    D: Deserializer<'de>,
{
    let s: String = Deserialize::deserialize(deserializer)?;
    bs58::decode(s.as_str())
        .into_vec()
        .map_err(D::Error::custom)
}
