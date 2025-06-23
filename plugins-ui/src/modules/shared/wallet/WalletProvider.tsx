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
    const address = await VulticonnectWalletService.connect();

    if (address) await signMessage(address);
  };

  const checkConnection = async () => {
    const address = await VulticonnectWalletService.getAccount();

    if (address) await signMessage(address);
  };

  const disconnect = async () => {
    try {
      await VulticonnectWalletService.disconnect();

      if (vault) deleteToken(vault.publicKeyEcdsa);

      setState(initialState);
    } catch {}
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

    setTimeout(() => checkConnection(), 100);

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
