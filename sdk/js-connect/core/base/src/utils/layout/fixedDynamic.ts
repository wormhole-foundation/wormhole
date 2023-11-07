import {
  Layout,
  ProperLayout,
  LayoutItem,
  NamedLayoutItem,
  NumLayoutItem,
  BytesLayoutItem,
  ObjectLayoutItem,
  ArrayLayoutItem,
  SwitchLayoutItem,
  LayoutToType,
  PrimitiveType,
  FixedConversion,
  isPrimitiveType,
} from "./layout";

type NonEmpty = readonly [unknown, ...unknown[]];

type IPLPair = readonly [any, ProperLayout];

type FilterItemsOfIPLPairs<ILA extends readonly IPLPair[], Fixed extends boolean> =
  ILA extends infer V extends readonly IPLPair[]
  ? V extends readonly [infer H extends IPLPair, ...infer T extends readonly IPLPair[]]
    ? FilterItemsOfLayout<H[1], Fixed> extends infer P extends ProperLayout | void
      ? P extends NonEmpty
        ? [[H[0], P], ...FilterItemsOfIPLPairs<T, Fixed>]
        : FilterItemsOfIPLPairs<T, Fixed>
      : never
    : []
  : never;

type FilterItem<Item extends LayoutItem, Fixed extends boolean> =
  Item extends infer I extends LayoutItem
  ? I extends NumLayoutItem | BytesLayoutItem
    ? I extends { custom: PrimitiveType | FixedConversion<PrimitiveType, any> }
      ? Fixed extends true ? I : void
      : Fixed extends true ? void : I
    : I extends ArrayLayoutItem
    ? FilterItemsOfLayout<I["layout"], Fixed> extends infer L extends Layout | void
      ? L extends LayoutItem | NonEmpty
        ? { readonly [K in keyof I]: K extends "layout" ? L : I[K] }
        : void
      : never
    : I extends ObjectLayoutItem
    ? FilterItemsOfLayout<I["layout"], Fixed> extends infer P extends ProperLayout
      ? P extends NonEmpty
        ? { readonly [K in keyof I]: K extends "layout" ? P : I[K] }
        : void
      : never
    : I extends SwitchLayoutItem
    ? { readonly [K in keyof I]:
        K extends "layouts" ? FilterItemsOfIPLPairs<I["layouts"], Fixed> : I[K]
      }
    : never
  : never;

type FilterItemsOfLayout<L extends Layout, Fixed extends boolean> =
  L extends infer LI extends LayoutItem
  ? FilterItem<LI, Fixed>
  : L extends infer P extends ProperLayout
  ? P extends readonly [infer H extends NamedLayoutItem, ...infer T extends ProperLayout]
    ? FilterItem<H, Fixed> extends infer NI
      ? NI extends NamedLayoutItem
        ? [NI, ...FilterItemsOfLayout<T, Fixed>]
        : FilterItemsOfLayout<T, Fixed>
      : never
    : []
  : never;

type StartFilterItemsOfLayout<L extends Layout, Fixed extends boolean> =
  FilterItemsOfLayout<L, Fixed> extends infer V extends Layout
  ? V
  : never;

function filterItem(item: LayoutItem, fixed: boolean): LayoutItem | null {
  switch (item.binary) {
    case "int":
    case "uint":
    case "bytes": {
      const isFixedItem = item["custom"] !== undefined && (
        isPrimitiveType(item["custom"]) || isPrimitiveType(item["custom"].from)
      );
      return (fixed && isFixedItem || !fixed && !isFixedItem) ? item : null;
    }
    case "array": {
      const filtered = internalFilterItemsOfLayout(item.layout, fixed);
      return (filtered !== null) ? { ...item, layout: filtered } : null;
    }
    case "object": {
      const filteredItems = internalFilterItemsOfProperLayout(item.layout, fixed);
      return (filteredItems.length > 0) ? { ...item, layout: filteredItems } : null;
    }
    case "switch": {
      const filteredIdLayoutPairs = (item.layouts as any[]).reduce(
        (acc: any, [idOrConversionId, idLayout]: any) => {
          const filteredItems = internalFilterItemsOfProperLayout(idLayout, fixed);
          return filteredItems.length > 0
            ? [...acc, [idOrConversionId, filteredItems]]
            : acc;
        },
        [] as any[]
      );
      return { ...item, layouts: filteredIdLayoutPairs };
    }
  }
}

function internalFilterItemsOfProperLayout(proper: ProperLayout, fixed: boolean): ProperLayout {
  return proper.reduce(
    (acc, item) => {
      const filtered = filterItem(item, fixed) as NamedLayoutItem | null;
      return filtered !== null ? [...acc, filtered] : acc;
    },
    [] as ProperLayout
  );
}

function internalFilterItemsOfLayout(layout: Layout, fixed: boolean): any {
  return (Array.isArray(layout)
    ? internalFilterItemsOfProperLayout(layout, fixed)
    : filterItem(layout as LayoutItem, fixed)
   ) as any;
}

function filterItemsOfLayout<L extends Layout, const Fixed extends boolean>(
  layout: L,
  fixed: Fixed
): FilterItemsOfLayout<L, Fixed> {
  return internalFilterItemsOfLayout(layout, fixed) as any;
}

export type FixedItemsOfLayout<L extends Layout> = StartFilterItemsOfLayout<L, true>;
export type DynamicItemsOfLayout<L extends Layout> = StartFilterItemsOfLayout<L, false>;

export const fixedItemsOfLayout = <L extends Layout>(layout: L) =>
  filterItemsOfLayout(layout, true);

export const dynamicItemsOfLayout = <L extends Layout>(layout: L) =>
  filterItemsOfLayout(layout, false);

function internalAddFixedValuesItem(item: LayoutItem, dynamicValue: any): any {
  switch (item.binary) {
    case "int":
    case "uint":
    case "bytes": {
      //look ma, ternary ternary operator!
      return !(item as {omit?: boolean})?.omit
        ? ( item.custom !== undefined &&
            (isPrimitiveType(item.custom) || isPrimitiveType((item.custom as {from: any}).from)) )
          ? isPrimitiveType(item.custom)
            ? item.custom
            : item.custom.to
          : dynamicValue
        : undefined;
    }
    case "array": {
      return Array.isArray(dynamicValue)
        ? dynamicValue.map(element =>
          internalAddFixedValues(item.layout, element))
        : undefined;
    }
    case "object": {
      return internalAddFixedValuesLayout(item.layout, dynamicValue ?? {});
    }
    case "switch": {
      const id = dynamicValue[item.idTag ?? "id"];
      const [_, idLayout] = (item.layouts as IPLPair[]).find(([idOrConversionId]) =>
        (Array.isArray(idOrConversionId) ? idOrConversionId[1] : idOrConversionId) == id
      )!;
      return {
        [item.idTag ?? "id"]: id,
        ...internalAddFixedValues(idLayout, dynamicValue)
      };
    }
  }
}

function internalAddFixedValuesLayout(proper: ProperLayout, dynamicValues: any): any {
  const ret = {} as any;
  for (const item of proper) {
    const r =
      internalAddFixedValuesItem(item, dynamicValues[item.name as keyof typeof dynamicValues]);
    if (r !== undefined)
      ret[item.name] = r;
  }
  return ret;
}

function internalAddFixedValues(layout: Layout, dynamicValues: any): any {
  return Array.isArray(layout)
    ? internalAddFixedValuesLayout(layout, dynamicValues)
    : internalAddFixedValuesItem(layout as LayoutItem, dynamicValues);
}

export function addFixedValues<L extends Layout>(
  layout: L,
  dynamicValues: LayoutToType<DynamicItemsOfLayout<L>>,
): LayoutToType<L> {
  return internalAddFixedValues(layout, dynamicValues) as LayoutToType<L>;
}
