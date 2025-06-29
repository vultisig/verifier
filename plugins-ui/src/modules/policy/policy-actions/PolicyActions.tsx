import Button from "@/modules/core/components/ui/button/Button";
import TrashIcon from "@/assets/Trash.svg?react";
import { PluginPolicy } from "@/modules/plugin/models/policy";

import { getMarketplaceUrl } from "@/modules/marketplace/services/marketplaceService";
import { publish } from "@/utils/eventBus";
import PolicyService from "../services/policyService";

type PolicyActionsProps = {
  policy: PluginPolicy;
};

const PolicyActions = ({ policy }: PolicyActionsProps) => {
  const handleRemove = async (policy: PluginPolicy) => {
    try {
      await PolicyService.deletePolicy(
        getMarketplaceUrl(),
        policy.id,
        policy.signature
      );
      publish("onToast", {
        message: "Policy removed",
        type: "success",
        duration: 2000,
      });
    } catch {
      publish("onToast", {
        message: "Failed to remove policy",
        type: "error",
        duration: 2000,
      });
    }
  };

  return (
    <>
      <div style={{ display: "flex", justifyContent: "end" }}>
        <Button
          ariaLabel="Delete policy"
          type="button"
          styleType="tertiary"
          size="small"
          style={{ color: "#DA2E2E", padding: "5px", margin: "0 5px" }}
          onClick={() => handleRemove(policy)}
        >
          <TrashIcon width="20px" height="20px" color="#FF5C5C" />
        </Button>
      </div>

      {/* <Modal
        isOpen={editModalId !== ""}
        onClose={() => setEditModalId("")}
        variant="panel"
      >
        <PolicyForm
          data={policyMap.get(editModalId)}
          onSubmitCallback={() => setEditModalId("")}
        />
      </Modal> */}
      {/* <Modal
        isOpen={transactionHistoryModalId !== ""}
        onClose={() => setTransactionHistoryModalId("")}
        variant="panel"
      >
        <TransactionHistory policyId={transactionHistoryModalId} />
      </Modal> */}
    </>
  );
};

export default PolicyActions;
