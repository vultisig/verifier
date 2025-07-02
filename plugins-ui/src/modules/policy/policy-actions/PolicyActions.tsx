import Button from "@/modules/core/components/ui/button/Button";
import TrashIcon from "@/assets/Trash.svg?react";
import { PluginPolicy } from "@/modules/plugin/models/policy";
import { usePolicies } from "../context/PolicyProvider";
import Modal from "@/modules/core/components/ui/modal/Modal";
import { useState } from "react";

type PolicyActionsProps = {
  policy: PluginPolicy;
};

const PolicyActions = ({ policy }: PolicyActionsProps) => {
  const [removeModalOpen, setRemoveModalOpen] = useState<boolean>(false);
  const { removePolicy } = usePolicies();
  const handleConfirmRemove = async (policyId: string) => {
    await removePolicy(policyId);
    setRemoveModalOpen(false);
  };

  const handleRemove = () => {
    setRemoveModalOpen(true);
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
          onClick={() => handleRemove()}
        >
          <TrashIcon width="20px" height="20px" color="#FF5C5C" />
        </Button>
      </div>

      <Modal
        isOpen={removeModalOpen}
        onClose={() => setRemoveModalOpen(false)}
        variant="modal"
      >
        <>
          <h4 className="">{`Are you sure you want to remove this policy from the plugin?`}</h4>
          <div className="modal-actions">
            <Button
              ariaLabel="Delete policy"
              className="button secondary medium"
              type="button"
              styleType="tertiary"
              size="small"
              onClick={() => setRemoveModalOpen(false)}
            >
              Cancel
            </Button>
            <Button
              size="small"
              type="button"
              styleType="danger"
              onClick={() => handleConfirmRemove(policy.id)}
            >
              Confirm
            </Button>
          </div>
        </>
      </Modal>
    </>
  );
};

export default PolicyActions;
