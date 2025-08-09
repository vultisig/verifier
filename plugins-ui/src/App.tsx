import { message, Modal } from "antd";
import { GlobalStyle } from "components/GlobalStyle";
import { Spin } from "components/Spin";
import { AppContext } from "context/AppContext";
import { hexlify, randomBytes } from "ethers";
import { i18nInstance } from "i18n/config";
import { DefaultLayout } from "layouts/default";
import { FaqPage } from "pages/faq";
import { NotFoundPage } from "pages/not_found";
import { PluginDetailsPage } from "pages/plugin_details";
import { PluginsPage } from "pages/plugins";
import { AntdProvider } from "providers/antd";
import { StyledProvider } from "providers/styled";
import { useCallback, useMemo, useState } from "react";
import { I18nextProvider } from "react-i18next";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { getChain, setChain as setChainStorage } from "storage/chain";
import { storageKeys } from "storage/constants";
import {
  getCurrency,
  setCurrency as setCurrencyStorage,
} from "storage/currency";
import { useLocalStorageWatcher } from "storage/hooks/useLocalStorageWatcher";
import {
  getLanguage,
  setLanguage as setLanguageStorage,
} from "storage/language";
import { getTheme, setTheme as setThemeStorage } from "storage/theme";
import { delToken, getToken, setToken } from "storage/token";
import { delVaultId, getVaultId, setVaultId } from "storage/vaultId";
import { Chain } from "utils/constants/chain";
import { Currency } from "utils/constants/currency";
import { Language } from "utils/constants/language";
import { routeTree } from "utils/constants/routes";
import { Theme } from "utils/constants/theme";
import {
  connect as connectToExtension,
  disconnect as disconnectFromExtension,
  getVault,
  signCustomMessage,
} from "utils/services/extension";
import { getAuthToken } from "utils/services/marketplace";

interface InitialState {
  address?: string;
  chain: Chain;
  currency: Currency;
  isConnected: boolean;
  language: Language;
  loaded: boolean;
  theme: Theme;
  token?: string;
  vaultId?: string;
}

export const App = () => {
  const initialState: InitialState = useMemo(() => {
    return {
      chain: getChain(),
      currency: getCurrency(),
      isConnected: false,
      language: getLanguage(),
      loaded: true,
      theme: getTheme(),
    };
  }, []);
  const [state, setState] = useState(initialState);
  const {
    address,
    chain,
    currency,
    isConnected,
    language,
    loaded,
    theme,
    vaultId,
  } = state;
  const [messageApi, messageHolder] = message.useMessage();
  const [modalAPI, modalHolder] = Modal.useModal();

  const clear = useCallback(() => {
    disconnectFromExtension()
      .then(() => {
        delToken(getVaultId());
        delVaultId();
        setState(initialState);
      })
      .catch(() => {
        messageApi.error("Disconnection failed");
      });
  }, [initialState, messageApi]);

  const connect = useCallback(() => {
    connectToExtension()
      .then((address) => {
        if (address) {
          signMessage(address).then((done) => {
            if (done) {
              messageApi.success("Successfully authenticated!");
            } else {
              messageApi.error("Authentication failed");
              clear();
            }
          });
        } else {
          messageApi.error("Connection failed");
          clear();
        }
      })
      .catch((error: Error) => {
        messageApi.error(error.message);
      });
  }, [clear, messageApi]);

  const disconnect = () => {
    modalAPI.confirm({
      title: "Are you sure you want to disconnect?",
      okText: "Yes",
      okType: "default",
      cancelText: "No",
      onOk() {
        clear();
      },
    });
  };

  const setChain = (chain: Chain) => {
    setChainStorage(chain);

    setState((prevState) => ({ ...prevState, chain }));
  };

  const setCurrency = (currency: Currency) => {
    setCurrencyStorage(currency);

    setState((prevState) => ({ ...prevState, currency }));
  };

  const setLanguage = (language: Language) => {
    setLanguageStorage(language);

    i18nInstance.changeLanguage(language);

    setState((prevState) => ({ ...prevState, language }));
  };

  const setTheme = (theme: Theme) => {
    setThemeStorage(theme);

    setState((prevState) => ({ ...prevState, theme }));
  };

  const signMessage = async (address: string) => {
    try {
      const vault = await getVault();
      const { hexChainCode, publicKeyEcdsa } = vault;
      const token = getToken(publicKeyEcdsa);

      if (token) {
        setState((prevState) => ({
          ...prevState,
          address,
          isConnected: true,
          token,
          vaultId: publicKeyEcdsa,
        }));
        setVaultId(publicKeyEcdsa);
        return true;
      }

      const nonce = hexlify(randomBytes(16));
      const expiryTime = new Date(Date.now() + 15 * 60 * 1000).toISOString();

      const message = JSON.stringify({
        message: "Sign into Vultisig App Store",
        nonce: nonce,
        expiresAt: expiryTime,
        address,
      });

      const signature = await signCustomMessage(message, address);

      const newToken = await getAuthToken({
        chainCodeHex: hexChainCode,
        publicKey: publicKeyEcdsa,
        signature,
        message,
      });

      setToken(publicKeyEcdsa, newToken);
      setVaultId(publicKeyEcdsa);

      setState((prevState) => ({
        ...prevState,
        address,
        isConnected: true,
        token: newToken,
        vaultId: publicKeyEcdsa,
      }));

      return true;
    } catch {
      return false;
    }
  };

  useLocalStorageWatcher(storageKeys.chain, () => {
    setChain(getChain());
  });

  useLocalStorageWatcher(storageKeys.currency, () => {
    setCurrency(getCurrency());
  });

  useLocalStorageWatcher(storageKeys.language, () => {
    setLanguage(getLanguage());
  });

  useLocalStorageWatcher(storageKeys.theme, () => {
    setTheme(getTheme());
  });

  return (
    <I18nextProvider i18n={i18nInstance}>
      <StyledProvider theme={theme}>
        <AntdProvider theme={theme}>
          <GlobalStyle />

          <AppContext.Provider
            value={{
              address,
              chain,
              connect,
              currency,
              disconnect,
              isConnected,
              language,
              setChain,
              setCurrency,
              setLanguage,
              setTheme,
              theme,
              vaultId,
            }}
          >
            {loaded ? (
              <BrowserRouter>
                <Routes>
                  <Route path={routeTree.root.path} element={<DefaultLayout />}>
                    <Route
                      element={<Navigate to={routeTree.plugins.path} replace />}
                      index
                    />
                    <Route
                      element={<PluginsPage />}
                      path={routeTree.plugins.path}
                    />
                    <Route
                      element={<PluginDetailsPage />}
                      path={routeTree.pluginDetails.path}
                    />
                    <Route element={<FaqPage />} path={routeTree.faq.path} />
                  </Route>
                  <Route
                    path={routeTree.notFound.path}
                    element={<NotFoundPage />}
                  />
                </Routes>
              </BrowserRouter>
            ) : (
              <Spin />
            )}
          </AppContext.Provider>

          {messageHolder}
          {modalHolder}
        </AntdProvider>
      </StyledProvider>
    </I18nextProvider>
  );
};
