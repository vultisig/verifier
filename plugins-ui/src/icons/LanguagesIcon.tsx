import { FC, SVGProps } from "react";

export const LanguagesIcon: FC<SVGProps<SVGSVGElement>> = (props) => (
  <svg
    fill="none"
    height="1em"
    stroke="currentColor"
    strokeLinecap="round"
    strokeLinejoin="round"
    strokeWidth="1.5"
    viewBox="0 0 24 24"
    width="1em"
    {...props}
  >
    <path d="M5 8L11 14M4 14L10 8L12 5M2 5H14M7 2H8M22 22L17 12L12 22M14 18H20" />
  </svg>
);
