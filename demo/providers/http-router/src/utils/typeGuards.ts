function isObject(obj: unknown): obj is Record<string, unknown> {
  return typeof obj === 'object' && obj !== null;
}

function hasProperty<T extends Record<string, unknown>, K extends string>(
  obj: T,
  prop: K,
): obj is T & Record<K, unknown> {
  return obj && prop in obj;
}

function isShapeShallow<T extends Record<string, unknown>>(obj: unknown, match: T): obj is T {
  if (!isObject(obj)) {
    return false;
  }

  for (const key in match) {
    if (typeof match[key] !== typeof obj[key]) {
      return false;
    }
  }

  return true;
}

export {isObject, hasProperty, isShapeShallow};
