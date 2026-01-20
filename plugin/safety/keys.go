package safety

func KeysignFlagKey(pluginID string) string { return pluginID + "-keysign" }
func KeygenFlagKey(pluginID string) string  { return pluginID + "-keygen" }
func GlobalKeysignKey() string              { return "global-keysign" }
func GlobalKeygenKey() string               { return "global-keygen" }
