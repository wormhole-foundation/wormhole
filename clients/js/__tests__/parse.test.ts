import { describe, expect, it } from "@jest/globals";
import { run_worm_command } from "./utils/cli";
import { test_command_positional_args } from "./utils/tests";

describe("worm parse", () => {
  describe("check arguments", () => {
    //Args must be defined in their specific order
    const args = ["vaa"];

    test_command_positional_args("parse", args);
  });

  describe("check functionality", () => {
    const vaa =
      "01000000010d0012e6b39c6da90c5dfd3c228edbb78c7a4c97c488ff8a346d161a91db067e51d638c17216f368aa9bdf4836b8645a98018ca67d2fec87d769cabfdf2406bf790a0002ef42b288091a670ef3556596f4f47323717882881eaf38e03345078d07a156f312b785b64dae6e9a87e3d32872f59cb1931f728cecf511762981baf48303668f0103cef2616b84c4e511ff03329e0853f1bd7ee9ac5ba71d70a4d76108bddf94f69c2a8a84e4ee94065e8003c334e899184943634e12043d0dda78d93996da073d190104e76d166b9dac98f602107cc4b44ac82868faf00b63df7d24f177aa391e050902413b71046434e67c770b19aecdf7fce1d1435ea0be7262e3e4c18f50ddc8175c0105d9450e8216d741e0206a50f93b750a47e0a258b80eb8fed1314cc300b3d905092de25cd36d366097b7103ae2d184121329ba3aa2d7c6cc53273f11af14798110010687477c8deec89d36a23e7948feb074df95362fc8dcbd8ae910ac556a1dee1e755c56b9db5d710c940938ed79bc1895a3646523a58bc55f475a23435a373ecfdd0107fb06734864f79def4e192497362513171530daea81f07fbb9f698afe7e66c6d44db21323144f2657d4a5386a954bb94eef9f64148c33aef6e477eafa2c5c984c01088769e82216310d1827d9bd48645ec23e90de4ef8a8de99e2d351d1df318608566248d80cdc83bdcac382b3c30c670352be87f9069aab5037d0b747208eae9c650109e9796497ff9106d0d1c62e184d83716282870cef61a1ee13d6fc485b521adcce255c96f7d1bca8d8e7e7d454b65783a830bddc9d94092091a268d311ecd84c26010c468c9fb6d41026841ff9f8d7368fa309d4dbea3ea4bbd2feccf94a92cc8a20a226338a8e2126cd16f70eaf15b4fc9be2c3fa19def14e071956a605e9d1ac4162010e23fcb6bd445b7c25afb722250c1acbc061ed964ba9de1326609ae012acdfb96942b2a102a2de99ab96327859a34a2b49a767dbdb62e0a1fb26af60fe44fd496a00106bb0bac77ac68b347645f2fb1ad789ea9bd76fb9b2324f25ae06f97e65246f142df717f662e73948317182c62ce87d79c73def0dba12e5242dfc038382812cfe00126da03c5e56cb15aeeceadc1e17a45753ab4dc0ec7bf6a75ca03143ed4a294f6f61bc3f478a457833e43084ecd7c985bf2f55a55f168aac0e030fc49e845e497101626e9d9a5d9e343f00010000000000000000000000000000000000000000000000000000000000000004c1759167c43f501c2000000000000000000000000000000000000000000000000000000000436f7265020000000000021358cc3ae5c097b213ce3c81979e1b9f9570746aa5ff6cb952589bde862c25ef4392132fb9d4a42157114de8460193bdf3a2fcf81f86a09765f4762fd1107a0086b32d7a0977926a205131d8731d39cbeb8c82b2fd82faed2711d59af0f2499d16e726f6b211b39756c042441be6d8650b69b54ebe715e234354ce5b4d348fb74b958e8966e2ec3dbd4958a7cd66b9590e1c41e0b226937bf9217d1d67fd4e91f574a3bf913953d695260d88bc1aa25a4eee363ef0000ac0076727b35fbea2dac28fee5ccb0fea768eaf45ced136b9d9e24903464ae889f5c8a723fc14f93124b7c738843cbb89e864c862c38cddcccf95d2cc37a4dc036a8d232b48f62cdd4731412f4890da798f6896a3331f64b48c12d1d57fd9cbe7081171aa1be1d36cafe3867910f99c09e347899c19c38192b6e7387ccd768277c17dab1b7a5027c0b3cf178e21ad2e77ae06711549cfbb1f9c7a9d8096e85e1487f35515d02a92753504a8d75471b9f49edb6fbebc898f403e4773e95feb15e80c9a99c8348d%";

    const expectedResult = {
      version: 1,
      guardianSetIndex: 1,
      signatures: [
        {
          guardianSetIndex: 0,
          signature:
            "12e6b39c6da90c5dfd3c228edbb78c7a4c97c488ff8a346d161a91db067e51d638c17216f368aa9bdf4836b8645a98018ca67d2fec87d769cabfdf2406bf790a00",
        },
        {
          guardianSetIndex: 2,
          signature:
            "ef42b288091a670ef3556596f4f47323717882881eaf38e03345078d07a156f312b785b64dae6e9a87e3d32872f59cb1931f728cecf511762981baf48303668f01",
        },
        {
          guardianSetIndex: 3,
          signature:
            "cef2616b84c4e511ff03329e0853f1bd7ee9ac5ba71d70a4d76108bddf94f69c2a8a84e4ee94065e8003c334e899184943634e12043d0dda78d93996da073d1901",
        },
        {
          guardianSetIndex: 4,
          signature:
            "e76d166b9dac98f602107cc4b44ac82868faf00b63df7d24f177aa391e050902413b71046434e67c770b19aecdf7fce1d1435ea0be7262e3e4c18f50ddc8175c01",
        },
        {
          guardianSetIndex: 5,
          signature:
            "d9450e8216d741e0206a50f93b750a47e0a258b80eb8fed1314cc300b3d905092de25cd36d366097b7103ae2d184121329ba3aa2d7c6cc53273f11af1479811001",
        },
        {
          guardianSetIndex: 6,
          signature:
            "87477c8deec89d36a23e7948feb074df95362fc8dcbd8ae910ac556a1dee1e755c56b9db5d710c940938ed79bc1895a3646523a58bc55f475a23435a373ecfdd01",
        },
        {
          guardianSetIndex: 7,
          signature:
            "fb06734864f79def4e192497362513171530daea81f07fbb9f698afe7e66c6d44db21323144f2657d4a5386a954bb94eef9f64148c33aef6e477eafa2c5c984c01",
        },
        {
          guardianSetIndex: 8,
          signature:
            "8769e82216310d1827d9bd48645ec23e90de4ef8a8de99e2d351d1df318608566248d80cdc83bdcac382b3c30c670352be87f9069aab5037d0b747208eae9c6501",
        },
        {
          guardianSetIndex: 9,
          signature:
            "e9796497ff9106d0d1c62e184d83716282870cef61a1ee13d6fc485b521adcce255c96f7d1bca8d8e7e7d454b65783a830bddc9d94092091a268d311ecd84c2601",
        },
        {
          guardianSetIndex: 12,
          signature:
            "468c9fb6d41026841ff9f8d7368fa309d4dbea3ea4bbd2feccf94a92cc8a20a226338a8e2126cd16f70eaf15b4fc9be2c3fa19def14e071956a605e9d1ac416201",
        },
        {
          guardianSetIndex: 14,
          signature:
            "23fcb6bd445b7c25afb722250c1acbc061ed964ba9de1326609ae012acdfb96942b2a102a2de99ab96327859a34a2b49a767dbdb62e0a1fb26af60fe44fd496a00",
        },
        {
          guardianSetIndex: 16,
          signature:
            "6bb0bac77ac68b347645f2fb1ad789ea9bd76fb9b2324f25ae06f97e65246f142df717f662e73948317182c62ce87d79c73def0dba12e5242dfc038382812cfe00",
        },
        {
          guardianSetIndex: 18,
          signature:
            "6da03c5e56cb15aeeceadc1e17a45753ab4dc0ec7bf6a75ca03143ed4a294f6f61bc3f478a457833e43084ecd7c985bf2f55a55f168aac0e030fc49e845e497101",
        },
      ],
      timestamp: 1651416474,
      nonce: 1570649151,
      emitterChain: 1,
      emitterAddress:
        "0x0000000000000000000000000000000000000000000000000000000000000004",
      sequence: "13940208096455381020",
      consistencyLevel: 32,
      payload: {
        module: "Core",
        type: "GuardianSetUpgrade",
        chain: 0,
        newGuardianSetIndex: 2,
        newGuardianSetLength: 19,
        newGuardianSet: [
          "58cc3ae5c097b213ce3c81979e1b9f9570746aa5",
          "ff6cb952589bde862c25ef4392132fb9d4a42157",
          "114de8460193bdf3a2fcf81f86a09765f4762fd1",
          "107a0086b32d7a0977926a205131d8731d39cbeb",
          "8c82b2fd82faed2711d59af0f2499d16e726f6b2",
          "11b39756c042441be6d8650b69b54ebe715e2343",
          "54ce5b4d348fb74b958e8966e2ec3dbd4958a7cd",
          "66b9590e1c41e0b226937bf9217d1d67fd4e91f5",
          "74a3bf913953d695260d88bc1aa25a4eee363ef0",
          "000ac0076727b35fbea2dac28fee5ccb0fea768e",
          "af45ced136b9d9e24903464ae889f5c8a723fc14",
          "f93124b7c738843cbb89e864c862c38cddcccf95",
          "d2cc37a4dc036a8d232b48f62cdd4731412f4890",
          "da798f6896a3331f64b48c12d1d57fd9cbe70811",
          "71aa1be1d36cafe3867910f99c09e347899c19c3",
          "8192b6e7387ccd768277c17dab1b7a5027c0b3cf",
          "178e21ad2e77ae06711549cfbb1f9c7a9d8096e8",
          "5e1487f35515d02a92753504a8d75471b9f49edb",
          "6fbebc898f403e4773e95feb15e80c9a99c8348d",
        ],
      },
      digest:
        "0x99656f88302bda18573212d4812daeea7d39f8af695db1fbc4d99fd94f552606",
    };

    it(`should return expected parse result from vaa`, async () => {
      const output = run_worm_command(`parse ${vaa}`);

      expect(JSON.parse(output)).toEqual(expectedResult);
    });
  });
});
