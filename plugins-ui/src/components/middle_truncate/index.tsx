import { FC, useEffect, useRef, useState } from "react";
import { Stack } from "styles/Stack";
import { CSSProperties } from "utils/types";

type MiddleTruncateProps = {
  onClick?: () => void;
  text: string;
  width?: CSSProperties["width"];
};

type InitialState = {
  counter: number;
  ellipsis: string;
  truncating: boolean;
};

export const MiddleTruncate: FC<MiddleTruncateProps> = ({
  onClick,
  text,
  width,
}) => {
  const initialState: InitialState = {
    counter: 0,
    ellipsis: "",
    truncating: true,
  };
  const [state, setState] = useState(initialState);
  const { counter, ellipsis, truncating } = state;
  const elmRef = useRef<HTMLSpanElement>(null);

  const handleClick = () => {
    if (onClick) onClick();
  };

  const ellipsisDidUpdate = (): void => {
    if (elmRef.current) {
      const [child] = elmRef.current.children;
      const parentWidth = elmRef.current.clientWidth;
      const childWidth = child?.clientWidth ?? 0;

      if (childWidth > parentWidth) {
        const chunkLen = Math.ceil(text.length / 2) - counter;

        setState((prevState) => ({
          ...prevState,
          counter: counter + 1,
          ellipsis: `${text.slice(0, chunkLen)}...${text.slice(chunkLen * -1)}`,
        }));
      } else {
        setState((prevState) => ({
          ...prevState,
          counter: 0,
          truncating: false,
        }));
      }
    }
  };

  const componentDidUpdate = (): void => {
    setState((prevState) => ({
      ...prevState,
      ellipsis: text,
      truncating: true,
    }));
  };

  useEffect(ellipsisDidUpdate, [ellipsis, counter, text]);
  useEffect(componentDidUpdate, [text]);

  return (
    <Stack
      as="span"
      ref={elmRef}
      $display="block"
      $position="relative"
      $width={width}
      {...() =>
        onClick
          ? {
              onClick: handleClick,
              onKeyDown: (e: React.KeyboardEvent) => {
                if (e.key === "Enter" || e.key === " ") handleClick();
              },
              role: "button" as const,
              tabIndex: 0,
            }
          : {}}
    >
      {truncating ? (
        <Stack as="span" $position="absolute" $visibility="hidden">
          {ellipsis}
        </Stack>
      ) : (
        ellipsis
      )}
    </Stack>
  );
};
