// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/utils/Create2.sol";
import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";
import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

/**
 * Contract factory that facilitates predfictable deployment addresses
 */
contract Create2Factory {
    event Created(address addr);

    address public immutable init;
    bytes32 immutable proxyBytecodeHash;

    constructor() {
        address initAddr = address(new Init());
        init = initAddr;
        proxyBytecodeHash = keccak256(
            abi.encodePacked(type(SimpleProxy).creationCode, abi.encode(address(initAddr)))
        );
    }

    /// @dev create2 hashes the userSalt with msg.sender, then uses the CREATE2 opcode to deterministically create a contract
    function create2(
        bytes memory userSalt,
        bytes memory bytecode
    ) public payable returns (address payable) {
        address addr = Create2.deploy(msg.value, salt(msg.sender, userSalt), bytecode);
        emit Created(addr);
        return payable(addr);
    }

    function create2Proxy(
        bytes memory userSalt,
        address impl,
        bytes memory call
    ) public payable returns (address payable) {
        address payable proxy = create2(
            userSalt, abi.encodePacked(type(SimpleProxy).creationCode, abi.encode(address(init)))
        );

        Init(proxy).upgrade(impl, call);
        return proxy;
    }

    function computeProxyAddress(
        address creator,
        bytes memory userSalt
    ) public view returns (address) {
        return Create2.computeAddress(salt(creator, userSalt), proxyBytecodeHash);
    }

    function computeAddress(
        address creator,
        bytes memory userSalt,
        bytes32 bytecodeHash
    ) public view returns (address) {
        return Create2.computeAddress(salt(creator, userSalt), bytecodeHash);
    }

    function salt(address creator, bytes memory userSalt) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(creator, userSalt));
    }
}

contract SimpleProxy is ERC1967Proxy {
    constructor(address impl) ERC1967Proxy(impl, new bytes(0)) {}
}

contract Init is ERC1967Upgrade {
    constructor() {}

    function upgrade(address impl, bytes memory call) external {
        _upgradeToAndCall(impl, call, false);
    }
}
