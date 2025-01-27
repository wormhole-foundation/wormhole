pub mod post_message;
pub mod verify_vaa;

const fn make_discriminator(input: &[u8]) -> [u8; 8] {
    let digest = sha2_const_stable::Sha256::new().update(input).finalize();
    let mut trimmed = [0; 8];
    let mut i = 0;

    loop {
        if i >= 8 {
            break;
        }
        trimmed[i] = digest[i];
        i += 1;
    }

    trimmed
}
