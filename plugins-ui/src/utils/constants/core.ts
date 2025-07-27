import { ConstraintType } from "proto/constraint_pb";
import { ScheduleFrequency } from "proto/scheduling_pb";

const SECONDS_IN_HOUR = 3600;
const SECONDS_IN_DAY = SECONDS_IN_HOUR * 24;
const SECONDS_IN_WEEK = SECONDS_IN_DAY * 7;
const SECONDS_IN_BIWEEK = SECONDS_IN_WEEK * 2;
const SECONDS_IN_MONTH = SECONDS_IN_DAY * 30;

export const constraintTypeLabels: Record<ConstraintType, string> = {
  [ConstraintType.FIXED]: "Fixed",
  [ConstraintType.MAGIC_CONSTANT]: "Magic Constant",
  [ConstraintType.MAX]: "Max",
  [ConstraintType.MAX_PER_PERIOD]: "Max Per Period",
  [ConstraintType.MIN]: "Min",
  [ConstraintType.RANGE]: "Range",
  [ConstraintType.UNSPECIFIED]: "Unspecified",
  [ConstraintType.WHITELIST]: "Whitelist",
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

export const scheduleFrequencyToSeconds: Record<ScheduleFrequency, number> = {
  [ScheduleFrequency.UNSPECIFIED]: 0,
  [ScheduleFrequency.HOURLY]: SECONDS_IN_HOUR,
  [ScheduleFrequency.DAILY]: SECONDS_IN_DAY,
  [ScheduleFrequency.WEEKLY]: SECONDS_IN_WEEK,
  [ScheduleFrequency.BIWEEKLY]: SECONDS_IN_BIWEEK,
  [ScheduleFrequency.MONTHLY]: SECONDS_IN_MONTH,
};
