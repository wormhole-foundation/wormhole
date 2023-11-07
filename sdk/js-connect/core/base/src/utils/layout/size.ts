import {
  Layout,
  LayoutItem,
  LayoutToType,
  LayoutItemToType,
  isBytesType,
} from "./layout";
import { findIdLayoutPair } from "./utils";

function staticCalcItemSize(item: LayoutItem) {
  switch (item.binary) {
    case "int":
    case "uint": {
      return item.size;
    }
    case "bytes": {
      if ("size" in item && item.size !== undefined)
        return item.size;

      if (isBytesType(item.custom))
        return item.custom.length;

      if (isBytesType(item?.custom?.from))
        return item!.custom!.from.length;

      throw new Error("Cannot statically determine size of dynamic bytes");
    }
    case "array":
      throw new Error("Cannot statically determine size of dynamic array");
    case "object": {
      return calcLayoutSize(item.layout);
    }
    case "switch": {
      let size = null;
      if (item.layouts.length === 0)
        throw new Error(`switch item has no layouts`);

      for (const [_, layout] of item.layouts) {
        const layoutSize = calcLayoutSize(layout);
        if (size === null)
          size = layoutSize;
        else if (layoutSize !== size)
          throw new Error(
            "Cannot statically determine size of switch item with different layout sizes"
          );
      }
      return item.idSize + size!;
    }
  }
}

function calcItemSize(item: LayoutItem, data: any) {
  switch (item.binary) {
    case "int":
    case "uint": {
      return item.size;
    }
    case "bytes": {
      if ("size" in item && item.size !== undefined)
        return item.size;

      if (isBytesType(item.custom))
        return item.custom.length;

      if (isBytesType(item?.custom?.from))
        return item!.custom!.from.length;

      let size = 0;
      if ((item as { lengthSize?: number })?.lengthSize !== undefined)
        size += (item as { lengthSize: number }).lengthSize;

      return size + (
        (item.custom !== undefined)
        ? item.custom.from(data)
        : (data as LayoutItemToType<typeof item>)
      ).length;
    }
    case "array": {
      const narrowedData = data as LayoutItemToType<typeof item>;

      let size = 0;
      if ("length" in item && item.length !== narrowedData.length)
        throw new Error(`array length mismatch: ` +
          `layout length: ${item.length}, data length: ${narrowedData.length}`
        );
      else if ("lengthSize" in item && item.lengthSize !== undefined)
        size += item.lengthSize;

      for (let i = 0; i < narrowedData.length; ++i)
        size += calcLayoutSize(item.layout, narrowedData[i]);

      return size;
    }
    case "object": {
      return calcLayoutSize(item.layout, data as LayoutItemToType<typeof item>)
    }
    case "switch": {
      const [_, layout] = findIdLayoutPair(item, data);
      return item.idSize + calcLayoutSize(layout, data);
    }
  }
}

export function calcLayoutSize(
  layout: Layout,
  data?: LayoutToType<typeof layout>
): number {
  return (Array.isArray(layout))
    ? layout.reduce((acc, item) =>
        acc + (
          data !== undefined
          ? calcItemSize(item, data[item.name])
          : staticCalcItemSize(item)),
          0
        )
    : data !== undefined
    ? calcItemSize(layout as LayoutItem, data)
    : staticCalcItemSize(layout as LayoutItem);
}