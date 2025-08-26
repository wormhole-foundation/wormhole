#!/bin/bash

# Aztec Deployment Wizard Script
# Interactive deployment script for Token and Wormhole contracts on Aztec testnet

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Configuration variables with defaults
DEFAULT_NODE_URL="https://aztec-alpha-testnet-fullnode.zkv.xyz"
DEFAULT_SPONSORED_FPC_ADDRESS="0x19b5539ca1b104d4c3705de94e4555c9630def411f025e023a13189d0c56f8f22"
DEFAULT_OWNER_SK="0x0ff5c4c050588f4614255a5a4f800215b473e442ae9984347b3a727c3bb7ca55"

# Actual configuration (will be set by wizard)
NODE_URL=""
SPONSORED_FPC_ADDRESS=""
OWNER_SK=""

# Contract addresses (captured during deployment)
OWNER_ADDRESS=""
RECEIVER_ADDRESS=""
TOKEN_CONTRACT_ADDRESS=""
WORMHOLE_CONTRACT_ADDRESS=""

# Contract file paths
WORMHOLE_CONTRACT_SRC="src/main.nr"
WORMHOLE_CONTRACT_BACKUP="src/main.nr.backup"

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

wizard_header() {
    echo -e "${MAGENTA}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║                  AZTEC DEPLOYMENT WIZARD                ║"
    echo "║              Token & Wormhole Contract Deployer         ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Wizard configuration function
setup_wizard() {
    wizard_header
    
    echo -e "\n${CYAN}Welcome to the Aztec Deployment Wizard!${NC}\n"
    
    echo "This wizard will help you deploy Token and Wormhole contracts on Aztec testnet."
    echo "You can use default values or customize the configuration."
    echo ""
    
    read -p "Do you want to use default configuration values? (y/n): " use_defaults
    
    if [[ $use_defaults =~ ^[Yy]$ ]]; then
        NODE_URL="$DEFAULT_NODE_URL"
        SPONSORED_FPC_ADDRESS="$DEFAULT_SPONSORED_FPC_ADDRESS"
        OWNER_SK="$DEFAULT_OWNER_SK"
        
        success "Using default configuration"
    else
        echo -e "\n${CYAN}Let's configure your deployment settings:${NC}\n"
        
        # Node URL configuration
        echo -e "${YELLOW}Node URL Configuration:${NC}"
        echo "Default: $DEFAULT_NODE_URL"
        read -p "Enter Node URL (press Enter for default): " input_node_url
        NODE_URL="${input_node_url:-$DEFAULT_NODE_URL}"
        
        # Sponsored FPC Address configuration
        echo -e "\n${YELLOW}Sponsored FPC Address Configuration:${NC}"
        echo "Default: $DEFAULT_SPONSORED_FPC_ADDRESS"
        read -p "Enter Sponsored FPC Address (press Enter for default): " input_fpc_address
        SPONSORED_FPC_ADDRESS="${input_fpc_address:-$DEFAULT_SPONSORED_FPC_ADDRESS}"
        
        # Owner Private Key configuration
        echo -e "\n${YELLOW}Owner Private Key Configuration:${NC}"
        echo "Default: $DEFAULT_OWNER_SK"
        read -p "Enter Owner Private Key (press Enter for default): " input_owner_sk
        OWNER_SK="${input_owner_sk:-$DEFAULT_OWNER_SK}"
        
        success "Custom configuration set"
    fi
    
    echo -e "\n${CYAN}Configuration Summary:${NC}"
    echo "Node URL: $NODE_URL"
    echo "FPC Address: $SPONSORED_FPC_ADDRESS"
    echo "Owner SK: ${OWNER_SK:0:10}..."
    echo ""
    
    warning "Important: Default Private Key Usage"
    if [[ "$OWNER_SK" == "$DEFAULT_OWNER_SK" ]]; then
        echo -e "${YELLOW}You are using the default owner private key.${NC}"
        echo ""
        echo -e "${CYAN}What this means:${NC}"
        echo "• This private key may have been used before on this network"
        echo "• If the account is already deployed, you'll see 'Existing nullifier' error"
        echo "• This is NORMAL and means your account is already ready to use"
        echo "• The script will detect this and continue successfully"
        echo ""
        echo -e "${CYAN}Why this happens:${NC}"
        echo "• Aztec uses nullifiers to prevent double-spending"
        echo "• Each account deployment creates a unique nullifier"
        echo "• Trying to deploy the same account twice triggers this protection"
        echo "• The error actually confirms your account exists and is secure"
        echo ""
    else
        echo -e "${GREEN}You are using a custom private key.${NC}"
        echo "• If this is a new key, the account will be deployed fresh"
        echo "• If you've used this key before, you may see 'Existing nullifier' error"
        echo "• Either way, the script will handle it correctly"
        echo ""
    fi
    
    read -p "Press Enter to continue with deployment..."
}

# Check dependencies
check_dependencies() {
    log "Checking dependencies..."
    
    local missing_deps=()
    
    if ! command -v aztec-wallet &> /dev/null; then
        missing_deps+=("aztec-wallet")
    fi
    
    if ! command -v aztec-nargo &> /dev/null; then
        missing_deps+=("aztec-nargo")
    fi
    
    if ! command -v aztec &> /dev/null; then
        missing_deps+=("aztec")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        error "Missing dependencies: ${missing_deps[*]}"
        error "Please install Aztec CLI tools before continuing."
        info "Installation instructions:"
        info "1. Install Aztec CLI: https://docs.aztec.network/getting_started"
        info "2. Install Aztec Nargo: https://docs.aztec.network/getting_started"
        exit 1
    fi
    
    success "All dependencies are installed"
    
    # Display versions for debugging
    info "Dependency versions:"
    info "- aztec-wallet: $(aztec-wallet --version 2>/dev/null || echo 'version unknown')"
    info "- aztec-nargo: $(aztec-nargo --version 2>/dev/null || echo 'version unknown')"
    info "- aztec: $(aztec --version 2>/dev/null || echo 'version unknown')"
}

# Extract transaction ID from output
extract_transaction_id() {
    local output="$1"
    echo "$output" | grep "Transaction hash:" | head -1 | sed 's/Transaction hash: //' | tr -d ' '
}

# Extract address using a flexible pattern
extract_address() {
    local output="$1"
    local pattern="${2:-0x[a-fA-F0-9]{64}}"
    echo "$output" | grep -o "$pattern" | head -1
}

# Check and handle stale transactions
check_and_handle_stale_transaction() {
    local tx_id="$1"
    local description="$2"
    
    warning "Transaction may be stale: $tx_id"
    info "You can check transaction status at: http://aztecscan.xyz/tx/$tx_id"
    
    echo ""
    echo "What would you like to do?"
    echo "1. Retry deployment (recommended)"
    echo "2. Wait and continue (if you think transaction will complete)"
    echo "3. Skip this step (not recommended)"
    
    local choice
    read -p "Choose option (1/2/3) [default: 1]: " choice
    choice=${choice:-1}
    
    case $choice in
        1)
            info "Will retry deployment"
            return 0  # Retry
            ;;
        2)
            info "Continuing with potentially stale transaction"
            return 1  # Continue
            ;;
        3)
            warning "Skipping deployment step"
            return 2  # Skip
            ;;
        *)
            info "Invalid choice, defaulting to retry"
            return 0  # Retry
            ;;
    esac
}

# Wait for transaction to be mined with retry logic
wait_for_transaction() {
    local description="$1"
    
    info "Transaction submitted for $description"
    info "Aztec transactions are processed automatically - continuing with deployment"
    info "You can monitor all transactions at: http://aztecscan.xyz/"
    
    # Short pause to allow transaction to propagate
    sleep 5
}

# Execute command with retry logic and specific error handling
execute_with_retry() {
    local description="$1"
    shift
    local max_retries=3
    local retry=0
    
    while [ $retry -lt $max_retries ]; do
        if [ $retry -gt 0 ]; then
            warning "Retrying $description (attempt $((retry + 1))/$max_retries)..."
            wait_for_transaction "$description"
        fi
        
        log "Executing: $description"
        
        # Capture both stdout and stderr
        local output
        local exit_code=0
        output=$("$@" 2>&1) || exit_code=$?
        
        if [ $exit_code -eq 0 ]; then
            # Check if output indicates we need to wait for mining
            if echo "$output" | grep -q "Waiting for account contract deployment"; then
                success "$description submitted successfully"
                info "Transaction is being mined - this may take several minutes"
                wait_for_transaction "$description"
            else
                success "$description completed successfully"
            fi
            echo "$output"
            return 0
        else
            # Check for specific error patterns
            if echo "$output" | grep -q "Existing nullifier"; then
                warning "Account already deployed (existing nullifier error)"
                success "$description completed (account already exists)"
                echo "$output"
                return 0
                
            elif echo "$output" | grep -q "Timeout awaiting isMined"; then
                local tx_id
                tx_id=$(extract_transaction_id "$output")
                
                warning "Transaction timed out waiting for mining"
                
                local retry_decision
                retry_decision=$(check_and_handle_stale_transaction "$tx_id" "$description")
                
                case $? in
                    0)  # Retry
                        warning "Retrying deployment..."
                        retry=$((retry + 1))
                        continue
                        ;;
                    1)  # Wait and continue
                        info "Continuing with timed-out transaction"
                        echo "$output"
                        return 0
                        ;;
                    2)  # Skip step
                        warning "Skipping $description"
                        return 0
                        ;;
                esac
            fi
            
            error "$description failed"
            retry=$((retry + 1))
            
            if [ $retry -lt $max_retries ]; then
                warning "Command failed, waiting before retry..."
                sleep 30
            else
                error "Output from failed command:"
                echo "$output"
            fi
        fi
    done
    
    error "$description failed after $max_retries attempts"
    return 1
}

# Extract address from command output
extract_address_from_output() {
    local output="$1"
    # Extract the address from "Address: 0x..." line
    echo "$output" | grep "^Address:" | sed 's/Address:[[:space:]]*//' | tr -d ' '
}

# Execute command that depends on previous deployments being mined
execute_with_dependency_retry() {
    local description="$1"
    shift
    local max_retries=5  # More retries for dependency-related commands
    local retry=0
    
    while [ $retry -lt $max_retries ]; do
        if [ $retry -gt 0 ]; then
            warning "Retrying $description - waiting for dependencies (attempt $((retry + 1))/$max_retries)..."
            info "Previous deployments may still be mining or transactions may be stale"
            
            # Longer wait for dependency issues, increasing with each retry
            local wait_time=$((60 + (retry * 30)))
            info "Waiting $wait_time seconds for blockchain state to sync..."
            sleep $wait_time
        fi
        
        log "Executing: $description"
        
        # Capture both stdout and stderr
        local output
        local exit_code=0
        output=$("$@" 2>&1) || exit_code=$?
        
        if [ $exit_code -eq 0 ]; then
            # Extract contract address if this is a deployment
            if [[ "$description" == *"deployment"* ]]; then
                local contract_address
                contract_address=$(echo "$output" | grep "Contract deployed at" | sed 's/Contract deployed at //' | tr -d ' ')
                
                if [ -n "$contract_address" ]; then
                    if [[ "$description" == *"Token"* ]]; then
                        TOKEN_CONTRACT_ADDRESS="$contract_address"
                        success "Token contract deployed at: $TOKEN_CONTRACT_ADDRESS"
                    elif [[ "$description" == *"Wormhole"* ]]; then
                        WORMHOLE_CONTRACT_ADDRESS="$contract_address"
                        success "Wormhole contract deployed at: $WORMHOLE_CONTRACT_ADDRESS"
                    fi
                fi
            fi
            
            # Check if we got a transaction hash and need to wait for mining
            local tx_hash
            tx_hash=$(echo "$output" | grep "Deploy tx hash:" | head -1 | sed 's/Deploy tx hash:[[:space:]]*//' | tr -d ' ')
            
            if [ -n "$tx_hash" ]; then
                info "Transaction submitted: $tx_hash"
                info "Check status at: http://aztecscan.xyz/tx/$tx_hash"
                
                # Check if transaction was already mined in the output
                if echo "$output" | grep -q "Transaction has been mined"; then
                    if echo "$output" | grep -q "Status: success"; then
                        success "$description completed successfully"
                    else
                        warning "$description transaction mined but check status"
                    fi
                else
                    info "Transaction deployment initiated - continuing with next step"
                fi
            else
                success "$description completed successfully"
            fi
            
            echo "$output"
            return 0
        else
            # Check for errors that indicate we need to wait for previous deployments
            if echo "$output" | grep -qi "contract.*not.*found\|contract.*not.*deployed\|account.*not.*found"; then
                warning "Dependency not ready - previous deployment may still be mining"
                retry=$((retry + 1))
                continue
            elif echo "$output" | grep -qi "Cannot find the leaf for nullifier\|nullifier.*not.*found"; then
                warning "Nullifier/state synchronization issue - blockchain state may not be ready"
                info "This often happens when previous transactions are stale or still processing"
                
                if [ $retry -ge 2 ]; then
                    warning "Multiple failures suggest previous transactions may be stale"
                    info "Consider checking transaction status at http://aztecscan.xyz/"
                    echo ""
                    echo "Options:"
                    echo "1. Continue retrying (may work if transactions eventually mine)"
                    echo "2. Exit and manually check/retry stale transactions"
                    echo "3. Skip this step (not recommended)"
                    
                    local choice
                    read -p "Choose option (1/2/3) [default: 1]: " choice
                    choice=${choice:-1}
                    
                    case $choice in
                        2)
                            info "Exiting for manual intervention"
                            exit 1
                            ;;
                        3)
                            warning "Skipping $description"
                            return 0
                            ;;
                    esac
                fi
                
                retry=$((retry + 1))
                continue
            elif echo "$output" | grep -qi "simulation.*failed\|transaction.*simulation.*error"; then
                warning "Transaction simulation failed - dependencies may not be ready"
                retry=$((retry + 1))
                continue
            fi
            
            error "$description failed"
            echo "$output"
            retry=$((retry + 1))
            
            if [ $retry -lt $max_retries ]; then
                warning "Command failed, waiting before retry..."
                sleep 30
            fi
        fi
    done
    
    error "$description failed after $max_retries attempts"
    error "This may indicate that previous deployments are stale or not properly synced"
    info "Check http://aztecscan.xyz/ for deployment status"
    info "You may need to restart deployment with fresh transactions"
    return 1
}

# Set environment variables
setup_environment() {
    log "Setting up environment variables..."
    export NODE_URL="$NODE_URL"
    export SPONSORED_FPC_ADDRESS="$SPONSORED_FPC_ADDRESS"
    export OWNER_SK="$OWNER_SK"
    success "Environment variables set"
}

# Step 2: Create wallets
create_wallets() {
    log "Creating wallets..."
    
    log "Creating owner wallet..."
    local owner_output
    owner_output=$(aztec-wallet create-account \
        -sk "$OWNER_SK" \
        --register-only \
        --node-url "$NODE_URL" \
        --alias owner-wallet 2>&1)
    
    # Extract owner address from output
    OWNER_ADDRESS=$(extract_address_from_output "$owner_output")
    
    if [ -n "$OWNER_ADDRESS" ]; then
        success "Owner wallet created. Address: $OWNER_ADDRESS"
    else
        error "Could not extract owner address from output"
        echo "Owner wallet creation output:"
        echo "$owner_output"
        read -p "Please enter the owner address: " OWNER_ADDRESS
    fi
    
    log "Creating receiver wallet..."
    local receiver_output
    receiver_output=$(aztec-wallet create-account \
        --register-only \
        --node-url "$NODE_URL" \
        --alias receiver-wallet 2>&1)
    
    # Extract receiver address from output  
    local temp_receiver_address
    temp_receiver_address=$(extract_address_from_output "$receiver_output")
    
    if [ -n "$temp_receiver_address" ]; then
        RECEIVER_ADDRESS="$temp_receiver_address"
        success "Receiver wallet created. Address: $RECEIVER_ADDRESS"
    else
        error "Could not extract receiver address from output"
        echo "Receiver wallet creation output:"
        echo "$receiver_output"
        read -p "Please enter the receiver address: " RECEIVER_ADDRESS
    fi
}

# Step 3: Register accounts with FPC
register_with_fpc() {
    log "Registering wallets with FPC..."
    
    execute_with_retry "owner wallet FPC registration" \
        aztec-wallet register-contract \
        --node-url "$NODE_URL" \
        --from owner-wallet \
        --alias sponsoredfpc \
        "$SPONSORED_FPC_ADDRESS" SponsoredFPC \
        --salt 0
    
    execute_with_retry "receiver wallet FPC registration" \
        aztec-wallet register-contract \
        --node-url "$NODE_URL" \
        --from receiver-wallet \
        --alias sponsoredfpc \
        "$SPONSORED_FPC_ADDRESS" SponsoredFPC \
        --salt 0
}

# Step 4: Deploy accounts
deploy_accounts() {
    log "Deploying accounts..."
    warning "Note: 'Existing nullifier' errors indicate accounts are already deployed"
    info "This is expected when reusing the same private keys and will be handled automatically"
    info "Account deployment now uses default payment method to avoid array size constraints"
    
    log "Deploying owner account..."
    # Try deployment without FPC first to avoid array size issues
    execute_with_retry "owner wallet deployment" \
        aztec-wallet deploy-account \
        --node-url "$NODE_URL" \
        --from owner-wallet
    
    log "Deploying receiver account..."
    # Try deployment without FPC first to avoid array size issues
    execute_with_retry "receiver wallet deployment" \
        aztec-wallet deploy-account \
        --node-url "$NODE_URL" \
        --from receiver-wallet
    
    success "Account deployment process completed"
    info "Owner Address: $OWNER_ADDRESS"
    info "Receiver Address: $RECEIVER_ADDRESS"
    info "Both accounts are now ready for contract deployments"
}

# Step 5: Deploy Token contract
deploy_token_contract() {
    log "Deploying Token contract..."
    
    execute_with_dependency_retry "Token contract deployment" \
        aztec-wallet deploy \
        --node-url "$NODE_URL" \
        --from accounts:owner-wallet \
        --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
        --alias token \
        TokenContract \
        --args accounts:owner-wallet WormToken WORM 18 --no-wait
}

# Step 6: Mint tokens
mint_tokens() {
    log "Minting tokens..."
    
    info "Note: Minting may fail initially if previous deployments are still being processed"
    info "The script will automatically retry if needed"
    
    execute_with_dependency_retry "private token minting" \
        aztec-wallet send mint_to_private \
        --node-url "$NODE_URL" \
        --from accounts:owner-wallet \
        --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
        --contract-address "$TOKEN_CONTRACT_ADDRESS" \
        --args accounts:owner-wallet accounts:owner-wallet 10000
    
    execute_with_dependency_retry "public token minting" \
        aztec-wallet send mint_to_public \
        --node-url "$NODE_URL" \
        --from accounts:owner-wallet \
        --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
        --contract-address "$TOKEN_CONTRACT_ADDRESS" \
        --args accounts:owner-wallet 10000
}

# Backup original contract file
backup_contract() {
    if [ ! -f "$WORMHOLE_CONTRACT_BACKUP" ] && [ -f "$WORMHOLE_CONTRACT_SRC" ]; then
        log "Creating backup of original contract..."
        cp "$WORMHOLE_CONTRACT_SRC" "$WORMHOLE_CONTRACT_BACKUP"
        success "Contract backed up to $WORMHOLE_CONTRACT_BACKUP"
    fi
}

# Restore original contract from backup
restore_contract() {
    if [ -f "$WORMHOLE_CONTRACT_BACKUP" ]; then
        log "Restoring original contract from backup..."
        cp "$WORMHOLE_CONTRACT_BACKUP" "$WORMHOLE_CONTRACT_SRC"
        success "Original contract restored"
    fi
}

# Modify Wormhole contract with captured addresses
modify_wormhole_contract() {
    log "Modifying Wormhole contract with deployment addresses..."
    
    if [ ! -f "$WORMHOLE_CONTRACT_SRC" ]; then
        error "Wormhole contract source file not found: $WORMHOLE_CONTRACT_SRC"
        info "Expected file structure:"
        info "  src/"
        info "  └── main.nr (Wormhole contract)"
        exit 1
    fi
    
    if [ -z "$RECEIVER_ADDRESS" ] || [ -z "$TOKEN_CONTRACT_ADDRESS" ]; then
        error "Missing required addresses for contract modification"
        error "Receiver Address: $RECEIVER_ADDRESS"
        error "Token Contract Address: $TOKEN_CONTRACT_ADDRESS"
        exit 1
    fi
    
    # Create backup first
    backup_contract
    
    info "Updating hardcoded addresses in contract..."
    info "Receiver Address: $RECEIVER_ADDRESS"
    info "Token Contract Address: $TOKEN_CONTRACT_ADDRESS"
    
    # Find and replace the hardcoded addresses in the publish_message_in_private function
    # The current hardcoded addresses that need replacement:
    # receiver_address: 0x2f73c9b19222c2a7931c6cba01eedbbabb51e01b405fe4e0cabe0de91c275d0e
    # token_address: 0x13babb369e8c237a78ed507fe7cc44336a5178ffd02312a979c1fa0921f02a06
    
    local temp_file=$(mktemp)
    
    # Use sed to replace the hardcoded addresses
    sed "s/inner: 0x2f73c9b19222c2a7931c6cba01eedbbabb51e01b405fe4e0cabe0de91c275d0e/inner: $RECEIVER_ADDRESS/g" "$WORMHOLE_CONTRACT_SRC" > "$temp_file" && \
    sed "s/inner: 0x13babb369e8c237a78ed507fe7cc44336a5178ffd02312a979c1fa0921f02a06/inner: $TOKEN_CONTRACT_ADDRESS/g" "$temp_file" > "$WORMHOLE_CONTRACT_SRC"
    
    rm -f "$temp_file"
    
    # Verify the changes were made
    if grep -q "$RECEIVER_ADDRESS" "$WORMHOLE_CONTRACT_SRC" && grep -q "$TOKEN_CONTRACT_ADDRESS" "$WORMHOLE_CONTRACT_SRC"; then
        success "Contract addresses updated successfully"
        info "Receiver address updated in contract"
        info "Token contract address updated in contract"
    else
        error "Failed to update contract addresses"
        warning "Restoring original contract..."
        restore_contract
        exit 1
    fi
}

# Prepare Wormhole contract
prepare_wormhole_contract() {
    log "Preparing Wormhole contract..."
    
    # Modify the contract with the captured addresses
    modify_wormhole_contract
    
    # Compile the contract with retry logic
    compile_contract_with_retry
    
    # Run tests with retry logic
    test_contract_with_retry
    
    # Verify the compiled contract exists
    if [ ! -f "target/wormhole_contracts-Wormhole.json" ]; then
        error "Compiled Wormhole contract not found at target/wormhole_contracts-Wormhole.json"
        info "Expected compilation output location: target/wormhole_contracts-Wormhole.json"
        warning "Restoring original contract..."
        restore_contract
        exit 1
    fi
    
    success "Wormhole contract prepared successfully"
}

# Compile contract with retry logic
compile_contract_with_retry() {
    local max_attempts=3
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        log "Compiling Wormhole contract (attempt $attempt/$max_attempts)..."
        
        local compile_output
        local compile_exit_code=0
        compile_output=$(aztec-nargo compile 2>&1) || compile_exit_code=$?
        
        if [ $compile_exit_code -eq 0 ]; then
            success "Contract compilation completed successfully"
            echo "$compile_output"
            return 0
        else
            error "Contract compilation failed (attempt $attempt/$max_attempts)"
            echo "Compilation output:"
            echo "$compile_output"
            echo ""
            
            if [ $attempt -lt $max_attempts ]; then
                warning "Compilation failed - this might happen if:"
                echo "• The contract file is being modified while the script runs"
                echo "• There are temporary file system issues"
                echo "• The contract has syntax errors that were just introduced"
                echo ""
                
                echo "Options:"
                echo "1. Retry compilation (recommended if file was being edited)"
                echo "2. Exit and fix compilation errors manually" 
                echo "3. Skip compilation and try with existing artifacts (risky)"
                
                local choice
                read -p "Choose option (1/2/3) [default: 1]: " choice
                choice=${choice:-1}
                
                case $choice in
                    1)
                        info "Retrying compilation..."
                        if [ $attempt -eq 1 ]; then
                            info "Waiting 10 seconds in case files are still being modified..."
                            sleep 10
                        else
                            info "Waiting 5 seconds before retry..."
                            sleep 5
                        fi
                        attempt=$((attempt + 1))
                        continue
                        ;;
                    2)
                        info "Exiting for manual compilation fix"
                        warning "Restoring original contract..."
                        restore_contract
                        exit 1
                        ;;
                    3)
                        warning "Skipping compilation - using existing artifacts"
                        warning "This may cause deployment failures if artifacts are outdated"
                        return 0
                        ;;
                    *)
                        info "Invalid choice, defaulting to retry"
                        attempt=$((attempt + 1))
                        continue
                        ;;
                esac
            else
                error "Compilation failed after $max_attempts attempts"
                warning "Restoring original contract..."
                restore_contract
                
                echo ""
                echo "Compilation has failed multiple times. This usually means:"
                echo "• There are syntax errors in the contract"
                echo "• Missing dependencies or incorrect paths"
                echo "• The contract modifications introduced errors"
                echo ""
                echo "Please check the compilation errors above and fix them manually."
                echo "You can then run the script again or compile manually with:"
                echo "  aztec-nargo compile"
                exit 1
            fi
        fi
    done
}

# Test contract with retry logic  
test_contract_with_retry() {
    local max_attempts=2
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        log "Running contract tests (attempt $attempt/$max_attempts)..."
        
        local test_output
        local test_exit_code=0
        test_output=$(aztec test --silence-warnings 2>&1) || test_exit_code=$?
        
        if [ $test_exit_code -eq 0 ]; then
            success "Contract tests passed"
            echo "$test_output"
            return 0
        else
            warning "Contract tests failed (attempt $attempt/$max_attempts)"
            echo "Test output:"
            echo "$test_output"
            echo ""
            
            if [ $attempt -lt $max_attempts ]; then
                warning "Tests failed - this might happen if:"
                echo "• Contract was recently compiled and test cache is stale"
                echo "• Temporary testing environment issues"
                echo "• Tests depend on external state that's not ready"
                echo ""
                
                echo "Options:"
                echo "1. Retry tests (recommended for transient issues)"
                echo "2. Continue with deployment despite test failures (risky)"
                echo "3. Exit and fix test failures manually"
                
                local choice
                read -p "Choose option (1/2/3) [default: 1]: " choice
                choice=${choice:-1}
                
                case $choice in
                    1)
                        info "Retrying tests..."
                        info "Waiting 5 seconds for test environment to stabilize..."
                        sleep 5
                        attempt=$((attempt + 1))
                        continue
                        ;;
                    2)
                        warning "Continuing with deployment despite test failures"
                        warning "Deployment may fail if tests revealed actual issues"
                        return 0
                        ;;
                    3)
                        info "Exiting for manual test fix"
                        warning "Restoring original contract..."
                        restore_contract
                        exit 1
                        ;;
                    *)
                        info "Invalid choice, defaulting to retry"
                        attempt=$((attempt + 1))
                        continue
                        ;;
                esac
            else
                warning "Tests failed after $max_attempts attempts"
                
                echo ""
                echo "Do you want to continue with deployment despite test failures?"
                echo "This is risky but sometimes tests fail due to environment issues"
                echo "while the actual contract functionality is correct."
                
                local continue_deploy
                read -p "Continue with deployment? (y/n) [default: n]: " continue_deploy
                continue_deploy=${continue_deploy:-n}
                
                if [[ $continue_deploy =~ ^[Yy]$ ]]; then
                    warning "Continuing with deployment despite test failures"
                    warning "Monitor deployment carefully for any issues"
                    return 0
                else
                    info "Deployment cancelled by user"
                    warning "Restoring original contract..."
                    restore_contract
                    exit 1
                fi
            fi
        fi
    done
}

# Step 7: Deploy Wormhole contract
deploy_wormhole_contract() {
    log "Deploying Wormhole contract..."
    
    if [ -z "$RECEIVER_ADDRESS" ] || [ -z "$TOKEN_CONTRACT_ADDRESS" ]; then
        error "Missing required addresses for Wormhole deployment"
        error "Receiver Address: $RECEIVER_ADDRESS"
        error "Token Contract Address: $TOKEN_CONTRACT_ADDRESS"
        exit 1
    fi
    
    execute_with_dependency_retry "Wormhole contract deployment" \
        aztec-wallet deploy \
        --node-url "$NODE_URL" \
        --from accounts:owner-wallet \
        --payment method=fpc-sponsored,fpc=contracts:sponsoredfpc \
        --alias wormhole \
        target/wormhole_contracts-Wormhole.json \
        --args 56 56 "$RECEIVER_ADDRESS" "$TOKEN_CONTRACT_ADDRESS" --no-wait --init init
    
    # Restore original contract after deployment attempt
    log "Restoring original contract file..."
    restore_contract
}

# Create environment file for verification service
create_env_file() {
    local env_file=".env"
    
    log "Creating environment file for verification service..."
    
    # Create or overwrite .env file
    cat > "$env_file" << EOF
# Aztec Deployment Configuration
# Generated on $(date)

# Network Configuration
NODE_URL=$NODE_URL
PRIVATE_KEY=$OWNER_SK
SALT=0x0000000000000000000000000000000000000000000000000000000000000000

# Contract Addresses
OWNER_ADDRESS=$OWNER_ADDRESS
RECEIVER_ADDRESS=$RECEIVER_ADDRESS
TOKEN_CONTRACT_ADDRESS=$TOKEN_CONTRACT_ADDRESS
WORMHOLE_CONTRACT_ADDRESS=$WORMHOLE_CONTRACT_ADDRESS

# Service Configuration
PORT=3000
NETWORK=testnet
EOF
    
    success "Environment file created: $env_file"
    info "The verification service will now use these deployed contract addresses"
}

# Export environment variables for immediate use
export_environment_variables() {
    log "Exporting environment variables for verification service..."
    
    export NODE_URL="$NODE_URL"
    export PRIVATE_KEY="$OWNER_SK"
    export CONTRACT_ADDRESS="$WORMHOLE_CONTRACT_ADDRESS"
    export SALT="0x0000000000000000000000000000000000000000000000000000000000000000"
    export OWNER_ADDRESS="$OWNER_ADDRESS"
    export RECEIVER_ADDRESS="$RECEIVER_ADDRESS"
    export TOKEN_CONTRACT_ADDRESS="$TOKEN_CONTRACT_ADDRESS"
    export WORMHOLE_CONTRACT_ADDRESS="$WORMHOLE_CONTRACT_ADDRESS"
    export PORT="3000"
    export NETWORK="testnet"
    
    success "Environment variables exported for current session"
}

# Cleanup function
cleanup_on_exit() {
    warning "Script interrupted - cleaning up..."
    if [ -f "$WORMHOLE_CONTRACT_BACKUP" ]; then
        log "Restoring original contract..."
        restore_contract
        rm -f "$WORMHOLE_CONTRACT_BACKUP"
    fi
    exit 1
}

# Main execution function
main() {
    setup_wizard
    check_dependencies
    setup_environment
    
    log "Starting deployment process..."
    
    create_wallets
    register_with_fpc
    deploy_accounts
    deploy_token_contract
    
    # Add a pause to let the token contract deploy before minting
    info "Waiting for token contract to be ready before minting..."
    sleep 30
    
    mint_tokens
    prepare_wormhole_contract
    deploy_wormhole_contract
    
    success "🎉 Deployment completed successfully!"
    echo -e "\n${CYAN}Final Deployment Summary:${NC}"
    echo "├─ Node URL: $NODE_URL"
    echo "├─ Owner Wallet: $OWNER_ADDRESS"
    echo "├─ Receiver Wallet: $RECEIVER_ADDRESS"
    echo "├─ Token Contract: $TOKEN_CONTRACT_ADDRESS"
    
    if [ -n "$WORMHOLE_CONTRACT_ADDRESS" ]; then
        echo "├─ Wormhole Contract: $WORMHOLE_CONTRACT_ADDRESS"
    else
        echo "├─ Wormhole Contract: Deployed (check aztecscan for address)"
    fi
    
    echo "└─ Transaction Explorer: http://aztecscan.xyz/"
    echo ""
    success "All contracts deployed and ready for use!"
    info "Note: Contract addresses may take a few minutes to be fully propagated"
    
    # Create environment file and export variables
    create_env_file
    export_environment_variables
    
    # Clean up backup file
    if [ -f "$WORMHOLE_CONTRACT_BACKUP" ]; then
        rm -f "$WORMHOLE_CONTRACT_BACKUP"
        info "Cleanup completed"
    fi
}

# Handle script interruption and cleanup
trap cleanup_on_exit SIGINT SIGTERM

# Run the script
main "$@"