import { Stack } from "components/Stack";
import { FC } from "react";
import { ThemeColorKeys } from "utils/constants/styled";

export const Tag: FC<{
  color?: ThemeColorKeys;
  text: string;
}> = ({ color = "alertSuccess", text }) => (
  <Stack
    as="span"
    $style={{
      backgroundColor: color,
      borderRadius: "6px",
      color: "textPrimary",
      lineHeight: "20px",
      padding: "0 8px",
    }}
  >
    {text}
  </Stack>
);
