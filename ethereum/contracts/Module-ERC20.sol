// contracts/Module-ERC20.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";
import "@openzeppelin/contracts/math/SafeMath.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "./BytesLib.sol";
import "./WrappedAsset.sol";
import "./Wormhole.sol";

contract ERC20Bridge is ReentrancyGuard {
    using SafeERC20 for IERC20;
    using BytesLib for bytes;
    using SafeMath for uint256;

    uint8 public CHAIN_ID = 2;
    uint64 constant MAX_UINT64 = 18_446_744_073_709_551_615;

    // Address of the Wrapped asset template
    address public wrappedAssetMaster;
    // Address of the Wormhole
    Wormhole public wormhole;

    // Address of the official WETH contract
    address constant WETHAddress = 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;

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

    // Mapping of already consumedVAAs
    mapping(bytes32 => bool) public consumedVAAs;

    // Mapping of wrapped asset ERC20 contracts
    mapping(bytes32 => address) public wrappedAssets;
    mapping(address => bool) public isWrappedAsset;

    constructor(address wrapped_asset_master, address payable wormhole_bridge) public {
        wrappedAssetMaster = wrapped_asset_master;
        wormhole = Wormhole(wormhole_bridge);
    }

    function submitVAA(
        bytes calldata vaa
    ) public nonReentrant {
        Wormhole.ParsedVAA memory parsed_vaa = wormhole.parseAndVerifyVAA(vaa);
        require(!consumedVAAs[parsed_vaa.hash], "vaa was already executed");

        // Set the VAA as consumed
        consumedVAAs[parsed_vaa.hash] = true;

        // Execute transfer
        vaaTransfer(parsed_vaa.payload);
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

            uint8 decimals = ERC20(token_address).decimals();

            // Readjust decimals if they've previously been truncated
            if (decimals > 9) {
                amount = amount.mul(10 ** uint256(decimals - 9));
            }
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
        require(target_chain != CHAIN_ID, "must not transfer to the same chain");

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
                    IERC20(asset).safeTransfer(msg.sender, original_amount.mod(10 ** uint256(decimals - 9)));
                }

                decimals = 9;
            }

            require(balanceAfter.div(10 ** uint256(ERC20(asset).decimals() - 9)) <= MAX_UINT64, "bridge balance would exceed maximum");

            asset_address = bytes32(uint256(asset));
        }

        // Check here after truncation
        require(amount != 0, "truncated amount must not be 0");

        emit LogTokensLocked(target_chain, asset_chain, decimals, asset_address, bytes32(uint256(msg.sender)), recipient, amount, nonce);
    }

    function lockETH(
        bytes32 recipient,
        uint8 target_chain,
        uint32 nonce
    ) public payable nonReentrant {
        require(target_chain != CHAIN_ID, "must not transfer to the same chain");

        uint256 remainder = msg.value.mod(10 ** 9);
        uint256 transfer_amount = msg.value.div(10 ** 9);
        require(transfer_amount != 0, "truncated amount must not be 0");

        // Transfer back remainder
        msg.sender.transfer(remainder);

        // Wrap tx value in WETH
        WETH(WETHAddress).deposit{value : msg.value - remainder}();

        // Log deposit of WETH
        emit LogTokensLocked(target_chain, CHAIN_ID, 9, bytes32(uint256(WETHAddress)), bytes32(uint256(msg.sender)), recipient, transfer_amount, nonce);
    }

    fallback() external payable {revert("please use lockETH to transfer ETH to Solana");}

    receive() external payable {revert("please use lockETH to transfer ETH to Solana");}
}


interface WETH is IERC20 {
    function deposit() external payable;

    function withdraw(uint256 amount) external;
}
