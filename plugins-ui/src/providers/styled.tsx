import { FC, ReactNode } from "react";
import { ThemeProvider } from "styled-components";
import { themes } from "utils/constants/styled";
import { Theme } from "utils/constants/theme";

type StyledProviderProps = {
  children?: ReactNode;
  theme: Theme;
};

export const StyledProvider: FC<StyledProviderProps> = ({
  children,
  theme,
}) => {
  return <ThemeProvider theme={themes[theme]}>{children}</ThemeProvider>;
};
