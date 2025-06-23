import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import VulticonnectWalletService from "./vulticonnectWalletService";
import { ethers } from "ethers";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import { publish } from "@/utils/eventBus";
import { createToken, deleteToken, selectToken } from "@/storage/token";
import {
  deleteCurrentVaultId,
  setCurrentVaultId,
} from "@/storage/currentVaultId";

interface Vault {
  hexChainCode: string;
  name: string;
  publicKeyEcdsa: string;
  publicKeyEddsa: string;
  uid: string;
}

interface InitialState {
  address?: string;
  isConnected?: boolean;
  token?: string;
  vault?: Vault;
}

interface WalletContextType extends InitialState {
  connect: () => Promise<void>;
  disconnect: () => void;
}

const WalletContext = createContext<WalletContextType | undefined>(undefined);

export const useWallet = () => {
  const context = useContext(WalletContext);
  if (!context) {
    throw new Error("useWallet must be used within a WalletProvider");
  }
  return context;
};

interface WalletProviderProps {
  children: React.ReactNode;
}

export const WalletProvider: React.FC<WalletProviderProps> = ({ children }) => {
  const initialState: InitialState = {};
  const [state, setState] = useState(initialState);
  const { address, vault } = state;

  const connect = async () => {
    try {
      const address = await VulticonnectWalletService.connectToVultiConnect();

      await signMessage(address);
    } catch (error) {
      if (error instanceof Error) {
        console.error("Failed to connect wallet:", error.message, error);
        publish("onToast", {
          message: "Wallet connection failed!",
          type: "error",
        });
      }
    }
  };

  const disconnect = () => {
    if (vault) {
      deleteToken(vault.publicKeyEcdsa);
      deleteCurrentVaultId();
    }

    setState(initialState);
  };

  const handleChangeWallet = useCallback(
    async ([address]: string[]) => {
      if (!address) {
        disconnect();
      } else if (address !== state.address) {
        await signMessage(address);
      }
    },
    [address]
  );

  const signMessage = async (address: string) => {
    try {
      const vault: Vault = await VulticonnectWalletService.getVault();
      const { hexChainCode, publicKeyEcdsa } = vault;
      const token = selectToken(publicKeyEcdsa);

      if (token) {
        setState((prevState) => ({
          ...prevState,
          address,
          isConnected: true,
          token,
          vault,
        }));

        return true;
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

      const newToken = await MarketplaceService.getAuthToken(
        signingMessage,
        signature,
        publicKeyEcdsa,
        hexChainCode
      );

      createToken(publicKeyEcdsa, newToken);
      setCurrentVaultId(publicKeyEcdsa);
      setState((prevState) => ({
        ...prevState,
        address,
        isConnected: true,
        token: newToken,
        vault,
      }));

      publish("onToast", {
        message: "Successfully authenticated!",
        type: "success",
      });

      return true;
    } catch (error) {
      console.error("Authentication failed:", error);
      publish("onToast", {
        message:
          error instanceof Error ? error.message : "Authentication failed",
        type: "error",
      });
      return false;
    }
  };

  // Listen for wallet changes from extension
  useEffect(() => {
    window.vultisig?.ethereum?.on?.("accountsChanged", handleChangeWallet);

    return () => {
      window.vultisig?.ethereum?.off?.("accountsChanged", handleChangeWallet);
    };
  }, []);

  return (
    <WalletContext.Provider
      value={{
        ...state,
        connect,
        disconnect,
      }}
    >
      {children}
    </WalletContext.Provider>
  );
};
