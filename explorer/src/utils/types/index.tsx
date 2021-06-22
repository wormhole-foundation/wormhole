import { Link } from 'gatsby';

export type GetRenderComponentProps<T> = T extends
  | React.ComponentType
  | typeof Link
  ? React.ComponentProps<T>
  : T extends 'a'
  ? React.HTMLProps<T>
  : {};
