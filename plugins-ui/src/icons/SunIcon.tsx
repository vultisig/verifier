import { FC, SVGProps } from "react";

export const SunIcon: FC<SVGProps<SVGSVGElement>> = (props) => (
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
    <path d="M12 2V4M12 20V22M4.93005 4.92993L6.34005 6.33993M17.66 17.6599L19.07 19.0699M2 12H4M20 12H22M6.34005 17.6599L4.93005 19.0699M19.07 4.92993L17.66 6.33993M16 12C16 14.2091 14.2091 16 12 16C9.79086 16 8 14.2091 8 12C8 9.79086 9.79086 8 12 8C14.2091 8 16 9.79086 16 12Z" />
  </svg>
);
