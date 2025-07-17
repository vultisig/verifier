import { create, toBinary } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import {
  Checkbox,
  DatePicker,
  Divider,
  Form,
  FormProps,
  Input,
  List,
  Modal,
  Select,
  SelectProps,
  Spin,
  Tag,
} from "antd";
import { Button } from "components/button";
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
import { FormInput } from "styles/FormInput";
import { Stack } from "styles/Stack";
import { modalHash, scheduleFrequencyLabels } from "utils/constants/core";
import { toTimestamp } from "utils/functions";
import { signPluginPolicy } from "utils/services/extension";
import { addPluginPolicy } from "utils/services/marketplace";
import { Plugin, PluginPolicy } from "utils/types";
import { v4 as uuidv4 } from "uuid";

type FieldType = {
  frequency: ScheduleFrequency;
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
      name: schema.pluginName,
      rules: [rule],
      ...schedule(),
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

  useEffect(() => {
    if (visible) form.setFieldValue("supportedResource", 0);
  }, [form, visible]);

  useEffect(() => {
    setState((prevState) => ({
      ...prevState,
      visible: hash === modalHash.policy,
    }));
  }, [hash]);

  const resourceOptions: SelectProps["options"] = useMemo(() => {
    return schema?.supportedResources.map((resource, index) => ({
      label: resource.resourcePath?.full,
      value: index,
    }));
  }, [schema]);

  const frequencyOptions: SelectProps["options"] = useMemo(() => {
    return (
      schema?.scheduling?.supportedFrequencies?.map((value) => ({
        label: scheduleFrequencyLabels[value],
        value,
      })) || []
    );
  }, [schema]);

  return (
    <Modal
      footer={
        <Stack $gap="8px" $justifyContent="end">
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
      onCancel={() => goBack()}
      open={visible}
      title={`Configure ${schema.pluginName}`}
      centered
    >
      <Form
        autoComplete="off"
        form={form}
        layout="vertical"
        onFinish={onFinishSuccess}
        onFinishFailed={onFinishFailed}
      >
        {schema ? (
          <>
            <Stack $display="block">
              <Divider orientation="start" orientationMargin={0}>
                <Tag>{`v${schema.pluginVersion}`}</Tag>
                {schema.pluginId.capitalizeFirst()}
              </Divider>
              <Form.Item<FieldType>
                name="supportedResource"
                label="Supported Resource"
                rules={[{ required: true }]}
              >
                <FormInput as={Select} options={resourceOptions} />
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
                            label={parameterName.capitalizeFirst()}
                            name={parameterName}
                            rules={[{ required }]}
                          >
                            <FormInput as={Input} />
                          </Form.Item>
                        )
                      )}
                    </>
                  );
                }}
              </Form.Item>
            </Stack>
            {schema.scheduling?.supportsScheduling ? (
              <Stack $display="block">
                <Divider orientation="start" orientationMargin={0}>
                  Scheduling
                </Divider>
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
                        label="start date"
                        rules={[{ required: true }]}
                      >
                        <FormInput
                          as={DatePicker}
                          disabledDate={(current) => {
                            return current && current.isBefore(dayjs(), "day");
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
                        <FormInput as={Select} options={frequencyOptions} />
                      </Form.Item>
                    ) : (
                      <></>
                    );
                  }}
                </Form.Item>
              </Stack>
            ) : (
              <></>
            )}
            <Stack $display="block">
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
          <Stack $alignItems="center" $justifyContent="center">
            <Spin />
          </Stack>
        )}
      </Form>
    </Modal>
  );
};
