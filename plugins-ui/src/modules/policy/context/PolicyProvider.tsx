import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { useParams } from "react-router-dom";

import MarketplaceService from "@/modules/marketplace/services/marketplaceService";

import { publish } from "@/utils/eventBus";

import { PluginPolicy, PolicySchema } from "@/modules/plugin/models/policy";

export const POLICY_ITEMS_PER_PAGE = 15;

export interface PolicyContextType {
  // pluginId: string;
  policyMap: Map<string, PluginPolicy>;
  policySchemaMap: Map<string, PolicySchema>;
  policiesTotalCount: number;
  fetchPolicies: () => void;
  // addPolicy: (policy: PluginPolicy) => Promise<boolean>;
  // updatePolicy: (policy: PluginPolicy) => Promise<boolean>;
  // removePolicy: (policyId: string) => Promise<void>;
  // getPolicyHistory: (
  //   policyId: string,
  //   skip: number,
  //   take: number
  // ) => Promise<TransactionHistory | null>;
  // currentPage: number;
  // setCurrentPage: (page: number) => void;
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

  const fetchPolicies = useCallback(async (): Promise<void> => {
    console.log("use policies pluginID:", pluginId);

    if (pluginId) {
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
