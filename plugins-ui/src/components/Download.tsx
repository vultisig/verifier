import { Stack } from "components/Stack";
import { CircleArrowDownIcon } from "icons/CircleArrowDownIcon";
import { FC } from "react";
import { useTheme } from "styled-components";
import { toNumeralFormat } from "utils/functions";

type DownloadProps = {
  value: number;
};

export const Download: FC<DownloadProps> = ({ value }) => {
  const colors = useTheme();

  return (
    <Stack $style={{ alignItems: "center", gap: "2px" }}>
      <Stack
        as={CircleArrowDownIcon}
        $style={{ color: colors.textTertiary.toHex(), fontSize: "16px" }}
      />
      <Stack
        as="span"
        $style={{
          color: colors.textTertiary.toHex(),
          fontSize: "12px",
          fontWeight: "500",
          lineHeight: "16px",
        }}
      >
        {toNumeralFormat(value)}
      </Stack>
    </Stack>
  );
};
