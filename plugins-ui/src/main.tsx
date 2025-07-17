import "utils/prototypes";
import "antd/dist/reset.css";

import { App } from "App";
import { StrictMode } from "react";
import ReactDOM from "react-dom/client";

if (import.meta.env.DEV) {
  ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
} else {
  ReactDOM.createRoot(document.getElementById("root")!).render(
    <StrictMode>
      <App />
    </StrictMode>
  );
}
