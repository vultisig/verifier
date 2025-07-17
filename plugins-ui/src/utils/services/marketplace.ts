import { RecipeSchema } from "proto/recipe_specification_pb";
import { getVaultId } from "storage/vaultId";
import { PAGE_SIZE } from "utils/constants/core";
import { toSnakeCase } from "utils/functions";
import { del, get, post } from "utils/services/api";
import {
  AuthTokenForm,
  Category,
  Plugin,
  PluginFilters,
  PluginPolicy,
  PluginPricing,
  PolicyTransactionHistory,
  ReshareForm,
  Review,
  ReviewForm,
} from "utils/types";

const baseUrl = import.meta.env.VITE_MARKETPLACE_URL;

export const addPluginPolicy = async (data: PluginPolicy) =>
  post<PluginPolicy>(`${baseUrl}/plugin/policy`, data);

export const addPluginReview = async (pluginId: string, review: ReviewForm) =>
  post<Review>(`${baseUrl}/plugins/${pluginId}/reviews`, review);

export const delPluginPolicy = (id: string, signature: string) =>
  del(`${baseUrl}/plugin/policy/${id}`, { data: { signature } });

export const getAuthToken = async (data: AuthTokenForm): Promise<string> =>
  post<{ token: string }>(`${baseUrl}/auth`, toSnakeCase(data)).then(
    ({ token }) => token
  );

export const getPlugin = async (id: string) =>
  get<Plugin>(`${baseUrl}/plugins/${id}`).then((plugin) => ({
    ...plugin,
    pricing: plugin.pricing || [],
  }));

export const getPlugins = (
  skip: number,
  { category, sort, term }: PluginFilters
) =>
  get<{ plugins: Plugin[]; totalCount: number }>(
    `${baseUrl}/plugins?skip=${skip}&take=12${term ? `&term=${term}` : ""}${
      category ? `&category_id=${category}` : ""
    }${sort ? `&sort=${sort}` : ""}`
  ).then(({ plugins, totalCount }) => ({
    plugins:
      plugins.map((plugin) => ({ ...plugin, pricing: plugin.pricing || [] })) ||
      [],
    totalCount,
  }));

export const getPluginCategories = () =>
  get<Category[]>(`${baseUrl}/categories`);

export const getPluginPolicies = async (
  pluginId: string,
  skip = 0,
  take = PAGE_SIZE
) =>
  get<{ policies: PluginPolicy[]; totalCount: number }>(
    `${baseUrl}/plugin/policies/${pluginId}?skip=${skip}&take=${take}`
  ).then(({ policies, totalCount }) => ({
    policies: policies || [],
    totalCount,
  }));

export const getPluginPricing = (id: string) =>
  get<PluginPricing>(`${baseUrl}/pricing/${id}`);

export const getPolicyTransactionHistory = async (
  policyId: string,
  skip: number,
  take: number
) =>
  get<{ history: PolicyTransactionHistory[]; totalCount: number }>(
    `${baseUrl}/plugins/policies/${policyId}/history?skip=${skip}&take=${take}`,
    {
      headers: toSnakeCase({ publicKey: getVaultId() }),
    }
  ).then(({ history, totalCount }) => ({ history: history || [], totalCount }));

export const getPluginReviews = async (
  pluginId: string,
  skip = 0,
  take = PAGE_SIZE
) =>
  get<{ reviews: Review[]; totalCount: number }>(
    `${baseUrl}/plugins/${pluginId}/reviews?skip=${skip}&take=${take}&sort=-created_at`
  ).then(({ reviews, totalCount }) => ({ reviews: reviews || [], totalCount }));

export const getRecipeSpecification = async (pluginId: string) =>
  get<RecipeSchema>(`${baseUrl}/plugins/${pluginId}/recipe-specification`);

export const isPluginInstalled = async (id: string) =>
  get(`${baseUrl}/vault/exist/${id}/${getVaultId()}`)
    .then(() => true)
    .catch(() => false);

export const reshareVault = async (data: ReshareForm) =>
  post(`${baseUrl}/vault/reshare`, toSnakeCase(data));

export const uninstallPlugin = (pluginId: string) =>
  del(`${baseUrl}/plugin/${pluginId}`);
