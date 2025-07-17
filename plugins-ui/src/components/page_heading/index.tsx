import { Typography } from "antd";
import { FC } from "react";
import { Stack } from "styles/Stack";

interface PageHeadingProps {
  description?: string;
  title: string;
}

export const PageHeading: FC<PageHeadingProps> = ({ description, title }) => (
  <Stack $flexDirection="column" $gap="14px">
    <Stack
      as={Typography.Title}
      $fontSize="40px"
      $fontWeight="500"
      $lineHeight="42px"
      $margin="0"
    >
      {title}
    </Stack>
    {description ? (
      <Stack
        as={Typography.Text}
        $color="textExtraLight"
        $fontSize="14px"
        $fontWeight="400"
        $lineHeight="20px"
      >
        {description}
      </Stack>
    ) : (
      <></>
    )}
  </Stack>
);
