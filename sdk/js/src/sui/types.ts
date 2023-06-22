export type ParsedMoveToml = {
  name: string;
  rows: { key: string; value: string }[];
}[];

export type SuiBuildOutput = {
  modules: string[];
  dependencies: string[];
};

export type SuiError = {
  code: number;
  message: string;
  data: any;
};

export type SuiCoinObject = {
  coinType: string;
  coinObjectId: string;
};
