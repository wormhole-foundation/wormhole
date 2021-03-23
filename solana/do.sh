#!/usr/bin/env bash
# This script is based on Solana's upstream do.sh. If our usage of
# bpf-sdk breaks, it is best to inspect its context and check with
# Solana's latest program build workflow.

cd "$(dirname "$0")"

usage() {
    cat <<EOF
Usage: do.sh <action> <project> <action specific arguments>
Supported actions:
    build
    build-lib
    clean
    clippy
    doc
    dump
    fmt
    test
    update
Supported projects:
    all
    any directory containing a Cargo.toml file
EOF
}

sdkParentDir=bin
sdkDir="$sdkParentDir"/bpf-sdk
profile=bpfel-unknown-unknown/release

perform_action() {
    set -e
    projectDir="$PWD"/$2
    targetDir=target
    case "$1" in
    build)
        if [[ -f "$projectDir"/Xargo.toml ]]; then
          "$sdkDir"/rust/build.sh "$projectDir"

          so_path="$targetDir/$profile"
	  files=`find $so_path -maxdepth 1 -type f \! -name "*_debug.so" -name  "*.so"`
	  for file in $files
	  do
	    cp $file ${file/.so/_debug.so} # Copy with rename
	    $sdkDir/scripts/strip.sh $file $file
	    # "$sdkDir"/dependencies/llvm-native/bin/llvm-objcopy --strip-all "$file"
	  done
        else
            echo "$projectDir does not contain a program, skipping"
        fi
        ;;
    build-lib)
        (
            cd "$projectDir"
            echo "build $projectDir"
            export RUSTFLAGS="${@:3}"
            cargo build
        )
        ;;
    clean)
        "$sdkDir"/rust/clean.sh "$projectDir"
        ;;
    clippy)
        (
            cd "$projectDir"
            echo "clippy $projectDir"
            cargo +nightly clippy  --features=program ${@:3}
        )
        ;;
    doc)
        (
            cd "$projectDir"
            echo "generating docs $projectDir"
            cargo doc ${@:3}
        )
        ;;
    dump)
        # Dump depends on tools that are not installed by default and must be installed manually
        # - greadelf
        # - rustfilt
        (
            pwd
            "$0" build "$2"

	    so_path="$targetDir/$profile"
	    files=`find $so_path -maxdepth 1 -type f \! -name "*_debug.so" -name  "*.so"`
	    for file in $files
	    do
		dump_filename="${file}_dump"
		echo $file
		echo $dump_filename

		if [ -f "$file" ]; then
		    ls \
			-la \
			"$file" \
			>"${dump_filename}_mangled.txt"
		    greadelf \
			-aW \
			"$file" \
			>>"${dump_filename}_mangled.txt"
		    "$sdkDir/dependencies/llvm-native/bin/llvm-objdump" \
			-print-imm-hex \
			--source \
			--disassemble \
			"$file" \
			>>"${dump_filename}_mangled.txt"
		    sed e
			s/://g \
			<"${dump_filename}_mangled.txt" |
			rustfilt \
			    >"${dump_filename}.txt"
		else
		    echo "Warning: No dump created, cannot find: $file"
		fi
	    done
        )
        ;;
    fmt)
        (
            cd "$projectDir"
            echo "formatting $projectDir"
            cargo fmt ${@:3}
        )
        ;;
    help)
        usage
        exit
        ;;
    test)
        (
            cd "$projectDir"
            echo "test $projectDir"
            cargo test --features=program ${@:3}
        )
        ;;
    update)
        mkdir -p $sdkParentDir
        ./bpf-sdk-install.sh $sdkParentDir
        ;;
    *)
        echo "Error: Unknown command"
        usage
        exit
        ;;
    esac
}

set -e
if [[ $1 == "update" ]]; then
    perform_action "$1"
    exit
else
    if [[ "$#" -lt 2 ]]; then
        usage
        exit
    fi
    if [[ ! -d "$sdkDir" ]]; then
        ./do.sh update
    fi
fi

if [[ $2 == "all" ]]; then
    # Perform operation on all projects
    for project in */; do
        if [[ -f "$project"Cargo.toml ]]; then
            perform_action "$1" "${project%/}" ${@:3}
        else
            continue
        fi
    done
else
    # Perform operation on requested project
    perform_action "$1" "$2" "${@:3}"
fi
