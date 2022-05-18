#!/usr/bin/env bash
set -ex

main() {
    local out="$1"

    # Copy rust build products
    mkdir -p "${out}"/rust
    cp -R "build/aarch64-unknown-linux-gnu/stage1/bin" "${out}"/rust/
    mkdir -p "${out}"/rust/lib/rustlib/
    cp -R "build/aarch64-unknown-linux-gnu/stage1/lib/rustlib/aarch64-unknown-linux-gnu" "${out}"/rust/lib/rustlib/
    cp -R "build/aarch64-unknown-linux-gnu/stage1/lib/rustlib/bpfel-unknown-unknown" "${out}"/rust/lib/rustlib/
    find . -maxdepth 6 -type f -path "./build/aarch64-unknown-linux-gnu/stage1/lib/*" -exec cp {} "${out}"/rust/lib \;

    "${out}/rust/bin/rustc" --version
    "${out}/rust/bin/rustdoc" --version

    # Copy llvm build products
    mkdir -p "${out}"/llvm/{bin,lib}
    local binaries=(
        clang
        clang++
        clang-cl
        clang-cpp
        clang-13
        ld.lld
        ld64.lld
        llc
        lld
        lld-link
        llvm-ar
        llvm-objcopy
        llvm-objdump
        llvm-readelf
        llvm-readobj
    )
    local bin
    for bin in "${binaries[@]}"; do
        local bin_file="build/aarch64-unknown-linux-gnu/llvm/build/bin/${bin}"
        cp "${bin_file}" "${out}/llvm/bin"
    done

    cp -R "build/aarch64-unknown-linux-gnu/llvm/build/lib/clang" "${out}"/llvm/lib/

    binaries=(
        clang
        clang++
        clang-cl
        clang-cpp
        ld.lld
        llc
        lld-link
        llvm-ar
        llvm-objcopy
        llvm-objdump
        llvm-readelf
        llvm-readobj
    )
    for bin in "${binaries[@]}"; do
        "${out}/llvm/bin/${bin}" --version
    done
}

main "$@"
