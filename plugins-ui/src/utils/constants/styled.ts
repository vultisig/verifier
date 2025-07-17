import { DefaultTheme } from "styled-components";
import { Theme } from "utils/constants/theme";

export const styledThemes: Record<Theme, DefaultTheme> = {
  default: {
    alertError: "#ff5c5c",
    alertInfo: "#5ca7ff",
    alertSuccess: "#13c89d",
    alertWarning: "#ffc25c",
    backgroundPrimary: "#02122b",
    backgroundSecondary: "#061b3a",
    backgroundTertiary: "#11284a",
    borderLight: "#12284a",
    borderNormal: "#1b3f73",
    buttonBackgroundDisabled: "#0b1a3a",
    buttonPrimary: "#2155df",
    buttonPrimaryHover: "#1e6ad1",
    buttonSecondary: "#11284a",
    buttonSecondaryHover: "#3b6a91",
    buttonTextDisabled: "#718096",
    neutralOne: "",
    neutralSix: "",
    neutralSeven: "",
    primaryAccentThree: "",
    primaryAccentFour: "",
    textExtraLight: "#8295ae",
    textLight: "#c9d6e8",
    textPrimary: "#f0f4fc",
    transparent: "transparent",
  },
} as const;

export type ThemeColorKeys = keyof DefaultTheme;
