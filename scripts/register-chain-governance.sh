#!/bin/bash

# This tool automates the process of writing bridge registration governance
# proposals in markdown format.
#
# There are two ways to run this script: either in "one-shot" mode, where a
# single governance VAA is generated:
#
#     ./register-chain-governance.sh -m TokenBridge -c solana > governance.md
#
# or in "multi" mode, where multiple VAAs are created in the same proposal:
#
#     ./register-chain-governance.sh -m TokenBridge -c solana -o my_proposal > governance.md
#     ./register-chain-governance.sh -m TokenBridge -c avalanche -o my_proposal > governance.md
#     ...                                                        -o my_proposal > governance.md
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

  $(basename "$0") [-h] [-m s] [-c s] [-o d] [-a s] -- Generate bridge registration governance proposal for a given module

  where:
    -h  show this help text
    -m  module (TokenBridge, NFTBridge, CoreRelayer)
    -c  chain name
    -a  emitter address (optional, derived by worm CLI by default)
    -o  multi-mode output directory
EOF
exit 1
}

# Check if guardiand and worm commands exist. They needed for generating the protoxt and
# computing the digest.
if ! command -v guardiand >/dev/null 2>&1; then
  echo "ERROR: guardiand binary not found" >&2
  exit 1
fi

if ! command -v worm >/dev/null 2>&1; then
  echo "ERROR: worm binary not found" >&2
  exit 1
fi

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
    a) address=$OPTARG
       ;;
    c) chain_name=$OPTARG
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


[ -z "$chain_name" ] && usage
[ -z "$module" ] && usage

# Use the worm client to get the emitter address and wormhole chain ID.
[ -z "$address" ] && address=`worm contract --emitter mainnet $chain_name $module`
[ -z "$address" ] && usage

chain=`worm chain-id $chain_name`
[ -z "$chain" ] && usage

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

# Generate the command to create the governance prototxt
function create_governance() {
  case "$module" in
  TokenBridge)
    echo "\
guardiand template token-bridge-register-chain \\
  --chain-id $chain --module \"TokenBridge\" \\
  --new-address $address"
    ;;
  NFTBridge)
    echo "\
guardiand template token-bridge-register-chain \\
  --chain-id $chain --module \"NFTBridge\" \\
  --new-address $address"
    ;;
  CoreRelayer)
    echo "\
guardiand template token-bridge-register-chain \\
  --chain-id $chain --module \"CoreRelayer\" \\
  --new-address $address"
    ;;
  *) echo "unknown module $module" >&2
     usage
     ;;
  esac
}

################################################################################
# Construct the governance proto

echo "# Registration $chain_name $module" >> "$gov_msg_file"
# Append the new governance message to the gov file
eval "$(create_governance)" >> "$gov_msg_file"

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

# massage the digest into the same format that the inject command prints it
digest=$(echo "$digest" | awk '{print toupper($0)}' | sed 's/^0X//')
# we use the first 7 characters of the digest as an identifier for the prototxt file
gov_id=$(echo "$digest" | cut -c1-7)

################################################################################
# Print vote command and expected digests

# This we only print to stdout, because in multi mode, it gets recomputed each
# time. The rest of the output gets printed into the instructions file
cat <<-EOD
	# Governance
	Shell command for voting:

	\`\`\`shell
	cat << EOF > governance-$gov_id.prototxt
	$(cat "$gov_msg_file")

	EOF

	guardiand admin governance-vaa-inject --socket /path/to/admin.sock governance-$gov_id.prototxt
	\`\`\`

	Expected digest(s):
	\`\`\`
	$digest
	\`\`\`
EOD
