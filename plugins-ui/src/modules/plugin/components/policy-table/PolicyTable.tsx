import Button from "@/modules/core/components/ui/button/Button";
import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { PluginPolicy } from "../../models/policy";

interface InitialState {
  data: PluginPolicy[];
}

const PolicyTable = () => {
  const initialState: InitialState = {
    data: [],
  };
  const [state, setState] = useState(initialState);
  const { data } = state;
  const { pluginId } = useParams<{ pluginId: string }>();

  useEffect(() => {
    if (pluginId) {
      setState((prevState) => ({ ...prevState, loading: true }));

      MarketplaceService.getPolicies(pluginId, 0, 10)
        .then(({ policies }) => {
          setState((prevState) => ({
            ...prevState,
            data: policies,
            loading: false,
          }));
          console.log("policies", policies);
        })
        .catch(() => {
          setState((prevState) => ({ ...prevState, loading: false }));
        });
    }
  }, [pluginId]);

  return (
    <table className="policy-table">
      <thead>
        <tr>
          <th>Row</th>
          <th>ID</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
        {data.map(({ id }, ind) => (
          <tr key={ind}>
            <td>{ind + 1}</td>
            <td>{id}</td>
            <td>
              <Button
                children="x"
                size="small"
                styleType="primary"
                type="button"
                style={{ margin: "0 auto", padding: "0.5rem 1rem" }}
              />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
};

export default PolicyTable;
