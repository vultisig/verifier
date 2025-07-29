import { Stack } from "components/Stack";
import { StarIcon } from "icons/StarIcon";
import { FC } from "react";

export const Rate: FC<{ count: number; value: number }> = ({
  count,
  value,
}) => (
  <Stack $style={{ gap: "6px" }}>
    <Stack
      as={StarIcon}
      $style={{ color: "warning", fill: "warning", fontSize: "16px" }}
    />
    <Stack as="span" $style={{ gap: "4px" }}>
      <Stack as="span" $style={{ fontWeight: "500" }}>
        {value}
      </Stack>
      <Stack as="span" $style={{ color: "textTertiary", fontWeight: "500" }}>
        {`(${count})`}
      </Stack>
    </Stack>
  </Stack>
);
