import { create, toBinary } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import {
  Col,
  Divider,
  Drawer,
  Form,
  FormProps,
  List,
  message,
  Row,
  SelectProps,
  Tag,
} from "antd";
import { Button } from "components/Button";
import { DatePicker } from "components/DatePicker";
import { Input } from "components/Input";
import { InputNumber } from "components/InputNumber";
import { Select } from "components/Select";
import { Spin } from "components/Spin";
import { Stack } from "components/Stack";
import dayjs, { Dayjs } from "dayjs";
import { useGoBack } from "hooks/useGoBack";
import { ConstraintSchema } from "proto/constraint_pb";
import { ParameterConstraintSchema } from "proto/parameter_constraint_pb";
import {
  BillingFrequency,
  FeePolicySchema,
  FeeType,
  PolicySchema,
} from "proto/policy_pb";
import { RecipeSchema } from "proto/recipe_specification_pb";
import { Effect, RuleSchema } from "proto/rule_pb";
import { FC, ReactNode, useEffect, useMemo, useState } from "react";
import { useLocation } from "react-router-dom";
import { getVaultId } from "storage/vaultId";
import { modalHash } from "utils/constants/core";
import { toCapitalizeFirst, toTimestamp } from "utils/functions";
import { signPluginPolicy } from "utils/services/extension";
import { addPluginPolicy } from "utils/services/marketplace";
import { Configuration, Plugin, PluginPolicy } from "utils/types";
import { v4 as uuidv4 } from "uuid";

type FieldType = {
  maxTxsPerWindow: number;
  rateLimitWindow: number;
  supportedResource: number;
} & {
  [key: string]: string | Dayjs;
};

interface PluginPolicyModalProps {
  onFinish: () => void;
  plugin: Plugin;
  schema: Omit<RecipeSchema, "configuration"> & {
    configuration?: Configuration;
  };
}

interface InitialState {
  submitting?: boolean;
  visible?: boolean;
}

export const PluginPolicyModal: FC<PluginPolicyModalProps> = ({
  onFinish,
  plugin,
  schema,
}) => {
  const initialState: InitialState = {};
  const [state, setState] = useState(initialState);
  const { submitting, visible } = state;
  const { hash } = useLocation();
  const [form] = Form.useForm<FieldType>();
  const [messageApi, messageHolder] = message.useMessage();
  const goBack = useGoBack();

  const isFeesPlugin = useMemo(() => {
    return schema.pluginId === "vultisig-fees-feee";
  }, [schema]);

  const resourceOptions: SelectProps["options"] = useMemo(() => {
    return schema?.supportedResources.map((resource, index) => ({
      label: resource.resourcePath?.full,
      value: index,
    }));
  }, [schema]);

  const onFinishSuccess: FormProps<FieldType>["onFinish"] = (values) => {
    setState((prevState) => ({ ...prevState, submitting: true }));

    const { parameterCapabilities, resourcePath } =
      schema.supportedResources[values.supportedResource];

    const feePolicies = plugin.pricing.map((price) => {
      let frequency = BillingFrequency.BILLING_FREQUENCY_UNSPECIFIED;
      let type = FeeType.FEE_TYPE_UNSPECIFIED;

      switch (price.frequency) {
        case "daily":
          frequency = BillingFrequency.DAILY;
          break;
        case "weekly":
          frequency = BillingFrequency.WEEKLY;
          break;
        case "biweekly":
          frequency = BillingFrequency.BIWEEKLY;
          break;
        case "monthly":
          frequency = BillingFrequency.MONTHLY;
          break;
      }

      switch (price.type) {
        case "once":
          type = FeeType.ONCE;
          break;
        case "recurring":
          type = FeeType.RECURRING;
          break;
        case "per-tx":
          type = FeeType.TRANSACTION;
          break;
      }

      return create(FeePolicySchema, {
        amount: BigInt(price.amount),
        description: "",
        frequency,
        id: uuidv4(),
        startDate: create(TimestampSchema, toTimestamp(dayjs())),
        type,
      });
    });

    const parameterConstraints = parameterCapabilities.map(
      ({ parameterName, required, supportedTypes }) => {
        const [type] = supportedTypes;

        const constraint = create(ConstraintSchema, {
          denominatedIn:
            resourcePath?.chainId.toLowerCase() === "ethereum" ? "wei" : "",
          period: "",
          required,
          type,
          value: { case: "fixedValue", value: values[parameterName] as string },
        });

        const parameterConstraint = create(ParameterConstraintSchema, {
          constraint: constraint,
          parameterName,
        });

        return parameterConstraint;
      }
    );

    const rule = create(RuleSchema, {
      constraints: {},
      description: "",
      effect: Effect.ALLOW,
      id: "",
      parameterConstraints,
      resource: resourcePath?.full,
    });

    const configuration = () => {
      if (schema.configuration) {
        const configuration: Record<string, any> = {};

        Object.entries(schema.configuration.properties).forEach(
          ([key, field]) => {
            if (values[key]) {
              switch (field.format) {
                case "date-time": {
                  configuration[key] = (values[key] as Dayjs).utc().format();
                  break;
                }
                default: {
                  configuration[key] = values[key];
                  break;
                }
              }
            }
          }
        );

        return { configuration };
      } else {
        return {};
      }
    };

    const jsonData = create(PolicySchema, {
      author: "",
      ...configuration(),
      description: "",
      feePolicies,
      id: schema.pluginId,
      maxTxsPerWindow: values.maxTxsPerWindow,
      name: schema.pluginName,
      rules: [rule],
      rateLimitWindow: values.rateLimitWindow,
      version: schema.pluginVersion,
    });

    const binaryData = toBinary(PolicySchema, jsonData);

    const base64Data = Buffer.from(binaryData).toString("base64");

    const finalData: PluginPolicy = {
      active: true,
      id: uuidv4(),
      pluginId: plugin.id,
      pluginVersion: String(schema.pluginVersion),
      policyVersion: 0,
      publicKey: getVaultId(),
      recipe: base64Data,
    };

    signPluginPolicy(finalData)
      .then((signature) => {
        addPluginPolicy({ ...finalData, signature })
          .then(() => {
            setState((prevState) => ({ ...prevState, submitting: false }));

            form.resetFields();

            goBack();

            onFinish();
          })
          .catch((error: Error) => {
            messageApi.error(error.message);

            setState((prevState) => ({ ...prevState, submitting: false }));
          });
      })
      .catch((error: Error) => {
        messageApi.error(error.message);

        setState((prevState) => ({ ...prevState, submitting: false }));
      });
  };

  useEffect(() => {
    if (hash === modalHash.policy) {
      setState((prevState) => ({ ...prevState, visible: true }));
    } else if (visible) {
      setState((prevState) => ({ ...prevState, visible: false }));

      form.resetFields();
    }
  }, [form, hash, visible]);

  return (
    <>
      <Drawer
        footer={
          <Stack $style={{ gap: "8px", justifyContent: "end" }}>
            <Button disabled={submitting} onClick={() => goBack()}>
              Cancel
            </Button>
            <Button
              kind="primary"
              loading={submitting}
              onClick={() => form.submit()}
            >
              Submit
            </Button>
          </Stack>
        }
        maskClosable={false}
        onClose={() => goBack()}
        open={visible}
        style={{ minWidth: 400 }}
        title={`Configure ${schema.pluginName}`}
        width={992}
      >
        <Form
          autoComplete="off"
          form={form}
          layout="vertical"
          initialValues={{
            maxTxsPerWindow: 2,
            supportedResource: 0,
            ...(isFeesPlugin && {
              amount: "500000000", // Fee Max
              recipient: "1", // Vultisig Treasury
              token: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC
            }),
          }}
          onFinish={onFinishSuccess}
        >
          {schema ? (
            <>
              <Stack $style={{ display: "block" }}>
                <Divider orientation="start" orientationMargin={0}>
                  <Tag>{`v${schema.pluginVersion}`}</Tag>
                  {toCapitalizeFirst(schema.pluginId)}
                </Divider>
                <Form.Item<FieldType>
                  name="supportedResource"
                  label="Supported Resource"
                  rules={[{ required: true }]}
                >
                  <Select disabled={isFeesPlugin} options={resourceOptions} />
                </Form.Item>
                <Form.Item<FieldType>
                  shouldUpdate={(prevValues, currentValues) =>
                    prevValues.supportedResource !==
                    currentValues.supportedResource
                  }
                  noStyle
                >
                  {({ getFieldsValue }) => {
                    const { supportedResource = 0 } = getFieldsValue();
                    const { parameterCapabilities, resourcePath } =
                      schema.supportedResources[supportedResource];

                    return (
                      <>
                        <Tag>Chain: {resourcePath?.chainId}</Tag>
                        <Tag>Protocol: {resourcePath?.protocolId}</Tag>
                        <Tag>Function: {resourcePath?.functionId}</Tag>
                        <Divider orientation="start" orientationMargin={0}>
                          Parameters
                        </Divider>
                        <Row gutter={16}>
                          {parameterCapabilities.map(
                            ({ parameterName, required }) => (
                              <Col key={parameterName} xs={24} md={12} lg={8}>
                                <Form.Item
                                  label={toCapitalizeFirst(parameterName)}
                                  name={parameterName}
                                  rules={[{ required }]}
                                >
                                  <Input disabled={isFeesPlugin} />
                                </Form.Item>
                              </Col>
                            )
                          )}
                        </Row>
                      </>
                    );
                  }}
                </Form.Item>
              </Stack>
              <Stack $style={{ display: "block" }}>
                <Divider orientation="start" orientationMargin={0}>
                  Scheduling
                </Divider>
                <Row gutter={16}>
                  <Col xs={24} md={12} lg={8}>
                    <Form.Item<FieldType>
                      name="maxTxsPerWindow"
                      label="Max Txs Per Window"
                    >
                      <InputNumber min={1} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12} lg={8}>
                    <Form.Item<FieldType>
                      name="rateLimitWindow"
                      label="Rate Limit Window (seconds)"
                    >
                      <InputNumber min={1} />
                    </Form.Item>
                  </Col>
                </Row>
              </Stack>
              {schema.configuration ? (
                <Stack $style={{ display: "block" }}>
                  <Divider orientation="start" orientationMargin={0}>
                    Configuration
                  </Divider>
                  {Object.entries(schema.configuration.properties).map(
                    ([key, field]) => {
                      const required =
                        schema.configuration?.required.includes(key);

                      let element: ReactNode;

                      if (field.enum) {
                        element = (
                          <Select
                            disabled={isFeesPlugin}
                            options={field.enum.map((value) => ({
                              label: toCapitalizeFirst(value),
                              value,
                            }))}
                          />
                        );
                      } else {
                        switch (field.format) {
                          case "date-time": {
                            element = (
                              <DatePicker
                                disabledDate={(current) => {
                                  return (
                                    current && current.isBefore(dayjs(), "day")
                                  );
                                }}
                                format="YYYY-MM-DD HH:mm"
                                showNow={false}
                                showTime={{
                                  disabledHours: () => {
                                    const nextHour = dayjs()
                                      .add(1, "hour")
                                      .startOf("hour")
                                      .hour();

                                    return Array.from(
                                      { length: nextHour },
                                      (_, i) => i
                                    );
                                  },
                                  format: "HH",
                                  showMinute: false,
                                  showSecond: false,
                                }}
                              />
                            );
                            break;
                          }
                          default: {
                            element = <Input />;
                            break;
                          }
                        }
                      }

                      return (
                        <Form.Item
                          key={key}
                          name={key}
                          label={toCapitalizeFirst(key)}
                          rules={[{ required }]}
                        >
                          {element}
                        </Form.Item>
                      );
                    }
                  )}
                </Stack>
              ) : (
                <></>
              )}
              <Stack $style={{ display: "block" }}>
                <Divider orientation="start" orientationMargin={0}>
                  Requirements
                </Divider>
                <List
                  dataSource={[
                    `Min Vultisig Version: ${schema.requirements?.minVultisigVersion}`,
                    `Supported Chains: ${schema.requirements?.supportedChains.join(
                      ", "
                    )}`,
                  ]}
                  renderItem={(item) => <List.Item>{item}</List.Item>}
                />
              </Stack>
            </>
          ) : (
            <Spin />
          )}
        </Form>
      </Drawer>

      {messageHolder}
    </>
  );
};
