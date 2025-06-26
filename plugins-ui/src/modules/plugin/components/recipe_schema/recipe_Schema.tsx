import React, { useEffect, useState } from "react";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import Button from "@/modules/core/components/ui/button/Button";
import "./recipe_Schema.styles.css";
import { SchemaProps } from "@/utils/interfaces";
import { Policy } from "@/gen/policy_pb";
import { ScheduleFrequency } from "@/gen/scheduling_pb";
import { ConstraintType } from "@/gen/constraint_pb";
import { Effect } from "@/gen/rule_pb";

interface InitialState {
  error?: string;
  frequency?: ScheduleFrequency;
  formData: Record<string, string>;
  loading: boolean;
  selectedResource: number;
  schedulingEnabled: boolean;
  schema?: SchemaProps;
  submitting?: boolean;
  validationErrors: Record<string, string>;
}

interface RecipeSchemaProps {
  onClose: () => void;
  pluginId: string;
}

const RecipeSchema: React.FC<RecipeSchemaProps> = ({ pluginId, onClose }) => {
  const initialState: InitialState = {
    formData: {},
    loading: true,
    selectedResource: 0,
    schedulingEnabled: false,
    validationErrors: {},
  };
  const [state, setState] = useState(initialState);
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

    MarketplaceService.getRecipeSpecification(pluginId)
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

    if (schedulingEnabled && !frequency) {
      validationErrors.frequency =
        "Please select a frequency for scheduled execution";
    }

    setState((prevState) => ({ ...prevState, validationErrors }));

    return Object.keys(validationErrors).length === 0;
  };

  const getParameterTypeLabel = (types: number[]) => {
    const typeMap: Record<number, string> = {
      0: "unspecified",
      1: "fixed",
      2: "max",
      3: "min",
      4: "range",
      5: "whitelist",
      6: "max_per_period",
    };
    return types.map((t) => typeMap[t] || `Type ${t}`).join(", ");
  };

  const getFrequencyLabel = (frequency: number) => {
    const frequencyMap: Record<number, string> = {
      0: "Unspecified",
      1: "Hourly",
      2: "Daily",
      3: "Weekly",
      4: "Biweekly",
      5: "Monthly",
    };
    return frequencyMap[frequency] || `Frequency ${frequency}`;
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

  const handleSubmit = () => {
    if (currentResource && schema && formValidation()) {
      const policyData: Policy = {
        $typeName: "types.Policy",
        author: "",
        description: "",
        feePolicies: [],
        id: schema.pluginId,
        name: schema.pluginName,
        rules: [
          {
            $typeName: "types.Rule",
            constraints: {},
            description: "",
            effect: Effect.ALLOW,
            id: "",
            parameterConstraints: currentResource.parameterCapabilities.map(
              ({ parameterName, required }) => ({
                $typeName: "types.ParameterConstraint",
                constraint: {
                  $typeName: "types.Constraint",
                  denominatedIn: "",
                  period: "",
                  required,
                  type: ConstraintType.FIXED,
                  value: {
                    case: "fixedValue",
                    value: formData[parameterName],
                  },
                },
                parameterName,
              })
            ),
            resource: currentResource.resourcePath.full,
          },
        ],
        scheduleVersion: schema.scheduleVersion,
        ...(schedulingEnabled &&
          frequency && {
            schedule: {
              $typeName: "types.Schedule",
              frequency,
              interval: 0,
              maxExecutions: 0,
            },
          }),
        version: schema.pluginVersion,
      };

      console.log("Exported data:", policyData);
      // TODO: Submit to backend or pass to parent
      //onClose();
    }
  };

  useEffect(() => fetchSchema(), [pluginId]);

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
  ) : schema ? (
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

          {currentResource ? (
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
                      Supported types:{" "}
                      {getParameterTypeLabel(param.supportedTypes)}
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
                        {schema.scheduling.supportedFrequencies.map((freq) => (
                          <option key={freq} value={freq}>
                            {getFrequencyLabel(freq)}
                          </option>
                        ))}
                      </select>
                      <div className="input-help">
                        Max {schema.scheduling.maxScheduledExecutions} scheduled
                        executions
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
          ) : (
            <></>
          )}
        </div>
      </div>
    </div>
  ) : (
    <></>
  );
};

export default RecipeSchema;
