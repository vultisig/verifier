import React, { createContext, useContext, useState } from "react";
import VulticonnectWalletService from "./vulticonnectWalletService";
import { ethers } from "ethers";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import { publish } from "@/utils/eventBus";

interface WalletProviderProps {
  children: React.ReactNode;
}

interface InitialState {
  address?: string;
  chain: string;
  isConnected: boolean;
  token?: string;
}

export const WalletProvider: React.FC<WalletProviderProps> = ({ children }) => {
  const initialState: InitialState = {
    isConnected: false,
    chain: localStorage.getItem("chain") || "ethereum",
  };
  const [state, setState] = useState(initialState);

  const disconnect = () => {
    setState(initialState);
  };

  const signMessage = async (address: string): Promise<boolean> => {
    try {
      const vault = await VulticonnectWalletService.getVault();

      if (!vault) {
        throw new Error("No vaults found");
      }

      const publicKey = vault.publicKeyEcdsa;
      const chainCodeHex = vault.hexChainCode;

      if (!publicKey || !chainCodeHex) {
        throw new Error("Missing required vault data");
      }

      const nonce = ethers.hexlify(ethers.randomBytes(16));
      const expiryTime = new Date(Date.now() + 15 * 60 * 1000).toISOString();

      const signingMessage = JSON.stringify({
        message: "Sign into Vultisig App Store",
        nonce: nonce,
        expiresAt: expiryTime,
        address,
      });

      const signature = await VulticonnectWalletService.signCustomMessage(
        signingMessage,
        address
      );

      const token = await MarketplaceService.getAuthToken(
        signingMessage,
        signature,
        publicKey,
        chainCodeHex
      );

      setState((prevState) => ({ ...prevState, token }));

      publish("onToast", {
        message: "Successfully authenticated!",
        type: "success",
      });

      return true;
    } catch (error) {
      publish("onToast", {
        message:
          error instanceof Error ? error.message : "Authentication failed",
        type: "error",
      });

      return false;
    }
  };

  const connectWallet = async (chain: string) => {
    switch (chain) {
      case "ethereum": {
        try {
          const address =
            await VulticonnectWalletService.connectToVultiConnect();

          setState((prevState) => ({
            ...prevState,
            address,
            isConnected: true,
          }));
        } catch (error) {
          if (error instanceof Error) {
            console.error("Failed to connect wallet:", error.message, error);

            publish("onToast", {
              message: "Wallet connection failed!",
              type: "error",
            });
          }
        }

        break;
      }
      default:
        publish("onToast", {
          message: `Chain ${chain} is currently not supported.`,
          type: "error",
        });

        break;
    }
  };

  return (
    <WalletContext.Provider
      value={{
        ...state,
        connectWallet,
        disconnect,
        signMessage,
      }}
    >
      {children}
    </WalletContext.Provider>
  );
};

interface WalletContextType extends InitialState {
  connectWallet: (chain: string) => Promise<void>;
  disconnect: () => void;
  signMessage: (address: string) => Promise<boolean>;
}

const WalletContext = createContext<WalletContextType | undefined>(undefined);

export const useWallet = () => {
  const context = useContext(WalletContext);

  if (!context) {
    throw new Error("useWallet must be used within a WalletProvider");
  }

  return context;
};
