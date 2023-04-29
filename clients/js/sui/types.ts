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

export type SuiCreateEvent = {
  sender: string;
  type: "created";
  objectType: string;
  objectId: string;
  version: string;
  digest: string;
  owner:
    | {
        AddressOwner: string;
      }
    | {
        ObjectOwner: string;
      }
    | {
        Shared: {
          initial_shared_version: number;
        };
      }
    | "Immutable";
};

export type SuiPublishEvent = {
  packageId: string;
  type: "published";
  version: number;
  digest: string;
  modules: string[];
};
