import { Stack } from "components/Stack";
import { FC } from "react";
import { Plugin, PluginPricing } from "utils/types";

export const Pricing: FC<Pick<Plugin, "pricing">> = ({ pricing }) => {
  const pricingText = ({ amount, frequency, type }: PluginPricing) => {
    switch (type) {
      case "once":
        return `$${amount / 1e6} one time installation fee`;
      case "recurring":
        return `$${amount / 1e6} ${frequency} recurring fee`;
      case "per-tx":
        return `$${amount / 1e6} per transaction fee`;
      default:
        return "Unknown pricing type";
    }
  };

  return (
    <Stack
      as="span"
      $style={{
        color: "textTertiary",
        flexDirection: "column",
        fontWeight: "500",
      }}
    >
      {pricing.length ? (
        pricing.map((price, index) => (
          <Stack key={index}>{pricingText(price)}</Stack>
        ))
      ) : (
        <Stack>This plugin is free</Stack>
      )}
    </Stack>
  );
};
