import { Stack } from "components/Stack";
import { FC } from "react";
import { ThemeColorKeys } from "utils/constants/styled";

export const Tag: FC<{
  color?: ThemeColorKeys;
  text: string;
}> = ({ color = "success", text }) => (
  <Stack
    as="span"
    $style={{
      backgroundColor: color,
      borderRadius: "6px",
      color: "buttonText",
      lineHeight: "20px",
      padding: "0 8px",
    }}
  >
    {text}
  </Stack>
);
