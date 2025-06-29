/// <reference types="vite-plugin-svgr/client" />

import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import Marketplace from "./modules/marketplace/components/marketplace-main/Marketplace";
import Layout from "./Layout";
import PluginDetail from "./modules/plugin/components/plugin-detail/PluginDetail";
import { PolicyProvider } from "./modules/policy/context/PolicyProvider";

const App = () => {
  return (
    <BrowserRouter>
      <Routes>
        {/* Redirect / to /plugins */}
        <Route path="/" element={<Navigate to="/plugins" replace />} />
        <Route path="/plugins" element={<Layout />}>
          <Route index element={<Marketplace />} />
          <Route
            path=":pluginId"
            element={
              <PolicyProvider>
                <PluginDetail />
              </PolicyProvider>
            }
          />
        </Route>
      </Routes>
    </BrowserRouter>
  );
};

export default App;
