#![feature(adt_const_params)]
#![allow(non_upper_case_globals)]
#![allow(incomplete_features)]

pub mod api;

use solitaire::*;

pub use api::{
    initialize,
    post_message,
    post_vaa,
    Initialize,
    InitializeData,
    PostMessage,
    PostMessageData,
    PostVAA,
    Signature,
    UninitializedMessage,
};

solitaire! {
    Initialize  => initialize,
    PostMessage => post_message,
    PostVAA     => post_vaa,
}
