import { storageKeys } from "storage/constants";
import { getState } from "storage/state/get";
import { setState } from "storage/state/set";
import { type Currency,defaultCurrency } from "utils/constants/currency";

export const getCurrency = () => {
  return getState(storageKeys.currency, defaultCurrency);
};

export const setCurrency = (currency: Currency) => {
  setState(storageKeys.currency, currency);
};
