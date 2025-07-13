import { RJSFSchema, UiSchema } from "@rjsf/utils";
import { ColumnDef } from "@tanstack/react-table";

export enum PluginProgress {
  InProgress = "IN PROGRESS",
  Done = "DONE",
}

export type Policy<
  T = string | number | boolean | string[] | null | undefined,
> = {
  [key: string]: T | Policy<T>;
};
export type BillingPolicy = {
  id: string;
  type: string;
  frequency?: string;
  start_date?: number;
  amount: number;
};

export type FeePolicies = {
  type: string;
  start_date: string;
  frequency: string;
  amount: number;
};
export type PluginPolicy = {
  id: string;
  public_key: string;
  plugin_version: string;
  plugin_id: string;
  policy_version: number;
  active: boolean;
  signature?: string;
  recipe: string;
};

export type PluginPoliciesMap = {
  policies: PluginPolicy[];
  total_count: number;
};

export type TransactionHistory = {
  history: PolicyTransactionHistory[];
  total_count: number;
};

export type PolicyTableColumn = ColumnDef<unknown> & {
  accessorKey: string;
  header: string;
  cellComponent?: string;
  expandable?: boolean;
};

export type PolicyTransactionHistory = {
  id: string;
  updated_at: string;
  status: string;
};

export type PolicySchema = {
  form: {
    schema: RJSFSchema;
    uiSchema: UiSchema;
    plugin_version: string;
    policy_version: string;
  };
  table: {
    columns: PolicyTableColumn[];
    mapping: Record<string, string | string[]>;
  };
};
