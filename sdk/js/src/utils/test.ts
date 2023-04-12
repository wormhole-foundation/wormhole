import { expect } from "@jest/globals";

// https://github.com/microsoft/TypeScript/issues/34523
export const assertIsNotNull: <T>(x: T | null) => asserts x is T = (x) => {
  expect(x).not.toBeNull();
};

export const assertIsNotNullOrUndefined: <T>(
  x: T | null | undefined
) => asserts x is T = (x) => {
  expect(x).not.toBeNull();
  expect(x).not.toBeUndefined();
};
