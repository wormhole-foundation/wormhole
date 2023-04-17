export type GithubTreeResponse = {
  sha: string;
  url: string;
  tree: {
    path: string;
    mode: string;
    type: string;
    sha: string;
    size: number;
    url: string;
  }[];
  truncated: boolean;
};

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
  type: string;
  objectId: string;
};
