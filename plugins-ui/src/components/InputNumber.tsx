import { InputNumber as DefaultInputNumber, InputNumberProps } from "antd";
import { FC } from "react";
import styled from "styled-components";

const StyledInputNumber = styled(DefaultInputNumber)<InputNumberProps>`
  width: 100%;

  .ant-input-number-input {
    height: 42px;
  }
`;

export const InputNumber: FC<InputNumberProps> = (props) => (
  <StyledInputNumber {...props} />
);
