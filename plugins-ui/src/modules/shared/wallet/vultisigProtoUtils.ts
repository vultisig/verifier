import SevenZip from '7z-wasm';
import { fromBinary } from '@bufbuild/protobuf';
import pMemoize from 'p-memoize';
// Import your proto schemas here
// import { ReshareMessageSchema } from '@/proto/reshare_message_pb';
// import { KeysignMessageSchema } from '@/proto/keysign_message_pb';

// Example: Map tssType to schema
const ReshareMessageSchema = {} as any; // Placeholder, replace with actual schema
const tssSchemas: Record<string, any> = {
  Reshare: ReshareMessageSchema,
  // Add other types as needed
};
 
// 7zz.wasm copied to /public, so instruct 7z-wasm to fetch from root
export const getSevenZip = pMemoize(async () => {
  return SevenZip({ locateFile: () => "/7zz.wasm" }).catch(() => SevenZip());
});

export const decompressQrPayload = async (value: string): Promise<Uint8Array> => {
  // Normalize base64 (handle URL-safe chars, missing padding, whitespace)
  const normalizeBase64 = (str: string) => {
    let out = str.replace(/-/g, '+').replace(/_/g, '/').replace(/\s+/g, '');
    while (out.length % 4) out += '='; // pad
    return out;
  };

  const b64 = normalizeBase64(value);
  const bufferData = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
  console.log("bufferData", bufferData);

  const sevenZip = await getSevenZip();
  console.log("sevenZip", sevenZip);
  sevenZip.FS.writeFile('data.xz', bufferData);
  sevenZip.callMain(['x', 'data.xz', '-y']);
  return sevenZip.FS.readFile('data');
};

// export const getSevenZip = memoize(async () => {
//   return SevenZip(); // Let 7z-wasm resolve its own WASM asset path
// });

export const decodeTssPayload = (tssType: string, payload: Uint8Array) => {
  const schema = tssSchemas[tssType];
  if (!schema) throw new Error(`Unknown TSS type: ${tssType}`);
  // @ts-ignore
  return fromBinary(schema, payload);
}; 