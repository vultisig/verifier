import { Spin } from "antd";
import {
  ButtonHTMLAttributes,
  FC,
  HTMLAttributes,
  ReactNode,
  useMemo,
} from "react";
import { Link } from "react-router-dom";
import { Stack, StackProps } from "styles/Stack";

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

const activeProps: Record<
  Kind,
  Record<
    Status,
    Pick<
      StackProps,
      | "$backgroundColor"
      | "$backgroundColorHover"
      | "$borderColor"
      | "$borderStyle"
      | "$borderWidth"
      | "$color"
      | "$colorHover"
    >
  >
> = {
  default: {
    default: {
      $borderColor: "buttonPrimary",
      $borderStyle: "solid",
      $borderWidth: "1px",
      $color: "textPrimary",
      $colorHover: "buttonPrimary",
    },
    danger: {
      $borderColor: "alertError",
      $borderStyle: "solid",
      $borderWidth: "1px",
      $color: "textPrimary",
      $colorHover: "alertError",
    },
    success: {
      $borderColor: "alertSuccess",
      $borderStyle: "solid",
      $borderWidth: "1px",
      $color: "textPrimary",
      $colorHover: "alertSuccess",
    },
    warning: {
      $borderColor: "alertWarning",
      $borderStyle: "solid",
      $borderWidth: "1px",
      $color: "textPrimary",
      $colorHover: "alertWarning",
    },
  },
  link: {
    default: {
      $color: "textPrimary",
    },
    danger: {
      $color: "alertError",
    },
    success: {
      $color: "alertSuccess",
    },
    warning: {
      $color: "alertWarning",
    },
  },
  primary: {
    default: {
      $backgroundColor: "buttonPrimary",
      $backgroundColorHover: "buttonPrimaryHover",
      $color: "textPrimary",
      $colorHover: "textPrimary",
    },
    danger: {
      $backgroundColor: "alertError",
      $backgroundColorHover: "alertError",
      $color: "textPrimary",
      $colorHover: "textPrimary",
    },
    success: {
      $backgroundColor: "alertSuccess",
      $backgroundColorHover: "alertSuccess",
      $color: "textPrimary",
      $colorHover: "textPrimary",
    },
    warning: {
      $backgroundColor: "alertWarning",
      $backgroundColorHover: "alertWarning",
      $color: "textPrimary",
      $colorHover: "textPrimary",
    },
  },
};

const disabledProps: Record<
  Kind,
  Pick<StackProps, "$backgroundColor" | "$borderColor" | "$color">
> = {
  default: {
    $borderColor: "buttonBackgroundDisabled",
    $color: "buttonTextDisabled",
  },
  link: {
    $color: "buttonTextDisabled",
  },
  primary: {
    $backgroundColor: "buttonBackgroundDisabled",
    $color: "buttonTextDisabled",
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
  ...rest
}) => {
  const props: StackProps = {
    $alignItems: "center",
    $backgroundColor: "transparent",
    $border: "none",
    $borderRadius: "44px",
    $cursor: disabled ? "default" : "pointer",
    $fontWeight: "500",
    $gap: "8px",
    $justifyContent: "center",
    $height: "44px",
    $padding: children ? "0 24px" : "0",
    ...(disabled ? disabledProps[kind] : activeProps[kind][status]),
  };

  const content = useMemo(() => {
    return (
      <>
        {loading ? <Spin size="small" /> : icon}
        {children}
      </>
    );
  }, [children, icon, loading]);

  return disabled ? (
    <Stack as="span" {...props} {...rest}>
      {content}
    </Stack>
  ) : href ? (
    <Stack as={Link} to={href} state={true} {...props} {...rest}>
      {content}
    </Stack>
  ) : (
    <Stack as="button" type={type} {...props} {...rest}>
      {content}
    </Stack>
  );
};
