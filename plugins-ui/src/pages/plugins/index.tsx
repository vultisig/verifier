import { Col, Empty, Form, Layout, Row, Select, SelectProps, Spin } from "antd";
import { SearchInput } from "components/InputSearch";
import { PageHeading } from "components/PageHeading";
import { PluginItem } from "components/PluginItem";
import { Stack } from "components/Stack";
import { useFilterParams } from "hooks/useFilterParams";
import { debounce } from "lodash-es";
import { useCallback, useEffect, useState } from "react";
import { getPluginCategories, getPlugins } from "utils/services/marketplace";
import { Plugin, PluginFilters } from "utils/types";

type InitialState = {
  categoryOptions: NonNullable<SelectProps["options"]>;
  loading: boolean;
  plugins: Plugin[];
  sortOptions: NonNullable<SelectProps["options"]>;
};

export const PluginsPage = () => {
  const initialState: InitialState = {
    categoryOptions: [],
    loading: true,
    plugins: [],
    sortOptions: [
      { value: "-created_at", label: "Newest" },
      { value: "created_at", label: "Oldest" },
    ],
  };
  const [state, setState] = useState(initialState);
  const { categoryOptions, loading, plugins, sortOptions } = state;
  const [form] = Form.useForm<PluginFilters>();
  const { filters, setFilters } = useFilterParams<PluginFilters>();

  const fetchPlugins = useCallback(
    debounce((skip: number, filters: PluginFilters) => {
      setState((prevState) => ({ ...prevState, loading: true }));

      getPlugins(skip, filters)
        .then(({ plugins }) => {
          setState((prevState) => ({ ...prevState, loading: false, plugins }));
        })
        .catch(() => {
          setState((prevState) => ({ ...prevState, loading: false }));
        });
    }, 500),
    []
  );

  const handleChange = (_: unknown, values: PluginFilters) => {
    setFilters(values);
  };

  useEffect(() => fetchPlugins(0, filters), [fetchPlugins, filters]);

  useEffect(() => {
    getPluginCategories()
      .then((categories) => {
        const categoryOptions: SelectProps["options"] = categories.map(
          ({ id, name }) => ({ value: id, label: name })
        );

        setState((prevState) => ({ ...prevState, categoryOptions }));
      })
      .catch(() => {});
  }, []);

  return (
    <Stack
      as={Layout.Content}
      $style={{ flexGrow: "1", justifyContent: "center", padding: "30px 0" }}
    >
      <Stack
        $style={{
          flexDirection: "column",
          gap: "16px",
          maxWidth: "1200px",
          padding: "0 16px",
          width: "100%",
        }}
      >
        <PageHeading
          description="Discover and install plugins to enhance your experience."
          title="Plugins Marketplace"
        />
        <Form
          form={form}
          initialValues={filters}
          layout="vertical"
          onValuesChange={handleChange}
        >
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} xl={8}>
              <Form.Item<PluginFilters> name="term" noStyle>
                <SearchInput placeholder="Search by" />
              </Form.Item>
            </Col>
            <Col xs={12} sm={6} xl={{ span: 4, offset: 8 }}>
              <Form.Item<PluginFilters> name="category" noStyle>
                <Stack
                  as={Select}
                  options={categoryOptions}
                  placeholder="Category"
                  allowClear
                  $style={{ height: "44px" }}
                />
              </Form.Item>
            </Col>
            <Col xs={12} sm={6} xl={4}>
              <Form.Item<PluginFilters> name="sort" noStyle>
                <Stack
                  as={Select}
                  options={sortOptions}
                  placeholder="Sort"
                  allowClear
                  $style={{ height: "44px" }}
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
        {loading ? (
          <Stack
            $style={{
              alignItems: "center",
              flexGrow: "1",
              justifyContent: "center",
            }}
          >
            <Spin />
          </Stack>
        ) : plugins.length ? (
          <Row gutter={[16, 16]}>
            {plugins.map((plugin) => (
              <Col key={plugin.id} xs={24} sm={12} xl={8}>
                <PluginItem {...plugin} />
              </Col>
            ))}
          </Row>
        ) : (
          <Empty />
        )}
      </Stack>
    </Stack>
  );
};
