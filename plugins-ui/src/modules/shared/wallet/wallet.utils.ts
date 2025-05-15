type ChainType = "ethereum"; // add more chains here
export const isSupportedChainType = (value: string): value is ChainType =>
  value === "ethereum"; // add more chains here

export const toHex = (str: string): string => {
  return (
    "0x" +
    Array.from(str)
      .map((char) => char.charCodeAt(0).toString(16).padStart(2, "0"))
      .join("")
  );
};

/**
 * Generates a hex message for wallet verification.
 * Matches the server's GenerateHexMessage implementation exactly.
 * @param {string} publicKey - The public key to include in the message.
 * @returns {string} A hex-encoded message in format "0x{publicKey}1"
 */
export const generateHexMessage = (publicKey: string): string => {
  // Remove 0x prefix if present
  const publicKeyTrimmed = publicKey.startsWith('0x') ? publicKey.slice(2) : publicKey;
  
  // Append "01" to the public key, as done in the server-side implementation
  const messageToSign = publicKeyTrimmed + "01";
  
  // Add 0x prefix
  return "0x" + messageToSign;
};

export const derivePathMap = {
  ethereum: "m/44'/60'/0'/0/0",
  thor: "thor derivation path",
};
