import { ConstraintType } from "@/gen/constraint_pb";
import { BillingFrequency, FeeType } from "@/gen/policy_pb";
import { ScheduleFrequency } from "@/gen/scheduling_pb";

export const frequencyName: Record<ScheduleFrequency, string> = {
  [ScheduleFrequency.UNSPECIFIED]: "Unspecified",
  [ScheduleFrequency.HOURLY]: "Hourly",
  [ScheduleFrequency.DAILY]: "Daily",
  [ScheduleFrequency.WEEKLY]: "Weekly",
  [ScheduleFrequency.BIWEEKLY]: "Biweekly",
  [ScheduleFrequency.MONTHLY]: "Monthly",
};

export const aliasToBillingFrequency: Record<string, BillingFrequency> = {
  daily: BillingFrequency.DAILY,
  weekly: BillingFrequency.WEEKLY,
  biweekly: BillingFrequency.BIWEEKLY,
  monthly: BillingFrequency.MONTHLY,
  unspecified: BillingFrequency.BILLING_FREQUENCY_UNSPECIFIED,
};

export const constraintTypeName: Record<ConstraintType, string> = {
  [ConstraintType.FIXED]: "fixed",
  [ConstraintType.MAX]: "max",
  [ConstraintType.MAX_PER_PERIOD]: "max_per_period",
  [ConstraintType.MIN]: "min",
  [ConstraintType.RANGE]: "range",
  [ConstraintType.UNSPECIFIED]: "unspecified",
  [ConstraintType.WHITELIST]: "whitelist",
};

export const PluginPricingType: Record<FeeType, string> = {
  [FeeType.FEE_TYPE_UNSPECIFIED]: "unspecified",
  [FeeType.RECURRING]: "recurring",
  [FeeType.ONCE]: "once",
  [FeeType.TRANSACTION]: "per trade",
};

export const AliasToFeeType: Record<string, FeeType> = {
  "per-tx": FeeType.TRANSACTION,
  once: FeeType.ONCE,
  recurring: FeeType.RECURRING,
};
