import { supportedTokens } from "../data/tokens";

export type TokenNameProps = {
  data: string;
};

const TokenName = ({ data }: TokenNameProps) => {
  const tokenName = supportedTokens[data]?.name || "Unknown";
  return <>{tokenName}</>;
};

export default TokenName;
