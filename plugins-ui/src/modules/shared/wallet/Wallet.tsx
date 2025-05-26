import Button from "@/modules/core/components/ui/button/Button";
import VulticonnectWalletService from "./vulticonnectWalletService";
import { useEffect, useState } from "react";
import {
  generateHexMessage,
  setLocalStorageAuthToken,
} from "./wallet.utils";
import { publish } from "@/utils/eventBus";
import { ethers } from "ethers";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";

const Wallet = () => {
  let chain = localStorage.getItem("chain") as string;

  if (!chain) {
    localStorage.setItem("chain", "ethereum");
    chain = localStorage.getItem("chain") as string;
  }
  const [authToken, setAuthToken] = useState(
    localStorage.getItem("authToken") || ""
  );

  const [connectedWallet, setConnectedWallet] = useState(false);
  const [walletAddress, setWalletAddress] = useState("");

  const connectWallet = async (chain: string) => {
    switch (chain) {
      // add more switch cases as more chains are supported
      case "ethereum": {
        try {
 
          const accounts = await VulticonnectWalletService.connectToVultiConnect();
 
          if (accounts.length && accounts[0]) {
            setConnectedWallet(true);
            setWalletAddress(accounts[0]);
          }

          break;
        } catch (error) {
          if (error instanceof Error) {
            console.error("Failed to connect wallet:", error.message, error);
            publish("onToast", {
              message: "Wallet connection failed!",
              type: "error",
            });
          }
        }
      }

      default:
        publish("onToast", {
          message: `Chain ${chain} is currently not supported.`,
          type: "error",
        });
        break;
    }
  };

  // sign message
  const signMessage = async () => {
    try {
      // 1. Get vaults from VultiConnect
      const vaults = await VulticonnectWalletService.getVaults();
      if (!vaults || vaults.length === 0) {
        throw new Error("No vaults found");
      }

      // 2. Get required data from first vault
      const publicKey = vaults[0].publicKeyEcdsa;
      const chainCodeHex = vaults[0].hexChainCode;

      if (!publicKey || !chainCodeHex) {
        throw new Error("Missing required vault data");
      }

      // 3. Store public key in localStorage
      localStorage.setItem("publicKey", publicKey);

      // 4. Generate nonce and expiry timestamp
      const nonce = ethers.hexlify(ethers.randomBytes(16));
      const expiryTime = new Date(Date.now() + 15 * 60 * 1000).toISOString(); // 15 minutes from now

      // 5. Generate hex message for signing
      const signingMessage = `Sign into Vultisig App Store
                                  Nonce: ${nonce}
                                  Expires At: ${expiryTime}
                                  Address: ${walletAddress}`;

      // 6. Sign the message using VultiConnect
      const signature = await VulticonnectWalletService.signCustomMessage(
        signingMessage,
        walletAddress
      );

      // 7. Call auth endpoint
      const token = await MarketplaceService.getAuthToken(
        signingMessage,
        signature,
        publicKey,
        chainCodeHex
      );

      // 8. Store token and update state
      localStorage.setItem("authToken", token);
      setAuthToken(token);
      setConnectedWallet(true);

      publish("onToast", {
        message: "Successfully authenticated!",
        type: "success",
      });
    } catch (error) {
      console.error("Authentication failed:", error);
      publish("onToast", {
        message: error instanceof Error ? error.message : "Authentication failed",
        type: "error",
      });
    }
  };


  useEffect(() => {
    const handleStorageChange = () => {
      const hasToken = !!localStorage.getItem("authToken");
      setConnectedWallet(hasToken);
    };

    // Listen for storage changes
    window.addEventListener("storage", handleStorageChange);

    return () => {
      window.removeEventListener("storage", handleStorageChange);
    };
  }, [authToken]);

  return (
    <>
      <Button
        size="medium"
        styleType="primary"
        type="button"
        onClick={() => connectWallet(chain)}
      >
        {connectedWallet ? "Connected " + walletAddress.slice(0, 6) + "..." + walletAddress.slice(-4) : "Connect Wallet"}
      </Button>

      {connectedWallet && (
        <Button
          size="medium"
          styleType="primary"
          type="button"
          onClick={() => signMessage()}
        >
          Sign Message
        </Button>
      )}
    </>
  );
};

export default Wallet;
