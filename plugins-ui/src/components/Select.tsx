import { Select as DefaultSelect, SelectProps } from "antd";
import { FC } from "react";
import styled from "styled-components";

const StyledSelect = styled(DefaultSelect)<SelectProps>`
  display: flex;
  height: 40px;
  width: 100%;
`;

export const Select: FC<SelectProps> = (props) => <StyledSelect {...props} />;
