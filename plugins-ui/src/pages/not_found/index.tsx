import { Layout, Result } from "antd";
import { Button } from "components/button";
import { useGoBack } from "hooks/useGoBack";
import { Stack } from "styles/Stack";
import { routeTree } from "utils/constants/routes";

export const NotFoundPage = () => {
  const goBack = useGoBack();

  return (
    <Stack
      as={Layout}
      $alignItems="center"
      $backgroundColor="backgroundPrimary"
      $justifyContent="center"
      $fullHeight
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
