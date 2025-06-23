import { getState } from "./get";
import { setState } from "./set";

type Tokens = Record<string, string>;

const initialTokens: Tokens = {};

const storageKey = "tokens";

const selectTokens = () => {
  return getState(storageKey, initialTokens);
};

const updateTokens = (tokens: Tokens) => {
  setState(storageKey, tokens);
};

export const createToken = (key: string, token: string) => {
  const tokens = selectTokens();
  tokens[key] = token;
  updateTokens(tokens);
};

export const deleteToken = (key: string) => {
  const tokens = selectTokens();
  delete tokens[key];
  updateTokens(tokens);
};

export const selectToken = (key: string) => {
  const tokens = selectTokens();
  return tokens[key];
};

export const updateToken = createToken;
