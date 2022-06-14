import defaultBackend from "./default";
import { Backend } from "./definitions";

let backend: Backend;
export const getBackend: () => Backend = () => {
  // Use the global one if it is already instantiated
  if (backend) {
    return backend;
  }
  if (process.env.CUSTOM_BACKEND) {
    try {
      backend = require(process.env.CUSTOM_BACKEND);
      return backend;
    } catch (e: any) {
      throw new Error(
        `Backend specified in CUSTOM_BACKEND is not importable: ${e?.message}`
      );
    }
  }
  if (!backend) {
    backend = defaultBackend;
  }
  return backend;
};
