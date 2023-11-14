export function lazyInstantiate<T>(factory: () => T): () => T {
  let instance: T | null = null;
  return () => {
    if (!instance)
      instance = factory();
    return instance;
  }
}

export function onlyOnce<T extends []>(fn: (...args: T) => any, ...args: T): () => void {
  let called = false;
  return () => {
    if (!called) {
      called = true;
      fn(...args);
    }
  }
}

export function throws(fn: () => any): boolean {
  try {
    fn();
    return false;
  } catch {
    return true;
  }
}
