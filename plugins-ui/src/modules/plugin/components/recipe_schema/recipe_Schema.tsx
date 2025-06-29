import React, { useEffect, useState } from "react";
import MarketplaceService, {
  getMarketplaceUrl,
} from "@/modules/marketplace/services/marketplaceService";
import Button from "@/modules/core/components/ui/button/Button";
import "./recipe_Schema.styles.css";
import { RecipeSchema } from "@/utils/interfaces";
import { PolicySchema, ScheduleSchema } from "@/gen/policy_pb";
import { ScheduleFrequency } from "@/gen/scheduling_pb";
import { ConstraintSchema, ConstraintType } from "@/gen/constraint_pb";
import { Effect, RuleSchema } from "@/gen/rule_pb";
import { create, toBinary } from "@bufbuild/protobuf";
import { constraintTypeName, frequencyName } from "@/utils/constants";
import { ParameterConstraintSchema } from "@/gen/parameter_constraint_pb";
import VulticonnectWalletService from "@/modules/shared/wallet/vulticonnectWalletService";
import { getCurrentVaultId } from "@/storage/currentVaultId";

import { v4 as uuidv4 } from "uuid";
import { Plugin } from "../../models/plugin";
import { publish } from "@/utils/eventBus";
import { PluginPolicy } from "../../models/policy";
import PolicyService from "@/modules/policy/services/policyService";
import { usePolicies } from "@/modules/policy/context/PolicyProvider";
interface InitialState {
  error?: string;
  frequency?: ScheduleFrequency;
  formData: Record<string, string>;
  loading: boolean;
  selectedResource: number;
  schedulingEnabled: boolean;
  schema?: RecipeSchema;
  submitting?: boolean;
  validationErrors: Record<string, string>;
}

interface RecipeSchemaProps {
  onClose: () => void;
  plugin: Plugin;
}

const RecipeSchemaForm: React.FC<RecipeSchemaProps> = ({ plugin, onClose }) => {
  const initialState: InitialState = {
    formData: {},
    loading: true,
    selectedResource: 0,
    schedulingEnabled: false,
    validationErrors: {},
  };
  const [state, setState] = useState(initialState);
  const { fetchPolicies } = usePolicies();
  const {
    error,
    frequency,
    formData,
    loading,
    selectedResource,
    schedulingEnabled,
    schema,
    validationErrors,
  } = state;
  const currentResource = schema?.supportedResources[selectedResource];

  const fetchSchema = () => {
    setState((prevState) => ({
      ...prevState,
      error: undefined,
      loading: true,
    }));

    MarketplaceService.getRecipeSpecification(plugin.id)
      .then((schema) => {
        if (schema.supportedResources) {
          const [{ parameterCapabilities }] = schema.supportedResources;
          const formData: InitialState["formData"] = {};

          parameterCapabilities.forEach((param) => {
            formData[param.parameterName] = "";
          });

          setState((prevState) => ({
            ...prevState,
            formData,
            loading: false,
            schema,
          }));
        } else {
          setState((prevState) => ({ ...prevState, loading: false, schema }));
        }
      })
      .catch(() => {
        setState((prevState) => ({
          ...prevState,
          error: "Failed to load policy schema",
          loading: false,
        }));
      });
  };

  const formValidation = () => {
    const validationErrors: InitialState["validationErrors"] = {};
    const currentResource = schema?.supportedResources[selectedResource];

    currentResource?.parameterCapabilities.forEach((param) => {
      if (param.required && !formData[param.parameterName]?.trim()) {
        validationErrors[param.parameterName] =
          `${param.parameterName} is required`;
      }
    });

    if (schedulingEnabled && frequency === undefined) {
      validationErrors.frequency =
        "Please select a frequency for scheduled execution";
    }

    setState((prevState) => ({ ...prevState, validationErrors }));

    return Object.keys(validationErrors).length === 0;
  };

  const handleInputChange = (paramName: string, value: string) => {
    if (validationErrors[paramName]) {
      setState((prevState) => ({
        ...prevState,
        formData: { ...prevState.formData, [paramName]: value },
        validationErrors: { ...prevState.validationErrors, [paramName]: "" },
      }));
    } else {
      setState((prevState) => ({
        ...prevState,
        formData: { ...prevState.formData, [paramName]: value },
      }));
    }
  };

  const handleSubmit = async () => {
    if (currentResource && schema && formValidation()) {
      const parameterConstraints = currentResource.parameterCapabilities.map(
        ({ parameterName, required }) => {
          const constraint = create(ConstraintSchema, {
            denominatedIn: "",
            period: "",
            required,
            type: ConstraintType.FIXED,
            value: { case: "fixedValue", value: formData[parameterName] },
          });

          const parameterConstraint = create(ParameterConstraintSchema, {
            constraint: constraint,
            parameterName,
          });

          return parameterConstraint;
        }
      );

      const rule = create(RuleSchema, {
        constraints: {},
        description: "",
        effect: Effect.ALLOW,
        id: "",
        parameterConstraints,
        resource: currentResource.resourcePath.full,
      });

      const schedule = () => {
        if (schedulingEnabled && frequency !== undefined) {
          const schedule = create(ScheduleSchema, {
            frequency,
            interval: 0,
            maxExecutions: 0,
          });

          return { schedule };
        } else {
          return {};
        }
      };

      const jsonData = create(PolicySchema, {
        author: "",
        description: "",
        feePolicies: [],
        id: schema.pluginId,
        name: schema.pluginName,
        rules: [rule],
        scheduleVersion: schema.scheduleVersion,
        ...schedule(),
        version: schema.pluginVersion,
      });

      const binaryData = toBinary(PolicySchema, jsonData);

      const base64Data = Buffer.from(binaryData).toString("base64");

      const currentVauldId = getCurrentVaultId();
      const signature = await VulticonnectWalletService.signPolicy(
        base64Data,
        currentVauldId,
        0,
        String(schema.pluginVersion)
      );

      const pluginPricing = await MarketplaceService.getPluginPricing(
        plugin.pricing_id
      );

      try {
        const finalData: PluginPolicy = {
          active: true,
          feePolicies: [
            {
              start_date: new Date(Date.now()).toISOString(),
              frequency: pluginPricing.frequency,
              amount: pluginPricing.amount,
              type: pluginPricing.type,
            },
          ],
          id: uuidv4(),
          plugin_id: plugin.id,
          plugin_version: String(schema.pluginVersion),
          policy_version: 0,
          public_key: currentVauldId,
          recipe: base64Data,
          signature: signature,
        };

        await PolicyService.createPolicy(getMarketplaceUrl(), finalData);
        publish("onToast", {
          message: "Policy created",
          type: "success",
          duration: 2000,
        });
        fetchPolicies();
        onClose();
      } catch {
        publish("onToast", {
          message: "Failed to create new policy",
          type: "error",
          duration: 2000,
        });
      }
    }
  };

  useEffect(() => fetchSchema(), [plugin]);

  return loading ? (
    <div className="recipe-schema-popup">
      <div className="recipe-schema-content">
        <button className="recipe-schema-close" onClick={onClose}>
          ×
        </button>
        <div className="recipe-schema-loading">
          <div className="loading-spinner"></div>
          <div>Loading policy schema...</div>
        </div>
      </div>
    </div>
  ) : error ? (
    <div className="recipe-schema-popup">
      <div className="recipe-schema-content error">
        <button className="recipe-schema-close" onClick={onClose}>
          ×
        </button>
        <div className="recipe-schema-error">
          <div className="recipe-schema-error-icon">⚠️</div>
          <div className="recipe-schema-error-text">{error}</div>
          <Button
            size="small"
            type="button"
            styleType="primary"
            onClick={fetchSchema}
          >
            Retry
          </Button>
        </div>
      </div>
    </div>
  ) : (
    schema && (
      <div className="recipe-schema-popup">
        <div className="recipe-schema-content">
          <button className="recipe-schema-close" onClick={onClose}>
            ×
          </button>

          <div className="recipe-schema-header">
            <h2>Configure {schema.pluginName}</h2>
            <div className="plugin-info">
              <span className="plugin-version">v{schema.pluginVersion}</span>
              <span className="plugin-id">{schema.pluginId}</span>
            </div>
          </div>

          <div className="recipe-schema-form">
            {/* Resource Selection */}
            {schema.supportedResources.length > 1 && (
              <div className="form-group">
                <label className="form-label">Select Resource:</label>
                <select
                  className="form-select"
                  value={selectedResource}
                  onChange={(e) =>
                    setState((prevState) => ({
                      ...prevState,
                      selectedResource: Number(e.target.value),
                    }))
                  }
                >
                  {schema.supportedResources.map((res, i) => (
                    <option key={i} value={i}>
                      {res.resourcePath.full}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {currentResource && (
              <>
                {/* Current Resource Info */}
                <div className="resource-info">
                  <h3>Resource: {currentResource.resourcePath.full}</h3>
                  <div className="resource-details">
                    <span>Chain: {currentResource.resourcePath.chainId}</span>
                    <span>
                      Protocol: {currentResource.resourcePath.protocolId}
                    </span>
                    <span>
                      Function: {currentResource.resourcePath.functionId}
                    </span>
                  </div>
                </div>
                {/* Parameters */}
                <div className="parameters-section">
                  <h4>Parameters</h4>
                  {currentResource.parameterCapabilities.map((param, i) => (
                    <div key={i} className="form-group">
                      <label className="form-label">
                        {param.parameterName}
                        {param.required && <span className="required">*</span>}
                      </label>
                      <input
                        type="text"
                        className={`form-input ${validationErrors[param.parameterName] ? "error" : ""}`}
                        value={formData[param.parameterName] || ""}
                        onChange={(e) =>
                          handleInputChange(param.parameterName, e.target.value)
                        }
                        placeholder={`Enter ${param.parameterName}...`}
                      />
                      <div className="input-help">
                        {`Supported types: ${param.supportedTypes.map((t) => constraintTypeName[t]).join(", ")}`}
                      </div>
                      {validationErrors[param.parameterName] && (
                        <div className="error-message">
                          {validationErrors[param.parameterName]}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
                {/* Scheduling */}
                {schema.scheduling.supportsScheduling && (
                  <div className="scheduling-section">
                    <h4>Scheduling</h4>
                    <div className="form-group">
                      <label className="checkbox-label">
                        <input
                          type="checkbox"
                          checked={schedulingEnabled}
                          onChange={(e) =>
                            setState((prevState) => ({
                              ...prevState,
                              schedulingEnabled: e.target.checked,
                            }))
                          }
                        />
                        Enable scheduled execution
                      </label>
                    </div>

                    {schedulingEnabled && (
                      <div className="form-group">
                        <label className="form-label">
                          Frequency <span className="required">*</span>
                        </label>
                        <select
                          className={`form-select ${validationErrors.frequency ? "error" : ""}`}
                          value={frequency || ""}
                          onChange={(e) =>
                            setState((prevState) => ({
                              ...prevState,
                              frequency: Number(e.target.value),
                            }))
                          }
                        >
                          <option value="">Select frequency...</option>
                          {schema.scheduling.supportedFrequencies.map(
                            (freq) => (
                              <option key={freq} value={freq}>
                                {frequencyName[freq]}
                              </option>
                            )
                          )}
                        </select>
                        <div className="input-help">
                          Max {schema.scheduling.maxScheduledExecutions}{" "}
                          scheduled executions
                        </div>
                        {validationErrors.frequency && (
                          <div className="error-message">
                            {validationErrors.frequency}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )}
                {/* Requirements Info */}
                <div className="requirements-section">
                  <h4>Requirements</h4>
                  <div className="requirements-list">
                    <div>
                      Min Vultisig Version:{" "}
                      {schema.requirements.minVultisigVersion}
                    </div>
                    <div>
                      Supported Chains:{" "}
                      {schema.requirements.supportedChains.join(", ")}
                    </div>
                  </div>
                </div>
                {/* Action Buttons */}
                <div className="form-actions">
                  <Button
                    size="medium"
                    type="button"
                    styleType="secondary"
                    onClick={onClose}
                  >
                    Cancel
                  </Button>
                  <Button
                    size="medium"
                    type="button"
                    styleType="primary"
                    onClick={handleSubmit}
                  >
                    Configure Plugin
                  </Button>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    )
  );
};

export default RecipeSchemaForm;
