import { Collapse } from "antd";
import { Divider } from "components/Divider";
import { Spin } from "components/Spin";
import { Stack } from "components/Stack";
import { useEffect, useState } from "react";

type InitialState = {
  data: string[];
  loading: boolean;
};

export const FaqPage = () => {
  const initialState: InitialState = { data: [], loading: true };
  const [state, setState] = useState(initialState);
  const { loading } = state;

  useEffect(() => {
    setTimeout(() => {
      setState((prevState) => ({ ...prevState, loading: false }));
    }, 1000);

    // getFAQ()
    //   .then(({ data }) => {
    //     setState((prevState) => ({ ...prevState, loading: false, data }));
    //   })
    //   .catch(() => {
    //     setState((prevState) => ({ ...prevState, loading: false }));
    //   });
  }, []);

  const text =
    "Maecenas in porttitor consequat aenean. In nulla cursus pulvinar at lacus ultricies et nulla. Non porta arcu vehicula rhoncus. Habitant integer lectus elit proin. Etiam morbi nunc pretium vestibulum sed convallis etiam. Pulvinar vitae porttitor elementum eget mattis sagittis facilisi magna. Et pulvinar pretium vitae odio non ultricies maecenas id. Non nibh scelerisque in facilisis tincidunt viverra fermentum sem. Quam varius pretium vitae neque. Senectus lectus ultricies nibh eget.";

  return (
    <Stack
      $style={{
        flexDirection: "column",
        gap: "32px",
        maxWidth: "768px",
        padding: "48px 16px",
        width: "100%",
      }}
    >
      <Stack
        as="span"
        $style={{
          fontSize: "40px",
          fontWeight: "500",
          justifyContent: "center",
          lineHeight: "42px",
        }}
      >
        FAQ
      </Stack>
      {loading ? (
        <Stack
          as={Spin}
          $style={{
            alignItems: "center",
            flexGrow: "1",
            justifyContent: "center",
          }}
        />
      ) : (
        <Stack $style={{ flexDirection: "column", gap: "72px" }}>
          <Stack $style={{ flexDirection: "column", gap: "24px" }}>
            <Stack
              as="span"
              $style={{
                fontSize: "22px",
                fontWeight: "500",
                lineHeight: "24px",
              }}
            >
              General
            </Stack>
            <Collapse
              bordered={false}
              defaultActiveKey={["1"]}
              items={[
                {
                  key: "1",
                  label: "How does it work?",
                  children: text,
                },
              ]}
              ghost
            />
            <Divider />
            <Collapse
              bordered={false}
              defaultActiveKey={["1"]}
              items={[
                {
                  key: "1",
                  label: "How does it work?",
                  children: text,
                },
              ]}
              ghost
            />
          </Stack>
          <Stack $style={{ flexDirection: "column", gap: "24px" }}>
            <Stack
              as="span"
              $style={{
                fontSize: "22px",
                fontWeight: "500",
                lineHeight: "24px",
              }}
            >
              Developers
            </Stack>
          </Stack>
        </Stack>
      )}
    </Stack>
  );
};
