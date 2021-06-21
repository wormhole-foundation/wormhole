fn main() {
    tonic_build::compile_protos("../../proto/agent/v1/service.proto").unwrap();
}
