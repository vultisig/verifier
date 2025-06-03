declare module '7z-wasm' {
  interface SevenZipInstance {
    FS: {
      writeFile: (path: string, data: Uint8Array) => void;
      readFile: (path: string) => Uint8Array;
    };
    callMain: (args: string[]) => void;
  }

  function SevenZip(options?: { locateFile?: (file: string) => string }): Promise<SevenZipInstance>;
  export default SevenZip;
} 