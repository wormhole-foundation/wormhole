export type ParsedMoveToml = {
  name: string;
  rows: { key: string; value: string }[];
}[];

export type SuiCreateEvent = {
  sender: string;
  type: "created";
  objectType: string;
  objectId: string;
  version: number;
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
