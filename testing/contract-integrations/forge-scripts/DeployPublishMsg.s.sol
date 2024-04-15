// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;
import {PublishMsg} from "../contracts/PublishMsg.sol";
import "forge-std/Script.sol";

contract DeployPublishMsg is Script {
    function dryRun(
        address wormhole
    ) public {
        _deploy(
            wormhole
        );
    }

    function run(
        address wormhole
    )
        public
        returns (
            address deployedAddress
        )
    {
        vm.startBroadcast();
        (
            deployedAddress
        ) = _deploy(
            wormhole
        );
        vm.stopBroadcast();
    }

    function _deploy(
        address wormhole
    )
        internal
        returns (
            address deployedAddress
        )
    {
        PublishMsg publishMsg = new PublishMsg(wormhole);

        return (
            address(publishMsg)
        );
    }
}
