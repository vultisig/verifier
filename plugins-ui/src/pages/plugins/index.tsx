import { Empty, SelectProps } from "antd";
import { Divider } from "components/Divider";
import { PluginItem } from "components/PluginItem";
import { Select } from "components/Select";
import { Spin } from "components/Spin";
import { Stack } from "components/Stack";
import { useFilterParams } from "hooks/useFilterParams";
import { debounce } from "lodash-es";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTheme } from "styled-components";
import { getPluginCategories, getPlugins } from "utils/services/marketplace";
import { Category, Plugin, PluginFilters } from "utils/types";

type InitialState = {
  categories: Category[];
  loading: boolean;
  plugins: Plugin[];
  sortOptions: NonNullable<SelectProps["options"]>;
};

export const PluginsPage = () => {
  const initialState: InitialState = {
    categories: [],
    loading: true,
    plugins: [],
    sortOptions: [
      { value: "-created_at", label: "Newest" },
      { value: "created_at", label: "Oldest" },
    ],
  };
  const [state, setState] = useState(initialState);
  const { categories, loading, plugins, sortOptions } = state;
  const { filters, setFilters } = useFilterParams<PluginFilters>();
  const colors = useTheme();
  const [newPlugin] = plugins;

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
        setState((prevState) => ({ ...prevState, categories }));
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

      <Stack $style={{ flexDirection: "column", flexGrow: "1", gap: "32px" }}>
        <Stack $style={{ flexDirection: "column", gap: "24px" }}>
          <Stack
            as="span"
            $style={{
              fontSize: "40px",
              fontWeight: "500",
              lineHeight: "42px",
            }}
          >
            Discover Apps
          </Stack>
          <Divider />
          <Stack $style={{ flexDirection: "row", gap: "12px" }}>
            <Stack
              $style={{ flexDirection: "row", flexGrow: "1", gap: "12px" }}
            >
              {categories.map(({ id, name }) => (
                <Stack
                  as="span"
                  key={id}
                  onClick={() => setFilters({ ...filters, category: id })}
                  $hover={{
                    backgroundColor: colors.textSecondary.toHex(),
                    color: colors.buttonText.toHex(),
                  }}
                  $style={{
                    alignItems: "center",
                    backgroundColor:
                      filters.category === id
                        ? colors.textSecondary.toHex()
                        : colors.bgSecondary.toHex(),
                    border: `solid 1px ${colors.borderNormal.toHex()}`,
                    borderRadius: "8px",
                    color:
                      filters.category === id
                        ? colors.buttonText.toHex()
                        : colors.textPrimary.toHex(),
                    cursor: "pointer",
                    flexDirection: "column",
                    fontSize: "12px",
                    fontWeight: "500",
                    gap: "8px",
                    justifyContent: "center",
                    height: "40px",
                    padding: "0 24px",
                    whiteSpace: "nowrap",
                  }}
                >
                  {name}
                </Stack>
              ))}
            </Stack>
            <Stack
              $style={{
                alignItems: "center",
                flexDirection: "row",
                gap: "12px",
                width: "200px",
              }}
            >
              <Stack as="span" $style={{ whiteSpace: "nowrap" }}>
                Sort By
              </Stack>
              <Select
                options={sortOptions}
                value={filters.sort}
                onChange={(sort) => setFilters({ ...filters, sort })}
                allowClear
              />
            </Stack>
          </Stack>
        </Stack>

        {loading ? (
          <Spin />
        ) : plugins.length ? (
          <>
            {newPlugin ? (
              <>
                <Stack $style={{ flexDirection: "column", gap: "16px" }}>
                  <Stack
                    as="span"
                    $style={{
                      fontSize: "17px",
                      fontWeight: "500",
                      lineHeight: "20px",
                    }}
                  >
                    New
                  </Stack>
                  <PluginItem plugin={newPlugin} horizontal />
                </Stack>
                <Divider />
              </>
            ) : (
              <></>
            )}
            <Stack $style={{ flexDirection: "column", gap: "16px" }}>
              <Stack
                as="span"
                $style={{
                  fontSize: "17px",
                  fontWeight: "500",
                  lineHeight: "20px",
                }}
              >
                All Apps
              </Stack>
              <Stack
                $style={{
                  display: "grid",
                  gap: "32px",
                  gridTemplateColumns: "repeat(2, 1fr)",
                }}
                $media={{
                  xl: { $style: { gridTemplateColumns: "repeat(3, 1fr)" } },
                }}
              >
                {plugins.map((plugin) => (
                  <PluginItem key={plugin.id} plugin={plugin} />
                ))}
              </Stack>
            </Stack>
          </>
        ) : (
          <Empty />
        )}
      </Stack>
    </Stack>
  );
};
