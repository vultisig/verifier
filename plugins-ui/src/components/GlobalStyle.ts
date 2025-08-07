import { createGlobalStyle } from "styled-components";

export const GlobalStyle = createGlobalStyle`
  body {
    background-color: ${({ theme }) => theme.bgPrimary.toHex()};
    color: ${({ theme }) => theme.textPrimary.toHex()};
    min-width: 768px;
  }
`;
