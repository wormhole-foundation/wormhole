export type SuiError = {
  code: number;
  message: string;
  data: any;
};

export type SuiCoinObject = {
  coinType: string;
  coinObjectId: string;
};
