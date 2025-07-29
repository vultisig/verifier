import { Input, InputProps } from "antd";
import { Stack } from "components/Stack";
import { SearchIcon } from "icons/SearchIcon";
import { FC } from "react";

export const SearchInput: FC<InputProps> = (props) => (
  <Stack $style={{ position: "relative", width: "100%" }}>
    <Stack
      as={Input}
      {...props}
      $style={{ height: "44px", paddingLeft: "40px" }}
    />
    <Stack
      as={SearchIcon}
      $style={{
        color: "textTertiary",
        fontSize: "24px",
        left: "8px",
        position: "absolute",
        transform: "translateY(-50%)",
        top: "50%",
      }}
    />
  </Stack>
);
