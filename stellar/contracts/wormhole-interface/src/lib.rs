#![no_std]

use soroban_sdk::contractclient;

#[contractclient(name = "WormholeClient")]
pub trait WormholeInterface {}
