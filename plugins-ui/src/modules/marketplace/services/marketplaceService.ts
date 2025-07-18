import { get, post, remove } from "@/modules/core/services/httpService";
import { Category } from "../models/category";
import {
  CreateReview,
  PluginMap,
  Review,
  ReviewMap,
} from "../models/marketplace";
import { Plugin } from "@/modules/plugin/models/plugin";
import { getCurrentVaultId } from "@/storage/currentVaultId";
import { toCamelCase } from "@/utils/functions";
import {
  PluginPolicy,
  TransactionHistory,
} from "@/modules/plugin/models/policy";
import { RecipeSchema } from "@/gen/recipe_specification_pb";
import { PluginFilters } from "@/modules/marketplace/components/marketplace-filters/MarketplaceFilters";

export const getMarketplaceUrl = () => import.meta.env.VITE_MARKETPLACE_URL;

interface ReshareRequest {
  name: string;
  public_key: string;
  session_id: string;
  hex_encryption_key: string;
  hex_chain_code: string;
  local_party_id: string;
  old_parties: string[];
  email: string;
  plugin_id: string;
}

const MarketplaceService = {
  /**
   * Get plugin status by id from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched plugin.
   */
  isPluginInstalled: async (id: string, key: string): Promise<boolean> => {
    return get(`${getMarketplaceUrl()}/vault/exist/${id}/${key}`)
      .then(() => true)
      .catch(() => false);
  },

  /**
   * Get plugins from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched plugins.
   */
  getPlugins: async (
    skip = 0,
    take = 12,
    term?: string,
    category?: string,
    sort?: string
  ): Promise<PluginMap> => {
    return get(
      `${getMarketplaceUrl()}/plugins?skip=${skip}&take=${take}${term ? `&term=${term}` : ""}${category ? `&category_id=${category}` : ""}&sort=${sort}`
    );
  },

  /**
   * Get all plugin categories
   * @returns {Promise<Object>} A promise that resolves to the fetched categories.
   */
  getCategories: async (): Promise<Category[]> => {
    return get(`${getMarketplaceUrl()}/categories`);
  },

  /**
   * Get plugin by id from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched plugin.
   */
  getPlugin: async (id: string): Promise<Plugin> => {
    return get(`${getMarketplaceUrl()}/plugins/${id}`);
  },

  /**
   * Post signature, publicKey, chainCodeHex, derivePath to the APi
   * @returns {Promise<Object>} A promise that resolves with auth token.
   */
  getAuthToken: async (
    message: string,
    signature: string,
    publicKey: string,
    chainCodeHex: string
  ): Promise<string> => {
    return post(`${getMarketplaceUrl()}/auth`, {
      message: message,
      signature: signature,
      public_key: publicKey,
      chain_code_hex: chainCodeHex,
    }).then(({ token }) => token);
  },

  /**
   * Get policies from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched policies.
   */
  getPolicies: async (
    pluginId: string,
    skip: number,
    take: number
  ): Promise<{ policies: PluginPolicy[]; total_count: number }> => {
    return get(
      `${getMarketplaceUrl()}/plugin/policies/${pluginId}?skip=${skip}&take=${take}`
    );
  },

  /**
   * Get policy transaction history from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched policies.
   */
  getPolicyTransactionHistory: async (
    policyId: string,
    skip: number,
    take: number
  ): Promise<TransactionHistory> => {
    return get(
      `${getMarketplaceUrl()}/plugins/policies/${policyId}/history?skip=${skip}&take=${take}`,
      {
        headers: { public_key: getCurrentVaultId() },
      }
    );
  },

  /**
   * Get review for specific pluginId from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched review for specific plugin.
   */
  getReviews: async (
    pluginId: string,
    skip: number,
    take: number,
    sort = "-created_at"
  ): Promise<ReviewMap> => {
    return get(
      `${getMarketplaceUrl()}/plugins/${pluginId}/reviews?skip=${skip}&take=${take}&sort=${sort}`
    );
  },

  /**
   * Post review for specific pluginId from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched review for specific plugin.
   */
  createReview: async (
    pluginId: string,
    review: CreateReview
  ): Promise<Review> => {
    return post(`${getMarketplaceUrl()}/plugins/${pluginId}/reviews`, review);
  },

  /**
   * Get recipe specification for a plugin.
   */
  getRecipeSpecification: async (pluginId: string): Promise<RecipeSchema> => {
    return get(
      `${getMarketplaceUrl()}/plugins/${pluginId}/recipe-specification`
    ).then((schema) => toCamelCase(schema));
  },

  /**
   * Send reshare request payload to the verifier backend.
   * @param payload Decoded ReshareMessage object from VultiConnect extension
   */
  reshareVault: async (payload: ReshareRequest): Promise<void> => {
    return post(`${getMarketplaceUrl()}/vault/reshare`, payload);
  },

  uninstallPlugin: (pluginId: string) =>
    remove(`${getMarketplaceUrl()}/plugin/${pluginId}`),
};

export default MarketplaceService;
