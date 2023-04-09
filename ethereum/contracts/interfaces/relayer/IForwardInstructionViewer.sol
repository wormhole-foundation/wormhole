// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../interfaces/IWormholeRelayerInternalStructs.sol";

interface IForwardInstructionViewer {
    function getForwardInstruction()
        external
        view
        returns (IWormholeRelayerInternalStructs.ForwardInstruction memory);

    function encodeDeliveryInstructionsContainer(
        IWormholeRelayerInternalStructs.DeliveryInstructionsContainer memory container
    ) external pure returns (bytes memory encoded);

    /**
     * @notice Helper function that converts an Wormhole format (32-byte) address to the EVM 'address' 20-byte format
     * @param whFormatAddress (32-byte address in Wormhole format)
     * @return addr (EVM 20-byte address)
     */
    function fromWormholeFormat(bytes32 whFormatAddress) external pure returns (address addr);
}
