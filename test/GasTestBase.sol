// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import "forge-std/Test.sol";

import { IWormhole        } from "wormhole-sdk/interfaces/IWormhole.sol";
import { PublishedMessage } from "wormhole-sdk/testing/WormholeOverride.sol";

contract GasTestBase is Test {
  IWormhole public wormhole;

  constructor() {
    wormhole = IWormhole(0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B);
  }

  function _tbPublishedMsg() internal pure returns (PublishedMessage memory) {
    return PublishedMessage({
      timestamp: 0x670acde5,
      emitterChainId: 0x17,
      emitterAddress: 0x0000000000000000000000000b2402144bb366a632d14b83f244d2e0e21bd39c,
      sequence: 0x45e61,
      nonce: 0x163d234e,
      consistencyLevel: 1,
      payload: _tbPayload()
    });
  }

  //source tx hash from ethereum mainnet:
  //  0xedd3ac96bc37961cce21a33fd50449dba257737c168006b40aa65496aaf92449
  function _tbOriginalVaa() internal pure returns (bytes memory) {
    bytes[] memory signatures = new bytes[](13);
    signatures[0] = abi.encodePacked(
      bytes1(0x00), //index
      bytes32(0xd7c298ee56b3860ca3597e2ce51568e029585cf3b4706d65df785f763b90f74e), //r
      bytes32(0x77c5e774eb2dcc0422ed2bad6ec7490fbfb18f4621b4a267cb583ba195797951), //s
      bytes1(0x01) //v
    );
    signatures[1] = abi.encodePacked(
      bytes1(0x01),
      bytes32(0x0391f369925909f013921495ca4c8d769746a63fa237a1f282c6557a1e121887),
      bytes32(0x5666d4f4cfcb88a9156a1684194d845ec94789f022a8dd38a99de7298bb5436b),
      bytes1(0x00)
    );
    signatures[2] = abi.encodePacked(
      bytes1(0x02),
      bytes32(0xe5ebcbe493fb9e5dddc5619956620efce4ac9735acad92954e24bebc6a639d94),
      bytes32(0x2d4512f823b248644f752e423f5087eb588ce188af5c68c51f7f32b70e18b801),
      bytes1(0x00)
    );
    signatures[3] = abi.encodePacked(
      bytes1(0x03),
      bytes32(0x7f54c89b8f646f940780d0029c50068c25e34ba312ae5450fdda0450c25143ce),
      bytes32(0x7e10a54368e7412a4970c6b6872aa1cc1f75e62a8640c3df43641163ccba7432),
      bytes1(0x01)
    );
    signatures[4] = abi.encodePacked(
      bytes1(0x04),
      bytes32(0x4a70b8d851b3c8dcdcd30328c8b7802fe91b4b8bf97871f9055fbc3fd7ccb7dc),
      bytes32(0x7a561005acde4932268ad79f80367b323dbfbffc911f395b02c0061dc6f4f5aa),
      bytes1(0x01)
    );
    signatures[5] = abi.encodePacked(
      bytes1(0x07),
      bytes32(0x3df702e117ca425813e4ddb073419ff7964df5a58f487839ad6241a46c6c0d44),
      bytes32(0x2d4fe29797ea43732b60b00b4c32b9bf0165e2fbe0092144ee3ce6f8dc2e9936),
      bytes1(0x00)
    );
    signatures[6] = abi.encodePacked(
      bytes1(0x09),
      bytes32(0xc048a769158bca4473eb492e2d79af32fdcc6a2615a0715f3c881d3f8cf0d897),
      bytes32(0x1039aeff11502fa76c55e72330e1996f540ce8f8f5523dee6c8e4c196e5d786f),
      bytes1(0x01)
    );
    signatures[7] = abi.encodePacked(
      bytes1(0x0a),
      bytes32(0x26997af10642283328110a4719c57beffd0b46d395cd71aeb4fac3c008060091),
      bytes32(0x711279551534826d1823160433001aea254939e85a474838c255aea60b10fc49),
      bytes1(0x01)
    );
    signatures[8] = abi.encodePacked(
      bytes1(0x0c),
      bytes32(0x5bcf6fda82527ac97320dc63312d82f22f267d3dae2b5acbf9e231219d84e2ee),
      bytes32(0x0bcdbc83fed098d491de778addd92d15a2e857b55c425ea594c15dd7de2a53c8),
      bytes1(0x00)
    );
    signatures[9] = abi.encodePacked(
      bytes1(0x0d),
      bytes32(0x8f091ad26b04859876d5d691a3bb178d13d4374bc76af94e0afcff6a4c6a9515),
      bytes32(0x08b962526b842a70559376d7dc7b55f28477d7528a40596270b1c45fe01b880d),
      bytes1(0x01)
    );
    signatures[10] = abi.encodePacked(
      bytes1(0x0e),
      bytes32(0x1f4bb6680b8ccc46adab4f4599a9d3affb77464ddb15a829cd6f231c53a80b59),
      bytes32(0x675039aa6d5c6c21a9b8dbae6270c8cc75c61b61137b4e21871d86d4076865b1),
      bytes1(0x00)
    );
    signatures[11] = abi.encodePacked(
      bytes1(0x10),
      bytes32(0xbc34ccbf4c7199b996e6af44cddbbf900b7cb2e6c2a930b280cae7c67f6513f2),
      bytes32(0x686b63ec6bbd3c2efdf46084972c8490bdef290119f0a19f40871389247dba0f),
      bytes1(0x00)
    );
    signatures[12] = abi.encodePacked(
      bytes1(0x12),
      bytes32(0x1be73670f42a9819a25a10aacad7a3b8082c411d824953ca655245c43987ee51),
      bytes32(0x1f0b0853ee4994b0c8f65b5f4fc748a1e5bbe4cc33d1ae2423e0e9469a1e0955),
      bytes1(0x00)
    );

    return abi.encodePacked(
      uint8(1), //version
      uint32(4), //guardian set
      uint8(13), //signature count
      signatures[0],
      signatures[1],
      signatures[2],
      signatures[3],
      signatures[4],
      signatures[5],
      signatures[6],
      signatures[7],
      signatures[8],
      signatures[9],
      signatures[10],
      signatures[11],
      signatures[12],
      _tbVaaPayload()
    );
  }

  function _tbVaaPayload() internal pure returns (bytes memory) {
    return abi.encodePacked(
      bytes32(0x670acde5163d234e00170000000000000000000000000b2402144bb366a632d1),
      bytes19(0x4b83f244d2e0e21bd39c0000000000045e6101),
      _tbPayload()
    );
  }

  function _tbPayload() internal pure returns (bytes memory) {
    return abi.encodePacked(
      bytes32(0x01000000000000000000000000000000000000000000000000000197020a499f),
      bytes32(0x3c000000000000000000000000ff836a5821e69066c87e268bc51b849fab9424),
      bytes32(0x0c00020000000000000000000000000a6c69327d517568e6308f1e1cd2fd2b2b),
      bytes32(0x3cd4bf0002000000000000000000000000000000000000000000000000000000),
      bytes5 (0x0000000000)
    );
  }
}
