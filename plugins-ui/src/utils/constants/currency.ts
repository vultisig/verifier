export const currencies = [
  "usd",
  "eur",
  "gbp",
  "chf",
  "jpy",
  "cny",
  "cad",
  "sgd",
  "sek",
] as const;

export type Currency = (typeof currencies)[number];

export const defaultCurrency: Currency = "usd";

export const currencySymbols: Record<Currency, string> = {
  usd: "US$",
  eur: "€",
  gbp: "£",
  chf: "CHF",
  jpy: "JP¥",
  cny: "CN¥",
  cad: "CA$",
  sgd: "SGD",
  sek: "SEK",
};
