import styled, { css } from "styled-components";
import { cssPropertiesToString } from "utils/functions";
import { CSSProperties } from "utils/types";

const stackPropertiesToString = ({
  $after,
  $before,
  $hover,
  $style,
}: StackProps) => css`
  ${$style && cssPropertiesToString($style)}

  ${$after &&
  css`
    &::after {
      ${cssPropertiesToString({ ...$after, content: $after.content || "" })}
    }
  `}

  ${$before &&
  css`
    &::before {
      ${cssPropertiesToString({ ...$before, content: $before.content || "" })}
    }
  `}

  ${$hover &&
  css`
    ${!$style?.transition &&
    css`
      transition: all 0.2s;
    `}

    &:hover {
      ${cssPropertiesToString($hover)}
    }
  `}
`;

export const Stack = styled.div<
  StackProps & { $media?: { lg?: StackProps; xl?: StackProps } }
>`
  ${({ $style, $before, $after, $hover, $media }) => css`
    ${stackPropertiesToString({
      $after,
      $before,
      $hover,
      $style: $style
        ? { ...$style, display: $style.display || "flex" }
        : $style,
    })}

    ${$media?.lg &&
    css`
      @media (min-width: 992px) {
        ${stackPropertiesToString($media.lg)}
      }
    `}

    ${$media?.xl &&
    css`
      @media (min-width: 1200px) {
        ${stackPropertiesToString($media.xl)}
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
