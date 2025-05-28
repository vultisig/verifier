import { Outlet, useNavigate } from "react-router-dom";
import Wallet from "./modules/shared/wallet/Wallet";
import "./Layout.css";
import Toast from "./modules/core/components/ui/toast/Toast";

const Layout = () => {
  const navigate = useNavigate();
  return (
    <div className="container">
      <Toast />
      <div className="navbar">
        <span
          style={{ cursor: "pointer" }}
          onClick={() => navigate("/plugins")}
        >
          Vultisig
        </span>
        <Wallet />
      </div>
      <div className="content">
        <Outlet />
      </div>
    </div>
  );
};

export default Layout;
