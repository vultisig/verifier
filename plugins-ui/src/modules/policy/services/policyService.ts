import { post, get, put, remove } from "@/modules/core/services/httpService";
import { getMarketplaceUrl } from "@/modules/marketplace/services/marketplaceService";
import { PluginPolicy } from "@/modules/plugin/models/policy";
const baseUrl = getMarketplaceUrl();
const PolicyService = {
  /**
   * Posts a new policy to the API.
   * @param {PluginPolicy} pluginPolicy - The policy to be created.
   * @returns {Promise<Object>} A promise that resolves to the created policy.
   */
  createPolicy: async (pluginPolicy: PluginPolicy): Promise<PluginPolicy> => {
    try {
      const endpoint = `${baseUrl}/plugin/policy`;
      const newPolicy = await post(endpoint, pluginPolicy);
      return newPolicy;
    } catch (error) {
      console.error("Error creating policy:", error);
      throw error;
    }
  },

  /**
   * Updates policy to the API.
   * @param {PluginPolicy} pluginPolicy - The policy to be created.
   * @returns {Promise<Object>} A promise that resolves to the created policy.
   */
  updatePolicy: async (pluginPolicy: PluginPolicy): Promise<PluginPolicy> => {
    try {
      const endpoint = `${baseUrl}/plugin/policy`;
      const newPolicy = await put(endpoint, pluginPolicy);
      return newPolicy;
    } catch (error) {
      console.error("Error updating policy:", error);
      throw error;
    }
  },

  /**
   * Delete policy from the API.
   * @param {id} string - The policy to be deleted.
   */
  deletePolicy: async (id: string, signature: string) => {
    try {
      const endpoint = `${baseUrl}/plugin/policy/${id}`;
      return await remove(endpoint, {
        data: { signature: signature },
      });
    } catch (error) {
      console.error("Error deleting policy:", error);
      throw error;
    }
  },

  /**
   * Get PolicySchema
   * @returns {Promise<Object>} A promise that resolves to the fetched schema.
   */
  getPolicySchema: (pluginId: string): Promise<any> => {
    return get(`${baseUrl}/plugin/policy/schema`, {
      headers: { plugin_id: pluginId },
    });
  },
};

export default PolicyService;
