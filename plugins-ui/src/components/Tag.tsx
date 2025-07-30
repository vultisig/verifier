import { Stack } from "components/Stack";
import { FC } from "react";
import { useTheme } from "styled-components";
import { ThemeColorKeys } from "utils/constants/styled";

export const Tag: FC<{
  color?: ThemeColorKeys;
  text: string;
}> = ({ color = "success", text }) => {
  const colors = useTheme();

  return (
    <Stack
      as="span"
      $style={{
        backgroundColor: colors[color].toHex(),
        borderRadius: "6px",
        color: colors.buttonText.toHex(),
        lineHeight: "20px",
        padding: "0 8px",
      }}
    >
      {text}
    </Stack>
  );
};
