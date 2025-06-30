export type PluginCreatePolicyProps = {
  recipe: string; // base64 encoded string
  publicKey: string;
  policyVersion: number;
  pluginVersion: string;
};

export function policyToHexMessage(policy: PluginCreatePolicyProps): string {
  const delimiter = "*#*";

  const fields = [
    policy.recipe,
    policy.publicKey,
    String(policy.policyVersion),
    policy.pluginVersion,
  ];

  for (const item of fields) {
    if (item.includes(delimiter)) {
      throw new Error("invalid policy signature");
    }
  }

  return fields.join(delimiter);
}
