import Button from "@/modules/core/components/ui/button/Button";
import { useState, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { useWallet } from "./WalletProvider";
import {
  DiscordIcon,
  TwitterIcon,
  GitHubIcon,
  DisconnectIcon,
} from "../icons/SocialIcons";
import "./wallet.styles.css";

const Wallet = () => {
  const { address, isConnected, connect, disconnect } = useWallet();

  // Add copy-to-clipboard logic
  const [copyTooltip, setCopyTooltip] = useState("Copy");
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [dropdownPosition, setDropdownPosition] = useState({
    top: 0,
    right: 0,
  });
  const containerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const handleCopy = () => {
    if (address) {
      navigator.clipboard.writeText(address);
      setCopyTooltip("Copied!");
      setTimeout(() => setCopyTooltip("Copy"), 1200);
    }
  };

  const toggleDropdown = () => {
    if (!isDropdownOpen && containerRef.current) {
      const rect = containerRef.current.getBoundingClientRect();
      setDropdownPosition({
        top: rect.bottom + 8,
        right: window.innerWidth - rect.right,
      });
    }
    setIsDropdownOpen(!isDropdownOpen);
  };

  const handleDisconnect = () => {
    disconnect();
    setIsDropdownOpen(false);
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node) &&
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setIsDropdownOpen(false);
      }
    };

    if (isDropdownOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isDropdownOpen]);

  // Close dropdown on scroll/resize
  useEffect(() => {
    const handleScroll = () => setIsDropdownOpen(false);
    const handleResize = () => setIsDropdownOpen(false);

    if (isDropdownOpen) {
      window.addEventListener("scroll", handleScroll, true);
      window.addEventListener("resize", handleResize);
    }

    return () => {
      window.removeEventListener("scroll", handleScroll, true);
      window.removeEventListener("resize", handleResize);
    };
  }, [isDropdownOpen]);

  const dropdownElement = isDropdownOpen ? (
    <div
      ref={dropdownRef}
      className="wallet-dropdown-portal"
      style={{
        position: "fixed",
        top: `${dropdownPosition.top}px`,
        right: `${dropdownPosition.right}px`,
        zIndex: 10000,
      }}
    >
      <div className="wallet-dropdown">
        <div className="wallet-dropdown-section">
          <h4>Community</h4>
          <a
            href="https://discord.gg/MYM3MapXD2"
            target="_blank"
            rel="noopener noreferrer"
            className="wallet-dropdown-link"
          >
            <DiscordIcon />
            Discord
          </a>
          <a
            href="https://twitter.com/vultisig"
            target="_blank"
            rel="noopener noreferrer"
            className="wallet-dropdown-link"
          >
            <TwitterIcon />
            Twitter
          </a>
          <a
            href="https://github.com/vultisig"
            target="_blank"
            rel="noopener noreferrer"
            className="wallet-dropdown-link"
          >
            <GitHubIcon />
            GitHub
          </a>
        </div>

        <div className="wallet-dropdown-divider"></div>

        <button onClick={handleDisconnect} className="wallet-disconnect-btn">
          <DisconnectIcon />
          Disconnect Wallet
        </button>
      </div>
    </div>
  ) : null;

  return (
    <>
      {isConnected && address ? (
        <div className="wallet-container" ref={containerRef}>
          <div className="wallet-address-container" onClick={toggleDropdown}>
            <div className="wallet-address-pill">
              {address.slice(0, 6)}...{address.slice(-4)}
            </div>
            <button
              onClick={(e) => {
                e.stopPropagation();
                handleCopy();
              }}
              className={`wallet-copy-btn${copyTooltip === "Copied!" ? " copied" : ""}${copyTooltip !== "Copy" ? " show-tooltip" : ""}`}
              title={copyTooltip}
              type="button"
            >
              <span className="wallet-copy-tooltip">{copyTooltip}</span>
              <svg
                width="18"
                height="18"
                viewBox="0 0 20 20"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <rect
                  x="5"
                  y="5"
                  width="10"
                  height="12"
                  rx="3"
                  fill="#64748b"
                />
                <rect x="8" y="2" width="9" height="12" rx="2" fill="#cbd5e1" />
              </svg>
            </button>
          </div>

          {dropdownElement && createPortal(dropdownElement, document.body)}
        </div>
      ) : (
        <Button
          size="medium"
          styleType="primary"
          type="button"
          onClick={() => connect()}
        >
          Connect Wallet
        </Button>
      )}
    </>
  );
};

export default Wallet;
