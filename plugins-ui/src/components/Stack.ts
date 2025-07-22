import styled, { css } from "styled-components";
import { ThemeColorKeys } from "utils/constants/styled";
import { CSSColorProperties } from "utils/constants/styles";
import { cssPropertiesToString } from "utils/functions";
import { CSSProperties } from "utils/types";

export const Stack = styled.div<StackProps>`
  ${({ $after, $before, $hover, $style, theme }) => css`
    ${$style &&
    cssPropertiesToString(
      { ...$style, display: $style.display || "flex" },
      theme
    )}
    ${$before &&
    `
      &::before {
        content: "";
        ${cssPropertiesToString($before, theme)}
      }
    `}
    ${$after &&
    `
      &::after {
        content: "";
        ${cssPropertiesToString($after, theme)}
      }
    `}
    ${$hover &&
    `
      transition: all .2s;

      &:hover {
        ${cssPropertiesToString($hover, theme)}
      }
    `}
  `}
`;

export type StackCSSProperties = Omit<CSSProperties, CSSColorProperties> & {
  backgroundColor?: ThemeColorKeys;
  borderColor?: ThemeColorKeys;
  color?: ThemeColorKeys;
  fill?: ThemeColorKeys;
};

export type StackProps = {
  $after?: StackCSSProperties;
  $before?: StackCSSProperties;
  $hover?: StackCSSProperties;
  $style?: StackCSSProperties;
};
