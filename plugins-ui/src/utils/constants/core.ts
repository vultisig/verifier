import { ConstraintType } from "proto/constraint_pb";
import { ScheduleFrequency } from "proto/scheduling_pb";

export const constraintTypeLabels: Record<ConstraintType, string> = {
  [ConstraintType.FIXED]: "fixed",
  [ConstraintType.MAX]: "max",
  [ConstraintType.MAX_PER_PERIOD]: "max_per_period",
  [ConstraintType.MIN]: "min",
  [ConstraintType.RANGE]: "range",
  [ConstraintType.UNSPECIFIED]: "unspecified",
  [ConstraintType.WHITELIST]: "whitelist",
};

export const modalHash = {
  currency: "#currency",
  language: "#language",
  policy: "#policy",
} as const;

export const PAGE_SIZE = 12;

export const scheduleFrequencyLabels: Record<ScheduleFrequency, string> = {
  [ScheduleFrequency.UNSPECIFIED]: "Unspecified",
  [ScheduleFrequency.HOURLY]: "Hourly",
  [ScheduleFrequency.DAILY]: "Daily",
  [ScheduleFrequency.WEEKLY]: "Weekly",
  [ScheduleFrequency.BIWEEKLY]: "Biweekly",
  [ScheduleFrequency.MONTHLY]: "Monthly",
};
