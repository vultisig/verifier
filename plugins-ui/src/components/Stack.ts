import styled, { css } from "styled-components";
import { cssPropertiesToString } from "utils/functions";
import { CSSProperties } from "utils/types";

export const Stack = styled.div<StackProps>`
  ${({ $after, $before, $hover, $style }) => css`
    ${$style &&
    cssPropertiesToString({ ...$style, display: $style.display || "flex" })}
    ${$before &&
    `
      &::before {
        content: "";
        ${cssPropertiesToString($before)}
      }
    `}
    ${$after &&
    `
      &::after {
        content: "";
        ${cssPropertiesToString($after)}
      }
    `}
    ${$hover &&
    `
      transition: all .2s;

      &:hover {
        ${cssPropertiesToString($hover)}
      }
    `}
  `}
`;

export type StackProps = {
  $after?: CSSProperties;
  $before?: CSSProperties;
  $hover?: CSSProperties;
  $style?: CSSProperties;
};
