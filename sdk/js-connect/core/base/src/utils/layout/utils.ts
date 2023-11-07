import { BytesLayoutItem, SwitchLayoutItem, isBytesType } from "./layout";

export const checkUint8ArraySize = (custom: Uint8Array, size: number): void => {
  if (custom.length !== size)
    throw new Error(
      `binary size mismatch: layout size: ${custom.length}, data size: ${size}`
    );
}

export const checkNumEquals = (custom: number | bigint, data: number | bigint): void => {
  if (custom != data)
    throw new Error(
      `value mismatch: (constant) layout value: ${custom}, data value: ${data}`
    );
}

export const checkUint8ArrayDeeplyEqual = (custom: Uint8Array, data: Uint8Array): void => {
  checkUint8ArraySize(custom, data.length);

  for (let i = 0; i < custom.length; ++i)
    if (custom[i] !== data[i])
      throw new Error(
        `binary data mismatch: layout value: ${custom}, data value: ${data}`
      );
}

export function getBytesItemSize(bytesItem: BytesLayoutItem): number | null {
  if ("size" in bytesItem && bytesItem.size !== undefined)
    return bytesItem.size;

  if (isBytesType(bytesItem.custom))
    return bytesItem.custom.length;

  if (isBytesType(bytesItem?.custom?.from))
    return bytesItem!.custom!.from.length;

  return null;
}

export function findIdLayoutPair(item: SwitchLayoutItem, data: any) {
  const id = data[item.idTag ?? "id"];
  return (item.layouts as any[]).find(([idOrConversionId]) =>
    (Array.isArray(idOrConversionId) ? idOrConversionId[1] : idOrConversionId) == id
  )!;
}
