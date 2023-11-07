import { describe, expect, it } from "@jest/globals";

import {
  Layout,
  serializeLayout,
  deserializeLayout,
  addFixedValues,
  layoutDiscriminator
} from "../src";

const testLayout = [
  { name: "fixedDirectPrimitive", binary: "uint", size: 1, custom: 3 },
  {
    name: "fixedDirectCustom",
    binary: "uint",
    size: 1,
    custom: { to: 42, from: 1 },
  },
  { name: "dynamicDirectPrimitive", binary: "uint", size: 1 },
  {
    name: "dynamicDirectCustom",
    binary: "uint",
    size: 1,
    custom: { to: (val: number) => val + 1, from: (val: number) => val - 1 },
  },
  {
    name: "someDynamicObject",
    binary: "object",
    layout: [
      { name: "someDynamicBytes", binary: "bytes", size: 4 },
      { name: "someDynamicLengthBytes", binary: "bytes", lengthSize: 4 },
    ],
  },
  {
    name: "objectWithOnlyFixed",
    binary: "object",
    layout: [
      {
        name: "someFixedObjectUint",
        binary: "uint",
        size: 1,
        custom: { to: 13, from: 1 },
      },
    ],
  },
  {
    name: "objectWithSomeFixed",
    binary: "object",
    layout: [
      {
        name: "someFixedBytes",
        binary: "bytes",
        custom: { to: new Uint8Array(4), from: new Uint8Array(4) },
      },
      {
        name: "someFixedUint",
        binary: "uint",
        size: 1,
        custom: { to: 33, from: 1 },
      },
      { name: "someDynamicUint", binary: "uint", size: 1 },
    ],
  },
  {
    name: "arrayWithOnlyFixed",
    binary: "array",
    lengthSize: 1,
    layout: { binary: "uint", size: 1, custom: 12 },
  },
  {
    name: "arrayWithSomeFixed",
    binary: "array",
    lengthSize: 1,
    layout: { binary: "object", layout: [
      { name: "someDynamicUint", binary: "uint", size: 1 },
      { name: "someFixedUint", binary: "uint", size: 1, custom: 25 },
      {
        name: "someFixedBytes",
        binary: "bytes",
        custom: { to: new Uint8Array(4), from: new Uint8Array(4) },
      },
    ]},
  },
  {
    name: "arrayWithOnlyDynamic",
    binary: "array",
    lengthSize: 1,
    layout: { binary: "uint", size: 1 },
  },
  {
    name: "switchWithSomeFixed",
    binary: "switch",
    idSize: 2,
    layouts: [
      [1, [
        { name: "case1FixedUint", binary: "uint", size: 1, custom: 4 },
        { name: "case1DynamicUint", binary: "uint", size: 1 }
      ]],
      [3, [
        { name: "case2FixedBytes", binary: "bytes", custom: new Uint8Array(2) },
        { name: "case2DynamicBytes", binary: "bytes", size: 2 }
      ]],
    ],
  }
] as const satisfies Layout;

// uncomment the following to "test" correct type resolution:
// import { LayoutToType, FixedItemsOfLayout, DynamicItemsOfLayout } from "../src";
// type FixedItems = FixedItemsOfLayout<typeof testLayout>;
// type FixedValues = LayoutToType<FixedItems>;
// type DynamicItems = DynamicItemsOfLayout<typeof testLayout>;
// type DynamicValues = LayoutToType<DynamicItems>;

describe("Layout tests", function () {

  const completeValues = {
    fixedDirectPrimitive: 3,
    fixedDirectCustom: 42,
    dynamicDirectPrimitive: 2,
    dynamicDirectCustom: 4,
    someDynamicObject: {
      someDynamicBytes: new Uint8Array(4),
      someDynamicLengthBytes: new Uint8Array(5),
    },
    objectWithOnlyFixed: { someFixedObjectUint: 13 },
    objectWithSomeFixed: {
      someDynamicUint: 8,
      someFixedBytes: new Uint8Array(4),
      someFixedUint: 33,
    },
    arrayWithOnlyFixed: [],
    arrayWithSomeFixed: [
      {
        someDynamicUint: 10,
        someFixedUint: 25,
        someFixedBytes: new Uint8Array(4),
      },
      {
        someDynamicUint: 11,
        someFixedUint: 25,
        someFixedBytes: new Uint8Array(4),
      },
    ],
    arrayWithOnlyDynamic: [14, 16],
    switchWithSomeFixed: {
      id: 1,
      case1FixedUint: 4,
      case1DynamicUint: 18,
    }
  } as const;

  it("should correctly add fixed values", function () {
    const dynamicValues = {
      dynamicDirectPrimitive: 2,
      dynamicDirectCustom: 4,
      someDynamicObject: {
        someDynamicBytes: new Uint8Array(4),
        someDynamicLengthBytes: new Uint8Array(5),
      },
      arrayWithOnlyFixed: [],
      objectWithSomeFixed: { someDynamicUint: 8 },
      arrayWithSomeFixed: [{ someDynamicUint: 10 }, { someDynamicUint: 11 }],
      arrayWithOnlyDynamic: [14, 16],
      switchWithOnlyFixed: {
        customIdName: "case2",
      },
      switchWithSomeFixed: {
        id: 1,
        case1DynamicUint: 18,
      }
    } as const;

    const complete = addFixedValues(testLayout, dynamicValues);
    expect(complete).toEqual(completeValues);
  });

  const fixedInt = { name: "fixedSignedInt", binary: "int", size: 2 } as const;

  it("should correctly serialize and deserialize signed integers", function () {
    const layout = [fixedInt] as const;
    const encoded = serializeLayout(layout, { fixedSignedInt: -257 });
    expect(encoded).toEqual(new Uint8Array([0xfe, 0xff]));
    const decoded = deserializeLayout(layout, encoded);
    expect(decoded).toEqual({ fixedSignedInt: -257 });
  });

  it("should correctly serialize and deserialize little endian signed integers", function () {
    const layout = [{...fixedInt, endianness: "little"}] as const;
    const encoded = serializeLayout(layout, { fixedSignedInt: -257 });
    expect(encoded).toEqual(new Uint8Array([0xff, 0xfe]));
    const decoded = deserializeLayout(layout, encoded);
    expect(decoded).toEqual({ fixedSignedInt: -257 });
  });

  it("should serialize and deserialze correctly", function () {
    const encoded = serializeLayout(testLayout, completeValues);
    const decoded = deserializeLayout(testLayout, encoded);
    expect(decoded).toEqual(completeValues);
  });

  describe("Discriminate tests", function () {
    it("trivially discriminate by byte", function () {
      const discriminator = layoutDiscriminator([
        [{name: "type", binary: "uint", size: 1, custom: 0}],
        [{name: "type", binary: "uint", size: 1, custom: 2}],
      ]);

      expect(discriminator(Uint8Array.from([0]))).toBe(0);
      expect(discriminator(Uint8Array.from([2]))).toBe(1);
      expect(discriminator(Uint8Array.from([1]))).toBe(null);
      expect(discriminator(Uint8Array.from([]))).toBe(null);
      expect(discriminator(Uint8Array.from([0, 0]))).toBe(0);
    });

    it("discriminate by byte with different length", function () {
      const discriminator = layoutDiscriminator([
        [{name: "type", binary: "uint", size: 1, custom: 0},
         {name: "data", binary: "uint", size: 1}],
        [{name: "type", binary: "uint", size: 1, custom: 2}],
      ]);

      expect(discriminator(Uint8Array.from([0, 7]))).toBe(0);
      expect(discriminator(Uint8Array.from([2]))).toBe(1);
      expect(discriminator(Uint8Array.from([1]))).toBe(null);
      expect(discriminator(Uint8Array.from([]))).toBe(null);
      expect(discriminator(Uint8Array.from([0, 0]))).toBe(0);
    });

    it("discriminate by byte with out of bounds length length", function () {
      const discriminator = layoutDiscriminator([
        [{name: "data", binary: "uint", size: 1},
         {name: "type", binary: "uint", size: 1, custom: 0}],
        [{name: "data", binary: "uint", size: 1},
         {name: "type", binary: "uint", size: 1, custom: 2}],
        [{name: "type", binary: "uint", size: 1}],
      ]);

      expect(discriminator(Uint8Array.from([0, 0]))).toBe(0);
      expect(discriminator(Uint8Array.from([0, 2]))).toBe(1);
      expect(discriminator(Uint8Array.from([2]))).toBe(2);
      expect(discriminator(Uint8Array.from([0, 1]))).toBe(null);
      expect(discriminator(Uint8Array.from([0, 0, 0]))).toBe(0);
    });

    it("trivially discriminate by length", function() {
      const discriminator = layoutDiscriminator([
        [{name: "type", binary: "uint", size: 1},
         {name: "data", binary: "uint", size: 1}],
        [{name: "type", binary: "uint", size: 1}],
      ]);

      expect(discriminator(Uint8Array.from([0, 7]))).toBe(0);
      expect(discriminator(Uint8Array.from([0]))).toBe(1);
      expect(discriminator(Uint8Array.from([1]))).toBe(1);
      expect(discriminator(Uint8Array.from([]))).toBe(null);
      expect(discriminator(Uint8Array.from([0, 0, 0]))).toBe(null);
    });

    it("discriminate by byte and then size", function () {
      const discriminator = layoutDiscriminator([
        [{name: "type", binary: "uint", size: 1, custom: 0}],
        [{name: "type", binary: "uint", size: 1, custom: 2}],
        [{name: "type", binary: "uint", size: 1},
         {name: "data", binary: "uint", size: 1}],
      ]);

      expect(discriminator(Uint8Array.from([0, 7]))).toBe(2);
      expect(discriminator(Uint8Array.from([2]))).toBe(1);
      expect(discriminator(Uint8Array.from([1]))).toBe(2);
      expect(discriminator(Uint8Array.from([]))).toBe(null);
    });

    it("discriminate by byte and then either size or byte", function () {
      const discriminator = layoutDiscriminator([
        [{name: "type", binary: "uint", size: 1, custom: 0},
         {name: "data", binary: "uint", size: 1, custom: 0}],
        [{name: "type", binary: "uint", size: 1, custom: 0},
         {name: "data", binary: "uint", size: 1, custom: 1}],
        [{name: "type", binary: "uint", size: 1, custom: 1},
         {name: "data", binary: "uint", size: 1}],
        [{name: "type", binary: "uint", size: 1, custom: 1},
         {name: "data", binary: "uint", size: 1},
         {name: "dat2", binary: "uint", size: 1}]
      ]);

      expect(discriminator(Uint8Array.from([0, 0]))).toBe(0);
      expect(discriminator(Uint8Array.from([0, 1]))).toBe(1);
      expect(discriminator(Uint8Array.from([1, 0]))).toBe(2);
      expect(discriminator(Uint8Array.from([1, 0, 0]))).toBe(3);
      expect(discriminator(Uint8Array.from([]))).toBe(null);
    });

    it("cannot be uniquely discriminated", function () {
      const layouts = [
        [{name: "type", binary: "uint", size: 1}],
        [{name: "type", binary: "uint", size: 1}],
        [
          {name: "type", binary: "uint", size: 1},
          {name: "data", binary: "uint", size: 1}
        ],
      ] as readonly Layout[];
      expect (()=>layoutDiscriminator(layouts, false)).toThrow()

      const discriminator = layoutDiscriminator(layouts, true);
      expect(discriminator(Uint8Array.from([0]))).toEqual([0, 1]);
      expect(discriminator(Uint8Array.from([0, 0]))).toEqual([2]);
    });

  });
});
