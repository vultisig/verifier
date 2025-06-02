import React, { useEffect, useState } from "react";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import Button from "@/modules/core/components/ui/button/Button";
import "./recipe_Schema.styles.css";

interface RecipeSchemaProps {
  pluginId: string;
  onClose: () => void;
}

const RecipeSchema: React.FC<RecipeSchemaProps> = ({ pluginId, onClose }) => {
  const [schema, setSchema] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSchema = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await MarketplaceService.getRecipeSpecification(pluginId);
      setSchema(data);
    } catch (err: any) {
      setError("Failed to load policy schema");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSchema();
    // eslint-disable-next-line
  }, [pluginId]);

  return (
    <div className="recipe-schema-popup">
      <div className={`recipe-schema-content${error ? " error" : ""}`}>
        <button className="recipe-schema-close" onClick={onClose} aria-label="Close popup">×</button>
        <h2>Policy Schema</h2>
        {loading && <div>Loading...</div>}
        {error && (
          <div className="recipe-schema-error">
            <div className="recipe-schema-error-icon">⚠️</div>
            <div className="recipe-schema-error-text">{error}</div>
            <Button size="small" type="button" styleType="primary" onClick={fetchSchema}>
              Retry
            </Button>
          </div>
        )}
        {schema && !error && (
          <div style={{ textAlign: 'left', marginTop: 16 }}>
            <div><b>Plugin:</b> {schema.plugin_name} (v{schema.plugin_version})</div>
            <div><b>Supported Resources:</b></div>
            <ul>
              {schema.supported_resources?.map((res: any, i: number) => (
                <li key={i}>
                  <div><b>Resource:</b> {res.resource_path?.full}</div>
                  <div>Parameters:</div>
                  <ul>
                    {res.parameter_capabilities?.map((param: any, j: number) => (
                      <li key={j}>
                        <b>{param.parameter_name}</b> (Required: {param.required ? 'Yes' : 'No'})<br />
                        Supported Types: {param.supported_types?.join(", ")}
                      </li>
                    ))}
                  </ul>
                </li>
              ))}
            </ul>
            {schema.scheduling && (
              <div>
                <b>Scheduling:</b> {schema.scheduling.supports_scheduling ? 'Yes' : 'No'}<br />
                Supported Frequencies: {schema.scheduling.supported_frequencies?.join(", ")}
                <br />Max Scheduled Executions: {schema.scheduling.max_scheduled_executions}
              </div>
            )}
            {schema.requirements && (
              <div>
                <b>Requirements:</b><br />
                Min Vultisig Version: {schema.requirements.min_vultisig_version}<br />
                Supported Chains: {schema.requirements.supported_chains?.join(", ")}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default RecipeSchema; 