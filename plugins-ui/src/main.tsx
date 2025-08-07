import "styles/normalize.css";
import "styles/brockmann.css";

import { App } from "App";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { StrictMode } from "react";
import ReactDOM from "react-dom/client";

dayjs.extend(utc);

if (import.meta.env.DEV) {
  ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
} else {
  ReactDOM.createRoot(document.getElementById("root")!).render(
    <StrictMode>
      <App />
    </StrictMode>
  );
}
