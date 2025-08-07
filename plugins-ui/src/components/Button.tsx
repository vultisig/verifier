import { Spin } from "components/Spin";
import { ButtonHTMLAttributes, FC, HTMLAttributes, ReactNode } from "react";
import { Link } from "react-router-dom";
import styled, { css } from "styled-components";
import { match } from "utils/functions";

type Kind = "default" | "primary" | "link";
type Status = "default" | "danger" | "success" | "warning";

type ButtonProps = HTMLAttributes<HTMLElement> & {
  disabled?: boolean;
  href?: string;
  icon?: ReactNode;
  kind?: Kind;
  loading?: boolean;
  status?: Status;
  type?: ButtonHTMLAttributes<HTMLButtonElement>["type"];
};

const StyledButton = styled.div<{
  $disabled: boolean;
  $kind: Kind;
  $status: Status;
}>`
  display: flex;
  align-items: center;
  border-radius: 44px;
  cursor: ${({ $disabled }) => ($disabled ? "default" : "pointer")};
  font-family: inherit;
  font-weight: 500;
  gap: 8px;
  justify-content: center;
  height: 44px;
  padding: 0 24px;
  transition: all 0.2s;

  ${({ $disabled, $kind, $status }) =>
    css`
      ${match($kind, {
        default: () =>
          $disabled
            ? css`
                background-color: transparent;
                color: ${({ theme }) => theme.buttonTextDisabled.toHex()};

                ${match($status, {
                  default: () => css`
                    border: solid 1px
                      ${({ theme }) => theme.buttonPrimary.toRgba(0.6)};
                  `,
                  danger: () => css`
                    border: solid 1px ${({ theme }) => theme.error.toRgba(0.6)};
                  `,
                  success: () => css`
                    border: solid 1px
                      ${({ theme }) => theme.success.toRgba(0.6)};
                  `,
                  warning: () => css`
                    border: solid 1px
                      ${({ theme }) => theme.warning.toRgba(0.6)};
                  `,
                })}
              `
            : css`
                background-color: transparent;
                color: ${({ theme }) => theme.textPrimary.toHex()};

                ${match($status, {
                  default: () => css`
                    border: solid 1px
                      ${({ theme }) => theme.buttonPrimary.toHex()};
                  `,
                  danger: () => css`
                    border: solid 1px ${({ theme }) => theme.error.toHex()};
                  `,
                  success: () => css`
                    border: solid 1px ${({ theme }) => theme.success.toHex()};
                  `,
                  warning: () => css`
                    border: solid 1px ${({ theme }) => theme.warning.toHex()};
                  `,
                })}

                &:hover {
                  ${match($status, {
                    default: () => css`
                      background-color: ${({ theme }) =>
                        theme.buttonPrimary.toRgba(0.2)};
                    `,
                    danger: () => css`
                      background-color: ${({ theme }) =>
                        theme.error.toRgba(0.2)};
                    `,
                    success: () => css`
                      background-color: ${({ theme }) =>
                        theme.success.toRgba(0.2)};
                    `,
                    warning: () => css`
                      background-color: ${({ theme }) =>
                        theme.warning.toRgba(0.2)};
                    `,
                  })}
                }
              `,
        link: () =>
          $disabled
            ? css`
                background-color: transparent;
                border: none;
                color: ${({ theme }) => theme.buttonTextDisabled.toHex()};
              `
            : css`
                background-color: transparent;
                border: none;
                color: ${({ theme }) => theme.textPrimary.toHex()};

                &:hover {
                  ${match($status, {
                    default: () => css`
                      color: ${({ theme }) => theme.buttonPrimary.toHex()};
                    `,
                    danger: () => css`
                      color: ${({ theme }) => theme.error.toHex()};
                    `,
                    success: () => css`
                      color: ${({ theme }) => theme.success.toHex()};
                    `,
                    warning: () => css`
                      color: ${({ theme }) => theme.warning.toHex()};
                    `,
                  })}
                }
              `,
        primary: () =>
          $disabled
            ? css`
                border: none;
                background-color: ${({ theme }) =>
                  theme.buttonDisabled.toHex()};
                color: ${({ theme }) => theme.buttonTextDisabled.toHex()};
              `
            : css`
                border: none;
                color: ${({ theme }) => theme.buttonText.toHex()};

                ${match($status, {
                  default: () => css`
                    background-color: ${({ theme }) =>
                      theme.buttonPrimary.toHex()};
                  `,
                  danger: () => css`
                    background-color: ${({ theme }) => theme.error.toHex()};
                  `,
                  success: () => css`
                    background-color: ${({ theme }) => theme.success.toHex()};
                  `,
                  warning: () => css`
                    background-color: ${({ theme }) => theme.warning.toHex()};
                  `,
                })}

                &:hover {
                  color: ${({ theme }) => theme.buttonText.toHex()};

                  ${match($status, {
                    default: () => css`
                      background-color: ${({ theme }) =>
                        theme.buttonPrimaryHover.toHex()};
                    `,
                    danger: () => css`
                      background-color: ${({ theme }) =>
                        theme.error.lighten(10).toHex()};
                    `,
                    success: () => css`
                      background-color: ${({ theme }) =>
                        theme.success.lighten(10).toHex()};
                    `,
                    warning: () => css`
                      background-color: ${({ theme }) =>
                        theme.warning.lighten(10).toHex()};
                    `,
                  })}
                }
              `,
      })}
    `}
`;

export const Button: FC<ButtonProps> = (props) => {
  const {
    children,
    disabled = false,
    href,
    icon,
    kind = "default",
    loading = false,
    status = "default",
    type = "button",
    ...rest
  } = props;

  return (
    <StyledButton
      $disabled={disabled}
      $kind={kind}
      $status={status}
      {...rest}
      {...(disabled
        ? { as: "span" }
        : href
        ? { as: Link, state: true, to: href }
        : { as: "button", type })}
    >
      {loading ? <Spin size="small" /> : icon}
      {children}
    </StyledButton>
  );
};
