import { Collapse } from "antd";
import { Divider } from "components/Divider";
import { Stack } from "components/Stack";
import { Fragment } from "react";
import styled from "styled-components";

const StyledCollapse = styled(Collapse)`
  &.ant-collapse {
    .ant-collapse-item {
      .ant-collapse-header {
        align-items: center;
        padding: 0;

        .ant-collapse-header-text {
          font-size: 16px;
          font-weight: 500;
          line-height: 24px;
        }
      }

      .ant-collapse-content {
        .ant-collapse-content-box {
          color: ${({ theme }) => theme.textSecondary.toHex()};
          padding: 24px 0 0;
        }
      }
    }
  }
`;

const text =
  "Maecenas in porttitor consequat aenean. In nulla cursus pulvinar at lacus ultricies et nulla. Non porta arcu vehicula rhoncus. Habitant integer lectus elit proin. Etiam morbi nunc pretium vestibulum sed convallis etiam. Pulvinar vitae porttitor elementum eget mattis sagittis facilisi magna. Et pulvinar pretium vitae odio non ultricies maecenas id. Non nibh scelerisque in facilisis tincidunt viverra fermentum sem. Quam varius pretium vitae neque. Senectus lectus ultricies nibh eget.";

export const FaqPage = () => {
  const data = [
    {
      heading: "General",
      items: [
        {
          answer: text,
          question: "How does it work?",
        },
        {
          answer: text,
          question: "How to install?",
        },
        {
          answer: text,
          question: "Is it safe? I don’t want to risk my funds.",
        },
        {
          answer: text,
          question: "Are apps audited?",
        },
      ],
    },
    {
      heading: "Developers",
      items: [
        {
          answer: text,
          question: "How does it work?",
        },
        {
          answer: text,
          question: "How to install?",
        },
        {
          answer: text,
          question: "Is it safe? I don’t want to risk my funds.",
        },
        {
          answer: text,
          question: "Are apps audited?",
        },
      ],
    },
  ];

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
      <Stack $style={{ flexDirection: "column", gap: "72px" }}>
        {data.map(({ heading, items }, index) => (
          <Stack key={index} $style={{ flexDirection: "column", gap: "24px" }}>
            <Stack
              as="span"
              $style={{
                fontSize: "22px",
                fontWeight: "500",
                lineHeight: "24px",
              }}
            >
              {heading}
            </Stack>
            {items.map(({ answer, question }, index) => (
              <Fragment key={index}>
                {index > 0 && <Divider />}
                <StyledCollapse
                  bordered={false}
                  items={[{ key: "1", label: question, children: answer }]}
                  expandIconPosition="right"
                  ghost
                />
              </Fragment>
            ))}
          </Stack>
        ))}
      </Stack>
    </Stack>
  );
};
