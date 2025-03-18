pragma solidity >=0.8.4;

abstract contract FuzzingHelpers {    
    event LogAddress(address);
    event LogUint256(uint256);
    event LogString(string);
    event AssertFail(string);

    // We need this to receive refunds for quotes
    receive() external payable {}

    function clampBetween(uint256 value, uint256 low, uint256 high) internal returns (uint256){
        if (value < low || value > high) {
            uint ans = low + (value % (high - low + 1));
            return ans;
        }
        return value;
    }

    function extractErrorSelector(
        bytes memory revertData
    ) internal returns (uint256) {
        if (revertData.length < 4) {
            emit LogString("Return data too short.");
            return 0;
        }

        uint256 errorSelector = uint256(
            (uint256(uint8(revertData[0])) << 24) |
                (uint256(uint8(revertData[1])) << 16) |
                (uint256(uint8(revertData[2])) << 8) |
                uint256(uint8(revertData[3]))
        );

        return errorSelector;
    }

    function extractErrorString(bytes memory revertData) internal returns (bytes32) {
        if (revertData.length < 68) revert();
        assembly {
            revertData := add(revertData, 0x04)
        }
        return keccak256(abi.encodePacked(abi.decode(revertData, (string))));
    }

    function selectorToUint(bytes4 selector) internal returns (uint256) {
        return uint256(uint32(selector));
    }

    function assertWithMsg(bool b, string memory reason) internal {
        if (!b) {
            emit AssertFail(reason);
            assert(false);
        }
    }

    function minUint8(uint8 a, uint8 b) internal returns (uint8) {
        return a < b ? a : b;
    }

    function minUint256(uint256 a, uint256 b) internal returns (uint256) {
        return a < b ? a : b;
    }
}
