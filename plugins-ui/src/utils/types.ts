import type * as CSS from "csstype";

export type AuthTokenForm = {
  chainCodeHex: string;
  message: string;
  publicKey: string;
  signature: string;
};

export type Category = {
  id: string;
  name: string;
};

export type Property = {
  enum: string[];
  format: string;
  type: string;
};

export type Configuration = {
  properties: Record<string, Property>;
  required: string[];
  type: "object";
};

export type Plugin = {
  categoryId: string;
  createdAt: string;
  description: string;
  id: string;
  pricing: PluginPricing[];
  rating: { count: number; rate: number };
  ratings: Rating[];
  serverEndpoint: string;
  title: string;
  updatedAt: string;
};

export type PluginPolicy = {
  active: boolean;
  id: string;
  pluginVersion: string;
  pluginId: string;
  policyVersion: number;
  publicKey: string;
  recipe: string;
  signature?: string;
};

export type PluginPricing = {
  amount: number;
  createdAt: string;
  frequency: string;
  id: string;
  metric: string;
  type: string;
  updatedAt: string;
};

export type PolicyTransactionHistory = {
  id: string;
  status: string;
  updatedAt: string;
};

export type Rating = {
  count: number;
  rating: number;
};

export type ReshareForm = {
  email: string;
  hexChainCode: string;
  hexEncryptionKey: string;
  localPartyId: string;
  name: string;
  oldParties: string[];
  pluginId: string;
  publicKey: string;
  sessionId: string;
};

export type Review = {
  address: string;
  comment: string;
  createdAt: string;
  id: string;
  pluginId: string;
  rating: number;
};

export type ReviewForm = {
  address: string;
  comment: string;
  rating: number;
};

export type Tag = {
  color: string;
  id: string;
  name: string;
};

export type Vault = {
  hexChainCode: string;
  name: string;
  publicKeyEcdsa: string;
  publicKeyEddsa: string;
  uid: string;
};

export type PluginFilters = {
  category: string;
  sort: string;
  term: string;
};

export type CSSProperties = CSS.Properties<string>;
