import { create, toBinary } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import {
  Checkbox,
  Divider,
  Drawer,
  Form,
  FormProps,
  List,
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
  ScheduleSchema,
} from "proto/policy_pb";
import { RecipeSchema } from "proto/recipe_specification_pb";
import { Effect, RuleSchema } from "proto/rule_pb";
import { ScheduleFrequency } from "proto/scheduling_pb";
import { FC, useEffect, useMemo, useState } from "react";
import { useLocation } from "react-router-dom";
import { getVaultId } from "storage/vaultId";
import {
  modalHash,
  scheduleFrequencyLabels,
  scheduleFrequencyToSeconds,
} from "utils/constants/core";
import { toCapitalizeFirst, toTimestamp } from "utils/functions";
import { signPluginPolicy } from "utils/services/extension";
import { addPluginPolicy } from "utils/services/marketplace";
import { Plugin, PluginPolicy } from "utils/types";
import { v4 as uuidv4 } from "uuid";

type FieldType = {
  frequency: ScheduleFrequency;
  maxTxsPerWindow: number;
  rateLimitWindow: number;
  schedulingEnabled: boolean;
  startDate: Dayjs;
  startFromNextMonth: boolean;
  supportedResource: number;
} & {
  [key: string]: string;
};

interface PluginPolicyModalProps {
  onFinish: () => void;
  plugin: Plugin;
  schema: RecipeSchema;
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
  const goBack = useGoBack();

  const isFeesPlugin = useMemo(() => {
    return schema.pluginId === "vultisig-fees-feee";
  }, [schema]);

  const frequencyOptions: SelectProps["options"] = useMemo(() => {
    return (
      schema?.scheduling?.supportedFrequencies?.map((value) => ({
        label: scheduleFrequencyLabels[value],
        value,
      })) || []
    );
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
          value: { case: "fixedValue", value: values[parameterName] },
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

    const schedule = () => {
      if (schema.scheduling?.supportsScheduling) {
        const startDate = values.startFromNextMonth
          ? dayjs().add(1, "month").startOf("month")
          : values.startDate;

        return {
          schedule: create(ScheduleSchema, {
            frequency: values.frequency,
            interval: 0,
            maxExecutions: 0,
            startTime: create(TimestampSchema, toTimestamp(startDate)),
          }),
        };
      } else {
        return {};
      }
    };

    const jsonData = create(PolicySchema, {
      author: "",
      description: "",
      feePolicies,
      id: schema.pluginId,
      maxTxsPerWindow: values.maxTxsPerWindow,
      name: schema.pluginName,
      rules: [rule],
      ...schedule(),
      rateLimitWindow: values.rateLimitWindow,
      scheduleVersion: schema.scheduleVersion,
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
          .catch(() => {
            setState((prevState) => ({ ...prevState, submitting: false }));
          });
      })
      .catch(() => {
        setState((prevState) => ({ ...prevState, submitting: false }));
      });
  };

  const onFinishFailed: FormProps<FieldType>["onFinishFailed"] = (
    errorInfo
  ) => {
    console.log("Failed:", errorInfo);
  };

  const onValuesChange: FormProps<FieldType>["onValuesChange"] = (
    changedValues
  ) => {
    if ("frequency" in changedValues) {
      const isTouched = form.isFieldTouched("rateLimitWindow");

      if (!isTouched) {
        form.setFields([
          {
            name: "rateLimitWindow",
            value:
              scheduleFrequencyToSeconds[
                changedValues.frequency as ScheduleFrequency
              ],
            touched: false,
          },
        ]);
      }
    }
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
          ...() =>
            isFeesPlugin
              ? {
                  amount: "500000000",
                  recipient: "0x7d760c17d798a7A9a4c4AcAf311A02dC95972503",
                }
              : {},
        }}
        onFinish={onFinishSuccess}
        onFinishFailed={onFinishFailed}
        onValuesChange={onValuesChange}
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
              <Form.Item
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
                      {parameterCapabilities.map(
                        ({ parameterName, required }) => (
                          <Form.Item
                            key={parameterName}
                            label={toCapitalizeFirst(parameterName)}
                            name={parameterName}
                            rules={[{ required }]}
                          >
                            <Input disabled={isFeesPlugin} />
                          </Form.Item>
                        )
                      )}
                    </>
                  );
                }}
              </Form.Item>
            </Stack>
            <Stack $style={{ display: "block" }}>
              <Divider orientation="start" orientationMargin={0}>
                Scheduling
              </Divider>
              {schema.scheduling?.supportsScheduling ? (
                <>
                  <Form.Item<FieldType>
                    name="startFromNextMonth"
                    valuePropName="checked"
                  >
                    <Checkbox>Start from the beginning of next month</Checkbox>
                  </Form.Item>
                  <Form.Item
                    shouldUpdate={(prevValues, currentValues) =>
                      prevValues.startFromNextMonth !==
                      currentValues.startFromNextMonth
                    }
                    noStyle
                  >
                    {({ getFieldsValue }) => {
                      const { startFromNextMonth = false } = getFieldsValue();

                      return startFromNextMonth ? (
                        <></>
                      ) : (
                        <Form.Item<FieldType>
                          name="startDate"
                          label="Start Date"
                          rules={[{ required: true }]}
                        >
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
                        </Form.Item>
                      );
                    }}
                  </Form.Item>
                  <Form.Item<FieldType>
                    name="schedulingEnabled"
                    valuePropName="checked"
                  >
                    <Checkbox>Enable scheduled execution</Checkbox>
                  </Form.Item>
                  <Form.Item
                    shouldUpdate={(prevValues, currentValues) =>
                      prevValues.schedulingEnabled !==
                      currentValues.schedulingEnabled
                    }
                    noStyle
                  >
                    {({ getFieldsValue }) => {
                      const { schedulingEnabled = false } = getFieldsValue();

                      return schedulingEnabled ? (
                        <Form.Item<FieldType>
                          name="frequency"
                          label="Frequency"
                          rules={[{ required: true }]}
                          help={`Max ${schema.scheduling?.maxScheduledExecutions} scheduled executions`}
                        >
                          <Select options={frequencyOptions} />
                        </Form.Item>
                      ) : (
                        <></>
                      );
                    }}
                  </Form.Item>
                </>
              ) : (
                <></>
              )}
              <Form.Item<FieldType>
                name="maxTxsPerWindow"
                label="Max Txs Per Window"
              >
                <InputNumber min={1} />
              </Form.Item>
              <Form.Item<FieldType>
                name="rateLimitWindow"
                label="Rate Limit Window (seconds)"
              >
                <InputNumber min={1} />
              </Form.Item>
            </Stack>
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
          <Stack $style={{ alignItems: "center", justifyContent: "center" }}>
            <Spin />
          </Stack>
        )}
      </Form>
    </Drawer>
  );
};
