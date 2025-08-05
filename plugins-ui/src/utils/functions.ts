import { Dayjs } from "dayjs";
import { CSSProperties, PluginPolicy } from "utils/types";

const isArray = (arr: any): arr is any[] => {
  return Array.isArray(arr);
};

const isObject = (obj: any): obj is Record<string, any> => {
  return obj === Object(obj) && !isArray(obj) && typeof obj !== "function";
};

const toCamel = (value: string): string => {
  return value.replace(/([-_][a-z])/gi, ($1) => {
    return $1.toUpperCase().replace("-", "").replace("_", "");
  });
};

const toKebab = (value: string): string => {
  return value.replace(/[A-Z]/g, (letter) => `-${letter.toLowerCase()}`);
};

const toSnake = (value: string): string => {
  return value.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);
};

export const cssPropertiesToString = (styles: CSSProperties): string => {
  return Object.entries(styles)
    .map(([key, value]) => `${toKebab(key)}: ${value};`)
    .join("\n");
};

export const getErrorMessage = (error: any, message: string) => {
  return isObject(error) ? error.message : message;
};

export const isUndefined = (val: any): val is undefined => {
  return typeof val === "undefined";
};

export const match = <T extends string | number | symbol, V>(
  value: T,
  handlers: { [key in T]: () => V }
): V => {
  const handler = handlers[value];

  return handler();
};

export const policyToHexMessage = ({
  pluginVersion,
  policyVersion,
  publicKey,
  recipe,
}: Pick<
  PluginPolicy,
  "pluginVersion" | "policyVersion" | "publicKey" | "recipe"
>): string => {
  const delimiter = "*#*";

  const fields = [recipe, publicKey, String(policyVersion), pluginVersion];

  for (const item of fields) {
    if (item.includes(delimiter)) {
      throw new Error("invalid policy signature");
    }
  }

  return fields.join(delimiter);
};

export const toCamelCase = <T>(obj: T): T => {
  if (isObject(obj)) {
    const result: Record<string, unknown> = {};

    Object.keys(obj).forEach((key) => {
      const camelKey = toCamel(key);
      result[camelKey] = toCamelCase((obj as Record<string, unknown>)[key]);
    });

    return result as T;
  } else if (isArray(obj)) {
    return obj.map((item) => toCamelCase(item)) as T;
  }

  return obj;
};

export const toCapitalizeFirst = (value: string) => {
  return value.length ? value.charAt(0).toUpperCase() + value.slice(1) : value;
};

export const toKebabCase = <T>(obj: T): T => {
  if (isObject(obj)) {
    const result: Record<string, unknown> = {};

    Object.keys(obj).forEach((key) => {
      const kebabKey = toKebab(key);
      result[kebabKey] = toKebabCase((obj as Record<string, unknown>)[key]);
    });

    return result as T;
  } else if (isArray(obj)) {
    return obj.map((item) => toKebabCase(item)) as T;
  }

  return obj;
};

export const toNumeralFormat = (value: number) => {
  const formattedValue = value.toLocaleString("en-US", {
    style: "decimal",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  });

  return formattedValue;
};

export const toSnakeCase = <T>(obj: T): T => {
  if (isObject(obj)) {
    const result: Record<string, unknown> = {};

    Object.keys(obj).forEach((key) => {
      const snakeKey = toSnake(key);
      result[snakeKey] = toSnakeCase((obj as Record<string, unknown>)[key]);
    });

    return result as T;
  } else if (isArray(obj)) {
    return obj.map((item) => toSnakeCase(item)) as T;
  }

  return obj;
};

export const toTimestamp = (input: Date | Dayjs) => {
  const date = input instanceof Date ? input : input.toDate();
  const millis = date.getTime();

  return {
    nanos: (millis % 1000) * 1_000_000,
    seconds: BigInt(Math.floor(millis / 1000)),
  };
};
