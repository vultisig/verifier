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
import { Download } from "./Download";

type PluginItemProps = {
  plugin: Plugin;
  horizontal?: boolean;
};

type InitialState = {
  isInstalled?: boolean;
};

export const PluginItem: FC<PluginItemProps> = ({ horizontal, plugin }) => {
  const initialState: InitialState = {};
  const [state, setState] = useState(initialState);
  const { isInstalled } = state;
  const { categoryId, description, id, pricing, title } = plugin;
  const { isConnected } = useApp();
  const colors = useTheme();

  // useEffect(() => {
  //   if (isConnected) {
  //     isPluginInstalled(id).then((isInstalled) => {
  //       setState((prevState) => ({ ...prevState, isInstalled }));
  //     });
  //   } else {
  //     setState((prevState) => ({ ...prevState, isInstalled: undefined }));
  //   }
  // }, [id, isConnected]);

  return (
    <Stack
      $style={{
        border: `solid 1px ${colors.borderNormal.toHex()}`,
        borderRadius: "24px",
        flexDirection: horizontal ? "row" : "column",
        gap: "24px",
        height: "100%",
        padding: "16px",
      }}
    >
      <Stack
        as="img"
        alt={title}
        src={`/plugins/automate-your-payrolls.jpg`}
        $style={{
          borderRadius: "12px",
          ...(horizontal ? { height: "224px" } : { width: "100%" }),
        }}
      />
      <Stack
        $style={{
          alignItems: horizontal ? "start" : "normal",
          flexDirection: "column",
          flexGrow: "1",
          gap: "20px",
        }}
      >
        <Stack $style={{ flexDirection: "row", gap: "12px" }}>
          <Stack
            as="img"
            alt={title}
            src={`/plugins/payroll.png`}
            $style={{ width: "56px" }}
          />
          <Stack
            $style={{
              flexDirection: "column",
              gap: "8px",
              justifyContent: "center",
            }}
          >
            <Stack
              as="span"
              $style={{
                fontSize: "17px",
                fontWeight: "500",
                lineHeight: "20px",
              }}
            >
              {title}
            </Stack>
            <Stack
              $style={{
                alignItems: "center",
                flexDirection: "row",
                gap: "8px",
              }}
            >
              <Download value={1258} />
              <Stack
                $style={{
                  backgroundColor: colors.borderLight.toHex(),
                  height: "3px",
                  width: "3px",
                }}
              />
              <Rate count={128} value={4.5} />
            </Stack>
          </Stack>
        </Stack>
        <Stack
          as="span"
          $style={{ flexGrow: 1, fontWeight: "500", lineHeight: "20px" }}
        >
          {description}
        </Stack>
        <Stack
          $style={{
            flexDirection: "column",
            gap: "12px",
          }}
        >
          <Button href={routeTree.pluginDetails.link(id)} kind="primary">
            See Details
          </Button>
          <Pricing pricing={pricing} center={!horizontal} />
        </Stack>
      </Stack>
    </Stack>
  );
};
