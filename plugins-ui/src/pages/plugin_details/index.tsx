import { Card, message, Modal, Spin } from "antd";
import { Button } from "components/button";
import { PluginPolicyList } from "components/plugin_policy_list";
import { PluginReviewList } from "components/plugin_review_list";
import { useApp } from "hooks/useApp";
import { useGoBack } from "hooks/useGoBack";
import { ChevronLeftIcon } from "icons/ChevronLeftIcon";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Container } from "styles/Container";
import { Stack } from "styles/Stack";
import { modalHash } from "utils/constants/core";
import { routeTree } from "utils/constants/routes";
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

  const handleReshareSession = () => {
    startReshareSession(id)
      .then(() => {})
      .catch(() => {});
  };

  useEffect(() => {
    if (isConnected) {
      isPluginInstalled(id)
        .then((isInstalled) => {
          setState((prevState) => ({ ...prevState, isInstalled }));
        })
        .catch(() => {});
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

  return (
    <>
      <Container $flexDirection="column" $gap="20px">
        <Stack onClick={() => goBack(routeTree.plugins.path)}>
          <Stack
            as="span"
            $alignItems="center"
            $colorHover="textLight"
            $cursor="pointer"
            $gap="8px"
          >
            <ChevronLeftIcon fontSize={20} />
            Back to All Plugins
          </Stack>
        </Stack>
        {plugin ? (
          <>
            <Stack $flexDirection="column" $gap="16px">
              <Card title={plugin.title} variant="borderless">
                {plugin.description}
              </Card>
              <Stack>
                {isConnected ? (
                  <>
                    {isInstalled === undefined ? (
                      <Button disabled loading>
                        Checking
                      </Button>
                    ) : isInstalled ? (
                      <Stack $gap="8px">
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
                          Add Policy
                        </Button>
                      </Stack>
                    ) : (
                      <Button
                        kind="primary"
                        loading={loading}
                        onClick={handleReshareSession}
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
            {isInstalled && <PluginPolicyList {...plugin} />}
            <PluginReviewList {...plugin} />
          </>
        ) : (
          <Stack $alignItems="center" $justifyContent="center" $flexGrow>
            <Spin />
          </Stack>
        )}
      </Container>

      {messageHolder}
      {modalHolder}
    </>
  );
};
