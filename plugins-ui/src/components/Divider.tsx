import { Stack } from "components/Stack";
import { useTheme } from "styled-components";

export const Divider = () => {
  const colors = useTheme();

  return (
    <Stack
      as="span"
      $style={{ backgroundColor: colors.borderLight.toHex(), height: "1px" }}
    />
  );
};
