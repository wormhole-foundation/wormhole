use wasm_bindgen::prelude::*;

/// sanity check for wasm compilation, TODO(sdrozd): remove after
/// meaningful endpoints are added
#[wasm_bindgen]
pub fn hello_p2w() -> String {
    "Ciao mondo!".to_owned()
}
