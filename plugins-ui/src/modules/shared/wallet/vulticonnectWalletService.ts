// more on the exposed methods here: https://github.com/vultisig/vultisig-windows/blob/main/clients/extension/docs/integration-guide.md

import { publish } from "@/utils/eventBus";

interface ProviderError {
  code: number;
  message: string;
}

const VulticonnectWalletService = {
  connectToVultiConnect: async () => {
    if (!window.vultisig?.ethereum) {
      publish("onToast", {
        message: "No ethereum provider found. Please install VultiConnect.",
        type: "error",
      });
      return;
    }

    try {
      const accounts = await window.vultisig.ethereum.request({
        method: "eth_requestAccounts",
      });

      return accounts;
    } catch (error) {
      const { code, message } = error as ProviderError;
      console.error(`Connection failed - Code: ${code}, Message: ${message}`);
      throw error;
    }
  },

  getConnectedEthAccounts: async () => {
    if (!window.vultisig?.ethereum) {
      publish("onToast", {
        message: "No ethereum provider found. Please install VultiConnect.",
        type: "error",
      });
      return;
    }

    try {
      const accounts = await window.vultisig.ethereum.request({
        method: "eth_accounts",
      });

      return accounts;
    } catch (error) {
      const { code, message } = error as ProviderError;
      console.error(
        `Failed to get accounts - Code: ${code}, Message: ${message}`
      );
      throw error;
    }
  },

  signCustomMessage: async (hexMessage: string, walletAddress: string) => {
    if (!window.vultisig?.ethereum) {
      publish("onToast", {
        message: "No ethereum provider found. Please install VultiConnect.",
        type: "error",
      });
      return;
    }

    console.log("hexMessage", hexMessage);
    console.log("walletAddress", walletAddress);

    try {
      const signature = await window.vultisig.ethereum.request({
        method: "personal_sign",
        params: [hexMessage, walletAddress],
      });

      console.log("signature", signature);

      if (signature && signature.error) {
        throw signature.error;
      }
      return signature;
    } catch (error) {
      console.error("Failed to sign the message", error);
      throw new Error("Failed to sign the message");
    }
  },

  getVaults: async () => {
    if (!window.vultisig) {
      publish("onToast", {
        message: "VultiConnect extension not found. Please install it first.",
        type: "error",
      });
      throw new Error("VultiConnect extension not found");
    }

    try {
      const vaults = await window.vultisig.getVaults();
      console.log("Retrieved vaults:", vaults);

      if (!vaults || vaults.length === 0) {
        publish("onToast", {
          message: "No vaults found. Please create a vault in VultiConnect first.",
          type: "error",
        });
        throw new Error("No vaults found");
      }

      return vaults;
    } catch (error) {
      console.error("Failed to get vaults:", error);
      publish("onToast", {
        message: "Failed to get vaults. Please make sure VultiConnect is properly installed and initialized.",
        type: "error",
      });
      throw error;
    }
  },
};

export default VulticonnectWalletService;
