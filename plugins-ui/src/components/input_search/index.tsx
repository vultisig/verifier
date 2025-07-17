import { Input, InputProps } from "antd";
import { SearchIcon } from "icons/SearchIcon";
import { FC } from "react";
import { FormInput } from "styles/FormInput";
import { Stack } from "styles/Stack";

export const SearchInput: FC<InputProps> = (props) => (
  <Stack $position="relative" $fullWidth>
    <FormInput as={Input} {...props} $paddingLeft="40px" />
    <Stack
      as={SearchIcon}
      $color="textExtraLight"
      $fontSize="24px"
      $left="8px"
      $position="absolute"
      $transform="translateY(-50%)"
      $top="50%"
    />
  </Stack>
);
