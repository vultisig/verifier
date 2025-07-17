import { theme, ThemeConfig } from "antd";
import { styledThemes } from "utils/constants/styled";
import { Theme } from "utils/constants/theme";

export const antdThemes: Record<Theme, ThemeConfig> = {
  default: {
    algorithm: theme.darkAlgorithm,
    token: {
      borderRadius: 10,
      colorBgBase: styledThemes.default.backgroundPrimary,
      colorBgContainer: styledThemes.default.backgroundSecondary,
      colorBgElevated: styledThemes.default.backgroundSecondary,
      colorBorder: styledThemes.default.borderLight,
      colorSplit: styledThemes.default.borderNormal,
      colorBorderSecondary: styledThemes.default.borderNormal,
      colorPrimary: styledThemes.default.buttonPrimary,
      colorWarning: styledThemes.default.alertWarning,
    },
    components: {
      Button: {
        borderRadius: 22,
        borderRadiusLG: 28,
        borderRadiusSM: 18,
        colorBorder: "transparent",
        controlHeight: 44,
        controlHeightSM: 36,
        controlHeightLG: 56,
        dangerShadow: "none",
        defaultShadow: "none",
        primaryShadow: "none",
      },
      Dropdown: {
        fontSize: 16,
        fontSizeSM: 20,
        paddingBlock: 8,
      },
      Input: {
        activeBorderColor: styledThemes.default.borderNormal,
        activeShadow: "none",
        hoverBorderColor: styledThemes.default.borderNormal,
      },
      Layout: {
        headerBg: styledThemes.default.backgroundSecondary,
        headerPadding: 0,
      },
      Select: {
        activeBorderColor: styledThemes.default.borderNormal,
        activeOutlineColor: "transparent",
        hoverBorderColor: styledThemes.default.borderNormal,
        optionLineHeight: 2,
        optionPadding: "4px 12px",
      },
      Table: {
        borderColor: styledThemes.default.borderLight,
        headerBg: styledThemes.default.backgroundTertiary,
        headerSplitColor: styledThemes.default.borderNormal,
      },
    },
  },
} as const;
