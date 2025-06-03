package types

type PluginID string

// Each plugin must reserve a unique ID.
//
// A plugin ID is comprised of a all-lowercase string, ending with 4 random hex digits.
//
// Example: vultisig-dca-00a1
const (
	PluginVultisigDCA_0000     PluginID = "vultisig-dca-0000"
	PluginVultisigPayroll_0000 PluginID = "vultisig-payroll-0000"
	PluginVultisigFees_feee    PluginID = "vultisig-fees-feee"
)

func (p PluginID) String() string {
	return string(p)
}
