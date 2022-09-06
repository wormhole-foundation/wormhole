#![feature(adt_const_params)]
#![allow(non_upper_case_globals)]
#![allow(incomplete_features)]

pub mod api;

use solitaire::*;

#[cfg(feature = "no-entrypoint")]
pub mod instructions;

pub use api::{
    post_message,
    PostMessage,
    PostMessageData,
};

solitaire! {
    PostMessage                => post_message,
}
