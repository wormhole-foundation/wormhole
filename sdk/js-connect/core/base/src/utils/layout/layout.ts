export type NumType = number | bigint;
export const isNumType = (x: any): x is NumType =>
  typeof x === "number" || typeof x === "bigint";

export type BytesType = Uint8Array;
export const isBytesType = (x: any): x is BytesType => x instanceof Uint8Array;

export type PrimitiveType = NumType | BytesType;
export const isPrimitiveType = (x: any): x is PrimitiveType =>
  isNumType(x) || isBytesType(x);

export type BinaryLiterals = "int" | "uint" | "bytes" | "array" | "object" | "switch";
export type Endianness = "little" | "big"; //default is always big

//Why only a max value of 2**(6*8)?
//quote from here: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/isInteger#description
//"In a similar sense, numbers around the magnitude of Number.MAX_SAFE_INTEGER will suffer from
//  loss of precision and make Number.isInteger return true even when it's not an integer.
//  (The actual threshold varies based on how many bits are needed to represent the decimal — for
//  example, Number.isInteger(4500000000000000.1) is true, but
//  Number.isInteger(4500000000000000.5) is false.)"
//So we are being conservative and just stay away from threshold.
export type NumberSize = 1 | 2 | 3 | 4 | 5 | 6;
export const numberMaxSize = 6;

export type NumSizeToPrimitive<Size extends number> =
  Size extends NumberSize ? number : bigint;

export type FixedConversion<FromType extends PrimitiveType, ToType> = {
  readonly to: ToType,
  readonly from: FromType,
};

export type CustomConversion<FromType extends PrimitiveType, ToType> = {
  readonly to: (val: FromType) => ToType,
  readonly from: (val: ToType) => FromType,
};

interface LayoutItemBase<BL extends BinaryLiterals> {
  readonly binary: BL,
};

interface FixedPrimitiveCustom<T extends PrimitiveType> {
  custom: T,
  omit?: boolean
};

interface OptionalToFromCustom<T extends PrimitiveType> {
  custom?: FixedConversion<T, any> | CustomConversion<T, any>
};

//size: number of bytes used to encode the item
interface NumLayoutItemBase<T extends NumType, Signed extends Boolean>
    extends LayoutItemBase<Signed extends true ? "int" : "uint"> {
  size: T extends bigint ? number : NumberSize,
  endianness?: Endianness, //default is big
};

export interface FixedPrimitiveNumLayoutItem<T extends NumType, Signed extends Boolean>
  extends NumLayoutItemBase<T, Signed>, FixedPrimitiveCustom<T> {};

export interface OptionalToFromNumLayoutItem<T extends NumType, Signed extends Boolean>
  extends NumLayoutItemBase<T, Signed>, OptionalToFromCustom<T> {};

export interface FixedPrimitiveBytesLayoutItem
  extends LayoutItemBase<"bytes">, FixedPrimitiveCustom<BytesType> {};

export interface FixedValueBytesLayoutItem extends LayoutItemBase<"bytes"> {
  readonly custom: FixedConversion<BytesType, any>,
};

export interface FixedSizeBytesLayoutItem extends LayoutItemBase<"bytes"> {
  readonly size: number,
  readonly custom?: CustomConversion<BytesType, any>,
};

//length size: number of bytes used to encode the preceeding length field which in turn
//  hold either the number of bytes (for bytes) or elements (for array)
//  undefined means it will consume the rest of the data
export interface LengthPrefixedBytesLayoutItem extends LayoutItemBase<"bytes"> {
  readonly lengthSize?: NumberSize,
  readonly lengthEndianness?: Endianness, //default is big
  readonly custom?: CustomConversion<BytesType, any>,
};

interface ArrayLayoutItemBase extends LayoutItemBase<"array"> {
  readonly layout: Layout,
};

export interface FixedSizeArrayLayoutItem extends ArrayLayoutItemBase {
  readonly length: number,
};

export interface LengthPrefixedArrayLayoutItem extends ArrayLayoutItemBase {
  readonly lengthSize?: NumberSize,
  readonly lengthEndianness?: Endianness, //default is big
};

export interface ObjectLayoutItem extends LayoutItemBase<"object"> {
  readonly layout: ProperLayout,
}

type PlainId = number;
type ConversionId = readonly [number, unknown];
type IdProperLayoutPair<
  Id extends PlainId | ConversionId,
  P extends ProperLayout = ProperLayout
> = readonly [Id, P];
type IdProperLayoutPairs =
  readonly IdProperLayoutPair<PlainId>[] | readonly IdProperLayoutPair<ConversionId>[];
export interface SwitchLayoutItem extends LayoutItemBase<"switch"> {
  readonly idSize: NumberSize,
  readonly idTag?: string,
  readonly idEndianness?: Endianness, //default is big
  readonly layouts: IdProperLayoutPairs,
}

export type NumLayoutItem<Signed extends boolean = boolean> =
  //force distribution over union
  Signed extends infer S extends boolean
  ? FixedPrimitiveNumLayoutItem<number, S> |
    OptionalToFromNumLayoutItem<number, S> |
    FixedPrimitiveNumLayoutItem<bigint, S> |
    OptionalToFromNumLayoutItem<bigint, S>
  : never;

export type UintLayoutItem = NumLayoutItem<false>;
export type IntLayoutItem = NumLayoutItem<true>;
export type BytesLayoutItem =
  FixedPrimitiveBytesLayoutItem |
  FixedValueBytesLayoutItem |
  FixedSizeBytesLayoutItem |
  LengthPrefixedBytesLayoutItem;
export type ArrayLayoutItem = FixedSizeArrayLayoutItem | LengthPrefixedArrayLayoutItem;
export type LayoutItem =
  NumLayoutItem |
  BytesLayoutItem |
  ArrayLayoutItem |
  ObjectLayoutItem |
  SwitchLayoutItem;
export type NamedLayoutItem = LayoutItem & { readonly name: string };
export type ProperLayout = readonly NamedLayoutItem[];
export type Layout = LayoutItem | ProperLayout;

type NameOrOmitted<T extends { name: string }> = T extends {omit: true} ? never : T["name"];

export type LayoutToType<L extends Layout> =
  L extends infer LI extends LayoutItem
  ? LayoutItemToType<LI>
  : L extends infer P extends ProperLayout
  ? { readonly [I in P[number] as NameOrOmitted<I>]: LayoutItemToType<I> }
  : never;

type MaybeConvert<Id extends PlainId | ConversionId> =
  Id extends readonly [number, infer Converted] ? Converted : Id;

type IdLayoutPairsToTypeUnion<A extends IdProperLayoutPairs, IdTag extends string> =
  A extends infer V extends IdProperLayoutPairs
  ? V extends readonly [infer Head,...infer Tail extends IdProperLayoutPairs]
    ? Head extends IdProperLayoutPair<infer MaybeConversionId, infer P extends ProperLayout>
      ? MaybeConvert<MaybeConversionId> extends infer Id
        ? LayoutToType<P> extends infer LT extends object
          ? { readonly [K in IdTag | keyof LT]: K extends keyof LT ? LT[K] : Id }
            | IdLayoutPairsToTypeUnion<Tail, IdTag>
          : never
        : never
      : never
    : never
  : never;

export type LayoutItemToType<Item extends LayoutItem> =
  Item extends infer I extends LayoutItem
  ? I extends NumLayoutItem
    ? I["custom"] extends NumType
      ? I["custom"]
      : I["custom"] extends CustomConversion<infer FromType extends NumType, infer ToType>
      ? ToType
      : I["custom"] extends FixedConversion<infer FromType extends NumType, infer ToType>
      ? ToType
      : NumSizeToPrimitive<I["size"]>
    : I extends BytesLayoutItem
    ? I["custom"] extends CustomConversion<BytesType, infer ToType>
      ? ToType
      : I["custom"] extends FixedConversion<BytesType, infer ToType>
      ? ToType
      : BytesType //this also covers FixedValueBytesLayoutItem (Uint8Arrays don't support literals)
    : I extends ArrayLayoutItem
    ? readonly LayoutToType<I["layout"]>[]
    : I extends ObjectLayoutItem
    ? LayoutToType<I["layout"]>
    : I extends SwitchLayoutItem
    ? IdLayoutPairsToTypeUnion<I["layouts"], I["idTag"] extends string ? I["idTag"] : "id">
    : never
  : never;
