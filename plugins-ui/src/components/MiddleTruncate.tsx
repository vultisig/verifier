import { Stack } from "components/Stack";
import { FC, useEffect, useRef, useState } from "react";
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

  useEffect(() => {
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
  }, [ellipsis, counter, text]);

  useEffect(() => {
    setState((prevState) => ({
      ...prevState,
      ellipsis: text,
      truncating: true,
    }));
  }, [text]);

  return (
    <Stack
      as="span"
      ref={elmRef}
      $style={{ display: "block", position: "relative", width }}
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
        <Stack
          as="span"
          $style={{ position: "absolute", visibility: "hidden" }}
        >
          {ellipsis}
        </Stack>
      ) : (
        ellipsis
      )}
    </Stack>
  );
};
