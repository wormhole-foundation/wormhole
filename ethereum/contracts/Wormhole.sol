// contracts/Wormhole.sol
// SPDX-License-Identifier: Apache 2

// TODO(hendrik): reentrancy protection for all methods
// TODO(hendrik): switch-over feature
// TODO(hendrik): add call for retrying a lockup that the guardian set have refused to sign

pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";
import "@openzeppelin/contracts/math/SafeMath.sol";
import "./BytesLib.sol";
import "./WrappedAsset.sol";

contract Wormhole {
    using SafeERC20 for IERC20;
    using BytesLib for bytes;
    using SafeMath for uint256;

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
        bytes32 indexed token,
        bytes32 indexed sender,
        bytes32 recipient,
        uint256 amount
    );

    // Mapping of guardian_set_index => guardian set
    mapping(uint32 => GuardianSet) public guardian_sets;
    // Current active guardian set
    uint32 public guardian_set_index;

    // Period for which an vaa is valid in seconds
    uint32 public vaa_expiry;

    // Mapping of already consumedVAAs
    mapping(bytes32 => bool) consumedVAAs;

    // Mapping of wrapped asset ERC20 contracts
    mapping(bytes32 => address) public wrappedAssets;
    mapping(address => bool) public isWrappedAsset;

    constructor(GuardianSet memory initial_guardian_set, address wrapped_asset_master, uint32 _vaa_expiry) public {
        guardian_sets[0] = initial_guardian_set;
        // Explicitly set for doc purposes
        guardian_set_index = 0;
        vaa_expiry = _vaa_expiry;

        wrappedAssetMaster = wrapped_asset_master;
    }

    function getGuardianSet(uint32 idx) view public returns (GuardianSet memory gs) {
        return guardian_sets[idx];
    }

    function submitVAA(
        bytes calldata vaa
    ) public {
        uint8 version = vaa.toUint8(0);
        require(version == 1, "VAA version incompatible");

        // Load 4 bytes starting from index 1
        uint32 vaa_guardian_set_index = vaa.toUint32(1);

        uint256 len_signers = vaa.toUint8(5);
        uint offset = 6 + 66 * len_signers;

        // Load 4 bytes timestamp
        uint32 timestamp = vaa.toUint32(offset);

        // Verify that the VAA is still valid
        require(timestamp + vaa_expiry > block.timestamp, "VAA has expired");

        // Hash the body
        bytes32 hash = keccak256(vaa.slice(offset, vaa.length - offset));
        require(!consumedVAAs[hash], "VAA was already executed");

        GuardianSet memory guardian_set = guardian_sets[vaa_guardian_set_index];
        require(guardian_set.expiration_time == 0 || guardian_set.expiration_time > block.timestamp, "guardian set has expired");
        require(guardian_set.keys.length * 3 / 4 + 1 <= len_signers, "no quorum");

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
        guardian_sets[old_guardian_set_index].expiration_time = uint32(block.timestamp) + vaa_expiry;

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
        uint256 amount = data.toUint256(103);

        require(source_chain != target_chain, "same chain transfers are not supported");
        require(target_chain == CHAIN_ID, "transfer must be incoming");

        if (token_chain != CHAIN_ID) {
            bytes32 token_address = data.toBytes32(71);
            bytes32 asset_id = keccak256(abi.encodePacked(token_chain, token_address));

            // if yes: mint to address
            // if no: create and mint
            address wrapped_asset = wrappedAssets[asset_id];
            if (wrapped_asset == address(0)) {
                wrapped_asset = deployWrappedAsset(asset_id, token_chain, token_address);
            }

            WrappedAsset(wrapped_asset).mint(target_address, amount);
        } else {
            address token_address = data.toAddress(71 + 12);

            IERC20(token_address).safeTransfer(target_address, amount);
        }
    }

    function deployWrappedAsset(bytes32 seed, uint8 token_chain, bytes32 token_address) private returns (address asset){
        // Taken from https://github.com/OpenZeppelin/openzeppelin-sdk/blob/master/packages/lib/contracts/upgradeability/ProxyFactory.sol
        // Licensed under MIT
        bytes20 targetBytes = bytes20(wrappedAssetMaster);
        assembly {
            let clone := mload(0x40)
            mstore(clone, 0x3d602d80600a3d3981f3363d3d373d3d3d363d73000000000000000000000000)
            mstore(add(clone, 0x14), targetBytes)
            mstore(add(clone, 0x28), 0x5af43d82803e903d91602b57fd5bf30000000000000000000000000000000000)
            asset := create(0, clone, 0x37)
        }

        // Call initializer
        WrappedAsset(asset).initialize(token_chain, token_address);

        // Store address
        wrappedAssets[seed] = asset;
        isWrappedAsset[asset] = true;
    }

    // TODO(hendrik): nonce
    function lockAssets(
        address asset,
        uint256 amount,
        bytes32 recipient,
        uint8 target_chain
    ) public {
        require(amount != 0, "amount must not be 0");

        uint8 asset_chain = CHAIN_ID;
        bytes32 asset_address;
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
            asset_address = bytes32(uint256(asset));
        }

        emit LogTokensLocked(target_chain, asset_chain, asset_address, bytes32(uint256(msg.sender)), recipient, amount);
    }

    function lockETH(
        bytes32 recipient,
        uint8 target_chain
    ) public payable {
        require(msg.value != 0, "amount must not be 0");

        // Wrap tx value in WETH
        WETH(WETHAddress).deposit{value : msg.value}();

        // Log deposit of WETH
        emit LogTokensLocked(target_chain, CHAIN_ID, bytes32(uint256(WETHAddress)), bytes32(uint256(msg.sender)), recipient, msg.value);
    }


fallback() external payable {revert("please use lockETH to transfer ETH to Solana");}
receive() external payable {revert("please use lockETH to transfer ETH to Solana");}
}


interface WETH is IERC20 {
function deposit() external payable;

function withdraw(uint256 amount) external;
}
