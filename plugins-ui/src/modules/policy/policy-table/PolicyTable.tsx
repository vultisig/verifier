import MarketplaceService from "@/modules/marketplace/services/marketplaceService";
import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { PluginPolicy } from "../../plugin/models/policy";

import "./PolicyTable.css";

import PolicyActions from "../policy-actions/PolicyActions";
import { base64Decode } from "@bufbuild/protobuf/wire";
import { fromBinary } from "@bufbuild/protobuf";

import { PolicySchema } from "@/gen/policy_pb";
import { POLICY_ITEMS_PER_PAGE, usePolicies } from "../context/PolicyProvider";
import Pagination from "@/modules/core/components/ui/pagination/Pagination";

interface InitialState {
  data: PluginPolicy[];
  tableHeaders: string[];
}

const PolicyTable = () => {
  const initialState: InitialState = {
    data: [],
    tableHeaders: [],
  };
  const [state, setState] = useState(initialState);
  const { policyMap, currentPage, policiesTotalCount, setCurrentPage } =
    usePolicies();
  const { tableHeaders } = state;
  const { pluginId } = useParams<{ pluginId: string }>();
  const [totalPages, setTotalPages] = useState(0);

  useEffect(() => {
    if (pluginId) {
      setState((prevState) => ({ ...prevState, loading: true }));
      MarketplaceService.getRecipeSpecification(pluginId).then((schema) => {
        let headers: string[] = [];
        if (schema.supportedResources?.[0]?.parameterCapabilities) {
          schema.supportedResources[0].parameterCapabilities.forEach(
            (param) => {
              // Capitalize first letter and push
              headers.push(
                param.parameterName.charAt(0).toUpperCase() +
                  param.parameterName.slice(1)
              );
            }
          );
        }
        setState((prevState) => ({ ...prevState, tableHeaders: headers }));
      });
    }
  }, [pluginId]);

  useEffect(() => {
    setTotalPages(Math.ceil(policiesTotalCount / POLICY_ITEMS_PER_PAGE));
    if (policiesTotalCount / POLICY_ITEMS_PER_PAGE > 1 && currentPage === 0) {
      setCurrentPage(1);
    }
  }, [currentPage, policiesTotalCount]);

  const onCurrentPageChange = (page: number): void => {
    setCurrentPage(page);
  };

  return (
    policyMap &&
    [...policyMap.values()].length > 0 &&
    tableHeaders && (
      <div style={{ width: "100%" }}>
        <table className="policy-table">
          <thead>
            <tr key={0}>
              <th>Index</th>
              {tableHeaders.map((header, idx) => (
                <th key={header + idx}>{header}</th>
              ))}
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {[...policyMap.values()].map((policy, index) => (
              <PolicyItem key={policy.id} policy={policy} index={index} />
            ))}
          </tbody>
        </table>

        {totalPages > 1 && (
          <Pagination
            currentPage={currentPage}
            totalPages={totalPages}
            onPageChange={onCurrentPageChange}
          />
        )}
      </div>
    )
  );
};
type PolicyItemProps = {
  policy: PluginPolicy;
  index: number;
};
const PolicyItem = ({ policy, index }: PolicyItemProps) => {
  const [policyValues, setPolicyValues] = useState<any[]>();
  useEffect(() => {
    const decoded = base64Decode(policy.recipe);
    const info = fromBinary(PolicySchema, decoded);
    const values = info.rules[0].parameterConstraints.map(
      (param) => param.constraint!.value.value
    );
    setPolicyValues(values);
  }, []);
  return (
    <tr>
      <td>{index + 1}</td>
      {policyValues?.map((value, index) => (
        <td key={index + value}>{value}</td>
      ))}
      <td>
        <PolicyActions policy={policy} />
      </td>
    </tr>
  );
};

export default PolicyTable;
