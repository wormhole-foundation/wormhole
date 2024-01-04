import { RoArray, RoArray2D, IsUnion } from "./metaprogramming";

//TODO the intent here is that number represents a number literal, but strictly speaking
//  the type allows for unions of number literals (and an array of such unions)
//The reason for not just sticking to unions is that unions lose order information which is
//  relevant in some cases (and iterating over them is a pain).
export type IndexEs = number | RoArray<number>;

export const range = (length: number) => [...Array(length).keys()];

export type Entries<T extends RoArray> =
  [...{ [K in keyof T]: K extends `${infer N extends number}` ? [T[K], N] : never }];

export type Flatten<T extends RoArray> =
  T extends readonly [infer Head, ...infer Tail extends RoArray]
  ? Head extends RoArray
    ? [...Head, ...Flatten<Tail>]
    : [Head, ...Flatten<Tail>]
  : [];

export type InnerFlatten<T extends RoArray> =
  [...{ [K in keyof T]:
    K extends `${number}`
    ? T[K] extends RoArray
      ? Flatten<T[K]>
      : T[K]
    : never
  }];

export type IsFlat<T extends RoArray> =
  T extends readonly [infer Head, ...infer Tail extends RoArray]
  ? Head extends RoArray
    ? false
    : IsFlat<Tail>
  : true;

export type Unflatten<T extends RoArray> =
  [...{ [K in keyof T]: K extends `${number}` ? [T[K]] : never }];

export type AllSameLength<T extends RoArray2D, L extends number | void = void> =
  T extends readonly [infer Head extends RoArray, ...infer Tail extends RoArray2D]
  ? L extends void
    ? AllSameLength<Tail, Head["length"]>
    : Head["length"] extends L
    ? AllSameLength<Tail, L>
    : false
  : true;

export type IsRectangular<T extends RoArray> =
  //1d array is rectangular
  T extends RoArray2D ? AllSameLength<T> : IsFlat<T>;

export type Column<A extends RoArray2D, I extends number> =
  [...{ [K in keyof A]: K extends `${number}` ? A[K][I] : never }];

export const column = <A extends RoArray2D, I extends number>(tupArr: A, index: I) =>
  tupArr.map((tuple) => tuple[index]) as Column<A, I>;

export type Zip<A extends RoArray2D> =
  //TODO remove, find max length, and return undefined for elements in shorter arrays
  A["length"] extends 0
  ? []
  : IsRectangular<A> extends true
  ? A[0] extends infer Head extends RoArray
    ? [...{ [K in keyof Head]:
        K extends `${number}`
        ? [...{ [K2 in keyof A]: K extends keyof A[K2] ? A[K2][K] : never }]
        : never
      }]
    : []
  : never

export const zip = <const Args extends RoArray2D>(arr: Args) =>
  range(arr[0]!.length).map(col =>
    range(arr.length).map(row => arr[row]![col])
  ) as unknown as ([Zip<Args>] extends [never] ? RoArray2D : Zip<Args>);

//extracts elements with the given indexes in the specified order, explicitly forbid unions
export type OnlyIndexes<E extends RoArray, I extends IndexEs> =
  IsUnion<I> extends false
    ? I extends number
    ? OnlyIndexes<E, [I]>
    : I extends readonly [infer Head extends number, ...infer Tail extends RoArray<number>]
    ? E[Head] extends undefined
      ? OnlyIndexes<E, Tail>
      : [E[Head], ...OnlyIndexes<E, Tail>]
    : []
  : never;

type ExcludeIndexesImpl<T extends RoArray, C extends number> =
  T extends readonly [infer Head, ...infer Tail]
  ? Head extends readonly [infer V, infer I extends number]
    ? I extends C
      ? ExcludeIndexesImpl<Tail, C>
      : [V, ...ExcludeIndexesImpl<Tail, C>]
    : never
  : [];

export type ExcludeIndexes<T extends RoArray, C extends IndexEs> =
  ExcludeIndexesImpl<Entries<T>, C extends RoArray<number> ? C[number] : C>;

export type Cartesian<L, R> =
  L extends RoArray
  ? Flatten<[...{ [K in keyof L]: K extends `${number}` ? Cartesian<L[K], R> : never }]>
  : R extends RoArray
  ? [...{ [K in keyof R]: K extends `${number}` ? [L, R[K]] : never }]
  : [L, R];
