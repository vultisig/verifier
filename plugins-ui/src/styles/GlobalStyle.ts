import { createGlobalStyle } from "styled-components";

export const GlobalStyle = createGlobalStyle`
  #root {
    height: 100%;
  }

  body {
    background-color: ${({ theme }) => theme.backgroundPrimary};
    color: ${({ theme }) => theme.textPrimary};
    min-width: 360px;
  }

  a {
    text-decoration: none;
  }
`;
