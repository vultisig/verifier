import React, { useEffect, useState } from "react";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import Button from "@/modules/core/components/ui/button/Button";
import "./recipe_Schema.styles.css";
import {
  BillingFrequency,
  FeePolicy,
  FeePolicySchema,
  FeeType,
  PolicySchema,
  ScheduleSchema,
} from "@/gen/policy_pb";
import { ScheduleFrequency } from "@/gen/scheduling_pb";
import { ConstraintSchema, ConstraintType } from "@/gen/constraint_pb";
import { Effect, RuleSchema } from "@/gen/rule_pb";
import { create, toBinary } from "@bufbuild/protobuf";
import { constraintTypeName, frequencyName } from "@/utils/constants";
import { ParameterConstraintSchema } from "@/gen/parameter_constraint_pb";
import { getCurrentVaultId } from "@/storage/currentVaultId";
import { RecipeSchema } from "@/gen/recipe_specification_pb";
import { v4 as uuidv4 } from "uuid";
import { Plugin } from "../../models/plugin";
import { publish } from "@/utils/eventBus";
import { PluginPolicy } from "../../models/policy";
import { usePolicies } from "@/modules/policy/context/PolicyProvider";
import { toProtoTimestamp } from "@/utils/functions";

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
  const [startDate, setStartDate] = useState(() => {
    const now = new Date();
    now.setSeconds(0, 0);
    const offset = now.getTimezoneOffset() * 60000;
    const localTime = new Date(now.getTime() - offset);
    return localTime.toISOString().slice(0, 16);
  });

  const [useNextMonthStart, setUseNextMonthStart] = useState(false);
  const [state, setState] = useState(initialState);
  const { fetchPolicies, addPolicy } = usePolicies();
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
    // Validate start date is in the future
    if (!useNextMonthStart && new Date(startDate) <= new Date()) {
      validationErrors.startDate = "Start date must be in the future";
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
            denominatedIn:
              currentResource.resourcePath?.chainId.toLowerCase() === "ethereum"
                ? "wei"
                : "",
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
        resource: currentResource.resourcePath?.full,
      });

      let feePolicies: FeePolicy[] = [];

      for (const price of plugin.pricing) {
        let ft = FeeType.FEE_TYPE_UNSPECIFIED;
        switch (price.type) {
          case "once":
            ft = FeeType.ONCE;
            break;
          case "recurring":
            ft = FeeType.RECURRING;
            break;
          case "per-tx":
            ft = FeeType.TRANSACTION;
            break;
        }

        let bf = BillingFrequency.BILLING_FREQUENCY_UNSPECIFIED;
        switch (price.frequency) {
          case "daily":
            bf = BillingFrequency.DAILY;
            break;
          case "weekly":
            bf = BillingFrequency.WEEKLY;
            break;
          case "biweekly":
            bf = BillingFrequency.BIWEEKLY;
            break;
          case "monthly":
            bf = BillingFrequency.MONTHLY;
            break;
        }

        const feePolicy = create(FeePolicySchema, {
          id: uuidv4(),
          type: ft,
          frequency: bf,
          amount: BigInt(price.amount),
          startDate: toProtoTimestamp(new Date(startDate + ":00")),
          description: "",
        });

        feePolicies.push(feePolicy);
      }

      const schedule = () => {
        const schedule = create(ScheduleSchema, {
          frequency,
          interval: 1,
          maxExecutions: -1,
          startTime: toProtoTimestamp(new Date(startDate + ":00")),
        });
        return { schedule };
      };

      const jsonData = create(PolicySchema, {
        author: "",
        description: "",
        feePolicies: feePolicies,
        id: schema.pluginId,
        name: schema.pluginName,
        rules: [rule],
        scheduleVersion: schema.scheduleVersion,
        ...schedule(),
        version: schema.pluginVersion,
      });

      const binaryData = toBinary(PolicySchema, jsonData);

      const base64Data = Buffer.from(binaryData).toString("base64");

      const currentVaultId = getCurrentVaultId();

      const finalData: PluginPolicy = {
        active: true,
        id: uuidv4(),
        plugin_id: plugin.id,
        plugin_version: String(schema.pluginVersion),
        policy_version: 0,
        public_key: currentVaultId,
        recipe: base64Data,
      };

      const addPolicyResponse = await addPolicy(finalData);
      if (addPolicyResponse) {
        publish("onToast", {
          message: "Policy created",
          type: "success",
        });
        fetchPolicies();
        onClose();
      } else {
        publish("onToast", {
          message: "Failed to create new policy",
          type: "error",
        });
      }
    } else {
      setState((prevState) => ({
        ...prevState,
        error: "Unable to validate policy",
        loading: false,
      }));
    }
  };

  useEffect(() => {
    if (useNextMonthStart) {
      const now = new Date();
      const firstOfNextMonth = new Date(
        now.getFullYear(),
        now.getMonth() + 1,
        1,
        0,
        0,
        0
      );
      setStartDate(firstOfNextMonth.toISOString().slice(0, 16));
    }
  }, [useNextMonthStart]);

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
                      {res.resourcePath?.full}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {currentResource && (
              <>
                {/* Current Resource Info */}
                <div className="resource-info">
                  <h3>Resource: {currentResource.resourcePath?.full}</h3>
                  <div className="resource-details">
                    <span>Chain: {currentResource.resourcePath?.chainId}</span>
                    <span>
                      Protocol: {currentResource.resourcePath?.protocolId}
                    </span>
                    <span>
                      Function: {currentResource.resourcePath?.functionId}
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
                <h4>Scheduling</h4>
                <div className="form-group">
                  <label className="form-label">
                    Start Date <span className="required">*</span>
                  </label>
                  <label className="checkbox-label">
                    <input
                      type="checkbox"
                      checked={useNextMonthStart}
                      onChange={(e) => setUseNextMonthStart(e.target.checked)}
                    />
                    Start from the beginning of next month
                  </label>

                  {!useNextMonthStart && (
                    <>
                      <input
                        type="datetime-local"
                        className="form-input"
                        value={startDate}
                        onChange={(e) => {
                          setState((prevState) => ({
                            ...prevState,
                            validationErrors: {
                              ...prevState.validationErrors,
                              startDate:
                                new Date(e.target.value) <= new Date()
                                  ? "Start date must be in the future"
                                  : "",
                            },
                          }));
                          setStartDate(e.target.value);
                        }}
                        min={new Date().toISOString().slice(0, 16)}
                      />
                      {validationErrors.startDate && (
                        <div className="error-message">
                          {validationErrors.startDate}
                        </div>
                      )}
                    </>
                  )}
                </div>

                {/* Scheduling */}
                {schema.scheduling?.supportsScheduling && (
                  <div className="scheduling-section">
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
                      {schema.requirements?.minVultisigVersion}
                    </div>
                    <div>
                      Supported Chains:{" "}
                      {schema.requirements?.supportedChains.join(", ")}
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
