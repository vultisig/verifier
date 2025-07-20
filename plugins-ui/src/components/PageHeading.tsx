import { Typography } from "antd";
import { Stack } from "components/Stack";
import { FC } from "react";

interface PageHeadingProps {
  description?: string;
  title: string;
}

export const PageHeading: FC<PageHeadingProps> = ({ description, title }) => (
  <Stack $style={{ flexDirection: "column", gap: "14px" }}>
    <Stack
      as={Typography.Title}
      $style={{
        fontSize: "40px",
        fontWeight: "500",
        lineHeight: "42px",
        margin: "0",
      }}
    >
      {title}
    </Stack>
    {description ? (
      <Stack
        as={Typography.Text}
        $style={{
          color: "textExtraLight",
          fontSize: "14px",
          fontWeight: "400",
          lineHeight: "20px",
        }}
      >
        {description}
      </Stack>
    ) : (
      <></>
    )}
  </Stack>
);
