// contracts/Wormhole.sol
// SPDX-License-Identifier: MIT

pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";
import "@openzeppelin/contracts/utils/EnumerableSet.sol";

contract Wormhole {
    using SafeERC20 for IERC20;
    using EnumerableSet for EnumerableSet.AddressSet;

    address constant WETHAddress = 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;

    uint256 CHAIN_ID = 2;

    struct Signature {
        uint8 v;
        bytes32 r;
        bytes32 s;
    }

    event LogGuardianKeyChanged(
        address indexed oldGuardian,
        address indexed newGuardian
    );

    event LogTokensLocked(
        address indexed token,
        bytes32 indexed recipient,
        uint256 amount
    );

    event LogTokensUnlocked(
        address indexed token,
        bytes32 indexed sender,
        address indexed recipient,
        uint256 amount
    );

    EnumerableSet.AddressSet private guardians;
    mapping(address => address) pendingGuardianTransfers;

    // Mappings guardian <=> authorized signer
    mapping(address => address) guardianToAuthorizedSigner;
    mapping(address => address) authorizedSignerToGuardian;

    // Mapping of already completed transactions
    mapping(bytes32 => bool) completedTransactions;

    constructor(address[] memory _guardians) public {
        require(_guardians.length > 0, "no guardians specified");

        for (uint i = 0; i < _guardians.length; i++) {
            guardians.add(_guardians[i]);
        }
    }

    function changeAuthorizedSigner(address newSigner) public {
        require(guardians.contains(msg.sender), "sender is not a guardian");
        require(authorizedSignerToGuardian[msg.sender] == address(0), "new signer is already a signer");

        // Unset old mapping
        address oldAuthorizedSigner = guardianToAuthorizedSigner[msg.sender];
        authorizedSignerToGuardian[oldAuthorizedSigner] = address(0);

        // Add new mapping
        authorizedSignerToGuardian[newSigner] = msg.sender;
        guardianToAuthorizedSigner[msg.sender] = newSigner;
    }

    function unlockERC20(
        address asset,
        uint256 amount,
        uint256 height,
        bytes32 sender,
        address recipient,
        Signature[] calldata signatures
    ) public {
        require(recipient != address(0), "assets should not be burned");

        // unlock data structure
        // asset  32bytes
        // height uint256
        // amount uint256
        // target_chain 32bytes
        // sender 32bytes
        // recipient 32bytes
        bytes32 hash = keccak256(
            abi.encodePacked(
                bytes32(uint256(asset)),
                amount,
                height,
                bytes32(CHAIN_ID),
                sender,
                bytes32(uint256(recipient))
            )
        );
        require(!completedTransactions[hash], "transfer was already executed");

        uint nSignatures = 0;
        address[] memory alreadySigned = new address[](signatures.length);
        for (uint256 i = 0; i < signatures.length; i++) {
            address signer = ecrecover(
                hash,
                signatures[i].v,
                signatures[i].r,
                signatures[i].s
            );

            address guardian = authorizedSignerToGuardian[signer];
            require(
                guardians.contains(authorizedSignerToGuardian[signer]),
                "signature of non-guardian included"
            );

            for (uint j = 0; j < alreadySigned.length; j++) {
                require(guardian != alreadySigned[j], "multiple signatures of the same guardian included");
            }

            alreadySigned[i] = guardian;
            nSignatures++;
        }

        // Check whether the threshold was met
        require(
            nSignatures > 5,
            "not enough valid signatures attached to unlock funds"
        );

        // Safely transfer tokens out
        IERC20(asset).safeTransfer(recipient, amount);

        // Set the transfer as completed
        completedTransactions[hash] = true;

        emit LogTokensUnlocked(asset, sender, recipient, amount);
    }

    function lockAssets(
        address asset,
        uint256 amount,
        bytes32 recipient
    ) public {
        require(amount != 0, "amount must not be 0");

        // TODO handle tokens that subtract fees
        IERC20(asset).safeTransferFrom(msg.sender, address(this), amount);
        emit LogTokensLocked(asset, recipient, amount);
    }

    function lockETH(
        bytes32 recipient
    ) public payable {
        require(msg.value != 0, "amount must not be 0");

        // Wrap tx value in WETH
        WETH(WETHAddress).deposit{value : msg.value}();

        // Log deposit of WETH
        emit LogTokensLocked(WETHAddress, recipient, msg.value);
    }


    fallback() external payable {revert("please use lockETH to transfer ETH to Solana");}
    receive() external payable {revert("please use lockETH to transfer ETH to Solana");}
}


interface WETH is IERC20 {
    function deposit() external payable;

    function withdraw(uint256 amount) external;
}
