
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

use bridge::PostVAAData;

solitaire! {
    Initialize(InitializeData)                  => initialize,
    PostMessage(PostMessageData)                => post_message,
    PostVAA(PostVAAData)                        => post_vaa,
}
