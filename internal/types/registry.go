package types

// Each plugin must reserve a unique ID.
//
// A plugin ID is comprised of a all-lowercase string, ending with 4 random hex digits.
//
// Example: vultisig-dca-00a1
const (
	PluginVultisigFees           = "vultisig-fees-feee"
	PluginVultisigRecurringSends = "vultisig-recurring-sends-0000"
)
