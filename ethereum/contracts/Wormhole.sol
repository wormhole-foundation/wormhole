// contracts/Wormhole.sol
// SPDX-License-Identifier: Apache 2

// TODO(hendrik): switch-over feature

pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";
import "@openzeppelin/contracts/math/SafeMath.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "./BytesLib.sol";
import "./WrappedAsset.sol";

contract Wormhole is ReentrancyGuard {
    using SafeERC20 for IERC20;
    using BytesLib for bytes;
    using SafeMath for uint256;

    uint64 constant MAX_UINT64 = 18_446_744_073_709_551_615;

    // Address of the Wrapped asset template
    address public wrappedAssetMaster;

    // Chain ID of Ethereum
    uint8 CHAIN_ID = 2;

    // Address of the official WETH contract
    address constant WETHAddress = 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;

    struct GuardianSet {
        address[] keys;
        uint32 expiration_time;
    }

    event LogGuardianSetChanged(
        uint32 oldGuardianIndex,
        uint32 newGuardianIndex
    );

    event LogTokensLocked(
        uint8 target_chain,
        uint8 token_chain,
        uint8 token_decimals,
        bytes32 indexed token,
        bytes32 indexed sender,
        bytes32 recipient,
        uint256 amount,
        uint32 nonce
    );

    // Mapping of guardian_set_index => guardian set
    mapping(uint32 => GuardianSet) public guardian_sets;
    // Current active guardian set
    uint32 public guardian_set_index;

    // Period for which a guardian set stays active after it has been replaced
    uint32 public guardian_set_expirity;

    // Mapping of already consumedVAAs
    mapping(bytes32 => bool) public consumedVAAs;

    // Mapping of wrapped asset ERC20 contracts
    mapping(bytes32 => address) public wrappedAssets;
    mapping(address => bool) public isWrappedAsset;

    constructor(GuardianSet memory initial_guardian_set, address wrapped_asset_master, uint32 _guardian_set_expirity) public {
        guardian_sets[0] = initial_guardian_set;
        // Explicitly set for doc purposes
        guardian_set_index = 0;
        guardian_set_expirity = _guardian_set_expirity;

        wrappedAssetMaster = wrapped_asset_master;
    }

    function getGuardianSet(uint32 idx) view public returns (GuardianSet memory gs) {
        return guardian_sets[idx];
    }

    function submitVAA(
        bytes calldata vaa
    ) public nonReentrant {
        uint8 version = vaa.toUint8(0);
        require(version == 1, "VAA version incompatible");

        // Load 4 bytes starting from index 1
        uint32 vaa_guardian_set_index = vaa.toUint32(1);

        uint256 len_signers = vaa.toUint8(5);
        uint offset = 6 + 66 * len_signers;

        // Load 4 bytes timestamp
        //uint32 timestamp = vaa.toUint32(offset);

        // Hash the body
        bytes32 hash = keccak256(vaa.slice(offset, vaa.length - offset));
        require(!consumedVAAs[hash], "VAA was already executed");

        GuardianSet memory guardian_set = guardian_sets[vaa_guardian_set_index];
        require(guardian_set.expiration_time == 0 || guardian_set.expiration_time > block.timestamp, "guardian set has expired");
        require(((guardian_set.keys.length / 4) * 3) + 1 <= len_signers, "no quorum");

        for (uint i = 0; i < len_signers; i++) {
            uint8 index = vaa.toUint8(6 + i * 66);
            bytes32 r = vaa.toBytes32(7 + i * 66);
            bytes32 s = vaa.toBytes32(39 + i * 66);
            uint8 v = vaa.toUint8(71 + i * 66);
            v += 27;
            require(ecrecover(hash, v, r, s) == guardian_set.keys[index], "VAA signature invalid");
        }

        uint8 action = vaa.toUint8(offset + 4);
        bytes memory payload = vaa.slice(offset + 5, vaa.length - (offset + 5));

        // Process VAA
        if (action == 0x01) {
            require(vaa_guardian_set_index == guardian_set_index, "only the current guardian set can change the guardian set");
            vaaUpdateGuardianSet(payload);
        } else if (action == 0x10) {
            vaaTransfer(payload);
        } else {
            revert("invalid VAA action");
        }

        // Set the VAA as consumed
        consumedVAAs[hash] = true;
    }

    function vaaUpdateGuardianSet(bytes memory data) private {
        uint32 new_guardian_set_index = data.toUint32(0);
        uint8 len = data.toUint8(4);

        address[] memory new_guardians = new address[](len);
        for (uint i = 0; i < len; i++) {
            address addr = data.toAddress(5 + i * 20);
            new_guardians[i] = addr;
        }

        uint32 old_guardian_set_index = guardian_set_index;
        guardian_set_index = new_guardian_set_index;

        GuardianSet memory new_guardian_set = GuardianSet(new_guardians, 0);
        guardian_sets[guardian_set_index] = new_guardian_set;
        guardian_sets[old_guardian_set_index].expiration_time = uint32(block.timestamp) + guardian_set_expirity;

        emit LogGuardianSetChanged(old_guardian_set_index, guardian_set_index);
    }

    function vaaTransfer(bytes memory data) private {
        //uint32 nonce = data.toUint64(0);
        uint8 source_chain = data.toUint8(4);

        uint8 target_chain = data.toUint8(5);
        //bytes32 source_address = data.toBytes32(6);
        //bytes32 target_address = data.toBytes32(38);
        address target_address = data.toAddress(38 + 12);

        uint8 token_chain = data.toUint8(70);
        //bytes32 token_address = data.toBytes32(71);
        uint256 amount = data.toUint256(104);

        require(source_chain != target_chain, "same chain transfers are not supported");
        require(target_chain == CHAIN_ID, "transfer must be incoming");

        if (token_chain != CHAIN_ID) {
            bytes32 token_address = data.toBytes32(71);
            bytes32 asset_id = keccak256(abi.encodePacked(token_chain, token_address));

            // if yes: mint to address
            // if no: create and mint
            address wrapped_asset = wrappedAssets[asset_id];
            if (wrapped_asset == address(0)) {
                uint8 asset_decimals = data.toUint8(103);
                wrapped_asset = deployWrappedAsset(asset_id, token_chain, token_address, asset_decimals);
            }

            WrappedAsset(wrapped_asset).mint(target_address, amount);
        } else {
            address token_address = data.toAddress(71 + 12);

            IERC20(token_address).safeTransfer(target_address, amount);
        }
    }

    function deployWrappedAsset(bytes32 seed, uint8 token_chain, bytes32 token_address, uint8 decimals) private returns (address asset){
        // Taken from https://github.com/OpenZeppelin/openzeppelin-sdk/blob/master/packages/lib/contracts/upgradeability/ProxyFactory.sol
        // Licensed under MIT
        bytes20 targetBytes = bytes20(wrappedAssetMaster);
        assembly {
            let clone := mload(0x40)
            mstore(clone, 0x3d602d80600a3d3981f3363d3d373d3d3d363d73000000000000000000000000)
            mstore(add(clone, 0x14), targetBytes)
            mstore(add(clone, 0x28), 0x5af43d82803e903d91602b57fd5bf30000000000000000000000000000000000)
            asset := create2(0, clone, 0x37, seed)
        }

        // Call initializer
        WrappedAsset(asset).initialize(token_chain, token_address, decimals);

        // Store address
        wrappedAssets[seed] = asset;
        isWrappedAsset[asset] = true;
    }

    function lockAssets(
        address asset,
        uint256 amount,
        bytes32 recipient,
        uint8 target_chain,
        uint32 nonce,
        bool refund_dust
    ) public nonReentrant {
        require(amount != 0, "amount must not be 0");

        uint8 asset_chain = CHAIN_ID;
        bytes32 asset_address;
        uint8 decimals = ERC20(asset).decimals();

        if (isWrappedAsset[asset]) {
            WrappedAsset(asset).burn(msg.sender, amount);
            asset_chain = WrappedAsset(asset).assetChain();
            asset_address = WrappedAsset(asset).assetAddress();
        } else {
            uint256 balanceBefore = IERC20(asset).balanceOf(address(this));
            IERC20(asset).safeTransferFrom(msg.sender, address(this), amount);
            uint256 balanceAfter = IERC20(asset).balanceOf(address(this));

            // The amount that was transferred in is the delta between balance before and after the transfer.
            // This is to properly handle tokens that charge a fee on transfer.
            amount = balanceAfter.sub(balanceBefore);

            // Decimal adjust amount - we keep the dust
            if (decimals > 9) {
                uint256 original_amount = amount;
                amount = amount.div(10 ** uint256(decimals - 9));

                if (refund_dust) {
                    ERC20(asset).transfer(msg.sender, original_amount.mod(10 ** uint256(decimals - 9)));
                }

                decimals = 9;
            }

            require(balanceAfter.div(10 ** uint256(ERC20(asset).decimals() - 9)) <= MAX_UINT64, "bridge balance would exceed maximum");

            asset_address = bytes32(uint256(asset));
        }

        emit LogTokensLocked(target_chain, asset_chain, decimals, asset_address, bytes32(uint256(msg.sender)), recipient, amount, nonce);
    }

    function lockETH(
        bytes32 recipient,
        uint8 target_chain,
        uint32 nonce
    ) public payable nonReentrant {
        require(msg.value != 0, "amount must not be 0");

        // Wrap tx value in WETH
        WETH(WETHAddress).deposit{value : msg.value}();

        // Log deposit of WETH
        emit LogTokensLocked(target_chain, CHAIN_ID, 18, bytes32(uint256(WETHAddress)), bytes32(uint256(msg.sender)), recipient, msg.value, nonce);
    }

    fallback() external payable {revert("please use lockETH to transfer ETH to Solana");}

    receive() external payable {revert("please use lockETH to transfer ETH to Solana");}
}


interface WETH is IERC20 {
    function deposit() external payable;

    function withdraw(uint256 amount) external;
}
