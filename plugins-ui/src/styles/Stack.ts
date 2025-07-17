import styled, { css } from "styled-components";
import { ThemeColorKeys } from "utils/constants/styled";
import { CSSProperties } from "utils/types";

export const Stack = styled.div<StackProps>`
  ${({ $alignItems }) =>
    $alignItems
      ? css`
          align-items: ${$alignItems};
        `
      : css``}

  ${({ $backgroundColor, theme }) =>
    $backgroundColor
      ? css`
          background-color: ${theme[$backgroundColor]};
        `
      : css``}

  ${({ $backgroundColorHover, theme }) =>
    $backgroundColorHover
      ? css`
          &:hover {
            background-color: ${theme[$backgroundColorHover]};
          }
        `
      : css``}

  ${({ $border }) =>
    $border
      ? css`
          border: ${$border};
        `
      : css``}

  ${({ $borderColor, theme }) =>
    $borderColor
      ? css`
          border-color: ${theme[$borderColor]};
        `
      : css``}

  ${({ $borderStyle }) =>
    $borderStyle
      ? css`
          border-style: ${$borderStyle};
        `
      : css``}

  ${({ $borderWidth }) =>
    $borderWidth
      ? css`
          border-width: ${$borderWidth};
        `
      : css``}

  ${({ $borderRadius }) =>
    $borderRadius
      ? css`
          border-radius: ${$borderRadius};
        `
      : css``}

  ${({ $color, theme }) =>
    $color
      ? css`
          color: ${theme[$color]};
        `
      : css``}

  ${({ $colorHover, theme }) =>
    $colorHover
      ? css`
          &:hover {
            color: ${theme[$colorHover]};
          }
        `
      : css``}
  
  ${({ $cursor }) =>
    $cursor
      ? css`
          cursor: ${$cursor};
        `
      : css``}
  
  ${({ $display = "flex" }) =>
    css`
      display: ${$display};
    `}

  ${({ $fill, theme }) =>
    $fill
      ? css`
          fill: ${theme[$fill]};
        `
      : css``}

  ${({ $flexDirection }) =>
    $flexDirection
      ? css`
          flex-direction: ${$flexDirection};
        `
      : css``}

  ${({ $flexGrow }) =>
    $flexGrow
      ? css`
          flex-grow: 1;
        `
      : css``}

  ${({ $fontSize }) =>
    $fontSize
      ? css`
          font-size: ${$fontSize};
        `
      : css``}
  
  ${({ $fontWeight }) =>
    $fontWeight
      ? css`
          font-weight: ${$fontWeight};
        `
      : css``}

  ${({ $fullHeight, $height }) =>
    $fullHeight
      ? css`
          height: 100%;
        `
      : $height
      ? css`
          height: ${$height};
        `
      : css``}

  ${({ $fullWidth, $width }) =>
    $fullWidth
      ? css`
          width: 100%;
        `
      : $width
      ? css`
          width: ${$width};
        `
      : css``}

  ${({ $gap }) =>
    $gap
      ? css`
          gap: ${$gap};
        `
      : css``}

  ${({ $justifyContent }) =>
    $justifyContent
      ? css`
          justify-content: ${$justifyContent};
        `
      : css``}
        
  ${({ $left }) =>
    $left
      ? css`
          left: ${$left};
        `
      : css``}
    
  ${({ $lineHeight }) =>
    $lineHeight
      ? css`
          line-height: ${$lineHeight};
        `
      : css``}

  ${({ $margin }) =>
    $margin
      ? css`
          margin: ${$margin};
        `
      : css``}

  ${({ $maxWidth }) =>
    $maxWidth
      ? css`
          max-width: ${$maxWidth};
        `
      : css``}
  
  ${({ $minHeight }) =>
    $minHeight
      ? css`
          min-height: ${$minHeight};
        `
      : css``}

  ${({ $padding }) =>
    $padding
      ? css`
          padding: ${$padding};
        `
      : css``}

  ${({ $paddingLeft }) =>
    $paddingLeft
      ? css`
          padding-left: ${$paddingLeft};
        `
      : css``}

  ${({ $position }) =>
    $position
      ? css`
          position: ${$position};
        `
      : css``}

  ${({ $top }) =>
    $top
      ? css`
          top: ${$top};
        `
      : css``}

  ${({ $transform }) =>
    $transform
      ? css`
          transform: ${$transform};
        `
      : css``}

  ${({ $backgroundColorHover, $colorHover }) =>
    $backgroundColorHover && $colorHover
      ? css`
          transition: all 0.2s;
        `
      : css``}

  ${({ $visibility }) =>
    $visibility
      ? css`
          visibility: ${$visibility};
        `
      : css``}

  ${({ $zIndex }) =>
    $zIndex
      ? css`
          z-index: ${$zIndex};
        `
      : css``}
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
  $margin?: CSSProperties["margin"];
  $maxWidth?: CSSProperties["maxWidth"];
  $minHeight?: CSSProperties["minHeight"];
  $padding?: CSSProperties["padding"];
  $paddingLeft?: CSSProperties["paddingLeft"];
  $position?: CSSProperties["position"];
  $top?: CSSProperties["top"];
  $transform?: CSSProperties["transform"];
  $visibility?: CSSProperties["visibility"];
  $width?: CSSProperties["width"];
  $zIndex?: CSSProperties["zIndex"];
};
