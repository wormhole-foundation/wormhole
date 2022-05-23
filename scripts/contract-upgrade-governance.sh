#!/bin/bash

# This tool automates the process of writing contract upgrade governance
# proposals in markdown format.
#
# There are two ways to run this script: either in "one-shot" mode, where a
# single governance VAA is generated:
#
#     ./contract-upgrade-governance.sh -m token_bridge -c solana -a Hp1YjsMbapQ75qpLaHQHuAv5Q8QwPoXs63zQrrcgg2HL > governance.md
#
# or in "multi" mode, where multiple VAAs are created in the same proposal:
#
#     ./contract-upgrade-governance.sh -m token_bridge -c solana -a Hp1YjsMbapQ75qpLaHQHuAv5Q8QwPoXs63zQrrcgg2HL -o my_proposal > governance.md
#     ./contract-upgrade-governance.sh -m token_bridge -c avalanche -a 0x45fC4b6DD26097F0E51B1C91bcc331E469Ca73c2 -o my_proposal > governance.md
#     ...                                                                                                         -o my_proposal > governance.md
#
# In multi mode, there's an additional "-o" flag, which takes a directory name,
# where intermediate progress is saved between runs. If the directory doesn't
# exist, the tool will create it.
#
# In both one-shot and multi modes, the script outputs the markdown-formatted
# proposal to STDOUT, so it's a good idea to pipe it into a file (as in the above examples).
#
# In multi-mode, it always outputs the most recent version, so it's safe to
# override the previous files.
#
# Once a multi-mode run is completed, the directory specified with the -o flag can be deleted.

set -euo pipefail

function usage() {
cat <<EOF >&2
Usage:

  $(basename "$0") [-h] [-m s] [-c s] [-a s] [-o d] -- Generate governance proposal for a given module to be upgraded to a given address

  where:
    -h  show this help text
    -m  module (bridge, token_bridge, nft_bridge)
    -c  chain name
    -a  new code address (example: 0x3f1a6729bb27350748f0a0bd85ca641a100bf0a1)
    -o  multi-mode output directory
EOF
exit 1
}


### Parse command line options
address=""
module=""
chain_name=""
multi_mode=false
out_dir=""
while getopts ':hm:c:a:o:' option; do
  case "$option" in
    h) usage
       ;;
    m) module=$OPTARG
       ;;
    c) chain_name=$OPTARG
       ;;
    a) address=$OPTARG
       ;;
    o) multi_mode=true
       out_dir=$OPTARG
       ;;
    :) printf "missing argument for -%s\n" "$OPTARG" >&2
       usage
       ;;
   \?) printf "illegal option: -%s\n" "$OPTARG" >&2
       usage
       ;;
  esac
done
shift $((OPTIND - 1))

[ -z "$address" ] && usage
[ -z "$chain_name" ] && usage
[ -z "$module" ] && usage

### The script constructs the governance proposal in two different steps. First,
### the governance prototxt (for VAA injection by the guardiand tool), then the voting/verification instructions.
gov_msg_file=""
instructions_file=""
if [ "$multi_mode" = true ]; then
  mkdir -p "$out_dir"
  gov_msg_file="$out_dir/governance.prototxt"
  instructions_file="$out_dir/instructions.md"
else
  gov_msg_file=$(mktemp)
  instructions_file=$(mktemp)
fi

explorer=""
evm=false
case "$chain_name" in
  solana)
    chain=1
    explorer="https://explorer.solana.com/address/"
    ;;
  ethereum)
    chain=2
    explorer="https://etherscan.io/address/"
    evm=true
    ;;
  terra)
    chain=3
    # This is not technically the explorer, but terra finder does not show
    # information about code ids, so this is the best we can do.
    explorer="https://lcd.terra.dev/terra/wasm/v1beta1/codes/"
    ;;
  bsc)
    chain=4
    explorer="https://bscscan.com/address/"
    evm=true
    ;;
  polygon)
    chain=5
    explorer="https://polygonscan.com/address/"
    evm=true
    ;;
  avalanche)
    chain=6
    explorer="https://snowtrace.io/address/"
    evm=true
    ;;
  oasis)
    chain=7
    explorer="https://explorer.emerald.oasis.dev/address/"
    evm=true
    ;;
  aurora)
    chain=9
    explorer="https://aurorascan.dev/address/"
    evm=true
    ;;
  fantom)
    chain=10
    explorer="https://ftmscan.com/address/"
    evm=true
    ;;
  karura)
    chain=11
    explorer="https://blockscout.karura.network/address/"
    evm=true
    ;;
  klaytn)
    chain=13
    explorer="https://scope.klaytn.com/account/"
    evm=true
    ;;
  celo)
    chain=14
    explorer="https://celoscan.xyz/address/"
    evm=true
    ;;
  *)
    echo "Unknown chain: $chain_name" >&2
    exit 1
    ;;
esac

# On terra, the contract given is a decimal code id. We convert it to a 32 byte
# hex first. The printf is escaped, which makes no difference when we actually
# evaluate the governance command later, but shows up unevaluated in the
# instructions (so it's easier to read)
terra_code_id=""
if [ "$chain_name" = "terra" ]; then
  terra_code_id="$address" # save code id for later
  address="\$(printf \"%064x\" $terra_code_id)"
fi

create_governance=""
evm_artifact=""
solana_artifact=""
terra_artifact=""
case "$module" in
  bridge)
    create_governance="\
guardiand template contract-upgrade \\
  --chain-id $chain \\
  --new-address $address"

    evm_artifact="build/contracts/Implementation.json"
    solana_artifact="artifacts-mainnet/bridge.so"
    terra_artifact="artifacts/wormhole.wasm"
    ;;
  token_bridge)
    create_governance="\
guardiand template token-bridge-upgrade-contract \\
  --chain-id $chain --module \"TokenBridge\" \\
  --new-address $address"

    evm_artifact="build/contracts/BridgeImplementation.json"
    solana_artifact="artifacts-mainnet/token_bridge.so"
    terra_artifact="artifacts/token_bridge_terra.wasm"
    ;;
  nft_bridge)
    create_governance="\
guardiand template token-bridge-upgrade-contract \\
  --chain-id $chain --module \"NFTBridge\" \\
  --new-address $address"

    evm_artifact="build/contracts/NFTBridgeImplementation.json"
    solana_artifact="artifacts-mainnet/nft_bridge.so"
    terra_artifact="artifacts/nft_bridge.wasm"
    ;;
  *) echo "unknown module $module" >&2
     usage
     ;;
esac

################################################################################
# Construct the governance proto

echo "# $module upgrade on $chain_name" >> "$gov_msg_file"
# Append the new governance message to the gov file
eval "$create_governance" >> "$gov_msg_file"

# Multiple messages will include multiple 'current_set_index' fields, but the
# proto format only takes one. This next part cleans up the file so there's only
# a single 'current_set_index' field.
# 1. we grab the first one and save it
current_set_index=$(grep "current_set_index" "$gov_msg_file" | head -n 1)
# 2. remove all 'current_set_index' fields
rest=$(grep -v "current_set_index" "$gov_msg_file")
# 3. write the set index
echo "$current_set_index" > "$gov_msg_file"
# 4. then the rest of the file
echo "$rest" >> "$gov_msg_file"

################################################################################
# Compute expected digests

# just use the 'guardiand' command, which spits out a bunch of text to
# stderr. We grab that output and pick out the VAA hashes
verify=$(guardiand admin governance-vaa-verify "$gov_msg_file" 2>&1)
digest=$(echo "$verify" | grep "VAA with digest" | cut -d' ' -f6 | sed 's/://g')

################################################################################
# Print vote command and expected digests

# This we only print to stdout, because in multi mode, it gets recomputed each
# time. The rest of the output gets printed into the instructions file
cat <<-EOD
	# Governance
	Shell command for voting:

	\`\`\`shell
	cat << EOF > governance.prototxt
	$(cat "$gov_msg_file")

	EOF

	guardiand admin governance-vaa-inject --socket /path/to/admin.sock governance.prototxt
	\`\`\`

	Expected digest(s):
	\`\`\`
	$digest
	\`\`\`
EOD

################################################################################
# Verification instructions
# The rest of the output is printed to the instructions file (which then also
# gets printed to stdout at the end)

echo "# Verification steps ($chain_name)
" >> "$instructions_file"

# Verification steps depend on the chain.

if [ "$evm" = true ]; then
  cat <<-EOF >> "$instructions_file"
	## Build
	\`\`\`shell
	wormhole/ethereum $ npm ci
	wormhole/ethereum $ npm run build
	\`\`\`

	## Verify
	Contract at [$explorer$address]($explorer$address)
	\`\`\`shell
	wormhole/ethereum $ export BYTECODE=<BYTECODE FROM EXPLORER HERE>
	wormhole/ethereum $ cat $evm_artifact | jq -r ".deployedBytecode" | sha256sum
	wormhole/ethereum $ echo \$BYTECODE | sha256sum
	\`\`\`

EOF
elif [ "$chain_name" = "solana" ]; then
  cat <<-EOF >> "$instructions_file"
	## Build
	\`\`\`shell
	wormhole/solana $ make clean
	wormhole/solana $ make NETWORK=mainnet artifacts
	\`\`\`

	This command will compile all the contracts into the \`artifacts-mainnet\` directory using Docker to ensure that the build artifacts are deterministic.

	## Verify
	Contract at [$explorer$address]($explorer$address)

	Next, use the \`verify\` script to verify that the deployed bytecodes we are upgrading to match the build artifacts:

	\`\`\`shell
	# $module
	wormhole/solana$ ./verify -n mainnet $solana_artifact $address
	\`\`\`
EOF
elif [ "$chain_name" = "terra" ]; then
  cat <<-EOF >> "$instructions_file"
	## Build
	\`\`\`shell
	wormhole/terra $ make clean
	wormhole/terra $ make artifacts
	\`\`\`

	This command will compile all the contracts into the \`artifacts\` directory using Docker to ensure that the build artifacts are deterministic.

	## Verify
	Contract at [$explorer$terra_code_id]($explorer$terra_code_id)
	Next, use the \`verify\` script to verify that the deployed bytecodes we are upgrading to match the build artifacts:

	\`\`\`shell
	# $module
	wormhole/terra$ ./verify -n mainnet $terra_artifact $terra_code_id
	\`\`\`
EOF
else
  echo "ERROR: no verification instructions for chain $chain_name" >&2
  exit 1
fi



cat <<-EOF >> "$instructions_file"
	## Create governance
	\`\`\`shell
	$create_governance
	\`\`\`

EOF

# Finally print instructions to stdout
cat "$instructions_file"
