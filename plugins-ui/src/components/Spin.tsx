import { Spin as DefaultSpin, SpinProps } from "antd";
import { FC } from "react";
import styled from "styled-components";

const StyledSpin = styled(DefaultSpin)<SpinProps>`
  color: currentColor;
  
  .ant-spin-dot-holder {
    color: currentColor;
  }
}`;

export const Spin: FC<SpinProps> = (props) => <StyledSpin {...props} />;
