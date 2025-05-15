import Button from "@/modules/core/components/ui/button/Button";
import VulticonnectWalletService from "./vulticonnectWalletService";
import { useState } from "react";
import PolicyService from "@/modules/policy/services/policyService";
import { derivePathMap, generateHexMessage } from "./wallet.utils";
import { ethers } from "ethers";

const Wallet = () => {
  let chain = localStorage.getItem("chain") as string;

  if (!chain) {
    localStorage.setItem("chain", "ethereum");
    chain = localStorage.getItem("chain") as string;
  }

  const [connectedWallet, setConnectedWallet] = useState(false);
  const [walletAddress, setWalletAddress] = useState("");

  // connect to wallet
  const connectWallet = async (chain: string) => {
    console.log("connectWallet", chain);

    switch (chain) {
      // add more switch cases as more chains are supported
      case "ethereum": {
        const accounts =   await VulticonnectWalletService.connectToVultiConnect();
        console.log("accounts", accounts);
        if (accounts.length && accounts[0]) {
          setConnectedWallet(true);
          setWalletAddress(accounts[0]);
        }
        break;
      }

      default:
        alert(`Chain ${chain} is currently not supported.`); // toast
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
      const token = await PolicyService.verifyWalletAndAuth(
        {
          publicKey,
          chainCodeHex,
          message: hexMessage,
          signature: signature.toString()
        }
      );
      console.log("token", token);
      localStorage.setItem("authToken", token);
      console.log("token", token);
    }
  };

/**
 * Converts a hex string to a Uint8Array
 * @param hexString - The hex string to convert (with or without 0x prefix)
 * @returns Uint8Array of the hex string
 */
const hexToBytes = (hexString: string): Uint8Array => {
  const hex = hexString.startsWith('0x') ? hexString.slice(2) : hexString;
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.slice(i, i + 2), 16);
  }
  return bytes;
};


  return (
    <> 
      <Button
        size="medium"
        styleType="primary"
        type="button"
        onClick={() => connectWallet(chain)}
    >
      {connectedWallet ? "Connected" : "Connect Wallet"}
      </Button>

      {connectedWallet && <div>{walletAddress}</div>}
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
