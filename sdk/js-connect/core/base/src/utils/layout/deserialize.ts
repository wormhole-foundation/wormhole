import {
  Endianness,
  Layout,
  LayoutItem,
  LayoutToType,
  FixedPrimitiveBytesLayoutItem,
  FixedValueBytesLayoutItem,
  CustomConversion,
  NumSizeToPrimitive,
  NumType,
  BytesType,
  isNumType,
  isBytesType,
  numberMaxSize,
} from "./layout";

import { checkUint8ArrayDeeplyEqual, checkNumEquals } from "./utils";

export function deserializeLayout<L extends Layout, B extends boolean = true>(
  layout: L,
  encoded: Uint8Array,
  offset?: number,
  consumeAll?: B,
) {
  const [decoded, finalOffset] = internalDeserializeLayout(layout, encoded, offset ?? 0);

  if ((consumeAll ?? true) && finalOffset !== encoded.length)
    throw new Error(`encoded data is longer than expected: ${encoded.length} > ${finalOffset}`);

  return (
    consumeAll ?? true ? decoded : [decoded, finalOffset]
  ) as B extends true ? LayoutToType<L> : readonly [LayoutToType<L>, number];
}

function internalDeserializeLayout(
  layout: Layout,
  encoded: Uint8Array,
  offset: number,
): readonly [any, number] {
  if (!Array.isArray(layout))
    return deserializeLayoutItem(layout as LayoutItem, encoded, offset);

  let decoded = {} as any;
  for (const item of layout)
    try {
      [((item as any).omit ? {} : decoded)[item.name], offset] =
        deserializeLayoutItem(item, encoded, offset);
    }
    catch (e) {
      (e as Error).message = `when deserializing item '${item.name}': ${(e as Error).message}`;
      throw e;
    }

  return [decoded, offset];
}

function updateOffset(
  encoded: Uint8Array,
  offset: number,
  size: number
): number {
  const newOffset = offset + size;
  if (newOffset > encoded.length)
    throw new Error(`encoded data is shorter than expected: ${encoded.length} < ${newOffset}`);

  return newOffset;
}

function deserializeNum<S extends number>(
  encoded: Uint8Array,
  offset: number,
  bytes: S,
  endianness: Endianness = "big",
  signed: boolean = false,
): readonly [NumSizeToPrimitive<S>, number] {
  let val = 0n;
  for (let i = 0; i < bytes; ++i)
    val |= BigInt(encoded[offset + i]!) << BigInt(8 * (endianness === "big" ? bytes - i - 1 : i));

  //check sign bit if value is indeed signed and adjust accordingly
  if (signed && (encoded[offset + (endianness === "big" ? 0 : bytes - 1)]! & 0x80))
    val -= 1n << BigInt(8 * bytes);

  return [
    ((bytes > numberMaxSize) ? val : Number(val)) as NumSizeToPrimitive<S>,
    updateOffset(encoded, offset, bytes)
  ] as const;
}

function deserializeLayoutItem(
  item: LayoutItem,
  encoded: Uint8Array,
  offset: number,
): readonly [any, number] {
  switch (item.binary) {
    case "int":
    case "uint": {
      const [value, newOffset] =
        deserializeNum(encoded, offset, item.size, item.endianness, item.binary === "int");

      if (isNumType(item.custom)) {
        checkNumEquals(item.custom, value);
        return [item.custom, newOffset];
      }

      if (isNumType(item?.custom?.from)) {
        checkNumEquals(item!.custom!.from, value);
        return [item!.custom!.to, newOffset];
      }

      //narrowing to CustomConver<UintType, any> is a bit hacky here, since the true type
      //  would be CustomConver<number, any> | CustomConver<bigint, any>, but then we'd have to
      //  further tease that apart still for no real gain...
      type narrowedCustom = CustomConversion<NumType, any>;
      return [
        item.custom !== undefined ? (item.custom as narrowedCustom).to(value) : value,
        newOffset
      ];
    }
    case "bytes": {
      let newOffset;
      let fixedFrom;
      let fixedTo;
      if (item.custom !== undefined) {
        if (isBytesType(item.custom))
          fixedFrom = item.custom;
        else if (isBytesType(item.custom.from)) {
          fixedFrom = item.custom.from;
          fixedTo = item.custom.to;
        }
      }

      if (fixedFrom !== undefined)
        newOffset = updateOffset(encoded, offset, fixedFrom.length);
      else {
        item = item as
          Exclude<typeof item, FixedPrimitiveBytesLayoutItem | FixedValueBytesLayoutItem>;
        if ("size" in item && item.size !== undefined)
          newOffset = updateOffset(encoded, offset, item.size);
        else if ("lengthSize" in item && item.lengthSize !== undefined) {
          let length;
          [length, offset] =
            deserializeNum(encoded, offset, item.lengthSize, item.lengthEndianness);
          newOffset = updateOffset(encoded, offset, length);
        }
        else
          newOffset = encoded.length;
      }

      const value = encoded.slice(offset, newOffset);
      if (fixedFrom !== undefined) {
        checkUint8ArrayDeeplyEqual(fixedFrom, value);
        return [fixedTo ?? fixedFrom, newOffset];
      }

      type narrowedCustom = CustomConversion<BytesType, any>;
      return [
        item.custom !== undefined ? (item.custom as narrowedCustom).to(value) : value,
        newOffset
      ];
    }
    case "array": {
      let ret = [] as any[];
      const { layout } = item;
      const deserializeArrayItem = () => {
        const [deserializedItem, newOffset] = internalDeserializeLayout(layout, encoded, offset);
        ret.push(deserializedItem);
        offset = newOffset;
      }

      let length = null;
      if ("length" in item)
        length = item.length;
      else if (item.lengthSize !== undefined)
        [length, offset] =
          deserializeNum(encoded, offset, item.lengthSize, item.lengthEndianness);

      if (length !== null)
        for (let i = 0; i < length; ++i)
          deserializeArrayItem();
      else
        while (offset < encoded.length)
          deserializeArrayItem();

      return [ret, offset];
    }
    case "object": {
      return internalDeserializeLayout(item.layout, encoded, offset);
    }
    case "switch": {
      const [id, newOffset] = deserializeNum(encoded, offset, item.idSize, item.idEndianness);
      const {layouts} = item;
      if (layouts.length === 0)
        throw new Error(`switch item has no layouts`);

      const hasPlainIds = typeof layouts[0]![0] === "number";
      const pair = (layouts as any[]).find(([idOrConversionId]) =>
        hasPlainIds ? idOrConversionId === id : (idOrConversionId)[0] === id);

      if (pair === undefined)
        throw new Error(`unknown id value: ${id}`);

      const [idOrConversionId, idLayout] = pair;
      const [decoded, nextOffset] = internalDeserializeLayout(idLayout, encoded, newOffset);
      return [
        { [item.idTag ?? "id"]: hasPlainIds ? id : (idOrConversionId as any)[1],
          ...decoded
        },
        nextOffset
      ];
    }
  }
}
