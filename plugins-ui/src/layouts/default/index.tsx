import { Avatar, Dropdown, MenuProps, message } from "antd";
import { Button } from "components/Button";
import { CurrencyModal } from "components/CurrencyModal";
import { LanguageModal } from "components/LanguageModal";
import { MiddleTruncate } from "components/MiddleTruncate";
import { Stack } from "components/Stack";
import { useApp } from "hooks/useApp";
import { BoxIcon } from "icons/BoxIcon";
import { CircleDollarSignIcon } from "icons/CircleDollarSignIcon";
import { LanguagesIcon } from "icons/LanguagesIcon";
import { LogOutIcon } from "icons/LogOutIcon";
import { MoonIcon } from "icons/MoonIcon";
import { SunIcon } from "icons/SunIcon";
import { VultisigLogoIcon } from "icons/VultisigLogoIcon";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Link, Outlet, useNavigate } from "react-router-dom";
import { useTheme } from "styled-components";
import { modalHash } from "utils/constants/core";
import { languageNames } from "utils/constants/language";
import { routeTree } from "utils/constants/routes";
import { getAccount } from "utils/services/extension";

export const DefaultLayout = () => {
  const { t } = useTranslation();
  const {
    address,
    connect,
    currency,
    disconnect,
    isConnected,
    language,
    setTheme,
    theme,
  } = useApp();
  const [messageApi, messageHolder] = message.useMessage();
  const navigate = useNavigate();
  const colors = useTheme();

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
      key: "3",
      label: `Theme: ${theme === "light" ? "Dark" : "Light"}`,
      icon: theme === "light" ? <MoonIcon /> : <SunIcon />,
      onClick: () => {
        setTheme(theme === "light" ? "dark" : "light");
      },
    },
    {
      danger: true,
      key: "4",
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
    <Stack $style={{ flexDirection: "column", minHeight: "100%" }}>
      <Stack
        $style={{
          alignItems: "center",
          backgroundColor: colors.bgPrimary.toHex(),
          borderBottomColor: colors.borderLight.toHex(),
          borderBottomStyle: "solid",
          borderBottomWidth: "1px",
          justifyContent: "center",
          height: "72px",
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
            $style={{
              alignItems: "center",
              color: colors.textPrimary.toHex(),
              gap: "10px",
            }}
            $hover={{ color: colors.textSecondary.toHex() }}
          >
            <Stack $style={{ position: "relative" }}>
              <BoxIcon color={colors.accentThree.toHex()} fontSize={40} />
              <Stack
                as={VultisigLogoIcon}
                color={colors.bgSecondary.toHex()}
                fontSize={24}
                $style={{
                  left: "50%",
                  position: "absolute",
                  top: "50%",
                  transform: "translate(-50%, -50%)",
                }}
              />
            </Stack>
            <Stack
              $style={{
                fontSize: "22px",
                fontWeight: "500",
                lineHeight: "40px",
              }}
            >
              App Store
            </Stack>
          </Stack>
          <Stack
            $style={{
              flexDirection: "row",
              fontWeight: "500",
              gap: "48px",
              lineHeight: "20px",
            }}
          >
            <Stack
              as={Link}
              to={routeTree.plugins.path}
              $hover={{ color: colors.accentThree.toHex() }}
            >
              Marketplace
            </Stack>
            {isConnected && (
              <Stack
                as={Link}
                to={routeTree.plugins.path}
                $hover={{ color: colors.accentThree.toHex() }}
              >
                My Apps
              </Stack>
            )}
            <Stack
              as={Link}
              to={routeTree.faq.path}
              $hover={{ color: colors.accentThree.toHex() }}
            >
              FAQ
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
      <Stack $style={{ flexGrow: "1", justifyContent: "center" }}>
        <Outlet />
      </Stack>

      {isConnected && (
        <>
          <CurrencyModal />
          <LanguageModal />
        </>
      )}
      {messageHolder}
    </Stack>
  );
};
