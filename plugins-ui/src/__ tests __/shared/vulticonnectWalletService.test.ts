import VulticonnectWalletService from "@/modules/shared/wallet/vulticonnectWalletService";
import { describe, it, expect, vi, afterEach } from "vitest";

const hoisted = vi.hoisted(() => ({
  mockEventBus: {
    publish: vi.fn(),
  },
}));

vi.mock("@/utils/eventBus", () => ({
  publish: hoisted.mockEventBus.publish,
}));

describe("VulticonnectWalletService", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("connectToVultiConnect", () => {
    it("should alert if no provider is found", async () => {
      vi.spyOn(window, "alert").mockImplementation(() => {});
      delete (window as any).vultisig;

      await VulticonnectWalletService.connect();

      expect(hoisted.mockEventBus.publish).toHaveBeenCalledWith("onToast", {
        message: "No ethereum provider found. Please install VultiConnect.",
        type: "error",
      });
    });

    it("should return accounts if provider exists", async () => {
      const mockAccounts = ["0x123", "0x456"];
      (window as any).vultisig = {
        ethereum: {
          request: vi.fn().mockResolvedValue(mockAccounts),
        },
      };

      const accounts = await VulticonnectWalletService.connect();

      expect(accounts).toEqual(mockAccounts);
    });

    it("should log error and throw when request fails", async () => {
      const error: { code: number; message: string } = {
        code: 401,
        message: "User rejected request",
      };
      const consoleErrorSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      (window as any).vultisig = {
        ethereum: {
          request: vi.fn().mockRejectedValue(error),
        },
      };

      await expect(VulticonnectWalletService.connect()).rejects.toThrowError();

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        `Connection failed - Code: 401, Message: User rejected request`
      );
    });
  });

  describe("getConnectedEthAccounts", () => {
    it("should alert if no provider is found", async () => {
      vi.spyOn(window, "alert").mockImplementation(() => {});
      delete (window as any).vultisig;

      await VulticonnectWalletService.getAccount();

      expect(hoisted.mockEventBus.publish).toHaveBeenCalledWith("onToast", {
        message: "No ethereum provider found. Please install VultiConnect.",
        type: "error",
      });
    });

    it("should return accounts if provider exists", async () => {
      const mockAccounts = ["0x123", "0x456"];
      (window as any).vultisig = {
        ethereum: {
          request: vi.fn().mockResolvedValue(mockAccounts),
        },
      };

      const accounts = await VulticonnectWalletService.getAccount();

      expect(accounts).toEqual(mockAccounts);
    });

    it("should log error and throw when request fails", async () => {
      const error: { code: number; message: string } = {
        code: 401,
        message: "User rejected request",
      };
      const consoleErrorSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      (window as any).vultisig = {
        ethereum: {
          request: vi.fn().mockRejectedValue(error),
        },
      };

      await expect(
        VulticonnectWalletService.getAccount()
      ).rejects.toThrowError();

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        `Failed to get accounts - Code: 401, Message: User rejected request`
      );
    });
  });

  describe("signCustomMessage", () => {
    it("should alert if no provider is found", async () => {
      vi.spyOn(window, "alert").mockImplementation(() => {});
      delete (window as any).vultisig;

      await VulticonnectWalletService.signCustomMessage(
        "hexMessage",
        "walletAddress"
      );

      expect(hoisted.mockEventBus.publish).toHaveBeenCalledWith("onToast", {
        message: "No ethereum provider found. Please install VultiConnect.",
        type: "error",
      });
    });

    it("should return signature if provider exists", async () => {
      const mockedsignature = "signature";
      (window as any).vultisig = {
        ethereum: {
          request: vi.fn().mockResolvedValue(mockedsignature),
        },
      };

      const signature = await VulticonnectWalletService.signCustomMessage(
        "hexMessage",
        "walletAddress"
      );

      expect(signature).toEqual(mockedsignature);
    });

    it("should log error and throw when request fails", async () => {
      const error: { code: number; message: string } = {
        code: 401,
        message: "User rejected request",
      };
      const consoleErrorSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      (window as any).vultisig = {
        ethereum: {
          request: vi.fn().mockRejectedValue(error),
        },
      };

      await expect(
        VulticonnectWalletService.signCustomMessage(
          "hexMessage",
          "walletAddress"
        )
      ).rejects.toThrowError();

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        `Failed to sign the message`,
        {
          code: 401,
          message: "User rejected request",
        }
      );
    });

    it("should log error and throw when request return signature with error", async () => {
      const mockedsignature = {
        error: "missing param",
      };

      const consoleErrorSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      (window as any).vultisig = {
        ethereum: {
          request: vi.fn().mockResolvedValue(mockedsignature),
        },
      };

      await expect(
        VulticonnectWalletService.signCustomMessage(
          "hexMessage",
          "walletAddress"
        )
      ).rejects.toThrow("Failed to sign the message");

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Failed to sign the message",
        "missing param"
      );
    });
  });
});
