import { Avatar, Dropdown, Layout, MenuProps, message } from "antd";
import { Button } from "components/button";
import { CurrencyModal } from "components/currency_modal";
import { LanguageModal } from "components/language_modal";
import { MiddleTruncate } from "components/middle_truncate";
import { useApp } from "hooks/useApp";
import { CircleDollarSignIcon } from "icons/CircleDollarSignIcon";
import { LanguagesIcon } from "icons/LanguagesIcon";
import { LogOutIcon } from "icons/LogOutIcon";
import { VultisigLogoIcon } from "icons/VultisigLogoIcon";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Link, Outlet, useNavigate } from "react-router-dom";
import { Container } from "styles/Container";
import { Stack } from "styles/Stack";
import { modalHash } from "utils/constants/core";
import { languageNames } from "utils/constants/language";
import { routeTree } from "utils/constants/routes";

export const DefaultLayout = () => {
  const { t } = useTranslation();
  const { address, connect, currency, disconnect, isConnected, language } =
    useApp();
  const [messageApi, messageHolder] = message.useMessage();
  const navigate = useNavigate();

  const dropdownMenu: MenuProps["items"] = [
    {
      key: "1",
      label: (
        <Stack $alignItems="center" $justifyContent="space-between">
          <span>{t("language")}</span>
          <span>{languageNames[language]}</span>
        </Stack>
      ),
      icon: <LanguagesIcon />,
      onClick: () => {
        navigate(modalHash.language, { state: true });
      },
    },
    {
      key: "2",
      label: (
        <Stack $alignItems="center" $justifyContent="space-between">
          <span>{t("currency")}</span>
          <span>{currency.toUpperCase()}</span>
        </Stack>
      ),
      icon: <CircleDollarSignIcon />,
      onClick: () => {
        navigate(modalHash.currency, { state: true });
      },
    },
    {
      danger: true,
      key: "3",
      label: "Disconnect",
      icon: <LogOutIcon />,
      onClick: disconnect,
    },
  ];

  const copyAddress = () => {
    if (address) {
      navigator.clipboard.writeText(address);

      messageApi.success("Address copied to clipboard!");
    } else {
      messageApi.error("No address to copy.");
    }
  };

  useEffect(() => {
    setTimeout(() => connect(), 100);
  }, []);

  return (
    <Stack as={Layout} $flexDirection="column" $minHeight="100%">
      <Stack
        as={Layout.Header}
        $alignItems="center"
        $justifyContent="center"
        $height="64px"
        $position="sticky"
        $top="0"
        $zIndex="1"
      >
        <Container $alignItems="center" $justifyContent="space-between">
          <Stack
            as={Link}
            state={true}
            to={routeTree.root.path}
            $alignItems="center"
            $color="textPrimary"
            $colorHover="textLight"
            $gap="4px"
          >
            <VultisigLogoIcon fontSize={32} />
            <Stack $fontSize="32px" $fontWeight="500" $lineHeight="32px">
              Vultisig
            </Stack>
          </Stack>
          {isConnected && address ? (
            <Stack $alignItems="center" $gap="20px">
              <Button kind="primary" onClick={copyAddress}>
                <MiddleTruncate text={address} width="118px" />
              </Button>
              <Dropdown
                menu={{ items: dropdownMenu }}
                overlayStyle={{ width: 302 }}
              >
                <Avatar src="/avatars/01.png" size={44} />
              </Dropdown>
            </Stack>
          ) : (
            <Button kind="primary" onClick={connect}>
              Connect Wallet
            </Button>
          )}
        </Container>
      </Stack>
      <Stack
        as={Layout.Content}
        $justifyContent="center"
        $padding="30px 0"
        $flexGrow
      >
        <Outlet />
      </Stack>

      <CurrencyModal />
      <LanguageModal />
      {messageHolder}
    </Stack>
  );
};
