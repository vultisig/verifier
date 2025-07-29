import { Layout, Result } from "antd";
import { Button } from "components/Button";
import { Stack } from "components/Stack";
import { useGoBack } from "hooks/useGoBack";
import { routeTree } from "utils/constants/routes";

export const NotFoundPage = () => {
  const goBack = useGoBack();

  return (
    <Stack
      as={Layout}
      $style={{
        alignItems: "center",
        backgroundColor: "bgPrimary",
        justifyContent: "center",
        height: "100%",
      }}
    >
      <Result
        status="404"
        title="404"
        subTitle="Sorry, the page you visited does not exist."
        extra={
          <Button kind="primary" onClick={() => goBack(routeTree.root.path)}>
            Back Home
          </Button>
        }
      />
    </Stack>
  );
};
