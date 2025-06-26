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
    supportedFrequencies: number[];
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
      supportedTypes: number[];
    }[];
  }[];
  version: number;
}
