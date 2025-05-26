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
          const accounts =  await VulticonnectWalletService.connectToVultiConnect();
          console.log("accounts", accounts);

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

    // Create a new random wallet
    const wallet = ethers.Wallet.createRandom();
    const address = wallet.address;
    const publicKey = address;
    
    console.log("address", address);
    console.log("publicKey", publicKey);

    const chainCodeHex = "0x" + "00".repeat(32);
    console.log("chainCodeHex", chainCodeHex);

    // const vaults = await VulticonnectWalletService.getVaults();
    // console.log("vaults", vaults);

    // const publicKey = vaults[0].publicKeyEcdsa;
    // if (publicKey) {
    //   localStorage.setItem("publicKey", publicKey);
    // }
    // console.log("publicKey", publicKey);

    // const chainCodeHex = vaults[0].hexChainCode;
    // console.log("chainCodeHex", chainCodeHex);

    // const derivePath = derivePathMap[chain as keyof typeof derivePathMap];
    // console.log("derivePath", derivePath);

    const hexMessage = generateHexMessage(publicKey);
    console.log("hexMessage", hexMessage);

    // const signature = await VulticonnectWalletService.signCustomMessage(
    //   hexMessage,
    //   walletAddress
    // );
    // console.log("signature", signature);

    // const messageBytes = hexToBytes(hexMessage);
    const signature = await wallet.signMessage( hexMessage);
    // let { r, s, v } = ethers.splitSignature(signature);

    console.log("signature", signature);

    if (signature && typeof signature === "string") {
      const token = await MarketplaceService.getAuthToken(
        hexMessage,
        signature.toString(),
        publicKey,
        chainCodeHex
      );
      console.log("token", token);
      localStorage.setItem("authToken", token);
      console.log("token", token);
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
