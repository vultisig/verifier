import { Input as DefaultInput, InputProps } from "antd";
import { FC } from "react";
import styled from "styled-components";

const StyledInput = styled(DefaultInput)<InputProps>`
  height: 44px;
`;

export const Input: FC<InputProps> = (props) => <StyledInput {...props} />;
