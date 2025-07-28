import { DatePicker as DefaultDatePicker, DatePickerProps } from "antd";
import { FC } from "react";
import styled from "styled-components";

const StyledDatePicker = styled(DefaultDatePicker)<DatePickerProps>`
  display: flex;
  height: 44px;
`;

export const DatePicker: FC<DatePickerProps> = (props) => (
  <StyledDatePicker {...props} />
);
