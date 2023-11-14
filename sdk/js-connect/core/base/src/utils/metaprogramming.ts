export type Extends<T, U> = [T] extends [U] ? true : false;

export type Function<P extends any[] = any[], R = any> = (...args: P) => R;
export type RoArray<T = unknown> = readonly T[];
export type RoArray2D<T = unknown> = RoArray<RoArray<T>>;
export type RoPair<T = unknown, U = unknown> = readonly [T, U];

export type Widen<T> =
  T extends string ? string :
  T extends number ? number :
  T extends boolean ? boolean :
  T extends bigint ? bigint :
  T extends object ? object :
  T;

//the Exclude<T, undefined> here seems silly but for some reason TypeScript incorrectly inferred
//  undefined as a possible value for T when used in conjunction with a generic type parameter
export type DefinedOrDefault<T, D> = undefined extends T ? D : Exclude<T, undefined>;

//see here: https://stackoverflow.com/a/55541672
export type IsAny<T> = Extends<0, 1 & T>;

export type IsNever<T> = Extends<T, never>;

//allow both And<reaonly boolean[]> and And<boolean, boolean>
export type And<T extends RoArray<boolean> | boolean, R extends boolean = true> =
  R extends true
  ? T extends RoArray<boolean>
    ? Extends<T[number], true>
    : Extends<T, true>
  : false;

export type Not<B extends boolean> = B extends true ? false : true;

export type ParseNumber<T> = T extends `${infer N extends number}` ? N : never;

//see here: https://stackoverflow.com/a/53955431
export type UnionToIntersection<U> =
  (U extends any ? (_: U) => void : never) extends ((_: infer I) => void) ? I : never;

export type IsUnion<T> = Not<Extends<T, UnionToIntersection<T>>>;

export type IsUnionMember<T, U> =
  And<[Extends<T,U>, IsUnion<U>, Not<IsUnion<T>>, Not<IsNever<T>>, Not<IsAny<T>>]>;

export type ConcatStringLiterals<A extends RoArray<string>> =
  A extends readonly [infer S extends string, ...infer Tail extends RoArray<string>]
  ? `${S}${ConcatStringLiterals<Tail>}`
  : "";

export type DistributiveOmit<T, K extends keyof any> = T extends any ? Omit<T, K> : never;

// type MyUnion = 'foo' | 'bar' | 'baz';
// type Test1 = IsUnionMember<'foo', MyUnion>; // true
// type Test2 = IsUnionMember<'bar', MyUnion>; // true
// type Test3 = IsUnionMember<'baz', MyUnion>; // true
// type Test4 = IsUnionMember<'foo' | 'baz', MyUnion>; // false
// type Test5 = IsUnionMember<any, MyUnion>; // false
// type Test6 = IsUnionMember<never, MyUnion>; // false
