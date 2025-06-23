// more on the exposed methods here: https://github.com/vultisig/vultisig-windows/blob/main/clients/extension/docs/integration-guide.md

import { publish } from "@/utils/eventBus";
import { decompressQrPayload, decodeTssPayload } from "./vultisigProtoUtils";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";

class Exception extends Error {
  code: number;

  constructor(code: number, message: string) {
    super(message);
    this.code = code;
  }
}

const VulticonnectWalletService = {
  isExtensionAvailable: async () => {
    if (!window.vultisig) {
      publish("onToast", {
        message:
          "No ethereum provider found. Please install Vultisig Extension.",
        type: "error",
      });

      throw new Exception(404, "Please install Vultisig Extension");
    }

    return;
  },
  connectToVultiConnect: async () => {
    await VulticonnectWalletService.isExtensionAvailable();

    try {
      const [account]: string[] = await window.vultisig.ethereum.request({
        method: "eth_requestAccounts",
      });

      return account;
    } catch (error) {
      const { code, message } = error as Exception;
      console.error(`Connection failed - Code: ${code}, Message: ${message}`);
      throw error;
    }
  },
  getConnectedEthAccounts: async () => {
    await VulticonnectWalletService.isExtensionAvailable();

    try {
      const accounts = await window.vultisig.ethereum.request({
        method: "eth_accounts",
      });

      return accounts;
    } catch (error) {
      const { code, message } = error as Exception;
      console.error(
        `Failed to get accounts - Code: ${code}, Message: ${message}`
      );
      throw error;
    }
  },
  signCustomMessage: async (hexMessage: string, walletAddress: string) => {
    await VulticonnectWalletService.isExtensionAvailable();

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
  getVault: async () => {
    await VulticonnectWalletService.isExtensionAvailable();

    try {
      const vault = await window.vultisig.getVault();

      if (vault) {
        if (!vault.hexChainCode || !vault.publicKeyEcdsa) {
          throw new Exception(400, "Missing required vault data");
        }

        return vault;
      } else {
        publish("onToast", {
          message:
            "Vault not found. Please create a vault in Vultisig Extension first.",
          type: "error",
        });

        throw new Exception(404, "Vault not found");
      }
    } catch (error) {
      publish("onToast", {
        message:
          "Failed to get vaults. Please make sure VultiConnect is properly installed and initialized.",
        type: "error",
      });

      throw new Exception(
        500,
        error instanceof Error ? error.message : String(error)
      );
    }
  },
  startReshareSession: async (pluginId: any) => {
    await VulticonnectWalletService.isExtensionAvailable();

    try {
      const response = await window.vultisig.plugin.request({
        method: "plugin_request_reshare",
        params: [{ id: pluginId }],
      });
      console.log("response", response);
      // Example response: vultisig://vultisig.com?type=NewVault&tssType=Reshare&jsonData=...
      const url = new URL(response);
      console.log("url", url);
      const jsonData = url.searchParams.get("jsonData");
      // const tssType = url.searchParams.get("tssType");
      // console.log("jsonData", jsonData);
      if (!jsonData) throw new Error("jsonData param missing in deeplink");
      // Decompress the payload
      const payload = await decompressQrPayload(jsonData);

      // Decode the binary using the schema and forward to verifier backend
      const reshareMsg: any = decodeTssPayload(payload);

      // Transform the payload to match backend ReshareRequest structure
      const backendPayload = {
        name: reshareMsg.vaultName,
        public_key: reshareMsg.publicKeyEcdsa,
        session_id: reshareMsg.sessionId,
        hex_encryption_key: reshareMsg.encryptionKeyHex,
        hex_chain_code: reshareMsg.hexChainCode,
        local_party_id: reshareMsg.serviceName,
        old_parties: reshareMsg.oldParties,
        email: "", // Not provided by extension, using empty string
        plugin_id: pluginId, // Use the pluginId parameter passed to function
      };

      console.log("Transformed payload for backend:", backendPayload);

      try {
        await MarketplaceService.reshareVault(backendPayload);
        publish("onToast", {
          message: "Reshare session started",
          type: "success",
        });
      } catch (err) {
        console.error("Failed to call reshare endpoint", err);
        publish("onToast", {
          message: "Failed to start reshare",
          type: "error",
        });
      }

      return backendPayload;
    } catch (error) {
      console.error("Failed to process reshare session", error);
      publish("onToast", {
        message:
          error instanceof Error
            ? error.message
            : "Failed to process reshare session",
        type: "error",
      });
      throw new Error("Failed to process reshare session");
    }
  },
};

export default VulticonnectWalletService;
