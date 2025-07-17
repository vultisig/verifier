import styled, { css } from "styled-components";
import { ThemeColorKeys } from "utils/constants/styled";
import { formatSize, isUndefined } from "utils/functions";
import { CSSProperties } from "utils/types";

export const Stack = styled.div<StackProps>`
  ${({ $alignItems }) =>
    isUndefined($alignItems)
      ? css``
      : css`
          align-items: ${$alignItems};
        `}

  ${({ $backgroundColor, theme }) =>
    isUndefined($backgroundColor)
      ? css``
      : css`
          background-color: ${theme[$backgroundColor]};
        `}

  ${({ $backgroundColorHover, theme }) =>
    isUndefined($backgroundColorHover)
      ? css``
      : css`
          &:hover {
            background-color: ${theme[$backgroundColorHover]};
          }
        `}

  ${({ $border }) =>
    isUndefined($border)
      ? css``
      : css`
          border: ${$border};
        `}

  ${({ $borderColor, theme }) =>
    isUndefined($borderColor)
      ? css``
      : css`
          border-color: ${theme[$borderColor]};
        `}

  ${({ $borderStyle }) =>
    isUndefined($borderStyle)
      ? css``
      : css`
          border-style: ${$borderStyle};
        `}

  ${({ $borderWidth }) =>
    isUndefined($borderWidth)
      ? css``
      : css`
          border-width: ${formatSize($borderWidth)};
        `}

  ${({ $borderRadius }) =>
    isUndefined($borderRadius)
      ? css``
      : css`
          border-radius: ${formatSize($borderRadius)};
        `}

  ${({ $color, theme }) =>
    isUndefined($color)
      ? css``
      : css`
          color: ${theme[$color]};
        `}

  ${({ $colorHover, theme }) =>
    isUndefined($colorHover)
      ? css``
      : css`
          &:hover {
            color: ${theme[$colorHover]};
          }
        `}
  
  ${({ $cursor }) =>
    isUndefined($cursor)
      ? css``
      : css`
          cursor: ${$cursor};
        `}
  
  ${({ $display = "flex" }) =>
    css`
      display: ${$display};
    `}

  ${({ $fill, theme }) =>
    isUndefined($fill)
      ? css``
      : css`
          fill: ${theme[$fill]};
        `}

  ${({ $flexDirection }) =>
    isUndefined($flexDirection)
      ? css``
      : css`
          flex-direction: ${$flexDirection};
        `}

  ${({ $flexGrow }) =>
    $flexGrow
      ? css`
          flex-grow: 1;
        `
      : css``}

  ${({ $fontSize }) =>
    isUndefined($fontSize)
      ? css``
      : css`
          font-size: ${formatSize($fontSize)};
        `}
  
  ${({ $fontWeight }) =>
    isUndefined($fontWeight)
      ? css``
      : css`
          font-weight: ${$fontWeight};
        `}

  ${({ $fullHeight, $height }) =>
    $fullHeight
      ? css`
          height: 100%;
        `
      : isUndefined($height)
      ? css``
      : css`
          height: ${formatSize($height)};
        `}

  ${({ $fullWidth }) =>
    $fullWidth
      ? css`
          width: 100%;
        `
      : css``}

  ${({ $gap }) =>
    isUndefined($gap)
      ? css``
      : css`
          gap: ${formatSize($gap)};
        `}

  ${({ $justifyContent }) =>
    isUndefined($justifyContent)
      ? css``
      : css`
          justify-content: ${$justifyContent};
        `}
        
  ${({ $left }) =>
    isUndefined($left)
      ? css``
      : css`
          left: ${formatSize($left)};
        `}
    
  ${({ $lineHeight }) =>
    isUndefined($lineHeight)
      ? css``
      : css`
          line-height: ${formatSize($lineHeight)};
        `}

  ${({ $maxWidth }) =>
    isUndefined($maxWidth)
      ? css``
      : css`
          max-width: ${formatSize($maxWidth)};
        `}
  
  ${({ $minHeight }) =>
    isUndefined($minHeight)
      ? css``
      : css`
          min-height: ${formatSize($minHeight)};
        `}

  ${({ $padding }) =>
    isUndefined($padding)
      ? css``
      : css`
          padding: ${formatSize($padding)};
        `}

  ${({ $paddingLeft }) =>
    isUndefined($paddingLeft)
      ? css``
      : css`
          padding-left: ${formatSize($paddingLeft)};
        `}

  ${({ $position }) =>
    isUndefined($position)
      ? css``
      : css`
          position: ${$position};
        `}

  ${({ $top }) =>
    isUndefined($top)
      ? css``
      : css`
          top: ${formatSize($top)};
        `}

  ${({ $transform }) =>
    isUndefined($transform)
      ? css``
      : css`
          transform: ${$transform};
        `}

  ${({ $backgroundColorHover, $colorHover }) =>
    isUndefined($backgroundColorHover) && isUndefined($colorHover)
      ? css``
      : css`
          transition: all 0.2s;
        `}

  ${({ $zIndex }) =>
    isUndefined($zIndex)
      ? css``
      : css`
          z-index: ${$zIndex};
        `}
`;

export type StackProps = {
  $alignItems?: CSSProperties["alignItems"];
  $backgroundColor?: ThemeColorKeys;
  $backgroundColorHover?: ThemeColorKeys;
  $border?: CSSProperties["border"];
  $borderColor?: ThemeColorKeys;
  $borderStyle?: CSSProperties["borderStyle"];
  $borderWidth?: CSSProperties["borderWidth"];
  $borderRadius?: CSSProperties["borderRadius"];
  $color?: ThemeColorKeys;
  $colorHover?: ThemeColorKeys;
  $cursor?: CSSProperties["cursor"];
  $display?: CSSProperties["display"];
  $gap?: CSSProperties["gap"];
  $fill?: ThemeColorKeys;
  $flexDirection?: CSSProperties["flexDirection"];
  $flexGrow?: boolean;
  $fontSize?: CSSProperties["fontSize"];
  $fontWeight?: CSSProperties["fontWeight"];
  $fullHeight?: boolean;
  $fullWidth?: boolean;
  $height?: CSSProperties["height"];
  $justifyContent?: CSSProperties["justifyContent"];
  $left?: CSSProperties["left"];
  $lineHeight?: CSSProperties["lineHeight"];
  $maxWidth?: CSSProperties["maxWidth"];
  $minHeight?: CSSProperties["minHeight"];
  $padding?: CSSProperties["padding"];
  $paddingLeft?: CSSProperties["paddingLeft"];
  $position?: CSSProperties["position"];
  $top?: CSSProperties["top"];
  $transform?: CSSProperties["transform"];
  $zIndex?: CSSProperties["zIndex"];
};
