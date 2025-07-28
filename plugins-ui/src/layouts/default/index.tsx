import { Avatar, Dropdown, Layout, MenuProps, message } from "antd";
import { Button } from "components/Button";
import { CurrencyModal } from "components/CurrencyModal";
import { LanguageModal } from "components/LanguageModal";
import { MiddleTruncate } from "components/MiddleTruncate";
import { Stack } from "components/Stack";
import { useApp } from "hooks/useApp";
import { CircleDollarSignIcon } from "icons/CircleDollarSignIcon";
import { LanguagesIcon } from "icons/LanguagesIcon";
import { LogOutIcon } from "icons/LogOutIcon";
import { VultisigLogoIcon } from "icons/VultisigLogoIcon";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Link, Outlet, useNavigate } from "react-router-dom";
import { modalHash } from "utils/constants/core";
import { languageNames } from "utils/constants/language";
import { routeTree } from "utils/constants/routes";
import { getAccount } from "utils/services/extension";

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
        <Stack
          $style={{ alignItems: "center", justifyContent: "space-between" }}
        >
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
        <Stack
          $style={{ alignItems: "center", justifyContent: "space-between" }}
        >
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
    setTimeout(() => {
      getAccount().then((account) => {
        if (account) connect();
      });
    }, 200);
  }, [connect]);

  return (
    <Stack as={Layout} $style={{ flexDirection: "column", minHeight: "100%" }}>
      <Stack
        as={Layout.Header}
        $style={{
          alignItems: "center",
          borderBottom: "solid 1px",
          borderColor: "borderLight",
          justifyContent: "center",
          height: "64px",
          position: "sticky",
          top: "0",
          zIndex: "2",
        }}
      >
        <Stack
          $style={{
            alignItems: "center",
            justifyContent: "space-between",
            maxWidth: "1200px",
            padding: "0 16px",
            width: "100%",
          }}
        >
          <Stack
            as={Link}
            state={true}
            to={routeTree.root.path}
            $style={{ alignItems: "center", color: "textPrimary", gap: "4px" }}
            $hover={{ color: "textLight" }}
          >
            <VultisigLogoIcon fontSize={32} />
            <Stack
              $style={{
                fontSize: "32px",
                fontWeight: "500",
                lineHeight: "32px",
              }}
            >
              Vultisig
            </Stack>
          </Stack>
          {isConnected && address ? (
            <Stack $style={{ alignItems: "center", gap: "20px" }}>
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
        </Stack>
      </Stack>
      <Outlet />

      <CurrencyModal />
      <LanguageModal />
      {messageHolder}
    </Stack>
  );
};
