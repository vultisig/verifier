import { ConstraintType } from "proto/constraint_pb";

export const constraintTypeLabels: Record<ConstraintType, string> = {
  [ConstraintType.FIXED]: "Fixed",
  [ConstraintType.MAGIC_CONSTANT]: "Magic Constant",
  [ConstraintType.MAX]: "Max",
  [ConstraintType.MIN]: "Min",
  [ConstraintType.UNSPECIFIED]: "Unspecified",
};

export const modalHash = {
  currency: "#currency",
  language: "#language",
  policy: "#policy",
} as const;

export const PAGE_SIZE = 12;
