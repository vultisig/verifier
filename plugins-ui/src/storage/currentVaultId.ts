import { delState } from "./del";
import { getState } from "./get";
import { setState } from "./set";

const initialValue = "";

const storageKey = "currentVaultId";

export const getCurrentVaultId = () => {
  return getState(storageKey, initialValue);
};

export const setCurrentVaultId = (vaultId: string) => {
  setState(storageKey, vaultId);
};
export const deleteCurrentVaultId = () => {
  delState(storageKey);
};
