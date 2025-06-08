import React, { useEffect, useState } from "react";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import Button from "@/modules/core/components/ui/button/Button";
import "./recipe_Schema.styles.css";

interface RecipeSchemaProps {
  pluginId: string;
  onClose: () => void;
}

interface Parameter {
  parameter_name: string;
  required: boolean;
  supported_types: number[];
}

interface Resource {
  resource_path: {
    full: string;
    chain_id: string;
    function_id: string;
    protocol_id: string;
  };
  parameter_capabilities: Parameter[];
}

interface SchedulingInfo {
  supports_scheduling: boolean;
  supported_frequencies: number[];
  max_scheduled_executions: number;
}

interface SchemaData {
  plugin_id: string;
  plugin_name: string;
  plugin_version: number;
  supported_resources: Resource[];
  scheduling: SchedulingInfo;
  requirements: {
    min_vultisig_version: number;
    supported_chains: string[];
  };
}

const RecipeSchema: React.FC<RecipeSchemaProps> = ({ pluginId, onClose }) => {
  const [schema, setSchema] = useState<SchemaData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [selectedResource, setSelectedResource] = useState<number>(0);
  const [schedulingEnabled, setSchedulingEnabled] = useState(false);
  const [frequency, setFrequency] = useState<number | null>(null);
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  const fetchSchema = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await MarketplaceService.getRecipeSpecification(pluginId);
      setSchema(data);
      // Initialize form data
      if (data.supported_resources?.[0]) {
        const initialData: Record<string, any> = {};
        data.supported_resources[0].parameter_capabilities.forEach((param: Parameter) => {
          initialData[param.parameter_name] = '';
        });
        setFormData(initialData);
      }
    } catch (err: any) {
      setError("Failed to load policy schema");
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (paramName: string, value: string) => {
    setFormData(prev => ({ ...prev, [paramName]: value }));
    // Clear validation error when user starts typing
    if (validationErrors[paramName]) {
      setValidationErrors(prev => ({ ...prev, [paramName]: '' }));
    }
  };

  const validateForm = () => {
    const errors: Record<string, string> = {};
    const currentResource = schema?.supported_resources[selectedResource];
    
    currentResource?.parameter_capabilities.forEach((param: Parameter) => {
      if (param.required && !formData[param.parameter_name]?.trim()) {
        errors[param.parameter_name] = `${param.parameter_name} is required`;
      }
    });

    if (schedulingEnabled && !frequency) {
      errors.frequency = 'Please select a frequency for scheduled execution';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = () => {
    if (!validateForm()) {
      return;
    }

    const submission = {
      plugin_id: schema?.plugin_id,
      resource: schema?.supported_resources[selectedResource],
      parameters: formData,
      scheduling: schedulingEnabled ? { frequency, enabled: true } : { enabled: false }
    };

    console.log('Form submission:', submission);
    // TODO: Submit to backend or pass to parent
    onClose();
  };

 

  const getParameterTypeLabel = (types: number[]) => {
    const typeMap: Record<number, string> = {
      0: 'unspecified',
      1: 'fixed', 
      2: 'max',
      3: 'min',
      4: 'range',
      5: 'whitelist',
      6: 'max_per_period'
    };
    return types.map(t => typeMap[t] || `Type ${t}`).join(', ');
  };

  useEffect(() => {
    fetchSchema();
  }, [pluginId]);

  if (loading) {
    return (
      <div className="recipe-schema-popup">
        <div className="recipe-schema-content">
          <button className="recipe-schema-close" onClick={onClose}>×</button>
          <div className="recipe-schema-loading">
            <div className="loading-spinner"></div>
            <div>Loading policy schema...</div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="recipe-schema-popup">
        <div className="recipe-schema-content error">
          <button className="recipe-schema-close" onClick={onClose}>×</button>
          <div className="recipe-schema-error">
            <div className="recipe-schema-error-icon">⚠️</div>
            <div className="recipe-schema-error-text">{error}</div>
            <Button size="small" type="button" styleType="primary" onClick={fetchSchema}>
              Retry
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (!schema) return null;

  const currentResource = schema.supported_resources[selectedResource];

  return (
    <div className="recipe-schema-popup">
      <div className="recipe-schema-content">
        <button className="recipe-schema-close" onClick={onClose}>×</button>
        
        <div className="recipe-schema-header">
          <h2>Configure {schema.plugin_name}</h2>
          <div className="plugin-info">
            <span className="plugin-version">v{schema.plugin_version}</span>
            <span className="plugin-id">{schema.plugin_id}</span>
          </div>
        </div>

        <div className="recipe-schema-form">
          {/* Resource Selection */}
          {schema.supported_resources.length > 1 && (
            <div className="form-group">
              <label className="form-label">Select Resource:</label>
              <select 
                className="form-select"
                value={selectedResource}
                onChange={(e) => setSelectedResource(Number(e.target.value))}
              >
                {schema.supported_resources.map((res, i) => (
                  <option key={i} value={i}>{res.resource_path.full}</option>
                ))}
              </select>
            </div>
          )}

          {/* Current Resource Info */}
          <div className="resource-info">
            <h3>Resource: {currentResource.resource_path.full}</h3>
            <div className="resource-details">
              <span>Chain: {currentResource.resource_path.chain_id}</span>
              <span>Protocol: {currentResource.resource_path.protocol_id}</span>
              <span>Function: {currentResource.resource_path.function_id}</span>
            </div>
          </div>

          {/* Parameters */}
          <div className="parameters-section">
            <h4>Parameters</h4>
            {currentResource.parameter_capabilities.map((param, i) => (
              <div key={i} className="form-group">
                <label className="form-label">
                  {param.parameter_name}
                  {param.required && <span className="required">*</span>}
                </label>
                <input
                  type="text"
                  className={`form-input ${validationErrors[param.parameter_name] ? 'error' : ''}`}
                  value={formData[param.parameter_name] || ''}
                  onChange={(e) => handleInputChange(param.parameter_name, e.target.value)}
                  placeholder={`Enter ${param.parameter_name}...`}
                />
                <div className="input-help">
                  Supported types: {getParameterTypeLabel(param.supported_types)}
                </div>
                {validationErrors[param.parameter_name] && (
                  <div className="error-message">{validationErrors[param.parameter_name]}</div>
                )}
              </div>
            ))}
          </div>

          {/* Scheduling */}
          {schema.scheduling.supports_scheduling && (
            <div className="scheduling-section">
              <h4>Scheduling</h4>
              <div className="form-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={schedulingEnabled}
                    onChange={(e) => setSchedulingEnabled(e.target.checked)}
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
                    className={`form-select ${validationErrors.frequency ? 'error' : ''}`}
                    value={frequency || ''}
                    onChange={(e) => setFrequency(Number(e.target.value))}
                  >
                    <option value="">Select frequency...</option>
                    {schema.scheduling.supported_frequencies.map(freq => (
                      <option key={freq} value={freq}>
                        {freq === 3 ? 'Daily' : freq === 4 ? 'Weekly' : freq === 5 ? 'Monthly' : `Frequency ${freq}`}
                      </option>
                    ))}
                  </select>
                  <div className="input-help">
                    Max {schema.scheduling.max_scheduled_executions} scheduled executions
                  </div>
                  {validationErrors.frequency && (
                    <div className="error-message">{validationErrors.frequency}</div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Requirements Info */}
          <div className="requirements-section">
            <h4>Requirements</h4>
            <div className="requirements-list">
              <div>Min Vultisig Version: {schema.requirements.min_vultisig_version}</div>
              <div>Supported Chains: {schema.requirements.supported_chains.join(', ')}</div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="form-actions">
            <Button size="medium" type="button" styleType="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button size="medium" type="button" styleType="primary" onClick={handleSubmit}>
              Configure Plugin
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default RecipeSchema; 