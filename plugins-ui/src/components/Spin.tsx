import { Spin as DefaultSpin, SpinProps } from "antd";
import { FC } from "react";
import styled from "styled-components";

const StyledSpin = styled(DefaultSpin)<SpinProps>`
  align-items: center;
  color: currentColor;
  display: flex;
  flex-grow: 1;
  justify-content: center;

  .ant-spin-dot-holder {
    color: currentColor;
  }
`;

export const Spin: FC<SpinProps> = (props) => <StyledSpin {...props} />;
