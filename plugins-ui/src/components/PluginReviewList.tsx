import {
  Card,
  Col,
  Divider,
  Empty,
  Form,
  FormProps,
  Input,
  Rate,
  Row,
  Spin,
  Tag,
} from "antd";
import { Button } from "components/Button";
import { MiddleTruncate } from "components/MiddleTruncate";
import { Stack } from "components/Stack";
import dayjs from "dayjs";
import { useApp } from "hooks/useApp";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { addPluginReview, getPluginReviews } from "utils/services/marketplace";
import { Plugin, Review, ReviewForm } from "utils/types";

interface InitialState {
  loading: boolean;
  reviews: Review[];
  submitting?: boolean;
  totalCount: number;
}

export const PluginReviewList: FC<Plugin> = (plugin) => {
  const initialState: InitialState = {
    loading: true,
    reviews: [],
    totalCount: 0,
  };
  const [state, setState] = useState(initialState);
  const { loading, reviews, submitting } = state;
  const { address, connect, isConnected } = useApp();
  const [form] = Form.useForm<ReviewForm>();
  const { id } = plugin;

  const fetchReviews = useCallback(
    (skip: number) => {
      setState((prevState) => ({ ...prevState, loading: true }));

      getPluginReviews(id, skip)
        .then(({ reviews, totalCount }) => {
          setState((prevState) => ({
            ...prevState,
            loading: false,
            reviews,
            totalCount,
          }));
        })
        .catch(() => {
          setState((prevState) => ({ ...prevState, loading: false }));
        });
    },
    [id]
  );

  const onFinishSuccess: FormProps<ReviewForm>["onFinish"] = (values) => {
    if (address) {
      setState((prevState) => ({ ...prevState, submitting: true }));

      addPluginReview(id, { ...values, address })
        .then(() => {
          setState((prevState) => ({ ...prevState, submitting: false }));

          form.resetFields();

          fetchReviews(0);
        })
        .catch(() => {
          setState((prevState) => ({ ...prevState, submitting: false }));
        });
    }
  };

  const onFinishFailed: FormProps<ReviewForm>["onFinishFailed"] = (
    errorInfo
  ) => {
    console.log("Failed:", errorInfo);
  };

  useEffect(() => fetchReviews(0), [id, fetchReviews]);

  const averageRating = useMemo(() => {
    return reviews.length
      ? parseFloat(
          (
            reviews.reduce((sum, item) => sum + item.rating, 0) / reviews.length
          ).toFixed(1)
        )
      : 0;
  }, [reviews]);

  return (
    <>
      <Divider>Reviews and Ratings</Divider>
      <Row gutter={[16, 16]}>
        <Col xs={24}>
          <Form
            autoComplete="off"
            form={form}
            layout="vertical"
            onFinish={onFinishSuccess}
            onFinishFailed={onFinishFailed}
          >
            <Card
              extra={
                <Form.Item<ReviewForm>
                  name="rating"
                  rules={[{ required: true }]}
                  noStyle
                >
                  <Rate count={5} />
                </Form.Item>
              }
              title="Leave a review"
              variant="borderless"
            >
              <Form.Item<ReviewForm>
                name="comment"
                rules={[{ required: true }]}
              >
                <Input.TextArea rows={4} />
              </Form.Item>
              <Form.Item shouldUpdate noStyle>
                {() => {
                  const hasErrors = form
                    .getFieldsError()
                    .some(({ errors }) => errors.length > 0);
                  const isTouched = form.isFieldsTouched(true);

                  return (
                    <Stack $style={{ justifyContent: "end" }}>
                      {isConnected ? (
                        <Button
                          disabled={!isTouched || hasErrors}
                          kind="primary"
                          loading={submitting}
                          type="submit"
                        >
                          Submit
                        </Button>
                      ) : (
                        <Button kind="primary" onClick={connect}>
                          Connect
                        </Button>
                      )}
                    </Stack>
                  );
                }}
              </Form.Item>
            </Card>
          </Form>
        </Col>
        {loading ? (
          <Stack
            as={Col}
            xs={24}
            $style={{ alignItems: "center", justifyContent: "center" }}
          >
            <Spin />
          </Stack>
        ) : reviews.length ? (
          <>
            <Col xs={24}>
              <Card
                extra={
                  <Stack $style={{ alignItems: "center" }}>
                    <Tag>{`${averageRating} / 5`}</Tag>
                    <Rate count={5} value={averageRating} disabled />
                  </Stack>
                }
                title={`${reviews.length} Reviews`}
                variant="borderless"
                styles={{
                  body: { display: "none" },
                  header: { border: "none" },
                }}
              />
            </Col>
            {reviews.map(({ address, comment, createdAt, id, rating }) => (
              <Col key={id} xs={24} md={12} lg={8}>
                <Card
                  extra={dayjs(createdAt).format("YYYY/MM/DD HH:mm")}
                  title={<Rate count={5} value={rating} disabled />}
                  variant="borderless"
                >
                  {comment}
                  <Divider />
                  <MiddleTruncate text={address} width="200px" />
                </Card>
              </Col>
            ))}
          </>
        ) : (
          <Col xs={24}>
            <Empty />
          </Col>
        )}
      </Row>
    </>
  );
};
