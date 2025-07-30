import { Button } from "components/Button";
import { Pricing } from "components/Pricing";
import { Rate } from "components/Rate";
import { Stack } from "components/Stack";
import { Tag } from "components/Tag";
import { useApp } from "hooks/useApp";
import { FC, useEffect, useState } from "react";
import { useTheme } from "styled-components";
import { routeTree } from "utils/constants/routes";
import { toCapitalizeFirst } from "utils/functions";
import { isPluginInstalled } from "utils/services/marketplace";
import { Plugin } from "utils/types";

type InitialState = {
  isInstalled?: boolean;
};

export const PluginItem: FC<Plugin> = ({
  categoryId,
  description,
  id,
  pricing,
  title,
}) => {
  const initialState: InitialState = {};
  const [state, setState] = useState(initialState);
  const { isInstalled } = state;
  const { isConnected } = useApp();
  const colors = useTheme();

  useEffect(() => {
    if (isConnected) {
      isPluginInstalled(id).then((isInstalled) => {
        setState((prevState) => ({ ...prevState, isInstalled }));
      });
    } else {
      setState((prevState) => ({ ...prevState, isInstalled: undefined }));
    }
  }, [id, isConnected]);

  return (
    <Stack
      $style={{
        backgroundColor: colors.bgSecondary.toHex(),
        borderRadius: "12px",
        flexDirection: "column",
        gap: "24px",
        height: "100%",
        padding: "12px",
      }}
    >
      <Stack $style={{ flexDirection: "column", flexGrow: "1", gap: "12px" }}>
        <Stack
          as="img"
          alt={title}
          src={`/plugins/${id}.jpg`}
          $style={{ borderRadius: "6px", width: "100%" }}
        />
        <Stack $style={{ gap: "8px" }}>
          <Tag color="success" text={toCapitalizeFirst(categoryId)} />
          {isInstalled && <Tag color="buttonPrimary" text="Installed" />}
        </Stack>
        <Stack $style={{ flexDirection: "column", gap: "4px" }}>
          <Stack
            as="span"
            $style={{ fontSize: "18px", fontWeight: "500", lineHeight: "28px" }}
          >
            {title}
          </Stack>
          <Stack
            as="span"
            $style={{ color: colors.textTertiary.toHex(), lineHeight: "20px" }}
          >
            {description}
          </Stack>
        </Stack>
      </Stack>
      <Stack $style={{ alignItems: "end", justifyContent: "space-between" }}>
        <Rate count={128} value={4.5} />
        <Pricing pricing={pricing} />
      </Stack>
      <Button href={routeTree.pluginDetails.link(id)} kind="primary">
        See Details
      </Button>
    </Stack>
  );
};
