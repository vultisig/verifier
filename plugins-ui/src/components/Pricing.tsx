import { Stack } from "components/Stack";
import { FC } from "react";
import { useTheme } from "styled-components";
import { Plugin, PluginPricing } from "utils/types";

type PricingProps = Pick<Plugin, "pricing"> & {
  center?: boolean;
};

export const Pricing: FC<PricingProps> = ({ center, pricing }) => {
  const colors = useTheme();

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
        alignItems: center ? "center" : "normal",
        color: colors.textSecondary.toHex(),
        flexDirection: "column",
        flexGrow: "1",
        fontWeight: "500",
      }}
    >
      {pricing.length ? (
        pricing.map((price, index) => (
          <Stack as="span" key={index}>
            {pricingText(price)}
          </Stack>
        ))
      ) : (
        <Stack as="span">This plugin is free</Stack>
      )}
    </Stack>
  );
};
