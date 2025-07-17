import { storageKeys } from "storage/constants";
import { delState } from "storage/state/del";
import { getState } from "storage/state/get";
import { setState } from "storage/state/set";

const initialVaultId: string = "";

export const delVaultId = () => {
  delState(storageKeys.vaultId);
};

export const getVaultId = () => {
  return getState(storageKeys.vaultId, initialVaultId);
};

export const setVaultId = (vaultId: string) => {
  setState(storageKeys.vaultId, vaultId);
};
