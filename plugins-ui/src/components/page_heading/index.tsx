import { Typography } from "antd";
import { FC } from "react";
import styled from "styled-components";
import { Stack } from "styles/Stack";

interface PageHeadingProps {
  description?: string;
  title: string;
}

export const PageHeading: FC<PageHeadingProps> = ({ description, title }) => (
  <Stack $flexDirection="column" $gap="14px">
    <StyledTitle>{title}</StyledTitle>
    {description ? <StyledText>{description}</StyledText> : <></>}
  </Stack>
);

const StyledText = styled(Typography.Text)`
  &.ant-typography {
    color: ${({ theme }) => theme.textExtraLight};
    font-size: 14px;
    font-weight: 400;
    line-height: 20px;
  }
`;

const StyledTitle = styled(Typography.Title)`
  &.ant-typography {
    font-size: 40px;
    font-weight: 500;
    line-height: 42px;
    margin: 0;
  }
`;
