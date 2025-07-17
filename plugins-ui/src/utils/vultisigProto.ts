import { fromBinary } from "@bufbuild/protobuf";
import SevenZip from "7z-wasm";
import pMemoize from "p-memoize";
import { ReshareMessageSchema } from "proto/reshare_message_pb";

const normalizeBase64 = (str: string) => {
  let out = str.replace(/ /g, "+"); // URLSearchParams converts '+' to space
  out = out.replace(/-/g, "+").replace(/_/g, "/").replace(/\s+/g, "");
  while (out.length % 4) out += "="; // pad
  return out;
};

export const getSevenZip = pMemoize(() =>
  SevenZip({
    locateFile: () => `/7zz.wasm`,
  }).catch(() => SevenZip())
);

export const decompressQrPayload = async (
  value: string
): Promise<Uint8Array> => {
  try {
    const b64 = normalizeBase64(value);
    const bufferData = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));

    const sevenZip = await getSevenZip();
    sevenZip.FS.writeFile("data.xz", bufferData);
    sevenZip.callMain(["x", "data.xz", "-y"]);
    return sevenZip.FS.readFile("data");
  } catch (error) {
    console.error("Failed to decompress QR payload", error);
    throw new Error("Failed to decompress QR payload");
  }
};

export const decodeTssPayload = (payload: Uint8Array) => {
  try {
    const schema = ReshareMessageSchema;
    return fromBinary(schema, payload);
  } catch (error) {
    console.error("Failed to decode TSS payload", error);
    throw new Error("Failed to decode TSS payload");
  }
};
