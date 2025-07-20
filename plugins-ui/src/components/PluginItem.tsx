import { Button } from "components/Button";
import { Stack } from "components/Stack";
import { FC } from "react";
import { routeTree } from "utils/constants/routes";
import { Plugin } from "utils/types";
import { Tag } from "components/Tag";
import { Rate } from "components/Rate";

export const PluginItem: FC<Plugin> = ({
  categoryId,
  description,
  id,
  title,
}) => {
  return (
    <Stack
      $style={{
        backgroundColor: "backgroundSecondary",
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
          <Tag color="alertSuccess" text={categoryId} />
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
            $style={{ color: "textExtraLight", lineHeight: "20px" }}
          >
            {description}
          </Stack>
        </Stack>
      </Stack>
      <Stack $style={{ justifyContent: "space-between" }}>
        <Rate count={128} value={4.5} />
        <Stack
          as="span"
          $style={{ color: "textExtraLight", fontWeight: "500" }}
        >
          Plugin fee: 0.1% per trade
        </Stack>
      </Stack>
      <Button href={routeTree.pluginDetails.link(id)} kind="primary">
        See Details
      </Button>
    </Stack>
  );
};
