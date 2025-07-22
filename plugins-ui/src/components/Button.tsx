import { Spin } from "antd";
import { Stack, StackProps } from "components/Stack";
import { ButtonHTMLAttributes, FC, HTMLAttributes, ReactNode } from "react";
import { Link } from "react-router-dom";

type Kind = "default" | "primary" | "link";
type Size = "sm" | "md";
type Status = "default" | "danger" | "success" | "warning";

type ButtonProps = HTMLAttributes<HTMLElement> & {
  disabled?: boolean;
  href?: string;
  icon?: ReactNode;
  kind?: Kind;
  loading?: boolean;
  size?: Size;
  status?: Status;
  type?: ButtonHTMLAttributes<HTMLButtonElement>["type"];
};

const activeProps: Record<Kind, Record<Status, StackProps>> = {
  default: {
    default: {
      $style: {
        borderColor: "buttonPrimary",
        borderStyle: "solid",
        borderWidth: "1px",
        color: "textPrimary",
      },
      $hover: {
        color: "buttonPrimary",
      },
    },
    danger: {
      $style: {
        borderColor: "alertError",
        borderStyle: "solid",
        borderWidth: "1px",
        color: "textPrimary",
      },
      $hover: {
        color: "alertError",
      },
    },
    success: {
      $style: {
        borderColor: "alertSuccess",
        borderStyle: "solid",
        borderWidth: "1px",
        color: "textPrimary",
      },
      $hover: {
        color: "alertSuccess",
      },
    },
    warning: {
      $style: {
        borderColor: "alertWarning",
        borderStyle: "solid",
        borderWidth: "1px",
        color: "textPrimary",
      },
      $hover: {
        color: "alertWarning",
      },
    },
  },
  link: {
    default: {
      $style: {
        borderColor: "transparent",
        borderStyle: "solid",
        borderWidth: "1px",
        color: "textPrimary",
      },
      $hover: {
        borderColor: "buttonPrimary",
      },
    },
    danger: {
      $style: {
        borderColor: "transparent",
        borderStyle: "solid",
        borderWidth: "1px",
        color: "alertError",
      },
      $hover: {
        borderColor: "alertError",
      },
    },
    success: {
      $style: {
        borderColor: "transparent",
        borderStyle: "solid",
        borderWidth: "1px",
        color: "alertSuccess",
      },
      $hover: {
        borderColor: "alertSuccess",
      },
    },
    warning: {
      $style: {
        borderColor: "transparent",
        borderStyle: "solid",
        borderWidth: "1px",
        color: "alertWarning",
      },
      $hover: {
        borderColor: "alertWarning",
      },
    },
  },
  primary: {
    default: {
      $style: {
        backgroundColor: "buttonPrimary",
        color: "textPrimary",
      },
      $hover: {
        backgroundColor: "buttonPrimaryHover",
        color: "textPrimary",
      },
    },
    danger: {
      $style: {
        backgroundColor: "alertError",
        color: "textPrimary",
      },
      $hover: {
        backgroundColor: "alertError",
        color: "textPrimary",
      },
    },
    success: {
      $style: {
        backgroundColor: "alertSuccess",
        color: "textPrimary",
      },
      $hover: {
        backgroundColor: "alertSuccess",
        color: "textPrimary",
      },
    },
    warning: {
      $style: {
        backgroundColor: "alertWarning",
        color: "textPrimary",
      },
      $hover: {
        backgroundColor: "alertWarning",
        color: "textPrimary",
      },
    },
  },
};

const disabledProps: Record<Kind, StackProps> = {
  default: {
    $style: {
      borderColor: "buttonBackgroundDisabled",
      color: "buttonTextDisabled",
    },
  },
  link: {
    $style: {
      color: "buttonTextDisabled",
    },
  },
  primary: {
    $style: {
      backgroundColor: "buttonBackgroundDisabled",
      color: "buttonTextDisabled",
    },
  },
};

export const Button: FC<ButtonProps> = ({
  children,
  disabled = false,
  href,
  icon,
  kind = "default",
  loading = false,
  size = "md",
  status = "default",
  type = "button",
  ...buttonProps
}) => {
  const stackProps: StackProps = {
    $style: {
      alignItems: "center",
      backgroundColor: "transparent",
      border: "none",
      borderRadius: "44px",
      cursor: disabled ? "default" : "pointer",
      fontWeight: "500",
      gap: "8px",
      justifyContent: "center",
      height: "44px",
      padding: children ? "0 24px" : "0",
      ...(disabled
        ? disabledProps[kind].$style
        : activeProps[kind][status].$style),
    },
    $hover: {
      ...(disabled
        ? disabledProps[kind].$hover
        : activeProps[kind][status].$hover),
    },
  };

  return (
    <Stack
      {...stackProps}
      {...buttonProps}
      {...(disabled
        ? { as: "span" }
        : href
        ? { as: Link, state: true, to: href }
        : { as: "button", type })}
    >
      {loading ? <Spin size="small" /> : icon}
      {children}
    </Stack>
  );
};
