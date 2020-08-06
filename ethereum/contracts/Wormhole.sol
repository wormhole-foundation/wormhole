// contracts/Wormhole.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";
import "./BytesLib.sol";
import "./SchnorrSECP256K1.sol";
import "./WrappedAsset.sol";

contract Wormhole {
    using SafeERC20 for IERC20;
    using BytesLib for bytes;

    // Address of the Wrapped asset template
    address public wrappedAssetMaster;

    // Chain ID of Ethereum
    uint8 CHAIN_ID = 2;

    // Address of the official WETH contract
    address constant WETHAddress = 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;

    struct GuardianSet {
        uint256 x;
        uint8 parity;
        uint32 expiration_time;
    }

    event LogGuardianSetChanged(
        GuardianSet indexed oldGuardian,
        GuardianSet indexed newGuardian
    );

    event LogTokensLocked(
        uint8 target_chain,
        uint8 token_chain,
        bytes32 indexed token,
        bytes32 indexed sender,
        bytes32 recipient,
        uint256 amount
    );

    event LogTokensUnlocked(
        address indexed token,
        bytes32 indexed sender,
        address recipient,
        uint256 amount
    );

    // Mapping of guardian_set_index => guardian set
    mapping(uint32 => GuardianSet)  private guardian_sets;
    // Current active guardian set
    uint32 public guardian_set_index;

    // Period for which an vaa is valid in seconds
    uint32 public vaa_expiry;

    // Mapping of already consumedVAAs
    mapping(bytes32 => bool) consumedVAAs;

    // Mapping of wrapped asset ERC20 contracts
    mapping(bytes32 => address) wrappedAssets;
    mapping(address => bool) isWrappedAsset;

    constructor(GuardianSet memory initial_guardian_set, address wrapped_asset_master) public {
        guardian_sets[0] = initial_guardian_set;
        // Explicitly set for doc purposes
        guardian_set_index = 0;

        wrappedAssetMaster = wrapped_asset_master;
    }

    function submitVAA(
        bytes calldata vaa
    ) public {
        uint8 version = vaa.toUint8(0);
        require(version == 1, "VAA version incompatible");

        // Load 4 bytes starting from index 1
        uint32 vaa_guardian_set_index = vaa.toUint32(1);

        uint256 signature = vaa.toUint256(5);
        address sig_address = vaa.toAddress(37);

        // Load 4 bytes starting from index 77
        uint32 timestamp = vaa.toUint32(57);

        // Verify that the VAA is still valid
        // TODO: the clock on Solana can't be trusted
        require(timestamp + vaa_expiry < block.timestamp, "VAA has expired");

        // Hash the body
        bytes32 hash = keccak256(vaa.slice(57, vaa.length - 57));
        require(!consumedVAAs[hash], "VAA was already executed");

        GuardianSet memory guardian_set = guardian_sets[vaa_guardian_set_index];
        require(guardian_set.expiration_time == 0 || guardian_set.expiration_time > block.timestamp, "guardian set has expired");
        require(
            Schnorr.verifySignature(
                guardian_set.x,
                guardian_set.parity,
                signature,
                uint256(hash),
                sig_address
            ),
            "VAA signature invalid");

        uint8 action = vaa.toUint8(61);
        uint8 payload_len = vaa.toUint8(62);
        bytes memory payload = vaa.slice(63, payload_len);

        // Process VAA
        if (action == 0x01) {
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
        uint256 new_key_x = data.toUint256(0);
        uint256 new_key_y = data.toUint256(32);
        uint32 new_guardian_set_index = data.toUint32(64);

        require(new_guardian_set_index > guardian_set_index, "index of new guardian set must be > current");
        require(new_key_x < Schnorr.HALF_Q, "invalid key for fast Schnorr verification");

        uint32 old_guardian_set_index = guardian_set_index;
        guardian_set_index = new_guardian_set_index;

        GuardianSet memory new_guardian_set = GuardianSet(new_key_x, uint8(new_key_y % 2), 0);
        guardian_sets[guardian_set_index] = new_guardian_set;
        guardian_sets[old_guardian_set_index].expiration_time = uint32(block.timestamp) + vaa_expiry;

        emit LogGuardianSetChanged(guardian_sets[old_guardian_set_index], new_guardian_set);
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
        // Safely transfer tokens out
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
            IERC20(asset).safeTransferFrom(msg.sender, address(this), amount);
            asset_address = bytes32(uint256(asset));
        }

        //        uint8 indexed target_chain,
        //        bytes32 indexed sender,
        //        bytes32 indexed recipient,
        //        uint8 indexed token_chain,
        //        address indexed token,
        //        uint256 amount
        emit LogTokensLocked(target_chain, asset_chain, asset_address, recipient, bytes32(uint256(msg.sender)), amount);
    }

    function lockETH(
        bytes32 recipient,
        uint8 target_chain
    ) public payable {
        require(msg.value != 0, "amount must not be 0");

        // Wrap tx value in WETH
        WETH(WETHAddress).deposit{value : msg.value}();

        // Log deposit of WETH
        emit LogTokensLocked(target_chain, CHAIN_ID, bytes32(uint256(WETHAddress)), recipient, bytes32(uint256(msg.sender)), msg.value);
    }


fallback() external payable {revert("please use lockETH to transfer ETH to Solana");}
receive() external payable {revert("please use lockETH to transfer ETH to Solana");}
}


interface WETH is IERC20 {
function deposit() external payable;

function withdraw(uint256 amount) external;
}
