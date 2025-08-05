import axios, { AxiosRequestConfig } from "axios";
import { jwtDecode } from "jwt-decode";
import { delToken, getToken, setToken } from "storage/token";
import { getVaultId } from "storage/vaultId";
import { toCamelCase, toSnakeCase } from "utils/functions";

type JwtPayload = {
  exp: number;
  iat: number;
  public_key: string;
  token_id: string;
};

const refreshAuthToken = async (oldToken: string) => {
  return axios
    .post<{ token: string }>(
      `${import.meta.env.VITE_MARKETPLACE_URL}/auth/refresh`,
      { token: oldToken },
      {
        headers: {
          Authorization: `Bearer ${oldToken}`,
          "Content-Type": "application/json",
        },
      }
    )
    .then(({ data }) => data.token);
};

const api = axios.create({
  headers: {
    "Content-Type": "application/json",
  },
});

api.interceptors.request.use(
  async (config) => {
    const publicKey = getVaultId();
    const token = getToken(publicKey);

    if (!token) {
      return { ...config, headers: config.headers.setAuthorization(null) };
    }

    try {
      const decoded = jwtDecode<JwtPayload>(token);
      const now = Math.floor(Date.now() / 1000);
      const issuedAt = decoded.iat ?? decoded.exp - 7 * 24 * 60 * 60;
      const totalLifetime = decoded.exp - issuedAt;
      const remainingLifetime = decoded.exp - now;

      if (totalLifetime <= 0 || remainingLifetime <= 0) {
        delToken(publicKey!);
        return config;
      }

      const percentRemaining = (remainingLifetime / totalLifetime) * 100;
      if (percentRemaining < 10) {
        const newToken = await refreshAuthToken(token);
        setToken(publicKey!, newToken);
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
      const publicKey = getVaultId();

      if (publicKey) delToken(publicKey);
    }

    return Promise.reject(
      new Error(error.response?.data?.error?.message || error.message)
    );
  }
);

export const del = async <T>(
  url: string,
  config?: AxiosRequestConfig
): Promise<T> =>
  api.delete<T>(url, config).then(({ data }) => toCamelCase(data));

export const get = async <T>(
  url: string,
  config?: AxiosRequestConfig
): Promise<T> =>
  await api.get<T>(url, config).then(({ data }) => toCamelCase(data));

export const post = async <T>(
  url: string,
  data?: any,
  config?: AxiosRequestConfig
): Promise<T> =>
  api
    .post<T>(url, toSnakeCase(data), config)
    .then(({ data }) => toCamelCase(data));

export const put = async <T>(
  url: string,
  data?: any,
  config?: AxiosRequestConfig
): Promise<T> =>
  api.put<T>(url, data, config).then(({ data }) => toCamelCase(data));
