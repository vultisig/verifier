type Tag = {
  id: string;
  name: string;
  color: string;
};

export type PluginRatings = {
  rating: number;
  count: number;
};
export type Plugin = {
  id: string;
  type: string;
  title: string;
  description: string;
  metadata: {};
  server_endpoint: string;
  pricing_id: string;
  category_id: string;
  tags: Tag[];
  ratings: PluginRatings[];
};

export type PluginPricing = {
  id: string;
  type: string;
  created_at: string;
  updated_at: string;
  frequency: string;
  amount: number;
  metric: string;
};
