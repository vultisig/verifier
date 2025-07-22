import { storageKeys } from "storage/constants";
import { getState } from "storage/state/get";
import { setState } from "storage/state/set";

const initialTokens: Record<string, string> = {};

const getTokens = () => {
  return getState(storageKeys.token, initialTokens);
};

export const delToken = (key: string) => {
  const tokens = getTokens();
  delete tokens[key];
  setState(storageKeys.token, tokens);
};

export const getToken = (key: string) => {
  const tokens = getTokens();
  return tokens[key];
};

export const setToken = (key: string, token: string) => {
  const tokens = getTokens();
  tokens[key] = token;
  setState(storageKeys.token, tokens);
};
