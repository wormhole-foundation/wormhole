import { jest, test } from "@jest/globals";

jest.setTimeout(60000);

/*
This file tests to make sure that the network can start from genesis, and then change out the guardian set.

Prerequesites: Have two nodes running - the tilt guardian validator, and the 'second' wormhole chain node.

This test will register the the public ket of the second node, and then process a governance VAA to switch over the network.

*/

test("Verify guardian validator", async () => {});

test("Process guardian set upgrade", async () => {});

test("Register guardian to validator", async () => {});
