import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { useParams } from "react-router-dom";

import MarketplaceService, {
  getMarketplaceUrl,
} from "@/modules/marketplace/services/marketplaceService";

import { publish } from "@/utils/eventBus";

import { PluginPolicy, PolicySchema } from "@/modules/plugin/models/policy";
import PolicyService from "../services/policyService";
import { getCurrentVaultId } from "@/storage/currentVaultId";
import { selectToken } from "@/storage/token";

export const POLICY_ITEMS_PER_PAGE = 15;

export interface PolicyContextType {
  // pluginId: string;
  policyMap: Map<string, PluginPolicy>;
  policySchemaMap: Map<string, PolicySchema>;
  policiesTotalCount: number;
  fetchPolicies: () => void;
  removePolicy: (policyId: string) => Promise<void>;
  // addPolicy: (policy: PluginPolicy) => Promise<boolean>;
  // updatePolicy: (policy: PluginPolicy) => Promise<boolean>;
  // removePolicy: (policyId: string) => Promise<void>;
  // getPolicyHistory: (
  //   policyId: string,
  //   skip: number,
  //   take: number
  // ) => Promise<TransactionHistory | null>;
  currentPage: number;
  setCurrentPage: (page: number) => void;
}

export const PolicyContext = createContext<PolicyContextType | undefined>(
  undefined
);
interface InitialState {
  currentPage: number;
  policiesTotalCount: number;
  policySchemaMap: Map<string, PolicySchema>;
  policyMap: Map<string, PluginPolicy>;
}
const initialState: InitialState = {
  currentPage: 0,
  policiesTotalCount: 0,
  policySchemaMap: new Map<string, PolicySchema>(),
  policyMap: new Map<string, PluginPolicy>(),
};

export const PolicyProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const { pluginId } = useParams<{ pluginId: string }>();
  const [state, setState] = useState(initialState);
  const { currentPage, policiesTotalCount, policyMap, policySchemaMap } = state;

  const setCurrentPage = (page: number) => {
    setState((prev) => ({ ...prev, currentPage: page }));
  };

  const removePolicy = async (policyId: string): Promise<void> => {
    let policy = policyMap.get(policyId);

    if (!policy) return;

    try {
      if (policy.signature) {
        await PolicyService.deletePolicy(
          getMarketplaceUrl(),
          policyId,
          policy.signature
        );

        setState((prev) => {
          const updatedPolicyMap = new Map(prev.policyMap);
          updatedPolicyMap.delete(policyId);

          return { ...prev, policyMap: updatedPolicyMap };
        });
        publish("onToast", {
          message: "Policy deleted successfully!",
          type: "success",
        });
      }
    } catch (error) {
      if (error instanceof Error) {
        console.error("Failed to delete policy:", error);
        publish("onToast", {
          message: error.message,
          type: "error",
        });
      }
    }
  };

  const fetchPolicies = useCallback(async (): Promise<void> => {
    const publicKey = getCurrentVaultId();
    const token = publicKey ? selectToken(publicKey) : undefined;
    if (pluginId && token) {
      const fetchedPolicies = await MarketplaceService.getPolicies(
        pluginId,
        currentPage > 1 ? (currentPage - 1) * POLICY_ITEMS_PER_PAGE : 0,
        POLICY_ITEMS_PER_PAGE
      );

      const constructPolicyMap: Map<string, PluginPolicy> = new Map(
        fetchedPolicies?.policies?.map((p: PluginPolicy) => [p.id, p]) // Convert the array into [key, value] pairs
      );

      setState((prev) => ({
        ...prev,
        policiesTotalCount: fetchedPolicies.total_count,
        policyMap: constructPolicyMap,
      }));
    }
  }, [pluginId, currentPage]);

  useEffect(() => {
    fetchPolicies().catch((error: any) => {
      console.error("Failed to get policies:", error.message);
      publish("onToast", {
        message: error.message || "Failed to get policies",
        type: "error",
      });
    });
  }, [fetchPolicies]);

  return (
    <PolicyContext.Provider
      value={{
        policyMap,
        policySchemaMap,
        policiesTotalCount,
        fetchPolicies,
        removePolicy,
        currentPage,
        setCurrentPage,
      }}
    >
      {children}
    </PolicyContext.Provider>
  );
};

export const usePolicies = (): PolicyContextType => {
  const context = useContext(PolicyContext);
  if (!context) {
    throw new Error("usePolicies must be used within a PolicyProvider");
  }
  return context;
};
