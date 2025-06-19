import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
} from "react";
import VulticonnectWalletService from "./vulticonnectWalletService";
import { ethers } from "ethers";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import { publish } from "@/utils/eventBus";

interface WalletState {
  isConnected: boolean;
  walletAddress: string | null;
  authToken: string | null;
  chain: string;
}

interface WalletContextType extends WalletState {
  connectWallet: (chain: string) => Promise<void>;
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
  const [walletState, setWalletState] = useState<WalletState>({
    isConnected: false,
    walletAddress: null,
    authToken: localStorage.getItem("authToken") || null,
    chain: localStorage.getItem("chain") || "ethereum",
  });

  // Initialize chain in localStorage if not set
  useEffect(() => {
    if (!localStorage.getItem("chain")) {
      localStorage.setItem("chain", "ethereum");
    }
  }, []);

  // Check if VultiConnect provider is available
  const isVultiConnectAvailable = () => {
    return !!window.vultisig?.ethereum;
  };

  // Define disconnect function early to avoid circular dependency
  const disconnect = useCallback(() => {
    localStorage.removeItem("authToken");
    localStorage.removeItem("publicKey");
    setWalletState((prev) => ({
      ...prev,
      isConnected: false,
      walletAddress: null,
      authToken: null,
    }));
  }, []);

  // Check for existing wallet connection on mount
  useEffect(() => {
    const checkExistingConnection = async () => {
      // Only check if we have an auth token and VultiConnect is available
      if (!walletState.authToken || !isVultiConnectAvailable()) {
        return;
      }

      try {
        const accounts =
          await VulticonnectWalletService.getConnectedEthAccounts();
        if (accounts && accounts.length > 0) {
          setWalletState((prev) => ({
            ...prev,
            isConnected: true,
            walletAddress: accounts[0],
          }));
        }
      } catch (error) {
        // Silently handle error - extension may not be loaded yet or no connection exists
        console.debug("No existing wallet connection found");
      }
    };

    // Add a small delay to allow extension to load
    const timeoutId = setTimeout(checkExistingConnection, 100);

    return () => clearTimeout(timeoutId);
  }, [walletState.authToken]);

  // Listen for account changes from wallet extension
  useEffect(() => {
    const handleAccountsChanged = (accounts: string[]) => {
      if (accounts.length === 0) {
        // Wallet disconnected - use inline disconnect logic
        localStorage.removeItem("authToken");
        localStorage.removeItem("publicKey");
        setWalletState((prev) => ({
          ...prev,
          isConnected: false,
          walletAddress: null,
          authToken: null,
        }));
      } else if (accounts[0] !== walletState.walletAddress) {
        // Account switched
        setWalletState((prev) => ({
          ...prev,
          walletAddress: accounts[0],
        }));
      }
    };

    if (isVultiConnectAvailable()) {
      window.vultisig.ethereum.on?.("accountsChanged", handleAccountsChanged);
    }

    return () => {
      if (isVultiConnectAvailable()) {
        window.vultisig.ethereum.off?.(
          "accountsChanged",
          handleAccountsChanged
        );
      }
    };
  }, [walletState.walletAddress]);

  const signMessage = async (walletAddress: string): Promise<boolean> => {
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

      if (
        walletState.authToken &&
        publicKey === localStorage.getItem("publicKey")
      ) {
        return true;
      }

      localStorage.setItem("publicKey", publicKey);

      const nonce = ethers.hexlify(ethers.randomBytes(16));
      const expiryTime = new Date(Date.now() + 15 * 60 * 1000).toISOString();

      const signingMessage = JSON.stringify({
        message: "Sign into Vultisig App Store",
        nonce: nonce,
        expiresAt: expiryTime,
        address: walletAddress,
      });

      const signature = await VulticonnectWalletService.signCustomMessage(
        signingMessage,
        walletAddress
      );

      const token = await MarketplaceService.getAuthToken(
        signingMessage,
        signature,
        publicKey,
        chainCodeHex
      );

      localStorage.setItem("authToken", token);
      setWalletState((prev) => ({
        ...prev,
        authToken: token,
        isConnected: true,
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

  const connectWallet = useCallback(async (chain: string) => {
    switch (chain) {
      case "ethereum": {
        try {
          const accounts =
            await VulticonnectWalletService.connectToVultiConnect();

          const isAuthenticated = await signMessage(accounts[0]);
          if (!isAuthenticated) {
            publish("onToast", {
              message: "Authentication failed!",
              type: "error",
            });
            return;
          }

          if (accounts.length && accounts[0]) {
            setWalletState((prev) => ({
              ...prev,
              isConnected: true,
              walletAddress: accounts[0],
            }));
          }
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
  }, []);

  // Listen for storage changes (for auth token updates from other tabs)
  useEffect(() => {
    const handleStorageChange = (event: StorageEvent) => {
      if (event.key === "authToken") {
        const newToken = event.newValue;
        setWalletState((prev) => ({
          ...prev,
          authToken: newToken,
          isConnected: !!newToken && !!prev.walletAddress,
        }));
      }
    };

    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, []);

  return (
    <WalletContext.Provider
      value={{
        ...walletState,
        connectWallet,
        disconnect,
      }}
    >
      {children}
    </WalletContext.Provider>
  );
};
