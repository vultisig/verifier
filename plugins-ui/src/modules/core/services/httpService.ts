import axios, { AxiosRequestConfig } from "axios";
import { deleteToken, selectToken, updateToken } from "../../../storage/token";
import { getPublicKey } from "../../marketplace/services/marketplaceService";
import { jwtDecode } from "jwt-decode";

type JwtPayload = {
  exp: number;
  iat: number;
  public_key: string;
  token_id: string;
};

const rawApi = axios.create({
  baseURL: import.meta.env.VITE_MARKETPLACE_URL,
  headers: {
    "Content-Type": "application/json",
  },
});

/**
 * Refreshes the token without triggering interceptors.
 */
export const refreshAuthToken = async (oldToken: string): Promise<string> => {
  const res = await rawApi.post(
    "/auth/refresh",
    { token: oldToken },
    {
      headers: {
        Authorization: `Bearer ${oldToken}`,
      },
    }
  );
  return res.data.token;
};

const api = axios.create({
  headers: {
    "Content-Type": "application/json",
  },
});

/**
 * Axios request interceptor to:
 * - Decode JWT token
 * - Refresh it if less than 10% lifetime remains
 * - Attach Authorization header
 */
api.interceptors.request.use(
  async (config) => {
    const publicKey = getPublicKey();
    const token = publicKey ? selectToken(publicKey) : undefined;

    if (!token) return config;

    try {
      const decoded = jwtDecode<JwtPayload>(token);
      const now = Math.floor(Date.now() / 1000);
      const issuedAt = decoded.iat ?? decoded.exp - 7 * 24 * 60 * 60;
      const totalLifetime = decoded.exp - issuedAt;
      const remainingLifetime = decoded.exp - now;

      if (totalLifetime <= 0 || remainingLifetime <= 0) {
        deleteToken(publicKey!);
        return config;
      }

      const percentRemaining = (remainingLifetime / totalLifetime) * 100;
      if (percentRemaining < 10) {
        const newToken = await refreshAuthToken(token);
        updateToken(publicKey!, newToken);
        config.headers.Authorization = `Bearer ${newToken}`;
      } else {
        config.headers.Authorization = `Bearer ${token}`;
      }
    } catch (err) {
      console.warn(
        "Failed to decode or refresh token. Using current token.",
        err
      );
      config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
  },
  (error) => Promise.reject(error)
);

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      const publicKey = getPublicKey();
      if (publicKey) {
        deleteToken(publicKey);
      }
    }

    return Promise.reject(error);
  }
);

/**
 * Performs a GET request.
 * @param {string} url - The API endpoint.
 * @param {AxiosRequestConfig} config - Optional Axios config (e.g., headers, params).
 * @returns {Promise<T>} - The response data.
 */
export const get = async <T = any>(
  url: string,
  config?: AxiosRequestConfig
): Promise<T> => {
  const res = await api.get<T>(url, config);
  return res.data;
};

/**
 * Performs a POST request.
 * @param {string} url - The API endpoint.
 * @param {Object} data - The request body to send.
 * @param {AxiosRequestConfig} config - Optional Axios config (e.g., headers).
 * @returns {Promise<T>} - The response data.
 */
export const post = async <T = any>(
  url: string,
  data?: any,
  config?: AxiosRequestConfig
): Promise<T> => {
  const res = await api.post<T>(url, data, config);
  return res.data;
};

/**
 * Performs a PUT request.
 * @param {string} url - The API endpoint.
 * @param {Object} data - The request body to send.
 * @param {AxiosRequestConfig} config - Optional Axios config (e.g., headers).
 * @returns {Promise<T>} - The response data.
 */
export const put = async <T = any>(
  url: string,
  data?: any,
  config?: AxiosRequestConfig
): Promise<T> => {
  const res = await api.put<T>(url, data, config);
  return res.data;
};

/**
 * Performs a DELETE request.
 * @param {string} url - The API endpoint.
 * @param {AxiosRequestConfig} config - Use `config.data` to include request body (e.g., { data: { ... } }).
 * @returns {Promise<T>} - The response data.
 */
export const remove = async <T = any>(
  url: string,
  config?: AxiosRequestConfig
): Promise<T> => {
  const res = await api.delete<T>(url, config);
  return res.data;
};
