import { ConstraintType } from "@/gen/constraint_pb";
import { ScheduleFrequency } from "@/gen/scheduling_pb";

export interface SchemaProps {
  pluginId: string;
  pluginName: string;
  pluginVersion: number;
  requirements: {
    minVultisigVersion: number;
    supportedChains: string[];
  };
  scheduleVersion: number;
  scheduling: {
    supportsScheduling: boolean;
    supportedFrequencies: ScheduleFrequency[];
    maxScheduledExecutions: number;
  };
  supportedResources: {
    required: boolean;
    resourcePath: {
      full: string;
      chainId: string;
      functionId: string;
      protocolId: string;
    };
    parameterCapabilities: {
      parameterName: string;
      required: boolean;
      supportedTypes: ConstraintType[];
    }[];
  }[];
  version: number;
}

export interface PolicyFormProps {
  id: string;
  publicKey: string;
  pluginId: string;
  pluginVersion: string;
  policyVersion: number;
  signature: string;
  recipe: string;
  billing: string[];
  active: boolean;
}
