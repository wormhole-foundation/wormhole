export type DataWrapper<T> = {
  data: T | null;
  error: any | null;
  isFetching: boolean;
  receivedAt: string | null;
  //possibly invalidate
};

export function getEmptyDataWrapper() {
  return {
    data: null,
    error: null,
    isFetching: false,
    receivedAt: null,
  };
}

export function receiveDataWrapper<T>(data: T): DataWrapper<T> {
  return {
    data,
    error: null,
    isFetching: false,
    receivedAt: new Date().toISOString(),
  };
}

export function errorDataWrapper<T>(error: string): DataWrapper<T> {
  return {
    data: null,
    error,
    isFetching: false,
    receivedAt: null,
  };
}

export function fetchDataWrapper() {
  return {
    data: null,
    error: null,
    isFetching: true,
    receivedAt: null,
  };
}
