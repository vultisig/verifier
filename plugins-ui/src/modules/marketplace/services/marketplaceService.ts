import { get, post } from "@/modules/core/services/httpService";
import { Category } from "../models/category";
import {
  CreateReview,
  PluginMap,
  Review,
  ReviewMap,
} from "../models/marketplace";
import { Plugin } from "@/modules/plugin/models/plugin";
import {
  PluginPoliciesMap,
  TransactionHistory,
} from "@/modules/policy/models/policy";
import { getCurrentVaultId } from "@/storage/currentVaultId";

const getMarketplaceUrl = () => import.meta.env.VITE_MARKETPLACE_URL;

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
    try {
      await get(`${getMarketplaceUrl()}/vault/exist/${id}/${key}`);
      return true;
    } catch {
      return false;
    }
  },

  /**
   * Get plugins from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched plugins.
   */
  getPlugins: async (): Promise<PluginMap> => {
    try {
      return await get(`${getMarketplaceUrl()}/plugins`);
    } catch (error) {
      console.error("Error getting plugins:", error);
      throw error;
    }
  },

  /**
   * Get all plugin categories
   * @returns {Promise<Object>} A promise that resolves to the fetched categories.
   */
  getCategories: async (): Promise<Category[]> => {
    try {
      return await get(`${getMarketplaceUrl()}/categories`);
    } catch (error) {
      console.error("Error getting categories:", error);
      throw error;
    }
  },

  /**
   * Get plugin by id from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched plugin.
   */
  getPlugin: async (id: string): Promise<Plugin> => {
    try {
      return await get(`${getMarketplaceUrl()}/plugins/${id}`);
    } catch (error) {
      console.error("Error getting plugin:", error);
      throw error;
    }
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
    try {
      const response = await post(`${getMarketplaceUrl()}/auth`, {
        message: message,
        signature: signature,
        public_key: publicKey,
        chain_code_hex: chainCodeHex,
      });

      return response?.token;
    } catch (error) {
      console.error("Failed to get auth token", error);
      throw error;
    }
  },

  /**
   * Get policies from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched policies.
   */
  getPolicies: async (
    pluginType: string,
    skip: number,
    take: number
  ): Promise<PluginPoliciesMap> => {
    try {
      return await get(
        `${getMarketplaceUrl()}/plugins/policies?skip=${skip}&take=${take}`,
        {
          headers: {
            plugin_type: pluginType,
            public_key: getCurrentVaultId(),
          },
        }
      );
    } catch (error: any) {
      if (error.message === "Unauthorized") {
        localStorage.removeItem("authToken");
        // Dispatch custom event to notify other components
        window.dispatchEvent(new Event("storage"));
      }
      console.error("Error getting policies:", error);
      throw error;
    }
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
    try {
      return await get(
        `${getMarketplaceUrl()}/plugins/policies/${policyId}/history?skip=${skip}&take=${take}`,
        {
          headers: {
            public_key: getCurrentVaultId(),
          },
        }
      );
    } catch (error) {
      console.error("Error getting policy history:", error);

      throw error;
    }
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
    try {
      return await get(
        `${getMarketplaceUrl()}/plugins/${pluginId}/reviews?skip=${skip}&take=${take}&sort=${sort}`
      );
    } catch (error: any) {
      console.error("Error getting reviews:", error);
      throw error;
    }
  },

  /**
   * Post review for specific pluginId from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched review for specific plugin.
   */
  createReview: async (
    pluginId: string,
    review: CreateReview
  ): Promise<Review> => {
    try {
      return await post(
        `${getMarketplaceUrl()}/plugins/${pluginId}/reviews`,
        review
      );
    } catch (error: any) {
      console.error("Error create review:", error);
      throw error;
    }
  },

  /**
   * Get recipe specification for a plugin.
   */
  getRecipeSpecification: async (pluginId: string): Promise<any> => {
    try {
      return await get(
        `${getMarketplaceUrl()}/plugins/${pluginId}/recipe-specification`
      );
    } catch (error) {
      console.error("Error getting recipe specification:", error);
      throw error;
    }
  },

  /**
   * Send reshare request payload to the verifier backend.
   * @param payload Decoded ReshareMessage object from VultiConnect extension
   */
  reshareVault: async (payload: ReshareRequest): Promise<void> => {
    try {
      await post(`${getMarketplaceUrl()}/vault/reshare`, payload);
    } catch (error) {
      console.error("Error initiating vault reshare:", error);
      throw error;
    }
  },
};

export default MarketplaceService;
