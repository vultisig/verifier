import { Col, Divider, Empty, Form, Row, SelectProps } from "antd";
import { SearchInput } from "components/InputSearch";
import { PageHeading } from "components/PageHeading";
import { PluginItem } from "components/PluginItem";
import { Select } from "components/Select";
import { Spin } from "components/Spin";
import { Stack } from "components/Stack";
import { useFilterParams } from "hooks/useFilterParams";
import { debounce } from "lodash-es";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTheme } from "styled-components";
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
  const colors = useTheme();

  const fetchPlugins = useCallback((skip: number, filters: PluginFilters) => {
    setState((prevState) => ({ ...prevState, loading: true }));

    getPlugins(skip, filters)
      .then(({ plugins }) => {
        setState((prevState) => ({ ...prevState, loading: false, plugins }));
      })
      .catch(() => {
        setState((prevState) => ({ ...prevState, loading: false }));
      });
  }, []);

  const handleChange = (_: unknown, values: PluginFilters) => {
    setFilters(values);
  };

  const debouncedFetchPlugins = useMemo(
    () => debounce(fetchPlugins, 500),
    [fetchPlugins]
  );

  useEffect(
    () => debouncedFetchPlugins(0, filters),
    [debouncedFetchPlugins, filters]
  );

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
      $style={{
        flexDirection: "column",
        gap: "48px",
        maxWidth: "1200px",
        padding: "16px",
        width: "100%",
      }}
    >
      <Stack
        $style={{
          backgroundImage: "url(/images/banner.jpg)",
          backgroundPosition: "center center",
          backgroundSize: "cover",
          borderRadius: "16px",
          height: "336px",
        }}
      />

      <Stack $style={{ flexDirection: "column", gap: "32px" }}>
        <Stack $style={{ flexDirection: "column", gap: "24px" }}>
          <Stack
            as="span"
            $style={{
              fontSize: "40px",
              fontWeight: "500",
              lineHeight: "42px",
              margin: "0",
            }}
          >
            Discover Apps
          </Stack>
          <Stack
            as="span"
            $style={{
              backgroundColor: colors.borderLight.toHex(),
              height: "1px",
            }}
          />
          <Form
            form={form}
            initialValues={filters}
            layout="vertical"
            onValuesChange={handleChange}
          >
            <Row gutter={[16, 16]}>
              <Col xs={24} md={12} xl={8}>
                <Form.Item<PluginFilters> name="term" noStyle>
                  <SearchInput placeholder="Search by" />
                </Form.Item>
              </Col>
              <Col xs={12} md={6} xl={{ span: 4, offset: 8 }}>
                <Form.Item<PluginFilters> name="category" noStyle>
                  <Select
                    options={categoryOptions}
                    placeholder="Category"
                    allowClear
                  />
                </Form.Item>
              </Col>
              <Col xs={12} md={6} xl={4}>
                <Form.Item<PluginFilters> name="sort" noStyle>
                  <Select options={sortOptions} placeholder="Sort" allowClear />
                </Form.Item>
              </Col>
            </Row>
          </Form>
        </Stack>

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
              <Col key={plugin.id} xs={24} md={12} xl={8}>
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
