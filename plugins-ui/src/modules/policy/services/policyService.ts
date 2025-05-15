import { post, get, put, remove } from "@/modules/core/services/httpService";
import { PluginPolicy, PolicyTransactionHistory } from "../models/policy";

const getPublicKey = () => localStorage.getItem("publicKey");
const getPluginUrl = () => import.meta.env.VITE_PLUGIN_URL;
 
/**
 * Service for managing plugin policies and plugins.
 * Provides methods for CRUD operations on policies and plugins, including authentication.
 */
const PolicyService = {
  /**
   * Creates a new plugin policy.
   * Maps to POST /plugin/policy endpoint.
   * @param {PluginPolicy} pluginPolicy - The policy data to create.
   * @returns {Promise<PluginPolicy>} A promise that resolves to the created policy.
   * @throws {Error} If there's an error creating the policy.
   */
  createPolicy: async (pluginPolicy: PluginPolicy): Promise<PluginPolicy> => {
    try {
      const endpoint = `${getPluginUrl()}/plugin/policy`;
      const newPolicy = await post(endpoint, pluginPolicy, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("authToken")}`,
        },
      });
      return newPolicy;
    } catch (error) {
      console.error("Error creating policy:", error);
      throw error;
    }
  },

  /**
   * Updates an existing plugin policy.
   * Maps to PUT /plugin/policy endpoint.
   * @param {PluginPolicy} pluginPolicy - The updated policy data.
   * @returns {Promise<PluginPolicy>} A promise that resolves to the updated policy.
   * @throws {Error} If there's an error updating the policy.
   */
  updatePolicy: async (pluginPolicy: PluginPolicy): Promise<PluginPolicy> => {
    try {
      const endpoint = `${getPluginUrl()}/plugin/policy`;
      const updatedPolicy = await put(endpoint, pluginPolicy, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("authToken")}`,
        },
      });
      return updatedPolicy;
    } catch (error) {
      console.error("Error updating policy:", error);
      throw error;
    }
  },

  /**
   * Retrieves all plugin policies for a specific plugin type.
   * Maps to GET /plugin/policy endpoint.
   * @param {string} pluginType - The type of plugin to get policies for.
   * @returns {Promise<PluginPolicy[]>} A promise that resolves to an array of policies.
   * @throws {Error} If there's an error fetching the policies.
   */
  getPolicies: async (pluginType: string): Promise<PluginPolicy[]> => {
    try {
      const endpoint = `${getPluginUrl()}/plugin/policy`;
      const policies = await get(endpoint, {
        headers: {
          plugin_type: pluginType,
          public_key: getPublicKey(),
          Authorization: `Bearer ${localStorage.getItem("authToken")}`,
        },
      });
      return policies;
    } catch (error) {
      console.error("Error getting policies:", error);
      throw error;
    }
  },

  /**
   * Retrieves a specific policy by its ID.
   * Maps to GET /plugin/policy/:policyId endpoint.
   * @param {string} policyId - The unique identifier of the policy.
   * @returns {Promise<PluginPolicy>} A promise that resolves to the policy data.
   * @throws {Error} If there's an error fetching the policy.
   */
  getPolicyById: async (policyId: string): Promise<PluginPolicy> => {
    try {
      const endpoint = `${getPluginUrl()}/plugin/policy/${policyId}`;
      const policy = await get(endpoint, {
        headers: {
          public_key: getPublicKey(),
          Authorization: `Bearer ${localStorage.getItem("authToken")}`,
        },
      });
      return policy;
    } catch (error) {
      console.error("Error getting policy by ID:", error);
      throw error;
    }
  },

  /**
   * Deletes a specific policy.
   * Maps to DELETE /plugin/policy/:policyId endpoint.
   * @param {string} id - The unique identifier of the policy to delete.
   * @param {string} signature - The signature for authorization.
   * @returns {Promise<void>} A promise that resolves when the policy is deleted.
   * @throws {Error} If there's an error deleting the policy.
   */
  deletePolicy: async (id: string, signature: string): Promise<void> => {
    try {
      const endpoint = `${getPluginUrl()}/plugin/policy/${id}`;
      await remove(endpoint, { signature }, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("authToken")}`,
        },
      });
    } catch (error) {
      console.error("Error deleting policy:", error);
      throw error;
    }
  },

  /**
   * Retrieves all available plugins.
   * Maps to GET /plugins endpoint.
   * @returns {Promise<any[]>} A promise that resolves to an array of plugins.
   * @throws {Error} If there's an error fetching the plugins.
   */
  getPlugins: async (): Promise<any[]> => {
    try {
      const endpoint = `${getPluginUrl()}/plugins`;
      const plugins = await get(endpoint);
      return plugins;
    } catch (error) {
      console.error("Error getting plugins:", error);
      throw error;
    }
  },

  /**
   * Retrieves a specific plugin by its ID.
   * Maps to GET /plugins/:pluginId endpoint.
   * @param {string} pluginId - The unique identifier of the plugin.
   * @returns {Promise<any>} A promise that resolves to the plugin data.
   * @throws {Error} If there's an error fetching the plugin.
   */
  getPluginById: async (pluginId: string): Promise<any> => {
    try {
      const endpoint = `${getPluginUrl()}/plugins/${pluginId}`;
      const plugin = await get(endpoint);
      return plugin;
    } catch (error) {
      console.error("Error getting plugin by ID:", error);
      throw error;
    }
  },

  /**
   * Creates a new plugin.
   * Maps to POST /plugins endpoint.
   * @param {any} pluginData - The data for creating the new plugin.
   * @returns {Promise<any>} A promise that resolves to the created plugin.
   * @throws {Error} If there's an error creating the plugin.
   */
  createPlugin: async (pluginData: any): Promise<any> => {
    try {
      const endpoint = `${getPluginUrl()}/plugins`;
      const plugin = await post(endpoint, pluginData, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("authToken")}`,
        },
      });
      return plugin;
    } catch (error) {
      console.error("Error creating plugin:", error);
      throw error;
    }
  },

  /**
   * Updates an existing plugin.
   * Maps to POST /plugins/:pluginId endpoint.
   * @param {string} pluginId - The unique identifier of the plugin to update.
   * @param {any} pluginData - The new data to update the plugin with.
   * @returns {Promise<any>} A promise that resolves to the updated plugin.
   * @throws {Error} If there's an error updating the plugin.
   */
  updatePlugin: async (pluginId: string, pluginData: any): Promise<any> => {
    try {
      const endpoint = `${getPluginUrl()}/plugins/${pluginId}`;
      const plugin = await post(endpoint, pluginData, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("authToken")}`,
        },
      });
      return plugin;
    } catch (error) {
      console.error("Error updating plugin:", error);
      throw error;
    }
  },

  /**
   * Deletes a plugin from the system.
   * Maps to DELETE /plugins/:pluginId endpoint.
   * @param {string} pluginId - The unique identifier of the plugin to delete.
   * @returns {Promise<void>} A promise that resolves when the plugin is deleted.
   * @throws {Error} If there's an error deleting the plugin.
   */
  deletePlugin: async (pluginId: string): Promise<void> => {
    try {
      const endpoint = `${getPluginUrl()}/plugins/${pluginId}`;
      await remove(endpoint, {}, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem("authToken")}`,
        },
      });
    } catch (error) {
      console.error("Error deleting plugin:", error);
      throw error;
    }
  },

  /**
   * Verifies wallet ownership and authenticates the user.
   * Maps to POST /auth endpoint.
   * The function handles:
   * 1. Generating the verification message
   * 2. Getting the message signed by the wallet
   * 3. Sending the signed message for verification
   * 4. Receiving and storing the auth token
   * 
   * @param {Object} params - The authentication parameters
   * @param {string} params.publicKey - The user's public key
   * @param {string} params.chainCodeHex - The chain code in hex format
   * @param {Function} params.signMessage - Function to sign message with wallet (should return hex string with 0x prefix)
   * @returns {Promise<string>} A promise that resolves to the auth token
   * @throws {Error} If authentication fails
   */
  verifyWalletAndAuth: async ({
    publicKey,
    chainCodeHex,
    message,
    signature
  }: {
    publicKey: string;
    chainCodeHex: string;
    message: string;
    signature: string;
  }): Promise<string> => {
    try {
 
      // Send auth request to server
      const endpoint = `${getPluginUrl()}/auth`;
      const response = await post(endpoint, {
        message,
        signature,
        public_key: publicKey,
        chain_code_hex: chainCodeHex,
      });

      // Store the token
      if (response.token) {
        localStorage.setItem("authToken", response.token);
      }

      return response.token;
    } catch (error) {
      console.error("Wallet verification failed:", error);
      throw error;
    }
  },

  /**
   * Get policy transaction history from the API.
   * @returns {Promise<Object>} A promise that resolves to the fetched policies.
   */
  getPolicyTransactionHistory: async (
    policyId: string
  ): Promise<PolicyTransactionHistory[]> => {
    try {
      const endpoint = `${getPluginUrl()}/plugin/policy/history/${policyId}`;
      const history = await get(endpoint, {
        headers: {
          public_key: getPublicKey(),
          Authorization: `Bearer ${localStorage.getItem("authToken")}`,
        },
      });
      return history;
    } catch (error) {
      console.error("Error getting policy history:", error);

      throw error;
    }
  },

  /**
   * Get PolicySchema
   * @returns {Promise<Object>} A promise that resolves to the fetched schema.
   */
  getPolicySchema: async (pluginType: string): Promise<any> => {
    try {
      const endpoint = `${getPluginUrl()}/plugin/policy/schema`;
      const newPolicy = await get(endpoint, {
        headers: {
          plugin_type: pluginType,
        },
      });
      return newPolicy;
    } catch (error) {
      console.error("Error getting policy schema:", error);
      throw error;
    }
  },
};

export default PolicyService;
