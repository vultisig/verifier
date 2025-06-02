import SevenZip from '7z-wasm';
import { fromBinary } from '@bufbuild/protobuf';
import pMemoize from 'p-memoize';
// Copied from extension codebase 
import { ReshareMessageSchema } from './reshareSchema/reshare_message_pb';
 
const normalizeBase64 = (str: string) => {
  let out = str.replace(/ /g, '+'); // URLSearchParams converts '+' to space
  out = out.replace(/-/g, '+').replace(/_/g, '/').replace(/\s+/g, '');
  while (out.length % 4) out += '='; // pad
  return out;
};

// 7zz.wasm copied from extension codebase to /public folder
export const getSevenZip = pMemoize(async () => {
  return SevenZip({ locateFile: () => "/7zz.wasm" }).catch(() => SevenZip());
});

export const decompressQrPayload = async (value: string): Promise<Uint8Array> => {
  const b64 = normalizeBase64(value);
  const bufferData = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
  // console.log("bufferData", bufferData);

  const sevenZip = await getSevenZip();
  // console.log("sevenZip", sevenZip);
  sevenZip.FS.writeFile('data.xz', bufferData);
  sevenZip.callMain(['x', 'data.xz', '-y']);
  return sevenZip.FS.readFile('data');
};

export const decodeTssPayload = (tssType: string, payload: Uint8Array) => {
  const schema = ReshareMessageSchema;
  if (!schema) throw new Error(`Unknown TSS type: ${tssType}`);
  return fromBinary(schema, payload);
}; 