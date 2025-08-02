import { Col, Layout, message, Modal, Row, Tabs } from "antd";
import { Button } from "components/Button";
import { PluginPolicyList } from "components/PluginPolicyList";
import { PluginReviewList } from "components/PluginReviewList";
import { Pricing } from "components/Pricing";
import { Rate } from "components/Rate";
import { Spin } from "components/Spin";
import { Stack } from "components/Stack";
import { Tag } from "components/Tag";
import { useApp } from "hooks/useApp";
import { useGoBack } from "hooks/useGoBack";
import { ChevronLeftIcon } from "icons/ChevronLeftIcon";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useTheme } from "styled-components";
import { modalHash } from "utils/constants/core";
import { routeTree } from "utils/constants/routes";
import { toCapitalizeFirst } from "utils/functions";
import { startReshareSession } from "utils/services/extension";
import {
  getPlugin,
  isPluginInstalled,
  uninstallPlugin,
} from "utils/services/marketplace";
import { Plugin } from "utils/types";

interface InitialState {
  isInstalled?: boolean;
  loading?: boolean;
  plugin?: Plugin;
}

export const PluginDetailsPage = () => {
  const initialState: InitialState = {};
  const [state, setState] = useState(initialState);
  const { isInstalled, loading, plugin } = state;
  const { id = "" } = useParams<{ id: string }>();
  const { connect, isConnected } = useApp();
  const [messageApi, messageHolder] = message.useMessage();
  const [modalAPI, modalHolder] = Modal.useModal();
  const navigate = useNavigate();
  const goBack = useGoBack();
  const colors = useTheme();
  const isMountedRef = useRef(true);

  const checkStatus = useCallback(() => {
    isPluginInstalled(id).then((isInstalled) => {
      if (isInstalled) {
        setState((prevState) => ({ ...prevState, isInstalled }));
      } else if (isMountedRef.current) {
        setTimeout(checkStatus, 1000);
      }
    });
  }, [id, isMountedRef.current]);

  const handleUninstall = () => {
    modalAPI.confirm({
      title: "Are you sure uninstall this plugin?",
      okText: "Yes",
      okType: "danger",
      cancelText: "No",
      onOk() {
        setState((prevState) => ({ ...prevState, loading: true }));

        uninstallPlugin(id)
          .then(() => {
            setState((prevState) => ({
              ...prevState,
              isInstalled: false,
              loading: false,
            }));

            messageApi.open({
              type: "success",
              content: "Plugin uninstalled successfully.",
            });
          })
          .catch(() => {
            setState((prevState) => ({ ...prevState, loading: false }));

            messageApi.open({
              type: "error",
              content: "Failed to uninstall plugin.",
            });
          });
      },
      onCancel() {},
    });
  };

  const handleInstall = () => {
    startReshareSession(id);
  };

  useEffect(() => {
    if (isInstalled === false) checkStatus();
  }, [checkStatus, isInstalled]);

  useEffect(() => {
    if (isConnected) {
      isPluginInstalled(id).then((isInstalled) => {
        setState((prevState) => ({ ...prevState, isInstalled }));
      });
    } else {
      setState((prevState) => ({ ...prevState, isInstalled: undefined }));
    }
  }, [id, isConnected]);

  useEffect(() => {
    getPlugin(id)
      .then((plugin) => {
        setState((prevState) => ({ ...prevState, plugin }));
      })
      .catch(() => {
        goBack(routeTree.plugins.path);
      });
  }, [id, goBack]);

  useEffect(() => {
    isMountedRef.current = true;

    return () => {
      isMountedRef.current = false;
    };
  }, []);

  return (
    <>
      {plugin ? (
        <Stack
          as={Layout.Content}
          $style={{
            justifyContent: "center",
            padding: "16px 0",
            position: "relative",
          }}
        >
          <Stack
            $style={{
              flexDirection: "column",
              gap: "16px",
              maxWidth: "1200px",
              padding: "0 16px",
              position: "relative",
              width: "100%",
              zIndex: "1",
            }}
          >
            <Stack>
              <Stack
                as="span"
                $style={{
                  alignItems: "center",
                  cursor: "pointer",
                  gap: "8px",
                  width: "fit-content",
                }}
                $hover={{ color: colors.textTertiary.toHex() }}
                onClick={() => goBack(routeTree.plugins.path)}
              >
                <ChevronLeftIcon fontSize={20} />
                Back to All Plugins
              </Stack>
            </Stack>
            <Row gutter={[16, 16]}>
              <Col xs={24} lg={12} xl={10}>
                <Stack
                  as="img"
                  alt={plugin.title}
                  src={`/plugins/${id}.jpg`}
                  $style={{ borderRadius: "12px", width: "100%" }}
                />
              </Col>
              <Stack
                as={Col}
                xs={24}
                lg={12}
                xl={14}
                $style={{ flexDirection: "column", gap: "16px" }}
              >
                <Stack $style={{ gap: "8px" }}>
                  <Tag
                    color="success"
                    text={toCapitalizeFirst(plugin.categoryId)}
                  />
                  {isInstalled && (
                    <Tag color="buttonPrimary" text="Installed" />
                  )}
                </Stack>
                <Stack
                  $style={{
                    flexDirection: "column",
                    flexGrow: "1",
                    gap: "8px",
                  }}
                >
                  <Stack
                    as="span"
                    $style={{
                      fontSize: "40px",
                      fontWeight: "500",
                      lineHeight: "42px",
                    }}
                  >
                    {plugin.title}
                  </Stack>
                  <Stack
                    as="span"
                    $style={{ flexGrow: "1", lineHeight: "20px" }}
                  >
                    {plugin.description}
                  </Stack>
                </Stack>
                <Stack $style={{ flexDirection: "column", gap: "24px" }}>
                  <Stack
                    $style={{
                      alignItems: "end",
                      justifyContent: "space-between",
                    }}
                  >
                    <Rate
                      count={plugin.rating.count}
                      value={plugin.rating.rate}
                    />
                    <Pricing pricing={plugin.pricing} />
                  </Stack>
                  <Stack $style={{ gap: "16px" }}>
                    {isConnected ? (
                      <>
                        {isInstalled === undefined ? (
                          <Button kind="primary" disabled loading>
                            Checking
                          </Button>
                        ) : isInstalled ? (
                          <>
                            <Button
                              loading={loading}
                              onClick={handleUninstall}
                              status="danger"
                            >
                              Uninstall
                            </Button>
                            <Button
                              disabled={loading}
                              kind="primary"
                              onClick={() =>
                                navigate(modalHash.policy, { state: true })
                              }
                            >
                              View Policy Schema
                            </Button>
                          </>
                        ) : (
                          <Button
                            kind="primary"
                            loading={loading}
                            onClick={handleInstall}
                          >
                            Install
                          </Button>
                        )}
                      </>
                    ) : (
                      <Button kind="primary" onClick={connect}>
                        Connect
                      </Button>
                    )}
                  </Stack>
                </Stack>
              </Stack>
            </Row>
            <Tabs
              items={[
                {
                  key: "1",
                  label: "Overview",
                  children: <PluginPolicyList {...plugin} />,
                },
                {
                  key: "2",
                  label: "Reviews and Ratings",
                  children: (
                    <PluginReviewList
                      isInstalled={isInstalled}
                      onInstall={handleInstall}
                      plugin={plugin}
                    />
                  ),
                },
              ]}
            />
          </Stack>
        </Stack>
      ) : (
        <Stack
          as={Layout.Content}
          $style={{
            alignItems: "center",
            flexGrow: "1",
            justifyContent: "center",
          }}
        >
          <Spin />
        </Stack>
      )}

      {messageHolder}
      {modalHolder}
    </>
  );
};
