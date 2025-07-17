import { Button } from "components/button";
import { StarIcon } from "icons/StarIcon";
import { FC } from "react";
import { Stack } from "styles/Stack";
import { routeTree } from "utils/constants/routes";
import { Plugin } from "utils/types";

export const PluginItem: FC<Plugin> = ({
  categoryId,
  description,
  id,
  title,
}) => {
  return (
    <Stack
      $backgroundColor="backgroundSecondary"
      $borderRadius="12px"
      $flexDirection="column"
      $gap="24px"
      $padding="12px"
      $fullHeight
    >
      <Stack $flexDirection="column" $gap="12px" $flexGrow>
        <Stack
          as="img"
          alt={title}
          src={`/plugins/${id}.jpg`}
          $borderRadius="6px"
          $fullWidth
        />
        <Stack $gap="8px">
          <Stack
            as="span"
            $backgroundColor="alertSuccess"
            $borderRadius="6px"
            $color="textPrimary"
            $lineHeight="20px"
            $padding="0 8px"
          >
            {categoryId}
          </Stack>
        </Stack>
        <Stack $flexDirection="column" $gap="4px">
          <Stack
            as="span"
            $fontSize="18px"
            $fontWeight="500"
            $lineHeight="28px"
          >
            {title}
          </Stack>
          <Stack as="span" $color="textExtraLight" $lineHeight="20px">
            {description}
          </Stack>
        </Stack>
      </Stack>
      <Stack $justifyContent="space-between">
        <Stack $gap="6px">
          <Stack
            as={StarIcon}
            $color="alertWarning"
            $fill="alertWarning"
            $fontSize="16px"
          />
          <Stack as="span" $gap="4px">
            <Stack as="span" $fontWeight={500}>
              4.5
            </Stack>
            <Stack as="span" $color="textExtraLight" $fontWeight={500}>
              (128)
            </Stack>
          </Stack>
        </Stack>
        <Stack as="span" $color="textExtraLight" $fontWeight={500}>
          Plugin fee: 0.1% per trade
        </Stack>
      </Stack>
      <Button href={routeTree.pluginDetails.link(id)} kind="primary">
        See Details
      </Button>
    </Stack>
  );
};
