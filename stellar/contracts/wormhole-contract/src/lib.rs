#![no_std]
use soroban_sdk::{contract, contractimpl};

use wormhole_interface::WormholeInterface;

#[contract]
pub struct Wormhole;

#[contractimpl]
impl WormholeInterface for Wormhole {}
