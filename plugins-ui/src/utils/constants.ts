import { ConstraintType } from "@/gen/constraint_pb";
import { ScheduleFrequency } from "@/gen/scheduling_pb";

export const frequencyName: Record<ScheduleFrequency, string> = {
  [ScheduleFrequency.UNSPECIFIED]: "Unspecified",
  [ScheduleFrequency.HOURLY]: "Hourly",
  [ScheduleFrequency.DAILY]: "Daily",
  [ScheduleFrequency.WEEKLY]: "Weekly",
  [ScheduleFrequency.BIWEEKLY]: "Biweekly",
  [ScheduleFrequency.MONTHLY]: "Monthly",
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

export const PluginPricingType: Record<string, string> = {
  "per-tx": "per trade",
  once: "once",
};
