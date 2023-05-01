export type ParsedMoveToml = {
  name: string;
  rows: { key: string; value: string }[];
}[];

export type SuiBuildOutput = {
  modules: string[];
  dependencies: string[];
};
