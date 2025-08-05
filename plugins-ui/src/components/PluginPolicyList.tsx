import { fromBinary } from "@bufbuild/protobuf";
import { base64Decode } from "@bufbuild/protobuf/wire";
import { List, message, Modal, Table, TableProps } from "antd";
import { Button } from "components/Button";
import { MiddleTruncate } from "components/MiddleTruncate";
import { PluginPolicyModal } from "components/PluginPolicyModal";
import { Stack } from "components/Stack";
import { TrashIcon } from "icons/TrashIcon";
import { Policy, PolicySchema } from "proto/policy_pb";
import { RecipeSchema } from "proto/recipe_specification_pb";
import { FC, useCallback, useEffect, useState } from "react";
import { toCapitalizeFirst, toNumeralFormat } from "utils/functions";
import {
  delPluginPolicy,
  getPluginPolicies,
  getRecipeSpecification,
} from "utils/services/marketplace";
import { Configuration, Plugin, PluginPolicy } from "utils/types";

interface ParsedPluginPolicy extends PluginPolicy {
  parsedRecipe: Policy;
}

interface InitialState {
  loading: boolean;
  policies: ParsedPluginPolicy[];
  schema?: Omit<RecipeSchema, "configuration"> & {
    configuration?: Configuration;
  };
  totalCount: number;
}

export const PluginPolicyList: FC<Plugin> = (plugin) => {
  const initialState: InitialState = {
    loading: true,
    policies: [],
    totalCount: 0,
  };
  const [state, setState] = useState(initialState);
  const { loading, policies, schema } = state;
  const [messageApi, messageHolder] = message.useMessage();
  const [modalAPI, modalHolder] = Modal.useModal();
  const { id } = plugin;

  const columns: TableProps<ParsedPluginPolicy>["columns"] = [
    {
      title: "Row",
      key: "row",
      render: (_, __, index) => index + 1,
      align: "center",
      width: 20,
    },
    Table.EXPAND_COLUMN,
    {
      title: "Resource",
      dataIndex: "parsedRecipe",
      key: "resource",
      render: ({ rules: [{ resource }] }: Policy) => resource,
    },
    {
      title: "Action",
      key: "action",
      render: (_, record) => (
        <Button
          icon={<TrashIcon />}
          kind="link"
          onClick={() => handleDelete(record)}
          status="danger"
        />
      ),
      align: "center",
      width: 80,
    },
  ];

  const fetchPolicies = useCallback(
    (skip: number) => {
      setState((prevState) => ({ ...prevState, loading: true }));

      getPluginPolicies(id, skip)
        .then(({ policies, totalCount }) => {
          setState((prevState) => ({
            ...prevState,
            loading: false,
            policies:
              policies?.map((policy) => {
                const decoded = base64Decode(policy.recipe);
                const parsedRecipe = fromBinary(PolicySchema, decoded);

                return { ...policy, parsedRecipe };
              }) || [],
            totalCount,
          }));
        })
        .catch(() => {
          setState((prevState) => ({ ...prevState, loading: false }));
        });
    },
    [id]
  );

  const handleCreate = () => {
    messageApi.success("Policy created successfully.");

    fetchPolicies(0);
  };

  const handleDelete = ({ id, signature }: ParsedPluginPolicy) => {
    if (signature) {
      modalAPI.confirm({
        title: "Are you sure delete this policy?",
        okText: "Yes",
        okType: "danger",
        cancelText: "No",
        onOk() {
          setState((prevState) => ({ ...prevState, loading: true }));

          delPluginPolicy(id, signature)
            .then(() => {
              messageApi.success("Policy deleted successfully.");

              fetchPolicies(0);
            })
            .catch(() => {
              setState((prevState) => ({ ...prevState, loading: false }));
            });
        },
        onCancel() {},
      });
    } else {
      messageApi.error("Unable to delete policy: signature is missing.");
    }
  };

  useEffect(() => fetchPolicies(0), [id, fetchPolicies]);

  useEffect(() => {
    getRecipeSpecification(id)
      .then((schema) => {
        setState((prevState) => ({ ...prevState, schema }));
      })
      .catch(() => {});
  }, [id]);

  return (
    <>
      <Table
        columns={columns}
        dataSource={policies}
        expandable={{
          expandedRowRender: ({
            parsedRecipe: {
              maxTxsPerWindow,
              rateLimitWindow,
              rules: [{ parameterConstraints }],
            },
          }) => (
            <List
              dataSource={[
                ...parameterConstraints.map(
                  ({ constraint, parameterName }) => ({
                    case: constraint?.value.case,
                    name: parameterName,
                    value: constraint?.value.value,
                  })
                ),
                {
                  case: undefined,
                  name: "Max Txs Per Window",
                  value: maxTxsPerWindow
                    ? toNumeralFormat(maxTxsPerWindow)
                    : "-",
                },
                {
                  case: "seconds",
                  name: "Rate Limit Window",
                  value: rateLimitWindow
                    ? toNumeralFormat(rateLimitWindow)
                    : "-",
                },
              ]}
              grid={{
                gutter: [16, 16],
                xs: 1,
                sm: 1,
                md: 2,
                lg: 2,
                xl: 4,
                xxl: 4,
              }}
              renderItem={(record) => {
                const value = String(record.value);
                const description = value.startsWith("0x") ? (
                  <MiddleTruncate text={value} />
                ) : (
                  value
                );

                return (
                  <List.Item key={record.name} style={{ margin: 0 }}>
                    <List.Item.Meta
                      title={
                        record.case ? (
                          <Stack $style={{ alignItems: "center", gap: "4px" }}>
                            <span>{toCapitalizeFirst(record.name)}</span>
                            <small>{`(${record.case})`}</small>
                          </Stack>
                        ) : (
                          toCapitalizeFirst(record.name)
                        )
                      }
                      description={description}
                    />
                  </List.Item>
                );
              }}
            />
          ),
        }}
        loading={loading}
        rowKey="id"
        size="small"
      />

      {!!schema && (
        <PluginPolicyModal
          onFinish={handleCreate}
          plugin={plugin}
          schema={schema}
        />
      )}

      {messageHolder}
      {modalHolder}
    </>
  );
};
