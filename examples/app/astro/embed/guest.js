// Injected before esbuild IIFE: workerd-style builtins for Astro/CF bundles.
(function () {
  function getBuiltinModule(name) {
    if (name === "node:process" || name === "process") {
      return {
        env: (globalThis.process && globalThis.process.env) || {},
        version: "v20.19.0",
        versions: { node: "20.19.0" },
        platform: "linux",
        features: {},
        exit: function () {},
        cwd: function () {
          return "/";
        },
        nextTick: function (fn) {
          var args = Array.prototype.slice.call(arguments, 1);
          Promise.resolve().then(function () {
            fn.apply(null, args);
          });
        },
      };
    }
    if (name === "node:async_hooks" || name === "async_hooks") {
      function ALS() {
        this._s = undefined;
      }
      ALS.prototype.getStore = function () {
        return this._s;
      };
      ALS.prototype.run = function (s, fn) {
        var p = this._s;
        this._s = s;
        try {
          return fn();
        } finally {
          this._s = p;
        }
      };
      return { AsyncLocalStorage: ALS };
    }
    return {};
  }
  globalThis.getBuiltinModule = getBuiltinModule;
  if (typeof globalThis.process === "undefined" || !globalThis.process) {
    globalThis.process = {};
  }
  if (!globalThis.process.env) {
    globalThis.process.env = globalThis.__orvalhoProcessEnv || {};
  }
  globalThis.process.getBuiltinModule = getBuiltinModule;
})();

if (typeof globalThis.__orvalhoDynamicImport !== "function") {
  globalThis.__orvalhoDynamicImport = function (spec) {
    return Promise.reject(new Error("dynamic import not supported in goja: " + String(spec)));
  };
}

"use strict";
var __orvalhoWorker = (() => {
  var __defProp = Object.defineProperty;
  var __defProps = Object.defineProperties;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropDescs = Object.getOwnPropertyDescriptors;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __getOwnPropSymbols = Object.getOwnPropertySymbols;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __propIsEnum = Object.prototype.propertyIsEnumerable;
  var __knownSymbol = (name, symbol) => (symbol = Symbol[name]) ? symbol : /* @__PURE__ */ Symbol.for("Symbol." + name);
  var __typeError = (msg) => {
    throw TypeError(msg);
  };
  var __pow = Math.pow;
  var __defNormalProp = (obj, key, value) => key in obj ? __defProp(obj, key, { enumerable: true, configurable: true, writable: true, value }) : obj[key] = value;
  var __spreadValues = (a2, b) => {
    for (var prop in b || (b = {}))
      if (__hasOwnProp.call(b, prop))
        __defNormalProp(a2, prop, b[prop]);
    if (__getOwnPropSymbols)
      for (var prop of __getOwnPropSymbols(b)) {
        if (__propIsEnum.call(b, prop))
          __defNormalProp(a2, prop, b[prop]);
      }
    return a2;
  };
  var __spreadProps = (a2, b) => __defProps(a2, __getOwnPropDescs(b));
  var __objRest = (source, exclude) => {
    var target = {};
    for (var prop in source)
      if (__hasOwnProp.call(source, prop) && exclude.indexOf(prop) < 0)
        target[prop] = source[prop];
    if (source != null && __getOwnPropSymbols)
      for (var prop of __getOwnPropSymbols(source)) {
        if (exclude.indexOf(prop) < 0 && __propIsEnum.call(source, prop))
          target[prop] = source[prop];
      }
    return target;
  };
  var __esm = (fn, res, err) => function __init() {
    if (err) throw err[0];
    try {
      return fn && (res = (0, fn[__getOwnPropNames(fn)[0]])(fn = 0)), res;
    } catch (e2) {
      throw err = [e2], e2;
    }
  };
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);
  var __publicField = (obj, key, value) => __defNormalProp(obj, typeof key !== "symbol" ? key + "" : key, value);
  var __accessCheck = (obj, member, msg) => member.has(obj) || __typeError("Cannot " + msg);
  var __privateGet = (obj, member, getter) => (__accessCheck(obj, member, "read from private field"), getter ? getter.call(obj) : member.get(obj));
  var __privateAdd = (obj, member, value) => member.has(obj) ? __typeError("Cannot add the same private member more than once") : member instanceof WeakSet ? member.add(obj) : member.set(obj, value);
  var __privateSet = (obj, member, value, setter) => (__accessCheck(obj, member, "write to private field"), setter ? setter.call(obj, value) : member.set(obj, value), value);
  var __privateMethod = (obj, member, method) => (__accessCheck(obj, member, "access private method"), method);
  var __async = (__this, __arguments, generator) => {
    return new Promise((resolve, reject) => {
      var fulfilled = (value) => {
        try {
          step(generator.next(value));
        } catch (e2) {
          reject(e2);
        }
      };
      var rejected = (value) => {
        try {
          step(generator.throw(value));
        } catch (e2) {
          reject(e2);
        }
      };
      var step = (x) => x.done ? resolve(x.value) : Promise.resolve(x.value).then(fulfilled, rejected);
      step((generator = generator.apply(__this, __arguments)).next());
    });
  };
  var __await = function(promise, isYieldStar) {
    this[0] = promise;
    this[1] = isYieldStar;
  };
  var __asyncGenerator = (__this, __arguments, generator) => {
    var resume = (k, v, yes, no) => {
      try {
        var x = generator[k](v), isAwait = (v = x.value) instanceof __await, done = x.done;
        Promise.resolve(isAwait ? v[0] : v).then((y) => isAwait ? resume(k === "return" ? k : "next", v[1] ? { done: y.done, value: y.value } : y, yes, no) : yes({ value: y, done })).catch((e2) => resume("throw", e2, yes, no));
      } catch (e2) {
        no(e2);
      }
    }, method = (k, call, wait, clear) => it[k] = (x) => (call = new Promise((yes, no, run2) => (run2 = () => resume(k, x, yes, no), q ? q.then(run2) : run2())), clear = () => q === wait && (q = 0), q = wait = call.then(clear, clear), call), q, it = {};
    return generator = generator.apply(__this, __arguments), it[__knownSymbol("asyncIterator")] = () => it, method("next"), method("throw"), method("return"), it;
  };
  var __forAwait = (obj, it, method) => (it = obj[__knownSymbol("asyncIterator")]) ? it.call(obj) : (obj = obj[__knownSymbol("iterator")](), it = {}, method = (key, fn) => (fn = obj[key]) && (it[key] = (arg) => new Promise((yes, no, done) => (arg = fn.call(obj, arg), done = arg.done, Promise.resolve(arg.value).then((value) => yes({ value, done }), no)))), method("next"), method("return"), it);

  // ../../../../../../../../tmp/orvalho-stubs-3192708952/cloudflare_workers.js
  function liveEnv() {
    return globalThis.__orvalhoCFEnv || {};
  }
  var env;
  var init_cloudflare_workers = __esm({
    "../../../../../../../../tmp/orvalho-stubs-3192708952/cloudflare_workers.js"() {
      env = new Proxy(
        {},
        {
          get(_t, prop) {
            var e2 = liveEnv();
            var v = e2[prop];
            return typeof v === "undefined" ? e2[prop] : v;
          },
          has(_t, prop) {
            return prop in liveEnv();
          },
          ownKeys() {
            return Reflect.ownKeys(liveEnv());
          },
          getOwnPropertyDescriptor(_t, prop) {
            var e2 = liveEnv();
            if (!(prop in e2)) return void 0;
            return { configurable: true, enumerable: true, value: e2[prop], writable: false };
          }
        }
      );
    }
  });

  // chunks/astro-component_CZmLnn6R.mjs
  function validateArgs(args) {
    if (args.length !== 3) return false;
    if (!args[0] || typeof args[0] !== "object") return false;
    return true;
  }
  function baseCreateComponent(cb, moduleId, propagation) {
    var _a29, _b13;
    const name = (_b13 = (_a29 = moduleId == null ? void 0 : moduleId.split("/").pop()) == null ? void 0 : _a29.replace(".astro", "")) != null ? _b13 : "";
    const fn = (...args) => {
      if (!validateArgs(args)) {
        throw new AstroError(__spreadProps(__spreadValues({}, InvalidComponentArgs), {
          message: InvalidComponentArgs.message(name)
        }));
      }
      return cb(...args);
    };
    Object.defineProperty(fn, "name", { value: name, writable: false });
    fn.isAstroComponentFactory = true;
    fn.moduleId = moduleId;
    fn.propagation = propagation;
    return fn;
  }
  function createComponentWithOptions(opts) {
    const cb = baseCreateComponent(opts.factory, opts.moduleId, opts.propagation);
    return cb;
  }
  function createComponent(arg1, moduleId, propagation) {
    if (typeof arg1 === "function") {
      return baseCreateComponent(arg1, moduleId, propagation);
    } else {
      return createComponentWithOptions(arg1);
    }
  }
  var _a, _a2, _b;
  var init_astro_component_CZmLnn6R = __esm({
    "chunks/astro-component_CZmLnn6R.mjs"() {
      "use strict";
      init_worker_entry_D8ZVWWpA();
      (_a = globalThis.process) != null ? _a : globalThis.process = {};
      (_b = (_a2 = globalThis.process).env) != null ? _b : _a2.env = {};
    }
  });

  // chunks/image-service-workerd_C84WONpr.mjs
  var image_service_workerd_C84WONpr_exports = {};
  __export(image_service_workerd_C84WONpr_exports, {
    default: () => image_service_workerd_default
  });
  var _a3, _a4, _b2, service, image_service_workerd_default;
  var init_image_service_workerd_C84WONpr = __esm({
    "chunks/image-service-workerd_C84WONpr.mjs"() {
      "use strict";
      init_image_transform_endpoint_BgUicGG2();
      (_a3 = globalThis.process) != null ? _a3 : globalThis.process = {};
      (_b2 = (_a4 = globalThis.process).env) != null ? _b2 : _a4.env = {};
      service = __spreadProps(__spreadValues({}, baseService), {
        transform(inputBuffer, transform3) {
          return __async(this, null, function* () {
            return { data: inputBuffer, format: transform3.format };
          });
        }
      });
      image_service_workerd_default = service;
    }
  });

  // chunks/image-transform-endpoint_BgUicGG2.mjs
  var image_transform_endpoint_BgUicGG2_exports = {};
  __export(image_transform_endpoint_BgUicGG2_exports, {
    b: () => baseService,
    i: () => imageTransformEndpoint___js
  });
  function isESMImportedImage(src) {
    return typeof src === "object" || typeof src === "function" && "src" in src;
  }
  function isRemoteImage(src) {
    return typeof src === "string";
  }
  function resolveSrc(src) {
    return __async(this, null, function* () {
      var _a29;
      if (typeof src === "object" && "then" in src) {
        const resource = yield src;
        return (_a29 = resource.default) != null ? _a29 : resource;
      }
      return src;
    });
  }
  function inferSourceFormat(src) {
    if (src.startsWith(DATA_PREFIX)) {
      const sepIndex = src.indexOf(";");
      const commaIndex = src.indexOf(",");
      const mimeEnd = sepIndex === -1 ? commaIndex : commaIndex === -1 ? sepIndex : Math.min(sepIndex, commaIndex);
      if (mimeEnd === -1) return void 0;
      const mime = src.slice(DATA_PREFIX.length, mimeEnd);
      if (mime === "image/svg+xml") return "svg";
      const sub = mime.split("/")[1];
      return sub || void 0;
    }
    try {
      const cleanSrc = removeQueryString(src).split("#")[0];
      const lastSlash = cleanSrc.lastIndexOf("/");
      const basename = lastSlash === -1 ? cleanSrc : cleanSrc.slice(lastSlash + 1);
      const lastDot = basename.lastIndexOf(".");
      if (lastDot === -1) return void 0;
      return basename.slice(lastDot + 1).toLowerCase();
    } catch (e2) {
      return void 0;
    }
  }
  function resolveDefaultOutputFormat(sourceFormat) {
    return sourceFormat === "svg" ? "svg" : DEFAULT_OUTPUT_FORMAT;
  }
  function readUInt(input, bits, offset = 0, isBigEndian = false) {
    const endian = isBigEndian ? "BE" : "LE";
    const methodName = `readUInt${bits}${endian}`;
    return methods[methodName](input, offset);
  }
  function readBox(input, offset) {
    if (input.length - offset < 4) return;
    const boxSize = readUInt32BE(input, offset);
    if (input.length - offset < boxSize) return;
    return {
      name: toUTF8String(input, 4 + offset, 8 + offset),
      offset,
      size: boxSize
    };
  }
  function findBox(input, boxName, currentOffset) {
    while (currentOffset < input.length) {
      const box = readBox(input, currentOffset);
      if (!box) break;
      if (box.name === boxName) return box;
      currentOffset += box.size > 0 ? box.size : 8;
    }
  }
  function getSizeFromOffset(input, offset) {
    const value = input[offset];
    return value === 0 ? 256 : value;
  }
  function getImageSize$1(input, imageIndex) {
    const offset = SIZE_HEADER$1 + imageIndex * SIZE_IMAGE_ENTRY;
    return {
      height: getSizeFromOffset(input, offset + 1),
      width: getSizeFromOffset(input, offset)
    };
  }
  function detectType(input, start, end) {
    let hasAvif = false;
    let hasHeic = false;
    let hasHeif = false;
    for (let i2 = start; i2 <= end; i2 += 4) {
      const brand = toUTF8String(input, i2, i2 + 4);
      if (brand === "avif" || brand === "avis") hasAvif = true;
      else if (brand === "heic" || brand === "heix" || brand === "hevc" || brand === "hevx") hasHeic = true;
      else if (brand === "mif1" || brand === "msf1") hasHeif = true;
    }
    if (hasAvif) return "avif";
    if (hasHeic) return "heic";
    if (hasHeif) return "heif";
  }
  function readImageHeader(input, imageOffset) {
    const imageLengthOffset = imageOffset + ENTRY_LENGTH_OFFSET;
    return [
      toUTF8String(input, imageOffset, imageLengthOffset),
      readUInt32BE(input, imageLengthOffset)
    ];
  }
  function getImageSize(type) {
    const size = ICON_TYPE_SIZE[type];
    return { width: size, height: size, type };
  }
  function isEXIF(input) {
    return toHexString(input, 2, 6) === EXIF_MARKER;
  }
  function extractSize(input, index) {
    return {
      height: readUInt16BE(input, index),
      width: readUInt16BE(input, index + 2)
    };
  }
  function extractOrientation(exifBlock, isBigEndian) {
    const idfOffset = 8;
    const offset = EXIF_HEADER_BYTES + idfOffset;
    const idfDirectoryEntries = readUInt(exifBlock, 16, offset, isBigEndian);
    for (let directoryEntryNumber = 0; directoryEntryNumber < idfDirectoryEntries; directoryEntryNumber++) {
      const start = offset + NUM_DIRECTORY_ENTRIES_BYTES + directoryEntryNumber * IDF_ENTRY_BYTES;
      const end = start + IDF_ENTRY_BYTES;
      if (start > exifBlock.length) {
        return;
      }
      const block = exifBlock.slice(start, end);
      const tagNumber = readUInt(block, 16, 0, isBigEndian);
      if (tagNumber === 274) {
        const dataFormat = readUInt(block, 16, 2, isBigEndian);
        if (dataFormat !== 3) {
          return;
        }
        const numberOfComponents = readUInt(block, 32, 4, isBigEndian);
        if (numberOfComponents !== 1) {
          return;
        }
        return readUInt(block, 16, 8, isBigEndian);
      }
    }
  }
  function validateExifBlock(input, index) {
    const exifBlock = input.slice(APP1_DATA_SIZE_BYTES, index);
    const byteAlign = toHexString(
      exifBlock,
      EXIF_HEADER_BYTES,
      EXIF_HEADER_BYTES + TIFF_BYTE_ALIGN_BYTES
    );
    const isBigEndian = byteAlign === BIG_ENDIAN_BYTE_ALIGN;
    const isLittleEndian = byteAlign === LITTLE_ENDIAN_BYTE_ALIGN;
    if (isBigEndian || isLittleEndian) {
      return extractOrientation(exifBlock, isBigEndian);
    }
  }
  function validateInput(input, index) {
    if (index > input.length) {
      throw new TypeError("Corrupt JPG, exceeded buffer limits");
    }
  }
  function calculateImageDimension(reader, isSmallImage) {
    if (isSmallImage) {
      return 8 * (1 + reader.getBits(5));
    }
    const sizeClass = reader.getBits(2);
    const extraBits = [9, 13, 18, 30][sizeClass];
    return 1 + reader.getBits(extraBits);
  }
  function calculateImageWidth(reader, isSmallImage, widthMode, height) {
    if (isSmallImage && widthMode === 0) {
      return 8 * (1 + reader.getBits(5));
    }
    if (widthMode === 0) {
      return calculateImageDimension(reader, false);
    }
    const aspectRatios = [1, 1.2, 4 / 3, 1.5, 16 / 9, 5 / 4, 2];
    return Math.floor(height * aspectRatios[widthMode - 1]);
  }
  function extractCodestream(input) {
    const jxlcBox = findBox(input, "jxlc", 0);
    if (jxlcBox) {
      return input.slice(jxlcBox.offset + 8, jxlcBox.offset + jxlcBox.size);
    }
    const partialStreams = extractPartialStreams(input);
    if (partialStreams.length > 0) {
      return concatenateCodestreams(partialStreams);
    }
    return void 0;
  }
  function extractPartialStreams(input) {
    const partialStreams = [];
    let offset = 0;
    while (offset < input.length) {
      const jxlpBox = findBox(input, "jxlp", offset);
      if (!jxlpBox) break;
      partialStreams.push(
        input.slice(jxlpBox.offset + 12, jxlpBox.offset + jxlpBox.size)
      );
      offset = jxlpBox.offset + jxlpBox.size;
    }
    return partialStreams;
  }
  function concatenateCodestreams(partialCodestreams) {
    const totalLength = partialCodestreams.reduce(
      (acc, curr) => acc + curr.length,
      0
    );
    const codestream = new Uint8Array(totalLength);
    let position = 0;
    for (const partial of partialCodestreams) {
      codestream.set(partial, position);
      position += partial.length;
    }
    return codestream;
  }
  function parseLength(len) {
    const m = unitsReg.exec(len);
    if (!m) {
      return void 0;
    }
    return Math.round(Number(m[1]) * (units[m[2]] || 1));
  }
  function parseViewbox(viewbox) {
    const bounds = viewbox.split(" ");
    return {
      height: parseLength(bounds[3]),
      width: parseLength(bounds[2])
    };
  }
  function parseAttributes(root) {
    const width = extractorRegExps.width.exec(root);
    const height = extractorRegExps.height.exec(root);
    const viewbox = extractorRegExps.viewbox.exec(root);
    return {
      height: height && parseLength(height[2]),
      viewbox: viewbox && parseViewbox(viewbox[2]),
      width: width && parseLength(width[2])
    };
  }
  function calculateByDimensions(attrs) {
    return {
      height: attrs.height,
      width: attrs.width
    };
  }
  function calculateByViewbox(attrs, viewbox) {
    const ratio = viewbox.width / viewbox.height;
    if (attrs.width) {
      return {
        height: Math.floor(attrs.width / ratio),
        width: attrs.width
      };
    }
    if (attrs.height) {
      return {
        height: attrs.height,
        width: Math.floor(attrs.height * ratio)
      };
    }
    return {
      height: viewbox.height,
      width: viewbox.width
    };
  }
  function readIFD(input, { isBigEndian, isBigTiff }) {
    const ifdOffset = isBigTiff ? Number(readUInt64(input, 8, isBigEndian)) : readUInt(input, 32, 4, isBigEndian);
    const entryCountSize = isBigTiff ? CONSTANTS.COUNT_SIZE.BIG : CONSTANTS.COUNT_SIZE.STANDARD;
    return input.slice(ifdOffset + entryCountSize);
  }
  function readTagValue(input, type, offset, isBigEndian) {
    switch (type) {
      case CONSTANTS.TYPE.SHORT:
        return readUInt(input, 16, offset, isBigEndian);
      case CONSTANTS.TYPE.LONG:
        return readUInt(input, 32, offset, isBigEndian);
      case CONSTANTS.TYPE.LONG8: {
        const value = Number(readUInt64(input, offset, isBigEndian));
        if (value > Number.MAX_SAFE_INTEGER) {
          throw new TypeError("Value too large");
        }
        return value;
      }
      default:
        return 0;
    }
  }
  function nextTag(input, isBigTiff) {
    const entrySize = isBigTiff ? CONSTANTS.ENTRY_SIZE.BIG : CONSTANTS.ENTRY_SIZE.STANDARD;
    if (input.length > entrySize) {
      return input.slice(entrySize);
    }
  }
  function extractTags(input, { isBigEndian, isBigTiff }) {
    const tags = {};
    let temp = input;
    while (temp == null ? void 0 : temp.length) {
      const code = readUInt(temp, 16, 0, isBigEndian);
      const type = readUInt(temp, 16, 2, isBigEndian);
      const length = isBigTiff ? Number(readUInt64(temp, 4, isBigEndian)) : readUInt(temp, 32, 4, isBigEndian);
      if (code === 0) break;
      if (length === 1 && (type === CONSTANTS.TYPE.SHORT || type === CONSTANTS.TYPE.LONG || isBigTiff && type === CONSTANTS.TYPE.LONG8)) {
        const valueOffset = isBigTiff ? 12 : 8;
        tags[code] = readTagValue(temp, type, valueOffset, isBigEndian);
      }
      temp = nextTag(temp, isBigTiff);
    }
    return tags;
  }
  function determineFormat(input) {
    const signature = toUTF8String(input, 0, 2);
    const version2 = readUInt(input, 16, 2, signature === "MM");
    return {
      isBigEndian: signature === "MM",
      isBigTiff: version2 === 43
    };
  }
  function validateBigTIFFHeader(input, isBigEndian) {
    const byteSize = readUInt(input, 16, 4, isBigEndian);
    const reserved = readUInt(input, 16, 6, isBigEndian);
    if (byteSize !== 8 || reserved !== 0) {
      throw new TypeError("Invalid BigTIFF header");
    }
  }
  function calculateExtended(input) {
    return {
      height: 1 + readUInt24LE(input, 7),
      width: 1 + readUInt24LE(input, 4)
    };
  }
  function calculateLossless(input) {
    return {
      height: 1 + ((input[4] & 15) << 10 | input[3] << 2 | (input[2] & 192) >> 6),
      width: 1 + ((input[2] & 63) << 8 | input[1])
    };
  }
  function calculateLossy(input) {
    return {
      height: readInt16LE(input, 8) & 16383,
      width: readInt16LE(input, 6) & 16383
    };
  }
  function detector(input) {
    const byte = input[0];
    const type = firstBytes.get(byte);
    if (type && typeHandlers.get(type).validate(input)) {
      return type;
    }
    return types.find((imageType) => typeHandlers.get(imageType).validate(input));
  }
  function lookup$1(input) {
    var _a29;
    const type = detector(input);
    if (typeof type !== "undefined") {
      const size = typeHandlers.get(type).calculate(input);
      if (size !== void 0) {
        size.type = (_a29 = size.type) != null ? _a29 : type;
        return size;
      }
    }
    throw new TypeError("unsupported file type: " + type);
  }
  function imageMetadata(data, src) {
    return __async(this, null, function* () {
      let result;
      try {
        result = lookup$1(data);
      } catch (e2) {
        throw new AstroError(__spreadProps(__spreadValues({}, NoImageMetadata), {
          message: NoImageMetadata.message(src)
        }));
      }
      if (result.height == null || result.width == null || !result.type) {
        throw new AstroError(__spreadProps(__spreadValues({}, NoImageMetadata), {
          message: NoImageMetadata.message(src)
        }));
      }
      const { width, height, type, orientation } = result;
      const isPortrait = (orientation || 0) >= 5;
      return {
        width: isPortrait ? height : width,
        height: isPortrait ? width : height,
        format: type,
        orientation
      };
    });
  }
  function fetchWithRedirects(options) {
    return __async(this, null, function* () {
      var _a29, _b13;
      const {
        url,
        headers,
        imageConfig: imageConfig2,
        fetchFn = globalThis.fetch,
        redirectLimit = 10,
        onMaxRedirectsExceeded = (_u) => new Error("Maximum redirect depth exceeded"),
        onMissingLocationHeader = (_s, _u) => new Error(`Redirect response ${_s} missing Location header`),
        onDisallowedRedirect = (_current, _target) => new Error(
          `The image at ${_current} redirected to ${_target}, which is not an allowed remote location.`
        )
      } = options;
      if (redirectLimit <= 0) {
        throw onMaxRedirectsExceeded(typeof url === "string" ? url : url.toString());
      }
      const urlString = typeof url === "string" ? url : url.toString();
      const req = new Request(url, { headers });
      const res = yield fetchFn(req, { redirect: "manual" });
      if ([301, 302, 303, 307, 308].includes(res.status)) {
        const location = res.headers.get("Location");
        if (!location) {
          throw onMissingLocationHeader(res.status, urlString);
        }
        const redirectUrl = new URL(location, urlString).toString();
        if (!isRemoteAllowed(redirectUrl, {
          domains: (_a29 = imageConfig2.domains) != null ? _a29 : [],
          remotePatterns: (_b13 = imageConfig2.remotePatterns) != null ? _b13 : []
        })) {
          throw onDisallowedRedirect(urlString, redirectUrl);
        }
        return fetchWithRedirects({
          url: redirectUrl,
          headers,
          imageConfig: imageConfig2,
          fetchFn,
          redirectLimit: redirectLimit - 1,
          onMaxRedirectsExceeded,
          onMissingLocationHeader,
          onDisallowedRedirect
        });
      }
      return res;
    });
  }
  function inferRemoteSize(url, imageConfig2) {
    return __async(this, null, function* () {
      var _a29, _b13;
      if (!URL.canParse(url)) {
        throw new AstroError(__spreadProps(__spreadValues({}, FailedToFetchRemoteImageDimensions), {
          message: FailedToFetchRemoteImageDimensions.message(url)
        }));
      }
      const allowlistConfig = imageConfig2 ? {
        domains: (_a29 = imageConfig2.domains) != null ? _a29 : [],
        remotePatterns: (_b13 = imageConfig2.remotePatterns) != null ? _b13 : []
      } : void 0;
      if (!allowlistConfig) {
        const parsedUrl = new URL(url);
        if (!["http:", "https:"].includes(parsedUrl.protocol)) {
          throw new AstroError(__spreadProps(__spreadValues({}, FailedToFetchRemoteImageDimensions), {
            message: FailedToFetchRemoteImageDimensions.message(url)
          }));
        }
      }
      if (allowlistConfig && !isRemoteAllowed(url, allowlistConfig)) {
        throw new AstroError(__spreadProps(__spreadValues({}, RemoteImageNotAllowed), {
          message: RemoteImageNotAllowed.message(url)
        }));
      }
      let response;
      try {
        response = yield fetchWithRedirects({
          url,
          onMaxRedirectsExceeded: (u) => new AstroError(__spreadProps(__spreadValues({}, FailedToFetchRemoteImageDimensions), {
            message: FailedToFetchRemoteImageDimensions.message(u)
          })),
          onMissingLocationHeader: (_status, u) => new AstroError(__spreadProps(__spreadValues({}, FailedToFetchRemoteImageDimensions), {
            message: FailedToFetchRemoteImageDimensions.message(u)
          })),
          imageConfig: imageConfig2 != null ? imageConfig2 : {
            remotePatterns: [],
            domains: []
          }
        });
      } catch (_err) {
        throw new AstroError(__spreadProps(__spreadValues({}, FailedToFetchRemoteImageDimensions), {
          message: FailedToFetchRemoteImageDimensions.message(url)
        }));
      }
      if (allowlistConfig && !isRemoteAllowed(response.url, allowlistConfig)) {
        throw new AstroError(__spreadProps(__spreadValues({}, RemoteImageNotAllowed), {
          message: RemoteImageNotAllowed.message(url)
        }));
      }
      if (!response.body || !response.ok) {
        throw new AstroError(__spreadProps(__spreadValues({}, FailedToFetchRemoteImageDimensions), {
          message: FailedToFetchRemoteImageDimensions.message(url)
        }));
      }
      const reader = response.body.getReader();
      let done, value;
      let accumulatedChunks = new Uint8Array();
      while (!done) {
        const readResult = yield reader.read();
        done = readResult.done;
        if (done) break;
        if (readResult.value) {
          value = readResult.value;
          let tmp = new Uint8Array(accumulatedChunks.length + value.length);
          tmp.set(accumulatedChunks, 0);
          tmp.set(value, accumulatedChunks.length);
          accumulatedChunks = tmp;
          try {
            const dimensions = yield imageMetadata(accumulatedChunks, url);
            if (dimensions) {
              yield reader.cancel();
              return dimensions;
            }
          } catch (e2) {
          }
        }
      }
      throw new AstroError(__spreadProps(__spreadValues({}, NoImageMetadata), {
        message: NoImageMetadata.message(url)
      }));
    });
  }
  function isLocalService(service2) {
    if (!service2) {
      return false;
    }
    return "transform" in service2;
  }
  function verifyOptions(options) {
    if (!options.src || !isRemoteImage(options.src) && !isESMImportedImage(options.src)) {
      throw new AstroError(__spreadProps(__spreadValues({}, ExpectedImage), {
        message: ExpectedImage.message(
          JSON.stringify(options.src),
          typeof options.src,
          JSON.stringify(options, (_, v) => v === void 0 ? null : v)
        )
      }));
    }
    if (!isESMImportedImage(options.src)) {
      if (options.src.startsWith("/@fs/") || !isRemotePath(options.src) && !options.src.startsWith("/")) {
        throw new AstroError(__spreadProps(__spreadValues({}, LocalImageUsedWrongly), {
          message: LocalImageUsedWrongly.message(options.src)
        }));
      }
      let missingDimension;
      if (!options.width && !options.height) {
        missingDimension = "both";
      } else if (!options.width && options.height) {
        missingDimension = "width";
      } else if (options.width && !options.height) {
        missingDimension = "height";
      }
      if (missingDimension) {
        throw new AstroError(__spreadProps(__spreadValues({}, MissingImageDimension), {
          message: MissingImageDimension.message(missingDimension, options.src)
        }));
      }
    } else {
      if (!VALID_SUPPORTED_FORMATS.includes(options.src.format)) {
        throw new AstroError(__spreadProps(__spreadValues({}, UnsupportedImageFormat), {
          message: UnsupportedImageFormat.message(
            options.src.format,
            options.src.src,
            VALID_SUPPORTED_FORMATS
          )
        }));
      }
      if (options.widths && options.densities) {
        throw new AstroError(IncompatibleDescriptorOptions);
      }
      if (options.src.format !== "svg" && options.format === "svg") {
        throw new AstroError(UnsupportedImageConversion);
      }
    }
  }
  function getTargetDimensions(options) {
    let targetWidth = options.width;
    let targetHeight = options.height;
    if (isESMImportedImage(options.src)) {
      const aspectRatio = options.src.width / options.src.height;
      if (targetHeight && !targetWidth) {
        targetWidth = Math.round(targetHeight * aspectRatio);
      } else if (targetWidth && !targetHeight) {
        targetHeight = Math.round(targetWidth / aspectRatio);
      } else if (!targetWidth && !targetHeight) {
        targetWidth = options.src.width;
        targetHeight = options.src.height;
      }
    }
    return {
      targetWidth,
      targetHeight
    };
  }
  function isImageMetadata(src) {
    return src.fsPath && !("fsPath" in src);
  }
  function createPlaceholderURL(pathOrUrl) {
    return new URL(pathOrUrl, PLACEHOLDER_BASE);
  }
  function stringifyPlaceholderURL(url) {
    return url.href.replace(PLACEHOLDER_BASE, "");
  }
  function getConfiguredImageService() {
    return __async(this, null, function* () {
      var _a29;
      if (!((_a29 = globalThis == null ? void 0 : globalThis.astroAsset) == null ? void 0 : _a29.imageService)) {
        const { default: service2 } = yield Promise.resolve().then(() => (init_image_service_workerd_C84WONpr(), image_service_workerd_C84WONpr_exports)).catch((e2) => {
          const error2 = new AstroError(InvalidImageService);
          error2.cause = e2;
          throw error2;
        });
        if (!globalThis.astroAsset) globalThis.astroAsset = {};
        globalThis.astroAsset.imageService = service2;
        return service2;
      }
      return globalThis.astroAsset.imageService;
    });
  }
  function getImage$1(options, imageConfig2) {
    return __async(this, null, function* () {
      var _a29, _b13, _c, _d, _e, _f, _g, _h, _i, _j, _k, _l, _m, _n, _o;
      if (!options || typeof options !== "object") {
        throw new AstroError(__spreadProps(__spreadValues({}, ExpectedImageOptions), {
          message: ExpectedImageOptions.message(JSON.stringify(options))
        }));
      }
      if (typeof options.src === "undefined") {
        throw new AstroError(__spreadProps(__spreadValues({}, ExpectedImage), {
          message: ExpectedImage.message(
            options.src,
            "undefined",
            JSON.stringify(options)
          )
        }));
      }
      if (isImageMetadata(options)) {
        throw new AstroError(ExpectedNotESMImage);
      }
      const service2 = yield getConfiguredImageService();
      const resolvedOptions = __spreadProps(__spreadValues({}, options), {
        src: yield resolveSrc(options.src)
      });
      let originalWidth;
      let originalHeight;
      if (resolvedOptions.inferSize) {
        delete resolvedOptions.inferSize;
        if (isRemoteImage(resolvedOptions.src) && isRemotePath(resolvedOptions.src)) {
          if (!isRemoteAllowed(resolvedOptions.src, imageConfig2)) {
            throw new AstroError(__spreadProps(__spreadValues({}, RemoteImageNotAllowed), {
              message: RemoteImageNotAllowed.message(resolvedOptions.src)
            }));
          }
          const getRemoteSize = (url) => {
            var _a30, _b14;
            return (_b14 = (_a30 = service2.getRemoteSize) == null ? void 0 : _a30.call(service2, url, imageConfig2)) != null ? _b14 : inferRemoteSize(url, imageConfig2);
          };
          const result = yield getRemoteSize(resolvedOptions.src);
          (_a29 = resolvedOptions.width) != null ? _a29 : resolvedOptions.width = result.width;
          (_b13 = resolvedOptions.height) != null ? _b13 : resolvedOptions.height = result.height;
          if (result.format) {
            (_c = resolvedOptions.format) != null ? _c : resolvedOptions.format = resolveDefaultOutputFormat(result.format);
          }
          originalWidth = result.width;
          originalHeight = result.height;
        }
      }
      const originalFilePath = isESMImportedImage(resolvedOptions.src) ? resolvedOptions.src.fsPath : void 0;
      const clonedSrc = isESMImportedImage(resolvedOptions.src) ? (
        // @ts-expect-error - clone is a private, hidden prop
        (_d = resolvedOptions.src.clone) != null ? _d : resolvedOptions.src
      ) : resolvedOptions.src;
      if (isESMImportedImage(clonedSrc)) {
        originalWidth = clonedSrc.width;
        originalHeight = clonedSrc.height;
      }
      if (originalWidth && originalHeight) {
        const aspectRatio = originalWidth / originalHeight;
        if (resolvedOptions.height && !resolvedOptions.width) {
          resolvedOptions.width = Math.round(resolvedOptions.height * aspectRatio);
        } else if (resolvedOptions.width && !resolvedOptions.height) {
          resolvedOptions.height = Math.round(resolvedOptions.width / aspectRatio);
        } else if (!resolvedOptions.width && !resolvedOptions.height) {
          resolvedOptions.width = originalWidth;
          resolvedOptions.height = originalHeight;
        }
      }
      resolvedOptions.src = clonedSrc;
      const layout = (_f = (_e = options.layout) != null ? _e : imageConfig2.layout) != null ? _f : "none";
      if (resolvedOptions.priority) {
        (_g = resolvedOptions.loading) != null ? _g : resolvedOptions.loading = "eager";
        (_h = resolvedOptions.decoding) != null ? _h : resolvedOptions.decoding = "sync";
        (_i = resolvedOptions.fetchpriority) != null ? _i : resolvedOptions.fetchpriority = "high";
        delete resolvedOptions.priority;
      } else {
        (_j = resolvedOptions.loading) != null ? _j : resolvedOptions.loading = "lazy";
        (_k = resolvedOptions.decoding) != null ? _k : resolvedOptions.decoding = "async";
        (_l = resolvedOptions.fetchpriority) != null ? _l : resolvedOptions.fetchpriority = void 0;
      }
      if (layout !== "none") {
        resolvedOptions.widths || (resolvedOptions.widths = getWidths({
          width: resolvedOptions.width,
          layout,
          originalWidth,
          breakpoints: ((_m = imageConfig2.breakpoints) == null ? void 0 : _m.length) ? imageConfig2.breakpoints : isLocalService(service2) ? LIMITED_RESOLUTIONS : DEFAULT_RESOLUTIONS
        }));
        resolvedOptions.sizes || (resolvedOptions.sizes = getSizesAttribute({ width: resolvedOptions.width, layout }));
        delete resolvedOptions.densities;
        resolvedOptions["data-astro-image"] = layout;
        if (resolvedOptions.fit && cssFitValues.includes(resolvedOptions.fit)) {
          resolvedOptions["data-astro-image-fit"] = resolvedOptions.fit;
        }
        const currentPosition = resolvedOptions.position || "center";
        resolvedOptions["data-astro-image-pos"] = currentPosition.replace(/\s+/g, "-");
      }
      const validatedOptions = service2.validateOptions ? yield service2.validateOptions(resolvedOptions, imageConfig2) : resolvedOptions;
      (_n = validatedOptions.format) != null ? _n : validatedOptions.format = yield peekRemoteFormatForStaticEmit(
        validatedOptions,
        imageConfig2,
        service2
      );
      const srcSetTransforms = service2.getSrcSet ? yield service2.getSrcSet(validatedOptions, imageConfig2) : [];
      const lazyImageURLFactory = (getValue) => {
        let cached = null;
        return () => cached != null ? cached : cached = getValue();
      };
      const initialImageURL = yield service2.getURL(validatedOptions, imageConfig2);
      let lazyImageURL = lazyImageURLFactory(() => initialImageURL);
      const matchesValidatedTransform = (transform22) => transform22.width === validatedOptions.width && transform22.height === validatedOptions.height && transform22.format === validatedOptions.format;
      let srcSets = yield Promise.all(
        srcSetTransforms.map((srcSet) => __async(null, null, function* () {
          return {
            transform: srcSet.transform,
            url: matchesValidatedTransform(srcSet.transform) ? initialImageURL : yield service2.getURL(srcSet.transform, imageConfig2),
            descriptor: srcSet.descriptor,
            attributes: srcSet.attributes
          };
        }))
      );
      if (isLocalService(service2) && globalThis.astroAsset.addStaticImage && !(isRemoteImage(validatedOptions.src) && initialImageURL === validatedOptions.src)) {
        const propsToHash = (_o = service2.propertiesToHash) != null ? _o : DEFAULT_HASH_PROPS;
        lazyImageURL = lazyImageURLFactory(
          () => globalThis.astroAsset.addStaticImage(validatedOptions, propsToHash, originalFilePath)
        );
        srcSets = srcSetTransforms.map((srcSet) => {
          return {
            transform: srcSet.transform,
            url: matchesValidatedTransform(srcSet.transform) ? lazyImageURL() : globalThis.astroAsset.addStaticImage(srcSet.transform, propsToHash, originalFilePath),
            descriptor: srcSet.descriptor,
            attributes: srcSet.attributes
          };
        });
      } else if (imageConfig2.assetQueryParams) {
        const imageURLObj = createPlaceholderURL(initialImageURL);
        imageConfig2.assetQueryParams.forEach((value, key) => {
          imageURLObj.searchParams.set(key, value);
        });
        lazyImageURL = lazyImageURLFactory(() => stringifyPlaceholderURL(imageURLObj));
        srcSets = srcSets.map((srcSet) => {
          const urlObj = createPlaceholderURL(srcSet.url);
          imageConfig2.assetQueryParams.forEach((value, key) => {
            urlObj.searchParams.set(key, value);
          });
          return __spreadProps(__spreadValues({}, srcSet), {
            url: stringifyPlaceholderURL(urlObj)
          });
        });
      }
      return {
        rawOptions: resolvedOptions,
        options: validatedOptions,
        get src() {
          return lazyImageURL();
        },
        srcSet: {
          values: srcSets,
          attribute: srcSets.map((srcSet) => `${srcSet.url} ${srcSet.descriptor}`).join(", ")
        },
        attributes: service2.getHTMLAttributes !== void 0 ? yield service2.getHTMLAttributes(validatedOptions, imageConfig2) : {}
      };
    });
  }
  function peekRemoteFormatForStaticEmit(options, imageConfig2, service2) {
    return __async(this, null, function* () {
      var _a29;
      if (!isRemoteImage(options.src) || !isRemoteAllowed(options.src, imageConfig2) || !((_a29 = globalThis.astroAsset) == null ? void 0 : _a29.addStaticImage) || !isLocalService(service2) || !service2.getRemoteSize) {
        return void 0;
      }
      try {
        const probed = yield service2.getRemoteSize(options.src, imageConfig2);
        return resolveDefaultOutputFormat(probed.format);
      } catch (e2) {
        return void 0;
      }
    });
  }
  function lookup(extn) {
    let tmp = ("" + extn).trim().toLowerCase();
    let idx = tmp.lastIndexOf(".");
    return mimes[!~idx ? tmp : tmp.substring(++idx)];
  }
  function filterPreloads(data, preload) {
    if (!preload) {
      return null;
    }
    if (preload === true) {
      return data;
    }
    return data.filter(
      ({ weight, style, subset }) => preload.some((p) => {
        if (p.weight !== void 0 && weight !== void 0 && !checkWeight(p.weight.toString(), weight)) {
          return false;
        }
        if (p.style !== void 0 && p.style !== style) {
          return false;
        }
        if (p.subset !== void 0 && p.subset !== subset) {
          return false;
        }
        return true;
      })
    );
  }
  function checkWeight(input, target) {
    const trimmedInput = input.trim();
    if (trimmedInput.includes(" ")) {
      return trimmedInput === target;
    }
    if (target.includes(" ")) {
      const [a2, b] = target.split(" ");
      const parsedInput = Number.parseInt(input);
      return parsedInput >= Number.parseInt(a2) && parsedInput <= Number.parseInt(b);
    }
    return input === target;
  }
  function transform(rawUrl, images, assets) {
    return __async(this, null, function* () {
      var _a29, _b13;
      const url = new URL(rawUrl);
      const href = url.searchParams.get("href");
      if (!href || isRemotePath(href) && !isRemoteAllowed(href, imageConfig)) {
        return new Response("Forbidden", { status: 403 });
      }
      const imageSrc = new URL(href, url.origin);
      let content;
      if (isRemotePath(href)) {
        try {
          content = yield fetchWithRedirects({
            url: imageSrc,
            imageConfig
          });
          if (!isRemoteAllowed(content.url, imageConfig)) {
            return new Response("Forbidden", { status: 403 });
          }
        } catch (e2) {
          return new Response("Not Found", { status: 404 });
        }
      } else {
        content = yield assets.fetch(imageSrc);
      }
      if (!content.body) {
        return new Response(null, { status: 404 });
      }
      const input = images.input(content.body);
      const supportedFormats = {
        jpeg: "image/jpeg",
        jpg: "image/jpeg",
        png: "image/png",
        gif: "image/gif",
        webp: "image/webp",
        avif: "image/avif"
      };
      const outputFormat = supportedFormats[(_a29 = url.searchParams.get("f")) != null ? _a29 : ""];
      if (!outputFormat) {
        return new Response(`Unsupported format: ${url.searchParams.get("f")}`, { status: 400 });
      }
      return (yield input.transform({
        width: url.searchParams.has("w") ? Number.parseInt(url.searchParams.get("w")) : void 0,
        height: url.searchParams.has("h") ? Number.parseInt(url.searchParams.get("h")) : void 0,
        fit: url.searchParams.get("fit")
      }).output({
        quality: url.searchParams.get("q") ? (_b13 = qualityTable[url.searchParams.get("q")]) != null ? _b13 : Number.parseInt(url.searchParams.get("q")) : void 0,
        format: outputFormat
      })).response();
    });
  }
  var _a5, _a6, _b3, VALID_SUPPORTED_FORMATS, DEFAULT_OUTPUT_FORMAT, DEFAULT_HASH_PROPS, DEFAULT_RESOLUTIONS, LIMITED_RESOLUTIONS, getWidths, getSizesAttribute, DATA_PREFIX, decoder, toUTF8String, toHexString, getView, readInt16LE, readUInt16BE, readUInt16LE, readUInt24LE, readInt32LE, readUInt32BE, readUInt32LE, readUInt64, methods, BMP, TYPE_ICON, SIZE_HEADER$1, SIZE_IMAGE_ENTRY, ICO, TYPE_CURSOR, CUR, DDS, gifRegexp, GIF, brandMap, HEIF, SIZE_HEADER, FILE_LENGTH_OFFSET, ENTRY_LENGTH_OFFSET, ICON_TYPE_SIZE, ICNS, J2C, JP2, EXIF_MARKER, APP1_DATA_SIZE_BYTES, EXIF_HEADER_BYTES, TIFF_BYTE_ALIGN_BYTES, BIG_ENDIAN_BYTE_ALIGN, LITTLE_ENDIAN_BYTE_ALIGN, IDF_ENTRY_BYTES, NUM_DIRECTORY_ENTRIES_BYTES, JPG, BitReader, JXLStream, JXL, KTX, pngSignature, pngImageHeaderChunkName, pngFriedChunkName, PNG, PNMTypes, handlers, PNM, PSD, svgReg, extractorRegExps, INCH_CM, units, unitsReg, SVG, TGA, CONSTANTS, signatures, TIFF, WEBP, typeHandlers, types, firstBytes, sortNumeric, baseService, PLACEHOLDER_BASE, cssFitValues, $$Image, mimes, $$Picture, componentDataByCssVariable, $$Font, _urls, SsrRuntimeFontFileUrlResolver, assetQueryParams, imageConfig, getImage, qualityTable, prerender, GET, _page, page, imageTransformEndpoint___js;
  var init_image_transform_endpoint_BgUicGG2 = __esm({
    "chunks/image-transform-endpoint_BgUicGG2.mjs"() {
      "use strict";
      init_cloudflare_workers();
      init_worker_entry_D8ZVWWpA();
      init_astro_component_CZmLnn6R();
      (_a5 = globalThis.process) != null ? _a5 : globalThis.process = {};
      (_b3 = (_a6 = globalThis.process).env) != null ? _b3 : _a6.env = {};
      VALID_SUPPORTED_FORMATS = [
        "jpeg",
        "jpg",
        "png",
        "tiff",
        "webp",
        "gif",
        "svg",
        "avif"
      ];
      DEFAULT_OUTPUT_FORMAT = "webp";
      DEFAULT_HASH_PROPS = [
        "src",
        "width",
        "height",
        "format",
        "quality",
        "fit",
        "position",
        "background"
      ];
      DEFAULT_RESOLUTIONS = [
        640,
        // older and lower-end phones
        750,
        // iPhone 6-8
        828,
        // iPhone XR/11
        960,
        // older horizontal phones
        1080,
        // iPhone 6-8 Plus
        1280,
        // 720p
        1668,
        // Various iPads
        1920,
        // 1080p
        2048,
        // QXGA
        2560,
        // WQXGA
        3200,
        // QHD+
        3840,
        // 4K
        4480,
        // 4.5K
        5120,
        // 5K
        6016
        // 6K
      ];
      LIMITED_RESOLUTIONS = [
        640,
        // older and lower-end phones
        750,
        // iPhone 6-8
        828,
        // iPhone XR/11
        1080,
        // iPhone 6-8 Plus
        1280,
        // 720p
        1668,
        // Various iPads
        2048,
        // QXGA
        2560
        // WQXGA
      ];
      getWidths = ({
        width,
        layout,
        breakpoints = DEFAULT_RESOLUTIONS,
        originalWidth
      }) => {
        const smallerThanOriginal = (w) => !originalWidth || w <= originalWidth;
        if (layout === "full-width") {
          return breakpoints.filter(smallerThanOriginal);
        }
        if (!width) {
          return [];
        }
        const doubleWidth = width * 2;
        const maxSize = originalWidth ? Math.min(doubleWidth, originalWidth) : doubleWidth;
        if (layout === "fixed") {
          return originalWidth && width > originalWidth ? [originalWidth] : [width, maxSize];
        }
        if (layout === "constrained") {
          return [
            // Always include the image at 1x and 2x the specified width
            width,
            doubleWidth,
            ...breakpoints
          ].filter((w) => w <= maxSize).sort((a2, b) => a2 - b);
        }
        return [];
      };
      getSizesAttribute = ({
        width,
        layout
      }) => {
        if (!width || !layout) {
          return void 0;
        }
        switch (layout) {
          // If screen is wider than the max size then image width is the max size,
          // otherwise it's the width of the screen
          case "constrained":
            return `(min-width: ${width}px) ${width}px, 100vw`;
          // Image is always the same width, whatever the size of the screen
          case "fixed":
            return `${width}px`;
          // Image is always the width of the screen
          case "full-width":
            return `100vw`;
          case "none":
          default:
            return void 0;
        }
      };
      DATA_PREFIX = "data:";
      decoder = new TextDecoder();
      toUTF8String = (input, start = 0, end = input.length) => decoder.decode(input.slice(start, end));
      toHexString = (input, start = 0, end = input.length) => input.slice(start, end).reduce((memo, i2) => memo + `0${i2.toString(16)}`.slice(-2), "");
      getView = (input, offset) => new DataView(input.buffer, input.byteOffset + offset);
      readInt16LE = (input, offset = 0) => getView(input, offset).getInt16(0, true);
      readUInt16BE = (input, offset = 0) => getView(input, offset).getUint16(0, false);
      readUInt16LE = (input, offset = 0) => getView(input, offset).getUint16(0, true);
      readUInt24LE = (input, offset = 0) => {
        const view = getView(input, offset);
        return view.getUint16(0, true) + (view.getUint8(2) << 16);
      };
      readInt32LE = (input, offset = 0) => getView(input, offset).getInt32(0, true);
      readUInt32BE = (input, offset = 0) => getView(input, offset).getUint32(0, false);
      readUInt32LE = (input, offset = 0) => getView(input, offset).getUint32(0, true);
      readUInt64 = (input, offset, isBigEndian) => getView(input, offset).getBigUint64(0, !isBigEndian);
      methods = {
        readUInt16BE,
        readUInt16LE,
        readUInt32BE,
        readUInt32LE
      };
      BMP = {
        validate: (input) => toUTF8String(input, 0, 2) === "BM",
        calculate: (input) => ({
          height: Math.abs(readInt32LE(input, 22)),
          width: readUInt32LE(input, 18)
        })
      };
      TYPE_ICON = 1;
      SIZE_HEADER$1 = 2 + 2 + 2;
      SIZE_IMAGE_ENTRY = 1 + 1 + 1 + 1 + 2 + 2 + 4 + 4;
      ICO = {
        validate(input) {
          const reserved = readUInt16LE(input, 0);
          const imageCount = readUInt16LE(input, 4);
          if (reserved !== 0 || imageCount === 0) return false;
          const imageType = readUInt16LE(input, 2);
          return imageType === TYPE_ICON;
        },
        calculate(input) {
          const nbImages = readUInt16LE(input, 4);
          const imageSize = getImageSize$1(input, 0);
          if (nbImages === 1) return imageSize;
          const images = [];
          for (let imageIndex = 0; imageIndex < nbImages; imageIndex += 1) {
            images.push(getImageSize$1(input, imageIndex));
          }
          return {
            width: imageSize.width,
            height: imageSize.height,
            images
          };
        }
      };
      TYPE_CURSOR = 2;
      CUR = {
        validate(input) {
          const reserved = readUInt16LE(input, 0);
          const imageCount = readUInt16LE(input, 4);
          if (reserved !== 0 || imageCount === 0) return false;
          const imageType = readUInt16LE(input, 2);
          return imageType === TYPE_CURSOR;
        },
        calculate: (input) => ICO.calculate(input)
      };
      DDS = {
        validate: (input) => readUInt32LE(input, 0) === 542327876,
        calculate: (input) => ({
          height: readUInt32LE(input, 12),
          width: readUInt32LE(input, 16)
        })
      };
      gifRegexp = /^GIF8[79]a/;
      GIF = {
        validate: (input) => gifRegexp.test(toUTF8String(input, 0, 6)),
        calculate: (input) => ({
          height: readUInt16LE(input, 8),
          width: readUInt16LE(input, 6)
        })
      };
      brandMap = {
        avif: "avif",
        avis: "avif",
        // avif-sequence
        mif1: "heif",
        msf1: "heif",
        // heif-sequence
        heic: "heic",
        heix: "heic",
        hevc: "heic",
        // heic-sequence
        hevx: "heic"
        // heic-sequence
      };
      HEIF = {
        validate(input) {
          const boxType = toUTF8String(input, 4, 8);
          if (boxType !== "ftyp") return false;
          const ftypBox = findBox(input, "ftyp", 0);
          if (!ftypBox) return false;
          const brand = toUTF8String(input, ftypBox.offset + 8, ftypBox.offset + 12);
          return brand in brandMap;
        },
        calculate(input) {
          const metaBox = findBox(input, "meta", 0);
          const iprpBox = metaBox && findBox(input, "iprp", metaBox.offset + 12);
          const ipcoBox = iprpBox && findBox(input, "ipco", iprpBox.offset + 8);
          if (!ipcoBox) {
            throw new TypeError("Invalid HEIF, no ipco box found");
          }
          const type = detectType(input, 8, metaBox.offset);
          const images = [];
          let currentOffset = ipcoBox.offset + 8;
          while (currentOffset < ipcoBox.offset + ipcoBox.size) {
            const ispeBox = findBox(input, "ispe", currentOffset);
            if (!ispeBox) break;
            const rawWidth = readUInt32BE(input, ispeBox.offset + 12);
            const rawHeight = readUInt32BE(input, ispeBox.offset + 16);
            const clapBox = findBox(input, "clap", currentOffset);
            let width = rawWidth;
            let height = rawHeight;
            if (clapBox && clapBox.offset < ipcoBox.offset + ipcoBox.size) {
              const cropRight = readUInt32BE(input, clapBox.offset + 12);
              width = rawWidth - cropRight;
            }
            images.push({ height, width });
            currentOffset = ispeBox.offset + ispeBox.size;
          }
          if (images.length === 0) {
            throw new TypeError("Invalid HEIF, no sizes found");
          }
          return __spreadValues({
            width: images[0].width,
            height: images[0].height,
            type
          }, images.length > 1 ? { images } : {});
        }
      };
      SIZE_HEADER = 4 + 4;
      FILE_LENGTH_OFFSET = 4;
      ENTRY_LENGTH_OFFSET = 4;
      ICON_TYPE_SIZE = {
        ICON: 32,
        "ICN#": 32,
        // m => 16 x 16
        "icm#": 16,
        icm4: 16,
        icm8: 16,
        // s => 16 x 16
        "ics#": 16,
        ics4: 16,
        ics8: 16,
        is32: 16,
        s8mk: 16,
        icp4: 16,
        // l => 32 x 32
        icl4: 32,
        icl8: 32,
        il32: 32,
        l8mk: 32,
        icp5: 32,
        ic11: 32,
        // h => 48 x 48
        ich4: 48,
        ich8: 48,
        ih32: 48,
        h8mk: 48,
        // . => 64 x 64
        icp6: 64,
        ic12: 32,
        // t => 128 x 128
        it32: 128,
        t8mk: 128,
        ic07: 128,
        // . => 256 x 256
        ic08: 256,
        ic13: 256,
        // . => 512 x 512
        ic09: 512,
        ic14: 512,
        // . => 1024 x 1024
        ic10: 1024
      };
      ICNS = {
        validate: (input) => toUTF8String(input, 0, 4) === "icns",
        calculate(input) {
          const inputLength = input.length;
          const fileLength = readUInt32BE(input, FILE_LENGTH_OFFSET);
          let imageOffset = SIZE_HEADER;
          const images = [];
          while (imageOffset < fileLength && imageOffset < inputLength) {
            const imageHeader = readImageHeader(input, imageOffset);
            const imageSize = getImageSize(imageHeader[0]);
            images.push(imageSize);
            imageOffset += imageHeader[1];
          }
          if (images.length === 0) {
            throw new TypeError("Invalid ICNS, no sizes found");
          }
          return __spreadValues({
            width: images[0].width,
            height: images[0].height
          }, images.length > 1 ? { images } : {});
        }
      };
      J2C = {
        // TODO: this doesn't seem right. SIZ marker doesn't have to be right after the SOC
        validate: (input) => readUInt32BE(input, 0) === 4283432785,
        calculate: (input) => ({
          height: readUInt32BE(input, 12),
          width: readUInt32BE(input, 8)
        })
      };
      JP2 = {
        validate(input) {
          const boxType = toUTF8String(input, 4, 8);
          if (boxType !== "jP  ") return false;
          const ftypBox = findBox(input, "ftyp", 0);
          if (!ftypBox) return false;
          const brand = toUTF8String(input, ftypBox.offset + 8, ftypBox.offset + 12);
          return brand === "jp2 ";
        },
        calculate(input) {
          const jp2hBox = findBox(input, "jp2h", 0);
          const ihdrBox = jp2hBox && findBox(input, "ihdr", jp2hBox.offset + 8);
          if (ihdrBox) {
            return {
              height: readUInt32BE(input, ihdrBox.offset + 8),
              width: readUInt32BE(input, ihdrBox.offset + 12)
            };
          }
          throw new TypeError("Unsupported JPEG 2000 format");
        }
      };
      EXIF_MARKER = "45786966";
      APP1_DATA_SIZE_BYTES = 2;
      EXIF_HEADER_BYTES = 6;
      TIFF_BYTE_ALIGN_BYTES = 2;
      BIG_ENDIAN_BYTE_ALIGN = "4d4d";
      LITTLE_ENDIAN_BYTE_ALIGN = "4949";
      IDF_ENTRY_BYTES = 12;
      NUM_DIRECTORY_ENTRIES_BYTES = 2;
      JPG = {
        validate: (input) => toHexString(input, 0, 2) === "ffd8",
        calculate(_input) {
          let input = _input.slice(4);
          let orientation;
          let next;
          while (input.length) {
            const i2 = readUInt16BE(input, 0);
            validateInput(input, i2);
            if (input[i2] !== 255) {
              input = input.slice(1);
              continue;
            }
            if (isEXIF(input)) {
              orientation = validateExifBlock(input, i2);
            }
            next = input[i2 + 1];
            if (next === 192 || next === 193 || next === 194) {
              const size = extractSize(input, i2 + 5);
              if (!orientation) {
                return size;
              }
              return {
                height: size.height,
                orientation,
                width: size.width
              };
            }
            input = input.slice(i2 + 2);
          }
          throw new TypeError("Invalid JPG, no size found");
        }
      };
      BitReader = class {
        constructor(input, endianness) {
          // Skip the first 16 bits (2 bytes) of signature
          __publicField(this, "byteOffset", 2);
          __publicField(this, "bitOffset", 0);
          __publicField(this, "input");
          __publicField(this, "endianness");
          this.input = input;
          this.endianness = endianness;
        }
        /** Reads a specified number of bits, and move the offset */
        getBits(length = 1) {
          let result = 0;
          let bitsRead = 0;
          while (bitsRead < length) {
            if (this.byteOffset >= this.input.length) {
              throw new Error("Reached end of input");
            }
            const currentByte = this.input[this.byteOffset];
            const bitsLeft = 8 - this.bitOffset;
            const bitsToRead = Math.min(length - bitsRead, bitsLeft);
            if (this.endianness === "little-endian") {
              const mask = (1 << bitsToRead) - 1;
              const bits = currentByte >> this.bitOffset & mask;
              result |= bits << bitsRead;
            } else {
              const mask = (1 << bitsToRead) - 1 << 8 - this.bitOffset - bitsToRead;
              const bits = (currentByte & mask) >> 8 - this.bitOffset - bitsToRead;
              result = result << bitsToRead | bits;
            }
            bitsRead += bitsToRead;
            this.bitOffset += bitsToRead;
            if (this.bitOffset === 8) {
              this.byteOffset++;
              this.bitOffset = 0;
            }
          }
          return result;
        }
      };
      JXLStream = {
        validate: (input) => {
          return toHexString(input, 0, 2) === "ff0a";
        },
        calculate(input) {
          const reader = new BitReader(input, "little-endian");
          const isSmallImage = reader.getBits(1) === 1;
          const height = calculateImageDimension(reader, isSmallImage);
          const widthMode = reader.getBits(3);
          const width = calculateImageWidth(reader, isSmallImage, widthMode, height);
          return { width, height };
        }
      };
      JXL = {
        validate: (input) => {
          const boxType = toUTF8String(input, 4, 8);
          if (boxType !== "JXL ") return false;
          const ftypBox = findBox(input, "ftyp", 0);
          if (!ftypBox) return false;
          const brand = toUTF8String(input, ftypBox.offset + 8, ftypBox.offset + 12);
          return brand === "jxl ";
        },
        calculate(input) {
          const codestream = extractCodestream(input);
          if (codestream) return JXLStream.calculate(codestream);
          throw new Error("No codestream found in JXL container");
        }
      };
      KTX = {
        validate: (input) => {
          const signature = toUTF8String(input, 1, 7);
          return ["KTX 11", "KTX 20"].includes(signature);
        },
        calculate: (input) => {
          const type = input[5] === 49 ? "ktx" : "ktx2";
          const offset = type === "ktx" ? 36 : 20;
          return {
            height: readUInt32LE(input, offset + 4),
            width: readUInt32LE(input, offset),
            type
          };
        }
      };
      pngSignature = "PNG\r\n\n";
      pngImageHeaderChunkName = "IHDR";
      pngFriedChunkName = "CgBI";
      PNG = {
        validate(input) {
          if (pngSignature === toUTF8String(input, 1, 8)) {
            let chunkName = toUTF8String(input, 12, 16);
            if (chunkName === pngFriedChunkName) {
              chunkName = toUTF8String(input, 28, 32);
            }
            if (chunkName !== pngImageHeaderChunkName) {
              throw new TypeError("Invalid PNG");
            }
            return true;
          }
          return false;
        },
        calculate(input) {
          if (toUTF8String(input, 12, 16) === pngFriedChunkName) {
            return {
              height: readUInt32BE(input, 36),
              width: readUInt32BE(input, 32)
            };
          }
          return {
            height: readUInt32BE(input, 20),
            width: readUInt32BE(input, 16)
          };
        }
      };
      PNMTypes = {
        P1: "pbm/ascii",
        P2: "pgm/ascii",
        P3: "ppm/ascii",
        P4: "pbm",
        P5: "pgm",
        P6: "ppm",
        P7: "pam",
        PF: "pfm"
      };
      handlers = {
        default: (lines) => {
          let dimensions = [];
          while (lines.length > 0) {
            const line = lines.shift();
            if (line[0] === "#") {
              continue;
            }
            dimensions = line.split(" ");
            break;
          }
          if (dimensions.length === 2) {
            return {
              height: Number.parseInt(dimensions[1], 10),
              width: Number.parseInt(dimensions[0], 10)
            };
          }
          throw new TypeError("Invalid PNM");
        },
        pam: (lines) => {
          const size = {};
          while (lines.length > 0) {
            const line = lines.shift();
            if (line.length > 16 || line.charCodeAt(0) > 128) {
              continue;
            }
            const [key, value] = line.split(" ");
            if (key && value) {
              size[key.toLowerCase()] = Number.parseInt(value, 10);
            }
            if (size.height && size.width) {
              break;
            }
          }
          if (size.height && size.width) {
            return {
              height: size.height,
              width: size.width
            };
          }
          throw new TypeError("Invalid PAM");
        }
      };
      PNM = {
        validate: (input) => toUTF8String(input, 0, 2) in PNMTypes,
        calculate(input) {
          const signature = toUTF8String(input, 0, 2);
          const type = PNMTypes[signature];
          const lines = toUTF8String(input, 3).split(/[\r\n]+/);
          const handler = handlers[type] || handlers.default;
          return handler(lines);
        }
      };
      PSD = {
        validate: (input) => toUTF8String(input, 0, 4) === "8BPS",
        calculate: (input) => ({
          height: readUInt32BE(input, 14),
          width: readUInt32BE(input, 18)
        })
      };
      svgReg = /<svg\s([^>"']|"[^"]*"|'[^']*')*>/;
      extractorRegExps = {
        height: /\sheight=(['"])([^%]+?)\1/,
        root: svgReg,
        viewbox: /\sviewBox=(['"])(.+?)\1/i,
        width: /\swidth=(['"])([^%]+?)\1/
      };
      INCH_CM = 2.54;
      units = {
        in: 96,
        cm: 96 / INCH_CM,
        em: 16,
        ex: 8,
        m: 96 / INCH_CM * 100,
        mm: 96 / INCH_CM / 10,
        pc: 96 / 72 / 12,
        pt: 96 / 72,
        px: 1
      };
      unitsReg = new RegExp(
        `^([0-9.]+(?:e\\d+)?)(${Object.keys(units).join("|")})?$`
      );
      SVG = {
        // Scan only the first kilo-byte to speed up the check on larger files
        validate: (input) => svgReg.test(toUTF8String(input, 0, 1e3)),
        calculate(input) {
          const root = extractorRegExps.root.exec(toUTF8String(input));
          if (root) {
            const attrs = parseAttributes(root[0]);
            if (attrs.width != null && attrs.height != null) {
              return calculateByDimensions(attrs);
            }
            if (attrs.viewbox) {
              return calculateByViewbox(attrs, attrs.viewbox);
            }
          }
          throw new TypeError("Invalid SVG");
        }
      };
      TGA = {
        validate(input) {
          return readUInt16LE(input, 0) === 0 && readUInt16LE(input, 4) === 0;
        },
        calculate(input) {
          return {
            height: readUInt16LE(input, 14),
            width: readUInt16LE(input, 12)
          };
        }
      };
      CONSTANTS = {
        TAG: {
          WIDTH: 256,
          HEIGHT: 257,
          COMPRESSION: 259
        },
        TYPE: {
          SHORT: 3,
          LONG: 4,
          LONG8: 16
        },
        ENTRY_SIZE: {
          STANDARD: 12,
          BIG: 20
        },
        COUNT_SIZE: {
          STANDARD: 2,
          BIG: 8
        }
      };
      signatures = /* @__PURE__ */ new Set([
        "49492a00",
        // Little Endian
        "4d4d002a",
        // Big Endian
        "49492b00",
        // BigTIFF Little Endian
        "4d4d002b"
        // BigTIFF Big Endian
      ]);
      TIFF = {
        validate: (input) => {
          const signature = toHexString(input, 0, 4);
          return signatures.has(signature);
        },
        calculate(input) {
          const format = determineFormat(input);
          if (format.isBigTiff) {
            validateBigTIFFHeader(input, format.isBigEndian);
          }
          const ifdBuffer = readIFD(input, format);
          const tags = extractTags(ifdBuffer, format);
          const info2 = {
            height: tags[CONSTANTS.TAG.HEIGHT],
            width: tags[CONSTANTS.TAG.WIDTH],
            type: format.isBigTiff ? "bigtiff" : "tiff"
          };
          if (tags[CONSTANTS.TAG.COMPRESSION]) {
            info2.compression = tags[CONSTANTS.TAG.COMPRESSION];
          }
          if (!info2.width || !info2.height) {
            throw new TypeError("Invalid Tiff. Missing tags");
          }
          return info2;
        }
      };
      WEBP = {
        validate(input) {
          const riffHeader = "RIFF" === toUTF8String(input, 0, 4);
          const webpHeader = "WEBP" === toUTF8String(input, 8, 12);
          const vp8Header = "VP8" === toUTF8String(input, 12, 15);
          return riffHeader && webpHeader && vp8Header;
        },
        calculate(_input) {
          const chunkHeader = toUTF8String(_input, 12, 16);
          const input = _input.slice(20, 30);
          if (chunkHeader === "VP8X") {
            const extendedHeader = input[0];
            const validStart = (extendedHeader & 192) === 0;
            const validEnd = (extendedHeader & 1) === 0;
            if (validStart && validEnd) {
              return calculateExtended(input);
            }
            throw new TypeError("Invalid WebP");
          }
          if (chunkHeader === "VP8 " && input[0] !== 47) {
            return calculateLossy(input);
          }
          const signature = toHexString(input, 3, 6);
          if (chunkHeader === "VP8L" && signature !== "9d012a") {
            return calculateLossless(input);
          }
          throw new TypeError("Invalid WebP");
        }
      };
      typeHandlers = /* @__PURE__ */ new Map([
        ["bmp", BMP],
        ["cur", CUR],
        ["dds", DDS],
        ["gif", GIF],
        ["heif", HEIF],
        ["icns", ICNS],
        ["ico", ICO],
        ["j2c", J2C],
        ["jp2", JP2],
        ["jpg", JPG],
        ["jxl", JXL],
        ["jxl-stream", JXLStream],
        ["ktx", KTX],
        ["png", PNG],
        ["pnm", PNM],
        ["psd", PSD],
        ["svg", SVG],
        ["tga", TGA],
        ["tiff", TIFF],
        ["webp", WEBP]
      ]);
      types = Array.from(typeHandlers.keys());
      firstBytes = /* @__PURE__ */ new Map([
        [0, "heif"],
        [56, "psd"],
        [66, "bmp"],
        [68, "dds"],
        [71, "gif"],
        [73, "tiff"],
        [77, "tiff"],
        [82, "webp"],
        [105, "icns"],
        [137, "png"],
        [255, "jpg"]
      ]);
      sortNumeric = (a2, b) => a2 - b;
      baseService = {
        propertiesToHash: DEFAULT_HASH_PROPS,
        validateOptions(options) {
          verifyOptions(options);
          if (!options.format) {
            if (isESMImportedImage(options.src)) {
              options.format = resolveDefaultOutputFormat(options.src.format);
            } else {
              const inferred = inferSourceFormat(options.src);
              if (inferred) options.format = resolveDefaultOutputFormat(inferred);
            }
          }
          if (options.width) options.width = Math.round(options.width);
          if (options.height) options.height = Math.round(options.height);
          if (options.layout) {
            delete options.layout;
          }
          if (options.fit === "none") {
            delete options.fit;
          }
          return options;
        },
        getHTMLAttributes(options) {
          var _b13, _c;
          const { targetWidth, targetHeight } = getTargetDimensions(options);
          const _a29 = options, {
            src,
            width,
            height,
            format,
            quality,
            densities,
            widths,
            formats,
            layout,
            priority,
            fit,
            position,
            background
          } = _a29, attributes = __objRest(_a29, [
            "src",
            "width",
            "height",
            "format",
            "quality",
            "densities",
            "widths",
            "formats",
            "layout",
            "priority",
            "fit",
            "position",
            "background"
          ]);
          return __spreadProps(__spreadValues({}, attributes), {
            width: targetWidth,
            height: targetHeight,
            loading: (_b13 = attributes.loading) != null ? _b13 : "lazy",
            decoding: (_c = attributes.decoding) != null ? _c : "async"
          });
        },
        getSrcSet(options) {
          const { targetWidth, targetHeight } = getTargetDimensions(options);
          const aspectRatio = targetWidth / targetHeight;
          const { widths, densities } = options;
          const targetFormat = options.format;
          let transformedWidths = (widths != null ? widths : []).sort(sortNumeric);
          let imageWidth = options.width;
          let maxWidth = Number.POSITIVE_INFINITY;
          if (isESMImportedImage(options.src)) {
            imageWidth = options.src.width;
            maxWidth = imageWidth;
            if (transformedWidths.length > 0 && transformedWidths.at(-1) > maxWidth) {
              transformedWidths = transformedWidths.filter((width) => width <= maxWidth);
              transformedWidths.push(maxWidth);
            }
          }
          transformedWidths = Array.from(new Set(transformedWidths));
          const _a29 = options, {
            width: transformWidth,
            height: transformHeight
          } = _a29, transformWithoutDimensions = __objRest(_a29, [
            "width",
            "height"
          ]);
          let allWidths = [];
          if (densities) {
            const densityValues = densities.map((density) => {
              if (typeof density === "number") {
                return density;
              } else {
                return Number.parseFloat(density);
              }
            });
            const densityWidths = densityValues.sort(sortNumeric).map((density) => Math.round(targetWidth * density));
            allWidths = densityWidths.map((width, index) => ({
              width,
              descriptor: `${densityValues[index]}x`
            }));
          } else if (transformedWidths.length > 0) {
            allWidths = transformedWidths.map((width) => ({
              width,
              descriptor: `${width}w`
            }));
          }
          return allWidths.map(({ width, descriptor }) => {
            const height = Math.round(width / aspectRatio);
            const transform22 = __spreadProps(__spreadValues({}, transformWithoutDimensions), { width, height });
            return {
              transform: transform22,
              descriptor,
              attributes: targetFormat ? { type: `image/${targetFormat}` } : {}
            };
          });
        },
        getURL(options, imageConfig2) {
          const searchParams = new URLSearchParams();
          if (isESMImportedImage(options.src)) {
            searchParams.append("href", options.src.src);
          } else if (isRemoteAllowed(options.src, imageConfig2)) {
            searchParams.append("href", options.src);
          } else {
            return options.src;
          }
          const params = {
            w: "width",
            h: "height",
            q: "quality",
            f: "format",
            fit: "fit",
            position: "position",
            background: "background"
          };
          Object.entries(params).forEach(([param, key]) => {
            options[key] && searchParams.append(param, options[key].toString());
          });
          const imageEndpoint = joinPaths("/", imageConfig2.endpoint.route);
          let url = `${imageEndpoint}?${searchParams}`;
          if (imageConfig2.assetQueryParams) {
            const assetQueryString = imageConfig2.assetQueryParams.toString();
            if (assetQueryString) {
              url += "&" + assetQueryString;
            }
          }
          return url;
        },
        parseURL(url) {
          var _a29, _b13;
          const params = url.searchParams;
          if (!params.has("href")) {
            return void 0;
          }
          const transform22 = {
            src: params.get("href"),
            width: params.has("w") ? Number.parseInt(params.get("w")) : void 0,
            height: params.has("h") ? Number.parseInt(params.get("h")) : void 0,
            format: params.has("f") ? params.get("f") : void 0,
            quality: params.get("q"),
            fit: params.get("fit"),
            position: (_a29 = params.get("position")) != null ? _a29 : void 0,
            background: (_b13 = params.get("background")) != null ? _b13 : void 0
          };
          return transform22;
        },
        getRemoteSize(url, imageConfig2) {
          return inferRemoteSize(url, imageConfig2);
        }
      };
      PLACEHOLDER_BASE = "astro://placeholder";
      cssFitValues = ["fill", "contain", "cover", "scale-down"];
      Function.prototype.toString.call(Object);
      $$Image = createComponent(($$result, $$props, $$slots) => __async(null, null, function* () {
        var _a29, _b13, _c, _d, _e, _f, _g, _h, _i;
        const Astro2 = $$result.createAstro($$props, $$slots);
        Astro2.self = $$Image;
        const props = Astro2.props;
        if (props.alt === void 0 || props.alt === null) {
          throw new AstroError(ImageMissingAlt);
        }
        if (typeof props.width === "string") {
          props.width = Number.parseInt(props.width);
        }
        if (typeof props.height === "string") {
          props.height = Number.parseInt(props.height);
        }
        const layout = (_b13 = (_a29 = props.layout) != null ? _a29 : imageConfig.layout) != null ? _b13 : "none";
        if (layout !== "none") {
          (_c = props.layout) != null ? _c : props.layout = imageConfig.layout;
          (_e = props.fit) != null ? _e : props.fit = (_d = imageConfig.objectFit) != null ? _d : "cover";
          (_g = props.position) != null ? _g : props.position = (_f = imageConfig.objectPosition) != null ? _f : "center";
        } else if (imageConfig.objectFit || imageConfig.objectPosition) {
          (_h = props.fit) != null ? _h : props.fit = imageConfig.objectFit;
          (_i = props.position) != null ? _i : props.position = imageConfig.objectPosition;
        }
        const image = yield getImage(props);
        const additionalAttributes = {};
        if (image.srcSet.values.length > 0) {
          additionalAttributes.srcset = image.srcSet.attribute;
        }
        const _j = __spreadValues(__spreadValues({}, additionalAttributes), image.attributes), { class: className } = _j, attributes = __objRest(_j, ["class"]);
        return renderTemplate`${maybeRenderHead()}<img${addAttribute(image.src, "src")}${spreadAttributes(attributes)}${addAttribute(className, "class")}>`;
      }), "/home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/node_modules/astro/components/Image.astro", void 0);
      mimes = {
        "3g2": "video/3gpp2",
        "3gp": "video/3gpp",
        "3gpp": "video/3gpp",
        "3mf": "model/3mf",
        "aac": "audio/aac",
        "ac": "application/pkix-attr-cert",
        "adp": "audio/adpcm",
        "adts": "audio/aac",
        "ai": "application/postscript",
        "aml": "application/automationml-aml+xml",
        "amlx": "application/automationml-amlx+zip",
        "amr": "audio/amr",
        "apng": "image/apng",
        "appcache": "text/cache-manifest",
        "appinstaller": "application/appinstaller",
        "appx": "application/appx",
        "appxbundle": "application/appxbundle",
        "asc": "application/pgp-keys",
        "atom": "application/atom+xml",
        "atomcat": "application/atomcat+xml",
        "atomdeleted": "application/atomdeleted+xml",
        "atomsvc": "application/atomsvc+xml",
        "au": "audio/basic",
        "avci": "image/avci",
        "avcs": "image/avcs",
        "avif": "image/avif",
        "aw": "application/applixware",
        "bdoc": "application/bdoc",
        "bin": "application/octet-stream",
        "bmp": "image/bmp",
        "bpk": "application/octet-stream",
        "btf": "image/prs.btif",
        "btif": "image/prs.btif",
        "buffer": "application/octet-stream",
        "ccxml": "application/ccxml+xml",
        "cdfx": "application/cdfx+xml",
        "cdmia": "application/cdmi-capability",
        "cdmic": "application/cdmi-container",
        "cdmid": "application/cdmi-domain",
        "cdmio": "application/cdmi-object",
        "cdmiq": "application/cdmi-queue",
        "cer": "application/pkix-cert",
        "cgm": "image/cgm",
        "cjs": "application/node",
        "class": "application/java-vm",
        "coffee": "text/coffeescript",
        "conf": "text/plain",
        "cpl": "application/cpl+xml",
        "cpt": "application/mac-compactpro",
        "crl": "application/pkix-crl",
        "css": "text/css",
        "csv": "text/csv",
        "cu": "application/cu-seeme",
        "cwl": "application/cwl",
        "cww": "application/prs.cww",
        "davmount": "application/davmount+xml",
        "dbk": "application/docbook+xml",
        "deb": "application/octet-stream",
        "def": "text/plain",
        "deploy": "application/octet-stream",
        "dib": "image/bmp",
        "disposition-notification": "message/disposition-notification",
        "dist": "application/octet-stream",
        "distz": "application/octet-stream",
        "dll": "application/octet-stream",
        "dmg": "application/octet-stream",
        "dms": "application/octet-stream",
        "doc": "application/msword",
        "dot": "application/msword",
        "dpx": "image/dpx",
        "drle": "image/dicom-rle",
        "dsc": "text/prs.lines.tag",
        "dssc": "application/dssc+der",
        "dtd": "application/xml-dtd",
        "dump": "application/octet-stream",
        "dwd": "application/atsc-dwd+xml",
        "ear": "application/java-archive",
        "ecma": "application/ecmascript",
        "elc": "application/octet-stream",
        "emf": "image/emf",
        "eml": "message/rfc822",
        "emma": "application/emma+xml",
        "emotionml": "application/emotionml+xml",
        "eps": "application/postscript",
        "epub": "application/epub+zip",
        "exe": "application/octet-stream",
        "exi": "application/exi",
        "exp": "application/express",
        "exr": "image/aces",
        "ez": "application/andrew-inset",
        "fdf": "application/fdf",
        "fdt": "application/fdt+xml",
        "fits": "image/fits",
        "g3": "image/g3fax",
        "gbr": "application/rpki-ghostbusters",
        "geojson": "application/geo+json",
        "gif": "image/gif",
        "glb": "model/gltf-binary",
        "gltf": "model/gltf+json",
        "gml": "application/gml+xml",
        "gpx": "application/gpx+xml",
        "gram": "application/srgs",
        "grxml": "application/srgs+xml",
        "gxf": "application/gxf",
        "gz": "application/gzip",
        "h261": "video/h261",
        "h263": "video/h263",
        "h264": "video/h264",
        "heic": "image/heic",
        "heics": "image/heic-sequence",
        "heif": "image/heif",
        "heifs": "image/heif-sequence",
        "hej2": "image/hej2k",
        "held": "application/atsc-held+xml",
        "hjson": "application/hjson",
        "hlp": "application/winhlp",
        "hqx": "application/mac-binhex40",
        "hsj2": "image/hsj2",
        "htm": "text/html",
        "html": "text/html",
        "ics": "text/calendar",
        "ief": "image/ief",
        "ifb": "text/calendar",
        "iges": "model/iges",
        "igs": "model/iges",
        "img": "application/octet-stream",
        "in": "text/plain",
        "ini": "text/plain",
        "ink": "application/inkml+xml",
        "inkml": "application/inkml+xml",
        "ipfix": "application/ipfix",
        "iso": "application/octet-stream",
        "its": "application/its+xml",
        "jade": "text/jade",
        "jar": "application/java-archive",
        "jhc": "image/jphc",
        "jls": "image/jls",
        "jp2": "image/jp2",
        "jpe": "image/jpeg",
        "jpeg": "image/jpeg",
        "jpf": "image/jpx",
        "jpg": "image/jpeg",
        "jpg2": "image/jp2",
        "jpgm": "image/jpm",
        "jpgv": "video/jpeg",
        "jph": "image/jph",
        "jpm": "image/jpm",
        "jpx": "image/jpx",
        "js": "text/javascript",
        "json": "application/json",
        "json5": "application/json5",
        "jsonld": "application/ld+json",
        "jsonml": "application/jsonml+json",
        "jsx": "text/jsx",
        "jt": "model/jt",
        "jxl": "image/jxl",
        "jxr": "image/jxr",
        "jxra": "image/jxra",
        "jxrs": "image/jxrs",
        "jxs": "image/jxs",
        "jxsc": "image/jxsc",
        "jxsi": "image/jxsi",
        "jxss": "image/jxss",
        "kar": "audio/midi",
        "ktx": "image/ktx",
        "ktx2": "image/ktx2",
        "less": "text/less",
        "lgr": "application/lgr+xml",
        "list": "text/plain",
        "litcoffee": "text/coffeescript",
        "log": "text/plain",
        "lostxml": "application/lost+xml",
        "lrf": "application/octet-stream",
        "m1v": "video/mpeg",
        "m21": "application/mp21",
        "m2a": "audio/mpeg",
        "m2t": "video/mp2t",
        "m2ts": "video/mp2t",
        "m2v": "video/mpeg",
        "m3a": "audio/mpeg",
        "m4a": "audio/mp4",
        "m4p": "application/mp4",
        "m4s": "video/iso.segment",
        "ma": "application/mathematica",
        "mads": "application/mads+xml",
        "maei": "application/mmt-aei+xml",
        "man": "text/troff",
        "manifest": "text/cache-manifest",
        "map": "application/json",
        "mar": "application/octet-stream",
        "markdown": "text/markdown",
        "mathml": "application/mathml+xml",
        "mb": "application/mathematica",
        "mbox": "application/mbox",
        "md": "text/markdown",
        "mdx": "text/mdx",
        "me": "text/troff",
        "mesh": "model/mesh",
        "meta4": "application/metalink4+xml",
        "metalink": "application/metalink+xml",
        "mets": "application/mets+xml",
        "mft": "application/rpki-manifest",
        "mid": "audio/midi",
        "midi": "audio/midi",
        "mime": "message/rfc822",
        "mj2": "video/mj2",
        "mjp2": "video/mj2",
        "mjs": "text/javascript",
        "mml": "text/mathml",
        "mods": "application/mods+xml",
        "mov": "video/quicktime",
        "mp2": "audio/mpeg",
        "mp21": "application/mp21",
        "mp2a": "audio/mpeg",
        "mp3": "audio/mpeg",
        "mp4": "video/mp4",
        "mp4a": "audio/mp4",
        "mp4s": "application/mp4",
        "mp4v": "video/mp4",
        "mpd": "application/dash+xml",
        "mpe": "video/mpeg",
        "mpeg": "video/mpeg",
        "mpf": "application/media-policy-dataset+xml",
        "mpg": "video/mpeg",
        "mpg4": "video/mp4",
        "mpga": "audio/mpeg",
        "mpp": "application/dash-patch+xml",
        "mrc": "application/marc",
        "mrcx": "application/marcxml+xml",
        "ms": "text/troff",
        "mscml": "application/mediaservercontrol+xml",
        "msh": "model/mesh",
        "msi": "application/octet-stream",
        "msix": "application/msix",
        "msixbundle": "application/msixbundle",
        "msm": "application/octet-stream",
        "msp": "application/octet-stream",
        "mtl": "model/mtl",
        "mts": "video/mp2t",
        "musd": "application/mmt-usd+xml",
        "mxf": "application/mxf",
        "mxmf": "audio/mobile-xmf",
        "mxml": "application/xv+xml",
        "n3": "text/n3",
        "nb": "application/mathematica",
        "nq": "application/n-quads",
        "nt": "application/n-triples",
        "obj": "model/obj",
        "oda": "application/oda",
        "oga": "audio/ogg",
        "ogg": "audio/ogg",
        "ogv": "video/ogg",
        "ogx": "application/ogg",
        "omdoc": "application/omdoc+xml",
        "onepkg": "application/onenote",
        "onetmp": "application/onenote",
        "onetoc": "application/onenote",
        "onetoc2": "application/onenote",
        "opf": "application/oebps-package+xml",
        "opus": "audio/ogg",
        "otf": "font/otf",
        "owl": "application/rdf+xml",
        "oxps": "application/oxps",
        "p10": "application/pkcs10",
        "p7c": "application/pkcs7-mime",
        "p7m": "application/pkcs7-mime",
        "p7s": "application/pkcs7-signature",
        "p8": "application/pkcs8",
        "pdf": "application/pdf",
        "pfr": "application/font-tdpfr",
        "pgp": "application/pgp-encrypted",
        "pkg": "application/octet-stream",
        "pki": "application/pkixcmp",
        "pkipath": "application/pkix-pkipath",
        "pls": "application/pls+xml",
        "png": "image/png",
        "prc": "model/prc",
        "prf": "application/pics-rules",
        "provx": "application/provenance+xml",
        "ps": "application/postscript",
        "pskcxml": "application/pskc+xml",
        "pti": "image/prs.pti",
        "qt": "video/quicktime",
        "raml": "application/raml+yaml",
        "rapd": "application/route-apd+xml",
        "rdf": "application/rdf+xml",
        "relo": "application/p2p-overlay+xml",
        "rif": "application/reginfo+xml",
        "rl": "application/resource-lists+xml",
        "rld": "application/resource-lists-diff+xml",
        "rmi": "audio/midi",
        "rnc": "application/relax-ng-compact-syntax",
        "rng": "application/xml",
        "roa": "application/rpki-roa",
        "roff": "text/troff",
        "rq": "application/sparql-query",
        "rs": "application/rls-services+xml",
        "rsat": "application/atsc-rsat+xml",
        "rsd": "application/rsd+xml",
        "rsheet": "application/urc-ressheet+xml",
        "rss": "application/rss+xml",
        "rtf": "text/rtf",
        "rtx": "text/richtext",
        "rusd": "application/route-usd+xml",
        "s3m": "audio/s3m",
        "sbml": "application/sbml+xml",
        "scq": "application/scvp-cv-request",
        "scs": "application/scvp-cv-response",
        "sdp": "application/sdp",
        "senmlx": "application/senml+xml",
        "sensmlx": "application/sensml+xml",
        "ser": "application/java-serialized-object",
        "setpay": "application/set-payment-initiation",
        "setreg": "application/set-registration-initiation",
        "sgi": "image/sgi",
        "sgm": "text/sgml",
        "sgml": "text/sgml",
        "shex": "text/shex",
        "shf": "application/shf+xml",
        "shtml": "text/html",
        "sieve": "application/sieve",
        "sig": "application/pgp-signature",
        "sil": "audio/silk",
        "silo": "model/mesh",
        "siv": "application/sieve",
        "slim": "text/slim",
        "slm": "text/slim",
        "sls": "application/route-s-tsid+xml",
        "smi": "application/smil+xml",
        "smil": "application/smil+xml",
        "snd": "audio/basic",
        "so": "application/octet-stream",
        "spdx": "text/spdx",
        "spp": "application/scvp-vp-response",
        "spq": "application/scvp-vp-request",
        "spx": "audio/ogg",
        "sql": "application/sql",
        "sru": "application/sru+xml",
        "srx": "application/sparql-results+xml",
        "ssdl": "application/ssdl+xml",
        "ssml": "application/ssml+xml",
        "stk": "application/hyperstudio",
        "stl": "model/stl",
        "stpx": "model/step+xml",
        "stpxz": "model/step-xml+zip",
        "stpz": "model/step+zip",
        "styl": "text/stylus",
        "stylus": "text/stylus",
        "svg": "image/svg+xml",
        "svgz": "image/svg+xml",
        "swidtag": "application/swid+xml",
        "t": "text/troff",
        "t38": "image/t38",
        "td": "application/urc-targetdesc+xml",
        "tei": "application/tei+xml",
        "teicorpus": "application/tei+xml",
        "text": "text/plain",
        "tfi": "application/thraud+xml",
        "tfx": "image/tiff-fx",
        "tif": "image/tiff",
        "tiff": "image/tiff",
        "toml": "application/toml",
        "tr": "text/troff",
        "trig": "application/trig",
        "ts": "video/mp2t",
        "tsd": "application/timestamped-data",
        "tsv": "text/tab-separated-values",
        "ttc": "font/collection",
        "ttf": "font/ttf",
        "ttl": "text/turtle",
        "ttml": "application/ttml+xml",
        "txt": "text/plain",
        "u3d": "model/u3d",
        "u8dsn": "message/global-delivery-status",
        "u8hdr": "message/global-headers",
        "u8mdn": "message/global-disposition-notification",
        "u8msg": "message/global",
        "ubj": "application/ubjson",
        "uri": "text/uri-list",
        "uris": "text/uri-list",
        "urls": "text/uri-list",
        "vcard": "text/vcard",
        "vrml": "model/vrml",
        "vtt": "text/vtt",
        "vxml": "application/voicexml+xml",
        "war": "application/java-archive",
        "wasm": "application/wasm",
        "wav": "audio/wav",
        "weba": "audio/webm",
        "webm": "video/webm",
        "webmanifest": "application/manifest+json",
        "webp": "image/webp",
        "wgsl": "text/wgsl",
        "wgt": "application/widget",
        "wif": "application/watcherinfo+xml",
        "wmf": "image/wmf",
        "woff": "font/woff",
        "woff2": "font/woff2",
        "wrl": "model/vrml",
        "wsdl": "application/wsdl+xml",
        "wspolicy": "application/wspolicy+xml",
        "x3d": "model/x3d+xml",
        "x3db": "model/x3d+fastinfoset",
        "x3dbz": "model/x3d+binary",
        "x3dv": "model/x3d-vrml",
        "x3dvz": "model/x3d+vrml",
        "x3dz": "model/x3d+xml",
        "xaml": "application/xaml+xml",
        "xav": "application/xcap-att+xml",
        "xca": "application/xcap-caps+xml",
        "xcs": "application/calendar+xml",
        "xdf": "application/xcap-diff+xml",
        "xdssc": "application/dssc+xml",
        "xel": "application/xcap-el+xml",
        "xenc": "application/xenc+xml",
        "xer": "application/patch-ops-error+xml",
        "xfdf": "application/xfdf",
        "xht": "application/xhtml+xml",
        "xhtml": "application/xhtml+xml",
        "xhvml": "application/xv+xml",
        "xlf": "application/xliff+xml",
        "xm": "audio/xm",
        "xml": "text/xml",
        "xns": "application/xcap-ns+xml",
        "xop": "application/xop+xml",
        "xpl": "application/xproc+xml",
        "xsd": "application/xml",
        "xsf": "application/prs.xsf+xml",
        "xsl": "application/xml",
        "xslt": "application/xml",
        "xspf": "application/xspf+xml",
        "xvm": "application/xv+xml",
        "xvml": "application/xv+xml",
        "yaml": "text/yaml",
        "yang": "application/yang",
        "yin": "application/yin+xml",
        "yml": "text/yaml",
        "zip": "application/zip"
      };
      $$Picture = createComponent(($$result, $$props, $$slots) => __async(null, null, function* () {
        var _b13, _c, _d, _e, _f, _g, _h, _i, _j, _k, _l, _m;
        const Astro2 = $$result.createAstro($$props, $$slots);
        Astro2.self = $$Picture;
        const defaultFormats = ["webp"];
        const defaultFallbackFormat = "png";
        const specialFormatsFallback = ["gif", "svg", "jpg", "jpeg"];
        const _a29 = Astro2.props, { formats = defaultFormats, pictureAttributes = {}, fallbackFormat } = _a29, props = __objRest(_a29, ["formats", "pictureAttributes", "fallbackFormat"]);
        if (props.alt === void 0 || props.alt === null) {
          throw new AstroError(ImageMissingAlt);
        }
        const scopedStyleClass = (_c = (_b13 = props.class) == null ? void 0 : _b13.match(/\bastro-\w{8}\b/)) == null ? void 0 : _c[0];
        if (scopedStyleClass) {
          if (pictureAttributes.class) {
            pictureAttributes.class = `${pictureAttributes.class} ${scopedStyleClass}`;
          } else {
            pictureAttributes.class = scopedStyleClass;
          }
        }
        const layout = (_e = (_d = props.layout) != null ? _d : imageConfig.layout) != null ? _e : "none";
        const useResponsive = layout !== "none";
        if (useResponsive) {
          (_f = props.layout) != null ? _f : props.layout = imageConfig.layout;
          (_h = props.fit) != null ? _h : props.fit = (_g = imageConfig.objectFit) != null ? _g : "cover";
          (_j = props.position) != null ? _j : props.position = (_i = imageConfig.objectPosition) != null ? _i : "center";
        } else if (imageConfig.objectFit || imageConfig.objectPosition) {
          (_k = props.fit) != null ? _k : props.fit = imageConfig.objectFit;
          (_l = props.position) != null ? _l : props.position = imageConfig.objectPosition;
        }
        for (const key in props) {
          if (key.startsWith("data-astro-cid")) {
            pictureAttributes[key] = props[key];
          }
        }
        const originalSrc = yield resolveSrc(props.src);
        const optimizedImages = yield Promise.all(
          formats.map(
            (format) => __async(null, null, function* () {
              return yield getImage(__spreadProps(__spreadValues({}, props), {
                src: originalSrc,
                format,
                widths: props.widths,
                densities: props.densities
              }));
            })
          )
        );
        const clonedSrc = isESMImportedImage(originalSrc) ? (
          // @ts-expect-error - clone is a private, hidden prop
          (_m = originalSrc.clone) != null ? _m : originalSrc
        ) : originalSrc;
        let resultFallbackFormat = fallbackFormat != null ? fallbackFormat : defaultFallbackFormat;
        if (!fallbackFormat && isESMImportedImage(clonedSrc) && specialFormatsFallback.includes(clonedSrc.format)) {
          resultFallbackFormat = clonedSrc.format;
        }
        const fallbackImage = yield getImage(__spreadProps(__spreadValues({}, props), {
          format: resultFallbackFormat,
          widths: props.widths,
          densities: props.densities
        }));
        const imgAdditionalAttributes = {};
        const sourceAdditionalAttributes = {};
        if (props.sizes) {
          sourceAdditionalAttributes.sizes = props.sizes;
        }
        if (fallbackImage.srcSet.values.length > 0) {
          imgAdditionalAttributes.srcset = fallbackImage.srcSet.attribute;
        }
        const _n = __spreadValues(__spreadValues({}, imgAdditionalAttributes), fallbackImage.attributes), { class: className } = _n, attributes = __objRest(_n, ["class"]);
        return renderTemplate`${maybeRenderHead()}<picture${spreadAttributes(pictureAttributes)}> ${Object.entries(optimizedImages).map(([_, image]) => {
          var _a30, _b14;
          const srcsetAttribute = props.densities || !props.densities && !props.widths && !useResponsive ? `${image.src}${image.srcSet.values.length > 0 ? ", " + image.srcSet.attribute : ""}` : image.srcSet.attribute;
          return renderTemplate`<source${addAttribute(srcsetAttribute, "srcset")}${addAttribute((_b14 = lookup((_a30 = image.options.format) != null ? _a30 : image.src)) != null ? _b14 : `image/${image.options.format}`, "type")}${spreadAttributes(sourceAdditionalAttributes)}>`;
        })}  <img${addAttribute(fallbackImage.src, "src")}${spreadAttributes(attributes)}${addAttribute(className, "class")}> </picture>`;
      }), "/home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/node_modules/astro/components/Picture.astro", void 0);
      componentDataByCssVariable = /* @__PURE__ */ new Map([]);
      $$Font = createComponent(($$result, $$props, $$slots) => {
        const Astro2 = $$result.createAstro($$props, $$slots);
        Astro2.self = $$Font;
        const { cssVariable, preload = false } = Astro2.props;
        const data = componentDataByCssVariable.get(cssVariable);
        if (!data) {
          throw new AstroError(__spreadProps(__spreadValues({}, FontFamilyNotFound), {
            message: FontFamilyNotFound.message(cssVariable)
          }));
        }
        const filteredPreloadData = filterPreloads(data.preloads, preload);
        return renderTemplate`<style>${unescapeHTML(data.css)}</style>${filteredPreloadData == null ? void 0 : filteredPreloadData.map(({ url, type }) => renderTemplate`<link rel="preload"${addAttribute(url, "href")} as="font"${addAttribute(`font/${type}`, "type")} crossorigin>`)}`;
      }, "/home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/node_modules/astro/components/Font.astro", void 0);
      SsrRuntimeFontFileUrlResolver = class {
        constructor({
          urls
        }) {
          __privateAdd(this, _urls);
          __privateSet(this, _urls, urls);
        }
        resolve(url, requestUrl) {
          if (!__privateGet(this, _urls).has(url)) {
            return null;
          }
          if (!url.startsWith("/")) {
            return url;
          }
          if (!requestUrl) {
            throw new AstroError(MissingGetFontFileRequestUrl);
          }
          return `${requestUrl.origin}${url}`;
        }
      };
      _urls = new WeakMap();
      new SsrRuntimeFontFileUrlResolver({
        urls: /* @__PURE__ */ new Set([])
      });
      assetQueryParams = void 0;
      imageConfig = { "endpoint": { "route": "/_image", "entrypoint": "@astrojs/cloudflare/image-transform-endpoint" }, "service": { "entrypoint": "@astrojs/cloudflare/image-service-workerd", "config": {} }, "dangerouslyProcessSVG": false, "domains": [], "remotePatterns": [], "responsiveStyles": false };
      Object.defineProperty(imageConfig, "assetQueryParams", {
        value: assetQueryParams,
        enumerable: false,
        configurable: true
      });
      getImage = (options) => __async(null, null, function* () {
        return yield getImage$1(options, imageConfig);
      });
      qualityTable = {
        low: 25,
        mid: 50,
        high: 80,
        max: 100
      };
      prerender = false;
      GET = (ctx) => __async(null, null, function* () {
        const cache = caches.default;
        if (cache) {
          const cached = yield cache.match(ctx.request.url);
          if (cached) return cached;
        }
        const response = yield transform(ctx.request.url, env.IMAGES, env.ASSETS);
        if (!response.ok) return response;
        const headers = new Headers(response.headers);
        headers.set("Cache-Control", "public, max-age=31536000, immutable");
        const cachedResponse = new Response(response.body, {
          status: response.status,
          headers
        });
        if (cache) {
          const cfContext = ctx.locals.cfContext;
          if (cfContext) {
            cfContext.waitUntil(cache.put(ctx.request.url, cachedResponse.clone()));
          }
        }
        return cachedResponse;
      });
      _page = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
        __proto__: null,
        GET,
        prerender
      }, Symbol.toStringTag, { value: "Module" }));
      page = () => _page;
      imageTransformEndpoint___js = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
        __proto__: null,
        page
      }, Symbol.toStringTag, { value: "Module" }));
    }
  });

  // chunks/index_DHcMpgJR.mjs
  var index_DHcMpgJR_exports = {};
  __export(index_DHcMpgJR_exports, {
    page: () => page2
  });
  var _a7, _a8, _b4, $$Index, $$file, $$url, _page2, page2;
  var init_index_DHcMpgJR = __esm({
    "chunks/index_DHcMpgJR.mjs"() {
      "use strict";
      init_astro_component_CZmLnn6R();
      init_worker_entry_D8ZVWWpA();
      (_a7 = globalThis.process) != null ? _a7 : globalThis.process = {};
      (_b4 = (_a8 = globalThis.process).env) != null ? _b4 : _a8.env = {};
      $$Index = createComponent(($$result, $$props, $$slots) => __async(null, null, function* () {
        const Astro2 = $$result.createAstro($$props, $$slots);
        Astro2.self = $$Index;
        const res = yield fetch("https://catfact.ninja/fact");
        if (!res.ok) {
          throw new Error(`catfact.ninja HTTP ${res.status}`);
        }
        const body = yield res.text();
        const data = JSON.parse(body);
        if (!(data == null ? void 0 : data.fact) || typeof data.fact !== "string") {
          throw new Error("catfact.ninja: unexpected JSON shape");
        }
        const fact = data.fact;
        const title = "eletrocromo \xB7 Astro cats";
        const renderedAt = (/* @__PURE__ */ new Date()).toISOString();
        return renderTemplate`<html lang="en" data-astro-cid-j7pv25f6> <head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>${title}</title>${renderHead()}</head> <body data-astro-cid-j7pv25f6> <h1 data-astro-cid-j7pv25f6>${title}</h1> <p class="lead" data-astro-cid-j7pv25f6>
Cat fact resolved in Astro frontmatter on each request (SSR) via host
<code data-astro-cid-j7pv25f6>fetch</code> — no client JS.
</p> <section class="card" data-astro-cid-j7pv25f6> <blockquote data-astro-cid-j7pv25f6>${fact}</blockquote> <span class="badge" data-astro-cid-j7pv25f6>live · catfact.ninja</span> </section> <p class="meta" data-astro-cid-j7pv25f6>
Stack: astro SSR → orvalho workers → eletrocromo · rendered ${renderedAt} </p> </body></html>`;
      }), "/home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/src/pages/index.astro", void 0);
      $$file = "/home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/src/pages/index.astro";
      $$url = "";
      _page2 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
        __proto__: null,
        default: $$Index,
        file: $$file,
        url: $$url
      }, Symbol.toStringTag, { value: "Module" }));
      page2 = () => _page2;
    }
  });

  // chunks/noop-entrypoint_DsBX4kaI.mjs
  var noop_entrypoint_DsBX4kaI_exports = {};
  __export(noop_entrypoint_DsBX4kaI_exports, {
    server: () => server
  });
  var _a9, _a10, _b5, server;
  var init_noop_entrypoint_DsBX4kaI = __esm({
    "chunks/noop-entrypoint_DsBX4kaI.mjs"() {
      "use strict";
      (_a9 = globalThis.process) != null ? _a9 : globalThis.process = {};
      (_b5 = (_a10 = globalThis.process).env) != null ? _b5 : _a10.env = {};
      server = {};
    }
  });

  // virtual_astro_middleware.mjs
  var virtual_astro_middleware_exports = {};
  __export(virtual_astro_middleware_exports, {
    onRequest: () => onRequest
  });
  var _a11, _a12, _b6, onRequest;
  var init_virtual_astro_middleware = __esm({
    "virtual_astro_middleware.mjs"() {
      "use strict";
      (_a11 = globalThis.process) != null ? _a11 : globalThis.process = {};
      (_b6 = (_a12 = globalThis.process).env) != null ? _b6 : _a12.env = {};
      onRequest = (_, next) => next();
    }
  });

  // chunks/_virtual_astro_session-driver_Csu6PJY6.mjs
  var virtual_astro_session_driver_Csu6PJY6_exports = {};
  __export(virtual_astro_session_driver_Csu6PJY6_exports, {
    default: () => _default
  });
  function defineDriver(factory) {
    return factory;
  }
  function normalizeKey(key, sep = ":") {
    if (!key) {
      return "";
    }
    return key.replace(/[:/\\]/g, sep).replace(/^[:/\\]|[:/\\]$/g, "");
  }
  function joinKeys(...keys) {
    return keys.map((key) => normalizeKey(key)).filter(Boolean).join(":");
  }
  function createError(driver, message, opts) {
    const err = new Error(`[unstorage] [${driver}] ${message}`, opts);
    if (Error.captureStackTrace) {
      Error.captureStackTrace(err, createError);
    }
    return err;
  }
  function getBinding(binding) {
    var _a29;
    let bindingName = "[binding]";
    if (typeof binding === "string") {
      bindingName = binding;
      binding = globalThis[bindingName] || ((_a29 = globalThis.__env__) == null ? void 0 : _a29[bindingName]);
    }
    if (!binding) {
      throw createError(
        "cloudflare",
        `Invalid binding \`${bindingName}\`: \`${binding}\``
      );
    }
    for (const key of ["get", "put", "delete"]) {
      if (!(key in binding)) {
        throw createError(
          "cloudflare",
          `Invalid binding \`${bindingName}\`: \`${key}\` key is missing`
        );
      }
    }
    return binding;
  }
  function getKVBinding(binding = "STORAGE") {
    return getBinding(binding);
  }
  var _a13, _a14, _b7, DRIVER_NAME, _default;
  var init_virtual_astro_session_driver_Csu6PJY6 = __esm({
    "chunks/_virtual_astro_session-driver_Csu6PJY6.mjs"() {
      "use strict";
      (_a13 = globalThis.process) != null ? _a13 : globalThis.process = {};
      (_b7 = (_a14 = globalThis.process).env) != null ? _b7 : _a14.env = {};
      DRIVER_NAME = "cloudflare-kv-binding";
      _default = defineDriver((opts) => {
        const r2 = (key = "") => opts.base ? joinKeys(opts.base, key) : key;
        function getKeys(base = "") {
          return __async(this, null, function* () {
            base = r2(base);
            const binding = getKVBinding(opts.binding);
            const keys = [];
            let cursor = void 0;
            do {
              const kvList = yield binding.list({ prefix: base || void 0, cursor });
              keys.push(...kvList.keys);
              cursor = kvList.list_complete ? void 0 : kvList.cursor;
            } while (cursor);
            return keys.map((key) => key.name);
          });
        }
        return {
          name: DRIVER_NAME,
          options: opts,
          getInstance: () => getKVBinding(opts.binding),
          hasItem(key) {
            return __async(this, null, function* () {
              key = r2(key);
              const binding = getKVBinding(opts.binding);
              return (yield binding.get(key)) !== null;
            });
          },
          getItem(key) {
            key = r2(key);
            const binding = getKVBinding(opts.binding);
            return binding.get(key);
          },
          setItem(key, value, topts) {
            var _a30;
            key = r2(key);
            const binding = getKVBinding(opts.binding);
            return binding.put(
              key,
              value,
              topts ? __spreadValues({
                expirationTtl: (topts == null ? void 0 : topts.ttl) ? Math.max(topts.ttl, (_a30 = opts.minTTL) != null ? _a30 : 60) : void 0
              }, topts) : void 0
            );
          },
          removeItem(key) {
            key = r2(key);
            const binding = getKVBinding(opts.binding);
            return binding.delete(key);
          },
          getKeys(base) {
            return getKeys(base).then(
              (keys) => keys.map((key) => opts.base ? key.slice(opts.base.length) : key)
            );
          },
          clear(base) {
            return __async(this, null, function* () {
              const binding = getKVBinding(opts.binding);
              const keys = yield getKeys(base);
              yield Promise.all(keys.map((key) => binding.delete(key)));
            });
          }
        };
      });
    }
  });

  // chunks/_virtual_astro_server-island-manifest_CFl7y3Qj.mjs
  var virtual_astro_server_island_manifest_CFl7y3Qj_exports = {};
  __export(virtual_astro_server_island_manifest_CFl7y3Qj_exports, {
    serverIslandMap: () => serverIslandMap,
    serverIslandNameMap: () => serverIslandNameMap
  });
  var _a15, _a16, _b8, serverIslandMap, serverIslandNameMap;
  var init_virtual_astro_server_island_manifest_CFl7y3Qj = __esm({
    "chunks/_virtual_astro_server-island-manifest_CFl7y3Qj.mjs"() {
      "use strict";
      (_a15 = globalThis.process) != null ? _a15 : globalThis.process = {};
      (_b8 = (_a16 = globalThis.process).env) != null ? _b8 : _a16.env = {};
      serverIslandMap = /* @__PURE__ */ new Map([]);
      serverIslandNameMap = /* @__PURE__ */ new Map([]);
    }
  });

  // chunks/worker-entry_D8ZVWWpA.mjs
  function appendForwardSlash(path) {
    return path.endsWith("/") ? path : path + "/";
  }
  function prependForwardSlash(path) {
    return path[0] === "/" ? path : "/" + path;
  }
  function collapseDuplicateLeadingSlashes(path) {
    if (!path) {
      return path;
    }
    return path.replace(MANY_LEADING_SLASHES, "/");
  }
  function collapseDuplicateSlashes(path) {
    if (!path) {
      return path;
    }
    return path.replace(MANY_SLASHES, "/");
  }
  function collapseDuplicateTrailingSlashes(path, trailingSlash) {
    if (!path) {
      return path;
    }
    return path.replace(MANY_TRAILING_SLASHES, trailingSlash ? "/" : "") || "/";
  }
  function removeTrailingForwardSlash(path) {
    return path.endsWith("/") ? path.slice(0, path.length - 1) : path;
  }
  function removeLeadingForwardSlash(path) {
    return path.startsWith("/") ? path.substring(1) : path;
  }
  function trimSlashes(path) {
    return path.replace(/^\/|\/$/g, "");
  }
  function isString(path) {
    return typeof path === "string" || path instanceof String;
  }
  function isInternalPath(path) {
    return INTERNAL_PREFIXES.has(path.slice(0, 2)) && !JUST_SLASHES.test(path);
  }
  function joinPaths(...paths) {
    return paths.filter(isString).map((path, i2) => {
      if (i2 === 0) {
        return removeTrailingForwardSlash(path);
      } else if (i2 === paths.length - 1) {
        return removeLeadingForwardSlash(path);
      } else {
        return trimSlashes(path);
      }
    }).join("/");
  }
  function removeQueryString(path) {
    const index = path.lastIndexOf("?");
    return index > 0 ? path.substring(0, index) : path;
  }
  function isRemotePath(src) {
    if (!src) return false;
    const trimmed = src.trim();
    if (!trimmed) return false;
    let decoded = trimmed;
    let previousDecoded = "";
    let maxIterations = 10;
    while (decoded !== previousDecoded && maxIterations > 0) {
      previousDecoded = decoded;
      try {
        decoded = decodeURIComponent(decoded);
      } catch (e2) {
        break;
      }
      maxIterations--;
    }
    if (/^[a-zA-Z]:/.test(decoded)) {
      return false;
    }
    if (decoded[0] === "/" && /^\/[\w.@-]/.test(decoded)) {
      return false;
    }
    if (decoded[0] === "\\") {
      return true;
    }
    if (decoded.startsWith("//")) {
      return true;
    }
    try {
      const url = new URL(decoded, "http://n");
      if (url.username || url.password) {
        return true;
      }
      if (decoded.includes("@") && !url.pathname.includes("@") && !url.search.includes("@")) {
        return true;
      }
      if (url.origin !== "http://n") {
        const protocol = url.protocol.toLowerCase();
        if (protocol === "file:") {
          return false;
        }
        return true;
      }
      if (URL.canParse(decoded)) {
        return true;
      }
      return false;
    } catch (e2) {
      return true;
    }
  }
  function slash(path) {
    return path.replace(/\\/g, "/");
  }
  function fileExtension(path) {
    const ext = path.split(".").pop();
    return ext !== path ? `.${ext}` : "";
  }
  function hasFileExtension(path) {
    return WITH_FILE_EXT.test(path);
  }
  function is_primitive(thing) {
    return thing === null || typeof thing !== "object" && typeof thing !== "function";
  }
  function is_plain_object(thing) {
    const proto = Object.getPrototypeOf(thing);
    return proto === Object.prototype || proto === null || Object.getPrototypeOf(proto) === null || Object.getOwnPropertyNames(proto).sort().join("\0") === object_proto_names;
  }
  function get_type(thing) {
    return Object.prototype.toString.call(thing).slice(8, -1);
  }
  function get_escaped_char(char) {
    switch (char) {
      case '"':
        return '\\"';
      case "<":
        return "\\u003C";
      case "\\":
        return "\\\\";
      case "\n":
        return "\\n";
      case "\r":
        return "\\r";
      case "	":
        return "\\t";
      case "\b":
        return "\\b";
      case "\f":
        return "\\f";
      case "\u2028":
        return "\\u2028";
      case "\u2029":
        return "\\u2029";
      default:
        return char < " " ? `\\u${char.charCodeAt(0).toString(16).padStart(4, "0")}` : "";
    }
  }
  function stringify_string(str) {
    let result = "";
    let last_pos = 0;
    const len = str.length;
    for (let i2 = 0; i2 < len; i2 += 1) {
      const char = str[i2];
      const replacement = get_escaped_char(char);
      if (replacement) {
        result += str.slice(last_pos, i2) + replacement;
        last_pos = i2 + 1;
      }
    }
    return `"${last_pos === 0 ? str : result + str.slice(last_pos)}"`;
  }
  function enumerable_symbols(object) {
    return Object.getOwnPropertySymbols(object).filter(
      (symbol) => Object.getOwnPropertyDescriptor(object, symbol).enumerable
    );
  }
  function stringify_key(key) {
    return is_identifier.test(key) ? "." + key : "[" + JSON.stringify(key) + "]";
  }
  function is_valid_array_index(n2) {
    if (!Number.isInteger(n2)) return false;
    if (n2 < 0) return false;
    if (n2 > MAX_ARRAY_INDEX) return false;
    return true;
  }
  function is_valid_array_len(n2) {
    if (!Number.isInteger(n2)) return false;
    if (n2 < 0) return false;
    if (n2 > MAX_ARRAY_LEN) return false;
    return true;
  }
  function is_valid_array_index_string(s2) {
    if (s2.length === 0) return false;
    if (s2.length > 1 && s2.charCodeAt(0) === 48) return false;
    for (let i2 = 0; i2 < s2.length; i2++) {
      const c = s2.charCodeAt(i2);
      if (c < 48 || c > 57) return false;
    }
    return is_valid_array_index(+s2);
  }
  function valid_array_indices(array2) {
    const keys = Object.keys(array2);
    for (var i2 = keys.length - 1; i2 >= 0; i2--) {
      if (is_valid_array_index_string(keys[i2])) {
        break;
      }
    }
    keys.length = i2 + 1;
    return keys;
  }
  function encode_native(array_buffer) {
    return new Uint8Array(array_buffer).toBase64();
  }
  function decode_native(base64) {
    return Uint8Array.fromBase64(base64).buffer;
  }
  function encode_buffer(array_buffer) {
    return Buffer.from(array_buffer).toString("base64");
  }
  function decode_buffer(base64) {
    return Uint8Array.from(Buffer.from(base64, "base64")).buffer;
  }
  function encode_legacy(array_buffer) {
    const array2 = new Uint8Array(array_buffer);
    let binary2 = "";
    const chunk_size = 32768;
    for (let i2 = 0; i2 < array2.length; i2 += chunk_size) {
      const chunk = array2.subarray(i2, i2 + chunk_size);
      binary2 += String.fromCharCode.apply(null, chunk);
    }
    return btoa(binary2);
  }
  function decode_legacy(base64) {
    const binary_string = atob(base64);
    const len = binary_string.length;
    const array2 = new Uint8Array(len);
    for (let i2 = 0; i2 < len; i2++) {
      array2[i2] = binary_string.charCodeAt(i2);
    }
    return array2.buffer;
  }
  function parse$1(serialized, revivers) {
    return unflatten$1(JSON.parse(serialized), revivers);
  }
  function unflatten$1(parsed, revivers) {
    if (typeof parsed === "number") return hydrate(parsed, true);
    if (!Array.isArray(parsed) || parsed.length === 0) {
      throw new Error("Invalid input");
    }
    const values = (
      /** @type {any[]} */
      parsed
    );
    const hydrated = Array(values.length);
    let hydrating = null;
    function hydrate(index, standalone = false) {
      if (index === UNDEFINED) return void 0;
      if (index === NAN) return NaN;
      if (index === POSITIVE_INFINITY) return Infinity;
      if (index === NEGATIVE_INFINITY) return -Infinity;
      if (index === NEGATIVE_ZERO) return -0;
      if (standalone || typeof index !== "number") {
        throw new Error(`Invalid input`);
      }
      if (index in hydrated) return hydrated[index];
      const value = values[index];
      if (!value || typeof value !== "object") {
        hydrated[index] = value;
      } else if (Array.isArray(value)) {
        if (typeof value[0] === "string") {
          const type = value[0];
          const reviver = revivers && Object.hasOwn(revivers, type) ? revivers[type] : void 0;
          if (reviver) {
            let i2 = value[1];
            if (typeof i2 !== "number") {
              i2 = values.push(value[1]) - 1;
            }
            if (Object.hasOwn(hydrated, i2)) {
              return hydrated[index] = reviver(hydrated[i2]);
            }
            hydrating != null ? hydrating : hydrating = /* @__PURE__ */ new Set();
            if (hydrating.has(i2)) {
              throw new Error("Invalid circular reference");
            }
            hydrating.add(i2);
            hydrated[index] = reviver(hydrate(i2));
            hydrating.delete(i2);
            return hydrated[index];
          }
          switch (type) {
            case "Date":
              hydrated[index] = new Date(value[1]);
              break;
            case "Set":
              const set = /* @__PURE__ */ new Set();
              hydrated[index] = set;
              for (let i2 = 1; i2 < value.length; i2 += 1) {
                set.add(hydrate(value[i2]));
              }
              break;
            case "Map":
              const map = /* @__PURE__ */ new Map();
              hydrated[index] = map;
              for (let i2 = 1; i2 < value.length; i2 += 2) {
                map.set(hydrate(value[i2]), hydrate(value[i2 + 1]));
              }
              break;
            case "RegExp":
              hydrated[index] = new RegExp(value[1], value[2]);
              break;
            case "Object": {
              const wrapped_index = value[1];
              if (typeof values[wrapped_index] === "object" && values[wrapped_index][0] !== "BigInt") {
                throw new Error("Invalid input");
              }
              hydrated[index] = Object(hydrate(wrapped_index));
              break;
            }
            case "BigInt":
              hydrated[index] = BigInt(value[1]);
              break;
            case "null":
              const obj = /* @__PURE__ */ Object.create(null);
              hydrated[index] = obj;
              for (let i2 = 1; i2 < value.length; i2 += 2) {
                if (value[i2] === "__proto__") {
                  throw new Error("Cannot parse an object with a `__proto__` property");
                }
                obj[value[i2]] = hydrate(value[i2 + 1]);
              }
              break;
            case "Int8Array":
            case "Uint8Array":
            case "Uint8ClampedArray":
            case "Int16Array":
            case "Uint16Array":
            case "Float16Array":
            case "Int32Array":
            case "Uint32Array":
            case "Float32Array":
            case "Float64Array":
            case "BigInt64Array":
            case "BigUint64Array":
            case "DataView": {
              if (values[value[1]][0] !== "ArrayBuffer") {
                throw new Error("Invalid data");
              }
              const TypedArrayConstructor = globalThis[type];
              const buffer2 = hydrate(value[1]);
              hydrated[index] = value[2] !== void 0 ? new TypedArrayConstructor(buffer2, value[2], value[3]) : new TypedArrayConstructor(buffer2);
              break;
            }
            case "ArrayBuffer": {
              const base64 = value[1];
              if (typeof base64 !== "string") {
                throw new Error("Invalid ArrayBuffer encoding");
              }
              const arraybuffer = decode64(base64);
              hydrated[index] = arraybuffer;
              break;
            }
            case "Temporal.Duration":
            case "Temporal.Instant":
            case "Temporal.PlainDate":
            case "Temporal.PlainTime":
            case "Temporal.PlainDateTime":
            case "Temporal.PlainMonthDay":
            case "Temporal.PlainYearMonth":
            case "Temporal.ZonedDateTime": {
              const temporalName = type.slice(9);
              hydrated[index] = Temporal[temporalName].from(value[1]);
              break;
            }
            case "URL": {
              const url = new URL(value[1]);
              hydrated[index] = url;
              break;
            }
            case "URLSearchParams": {
              const url = new URLSearchParams(value[1]);
              hydrated[index] = url;
              break;
            }
            default:
              throw new Error(`Unknown type ${type}`);
          }
        } else if (value[0] === SPARSE) {
          const len = value[1];
          if (!is_valid_array_len(len)) {
            throw new Error("Invalid input");
          }
          const array2 = [];
          hydrated[index] = array2;
          array2[MAX_ARRAY_INDEX] = void 0;
          delete array2[MAX_ARRAY_INDEX];
          for (let i2 = 2; i2 < value.length; i2 += 2) {
            const idx = value[i2];
            if (!is_valid_array_index(idx) || idx >= len) {
              throw new Error("Invalid input");
            }
            array2[idx] = hydrate(value[i2 + 1]);
          }
          array2.length = len;
        } else {
          const array2 = new Array(value.length);
          hydrated[index] = array2;
          for (let i2 = 0; i2 < value.length; i2 += 1) {
            const n2 = value[i2];
            if (n2 === HOLE) continue;
            array2[i2] = hydrate(n2);
          }
        }
      } else {
        const object = {};
        hydrated[index] = object;
        for (const key of Object.keys(value)) {
          if (key === "__proto__") {
            throw new Error("Cannot parse an object with a `__proto__` property");
          }
          const n2 = value[key];
          object[key] = hydrate(n2);
        }
      }
      return hydrated[index];
    }
    return hydrate(0);
  }
  function stringify$2(value, reducers) {
    const stringified = run(false, value, reducers);
    return typeof stringified === "string" ? stringified : `[${stringified.join(",")}]`;
  }
  function run(async, value, reducers) {
    const stringified = [];
    const indexes = /* @__PURE__ */ new Map();
    const custom2 = [];
    if (reducers) {
      for (const key of Object.getOwnPropertyNames(reducers)) {
        custom2.push({ key, fn: reducers[key] });
      }
    }
    const keys = [];
    let p = 0;
    function flatten(thing, index2) {
      if (thing === void 0) return UNDEFINED;
      if (Number.isNaN(thing)) return NAN;
      if (thing === Infinity) return POSITIVE_INFINITY;
      if (thing === -Infinity) return NEGATIVE_INFINITY;
      if (thing === 0 && 1 / thing < 0) return NEGATIVE_ZERO;
      if (indexes.has(thing)) return (
        /** @type {number} */
        indexes.get(thing)
      );
      index2 != null ? index2 : index2 = p++;
      indexes.set(thing, index2);
      for (const { key, fn } of custom2) {
        const value2 = fn(thing);
        if (value2) {
          stringified[index2] = `["${key}",${flatten(value2)}]`;
          return index2;
        }
      }
      if (typeof thing === "function") {
        throw new DevalueError(`Cannot stringify a function`, keys, thing, value);
      } else if (typeof thing === "symbol") {
        throw new DevalueError(`Cannot stringify a Symbol primitive`, keys, thing, value);
      }
      let str = "";
      if (is_primitive(thing)) {
        str = stringify_primitive(thing);
      } else if (typeof thing.then === "function") {
        {
          throw new DevalueError(
            `Cannot stringify a Promise or thenable \u2014 use stringifyAsync instead`,
            keys,
            thing,
            value
          );
        }
      } else {
        const type = get_type(thing);
        switch (type) {
          case "Number":
          case "String":
          case "Boolean":
          case "BigInt":
            str = `["Object",${flatten(thing.valueOf())}]`;
            break;
          case "Date":
            const valid = !isNaN(thing.getDate());
            str = `["Date","${valid ? thing.toISOString() : ""}"]`;
            break;
          case "URL":
            str = `["URL",${stringify_string(thing.toString())}]`;
            break;
          case "URLSearchParams":
            str = `["URLSearchParams",${stringify_string(thing.toString())}]`;
            break;
          case "RegExp":
            const { source, flags } = thing;
            str = flags ? `["RegExp",${stringify_string(source)},"${flags}"]` : `["RegExp",${stringify_string(source)}]`;
            break;
          case "Array": {
            let mostly_dense = false;
            str = "[";
            for (let i2 = 0; i2 < thing.length; i2 += 1) {
              if (i2 > 0) str += ",";
              if (Object.hasOwn(thing, i2)) {
                keys.push(`[${i2}]`);
                str += flatten(thing[i2]);
                keys.pop();
              } else if (mostly_dense) {
                str += HOLE;
              } else {
                const populated_keys = valid_array_indices(
                  /** @type {any[]} */
                  thing
                );
                const population = populated_keys.length;
                const d = String(thing.length).length;
                const hole_cost = (thing.length - population) * 3;
                const sparse_cost = 4 + d + population * (d + 1);
                if (hole_cost > sparse_cost) {
                  str = "[" + SPARSE + "," + thing.length;
                  for (let j = 0; j < populated_keys.length; j++) {
                    const key = populated_keys[j];
                    keys.push(`[${key}]`);
                    str += "," + key + "," + flatten(thing[key]);
                    keys.pop();
                  }
                  break;
                } else {
                  mostly_dense = true;
                  str += HOLE;
                }
              }
            }
            str += "]";
            break;
          }
          case "Set":
            str = '["Set"';
            for (const value2 of thing) {
              str += `,${flatten(value2)}`;
            }
            str += "]";
            break;
          case "Map":
            str = '["Map"';
            for (const [key, value2] of thing) {
              keys.push(`.get(${is_primitive(key) ? stringify_primitive(key) : "..."})`);
              str += `,${flatten(key)},${flatten(value2)}`;
              keys.pop();
            }
            str += "]";
            break;
          case "Int8Array":
          case "Uint8Array":
          case "Uint8ClampedArray":
          case "Int16Array":
          case "Uint16Array":
          case "Float16Array":
          case "Int32Array":
          case "Uint32Array":
          case "Float32Array":
          case "Float64Array":
          case "BigInt64Array":
          case "BigUint64Array": {
            const typedArray = thing;
            str = '["' + type + '",' + flatten(typedArray.buffer);
            if (typedArray.byteLength !== typedArray.buffer.byteLength) {
              str += `,${typedArray.byteOffset},${typedArray.length}`;
            }
            str += "]";
            break;
          }
          case "DataView": {
            const view = thing;
            str = '["' + type + '",' + flatten(view.buffer);
            if (view.byteLength !== view.buffer.byteLength) {
              str += `,${view.byteOffset},${view.byteLength}`;
            }
            str += "]";
            break;
          }
          case "ArrayBuffer": {
            const arraybuffer = thing;
            const base64 = encode64(arraybuffer);
            str = `["ArrayBuffer","${base64}"]`;
            break;
          }
          case "Temporal.Duration":
          case "Temporal.Instant":
          case "Temporal.PlainDate":
          case "Temporal.PlainTime":
          case "Temporal.PlainDateTime":
          case "Temporal.PlainMonthDay":
          case "Temporal.PlainYearMonth":
          case "Temporal.ZonedDateTime":
            str = `["${type}",${stringify_string(thing.toString())}]`;
            break;
          default:
            if (!is_plain_object(thing)) {
              throw new DevalueError(`Cannot stringify arbitrary non-POJOs`, keys, thing, value);
            }
            if (enumerable_symbols(thing).length > 0) {
              throw new DevalueError(`Cannot stringify POJOs with symbolic keys`, keys, thing, value);
            }
            if (Object.getPrototypeOf(thing) === null) {
              str = '["null"';
              for (const key of Object.keys(thing)) {
                if (key === "__proto__") {
                  throw new DevalueError(
                    `Cannot stringify objects with __proto__ keys`,
                    keys,
                    thing,
                    value
                  );
                }
                keys.push(stringify_key(key));
                str += `,${stringify_string(key)},${flatten(thing[key])}`;
                keys.pop();
              }
              str += "]";
            } else {
              str = "{";
              let started = false;
              for (const key of Object.keys(thing)) {
                if (key === "__proto__") {
                  throw new DevalueError(
                    `Cannot stringify objects with __proto__ keys`,
                    keys,
                    thing,
                    value
                  );
                }
                if (started) str += ",";
                started = true;
                keys.push(stringify_key(key));
                str += `${stringify_string(key)}:${flatten(thing[key])}`;
                keys.pop();
              }
              str += "}";
            }
        }
      }
      stringified[index2] = str;
      return index2;
    }
    const index = flatten(value);
    if (index < 0) return `${index}`;
    return stringified;
  }
  function stringify_primitive(thing) {
    const type = typeof thing;
    if (type === "string") return stringify_string(thing);
    if (thing === void 0) return UNDEFINED.toString();
    if (thing === 0 && 1 / thing < 0) return NEGATIVE_ZERO.toString();
    if (type === "bigint") return `["BigInt","${thing}"]`;
    return String(thing);
  }
  function isActionError(error2) {
    return typeof error2 === "object" && error2 != null && "type" in error2 && error2.type === "AstroActionError";
  }
  function isInputError(error2) {
    return typeof error2 === "object" && error2 != null && "type" in error2 && error2.type === "AstroActionInputError" && "issues" in error2 && Array.isArray(error2.issues);
  }
  function deserializeActionResult(res) {
    var _a29;
    if (res.type === "error") {
      let json;
      try {
        json = JSON.parse(res.body);
      } catch (e2) {
        return {
          data: void 0,
          error: new ActionError({
            message: res.body,
            code: "INTERNAL_SERVER_ERROR"
          })
        };
      }
      if ((_a29 = Object.assign(__vite_import_meta_env__$1, { _: "/home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/node_modules/.bin/astro" })) == null ? void 0 : _a29.PROD) {
        return { error: ActionError.fromJson(json), data: void 0 };
      } else {
        const error2 = ActionError.fromJson(json);
        error2.stack = actionResultErrorStack.get();
        return {
          error: error2,
          data: void 0
        };
      }
    }
    if (res.type === "empty") {
      return { data: void 0, error: void 0 };
    }
    return {
      data: parse$1(res.body, {
        URL: (href) => new URL(href)
      }),
      error: void 0
    };
  }
  function getActionQueryString(name) {
    const searchParams = new URLSearchParams({ [ACTION_QUERY_PARAMS.actionName]: name });
    return `?${searchParams.toString()}`;
  }
  function $constructor(name, initializer2, params) {
    var _a29;
    function init(inst, def) {
      if (!inst._zod) {
        Object.defineProperty(inst, "_zod", {
          value: {
            def,
            constr: _,
            traits: /* @__PURE__ */ new Set()
          },
          enumerable: false
        });
      }
      if (inst._zod.traits.has(name)) {
        return;
      }
      inst._zod.traits.add(name);
      initializer2(inst, def);
      const proto = _.prototype;
      const keys = Object.keys(proto);
      for (let i2 = 0; i2 < keys.length; i2++) {
        const k = keys[i2];
        if (!(k in inst)) {
          inst[k] = proto[k].bind(inst);
        }
      }
    }
    const Parent = (_a29 = params == null ? void 0 : params.Parent) != null ? _a29 : Object;
    class Definition extends Parent {
    }
    Object.defineProperty(Definition, "name", { value: name });
    function _(def) {
      var _a30;
      var _a210;
      const inst = (params == null ? void 0 : params.Parent) ? new Definition() : this;
      init(inst, def);
      (_a30 = (_a210 = inst._zod).deferred) != null ? _a30 : _a210.deferred = [];
      for (const fn of inst._zod.deferred) {
        fn();
      }
      return inst;
    }
    Object.defineProperty(_, "init", { value: init });
    Object.defineProperty(_, Symbol.hasInstance, {
      value: (inst) => {
        var _a30, _b13;
        if ((params == null ? void 0 : params.Parent) && inst instanceof params.Parent)
          return true;
        return (_b13 = (_a30 = inst == null ? void 0 : inst._zod) == null ? void 0 : _a30.traits) == null ? void 0 : _b13.has(name);
      }
    });
    Object.defineProperty(_, "name", { value: name });
    return _;
  }
  function config(newConfig) {
    return globalConfig;
  }
  function getEnumValues(entries) {
    const numericValues = Object.values(entries).filter((v) => typeof v === "number");
    const values = Object.entries(entries).filter(([k, _]) => numericValues.indexOf(+k) === -1).map(([_, v]) => v);
    return values;
  }
  function jsonStringifyReplacer(_, value) {
    if (typeof value === "bigint")
      return value.toString();
    return value;
  }
  function nullish(input) {
    return input === null || input === void 0;
  }
  function cleanRegex(source) {
    const start = source.startsWith("^") ? 1 : 0;
    const end = source.endsWith("$") ? source.length - 1 : source.length;
    return source.slice(start, end);
  }
  function defineLazy(object, key, getter) {
    let value = void 0;
    Object.defineProperty(object, key, {
      get() {
        if (value === EVALUATING) {
          return void 0;
        }
        if (value === void 0) {
          value = EVALUATING;
          value = getter();
        }
        return value;
      },
      set(v) {
        Object.defineProperty(object, key, {
          value: v
          // configurable: true,
        });
      },
      configurable: true
    });
  }
  function mergeDefs(...defs) {
    const mergedDescriptors = {};
    for (const def of defs) {
      const descriptors = Object.getOwnPropertyDescriptors(def);
      Object.assign(mergedDescriptors, descriptors);
    }
    return Object.defineProperties({}, mergedDescriptors);
  }
  function isObject(data) {
    return typeof data === "object" && data !== null && !Array.isArray(data);
  }
  function isPlainObject(o2) {
    if (isObject(o2) === false)
      return false;
    const ctor = o2.constructor;
    if (ctor === void 0)
      return true;
    if (typeof ctor !== "function")
      return true;
    const prot = ctor.prototype;
    if (isObject(prot) === false)
      return false;
    if (Object.prototype.hasOwnProperty.call(prot, "isPrototypeOf") === false) {
      return false;
    }
    return true;
  }
  function shallowClone(o2) {
    if (isPlainObject(o2))
      return __spreadValues({}, o2);
    if (Array.isArray(o2))
      return [...o2];
    if (o2 instanceof Map)
      return new Map(o2);
    if (o2 instanceof Set)
      return new Set(o2);
    return o2;
  }
  function escapeRegex(str) {
    return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  }
  function clone(inst, def, params) {
    const cl = new inst._zod.constr(def != null ? def : inst._zod.def);
    if (!def || (params == null ? void 0 : params.parent))
      cl._zod.parent = inst;
    return cl;
  }
  function normalizeParams(_params2) {
    const params = _params2;
    if (!params)
      return {};
    if (typeof params === "string")
      return { error: () => params };
    if ((params == null ? void 0 : params.message) !== void 0) {
      if ((params == null ? void 0 : params.error) !== void 0)
        throw new Error("Cannot specify both `message` and `error` params");
      params.error = params.message;
    }
    delete params.message;
    if (typeof params.error === "string")
      return __spreadProps(__spreadValues({}, params), { error: () => params.error });
    return params;
  }
  function aborted(x, startIndex = 0) {
    var _a29;
    if (x.aborted === true)
      return true;
    for (let i2 = startIndex; i2 < x.issues.length; i2++) {
      if (((_a29 = x.issues[i2]) == null ? void 0 : _a29.continue) !== true) {
        return true;
      }
    }
    return false;
  }
  function explicitlyAborted(x, startIndex = 0) {
    var _a29;
    if (x.aborted === true)
      return true;
    for (let i2 = startIndex; i2 < x.issues.length; i2++) {
      if (((_a29 = x.issues[i2]) == null ? void 0 : _a29.continue) === false) {
        return true;
      }
    }
    return false;
  }
  function prefixIssues(path, issues) {
    return issues.map((iss) => {
      var _a30;
      var _a29;
      (_a30 = (_a29 = iss).path) != null ? _a30 : _a29.path = [];
      iss.path.unshift(path);
      return iss;
    });
  }
  function unwrapMessage(message) {
    return typeof message === "string" ? message : message == null ? void 0 : message.message;
  }
  function finalizeIssue(iss, ctx, config2) {
    var _a29, _b13, _c, _d, _e, _f, _g, _h, _i, _j, _l;
    const message = iss.message ? iss.message : (_j = (_i = (_g = (_e = unwrapMessage((_c = (_b13 = (_a29 = iss.inst) == null ? void 0 : _a29._zod.def) == null ? void 0 : _b13.error) == null ? void 0 : _c.call(_b13, iss))) != null ? _e : unwrapMessage((_d = ctx == null ? void 0 : ctx.error) == null ? void 0 : _d.call(ctx, iss))) != null ? _g : unwrapMessage((_f = config2.customError) == null ? void 0 : _f.call(config2, iss))) != null ? _i : unwrapMessage((_h = config2.localeError) == null ? void 0 : _h.call(config2, iss))) != null ? _j : "Invalid input";
    const _k = iss, { inst: _inst, continue: _continue, input: _input } = _k, rest = __objRest(_k, ["inst", "continue", "input"]);
    (_l = rest.path) != null ? _l : rest.path = [];
    rest.message = message;
    if (ctx == null ? void 0 : ctx.reportInput) {
      rest.input = _input;
    }
    return rest;
  }
  function getLengthableOrigin(input) {
    if (Array.isArray(input))
      return "array";
    if (typeof input === "string")
      return "string";
    return "unknown";
  }
  function issue(...args) {
    const [iss, input, inst] = args;
    if (typeof iss === "string") {
      return {
        message: iss,
        code: "custom",
        input,
        inst
      };
    }
    return __spreadValues({}, iss);
  }
  function flattenError(error2, mapper = (issue2) => issue2.message) {
    const fieldErrors = {};
    const formErrors = [];
    for (const sub of error2.issues) {
      if (sub.path.length > 0) {
        fieldErrors[sub.path[0]] = fieldErrors[sub.path[0]] || [];
        fieldErrors[sub.path[0]].push(mapper(sub));
      } else {
        formErrors.push(mapper(sub));
      }
    }
    return { formErrors, fieldErrors };
  }
  function formatError(error2, mapper = (issue2) => issue2.message) {
    const fieldErrors = { _errors: [] };
    const processError = (error3, path = []) => {
      for (const issue2 of error3.issues) {
        if (issue2.code === "invalid_union" && issue2.errors.length) {
          issue2.errors.map((issues) => processError({ issues }, [...path, ...issue2.path]));
        } else if (issue2.code === "invalid_key") {
          processError({ issues: issue2.issues }, [...path, ...issue2.path]);
        } else if (issue2.code === "invalid_element") {
          processError({ issues: issue2.issues }, [...path, ...issue2.path]);
        } else {
          const fullpath = [...path, ...issue2.path];
          if (fullpath.length === 0) {
            fieldErrors._errors.push(mapper(issue2));
          } else {
            let curr = fieldErrors;
            let i2 = 0;
            while (i2 < fullpath.length) {
              const el = fullpath[i2];
              const terminal = i2 === fullpath.length - 1;
              if (!terminal) {
                curr[el] = curr[el] || { _errors: [] };
              } else {
                curr[el] = curr[el] || { _errors: [] };
                curr[el]._errors.push(mapper(issue2));
              }
              curr = curr[el];
              i2++;
            }
          }
        }
      }
    };
    processError(error2);
    return fieldErrors;
  }
  function handleArrayResult(result, final, index) {
    if (result.issues.length) {
      final.issues.push(...prefixIssues(index, result.issues));
    }
    final.value[index] = result.value;
  }
  function handleUnionResults(results, final, inst, ctx) {
    for (const result of results) {
      if (result.issues.length === 0) {
        final.value = result.value;
        return final;
      }
    }
    const nonaborted = results.filter((r2) => !aborted(r2));
    if (nonaborted.length === 1) {
      final.value = nonaborted[0].value;
      return nonaborted[0];
    }
    final.issues.push({
      code: "invalid_union",
      input: final.value,
      inst,
      errors: results.map((result) => result.issues.map((iss) => finalizeIssue(iss, ctx, config())))
    });
    return final;
  }
  function mergeValues(a2, b) {
    if (a2 === b) {
      return { valid: true, data: a2 };
    }
    if (a2 instanceof Date && b instanceof Date && +a2 === +b) {
      return { valid: true, data: a2 };
    }
    if (isPlainObject(a2) && isPlainObject(b)) {
      const bKeys = Object.keys(b);
      const sharedKeys = Object.keys(a2).filter((key) => bKeys.indexOf(key) !== -1);
      const newObj = __spreadValues(__spreadValues({}, a2), b);
      for (const key of sharedKeys) {
        const sharedValue = mergeValues(a2[key], b[key]);
        if (!sharedValue.valid) {
          return {
            valid: false,
            mergeErrorPath: [key, ...sharedValue.mergeErrorPath]
          };
        }
        newObj[key] = sharedValue.data;
      }
      return { valid: true, data: newObj };
    }
    if (Array.isArray(a2) && Array.isArray(b)) {
      if (a2.length !== b.length) {
        return { valid: false, mergeErrorPath: [] };
      }
      const newArray = [];
      for (let index = 0; index < a2.length; index++) {
        const itemA = a2[index];
        const itemB = b[index];
        const sharedValue = mergeValues(itemA, itemB);
        if (!sharedValue.valid) {
          return {
            valid: false,
            mergeErrorPath: [index, ...sharedValue.mergeErrorPath]
          };
        }
        newArray.push(sharedValue.data);
      }
      return { valid: true, data: newArray };
    }
    return { valid: false, mergeErrorPath: [] };
  }
  function handleIntersectionResults(result, left, right) {
    const unrecKeys = /* @__PURE__ */ new Map();
    let unrecIssue;
    for (const iss of left.issues) {
      if (iss.code === "unrecognized_keys") {
        unrecIssue != null ? unrecIssue : unrecIssue = iss;
        for (const k of iss.keys) {
          if (!unrecKeys.has(k))
            unrecKeys.set(k, {});
          unrecKeys.get(k).l = true;
        }
      } else {
        result.issues.push(iss);
      }
    }
    for (const iss of right.issues) {
      if (iss.code === "unrecognized_keys") {
        for (const k of iss.keys) {
          if (!unrecKeys.has(k))
            unrecKeys.set(k, {});
          unrecKeys.get(k).r = true;
        }
      } else {
        result.issues.push(iss);
      }
    }
    const bothKeys = [...unrecKeys].filter(([, f]) => f.l && f.r).map(([k]) => k);
    if (bothKeys.length && unrecIssue) {
      result.issues.push(__spreadProps(__spreadValues({}, unrecIssue), { keys: bothKeys }));
    }
    if (aborted(result))
      return result;
    const merged = mergeValues(left.value, right.value);
    if (!merged.valid) {
      throw new Error(`Unmergable intersection. Error path: ${JSON.stringify(merged.mergeErrorPath)}`);
    }
    result.value = merged.data;
    return result;
  }
  function handleOptionalResult(result, input) {
    if (input === void 0 && (result.issues.length || result.fallback)) {
      return { issues: [], value: void 0 };
    }
    return result;
  }
  function handleDefaultResult(payload, def) {
    if (payload.value === void 0) {
      payload.value = def.defaultValue;
    }
    return payload;
  }
  function handleNonOptionalResult(payload, inst) {
    if (!payload.issues.length && payload.value === void 0) {
      payload.issues.push({
        code: "invalid_type",
        expected: "nonoptional",
        input: payload.value,
        inst
      });
    }
    return payload;
  }
  function handlePipeResult(left, next, ctx) {
    if (left.issues.length) {
      left.aborted = true;
      return left;
    }
    return next._zod.run({ value: left.value, issues: left.issues, fallback: left.fallback }, ctx);
  }
  function handleReadonlyResult(payload) {
    payload.value = Object.freeze(payload.value);
    return payload;
  }
  function handleRefineResult(result, payload, input, inst) {
    var _a29;
    if (!result) {
      const _iss = {
        code: "custom",
        input,
        inst,
        // incorporates params.error into issue reporting
        path: [...(_a29 = inst._zod.def.path) != null ? _a29 : []],
        // incorporates params.error into issue reporting
        continue: !inst._zod.def.abort
        // params: inst._zod.def.params,
      };
      if (inst._zod.def.params)
        _iss.params = inst._zod.def.params;
      payload.issues.push(issue(_iss));
    }
  }
  function registry() {
    return new $ZodRegistry();
  }
  // @__NO_SIDE_EFFECTS__
  function _maxLength(maximum, params) {
    const ch = new $ZodCheckMaxLength(__spreadProps(__spreadValues({
      check: "max_length"
    }, normalizeParams(params)), {
      maximum
    }));
    return ch;
  }
  // @__NO_SIDE_EFFECTS__
  function _minLength(minimum, params) {
    return new $ZodCheckMinLength(__spreadProps(__spreadValues({
      check: "min_length"
    }, normalizeParams(params)), {
      minimum
    }));
  }
  // @__NO_SIDE_EFFECTS__
  function _length(length, params) {
    return new $ZodCheckLengthEquals(__spreadProps(__spreadValues({
      check: "length_equals"
    }, normalizeParams(params)), {
      length
    }));
  }
  // @__NO_SIDE_EFFECTS__
  function _overwrite(tx) {
    return new $ZodCheckOverwrite({
      check: "overwrite",
      tx
    });
  }
  // @__NO_SIDE_EFFECTS__
  function _array(Class, element, params) {
    return new Class(__spreadValues({
      type: "array",
      element
    }, normalizeParams(params)));
  }
  // @__NO_SIDE_EFFECTS__
  function _custom(Class, fn, _params2) {
    var _a29;
    const norm = normalizeParams(_params2);
    (_a29 = norm.abort) != null ? _a29 : norm.abort = true;
    const schema = new Class(__spreadValues({
      type: "custom",
      check: "custom",
      fn
    }, norm));
    return schema;
  }
  // @__NO_SIDE_EFFECTS__
  function _refine(Class, fn, _params2) {
    const schema = new Class(__spreadValues({
      type: "custom",
      check: "custom",
      fn
    }, normalizeParams(_params2)));
    return schema;
  }
  // @__NO_SIDE_EFFECTS__
  function _superRefine(fn, params) {
    const ch = /* @__PURE__ */ _check((payload) => {
      payload.addIssue = (issue$1) => {
        var _a29, _b13, _c, _d;
        if (typeof issue$1 === "string") {
          payload.issues.push(issue(issue$1, payload.value, ch._zod.def));
        } else {
          const _issue = issue$1;
          if (_issue.fatal)
            _issue.continue = false;
          (_a29 = _issue.code) != null ? _a29 : _issue.code = "custom";
          (_b13 = _issue.input) != null ? _b13 : _issue.input = payload.value;
          (_c = _issue.inst) != null ? _c : _issue.inst = ch;
          (_d = _issue.continue) != null ? _d : _issue.continue = !ch._zod.def.abort;
          payload.issues.push(issue(_issue));
        }
      };
      return fn(payload.value, payload);
    }, params);
    return ch;
  }
  // @__NO_SIDE_EFFECTS__
  function _check(fn, params) {
    const ch = new $ZodCheck(__spreadValues({
      check: "custom"
    }, normalizeParams(params)));
    ch._zod.check = fn;
    return ch;
  }
  function initializeContext(params) {
    var _a29, _b13, _c, _d, _e, _f, _g, _h, _i;
    let target = (_a29 = params == null ? void 0 : params.target) != null ? _a29 : "draft-2020-12";
    if (target === "draft-4")
      target = "draft-04";
    if (target === "draft-7")
      target = "draft-07";
    return {
      processors: (_b13 = params.processors) != null ? _b13 : {},
      metadataRegistry: (_c = params == null ? void 0 : params.metadata) != null ? _c : globalRegistry,
      target,
      unrepresentable: (_d = params == null ? void 0 : params.unrepresentable) != null ? _d : "throw",
      override: (_e = params == null ? void 0 : params.override) != null ? _e : (() => {
      }),
      io: (_f = params == null ? void 0 : params.io) != null ? _f : "output",
      counter: 0,
      seen: /* @__PURE__ */ new Map(),
      cycles: (_g = params == null ? void 0 : params.cycles) != null ? _g : "ref",
      reused: (_h = params == null ? void 0 : params.reused) != null ? _h : "inline",
      external: (_i = params == null ? void 0 : params.external) != null ? _i : void 0
    };
  }
  function process$1(schema, ctx, _params2 = { path: [], schemaPath: [] }) {
    var _a30, _b13, _c;
    var _a29;
    const def = schema._zod.def;
    const seen = ctx.seen.get(schema);
    if (seen) {
      seen.count++;
      const isCycle = _params2.schemaPath.includes(schema);
      if (isCycle) {
        seen.cycle = _params2.path;
      }
      return seen.schema;
    }
    const result = { schema: {}, count: 1, cycle: void 0, path: _params2.path };
    ctx.seen.set(schema, result);
    const overrideSchema = (_b13 = (_a30 = schema._zod).toJSONSchema) == null ? void 0 : _b13.call(_a30);
    if (overrideSchema) {
      result.schema = overrideSchema;
    } else {
      const params = __spreadProps(__spreadValues({}, _params2), {
        schemaPath: [..._params2.schemaPath, schema],
        path: _params2.path
      });
      if (schema._zod.processJSONSchema) {
        schema._zod.processJSONSchema(ctx, result.schema, params);
      } else {
        const _json = result.schema;
        const processor = ctx.processors[def.type];
        if (!processor) {
          throw new Error(`[toJSONSchema]: Non-representable type encountered: ${def.type}`);
        }
        processor(schema, ctx, _json, params);
      }
      const parent = schema._zod.parent;
      if (parent) {
        if (!result.ref)
          result.ref = parent;
        process$1(parent, ctx, params);
        ctx.seen.get(parent).isParent = true;
      }
    }
    const meta = ctx.metadataRegistry.get(schema);
    if (meta)
      Object.assign(result.schema, meta);
    if (ctx.io === "input" && isTransforming(schema)) {
      delete result.schema.examples;
      delete result.schema.default;
    }
    if (ctx.io === "input" && "_prefault" in result.schema)
      (_c = (_a29 = result.schema).default) != null ? _c : _a29.default = result.schema._prefault;
    delete result.schema._prefault;
    const _result2 = ctx.seen.get(schema);
    return _result2.schema;
  }
  function extractDefs(ctx, schema) {
    var _a29, _b13, _c, _d;
    const root = ctx.seen.get(schema);
    if (!root)
      throw new Error("Unprocessed schema. This is a bug in Zod.");
    const idToSchema = /* @__PURE__ */ new Map();
    for (const entry of ctx.seen.entries()) {
      const id = (_a29 = ctx.metadataRegistry.get(entry[0])) == null ? void 0 : _a29.id;
      if (id) {
        const existing = idToSchema.get(id);
        if (existing && existing !== entry[0]) {
          throw new Error(`Duplicate schema id "${id}" detected during JSON Schema conversion. Two different schemas cannot share the same id when converted together.`);
        }
        idToSchema.set(id, entry[0]);
      }
    }
    const makeURI = (entry) => {
      var _a30, _b14, _c2, _d2, _e;
      const defsSegment = ctx.target === "draft-2020-12" ? "$defs" : "definitions";
      if (ctx.external) {
        const externalId = (_a30 = ctx.external.registry.get(entry[0])) == null ? void 0 : _a30.id;
        const uriGenerator = (_b14 = ctx.external.uri) != null ? _b14 : ((id2) => id2);
        if (externalId) {
          return { ref: uriGenerator(externalId) };
        }
        const id = (_d2 = (_c2 = entry[1].defId) != null ? _c2 : entry[1].schema.id) != null ? _d2 : `schema${ctx.counter++}`;
        entry[1].defId = id;
        return { defId: id, ref: `${uriGenerator("__shared")}#/${defsSegment}/${id}` };
      }
      if (entry[1] === root) {
        return { ref: "#" };
      }
      const uriPrefix = `#`;
      const defUriPrefix = `${uriPrefix}/${defsSegment}/`;
      const defId = (_e = entry[1].schema.id) != null ? _e : `__schema${ctx.counter++}`;
      return { defId, ref: defUriPrefix + defId };
    };
    const extractToDef = (entry) => {
      if (entry[1].schema.$ref) {
        return;
      }
      const seen = entry[1];
      const { ref, defId } = makeURI(entry);
      seen.def = __spreadValues({}, seen.schema);
      if (defId)
        seen.defId = defId;
      const schema2 = seen.schema;
      for (const key in schema2) {
        delete schema2[key];
      }
      schema2.$ref = ref;
    };
    if (ctx.cycles === "throw") {
      for (const entry of ctx.seen.entries()) {
        const seen = entry[1];
        if (seen.cycle) {
          throw new Error(`Cycle detected: #/${(_b13 = seen.cycle) == null ? void 0 : _b13.join("/")}/<root>

Set the \`cycles\` parameter to \`"ref"\` to resolve cyclical schemas with defs.`);
        }
      }
    }
    for (const entry of ctx.seen.entries()) {
      const seen = entry[1];
      if (schema === entry[0]) {
        extractToDef(entry);
        continue;
      }
      if (ctx.external) {
        const ext = (_c = ctx.external.registry.get(entry[0])) == null ? void 0 : _c.id;
        if (schema !== entry[0] && ext) {
          extractToDef(entry);
          continue;
        }
      }
      const id = (_d = ctx.metadataRegistry.get(entry[0])) == null ? void 0 : _d.id;
      if (id) {
        extractToDef(entry);
        continue;
      }
      if (seen.cycle) {
        extractToDef(entry);
        continue;
      }
      if (seen.count > 1) {
        if (ctx.reused === "ref") {
          extractToDef(entry);
          continue;
        }
      }
    }
  }
  function finalize(ctx, schema) {
    var _a29, _b13, _c, _d, _e, _f;
    const root = ctx.seen.get(schema);
    if (!root)
      throw new Error("Unprocessed schema. This is a bug in Zod.");
    const flattenRef = (zodSchema) => {
      var _a30, _b14, _c2;
      const seen = ctx.seen.get(zodSchema);
      if (seen.ref === null)
        return;
      const schema2 = (_a30 = seen.def) != null ? _a30 : seen.schema;
      const _cached = __spreadValues({}, schema2);
      const ref = seen.ref;
      seen.ref = null;
      if (ref) {
        flattenRef(ref);
        const refSeen = ctx.seen.get(ref);
        const refSchema = refSeen.schema;
        if (refSchema.$ref && (ctx.target === "draft-07" || ctx.target === "draft-04" || ctx.target === "openapi-3.0")) {
          schema2.allOf = (_b14 = schema2.allOf) != null ? _b14 : [];
          schema2.allOf.push(refSchema);
        } else {
          Object.assign(schema2, refSchema);
        }
        Object.assign(schema2, _cached);
        const isParentRef = zodSchema._zod.parent === ref;
        if (isParentRef) {
          for (const key in schema2) {
            if (key === "$ref" || key === "allOf")
              continue;
            if (!(key in _cached)) {
              delete schema2[key];
            }
          }
        }
        if (refSchema.$ref && refSeen.def) {
          for (const key in schema2) {
            if (key === "$ref" || key === "allOf")
              continue;
            if (key in refSeen.def && JSON.stringify(schema2[key]) === JSON.stringify(refSeen.def[key])) {
              delete schema2[key];
            }
          }
        }
      }
      const parent = zodSchema._zod.parent;
      if (parent && parent !== ref) {
        flattenRef(parent);
        const parentSeen = ctx.seen.get(parent);
        if (parentSeen == null ? void 0 : parentSeen.schema.$ref) {
          schema2.$ref = parentSeen.schema.$ref;
          if (parentSeen.def) {
            for (const key in schema2) {
              if (key === "$ref" || key === "allOf")
                continue;
              if (key in parentSeen.def && JSON.stringify(schema2[key]) === JSON.stringify(parentSeen.def[key])) {
                delete schema2[key];
              }
            }
          }
        }
      }
      ctx.override({
        zodSchema,
        jsonSchema: schema2,
        path: (_c2 = seen.path) != null ? _c2 : []
      });
    };
    for (const entry of [...ctx.seen.entries()].reverse()) {
      flattenRef(entry[0]);
    }
    const result = {};
    if (ctx.target === "draft-2020-12") {
      result.$schema = "https://json-schema.org/draft/2020-12/schema";
    } else if (ctx.target === "draft-07") {
      result.$schema = "http://json-schema.org/draft-07/schema#";
    } else if (ctx.target === "draft-04") {
      result.$schema = "http://json-schema.org/draft-04/schema#";
    } else if (ctx.target === "openapi-3.0") ;
    else ;
    if ((_a29 = ctx.external) == null ? void 0 : _a29.uri) {
      const id = (_b13 = ctx.external.registry.get(schema)) == null ? void 0 : _b13.id;
      if (!id)
        throw new Error("Schema is missing an `id` property");
      result.$id = ctx.external.uri(id);
    }
    Object.assign(result, (_c = root.def) != null ? _c : root.schema);
    const rootMetaId = (_d = ctx.metadataRegistry.get(schema)) == null ? void 0 : _d.id;
    if (rootMetaId !== void 0 && result.id === rootMetaId)
      delete result.id;
    const defs = (_f = (_e = ctx.external) == null ? void 0 : _e.defs) != null ? _f : {};
    for (const entry of ctx.seen.entries()) {
      const seen = entry[1];
      if (seen.def && seen.defId) {
        if (seen.def.id === seen.defId)
          delete seen.def.id;
        defs[seen.defId] = seen.def;
      }
    }
    if (ctx.external) ;
    else {
      if (Object.keys(defs).length > 0) {
        if (ctx.target === "draft-2020-12") {
          result.$defs = defs;
        } else {
          result.definitions = defs;
        }
      }
    }
    try {
      const finalized = JSON.parse(JSON.stringify(result));
      Object.defineProperty(finalized, "~standard", {
        value: __spreadProps(__spreadValues({}, schema["~standard"]), {
          jsonSchema: {
            input: createStandardJSONSchemaMethod(schema, "input", ctx.processors),
            output: createStandardJSONSchemaMethod(schema, "output", ctx.processors)
          }
        }),
        enumerable: false,
        writable: false
      });
      return finalized;
    } catch (_err) {
      throw new Error("Error converting schema to JSON.");
    }
  }
  function isTransforming(_schema, _ctx) {
    const ctx = _ctx != null ? _ctx : { seen: /* @__PURE__ */ new Set() };
    if (ctx.seen.has(_schema))
      return false;
    ctx.seen.add(_schema);
    const def = _schema._zod.def;
    if (def.type === "transform")
      return true;
    if (def.type === "array")
      return isTransforming(def.element, ctx);
    if (def.type === "set")
      return isTransforming(def.valueType, ctx);
    if (def.type === "lazy")
      return isTransforming(def.getter(), ctx);
    if (def.type === "promise" || def.type === "optional" || def.type === "nonoptional" || def.type === "nullable" || def.type === "readonly" || def.type === "default" || def.type === "prefault") {
      return isTransforming(def.innerType, ctx);
    }
    if (def.type === "intersection") {
      return isTransforming(def.left, ctx) || isTransforming(def.right, ctx);
    }
    if (def.type === "record" || def.type === "map") {
      return isTransforming(def.keyType, ctx) || isTransforming(def.valueType, ctx);
    }
    if (def.type === "pipe") {
      if (_schema._zod.traits.has("$ZodCodec"))
        return true;
      return isTransforming(def.in, ctx) || isTransforming(def.out, ctx);
    }
    if (def.type === "object") {
      for (const key in def.shape) {
        if (isTransforming(def.shape[key], ctx))
          return true;
      }
      return false;
    }
    if (def.type === "union") {
      for (const option of def.options) {
        if (isTransforming(option, ctx))
          return true;
      }
      return false;
    }
    if (def.type === "tuple") {
      for (const item of def.items) {
        if (isTransforming(item, ctx))
          return true;
      }
      if (def.rest && isTransforming(def.rest, ctx))
        return true;
      return false;
    }
    return false;
  }
  function shouldAppendForwardSlash(trailingSlash, buildFormat) {
    switch (trailingSlash) {
      case "always":
        return true;
      case "never":
        return false;
      case "ignore": {
        switch (buildFormat) {
          case "directory":
            return true;
          case "preserve":
          case "file":
            return false;
        }
      }
    }
  }
  function normalizeLF(code) {
    return code.replace(/\r\n|\r(?!\n)|\n/g, "\n");
  }
  function codeFrame(src, loc) {
    if (!loc || loc.line === void 0 || loc.column === void 0) {
      return "";
    }
    const lines = normalizeLF(src).split("\n").map((ln) => ln.replace(/\t/g, "  "));
    const visibleLines = [];
    for (let n2 = -2; n2 <= 2; n2++) {
      if (lines[loc.line + n2]) visibleLines.push(loc.line + n2);
    }
    let gutterWidth = 0;
    for (const lineNo of visibleLines) {
      let w = `> ${lineNo}`;
      if (w.length > gutterWidth) gutterWidth = w.length;
    }
    let output = "";
    for (const lineNo of visibleLines) {
      const isFocusedLine = lineNo === loc.line - 1;
      output += isFocusedLine ? "> " : "  ";
      output += `${lineNo + 1} | ${lines[lineNo]}
`;
      if (isFocusedLine)
        output += `${Array.from({ length: gutterWidth }).join(" ")}  | ${Array.from({
          length: loc.column
        }).join(" ")}^
`;
    }
    return output;
  }
  function readBodyWithLimit(request, limit) {
    return __async(this, null, function* () {
      const contentLengthHeader = request.headers.get("content-length");
      if (contentLengthHeader) {
        const contentLength = Number.parseInt(contentLengthHeader, 10);
        if (Number.isFinite(contentLength) && contentLength > limit) {
          throw new BodySizeLimitError(limit);
        }
      }
      if (!request.body) return new Uint8Array();
      const reader = request.body.getReader();
      const chunks = [];
      let received = 0;
      while (true) {
        const { done, value } = yield reader.read();
        if (done) break;
        if (value) {
          received += value.byteLength;
          if (received > limit) {
            throw new BodySizeLimitError(limit);
          }
          chunks.push(value);
        }
      }
      const buffer2 = new Uint8Array(received);
      let offset = 0;
      for (const chunk of chunks) {
        buffer2.set(chunk, offset);
        offset += chunk.byteLength;
      }
      return buffer2;
    });
  }
  function getActionContext(context) {
    const callerInfo = getCallerInfo(context);
    const actionResultAlreadySet = Boolean(context.locals._actionPayload);
    let action = void 0;
    if (callerInfo && context.request.method === "POST" && !actionResultAlreadySet) {
      action = {
        calledFrom: callerInfo.from,
        name: callerInfo.name,
        handler: () => __async(null, null, function* () {
          const pipeline = Reflect.get(context, pipelineSymbol);
          const callerInfoName = shouldAppendForwardSlash(
            pipeline.manifest.trailingSlash,
            pipeline.manifest.buildFormat
          ) ? removeTrailingForwardSlash(callerInfo.name) : callerInfo.name;
          let baseAction;
          try {
            baseAction = yield pipeline.getAction(callerInfoName);
          } catch (error2) {
            if (error2 instanceof Error && "name" in error2 && typeof error2.name === "string" && error2.name === ActionNotFoundError.name) {
              return { data: void 0, error: new ActionError({ code: "NOT_FOUND" }) };
            }
            throw error2;
          }
          const bodySizeLimit = pipeline.manifest.actionBodySizeLimit;
          let input;
          try {
            input = yield parseRequestBody(context.request, bodySizeLimit);
          } catch (e2) {
            if (e2 instanceof ActionError) {
              return { data: void 0, error: e2 };
            }
            if (e2 instanceof TypeError) {
              return { data: void 0, error: new ActionError({ code: "UNSUPPORTED_MEDIA_TYPE" }) };
            }
            throw e2;
          }
          const omitKeys = ["props", "getActionResult", "callAction", "redirect"];
          const actionAPIContext = Object.create(
            Object.getPrototypeOf(context),
            Object.fromEntries(
              Object.entries(Object.getOwnPropertyDescriptors(context)).filter(
                ([key]) => !omitKeys.includes(key)
              )
            )
          );
          Reflect.set(actionAPIContext, ACTION_API_CONTEXT_SYMBOL, true);
          const handler = baseAction.bind(actionAPIContext);
          return handler(input);
        })
      };
    }
    function setActionResult(actionName, actionResult) {
      context.locals._actionPayload = {
        actionResult,
        actionName
      };
    }
    return {
      action,
      setActionResult,
      serializeActionResult,
      deserializeActionResult
    };
  }
  function getCallerInfo(ctx) {
    if (ctx.routePattern === ACTION_RPC_ROUTE_PATTERN) {
      return { from: "rpc", name: ctx.url.pathname.replace(/^.*\/_actions\//, "") };
    }
    const queryParam = ctx.url.searchParams.get(ACTION_QUERY_PARAMS.actionName);
    if (queryParam) {
      return { from: "form", name: queryParam };
    }
    return void 0;
  }
  function parseRequestBody(request, bodySizeLimit) {
    return __async(this, null, function* () {
      const contentType = request.headers.get("content-type");
      const contentLengthHeader = request.headers.get("content-length");
      const contentLength = contentLengthHeader ? Number.parseInt(contentLengthHeader, 10) : void 0;
      const hasContentLength = typeof contentLength === "number" && Number.isFinite(contentLength);
      if (!contentType) return void 0;
      if (hasContentLength && contentLength > bodySizeLimit) {
        throw new ActionError({
          code: "CONTENT_TOO_LARGE",
          message: `Request body exceeds ${bodySizeLimit} bytes`
        });
      }
      try {
        if (hasContentType(contentType, formContentTypes)) {
          if (!hasContentLength) {
            const body = yield readBodyWithLimit(request.clone(), bodySizeLimit);
            const formRequest = new Request(request.url, {
              method: request.method,
              headers: request.headers,
              body: toArrayBuffer(body)
            });
            return yield formRequest.formData();
          }
          return yield request.clone().formData();
        }
        if (hasContentType(contentType, ["application/json"])) {
          if (contentLength === 0) return void 0;
          if (!hasContentLength) {
            const body = yield readBodyWithLimit(request.clone(), bodySizeLimit);
            if (body.byteLength === 0) return void 0;
            return JSON.parse(new TextDecoder().decode(body));
          }
          return yield request.clone().json();
        }
      } catch (e2) {
        if (e2 instanceof BodySizeLimitError) {
          throw new ActionError({
            code: "CONTENT_TOO_LARGE",
            message: `Request body exceeds ${bodySizeLimit} bytes`
          });
        }
        throw e2;
      }
      throw new TypeError("Unsupported content type");
    });
  }
  function hasContentType(contentType, expected) {
    const type = contentType.split(";")[0].toLowerCase();
    return expected.some((t2) => type === t2);
  }
  function serializeActionResult(res) {
    var _a29;
    if (res.error) {
      if ((_a29 = Object.assign(__vite_import_meta_env__, { _: "/home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/node_modules/.bin/astro" })) == null ? void 0 : _a29.DEV) {
        actionResultErrorStack.set(res.error.stack);
      }
      let body2;
      if (res.error instanceof ActionInputError) {
        body2 = {
          type: res.error.type,
          issues: res.error.issues,
          fields: res.error.fields
        };
      } else {
        body2 = __spreadProps(__spreadValues({}, res.error), {
          message: res.error.message
        });
      }
      return {
        type: "error",
        status: res.error.status,
        contentType: "application/json",
        body: JSON.stringify(body2)
      };
    }
    if (res.data === void 0) {
      return {
        type: "empty",
        status: 204
      };
    }
    let body;
    try {
      body = stringify$2(res.data, {
        // Add support for URL objects
        URL: (value) => value instanceof URL && value.href
      });
    } catch (e2) {
      let hint = ActionsReturnedInvalidDataError.hint;
      if (res.data instanceof Response) {
        hint = REDIRECT_STATUS_CODES.includes(res.data.status) ? "If you need to redirect when the action succeeds, trigger a redirect where the action is called. See the Actions guide for server and client redirect examples: https://docs.astro.build/en/guides/actions." : "If you need to return a Response object, try using a server endpoint instead. See https://docs.astro.build/en/guides/endpoints/#server-endpoints-api-routes";
      }
      throw new AstroError(__spreadProps(__spreadValues({}, ActionsReturnedInvalidDataError), {
        message: ActionsReturnedInvalidDataError.message(String(e2)),
        hint
      }));
    }
    return {
      type: "data",
      status: 200,
      contentType: "application/json+devalue",
      body
    };
  }
  function toArrayBuffer(buffer2) {
    const copy = new Uint8Array(buffer2.byteLength);
    copy.set(buffer2);
    return copy.buffer;
  }
  function hasActionPayload(locals) {
    return "_actionPayload" in locals;
  }
  function createGetActionResult(locals) {
    return (actionFn) => {
      if (!hasActionPayload(locals) || actionFn.toString() !== getActionQueryString(locals._actionPayload.actionName)) {
        return void 0;
      }
      return deserializeActionResult(locals._actionPayload.actionResult);
    };
  }
  function createCallAction(context) {
    return (baseAction, input) => {
      Reflect.set(context, ACTION_API_CONTEXT_SYMBOL, true);
      const action = baseAction.bind(context);
      return action(input);
    };
  }
  function requireDist() {
    if (hasRequiredDist) return dist;
    hasRequiredDist = 1;
    Object.defineProperty(dist, "__esModule", { value: true });
    dist.parseCookie = parseCookie;
    dist.parse = parseCookie;
    dist.stringifyCookie = stringifyCookie;
    dist.stringifySetCookie = stringifySetCookie;
    dist.serialize = stringifySetCookie;
    dist.parseSetCookie = parseSetCookie;
    dist.stringifySetCookie = stringifySetCookie;
    dist.serialize = stringifySetCookie;
    const cookieNameRegExp = /^[\u0021-\u003A\u003C\u003E-\u007E]+$/;
    const cookieValueRegExp = /^[\u0021-\u003A\u003C-\u007E]*$/;
    const domainValueRegExp = /^([.]?[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)([.][a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/i;
    const pathValueRegExp = /^[\u0020-\u003A\u003D-\u007E]*$/;
    const maxAgeRegExp = /^-?\d+$/;
    const __toString = Object.prototype.toString;
    const NullObject = /* @__PURE__ */ (() => {
      const C2 = function() {
      };
      C2.prototype = /* @__PURE__ */ Object.create(null);
      return C2;
    })();
    function parseCookie(str, options) {
      const obj = new NullObject();
      const len = str.length;
      if (len < 2)
        return obj;
      const dec = (options == null ? void 0 : options.decode) || decode2;
      let index = 0;
      do {
        const eqIdx = eqIndex(str, index, len);
        if (eqIdx === -1)
          break;
        const endIdx = endIndex(str, index, len);
        if (eqIdx > endIdx) {
          index = str.lastIndexOf(";", eqIdx - 1) + 1;
          continue;
        }
        const key = valueSlice(str, index, eqIdx);
        if (obj[key] === void 0) {
          obj[key] = dec(valueSlice(str, eqIdx + 1, endIdx));
        }
        index = endIdx + 1;
      } while (index < len);
      return obj;
    }
    function stringifyCookie(cookie, options) {
      const enc = (options == null ? void 0 : options.encode) || encodeURIComponent;
      const cookieStrings = [];
      for (const name of Object.keys(cookie)) {
        const val = cookie[name];
        if (val === void 0)
          continue;
        if (!cookieNameRegExp.test(name)) {
          throw new TypeError(`cookie name is invalid: ${name}`);
        }
        const value = enc(val);
        if (!cookieValueRegExp.test(value)) {
          throw new TypeError(`cookie val is invalid: ${val}`);
        }
        cookieStrings.push(`${name}=${value}`);
      }
      return cookieStrings.join("; ");
    }
    function stringifySetCookie(_name, _val, _opts) {
      const cookie = typeof _name === "object" ? _name : __spreadProps(__spreadValues({}, _opts), { name: _name, value: String(_val) });
      const options = typeof _val === "object" ? _val : _opts;
      const enc = (options == null ? void 0 : options.encode) || encodeURIComponent;
      if (!cookieNameRegExp.test(cookie.name)) {
        throw new TypeError(`argument name is invalid: ${cookie.name}`);
      }
      const value = cookie.value ? enc(cookie.value) : "";
      if (!cookieValueRegExp.test(value)) {
        throw new TypeError(`argument val is invalid: ${cookie.value}`);
      }
      let str = cookie.name + "=" + value;
      if (cookie.maxAge !== void 0) {
        if (!Number.isInteger(cookie.maxAge)) {
          throw new TypeError(`option maxAge is invalid: ${cookie.maxAge}`);
        }
        str += "; Max-Age=" + cookie.maxAge;
      }
      if (cookie.domain) {
        if (!domainValueRegExp.test(cookie.domain)) {
          throw new TypeError(`option domain is invalid: ${cookie.domain}`);
        }
        str += "; Domain=" + cookie.domain;
      }
      if (cookie.path) {
        if (!pathValueRegExp.test(cookie.path)) {
          throw new TypeError(`option path is invalid: ${cookie.path}`);
        }
        str += "; Path=" + cookie.path;
      }
      if (cookie.expires) {
        if (!isDate(cookie.expires) || !Number.isFinite(cookie.expires.valueOf())) {
          throw new TypeError(`option expires is invalid: ${cookie.expires}`);
        }
        str += "; Expires=" + cookie.expires.toUTCString();
      }
      if (cookie.httpOnly) {
        str += "; HttpOnly";
      }
      if (cookie.secure) {
        str += "; Secure";
      }
      if (cookie.partitioned) {
        str += "; Partitioned";
      }
      if (cookie.priority) {
        const priority = typeof cookie.priority === "string" ? cookie.priority.toLowerCase() : void 0;
        switch (priority) {
          case "low":
            str += "; Priority=Low";
            break;
          case "medium":
            str += "; Priority=Medium";
            break;
          case "high":
            str += "; Priority=High";
            break;
          default:
            throw new TypeError(`option priority is invalid: ${cookie.priority}`);
        }
      }
      if (cookie.sameSite) {
        const sameSite = typeof cookie.sameSite === "string" ? cookie.sameSite.toLowerCase() : cookie.sameSite;
        switch (sameSite) {
          case true:
          case "strict":
            str += "; SameSite=Strict";
            break;
          case "lax":
            str += "; SameSite=Lax";
            break;
          case "none":
            str += "; SameSite=None";
            break;
          default:
            throw new TypeError(`option sameSite is invalid: ${cookie.sameSite}`);
        }
      }
      return str;
    }
    function parseSetCookie(str, options) {
      const dec = (options == null ? void 0 : options.decode) || decode2;
      const len = str.length;
      const endIdx = endIndex(str, 0, len);
      const eqIdx = eqIndex(str, 0, endIdx);
      const setCookie = eqIdx === -1 ? { name: "", value: dec(valueSlice(str, 0, endIdx)) } : {
        name: valueSlice(str, 0, eqIdx),
        value: dec(valueSlice(str, eqIdx + 1, endIdx))
      };
      let index = endIdx + 1;
      while (index < len) {
        const endIdx2 = endIndex(str, index, len);
        const eqIdx2 = eqIndex(str, index, endIdx2);
        const attr = eqIdx2 === -1 ? valueSlice(str, index, endIdx2) : valueSlice(str, index, eqIdx2);
        const val = eqIdx2 === -1 ? void 0 : valueSlice(str, eqIdx2 + 1, endIdx2);
        switch (attr.toLowerCase()) {
          case "httponly":
            setCookie.httpOnly = true;
            break;
          case "secure":
            setCookie.secure = true;
            break;
          case "partitioned":
            setCookie.partitioned = true;
            break;
          case "domain":
            setCookie.domain = val;
            break;
          case "path":
            setCookie.path = val;
            break;
          case "max-age":
            if (val && maxAgeRegExp.test(val))
              setCookie.maxAge = Number(val);
            break;
          case "expires":
            if (!val)
              break;
            const date = new Date(val);
            if (Number.isFinite(date.valueOf()))
              setCookie.expires = date;
            break;
          case "priority":
            if (!val)
              break;
            const priority = val.toLowerCase();
            if (priority === "low" || priority === "medium" || priority === "high") {
              setCookie.priority = priority;
            }
            break;
          case "samesite":
            if (!val)
              break;
            const sameSite = val.toLowerCase();
            if (sameSite === "lax" || sameSite === "strict" || sameSite === "none") {
              setCookie.sameSite = sameSite;
            }
            break;
        }
        index = endIdx2 + 1;
      }
      return setCookie;
    }
    function endIndex(str, min, len) {
      const index = str.indexOf(";", min);
      return index === -1 ? len : index;
    }
    function eqIndex(str, min, max) {
      const index = str.indexOf("=", min);
      return index < max ? index : -1;
    }
    function valueSlice(str, min, max) {
      let start = min;
      let end = max;
      do {
        const code = str.charCodeAt(start);
        if (code !== 32 && code !== 9)
          break;
      } while (++start < end);
      while (end > start) {
        const code = str.charCodeAt(end - 1);
        if (code !== 32 && code !== 9)
          break;
        end--;
      }
      return str.slice(start, end);
    }
    function decode2(str) {
      if (str.indexOf("%") === -1)
        return str;
      try {
        return decodeURIComponent(str);
      } catch (e2) {
        return str;
      }
    }
    function isDate(val) {
      return __toString.call(val) === "[object Date]";
    }
    return dist;
  }
  function attachCookiesToResponse(response, cookies) {
    Reflect.set(response, astroCookiesSymbol, cookies);
  }
  function getCookiesFromResponse(response) {
    let cookies = Reflect.get(response, astroCookiesSymbol);
    if (cookies != null) {
      return cookies;
    } else {
      return void 0;
    }
  }
  function* getSetCookiesFromResponse(response) {
    const cookies = getCookiesFromResponse(response);
    if (!cookies) {
      return [];
    }
    for (const headerValue of cookies.consume()) {
      yield headerValue;
    }
    return [];
  }
  function defineMiddleware(fn) {
    return fn;
  }
  function createOriginCheckMiddleware() {
    return defineMiddleware((context, next) => {
      const { request, url, isPrerendered } = context;
      if (isPrerendered) {
        return next();
      }
      if (SAFE_METHODS.includes(request.method)) {
        return next();
      }
      const isSameOrigin = request.headers.get("origin") === url.origin;
      const hasContentType2 = request.headers.has("content-type");
      if (hasContentType2) {
        const formLikeHeader = hasFormLikeHeader(request.headers.get("content-type"));
        if (formLikeHeader && !isSameOrigin) {
          return new Response(`Cross-site ${request.method} form submissions are forbidden`, {
            status: 403
          });
        }
      } else {
        if (!isSameOrigin) {
          return new Response(`Cross-site ${request.method} form submissions are forbidden`, {
            status: 403
          });
        }
      }
      return next();
    });
  }
  function hasFormLikeHeader(contentType) {
    if (contentType) {
      for (const FORM_CONTENT_TYPE of FORM_CONTENT_TYPES) {
        if (contentType.toLowerCase().includes(FORM_CONTENT_TYPE)) {
          return true;
        }
      }
    }
    return false;
  }
  function createRequest({
    url,
    headers,
    method = "GET",
    body = void 0,
    logger,
    isPrerendered = false,
    routePattern,
    init
  }) {
    const headersObj = isPrerendered ? void 0 : headers instanceof Headers ? headers : new Headers(
      // Filter out HTTP/2 pseudo-headers. These are internally-generated headers added to all HTTP/2 requests with trusted metadata about the request.
      // Examples include `:method`, `:scheme`, `:authority`, and `:path`.
      // They are always prefixed with a colon to distinguish them from other headers, and it is an error to add the to a Headers object manually.
      // See https://httpwg.org/specs/rfc7540.html#HttpRequest
      Object.entries(headers).filter(([name]) => !name.startsWith(":"))
    );
    if (typeof url === "string") url = new URL(url);
    if (isPrerendered) {
      url.search = "";
    }
    const request = new Request(url, __spreadValues({
      method,
      headers: headersObj,
      // body is made available only if the request is for a page that will be on-demand rendered
      body: isPrerendered ? null : body
    }, init));
    if (isPrerendered) {
      let _headers = request.headers;
      const _a29 = Object.getOwnPropertyDescriptor(request, "headers") || {}, { value, writable } = _a29, headersDesc = __objRest(_a29, ["value", "writable"]);
      Object.defineProperty(request, "headers", __spreadProps(__spreadValues({}, headersDesc), {
        get() {
          logger.warn(
            null,
            `\`Astro.request.headers\` was used when rendering the route \`${routePattern}'\`. \`Astro.request.headers\` is not available on prerendered pages. If you need access to request headers, make sure that the page is server-rendered using \`export const prerender = false;\` or by setting \`output\` to \`"server"\` in your Astro config to make all your pages server-rendered by default.`
          );
          return _headers;
        },
        set(newHeaders) {
          _headers = newHeaders;
        }
      }));
    }
    return request;
  }
  function validateAndDecodePathname(pathname) {
    let decoded;
    try {
      decoded = decodeURI(pathname);
    } catch (_e) {
      throw new Error("Invalid URL encoding");
    }
    let iterations = 0;
    while (decoded !== pathname) {
      if (iterations >= MAX_DECODE_ITERATIONS) {
        throw new MultiLevelEncodingError();
      }
      pathname = decoded;
      try {
        decoded = decodeURI(pathname);
      } catch (e2) {
        break;
      }
      iterations++;
    }
    return decoded;
  }
  function template({
    title,
    pathname,
    statusCode = 404,
    tabTitle,
    body
  }) {
    return `<!doctype html>
<html lang="en">
	<head>
		<meta charset="UTF-8">
		<title>${tabTitle}</title>
		<style>
			:root {
				--gray-10: hsl(258, 7%, 10%);
				--gray-20: hsl(258, 7%, 20%);
				--gray-30: hsl(258, 7%, 30%);
				--gray-40: hsl(258, 7%, 40%);
				--gray-50: hsl(258, 7%, 50%);
				--gray-60: hsl(258, 7%, 60%);
				--gray-70: hsl(258, 7%, 70%);
				--gray-80: hsl(258, 7%, 80%);
				--gray-90: hsl(258, 7%, 90%);
				--black: #13151A;
				--accent-light: #E0CCFA;
			}

			* {
				box-sizing: border-box;
			}

			html {
				background: var(--black);
				color-scheme: dark;
				accent-color: var(--accent-light);
			}

			body {
				background-color: var(--gray-10);
				color: var(--gray-80);
				font-family: ui-monospace, Menlo, Monaco, "Cascadia Mono", "Segoe UI Mono", "Roboto Mono", "Oxygen Mono", "Ubuntu Monospace", "Source Code Pro", "Fira Mono", "Droid Sans Mono", "Courier New", monospace;
				line-height: 1.5;
				margin: 0;
			}

			a {
				color: var(--accent-light);
			}

			.center {
				display: flex;
				flex-direction: column;
				justify-content: center;
				align-items: center;
				height: 100vh;
				width: 100vw;
			}

			h1 {
				margin-bottom: 8px;
				color: white;
				font-family: system-ui, "Segoe UI", Roboto, Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol";
				font-weight: 700;
				margin-top: 1rem;
				margin-bottom: 0;
			}

			.statusCode {
				color: var(--accent-light);
			}

			.astro-icon {
				height: 124px;
				width: 124px;
			}

			pre, code {
				padding: 2px 8px;
				background: rgba(0,0,0, 0.25);
				border: 1px solid rgba(255,255,255, 0.25);
				border-radius: 4px;
				font-size: 1.2em;
				margin-top: 0;
				max-width: 60em;
			}
		</style>
	</head>
	<body>
		<main class="center">
			<svg class="astro-icon" xmlns="http://www.w3.org/2000/svg" width="64" height="80" viewBox="0 0 64 80" fill="none"> <path d="M20.5253 67.6322C16.9291 64.3531 15.8793 57.4632 17.3776 52.4717C19.9755 55.6188 23.575 56.6157 27.3035 57.1784C33.0594 58.0468 38.7122 57.722 44.0592 55.0977C44.6709 54.7972 45.2362 54.3978 45.9045 53.9931C46.4062 55.4451 46.5368 56.9109 46.3616 58.4028C45.9355 62.0362 44.1228 64.8429 41.2397 66.9705C40.0868 67.8215 38.8669 68.5822 37.6762 69.3846C34.0181 71.8508 33.0285 74.7426 34.403 78.9491C34.4357 79.0516 34.4649 79.1541 34.5388 79.4042C32.6711 78.5705 31.3069 77.3565 30.2674 75.7604C29.1694 74.0757 28.6471 72.2121 28.6196 70.1957C28.6059 69.2144 28.6059 68.2244 28.4736 67.257C28.1506 64.8985 27.0406 63.8425 24.9496 63.7817C22.8036 63.7192 21.106 65.0426 20.6559 67.1268C20.6215 67.2865 20.5717 67.4446 20.5218 67.6304L20.5253 67.6322Z" fill="white"/> <path d="M20.5253 67.6322C16.9291 64.3531 15.8793 57.4632 17.3776 52.4717C19.9755 55.6188 23.575 56.6157 27.3035 57.1784C33.0594 58.0468 38.7122 57.722 44.0592 55.0977C44.6709 54.7972 45.2362 54.3978 45.9045 53.9931C46.4062 55.4451 46.5368 56.9109 46.3616 58.4028C45.9355 62.0362 44.1228 64.8429 41.2397 66.9705C40.0868 67.8215 38.8669 68.5822 37.6762 69.3846C34.0181 71.8508 33.0285 74.7426 34.403 78.9491C34.4357 79.0516 34.4649 79.1541 34.5388 79.4042C32.6711 78.5705 31.3069 77.3565 30.2674 75.7604C29.1694 74.0757 28.6471 72.2121 28.6196 70.1957C28.6059 69.2144 28.6059 68.2244 28.4736 67.257C28.1506 64.8985 27.0406 63.8425 24.9496 63.7817C22.8036 63.7192 21.106 65.0426 20.6559 67.1268C20.6215 67.2865 20.5717 67.4446 20.5218 67.6304L20.5253 67.6322Z" fill="url(#paint0_linear_738_686)"/> <path d="M0 51.6401C0 51.6401 10.6488 46.4654 21.3274 46.4654L29.3786 21.6102C29.6801 20.4082 30.5602 19.5913 31.5538 19.5913C32.5474 19.5913 33.4275 20.4082 33.7289 21.6102L41.7802 46.4654C54.4274 46.4654 63.1076 51.6401 63.1076 51.6401C63.1076 51.6401 45.0197 2.48776 44.9843 2.38914C44.4652 0.935933 43.5888 0 42.4073 0H20.7022C19.5206 0 18.6796 0.935933 18.1251 2.38914C18.086 2.4859 0 51.6401 0 51.6401Z" fill="white"/> <defs> <linearGradient id="paint0_linear_738_686" x1="31.554" y1="75.4423" x2="39.7462" y2="48.376" gradientUnits="userSpaceOnUse"> <stop stop-color="#D83333"/> <stop offset="1" stop-color="#F041FF"/> </linearGradient> </defs> </svg>
			<h1>${statusCode ? `<span class="statusCode">${statusCode}: </span> ` : ""}<span class="statusMessage">${title}</span></h1>
			${body || `
				<pre>Path: ${escape(pathname)}</pre>
			`}
			</main>
	</body>
</html>`;
  }
  function default404Page(_0) {
    return __async(this, arguments, function* ({ pathname }) {
      return new Response(
        template({
          statusCode: 404,
          title: "Not found",
          tabTitle: "404: Not Found",
          pathname
        }),
        { status: 404, headers: { "Content-Type": "text/html" } }
      );
    });
  }
  function isRoute404(route) {
    return ROUTE404_RE.test(route);
  }
  function isRoute500(route) {
    return ROUTE500_RE.test(route);
  }
  function findRouteToRewrite({
    payload,
    routes,
    request,
    trailingSlash,
    buildFormat,
    base,
    outDir
  }) {
    let newUrl = void 0;
    if (payload instanceof URL) {
      newUrl = payload;
    } else if (payload instanceof Request) {
      newUrl = new URL(payload.url);
    } else {
      newUrl = new URL(collapseDuplicateSlashes(payload), new URL(request.url).origin);
    }
    const { pathname, resolvedUrlPathname } = normalizeRewritePathname(
      newUrl.pathname,
      base,
      trailingSlash,
      buildFormat
    );
    newUrl.pathname = resolvedUrlPathname;
    const decodedPathname = validateAndDecodePathname(pathname);
    if (isRoute404(decodedPathname)) {
      const errorRoute = routes.find((route) => route.route === "/404");
      if (errorRoute) {
        return { routeData: errorRoute, newUrl, pathname: decodedPathname };
      }
    }
    if (isRoute500(decodedPathname)) {
      const errorRoute = routes.find((route) => route.route === "/500");
      if (errorRoute) {
        return { routeData: errorRoute, newUrl, pathname: decodedPathname };
      }
    }
    let foundRoute;
    for (const route of routes) {
      if (route.pattern.test(decodedPathname)) {
        if (route.params && route.params.length !== 0 && route.distURL && route.distURL.length !== 0) {
          if (!route.distURL.find(
            (url) => url.href.replace(outDir.toString(), "").replace(/(?:\/index\.html|\.html)$/, "") === trimSlashes(pathname)
          )) {
            continue;
          }
        }
        foundRoute = route;
        break;
      }
    }
    if (foundRoute) {
      return {
        routeData: foundRoute,
        newUrl,
        pathname: decodedPathname
      };
    } else {
      const custom404 = routes.find((route) => route.route === "/404");
      if (custom404) {
        return { routeData: custom404, newUrl, pathname };
      } else {
        return { routeData: DEFAULT_404_ROUTE, newUrl, pathname };
      }
    }
  }
  function copyRequest(newUrl, oldRequest, isPrerendered, logger, routePattern) {
    if (oldRequest.bodyUsed) {
      throw new AstroError(RewriteWithBodyUsed);
    }
    return createRequest({
      url: newUrl,
      method: oldRequest.method,
      body: oldRequest.body,
      isPrerendered,
      logger,
      headers: isPrerendered ? {} : oldRequest.headers,
      routePattern,
      init: {
        referrer: oldRequest.referrer,
        referrerPolicy: oldRequest.referrerPolicy,
        mode: oldRequest.mode,
        credentials: oldRequest.credentials,
        cache: oldRequest.cache,
        redirect: oldRequest.redirect,
        integrity: oldRequest.integrity,
        signal: oldRequest.signal,
        keepalive: oldRequest.keepalive,
        // https://fetch.spec.whatwg.org/#dom-request-duplex
        // @ts-expect-error It isn't part of the types, but undici accepts it and it allows carrying over the body to a new request
        duplex: "half"
      }
    });
  }
  function setOriginPathname(request, pathname, trailingSlash, buildFormat) {
    if (!pathname) {
      pathname = "/";
    }
    const shouldAppendSlash = shouldAppendForwardSlash(trailingSlash, buildFormat);
    let finalPathname;
    if (pathname === "/") {
      finalPathname = "/";
    } else if (shouldAppendSlash) {
      finalPathname = appendForwardSlash(pathname);
    } else {
      finalPathname = removeTrailingForwardSlash(pathname);
    }
    Reflect.set(request, originPathnameSymbol, encodeURIComponent(finalPathname));
  }
  function getOriginPathname(request) {
    const origin = Reflect.get(request, originPathnameSymbol);
    if (origin) {
      return decodeURIComponent(origin);
    }
    return new URL(request.url).pathname;
  }
  function normalizeRewritePathname(urlPathname, base, trailingSlash, buildFormat) {
    let pathname = collapseDuplicateSlashes(urlPathname);
    const shouldAppendSlash = shouldAppendForwardSlash(trailingSlash, buildFormat);
    if (base !== "/") {
      const isBasePathRequest = urlPathname === base || urlPathname === removeTrailingForwardSlash(base);
      if (isBasePathRequest) {
        pathname = "/";
      } else if (urlPathname.startsWith(base)) {
        pathname = shouldAppendSlash ? appendForwardSlash(urlPathname) : removeTrailingForwardSlash(urlPathname);
        pathname = pathname.slice(base.length);
      }
    }
    if (!pathname.startsWith("/") && shouldAppendSlash && urlPathname.endsWith("/")) {
      pathname = prependForwardSlash(pathname);
    }
    if (buildFormat === "file") {
      pathname = pathname.replace(/\.html$/, "");
    }
    let resolvedUrlPathname;
    if (base !== "/" && (pathname === "" || pathname === "/") && !shouldAppendSlash) {
      resolvedUrlPathname = removeTrailingForwardSlash(base);
    } else {
      resolvedUrlPathname = joinPaths(...[base, pathname].filter(Boolean));
    }
    return { pathname, resolvedUrlPathname };
  }
  function sequence(...handlers2) {
    const filtered = handlers2.filter((h) => !!h);
    const length = filtered.length;
    if (!length) {
      return defineMiddleware((_context, next) => {
        return next();
      });
    }
    return defineMiddleware((context, next) => {
      let carriedPayload = void 0;
      return applyHandle(0, context);
      function applyHandle(i2, handleContext) {
        const handle2 = filtered[i2];
        const result = handle2(handleContext, (payload) => __async(null, null, function* () {
          if (i2 < length - 1) {
            if (payload) {
              let newRequest;
              if (payload instanceof Request) {
                newRequest = payload;
              } else if (payload instanceof URL) {
                newRequest = new Request(payload, handleContext.request.clone());
              } else {
                newRequest = new Request(
                  new URL(payload, handleContext.url.origin),
                  handleContext.request.clone()
                );
              }
              const oldPathname = handleContext.url.pathname;
              const pipeline = Reflect.get(handleContext, pipelineSymbol);
              const { routeData, pathname } = yield pipeline.tryRewrite(
                payload,
                handleContext.request
              );
              if (pipeline.manifest.serverLike === true && handleContext.isPrerendered === false && routeData.prerender === true) {
                throw new AstroError(__spreadProps(__spreadValues({}, ForbiddenRewrite), {
                  message: ForbiddenRewrite.message(
                    handleContext.url.pathname,
                    pathname,
                    routeData.component
                  ),
                  hint: ForbiddenRewrite.hint(routeData.component)
                }));
              }
              carriedPayload = payload;
              handleContext.request = newRequest;
              handleContext.url = new URL(newRequest.url);
              handleContext.params = getParams(routeData, pathname);
              handleContext.routePattern = routeData.route;
              setOriginPathname(
                handleContext.request,
                oldPathname,
                pipeline.manifest.trailingSlash,
                pipeline.manifest.buildFormat
              );
            }
            return applyHandle(i2 + 1, handleContext);
          } else {
            return next(payload != null ? payload : carriedPayload);
          }
        }));
        return result;
      }
    });
  }
  function sanitizeParams(params) {
    return Object.fromEntries(
      Object.entries(params).map(([key, value]) => {
        if (typeof value === "string") {
          return [key, value.normalize().replace(/#/g, "%23").replace(/\?/g, "%3F")];
        }
        return [key, value];
      })
    );
  }
  function getParameter(part, params) {
    if (part.spread) {
      return params[part.content.slice(3)] || "";
    }
    if (part.dynamic) {
      if (!params[part.content]) {
        throw new TypeError(`Missing parameter: ${part.content}`);
      }
      return params[part.content];
    }
    return part.content.normalize().replace(/\?/g, "%3F").replace(/#/g, "%23").replace(/%5B/g, "[").replace(/%5D/g, "]");
  }
  function getSegment(segment, params) {
    const segmentPath = segment.map((part) => getParameter(part, params)).join("");
    return segmentPath ? collapseDuplicateLeadingSlashes("/" + segmentPath) : "";
  }
  function getRouteGenerator(segments, addTrailingSlash) {
    return (params) => {
      const sanitizedParams = sanitizeParams(params);
      let trailing = "";
      if (addTrailingSlash === "always" && segments.length) {
        trailing = "/";
      }
      const path = segments.map((segment) => getSegment(segment, sanitizedParams)).join("") + trailing;
      return path || "/";
    };
  }
  function validateGetStaticPathsParameter([key, value], route) {
    if (!VALID_PARAM_TYPES.includes(typeof value)) {
      throw new AstroError(__spreadProps(__spreadValues({}, GetStaticPathsInvalidRouteParam), {
        message: GetStaticPathsInvalidRouteParam.message(key, value, typeof value),
        location: {
          file: route
        }
      }));
    }
  }
  function stringifyParams(params, route, trailingSlash) {
    const validatedParams = {};
    for (const [key, value] of Object.entries(params)) {
      validateGetStaticPathsParameter([key, value], route.component);
      if (value !== void 0) {
        validatedParams[key] = trimSlashes(value);
      }
    }
    return getRouteGenerator(route.segments, trailingSlash)(validatedParams);
  }
  function validateDynamicRouteModule(mod, {
    ssr,
    route
  }) {
    if ((!ssr || route.prerender) && route.origin !== "internal" && !mod.getStaticPaths) {
      throw new AstroError(__spreadProps(__spreadValues({}, GetStaticPathsRequired), {
        location: { file: route.component }
      }));
    }
  }
  function validateGetStaticPathsResult(result, route) {
    if (!Array.isArray(result)) {
      throw new AstroError(__spreadProps(__spreadValues({}, InvalidGetStaticPathsReturn), {
        message: InvalidGetStaticPathsReturn.message(typeof result),
        location: {
          file: route.component
        }
      }));
    }
    result.forEach((pathObject) => {
      if (typeof pathObject === "object" && Array.isArray(pathObject) || pathObject === null) {
        throw new AstroError(__spreadProps(__spreadValues({}, InvalidGetStaticPathsEntry), {
          message: InvalidGetStaticPathsEntry.message(
            Array.isArray(pathObject) ? "array" : typeof pathObject
          )
        }));
      }
      if (pathObject.params === void 0 || pathObject.params === null || pathObject.params && Object.keys(pathObject.params).length === 0) {
        throw new AstroError(__spreadProps(__spreadValues({}, GetStaticPathsExpectedParams), {
          location: {
            file: route.component
          }
        }));
      }
    });
  }
  function generatePaginateFunction(routeMatch, base, trailingSlash) {
    return function paginateUtility(data, args = {}) {
      const generate = getRouteGenerator(routeMatch.segments, trailingSlash);
      let { pageSize: _pageSize, params: _params2, props: _props } = args;
      const pageSize = _pageSize || 10;
      const paramName = "page";
      const additionalParams = _params2 || {};
      const additionalProps = _props || {};
      let includesFirstPageNumber;
      if (routeMatch.params.includes(`...${paramName}`)) {
        includesFirstPageNumber = false;
      } else if (routeMatch.params.includes(`${paramName}`)) {
        includesFirstPageNumber = true;
      } else {
        throw new AstroError(__spreadProps(__spreadValues({}, PageNumberParamNotFound), {
          message: PageNumberParamNotFound.message(paramName)
        }));
      }
      const lastPage = Math.max(1, Math.ceil(data.length / pageSize));
      const result = [...Array(lastPage).keys()].map((num) => {
        const pageNum = num + 1;
        const start = pageSize === Number.POSITIVE_INFINITY ? 0 : (pageNum - 1) * pageSize;
        const end = Math.min(start + pageSize, data.length);
        const params = __spreadProps(__spreadValues({}, additionalParams), {
          [paramName]: includesFirstPageNumber || pageNum > 1 ? String(pageNum) : void 0
        });
        const current = addRouteBase(generate(__spreadValues({}, params)), base);
        const next = pageNum === lastPage ? void 0 : addRouteBase(generate(__spreadProps(__spreadValues({}, params), { page: String(pageNum + 1) })), base);
        const prev = pageNum === 1 ? void 0 : addRouteBase(
          generate(__spreadProps(__spreadValues({}, params), {
            page: !includesFirstPageNumber && pageNum - 1 === 1 ? void 0 : String(pageNum - 1)
          })),
          base
        );
        const first = pageNum === 1 ? void 0 : addRouteBase(
          generate(__spreadProps(__spreadValues({}, params), {
            page: includesFirstPageNumber ? "1" : void 0
          })),
          base
        );
        const last = pageNum === lastPage ? void 0 : addRouteBase(generate(__spreadProps(__spreadValues({}, params), { page: String(lastPage) })), base);
        return {
          params,
          props: __spreadProps(__spreadValues({}, additionalProps), {
            page: {
              data: data.slice(start, end),
              start,
              end: end - 1,
              size: pageSize,
              total: data.length,
              currentPage: pageNum,
              lastPage,
              url: { current, next, prev, first, last }
            }
          })
        };
      });
      return result;
    };
  }
  function addRouteBase(route, base) {
    let routeWithBase = joinPaths(base, route);
    if (routeWithBase === "") routeWithBase = "/";
    return routeWithBase;
  }
  function callGetStaticPaths(_0) {
    return __async(this, arguments, function* ({
      mod,
      route,
      routeCache,
      ssr,
      base,
      trailingSlash
    }) {
      const cached = routeCache.get(route);
      if (!mod) {
        throw new Error("This is an error caused by Astro and not your code. Please file an issue.");
      }
      if ((cached == null ? void 0 : cached.staticPaths) && cached.mod === mod) {
        return cached.staticPaths;
      }
      validateDynamicRouteModule(mod, { ssr, route });
      if (ssr && !route.prerender || route.origin === "internal") {
        const entry = Object.assign([], { keyed: /* @__PURE__ */ new Map() });
        routeCache.set(route, __spreadProps(__spreadValues({}, cached), { mod, staticPaths: entry }));
        return entry;
      }
      let staticPaths = [];
      if (!mod.getStaticPaths) {
        throw new Error("Unexpected Error.");
      }
      staticPaths = yield mod.getStaticPaths({
        // Q: Why the cast?
        // A: So users downstream can have nicer typings, we have to make some sacrifice in our internal typings, which necessitate a cast here
        paginate: generatePaginateFunction(route, base, trailingSlash),
        routePattern: route.route
      });
      validateGetStaticPathsResult(staticPaths, route);
      const keyedStaticPaths = staticPaths;
      keyedStaticPaths.keyed = /* @__PURE__ */ new Map();
      for (const sp of keyedStaticPaths) {
        const paramsKey = stringifyParams(sp.params, route, trailingSlash);
        keyedStaticPaths.keyed.set(paramsKey, sp);
      }
      routeCache.set(route, __spreadProps(__spreadValues({}, cached), { mod, staticPaths: keyedStaticPaths }));
      return keyedStaticPaths;
    });
  }
  function findPathItemByKey(staticPaths, params, route, logger, trailingSlash) {
    const paramsKey = stringifyParams(params, route, trailingSlash);
    const matchedStaticPath = staticPaths.keyed.get(paramsKey);
    if (matchedStaticPath) {
      return matchedStaticPath;
    }
    logger.debug("router", `findPathItemByKey() - Unexpected cache miss looking for ${paramsKey}`);
  }
  function renderEndpoint(mod, context, isPrerendered, logger) {
    return __async(this, null, function* () {
      var _a29, _b13;
      const { request, url } = context;
      const method = request.method.toUpperCase();
      let handler = (_a29 = mod[method]) != null ? _a29 : mod["ALL"];
      if (!handler && method === "HEAD" && mod["GET"]) {
        handler = mod["GET"];
      }
      if (isPrerendered && !["GET", "HEAD"].includes(method)) {
        logger.warn(
          "router",
          `${url.pathname} ${s.bold(
            method
          )} requests are not available in static endpoints. Mark this page as server-rendered (\`export const prerender = false;\`) or update your config to \`output: 'server'\` to make all your pages server-rendered by default.`
        );
      }
      if (handler === void 0) {
        logger.warn(
          "router",
          `No API Route handler exists for the method "${method}" for the route "${url.pathname}".
Found handlers: ${Object.keys(mod).map((exp) => JSON.stringify(exp)).join(", ")}
` + ("all" in mod ? `One of the exported handlers is "all" (lowercase), did you mean to export 'ALL'?
` : "")
        );
        return new Response(null, { status: 404 });
      }
      if (typeof handler !== "function") {
        logger.error(
          "router",
          `The route "${url.pathname}" exports a value for the method "${method}", but it is of the type ${typeof handler} instead of a function.`
        );
        return new Response(null, { status: 500 });
      }
      let response = yield handler.call(mod, context);
      if (!response || response instanceof Response === false) {
        throw new AstroError(EndpointDidNotReturnAResponse);
      }
      if (REROUTABLE_STATUS_CODES.includes(response.status)) {
        try {
          response.headers.set(REROUTE_DIRECTIVE_HEADER, "no");
        } catch (err) {
          if ((_b13 = err.message) == null ? void 0 : _b13.includes("immutable")) {
            response = new Response(response.body, response);
            response.headers.set(REROUTE_DIRECTIVE_HEADER, "no");
          } else {
            throw err;
          }
        }
      }
      if (method === "HEAD") {
        return new Response(null, response);
      }
      return response;
    });
  }
  function isPromise(value) {
    return !!value && typeof value === "object" && "then" in value && typeof value.then === "function";
  }
  function streamAsyncIterator(stream) {
    return __asyncGenerator(this, null, function* () {
      const reader = stream.getReader();
      try {
        while (true) {
          const { done, value } = yield new __await(reader.read());
          if (done) return;
          yield value;
        }
      } finally {
        reader.releaseLock();
      }
    });
  }
  function stringifyForScript(value) {
    var _a29;
    return (_a29 = JSON.stringify(value)) == null ? void 0 : _a29.replace(/</g, "\\u003c");
  }
  function isHTMLString(value) {
    return !!(value == null ? void 0 : value[htmlStringSymbol]);
  }
  function markHTMLBytes(bytes) {
    return new HTMLBytes(bytes);
  }
  function hasGetReader(obj) {
    return typeof obj.getReader === "function";
  }
  function unescapeChunksAsync(iterable) {
    return __asyncGenerator(this, null, function* () {
      if (hasGetReader(iterable)) {
        try {
          for (var iter = __forAwait(streamAsyncIterator(iterable)), more, temp, error2; more = !(temp = yield new __await(iter.next())).done; more = false) {
            const chunk = temp.value;
            yield unescapeHTML(chunk);
          }
        } catch (temp) {
          error2 = [temp];
        } finally {
          try {
            more && (temp = iter.return) && (yield new __await(temp.call(iter)));
          } finally {
            if (error2)
              throw error2[0];
          }
        }
      } else {
        try {
          for (var iter2 = __forAwait(iterable), more2, temp2, error3; more2 = !(temp2 = yield new __await(iter2.next())).done; more2 = false) {
            const chunk = temp2.value;
            yield unescapeHTML(chunk);
          }
        } catch (temp2) {
          error3 = [temp2];
        } finally {
          try {
            more2 && (temp2 = iter2.return) && (yield new __await(temp2.call(iter2)));
          } finally {
            if (error3)
              throw error3[0];
          }
        }
      }
    });
  }
  function* unescapeChunks(iterable) {
    for (const chunk of iterable) {
      yield unescapeHTML(chunk);
    }
  }
  function unescapeHTML(str) {
    if (!!str && typeof str === "object") {
      if (str instanceof Uint8Array) {
        return markHTMLBytes(str);
      } else if (str instanceof Response && str.body) {
        const body = str.body;
        return unescapeChunksAsync(body);
      } else if (typeof str.then === "function") {
        return Promise.resolve(str).then((value) => {
          return unescapeHTML(value);
        });
      } else if (str[/* @__PURE__ */ Symbol.for("astro:slot-string")]) {
        return str;
      } else if (Symbol.iterator in str) {
        return unescapeChunks(str);
      } else if (Symbol.asyncIterator in str || hasGetReader(str)) {
        return unescapeChunksAsync(str);
      }
    }
    return markHTMLString(str);
  }
  function isVNode(vnode) {
    return vnode && typeof vnode === "object" && vnode[AstroJSX];
  }
  function resolvePropagationHint(input) {
    var _a29, _b13;
    const explicitHint = (_a29 = input.factoryHint) != null ? _a29 : "none";
    if (explicitHint !== "none") {
      return explicitHint;
    }
    if (!input.moduleId) {
      return "none";
    }
    return (_b13 = input.metadataLookup(input.moduleId)) != null ? _b13 : "none";
  }
  function isPropagatingHint(hint) {
    return hint === "self" || hint === "in-tree";
  }
  function getPropagationHint$1(result, factory) {
    return resolvePropagationHint({
      factoryHint: factory.propagation,
      moduleId: factory.moduleId,
      metadataLookup: (moduleId) => {
        var _a29;
        return (_a29 = result.componentMetadata.get(moduleId)) == null ? void 0 : _a29.propagation;
      }
    });
  }
  function isAstroComponentFactory(obj) {
    return obj == null ? false : obj.isAstroComponentFactory === true;
  }
  function isAPropagatingComponent(result, factory) {
    return isPropagatingHint(getPropagationHint(result, factory));
  }
  function getPropagationHint(result, factory) {
    return getPropagationHint$1(result, factory);
  }
  function r(e2) {
    var t2, f, n2 = "";
    if ("string" == typeof e2 || "number" == typeof e2) n2 += e2;
    else if ("object" == typeof e2) if (Array.isArray(e2)) {
      var o2 = e2.length;
      for (t2 = 0; t2 < o2; t2++) e2[t2] && (f = r(e2[t2])) && (n2 && (n2 += " "), n2 += f);
    } else for (f in e2) e2[f] && (n2 && (n2 += " "), n2 += f);
    return n2;
  }
  function clsx() {
    for (var e2, t2, f = 0, n2 = "", o2 = arguments.length; f < o2; f++) (e2 = arguments[f]) && (t2 = r(e2)) && (n2 && (n2 += " "), n2 += t2);
    return n2;
  }
  function serializeArray(value, metadata = {}, parents = /* @__PURE__ */ new WeakSet()) {
    if (parents.has(value)) {
      throw new Error(`Cyclic reference detected while serializing props for <${metadata.displayName} client:${metadata.hydrate}>!

Cyclic references cannot be safely serialized for client-side usage. Please remove the cyclic reference.`);
    }
    parents.add(value);
    const serialized = value.map((v) => {
      return convertToSerializedForm(v, metadata, parents);
    });
    parents.delete(value);
    return serialized;
  }
  function serializeObject(value, metadata = {}, parents = /* @__PURE__ */ new WeakSet()) {
    if (parents.has(value)) {
      throw new Error(`Cyclic reference detected while serializing props for <${metadata.displayName} client:${metadata.hydrate}>!

Cyclic references cannot be safely serialized for client-side usage. Please remove the cyclic reference.`);
    }
    parents.add(value);
    const serialized = Object.fromEntries(
      Object.entries(value).map(([k, v]) => {
        return [k, convertToSerializedForm(v, metadata, parents)];
      })
    );
    parents.delete(value);
    return serialized;
  }
  function convertToSerializedForm(value, metadata = {}, parents = /* @__PURE__ */ new WeakSet()) {
    const tag = Object.prototype.toString.call(value);
    switch (tag) {
      case "[object Date]": {
        return [PROP_TYPE.Date, value.toISOString()];
      }
      case "[object RegExp]": {
        return [PROP_TYPE.RegExp, value.source];
      }
      case "[object Map]": {
        return [PROP_TYPE.Map, serializeArray(Array.from(value), metadata, parents)];
      }
      case "[object Set]": {
        return [PROP_TYPE.Set, serializeArray(Array.from(value), metadata, parents)];
      }
      case "[object BigInt]": {
        return [PROP_TYPE.BigInt, value.toString()];
      }
      case "[object URL]": {
        return [PROP_TYPE.URL, value.toString()];
      }
      case "[object Array]": {
        return [PROP_TYPE.JSON, serializeArray(value, metadata, parents)];
      }
      case "[object Uint8Array]": {
        return [PROP_TYPE.Uint8Array, Array.from(value)];
      }
      case "[object Uint16Array]": {
        return [PROP_TYPE.Uint16Array, Array.from(value)];
      }
      case "[object Uint32Array]": {
        return [PROP_TYPE.Uint32Array, Array.from(value)];
      }
      default: {
        if (value !== null && typeof value === "object") {
          return [PROP_TYPE.Value, serializeObject(value, metadata, parents)];
        }
        if (value === Number.POSITIVE_INFINITY) {
          return [PROP_TYPE.Infinity, 1];
        }
        if (value === Number.NEGATIVE_INFINITY) {
          return [PROP_TYPE.Infinity, -1];
        }
        if (value === void 0) {
          return [PROP_TYPE.Value];
        }
        return [PROP_TYPE.Value, value];
      }
    }
  }
  function serializeProps(props, metadata) {
    const serialized = JSON.stringify(serializeObject(props, metadata));
    return serialized;
  }
  function extractDirectives(inputProps, clientDirectives) {
    let extracted = {
      isPage: false,
      hydration: null,
      props: {},
      propsWithoutTransitionAttributes: {}
    };
    for (const [key, value] of Object.entries(inputProps)) {
      if (key.startsWith("server:")) {
        if (key === "server:root") {
          extracted.isPage = true;
        }
      }
      if (key.startsWith("client:")) {
        if (!extracted.hydration) {
          extracted.hydration = {
            directive: "",
            value: "",
            componentUrl: "",
            componentExport: { value: "" }
          };
        }
        switch (key) {
          case "client:component-path": {
            extracted.hydration.componentUrl = value;
            break;
          }
          case "client:component-export": {
            extracted.hydration.componentExport.value = value;
            break;
          }
          // This is a special prop added to prove that the client hydration method
          // was added statically.
          case "client:component-hydration": {
            break;
          }
          case "client:display-name": {
            break;
          }
          default: {
            extracted.hydration.directive = key.split(":")[1];
            extracted.hydration.value = value;
            if (!clientDirectives.has(extracted.hydration.directive)) {
              const hydrationMethods = Array.from(clientDirectives.keys()).map((d) => `client:${d}`).join(", ");
              throw new Error(
                `Error: invalid hydration directive "${key}". Supported hydration methods: ${hydrationMethods}`
              );
            }
            if (extracted.hydration.directive === "media" && typeof extracted.hydration.value !== "string") {
              throw new AstroError(MissingMediaQueryDirective);
            }
            break;
          }
        }
      } else {
        extracted.props[key] = value;
        if (!transitionDirectivesToCopyOnIsland.includes(key)) {
          extracted.propsWithoutTransitionAttributes[key] = value;
        }
      }
    }
    for (const sym of Object.getOwnPropertySymbols(inputProps)) {
      extracted.props[sym] = inputProps[sym];
      extracted.propsWithoutTransitionAttributes[sym] = inputProps[sym];
    }
    return extracted;
  }
  function generateHydrateScript(scriptOptions, metadata) {
    return __async(this, null, function* () {
      const { renderer, result, astroId, props, attrs } = scriptOptions;
      const { hydrate, componentUrl, componentExport } = metadata;
      if (!componentExport.value) {
        throw new AstroError(__spreadProps(__spreadValues({}, NoMatchingImport), {
          message: NoMatchingImport.message(metadata.displayName)
        }));
      }
      const island = {
        children: "",
        props: {
          // This is for HMR, probably can avoid it in prod
          uid: astroId
        }
      };
      if (attrs) {
        for (const [key, value] of Object.entries(attrs)) {
          island.props[key] = escapeHTML(value);
        }
      }
      island.props["component-url"] = yield result.resolve(decodeURI(componentUrl));
      if (renderer.clientEntrypoint) {
        island.props["component-export"] = componentExport.value;
        island.props["renderer-url"] = yield result.resolve(
          decodeURI(renderer.clientEntrypoint.toString())
        );
        island.props["props"] = escapeHTML(serializeProps(props, metadata));
      }
      island.props["ssr"] = "";
      island.props["client"] = hydrate;
      let beforeHydrationUrl = yield result.resolve("astro:scripts/before-hydration.js");
      if (beforeHydrationUrl.length) {
        island.props["before-hydration-url"] = beforeHydrationUrl;
      }
      island.props["opts"] = escapeHTML(
        JSON.stringify({
          name: metadata.displayName,
          value: metadata.hydrateArgs || ""
        })
      );
      transitionDirectivesToCopyOnIsland.forEach((name) => {
        if (typeof props[name] !== "undefined") {
          island.props[name] = props[name];
        }
      });
      return island;
    });
  }
  function bitwise(str) {
    let hash = 0;
    if (str.length === 0) return hash;
    for (let i2 = 0; i2 < str.length; i2++) {
      const ch = str.charCodeAt(i2);
      hash = (hash << 5) - hash + ch;
      hash = hash & hash;
    }
    return hash;
  }
  function shorthash(text) {
    let num;
    let result = "";
    let integer = bitwise(text);
    const sign = integer < 0 ? "Z" : "";
    integer = Math.abs(integer);
    while (integer >= binary) {
      num = integer % binary;
      integer = Math.floor(integer / binary);
      result = dictionary[num] + result;
    }
    if (integer > 0) {
      result = dictionary[integer] + result;
    }
    return sign + result;
  }
  function isHeadAndContent(obj) {
    return typeof obj === "object" && obj !== null && !!obj[headAndContentSym];
  }
  function createThinHead() {
    return {
      [headAndContentSym]: true
    };
  }
  function determineIfNeedsHydrationScript(result) {
    if (result._metadata.templateDepth > 0) {
      return !result._metadata.hasHydrationScript;
    }
    if (result._metadata.hasHydrationScript) {
      return false;
    }
    return result._metadata.hasHydrationScript = true;
  }
  function determinesIfNeedsDirectiveScript(result, directive) {
    if (result._metadata.templateDepth > 0) {
      return !result._metadata.hasDirectives.has(directive);
    }
    if (result._metadata.hasDirectives.has(directive)) {
      return false;
    }
    result._metadata.hasDirectives.add(directive);
    return true;
  }
  function getDirectiveScriptText(result, directive) {
    const clientDirectives = result.clientDirectives;
    const clientDirective = clientDirectives.get(directive);
    if (!clientDirective) {
      throw new Error(`Unknown directive: ${directive}`);
    }
    return clientDirective;
  }
  function getPrescripts(result, type, directive) {
    switch (type) {
      case "both":
        return `<style>${ISLAND_STYLES}</style><script>${getDirectiveScriptText(result, directive)}</script><script>${astro_island_prebuilt_default}</script>`;
      case "directive":
        return `<script>${getDirectiveScriptText(result, directive)}</script>`;
    }
  }
  function collectPropagatedHeadParts(input) {
    return __async(this, null, function* () {
      const collectedHeadParts = [];
      const iterator = input.propagators.values();
      while (true) {
        const { value, done } = iterator.next();
        if (done) {
          break;
        }
        const returnValue = yield value.init(input.result);
        if (input.isHeadAndContent(returnValue) && returnValue.head) {
          collectedHeadParts.push(returnValue.head);
        }
      }
      return collectedHeadParts;
    });
  }
  function shouldRenderHeadInstruction(state) {
    return !state.hasRenderedHead && !state.partial;
  }
  function shouldRenderMaybeHeadInstruction(state) {
    return !state.hasRenderedHead && !state.headInTree && !state.partial;
  }
  function shouldRenderInstruction$1(type, state) {
    return type === "head" ? shouldRenderHeadInstruction(state) : shouldRenderMaybeHeadInstruction(state);
  }
  function registerIfPropagating(result, factory, instance) {
    var _a29;
    if (factory.propagation === "self" || factory.propagation === "in-tree") {
      result._metadata.propagators.add(
        instance
      );
      return;
    }
    if (factory.moduleId) {
      const hint = (_a29 = result.componentMetadata.get(factory.moduleId)) == null ? void 0 : _a29.propagation;
      if (isPropagatingHint(hint != null ? hint : "none")) {
        result._metadata.propagators.add(
          instance
        );
      }
    }
  }
  function bufferPropagatedHead(result) {
    return __async(this, null, function* () {
      const collected = yield collectPropagatedHeadParts({
        propagators: result._metadata.propagators,
        result,
        isHeadAndContent
      });
      result._metadata.extraHead.push(...collected);
    });
  }
  function shouldRenderInstruction(type, state) {
    return shouldRenderInstruction$1(type, state);
  }
  function getInstructionRenderState(result) {
    return {
      hasRenderedHead: result._metadata.hasRenderedHead,
      headInTree: result._metadata.headInTree,
      partial: result.partial
    };
  }
  function renderCspContent(result) {
    const finalScriptHashes = /* @__PURE__ */ new Set();
    const finalStyleHashes = /* @__PURE__ */ new Set();
    for (const scriptHash of result.scriptHashes) {
      finalScriptHashes.add(`'${scriptHash}'`);
    }
    for (const styleHash of result.styleHashes) {
      finalStyleHashes.add(`'${styleHash}'`);
    }
    for (const styleHash of result._metadata.extraStyleHashes) {
      finalStyleHashes.add(`'${styleHash}'`);
    }
    for (const scriptHash of result._metadata.extraScriptHashes) {
      finalScriptHashes.add(`'${scriptHash}'`);
    }
    let directives;
    if (result.directives.length > 0) {
      directives = result.directives.join(";") + ";";
    }
    let scriptResources = "'self'";
    if (result.scriptResources.length > 0) {
      scriptResources = result.scriptResources.map((r2) => `${r2}`).join(" ");
    }
    let styleResources = "'self'";
    if (result.styleResources.length > 0) {
      styleResources = result.styleResources.map((r2) => `${r2}`).join(" ");
    }
    const strictDynamic = result.isStrictDynamic ? ` 'strict-dynamic'` : "";
    const scriptSrc = `script-src ${scriptResources} ${Array.from(finalScriptHashes).join(" ")}${strictDynamic};`;
    const styleSrc = `style-src ${styleResources} ${Array.from(finalStyleHashes).join(" ")};`;
    return [directives, scriptSrc, styleSrc].filter(Boolean).join(" ");
  }
  function createRenderInstruction(instruction) {
    return Object.defineProperty(instruction, RenderInstructionSymbol, {
      value: true
    });
  }
  function isRenderInstruction(chunk) {
    return chunk && typeof chunk === "object" && chunk[RenderInstructionSymbol];
  }
  function defineScriptVars(vars) {
    let output = "";
    for (const [key, value] of Object.entries(vars)) {
      output += `const ${toIdent(key)} = ${stringifyForScript(value)};
`;
    }
    return markHTMLString(output);
  }
  function formatList(values) {
    if (values.length === 1) {
      return values[0];
    }
    return `${values.slice(0, -1).join(", ")} or ${values[values.length - 1]}`;
  }
  function isCustomElement(tagName) {
    return tagName.includes("-");
  }
  function handleBooleanAttribute(key, value, shouldEscape, tagName) {
    if (tagName && isCustomElement(tagName)) {
      return markHTMLString(` ${key}="${toAttributeString(value, shouldEscape)}"`);
    }
    return markHTMLString(value ? ` ${key}` : "");
  }
  function addAttribute(value, key, shouldEscape = true, tagName = "") {
    if (value == null) {
      return "";
    }
    if (INVALID_ATTR_NAME_CHAR.test(key)) {
      return "";
    }
    if (STATIC_DIRECTIVES.has(key)) {
      console.warn(`[astro] The "${key}" directive cannot be applied dynamically at runtime. It will not be rendered as an attribute.

Make sure to use the static attribute syntax (\`${key}={value}\`) instead of the dynamic spread syntax (\`{...{ "${key}": value }}\`).`);
      return "";
    }
    if (key === "class:list") {
      const listValue = toAttributeString(clsx(value), shouldEscape);
      if (listValue === "") {
        return "";
      }
      return markHTMLString(` ${key.slice(0, -5)}="${listValue}"`);
    }
    if (key === "style" && !(value instanceof HTMLString)) {
      if (Array.isArray(value) && value.length === 2) {
        return markHTMLString(
          ` ${key}="${toAttributeString(`${toStyleString(value[0])};${value[1]}`, shouldEscape)}"`
        );
      }
      if (typeof value === "object") {
        return markHTMLString(` ${key}="${toAttributeString(toStyleString(value), shouldEscape)}"`);
      }
    }
    if (key === "className") {
      return markHTMLString(` class="${toAttributeString(value, shouldEscape)}"`);
    }
    if (htmlBooleanAttributes.test(key)) {
      return handleBooleanAttribute(key, value, shouldEscape, tagName);
    }
    if (value === "") {
      return markHTMLString(` ${key}`);
    }
    if (key === "popover" && typeof value === "boolean") {
      return handleBooleanAttribute(key, value, shouldEscape, tagName);
    }
    if (key === "download" && typeof value === "boolean") {
      return handleBooleanAttribute(key, value, shouldEscape, tagName);
    }
    if (key === "hidden" && typeof value === "boolean") {
      return handleBooleanAttribute(key, value, shouldEscape, tagName);
    }
    return markHTMLString(` ${key}="${toAttributeString(value, shouldEscape)}"`);
  }
  function internalSpreadAttributes(values, shouldEscape = true, tagName) {
    let output = "";
    for (const [key, value] of Object.entries(values)) {
      output += addAttribute(value, key, shouldEscape, tagName);
    }
    return markHTMLString(output);
  }
  function renderElement$1(name, { props: _props, children = "" }, shouldEscape = true) {
    const _a29 = _props, { lang: _, "data-astro-id": astroId, "define:vars": defineVars } = _a29, props = __objRest(_a29, ["lang", "data-astro-id", "define:vars"]);
    if (defineVars) {
      if (name === "style") {
        delete props["is:global"];
        delete props["is:scoped"];
      }
      if (name === "script") {
        delete props.hoist;
        children = defineScriptVars(defineVars) + "\n" + children;
      }
    }
    if ((children == null || children === "") && voidElementNames.test(name)) {
      return `<${name}${internalSpreadAttributes(props, shouldEscape, name)}>`;
    }
    return `<${name}${internalSpreadAttributes(props, shouldEscape, name)}>${children}</${name}>`;
  }
  function createBufferedRenderer(destination, renderFunction) {
    return new BufferedRenderer(destination, renderFunction);
  }
  function promiseWithResolvers() {
    let resolve, reject;
    const promise = new Promise((_resolve, _reject) => {
      resolve = _resolve;
      reject = _reject;
    });
    return {
      promise,
      resolve,
      reject
    };
  }
  function stablePropsKey(props) {
    const keys = Object.keys(props).sort();
    let result = "{";
    for (let i2 = 0; i2 < keys.length; i2++) {
      if (i2 > 0) result += ",";
      result += JSON.stringify(keys[i2]) + ":" + JSON.stringify(props[keys[i2]]);
    }
    result += "}";
    return result;
  }
  function deduplicateElements(elements) {
    if (elements.length <= 1) return elements;
    const seen = /* @__PURE__ */ new Set();
    return elements.filter((item) => {
      const key = stablePropsKey(item.props) + item.children;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
  }
  function renderAllHeadContent(result) {
    result._metadata.hasRenderedHead = true;
    let content = "";
    if (result.shouldInjectCspMetaTags && result.cspDestination === "meta") {
      content += renderElement$1(
        "meta",
        {
          props: {
            "http-equiv": "content-security-policy",
            content: renderCspContent(result)
          },
          children: ""
        },
        false
      );
    }
    const styles = deduplicateElements(Array.from(result.styles)).map(
      (style) => style.props.rel === "stylesheet" ? renderElement$1("link", style) : renderElement$1("style", style)
    );
    result.styles.clear();
    const scripts = deduplicateElements(Array.from(result.scripts)).map((script) => {
      if (result.userAssetsBase) {
        script.props.src = (result.base === "/" ? "" : result.base) + result.userAssetsBase + script.props.src;
      }
      return renderElement$1("script", script, false);
    });
    const links = deduplicateElements(Array.from(result.links)).map(
      (link) => renderElement$1("link", link, false)
    );
    const sep = result.compressHTML === true || result.compressHTML === "jsx" ? "" : "\n";
    content += styles.join(sep) + links.join(sep) + scripts.join(sep);
    content += result._metadata.extraHead.join("");
    return markHTMLString(content);
  }
  function renderHead() {
    return createRenderInstruction({ type: "head" });
  }
  function maybeRenderHead() {
    return createRenderInstruction({ type: "maybe-head" });
  }
  function encodeHexUpperCase(data) {
    let result = "";
    for (let i2 = 0; i2 < data.length; i2++) {
      result += alphabetUpperCase[data[i2] >> 4];
      result += alphabetUpperCase[data[i2] & 15];
    }
    return result;
  }
  function decodeHex(data) {
    if (data.length % 2 !== 0) {
      throw new Error("Invalid hex string");
    }
    const result = new Uint8Array(data.length / 2);
    for (let i2 = 0; i2 < data.length; i2 += 2) {
      if (!(data[i2] in decodeMap)) {
        throw new Error("Invalid character");
      }
      if (!(data[i2 + 1] in decodeMap)) {
        throw new Error("Invalid character");
      }
      result[i2 / 2] |= decodeMap[data[i2]] << 4;
      result[i2 / 2] |= decodeMap[data[i2 + 1]];
    }
    return result;
  }
  function encodeBase64(bytes) {
    return encodeBase64_internal(bytes, base64Alphabet, EncodingPadding.Include);
  }
  function encodeBase64_internal(bytes, alphabet, padding) {
    let result = "";
    for (let i2 = 0; i2 < bytes.byteLength; i2 += 3) {
      let buffer2 = 0;
      let bufferBitSize = 0;
      for (let j = 0; j < 3 && i2 + j < bytes.byteLength; j++) {
        buffer2 = buffer2 << 8 | bytes[i2 + j];
        bufferBitSize += 8;
      }
      for (let j = 0; j < 4; j++) {
        if (bufferBitSize >= 6) {
          result += alphabet[buffer2 >> bufferBitSize - 6 & 63];
          bufferBitSize -= 6;
        } else if (bufferBitSize > 0) {
          result += alphabet[buffer2 << 6 - bufferBitSize & 63];
          bufferBitSize = 0;
        } else if (padding === EncodingPadding.Include) {
          result += "=";
        }
      }
    }
    return result;
  }
  function decodeBase64(encoded) {
    return decodeBase64_internal(encoded, base64DecodeMap, DecodingPadding.Required);
  }
  function decodeBase64_internal(encoded, decodeMap2, padding) {
    const result = new Uint8Array(Math.ceil(encoded.length / 4) * 3);
    let totalBytes = 0;
    for (let i2 = 0; i2 < encoded.length; i2 += 4) {
      let chunk = 0;
      let bitsRead = 0;
      for (let j = 0; j < 4; j++) {
        if (padding === DecodingPadding.Required && encoded[i2 + j] === "=") {
          continue;
        }
        if (padding === DecodingPadding.Ignore && (i2 + j >= encoded.length || encoded[i2 + j] === "=")) {
          continue;
        }
        if (j > 0 && encoded[i2 + j - 1] === "=") {
          throw new Error("Invalid padding");
        }
        if (!(encoded[i2 + j] in decodeMap2)) {
          throw new Error("Invalid character");
        }
        chunk |= decodeMap2[encoded[i2 + j]] << (3 - j) * 6;
        bitsRead += 6;
      }
      if (bitsRead < 24) {
        let unused;
        if (bitsRead === 12) {
          unused = chunk & 65535;
        } else if (bitsRead === 18) {
          unused = chunk & 255;
        } else {
          throw new Error("Invalid padding");
        }
        if (unused !== 0) {
          throw new Error("Invalid padding");
        }
      }
      const byteLength = Math.floor(bitsRead / 8);
      for (let i3 = 0; i3 < byteLength; i3++) {
        result[totalBytes] = chunk >> 16 - i3 * 8 & 255;
        totalBytes++;
      }
    }
    return result.slice(0, totalBytes);
  }
  function _installLazyMethods(inst, group, methods2) {
    const proto = Object.getPrototypeOf(inst);
    let installed = _installedGroups.get(proto);
    if (!installed) {
      installed = /* @__PURE__ */ new Set();
      _installedGroups.set(proto, installed);
    }
    if (installed.has(group))
      return;
    installed.add(group);
    for (const key in methods2) {
      const fn = methods2[key];
      Object.defineProperty(proto, key, {
        configurable: true,
        enumerable: false,
        get() {
          const bound = fn.bind(this);
          Object.defineProperty(this, key, {
            configurable: true,
            writable: true,
            enumerable: true,
            value: bound
          });
          return bound;
        },
        set(v) {
          Object.defineProperty(this, key, {
            configurable: true,
            writable: true,
            enumerable: true,
            value: v
          });
        }
      });
    }
  }
  function array(element, params) {
    return /* @__PURE__ */ _array(ZodArray, element, params);
  }
  function union(options, params) {
    return new ZodUnion(__spreadValues({
      type: "union",
      options
    }, normalizeParams(params)));
  }
  function intersection(left, right) {
    return new ZodIntersection({
      type: "intersection",
      left,
      right
    });
  }
  function _enum(values, params) {
    const entries = Array.isArray(values) ? Object.fromEntries(values.map((v) => [v, v])) : values;
    return new ZodEnum(__spreadValues({
      type: "enum",
      entries
    }, normalizeParams(params)));
  }
  function transform2(fn) {
    return new ZodTransform({
      type: "transform",
      transform: fn
    });
  }
  function optional(innerType) {
    return new ZodOptional({
      type: "optional",
      innerType
    });
  }
  function exactOptional(innerType) {
    return new ZodExactOptional({
      type: "optional",
      innerType
    });
  }
  function nullable(innerType) {
    return new ZodNullable({
      type: "nullable",
      innerType
    });
  }
  function _default2(innerType, defaultValue) {
    return new ZodDefault({
      type: "default",
      innerType,
      get defaultValue() {
        return typeof defaultValue === "function" ? defaultValue() : shallowClone(defaultValue);
      }
    });
  }
  function prefault(innerType, defaultValue) {
    return new ZodPrefault({
      type: "prefault",
      innerType,
      get defaultValue() {
        return typeof defaultValue === "function" ? defaultValue() : shallowClone(defaultValue);
      }
    });
  }
  function nonoptional(innerType, params) {
    return new ZodNonOptional(__spreadValues({
      type: "nonoptional",
      innerType
    }, normalizeParams(params)));
  }
  function _catch(innerType, catchValue) {
    return new ZodCatch({
      type: "catch",
      innerType,
      catchValue: typeof catchValue === "function" ? catchValue : () => catchValue
    });
  }
  function pipe(in_, out) {
    return new ZodPipe({
      type: "pipe",
      in: in_,
      out
      // ...util.normalizeParams(params),
    });
  }
  function readonly(innerType) {
    return new ZodReadonly({
      type: "readonly",
      innerType
    });
  }
  function custom(fn, _params2) {
    return /* @__PURE__ */ _custom(ZodCustom, fn != null ? fn : (() => true), _params2);
  }
  function refine(fn, _params2 = {}) {
    return /* @__PURE__ */ _refine(ZodCustom, fn, _params2);
  }
  function superRefine(fn, params) {
    return /* @__PURE__ */ _superRefine(fn, params);
  }
  function decodeKey(encoded) {
    return __async(this, null, function* () {
      const bytes = decodeBase64(encoded);
      return crypto.subtle.importKey("raw", bytes.buffer, ALGORITHM, true, [
        "encrypt",
        "decrypt"
      ]);
    });
  }
  function encryptString(key, raw, additionalData) {
    return __async(this, null, function* () {
      const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH / 2));
      const data = encoder$1.encode(raw);
      const params = { name: ALGORITHM, iv };
      if (additionalData) {
        params.additionalData = encoder$1.encode(additionalData);
      }
      const buffer2 = yield crypto.subtle.encrypt(params, key, data);
      return encodeHexUpperCase(iv) + encodeBase64(new Uint8Array(buffer2));
    });
  }
  function decryptString(key, encoded, additionalData) {
    return __async(this, null, function* () {
      const iv = decodeHex(encoded.slice(0, IV_LENGTH));
      const dataArray = decodeBase64(encoded.slice(IV_LENGTH));
      const params = { name: ALGORITHM, iv };
      if (additionalData) {
        params.additionalData = encoder$1.encode(additionalData);
      }
      const decryptedBuffer = yield crypto.subtle.decrypt(params, key, dataArray);
      const decryptedString = decoder$1.decode(decryptedBuffer);
      return decryptedString;
    });
  }
  function generateCspDigest(data, algorithm) {
    return __async(this, null, function* () {
      const hashBuffer = yield crypto.subtle.digest(algorithm, encoder$1.encode(data));
      const hash = encodeBase64(new Uint8Array(hashBuffer));
      return `${ALGORITHMS[algorithm]}${hash}`;
    });
  }
  function isRenderTemplateResult(obj) {
    return typeof obj === "object" && obj !== null && !!obj[renderTemplateResultSym];
  }
  function renderTemplate(htmlParts, ...expressions) {
    return new RenderTemplateResult(htmlParts, expressions);
  }
  function isSlotString(str) {
    return !!str[slotString];
  }
  function mergeSlotInstructions(target, source) {
    var _a29;
    if ((_a29 = source.instructions) == null ? void 0 : _a29.length) {
      target != null ? target : target = [];
      target.push(...source.instructions);
    }
    return target;
  }
  function renderSlot(result, slotted, fallback) {
    return {
      render(destination) {
        return __async(this, null, function* () {
          yield renderChild(destination, typeof slotted === "function" ? slotted(result) : slotted);
        });
      }
    };
  }
  function renderSlotToString(result, slotted, fallback) {
    return __async(this, null, function* () {
      let content = "";
      let instructions = null;
      const temporaryDestination = {
        write(chunk) {
          if (chunk instanceof SlotString) {
            content += chunk;
            instructions = mergeSlotInstructions(instructions, chunk);
          } else if (chunk instanceof Response) return;
          else if (typeof chunk === "object" && "type" in chunk && typeof chunk.type === "string") {
            if (instructions === null) {
              instructions = [];
            }
            instructions.push(chunk);
          } else {
            content += chunkToString(result, chunk);
          }
        }
      };
      const renderInstance = renderSlot(result, slotted);
      yield renderInstance.render(temporaryDestination);
      return markHTMLString(new SlotString(content, instructions));
    });
  }
  function renderSlots(_0) {
    return __async(this, arguments, function* (result, slots = {}) {
      let slotInstructions = null;
      let children = {};
      if (slots) {
        yield Promise.all(
          Object.entries(slots).map(
            ([key, value]) => renderSlotToString(result, value).then((output) => {
              if (output.instructions) {
                if (slotInstructions === null) {
                  slotInstructions = [];
                }
                slotInstructions.push(...output.instructions);
              }
              children[key] = output;
            })
          )
        );
      }
      return { slotInstructions, children };
    });
  }
  function createSlotValueFromString(content) {
    return function() {
      return renderTemplate`${unescapeHTML(content)}`;
    };
  }
  function containsServerDirective(props) {
    return "server:component-directive" in props;
  }
  function createSearchParams(encryptedComponentExport, encryptedProps, slots) {
    const params = new URLSearchParams();
    params.set("e", encryptedComponentExport);
    params.set("p", encryptedProps);
    params.set("s", slots);
    return params;
  }
  function isWithinURLLimit(pathname, params) {
    const url = pathname + "?" + params.toString();
    const chars = url.length;
    return chars < 2048;
  }
  function stringifyChunk(result, chunk) {
    if (isRenderInstruction(chunk)) {
      const instruction = chunk;
      switch (instruction.type) {
        case "directive": {
          const { hydration } = instruction;
          const needsHydrationScript = hydration && determineIfNeedsHydrationScript(result);
          const needsDirectiveScript = hydration && determinesIfNeedsDirectiveScript(result, hydration.directive);
          if (needsHydrationScript) {
            const prescripts = getPrescripts(result, "both", hydration.directive);
            return markHTMLString(prescripts);
          } else if (needsDirectiveScript) {
            const prescripts = getPrescripts(result, "directive", hydration.directive);
            return markHTMLString(prescripts);
          } else {
            return "";
          }
        }
        case "head": {
          if (!shouldRenderInstruction("head", getInstructionRenderState(result))) {
            return "";
          }
          return renderAllHeadContent(result);
        }
        case "maybe-head": {
          if (!shouldRenderInstruction("maybe-head", getInstructionRenderState(result))) {
            return "";
          }
          return renderAllHeadContent(result);
        }
        case "renderer-hydration-script": {
          const { rendererSpecificHydrationScripts } = result._metadata;
          const { rendererName } = instruction;
          if (result._metadata.templateDepth > 0) {
            return instruction.render();
          }
          if (!rendererSpecificHydrationScripts.has(rendererName)) {
            rendererSpecificHydrationScripts.add(rendererName);
            return instruction.render();
          }
          return "";
        }
        case "server-island-runtime": {
          if (result._metadata.templateDepth > 0) {
            return renderServerIslandRuntime();
          }
          if (result._metadata.hasRenderedServerIslandRuntime) {
            return "";
          }
          result._metadata.hasRenderedServerIslandRuntime = true;
          return renderServerIslandRuntime();
        }
        case "script": {
          const { id, content } = instruction;
          if (result._metadata.templateDepth > 0) {
            return content;
          }
          if (result._metadata.renderedScripts.has(id)) {
            return "";
          }
          result._metadata.renderedScripts.add(id);
          return content;
        }
        case "template-enter": {
          result._metadata.templateDepth++;
          return "";
        }
        case "template-exit": {
          if (result._metadata.templateDepth <= 0) {
            throw new Error(
              "Unexpected template-exit instruction without a matching template-enter. This may indicate that the compiler emitted unbalanced template boundaries, or that a component manually injected a template-exit render instruction."
            );
          }
          result._metadata.templateDepth--;
          return "";
        }
        default: {
          throw new Error(`Unknown chunk type: ${chunk.type}`);
        }
      }
    } else if (chunk instanceof Response) {
      return "";
    } else if (isSlotString(chunk)) {
      let out = "";
      const c = chunk;
      if (c.instructions) {
        for (const instr of c.instructions) {
          out += stringifyChunk(result, instr);
        }
      }
      out += chunk.toString();
      return out;
    }
    return chunk.toString();
  }
  function chunkToString(result, chunk) {
    if (ArrayBuffer.isView(chunk)) {
      return decoder2.decode(chunk);
    } else {
      return stringifyChunk(result, chunk);
    }
  }
  function chunkToByteArray(result, chunk) {
    if (ArrayBuffer.isView(chunk)) {
      return chunk;
    } else {
      const stringified = stringifyChunk(result, chunk);
      return encoder.encode(stringified.toString());
    }
  }
  function chunkToByteArrayOrString(result, chunk) {
    if (ArrayBuffer.isView(chunk)) {
      return chunk;
    } else {
      return stringifyChunk(result, chunk).toString();
    }
  }
  function isRenderInstance(obj) {
    return !!obj && typeof obj === "object" && "render" in obj && typeof obj.render === "function";
  }
  function renderChild(destination, child) {
    if (typeof child === "string") {
      destination.write(markHTMLString(escapeHTML(child)));
      return;
    }
    if (isPromise(child)) {
      return child.then((x) => renderChild(destination, x));
    }
    if (child instanceof SlotString) {
      destination.write(child);
      return;
    }
    if (isHTMLString(child)) {
      destination.write(child);
      return;
    }
    if (!child && child !== 0) {
      return;
    }
    if (Array.isArray(child)) {
      return renderArray(destination, child);
    }
    if (typeof child === "function") {
      return renderChild(destination, child());
    }
    if (isRenderInstance(child)) {
      return child.render(destination);
    }
    if (isRenderTemplateResult(child)) {
      return child.render(destination);
    }
    if (isAstroComponentInstance(child)) {
      return child.render(destination);
    }
    if (ArrayBuffer.isView(child)) {
      destination.write(child);
      return;
    }
    if (typeof child === "object" && (Symbol.asyncIterator in child || Symbol.iterator in child)) {
      if (Symbol.asyncIterator in child) {
        return renderAsyncIterable(destination, child);
      }
      return renderIterable(destination, child);
    }
    destination.write(child);
  }
  function renderArray(destination, children) {
    for (let i2 = 0; i2 < children.length; i2++) {
      const result = renderChild(destination, children[i2]);
      if (isPromise(result)) {
        if (i2 + 1 >= children.length) {
          return result;
        }
        const remaining = children.length - i2 - 1;
        const flushers = new Array(remaining);
        for (let j = 0; j < remaining; j++) {
          flushers[j] = createBufferedRenderer(destination, (bufferDestination) => {
            return renderChild(bufferDestination, children[i2 + 1 + j]);
          });
        }
        return result.then(() => {
          let k = 0;
          const iterate = () => {
            while (k < flushers.length) {
              const flushResult = flushers[k++].flush();
              if (isPromise(flushResult)) {
                return flushResult.then(iterate);
              }
            }
          };
          return iterate();
        });
      }
    }
  }
  function renderIterable(destination, children) {
    const iterator = children[Symbol.iterator]();
    const iterate = () => {
      for (; ; ) {
        const { value, done } = iterator.next();
        if (done) {
          break;
        }
        const result = renderChild(destination, value);
        if (isPromise(result)) {
          return result.then(iterate);
        }
      }
    };
    return iterate();
  }
  function renderAsyncIterable(destination, children) {
    return __async(this, null, function* () {
      try {
        for (var iter = __forAwait(children), more, temp, error2; more = !(temp = yield iter.next()).done; more = false) {
          const value = temp.value;
          yield renderChild(destination, value);
        }
      } catch (temp) {
        error2 = [temp];
      } finally {
        try {
          more && (temp = iter.return) && (yield temp.call(iter));
        } finally {
          if (error2)
            throw error2[0];
        }
      }
    });
  }
  function validateComponentProps(props, clientDirectives, displayName) {
    if (props != null) {
      const directives = [...clientDirectives.keys()].map((directive) => `client:${directive}`);
      for (const prop of Object.keys(props)) {
        if (directives.includes(prop)) {
          console.warn(
            `You are attempting to render <${displayName} ${prop} />, but ${displayName} is an Astro component. Astro components do not render in the client and should not have a hydration directive. Please use a framework component for client rendering.`
          );
        }
      }
    }
  }
  function createAstroComponentInstance(result, displayName, factory, props, slots = {}) {
    validateComponentProps(props, result.clientDirectives, displayName);
    const instance = new AstroComponentInstance(result, props, slots, factory);
    registerIfPropagating(result, factory, instance);
    return instance;
  }
  function isAstroComponentInstance(obj) {
    return typeof obj === "object" && obj !== null && !!obj[astroComponentInstanceSym];
  }
  function renderToString(result, componentFactory, props, children, isPage = false, route) {
    return __async(this, null, function* () {
      const templateResult = yield callComponentAsTemplateResultOrResponse(
        result,
        componentFactory,
        props,
        children,
        route
      );
      if (templateResult instanceof Response) return templateResult;
      let str = "";
      let renderedFirstPageChunk = false;
      if (isPage) {
        yield bufferHeadContent(result);
      }
      const destination = {
        write(chunk) {
          if (isPage && !renderedFirstPageChunk) {
            renderedFirstPageChunk = true;
            if (!result.partial && !DOCTYPE_EXP.test(String(chunk))) {
              const doctype = result.compressHTML ? "<!DOCTYPE html>" : "<!DOCTYPE html>\n";
              str += doctype;
            }
          }
          if (chunk instanceof Response) return;
          str += chunkToString(result, chunk);
        }
      };
      yield templateResult.render(destination);
      return str;
    });
  }
  function renderToReadableStream(result, componentFactory, props, children, isPage = false, route) {
    return __async(this, null, function* () {
      const templateResult = yield callComponentAsTemplateResultOrResponse(
        result,
        componentFactory,
        props,
        children,
        route
      );
      if (templateResult instanceof Response) return templateResult;
      let renderedFirstPageChunk = false;
      if (isPage) {
        yield bufferHeadContent(result);
      }
      return new ReadableStream({
        start(controller) {
          const destination = {
            write(chunk) {
              if (isPage && !renderedFirstPageChunk) {
                renderedFirstPageChunk = true;
                if (!result.partial && !DOCTYPE_EXP.test(String(chunk))) {
                  const doctype = result.compressHTML ? "<!DOCTYPE html>" : "<!DOCTYPE html>\n";
                  controller.enqueue(encoder.encode(doctype));
                }
              }
              if (chunk instanceof Response) {
                throw new AstroError(__spreadValues({}, ResponseSentError));
              }
              const bytes = chunkToByteArray(result, chunk);
              controller.enqueue(bytes);
            }
          };
          (() => __async(null, null, function* () {
            try {
              yield templateResult.render(destination);
              controller.close();
            } catch (e2) {
              if (AstroError.is(e2) && !e2.loc) {
                e2.setLocation({
                  file: route == null ? void 0 : route.component
                });
              }
              setTimeout(() => controller.error(e2), 0);
            }
          }))();
        },
        cancel() {
          result.cancelled = true;
        }
      });
    });
  }
  function callComponentAsTemplateResultOrResponse(result, componentFactory, props, children, route) {
    return __async(this, null, function* () {
      const factoryResult = yield componentFactory(result, props, children);
      if (factoryResult instanceof Response) {
        return factoryResult;
      } else if (isHeadAndContent(factoryResult)) {
        if (!isRenderTemplateResult(factoryResult.content)) {
          throw new AstroError(__spreadProps(__spreadValues({}, OnlyResponseCanBeReturned), {
            message: OnlyResponseCanBeReturned.message(
              route == null ? void 0 : route.route,
              typeof factoryResult
            ),
            location: {
              file: route == null ? void 0 : route.component
            }
          }));
        }
        return factoryResult.content;
      } else if (!isRenderTemplateResult(factoryResult)) {
        throw new AstroError(__spreadProps(__spreadValues({}, OnlyResponseCanBeReturned), {
          message: OnlyResponseCanBeReturned.message(route == null ? void 0 : route.route, typeof factoryResult),
          location: {
            file: route == null ? void 0 : route.component
          }
        }));
      }
      return factoryResult;
    });
  }
  function bufferHeadContent(result) {
    return __async(this, null, function* () {
      yield bufferPropagatedHead(result);
    });
  }
  function renderToAsyncIterable(result, componentFactory, props, children, isPage = false, route) {
    return __async(this, null, function* () {
      const templateResult = yield callComponentAsTemplateResultOrResponse(
        result,
        componentFactory,
        props,
        children,
        route
      );
      if (templateResult instanceof Response) return templateResult;
      let renderedFirstPageChunk = false;
      if (isPage) {
        yield bufferHeadContent(result);
      }
      let error2 = null;
      let next = null;
      const buffer2 = [];
      let renderingComplete = false;
      const iterator = {
        next() {
          return __async(this, null, function* () {
            if (result.cancelled) return { done: true, value: void 0 };
            if (next !== null) {
              yield next.promise;
            } else if (!renderingComplete && !buffer2.length) {
              next = promiseWithResolvers();
              yield next.promise;
            }
            if (!renderingComplete) {
              next = promiseWithResolvers();
            }
            if (error2) {
              throw error2;
            }
            let length = 0;
            let stringToEncode = "";
            for (let i2 = 0, len = buffer2.length; i2 < len; i2++) {
              const bufferEntry = buffer2[i2];
              if (typeof bufferEntry === "string") {
                const nextIsString = i2 + 1 < len && typeof buffer2[i2 + 1] === "string";
                stringToEncode += bufferEntry;
                if (!nextIsString) {
                  const encoded = encoder.encode(stringToEncode);
                  length += encoded.length;
                  stringToEncode = "";
                  buffer2[i2] = encoded;
                } else {
                  buffer2[i2] = "";
                }
              } else {
                length += bufferEntry.length;
              }
            }
            let mergedArray = new Uint8Array(length);
            let offset = 0;
            for (let i2 = 0, len = buffer2.length; i2 < len; i2++) {
              const item = buffer2[i2];
              if (item === "") {
                continue;
              }
              mergedArray.set(item, offset);
              offset += item.length;
            }
            buffer2.length = 0;
            const returnValue = {
              // The iterator is done when rendering has finished
              // and there are no more chunks to return.
              done: length === 0 && renderingComplete,
              value: mergedArray
            };
            return returnValue;
          });
        },
        return() {
          return __async(this, null, function* () {
            result.cancelled = true;
            return { done: true, value: void 0 };
          });
        }
      };
      const destination = {
        write(chunk) {
          if (isPage && !renderedFirstPageChunk) {
            renderedFirstPageChunk = true;
            if (!result.partial && !DOCTYPE_EXP.test(String(chunk))) {
              const doctype = result.compressHTML ? "<!DOCTYPE html>" : "<!DOCTYPE html>\n";
              buffer2.push(encoder.encode(doctype));
            }
          }
          if (chunk instanceof Response) {
            throw new AstroError(ResponseSentError);
          }
          const bytes = chunkToByteArrayOrString(result, chunk);
          if (bytes.length > 0) {
            buffer2.push(bytes);
            next == null ? void 0 : next.resolve();
          } else if (buffer2.length > 0) {
            next == null ? void 0 : next.resolve();
          }
        }
      };
      const renderResult = toPromise(() => templateResult.render(destination));
      renderResult.catch((err) => {
        error2 = err;
      }).finally(() => {
        renderingComplete = true;
        next == null ? void 0 : next.resolve();
      });
      return {
        [Symbol.asyncIterator]() {
          return iterator;
        }
      };
    });
  }
  function toPromise(fn) {
    try {
      const result = fn();
      return isPromise(result) ? result : Promise.resolve(result);
    } catch (err) {
      return Promise.reject(err);
    }
  }
  function componentIsHTMLElement(Component) {
    return typeof HTMLElement !== "undefined" && HTMLElement.isPrototypeOf(Component);
  }
  function renderHTMLElement$1(result, constructor, props, slots) {
    return __async(this, null, function* () {
      const name = getHTMLElementName(constructor);
      let attrHTML = "";
      for (const attr in props) {
        attrHTML += ` ${attr}="${toAttributeString(yield props[attr])}"`;
      }
      return markHTMLString(
        `<${name}${attrHTML}>${yield renderSlotToString(result, slots == null ? void 0 : slots.default)}</${name}>`
      );
    });
  }
  function getHTMLElementName(constructor) {
    const definedName = customElements.getName(constructor);
    if (definedName) return definedName;
    const assignedName = constructor.name.replace(/^HTML|Element$/g, "").replace(/[A-Z]/g, "-$&").toLowerCase().replace(/^-/, "html-");
    return assignedName;
  }
  function guessRenderers(componentUrl) {
    const extname = componentUrl == null ? void 0 : componentUrl.split(".").pop();
    switch (extname) {
      case "svelte":
        return ["@astrojs/svelte"];
      case "vue":
        return ["@astrojs/vue"];
      case "jsx":
      case "tsx":
        return ["@astrojs/react", "@astrojs/preact", "@astrojs/solid-js", "@astrojs/vue (jsx)"];
      case void 0:
      default:
        return [
          "@astrojs/react",
          "@astrojs/preact",
          "@astrojs/solid-js",
          "@astrojs/vue",
          "@astrojs/svelte"
        ];
    }
  }
  function isFragmentComponent(Component) {
    return Component === Fragment;
  }
  function isHTMLComponent(Component) {
    return Component && Component["astro:html"] === true;
  }
  function removeStaticAstroSlot(html, supportsAstroStaticSlot = true) {
    const exp = supportsAstroStaticSlot ? ASTRO_STATIC_SLOT_EXP : ASTRO_SLOT_EXP;
    return html.replace(exp, "");
  }
  function renderFrameworkComponent(_0, _1, _2, _3) {
    return __async(this, arguments, function* (result, displayName, Component, _props, slots = {}) {
      var _a29, _b13, _c, _d;
      if (!Component && "client:only" in _props === false) {
        throw new Error(
          `Unable to render ${displayName} because it is ${Component}!
Did you forget to import the component or is it possible there is a typo?`
        );
      }
      const { renderers: renderers2, clientDirectives } = result;
      const metadata = {
        astroStaticSlot: true,
        displayName
      };
      const { hydration, isPage, props, propsWithoutTransitionAttributes } = extractDirectives(
        _props,
        clientDirectives
      );
      let html = "";
      let attrs = void 0;
      if (hydration) {
        metadata.hydrate = hydration.directive;
        metadata.hydrateArgs = hydration.value;
        metadata.componentExport = hydration.componentExport;
        metadata.componentUrl = hydration.componentUrl;
      }
      const probableRendererNames = guessRenderers(metadata.componentUrl);
      const validRenderers = renderers2.filter((r2) => r2.name !== "astro:jsx");
      const { children, slotInstructions } = yield renderSlots(result, slots);
      let renderer;
      if (metadata.hydrate !== "only") {
        let isTagged = false;
        try {
          isTagged = Component && Component[Renderer];
        } catch (e2) {
        }
        if (isTagged) {
          const rendererName = Component[Renderer];
          renderer = renderers2.find(({ name }) => name === rendererName);
        }
        if (!renderer) {
          let error2;
          for (const r2 of renderers2) {
            try {
              if (yield r2.ssr.check.call({ result }, Component, props, children, metadata)) {
                renderer = r2;
                break;
              }
            } catch (e2) {
              error2 != null ? error2 : error2 = e2;
            }
          }
          if (!renderer && error2) {
            throw error2;
          }
        }
        if (!renderer && typeof HTMLElement === "function" && componentIsHTMLElement(Component)) {
          const output = yield renderHTMLElement$1(
            result,
            Component,
            _props,
            slots
          );
          return {
            render(destination) {
              destination.write(output);
            }
          };
        }
      } else {
        if (metadata.hydrateArgs) {
          const rendererName = rendererAliases.has(metadata.hydrateArgs) ? rendererAliases.get(metadata.hydrateArgs) : metadata.hydrateArgs;
          if (clientOnlyValues.has(rendererName)) {
            renderer = renderers2.find(
              ({ name }) => name === `@astrojs/${rendererName}` || name === rendererName
            );
          }
        }
        if (!renderer && validRenderers.length === 1) {
          renderer = validRenderers[0];
        }
        if (!renderer) {
          const extname = (_a29 = metadata.componentUrl) == null ? void 0 : _a29.split(".").pop();
          renderer = renderers2.find(({ name }) => name === `@astrojs/${extname}` || name === extname);
        }
        if (!renderer && metadata.hydrateArgs) {
          const rendererName = metadata.hydrateArgs;
          if (typeof rendererName === "string") {
            renderer = renderers2.find(({ name }) => name === rendererName);
          }
        }
      }
      if (!renderer) {
        if (metadata.hydrate === "only") {
          const rendererName = rendererAliases.has(metadata.hydrateArgs) ? rendererAliases.get(metadata.hydrateArgs) : metadata.hydrateArgs;
          if (clientOnlyValues.has(rendererName)) {
            const plural = validRenderers.length > 1;
            throw new AstroError(__spreadProps(__spreadValues({}, NoMatchingRenderer), {
              message: NoMatchingRenderer.message(
                metadata.displayName,
                (_b13 = metadata == null ? void 0 : metadata.componentUrl) == null ? void 0 : _b13.split(".").pop(),
                plural,
                validRenderers.length
              ),
              hint: NoMatchingRenderer.hint(
                formatList(probableRendererNames.map((r2) => "`" + r2 + "`"))
              )
            }));
          } else {
            throw new AstroError(__spreadProps(__spreadValues({}, NoClientOnlyHint), {
              message: NoClientOnlyHint.message(metadata.displayName),
              hint: NoClientOnlyHint.hint(
                probableRendererNames.map((r2) => r2.replace("@astrojs/", "")).join("|")
              )
            }));
          }
        } else if (typeof Component !== "string") {
          const matchingRenderers = validRenderers.filter(
            (r2) => probableRendererNames.includes(r2.name)
          );
          const plural = validRenderers.length > 1;
          if (matchingRenderers.length === 0) {
            throw new AstroError(__spreadProps(__spreadValues({}, NoMatchingRenderer), {
              message: NoMatchingRenderer.message(
                metadata.displayName,
                (_c = metadata == null ? void 0 : metadata.componentUrl) == null ? void 0 : _c.split(".").pop(),
                plural,
                validRenderers.length
              ),
              hint: NoMatchingRenderer.hint(
                formatList(probableRendererNames.map((r2) => "`" + r2 + "`"))
              )
            }));
          } else if (matchingRenderers.length === 1) {
            renderer = matchingRenderers[0];
            ({ html, attrs } = yield renderer.ssr.renderToStaticMarkup.call(
              { result },
              Component,
              propsWithoutTransitionAttributes,
              children,
              metadata
            ));
          } else {
            throw new Error(`Unable to render ${metadata.displayName}!

This component likely uses ${formatList(probableRendererNames)},
but Astro encountered an error during server-side rendering.

Please ensure that ${metadata.displayName}:
1. Does not unconditionally access browser-specific globals like \`window\` or \`document\`.
   If this is unavoidable, use the \`client:only\` hydration directive.
2. Does not conditionally return \`null\` or \`undefined\` when rendered on the server.
3. If using multiple JSX frameworks at the same time (e.g. React + Preact), pass the correct \`include\`/\`exclude\` options to integrations.

If you're still stuck, please open an issue on GitHub or join us at https://astro.build/chat.`);
          }
        }
      } else {
        if (metadata.hydrate === "only") {
          html = yield renderSlotToString(result, slots == null ? void 0 : slots.fallback);
        } else {
          performance.now();
          ({ html, attrs } = yield renderer.ssr.renderToStaticMarkup.call(
            { result },
            Component,
            propsWithoutTransitionAttributes,
            children,
            metadata
          ));
        }
      }
      if (!html && typeof Component === "string") {
        const Tag = sanitizeElementName(Component);
        const childSlots = Object.values(children).join("");
        const renderTemplateResult = renderTemplate`<${Tag}${internalSpreadAttributes(
          props,
          true,
          Tag
        )}${markHTMLString(
          childSlots === "" && voidElementNames.test(Tag) ? `/>` : `>${childSlots}</${Tag}>`
        )}`;
        html = "";
        const destination = {
          write(chunk) {
            if (chunk instanceof Response) return;
            html += chunkToString(result, chunk);
          }
        };
        yield renderTemplateResult.render(destination);
      }
      if (!hydration) {
        return {
          render(destination) {
            var _a30;
            if (slotInstructions) {
              for (const instruction of slotInstructions) {
                destination.write(instruction);
              }
            }
            if (isPage || (renderer == null ? void 0 : renderer.name) === "astro:jsx") {
              destination.write(html);
            } else if (html && html.length > 0) {
              destination.write(
                markHTMLString(removeStaticAstroSlot(html, (_a30 = renderer == null ? void 0 : renderer.ssr) == null ? void 0 : _a30.supportsAstroStaticSlot))
              );
            }
          }
        };
      }
      const astroId = shorthash(
        `<!--${metadata.componentExport.value}:${metadata.componentUrl}-->
${html}
${serializeProps(
          props,
          metadata
        )}`
      );
      const island = yield generateHydrateScript(
        { renderer, result, astroId, props, attrs },
        metadata
      );
      let unrenderedSlots = [];
      if (html) {
        if (Object.keys(children).length > 0) {
          for (const key of Object.keys(children)) {
            let tagName = ((_d = renderer == null ? void 0 : renderer.ssr) == null ? void 0 : _d.supportsAstroStaticSlot) ? !!metadata.hydrate ? "astro-slot" : "astro-static-slot" : "astro-slot";
            let expectedHTML = key === "default" ? `<${tagName}>` : `<${tagName} name="${escapeHTML(key)}">`;
            if (!html.includes(expectedHTML)) {
              unrenderedSlots.push(key);
            }
          }
        }
      } else {
        unrenderedSlots = Object.keys(children);
      }
      const template2 = unrenderedSlots.length > 0 ? unrenderedSlots.map(
        (key) => `<template data-astro-template${key !== "default" ? `="${escapeHTML(key)}"` : ""}>${children[key]}</template>`
      ).join("") : "";
      island.children = `${html != null ? html : ""}${template2}`;
      if (island.children) {
        island.props["await-children"] = "";
        island.children += `<!--astro:end-->`;
      }
      return {
        render(destination) {
          if (slotInstructions) {
            for (const instruction of slotInstructions) {
              destination.write(instruction);
            }
          }
          destination.write(createRenderInstruction({ type: "directive", hydration }));
          if (hydration.directive !== "only" && (renderer == null ? void 0 : renderer.ssr.renderHydrationScript)) {
            destination.write(
              createRenderInstruction({
                type: "renderer-hydration-script",
                rendererName: renderer.name,
                render: renderer.ssr.renderHydrationScript
              })
            );
          }
          const renderedElement = renderElement$1("astro-island", island, false);
          destination.write(markHTMLString(renderedElement));
        }
      };
    });
  }
  function sanitizeElementName(tag) {
    const unsafe = /[&<>'"\s]+/;
    if (!unsafe.test(tag)) return tag;
    return tag.trim().split(unsafe)[0].trim();
  }
  function renderFragmentComponent(result, slots = {}) {
    const slot = slots == null ? void 0 : slots.default;
    const preRendered = slot == null ? void 0 : slot(result);
    return {
      render(destination) {
        if (preRendered == null) return;
        return renderChild(destination, preRendered);
      }
    };
  }
  function renderHTMLComponent(_0, _1, _2) {
    return __async(this, arguments, function* (result, Component, _props, slots = {}) {
      const { slotInstructions, children } = yield renderSlots(result, slots);
      const html = Component({ slots: children });
      const hydrationHtml = slotInstructions ? slotInstructions.map((instr) => chunkToString(result, instr)).join("") : "";
      return {
        render(destination) {
          destination.write(markHTMLString(hydrationHtml + html));
        }
      };
    });
  }
  function renderAstroComponent(result, displayName, Component, props, slots = {}) {
    if (containsServerDirective(props)) {
      const serverIslandComponent = new ServerIslandComponent(result, props, slots, displayName);
      result._metadata.propagators.add(serverIslandComponent);
      return serverIslandComponent;
    }
    const instance = createAstroComponentInstance(result, displayName, Component, props, slots);
    return {
      render(destination) {
        return instance.render(destination);
      }
    };
  }
  function renderComponent(result, displayName, Component, props, slots = {}) {
    if (isPromise(Component)) {
      return Component.catch(handleCancellation).then((x) => {
        return renderComponent(result, displayName, x, props, slots);
      });
    }
    if (isFragmentComponent(Component)) {
      return renderFragmentComponent(result, slots);
    }
    props = normalizeProps(props);
    if (isHTMLComponent(Component)) {
      return renderHTMLComponent(result, Component, props, slots).catch(handleCancellation);
    }
    if (isAstroComponentFactory(Component)) {
      return renderAstroComponent(result, displayName, Component, props, slots);
    }
    return renderFrameworkComponent(result, displayName, Component, props, slots).catch(
      handleCancellation
    );
    function handleCancellation(e2) {
      if (result.cancelled)
        return {
          render() {
          }
        };
      throw e2;
    }
  }
  function normalizeProps(props) {
    if (props["class:list"] !== void 0) {
      const value = props["class:list"];
      delete props["class:list"];
      props["class"] = clsx(props["class"], value);
      if (props["class"] === "") {
        delete props["class"];
      }
    }
    return props;
  }
  function renderComponentToString(_0, _1, _2, _3) {
    return __async(this, arguments, function* (result, displayName, Component, props, slots = {}, isPage = false, route) {
      let str = "";
      let renderedFirstPageChunk = false;
      let head = "";
      if (isPage && !result.partial && nonAstroPageNeedsHeadInjection(Component)) {
        head += chunkToString(result, maybeRenderHead());
      }
      try {
        const destination = {
          write(chunk) {
            if (isPage && !result.partial && !renderedFirstPageChunk) {
              renderedFirstPageChunk = true;
              if (!/<!doctype html/i.test(String(chunk))) {
                const doctype = result.compressHTML ? "<!DOCTYPE html>" : "<!DOCTYPE html>\n";
                str += doctype + head;
              }
            }
            if (chunk instanceof Response) return;
            str += chunkToString(result, chunk);
          }
        };
        const renderInstance = yield renderComponent(result, displayName, Component, props, slots);
        if (containsServerDirective(props)) {
          yield bufferHeadContent(result);
        }
        yield renderInstance.render(destination);
      } catch (e2) {
        if (AstroError.is(e2) && !e2.loc) {
          e2.setLocation({
            file: route == null ? void 0 : route.component
          });
        }
        throw e2;
      }
      return str;
    });
  }
  function nonAstroPageNeedsHeadInjection(pageComponent) {
    return !!(pageComponent == null ? void 0 : pageComponent[needsHeadRenderingSymbol]);
  }
  function renderJSX(result, vnode) {
    return __async(this, null, function* () {
      switch (true) {
        case vnode instanceof HTMLString:
          if (vnode.toString().trim() === "") {
            return "";
          }
          return vnode;
        case typeof vnode === "string":
          return markHTMLString(escapeHTML(vnode));
        case typeof vnode === "function":
          return vnode;
        case (!vnode && vnode !== 0):
          return "";
        case Array.isArray(vnode): {
          const renderedItems = yield Promise.all(vnode.map((v) => renderJSX(result, v)));
          let instructions = null;
          let content = "";
          for (const item of renderedItems) {
            if (item instanceof SlotString) {
              content += item;
              instructions = mergeSlotInstructions(instructions, item);
            } else {
              content += item;
            }
          }
          if (instructions) {
            return markHTMLString(new SlotString(content, instructions));
          }
          return markHTMLString(content);
        }
      }
      return renderJSXVNode(result, vnode);
    });
  }
  function renderJSXVNode(result, vnode) {
    return __async(this, null, function* () {
      var _a29, _b13, _c, _d, _e, _g;
      if (isVNode(vnode)) {
        switch (true) {
          case !vnode.type: {
            throw new Error(`Unable to render ${result.pathname} because it contains an undefined Component!
Did you forget to import the component or is it possible there is a typo?`);
          }
          case vnode.type === /* @__PURE__ */ Symbol.for("astro:fragment"):
            return renderJSX(result, vnode.props.children);
          case isAstroComponentFactory(vnode.type): {
            let props = {};
            let slots = {};
            for (const [key, value] of Object.entries((_a29 = vnode.props) != null ? _a29 : {})) {
              if (key === "children" || value && typeof value === "object" && value["$$slot"]) {
                slots[key === "children" ? "default" : key] = () => renderJSX(result, value);
              } else {
                props[key] = value;
              }
            }
            const str = yield renderComponentToString(
              result,
              vnode.type.name,
              vnode.type,
              props,
              slots
            );
            const html = markHTMLString(str);
            return html;
          }
          case (!vnode.type && vnode.type !== 0):
            return "";
          case (typeof vnode.type === "string" && vnode.type !== ClientOnlyPlaceholder$1 && !vnode.type.includes("-")):
            return markHTMLString(yield renderElement(result, vnode.type, (_b13 = vnode.props) != null ? _b13 : {}));
        }
        if (vnode.type) {
          let extractSlots2 = function(child) {
            var _a30;
            if (Array.isArray(child)) {
              return child.map((c) => extractSlots2(c));
            }
            if (!isVNode(child)) {
              _slots2.default.push(child);
              return;
            }
            if ("slot" in child.props && !isCustomElement2) {
              _slots2[child.props.slot] = [...(_a30 = _slots2[child.props.slot]) != null ? _a30 : [], child];
              delete child.props.slot;
              return;
            }
            _slots2.default.push(child);
          };
          if (typeof vnode.type === "function" && vnode.props["server:root"]) {
            const output2 = yield vnode.type((_c = vnode.props) != null ? _c : {});
            return yield renderJSX(result, output2);
          }
          if (typeof vnode.type === "function") {
            if (vnode.props[hasTriedRenderComponentSymbol]) {
              delete vnode.props[hasTriedRenderComponentSymbol];
              const output2 = yield vnode.type((_d = vnode.props) != null ? _d : {});
              if ((output2 == null ? void 0 : output2[AstroJSX]) || !output2) {
                return yield renderJSXVNode(result, output2);
              } else {
                return;
              }
            } else {
              vnode.props[hasTriedRenderComponentSymbol] = true;
            }
          }
          const _f = (_e = vnode.props) != null ? _e : {}, { children = null } = _f, props = __objRest(_f, ["children"]);
          const _slots2 = {
            default: []
          };
          const isCustomElement2 = typeof vnode.type === "string" && vnode.type.includes("-");
          extractSlots2(children);
          for (const [key, value] of Object.entries(props)) {
            if (value == null ? void 0 : value["$$slot"]) {
              _slots2[key] = value;
              delete props[key];
            }
          }
          const slotPromises = [];
          const slots = {};
          for (const [key, value] of Object.entries(_slots2)) {
            slotPromises.push(
              renderJSX(result, value).then((output2) => {
                if (output2.toString().trim().length === 0) return;
                slots[key] = () => output2;
              })
            );
          }
          yield Promise.all(slotPromises);
          let output;
          if (vnode.type === ClientOnlyPlaceholder$1 && vnode.props["client:only"]) {
            output = yield renderComponentToString(
              result,
              (_g = vnode.props["client:display-name"]) != null ? _g : "",
              null,
              props,
              slots
            );
          } else {
            output = yield renderComponentToString(
              result,
              typeof vnode.type === "function" ? vnode.type.name : vnode.type,
              vnode.type,
              props,
              slots
            );
          }
          return markHTMLString(output);
        }
      }
      return markHTMLString(`${vnode}`);
    });
  }
  function renderElement(result, tag, _a29) {
    return __async(this, null, function* () {
      var _b13 = _a29, { children } = _b13, props = __objRest(_b13, ["children"]);
      return markHTMLString(
        `<${tag}${spreadAttributes(props)}${markHTMLString(
          (children == null || children === "") && voidElementNames.test(tag) ? `/>` : `>${children == null ? "" : yield renderJSX(result, prerenderElementChildren$1(tag, children))}</${tag}>`
        )}`
      );
    });
  }
  function prerenderElementChildren$1(tag, children) {
    if (typeof children === "string" && (tag === "style" || tag === "script")) {
      return markHTMLString(children);
    } else {
      return children;
    }
  }
  function renderJSXToQueue(vnode, result, queue, pool, stack, parent, metadata) {
    if (vnode instanceof HTMLString) {
      const html = vnode.toString();
      if (html.trim() === "") return;
      const node = pool.acquire("html-string", html);
      node.html = html;
      queue.nodes.push(node);
      return;
    }
    if (typeof vnode === "string") {
      const node = pool.acquire("text", vnode);
      node.content = vnode;
      queue.nodes.push(node);
      return;
    }
    if (typeof vnode === "number" || typeof vnode === "boolean") {
      const str = String(vnode);
      const node = pool.acquire("text", str);
      node.content = str;
      queue.nodes.push(node);
      return;
    }
    if (vnode == null || vnode === false) {
      return;
    }
    if (Array.isArray(vnode)) {
      for (let i2 = vnode.length - 1; i2 >= 0; i2 = i2 - 1) {
        stack.push({ node: vnode[i2], parent, metadata });
      }
      return;
    }
    if (!isVNode(vnode)) {
      const str = String(vnode);
      const node = pool.acquire("text", str);
      node.content = str;
      queue.nodes.push(node);
      return;
    }
    handleVNode(vnode, result, queue, pool, stack, parent, metadata);
  }
  function handleVNode(vnode, result, queue, pool, stack, parent, metadata) {
    var _a29, _b13, _c, _d, _e;
    if (!vnode.type) {
      throw new Error(
        `Unable to render ${result.pathname} because it contains an undefined Component!
Did you forget to import the component or is it possible there is a typo?`
      );
    }
    if (vnode.type === /* @__PURE__ */ Symbol.for("astro:fragment")) {
      stack.push({ node: (_a29 = vnode.props) == null ? void 0 : _a29.children, parent, metadata });
      return;
    }
    if (isAstroComponentFactory(vnode.type)) {
      const factory = vnode.type;
      let props = {};
      let slots = {};
      for (const [key, value] of Object.entries((_b13 = vnode.props) != null ? _b13 : {})) {
        if (key === "children" || value && typeof value === "object" && value["$$slot"]) {
          slots[key === "children" ? "default" : key] = () => renderJSX(result, value);
        } else {
          props[key] = value;
        }
      }
      const displayName = (metadata == null ? void 0 : metadata.displayName) || factory.name || "Anonymous";
      const instance = createAstroComponentInstance(result, displayName, factory, props, slots);
      const queueNode = pool.acquire("component");
      queueNode.instance = instance;
      queue.nodes.push(queueNode);
      return;
    }
    if (typeof vnode.type === "string" && vnode.type !== ClientOnlyPlaceholder) {
      renderHTMLElement(vnode, result, queue, pool, stack, parent, metadata);
      return;
    }
    if (typeof vnode.type === "function") {
      if ((_c = vnode.props) == null ? void 0 : _c["server:root"]) {
        const output3 = vnode.type((_d = vnode.props) != null ? _d : {});
        stack.push({ node: output3, parent, metadata });
        return;
      }
      const output2 = vnode.type((_e = vnode.props) != null ? _e : {});
      stack.push({ node: output2, parent, metadata });
      return;
    }
    const output = renderJSX(result, vnode);
    stack.push({ node: output, parent, metadata });
  }
  function renderHTMLElement(vnode, _result2, queue, pool, stack, parent, metadata) {
    var _a29;
    const tag = vnode.type;
    const _b13 = (_a29 = vnode.props) != null ? _a29 : {}, { children } = _b13, props = __objRest(_b13, ["children"]);
    const attrs = spreadAttributes(props);
    const isVoidElement = (children == null || children === "") && voidElementNames.test(tag);
    if (isVoidElement) {
      const html = `<${tag}${attrs}/>`;
      const node = pool.acquire("html-string", html);
      node.html = html;
      queue.nodes.push(node);
      return;
    }
    const openTag = `<${tag}${attrs}>`;
    const openTagHtml = queue.htmlStringCache ? queue.htmlStringCache.getOrCreate(openTag) : markHTMLString(openTag);
    stack.push({ node: openTagHtml, parent, metadata });
    if (children != null && children !== "") {
      const processedChildren = prerenderElementChildren(tag, children, queue.htmlStringCache);
      stack.push({ node: processedChildren, parent, metadata });
    }
    const closeTag = `</${tag}>`;
    const closeTagHtml = queue.htmlStringCache ? queue.htmlStringCache.getOrCreate(closeTag) : markHTMLString(closeTag);
    stack.push({ node: closeTagHtml, parent, metadata });
  }
  function prerenderElementChildren(tag, children, htmlStringCache) {
    if (typeof children === "string" && (tag === "style" || tag === "script")) {
      return htmlStringCache ? htmlStringCache.getOrCreate(children) : markHTMLString(children);
    }
    return children;
  }
  function buildRenderQueue(root, result, pool) {
    return __async(this, null, function* () {
      var _a29, _b13, _c, _d;
      const queue = {
        nodes: [],
        result,
        pool,
        htmlStringCache: (_a29 = result._experimentalQueuedRendering) == null ? void 0 : _a29.htmlStringCache
      };
      const stack = [{ node: root, parent: null }];
      while (stack.length > 0) {
        const item = stack.pop();
        if (!item) {
          continue;
        }
        let { node, parent } = item;
        if (isPromise(node)) {
          try {
            const resolved = yield node;
            stack.push({ node: resolved, parent, metadata: item.metadata });
          } catch (error22) {
            throw error22;
          }
          continue;
        }
        if (node == null || node === false) {
          continue;
        }
        if (typeof node === "string") {
          const queueNode = pool.acquire("text", node);
          queueNode.content = node;
          queue.nodes.push(queueNode);
          continue;
        }
        if (typeof node === "number" || typeof node === "boolean") {
          const str = String(node);
          const queueNode = pool.acquire("text", str);
          queueNode.content = str;
          queue.nodes.push(queueNode);
          continue;
        }
        if (isHTMLString(node)) {
          const html = node.toString();
          const queueNode = pool.acquire("html-string", html);
          queueNode.html = html;
          queue.nodes.push(queueNode);
          continue;
        }
        if (node instanceof SlotString) {
          const html = node.toString();
          const queueNode = pool.acquire("html-string", html);
          queueNode.html = html;
          queue.nodes.push(queueNode);
          continue;
        }
        if (isVNode(node)) {
          renderJSXToQueue(node, result, queue, pool, stack, parent, item.metadata);
          continue;
        }
        if (Array.isArray(node)) {
          for (const n2 of node) {
            stack.push({ node: n2, parent, metadata: item.metadata });
          }
          continue;
        }
        if (isRenderInstruction(node)) {
          const queueNode = pool.acquire("instruction");
          queueNode.instruction = node;
          queue.nodes.push(queueNode);
          continue;
        }
        if (isRenderTemplateResult(node)) {
          const htmlParts = node["htmlParts"];
          const expressions = node["expressions"];
          if (htmlParts[0]) {
            const htmlString = queue.htmlStringCache ? queue.htmlStringCache.getOrCreate(htmlParts[0]) : markHTMLString(htmlParts[0]);
            stack.push({
              node: htmlString,
              parent,
              metadata: item.metadata
            });
          }
          for (let i2 = 0; i2 < expressions.length; i2 = i2 + 1) {
            stack.push({ node: expressions[i2], parent, metadata: item.metadata });
            if (htmlParts[i2 + 1]) {
              const htmlString = queue.htmlStringCache ? queue.htmlStringCache.getOrCreate(htmlParts[i2 + 1]) : markHTMLString(htmlParts[i2 + 1]);
              stack.push({
                node: htmlString,
                parent,
                metadata: item.metadata
              });
            }
          }
          continue;
        }
        if (isAstroComponentInstance(node)) {
          const queueNode = pool.acquire("component");
          queueNode.instance = node;
          queue.nodes.push(queueNode);
          continue;
        }
        if (isAstroComponentFactory(node)) {
          const factory = node;
          const props = ((_b13 = item.metadata) == null ? void 0 : _b13.props) || {};
          const slots = ((_c = item.metadata) == null ? void 0 : _c.slots) || {};
          const displayName = ((_d = item.metadata) == null ? void 0 : _d.displayName) || factory.name || "Anonymous";
          const instance = createAstroComponentInstance(result, displayName, factory, props, slots);
          const queueNode = pool.acquire("component");
          queueNode.instance = instance;
          if (isAPropagatingComponent(result, factory)) {
            try {
              const returnValue = yield instance.init(result);
              if (isHeadAndContent(returnValue) && returnValue.head) {
                result._metadata.extraHead.push(returnValue.head);
              }
            } catch (error22) {
              throw error22;
            }
          }
          queue.nodes.push(queueNode);
          continue;
        }
        if (isRenderInstance(node)) {
          const queueNode = pool.acquire("component");
          queueNode.instance = node;
          queue.nodes.push(queueNode);
          continue;
        }
        if (typeof node === "object" && Symbol.iterator in node) {
          const items = Array.from(node);
          for (const iterItem of items) {
            stack.push({ node: iterItem, parent, metadata: item.metadata });
          }
          continue;
        }
        if (typeof node === "object" && Symbol.asyncIterator in node) {
          try {
            const items = [];
            try {
              for (var iter = __forAwait(node), more, temp, error2; more = !(temp = yield iter.next()).done; more = false) {
                const asyncItem = temp.value;
                items.push(asyncItem);
              }
            } catch (temp) {
              error2 = [temp];
            } finally {
              try {
                more && (temp = iter.return) && (yield temp.call(iter));
              } finally {
                if (error2)
                  throw error2[0];
              }
            }
            for (const iterItem of items) {
              stack.push({ node: iterItem, parent, metadata: item.metadata });
            }
          } catch (error22) {
            throw error22;
          }
          continue;
        }
        if (node instanceof Response) {
          const queueNode = pool.acquire("html-string", "");
          queueNode.html = "";
          queue.nodes.push(queueNode);
          continue;
        }
        if (isHTMLString(node)) {
          const html = String(node);
          const queueNode = pool.acquire("html-string", html);
          queueNode.html = html;
          queue.nodes.push(queueNode);
        } else {
          const str = String(node);
          const queueNode = pool.acquire("text", str);
          queueNode.content = str;
          queue.nodes.push(queueNode);
        }
      }
      queue.nodes.reverse();
      return queue;
    });
  }
  function renderQueue(queue, destination) {
    return __async(this, null, function* () {
      const result = queue.result;
      const pool = queue.pool;
      const cache = queue.htmlStringCache;
      let batchBuffer = "";
      let i2 = 0;
      while (i2 < queue.nodes.length) {
        const node = queue.nodes[i2];
        try {
          if (canBatch(node)) {
            const batchStart = i2;
            while (i2 < queue.nodes.length && canBatch(queue.nodes[i2])) {
              batchBuffer += renderNodeToString(queue.nodes[i2]);
              i2 = i2 + 1;
            }
            if (batchBuffer) {
              const htmlString = cache ? cache.getOrCreate(batchBuffer) : markHTMLString(batchBuffer);
              destination.write(htmlString);
              batchBuffer = "";
            }
            if (pool) {
              for (let j = batchStart; j < i2; j++) {
                pool.release(queue.nodes[j]);
              }
            }
          } else {
            yield renderNode(node, destination, result);
            if (pool) {
              pool.release(node);
            }
            i2 = i2 + 1;
          }
        } catch (error2) {
          throw error2;
        }
      }
      if (batchBuffer) {
        const htmlString = cache ? cache.getOrCreate(batchBuffer) : markHTMLString(batchBuffer);
        destination.write(htmlString);
      }
    });
  }
  function canBatch(node) {
    return node.type === "text" || node.type === "html-string";
  }
  function renderNodeToString(node) {
    switch (node.type) {
      case "text":
        return node.content ? escapeHTML(node.content) : "";
      case "html-string":
        return node.html || "";
      case "component":
      case "instruction": {
        return "";
      }
    }
  }
  function renderNode(node, destination, result) {
    return __async(this, null, function* () {
      var _a29;
      const cache = (_a29 = result._experimentalQueuedRendering) == null ? void 0 : _a29.htmlStringCache;
      switch (node.type) {
        case "text": {
          if (node.content) {
            const escaped = escapeHTML(node.content);
            const htmlString = cache ? cache.getOrCreate(escaped) : markHTMLString(escaped);
            destination.write(htmlString);
          }
          break;
        }
        case "html-string": {
          if (node.html) {
            const htmlString = cache ? cache.getOrCreate(node.html) : markHTMLString(node.html);
            destination.write(htmlString);
          }
          break;
        }
        case "instruction": {
          if (node.instruction) {
            destination.write(node.instruction);
          }
          break;
        }
        case "component": {
          if (node.instance) {
            let componentHtml = "";
            const componentDestination = {
              write(chunk) {
                if (chunk instanceof Response) return;
                componentHtml += chunkToString(result, chunk);
              }
            };
            yield node.instance.render(componentDestination);
            if (componentHtml) {
              destination.write(componentHtml);
            }
          }
          break;
        }
      }
    });
  }
  function renderPage(result, componentFactory, props, children, streaming, route) {
    return __async(this, null, function* () {
      var _a29, _b13, _c, _d;
      if (!isAstroComponentFactory(componentFactory)) {
        result._metadata.headInTree = (_b13 = (_a29 = result.componentMetadata.get(componentFactory.moduleId)) == null ? void 0 : _a29.containsHead) != null ? _b13 : false;
        const pageProps = __spreadProps(__spreadValues({}, props != null ? props : {}), { "server:root": true });
        let str;
        if (result._experimentalQueuedRendering && result._experimentalQueuedRendering.enabled) {
          let vnode = yield componentFactory(pageProps);
          if (componentFactory["astro:html"] && typeof vnode === "string") {
            vnode = markHTMLString(vnode);
          }
          const queue = yield buildRenderQueue(
            vnode,
            result,
            result._experimentalQueuedRendering.pool
          );
          let html = "";
          let renderedFirst = false;
          const destination = {
            write(chunk) {
              if (chunk instanceof Response) return;
              if (!renderedFirst && !result.partial) {
                renderedFirst = true;
                const chunkStr = String(chunk);
                if (!/<!doctype html/i.test(chunkStr)) {
                  const doctype = result.compressHTML ? "<!DOCTYPE html>" : "<!DOCTYPE html>\n";
                  html += doctype;
                }
              }
              html += chunkToString(result, chunk);
            }
          };
          yield renderQueue(queue, destination);
          str = html;
        } else {
          str = yield renderComponentToString(
            result,
            componentFactory.name,
            componentFactory,
            pageProps,
            {},
            true,
            route
          );
        }
        const bytes = encoder.encode(str);
        const headers2 = new Headers([
          ["Content-Type", "text/html"],
          ["Content-Length", bytes.byteLength.toString()]
        ]);
        if (result.shouldInjectCspMetaTags && (result.cspDestination === "header" || result.cspDestination === "adapter")) {
          headers2.set("content-security-policy", renderCspContent(result));
        }
        return new Response(bytes, {
          headers: headers2,
          status: result.response.status
        });
      }
      result._metadata.headInTree = (_d = (_c = result.componentMetadata.get(componentFactory.moduleId)) == null ? void 0 : _c.containsHead) != null ? _d : false;
      let body;
      if (streaming) {
        if (isNode && !isDeno) {
          const nodeBody = yield renderToAsyncIterable(
            result,
            componentFactory,
            props,
            children,
            true,
            route
          );
          body = nodeBody;
        } else {
          body = yield renderToReadableStream(result, componentFactory, props, children, true, route);
        }
      } else {
        body = yield renderToString(result, componentFactory, props, children, true, route);
      }
      if (body instanceof Response) return body;
      const init = result.response;
      const headers = new Headers(init.headers);
      if (result.shouldInjectCspMetaTags && result.cspDestination === "header" || result.cspDestination === "adapter") {
        headers.set("content-security-policy", renderCspContent(result));
      }
      if (!streaming && typeof body === "string") {
        body = encoder.encode(body);
        headers.set("Content-Length", body.byteLength.toString());
      }
      let status = init.status;
      let statusText = init.statusText;
      if ((route == null ? void 0 : route.route) === "/404") {
        status = 404;
        if (statusText === "OK") {
          statusText = "Not Found";
        }
      } else if ((route == null ? void 0 : route.route) === "/500") {
        status = 500;
        if (statusText === "OK") {
          statusText = "Internal Server Error";
        }
      }
      if (status) {
        return new Response(body, __spreadProps(__spreadValues({}, init), { headers, status, statusText }));
      } else {
        return new Response(body, __spreadProps(__spreadValues({}, init), { headers }));
      }
    });
  }
  function spreadAttributes(values = {}, _name, { class: scopedClassName } = {}) {
    let output = "";
    if (scopedClassName) {
      if (typeof values.class !== "undefined") {
        values.class += ` ${scopedClassName}`;
      } else if (typeof values["class:list"] !== "undefined") {
        values["class:list"] = [values["class:list"], scopedClassName];
      } else {
        values.class = scopedClassName;
      }
    }
    for (const [key, value] of Object.entries(values)) {
      output += addAttribute(value, key, true, _name);
    }
    return markHTMLString(output);
  }
  function getPattern(segments, base, addTrailingSlash) {
    const pathname = segments.map((segment) => {
      if (segment.length === 1 && segment[0].spread) {
        return "(?:\\/(.*?))?";
      } else {
        return "\\/" + segment.map((part) => {
          if (part.spread) {
            return "(.*?)";
          } else if (part.dynamic) {
            return "([^/]+?)";
          } else {
            return part.content.normalize().replace(/\?/g, "%3F").replace(/#/g, "%23").replace(/%5B/g, "[").replace(/%5D/g, "]").replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
          }
        }).join("");
      }
    }).join("");
    const trailing = addTrailingSlash && segments.length ? getTrailingSlashPattern(addTrailingSlash) : "$";
    let initial = "\\/";
    if (addTrailingSlash === "never" && base !== "/" && pathname !== "") {
      initial = "";
    }
    return new RegExp(`^${pathname || initial}${trailing}`);
  }
  function getTrailingSlashPattern(addTrailingSlash) {
    if (addTrailingSlash === "always") {
      return "\\/$";
    }
    if (addTrailingSlash === "never") {
      return "$";
    }
    return "\\/?$";
  }
  function badRequest(reason) {
    return new Response(null, {
      status: 400,
      statusText: "Bad request: " + reason
    });
  }
  function getRequestData(_0) {
    return __async(this, arguments, function* (request, bodySizeLimit = DEFAULT_BODY_SIZE_LIMIT) {
      switch (request.method) {
        case "GET": {
          const url = new URL(request.url);
          const params = url.searchParams;
          if (!params.has("s") || !params.has("e") || !params.has("p")) {
            return badRequest("Missing required query parameters.");
          }
          const encryptedSlots = params.get("s");
          return {
            encryptedComponentExport: params.get("e"),
            encryptedProps: params.get("p"),
            encryptedSlots
          };
        }
        case "POST": {
          try {
            const body = yield readBodyWithLimit(request, bodySizeLimit);
            const raw = new TextDecoder().decode(body);
            const data = JSON.parse(raw);
            if (Object.hasOwn(data, "slots") && typeof data.slots === "object") {
              return badRequest("Plaintext slots are not allowed. Slots must be encrypted.");
            }
            if (Object.hasOwn(data, "componentExport") && typeof data.componentExport === "string") {
              return badRequest(
                "Plaintext componentExport is not allowed. componentExport must be encrypted."
              );
            }
            return data;
          } catch (e2) {
            if (e2 instanceof BodySizeLimitError) {
              return new Response(null, {
                status: 413,
                statusText: e2.message
              });
            }
            if (e2 instanceof SyntaxError) {
              return badRequest("Request format is invalid.");
            }
            throw e2;
          }
        }
        default: {
          return new Response(null, { status: 405 });
        }
      }
    });
  }
  function createEndpoint(manifest2) {
    const page3 = (result) => __async(null, null, function* () {
      var _a29;
      const params = result.params;
      if (!params.name) {
        return new Response(null, {
          status: 400,
          statusText: "Bad request"
        });
      }
      const componentId = params.name;
      const data = yield getRequestData(result.request, manifest2.serverIslandBodySizeLimit);
      if (data instanceof Response) {
        return data;
      }
      const serverIslandMappings = yield (_a29 = manifest2.serverIslandMappings) == null ? void 0 : _a29.call(manifest2);
      const serverIslandMap2 = yield serverIslandMappings == null ? void 0 : serverIslandMappings.serverIslandMap;
      let imp = serverIslandMap2 == null ? void 0 : serverIslandMap2.get(componentId);
      if (!imp) {
        return new Response(null, {
          status: 404,
          statusText: "Not found"
        });
      }
      const key = yield manifest2.key;
      let componentExport;
      try {
        componentExport = yield decryptString(
          key,
          data.encryptedComponentExport,
          `export:${componentId}`
        );
      } catch (_e) {
        return badRequest("Encrypted componentExport value is invalid.");
      }
      const encryptedProps = data.encryptedProps;
      let props = {};
      if (encryptedProps !== "") {
        try {
          const propString = yield decryptString(key, encryptedProps, `props:${componentId}`);
          props = JSON.parse(propString);
        } catch (_e) {
          return badRequest("Encrypted props value is invalid.");
        }
      }
      let decryptedSlots = {};
      const encryptedSlots = data.encryptedSlots;
      if (encryptedSlots !== "") {
        try {
          const slotsString = yield decryptString(key, encryptedSlots, `slots:${componentId}`);
          decryptedSlots = JSON.parse(slotsString);
        } catch (_e) {
          return badRequest("Encrypted slots value is invalid.");
        }
      }
      const componentModule = yield imp();
      let Component = componentModule[componentExport];
      const slots = {};
      for (const prop in decryptedSlots) {
        slots[prop] = createSlotValueFromString(decryptedSlots[prop]);
      }
      result.response.headers.set("X-Robots-Tag", "noindex");
      if (isAstroComponentFactory(Component)) {
        const ServerIsland = Component;
        Component = function(...args) {
          return ServerIsland.apply(this, args);
        };
        Object.assign(Component, ServerIsland);
        Component.propagation = "self";
      }
      return renderTemplate`${renderComponent(result, "Component", Component, props, slots)}`;
    });
    page3.isAstroComponentFactory = true;
    const instance = {
      default: page3,
      partial: true
    };
    return instance;
  }
  function createDefaultRoutes(manifest2) {
    const root = new URL(manifest2.rootDir);
    return [
      {
        instance: default404Instance,
        matchesComponent: (filePath) => filePath.href === new URL(DEFAULT_404_COMPONENT, root).href,
        route: DEFAULT_404_ROUTE.route,
        component: DEFAULT_404_COMPONENT
      },
      {
        instance: createEndpoint(manifest2),
        matchesComponent: (filePath) => filePath.href === new URL(SERVER_ISLAND_COMPONENT, root).href,
        route: SERVER_ISLAND_ROUTE,
        component: SERVER_ISLAND_COMPONENT
      }
    ];
  }
  function ensure404Route(manifest2) {
    if (!manifest2.routes.some((route) => route.route === "/404")) {
      manifest2.routes.push(DEFAULT_404_ROUTE);
    }
    return manifest2;
  }
  function routeIsRedirect(route) {
    return (route == null ? void 0 : route.type) === "redirect";
  }
  function routeIsFallback(route) {
    return (route == null ? void 0 : route.type) === "fallback";
  }
  function getFallbackRoute(route, routeList) {
    const fallbackRoute = routeList.find((r2) => {
      if (route.route === "/" && r2.routeData.route === "/") {
        return true;
      }
      return r2.routeData.fallbackRoutes.find((f) => {
        return f.route === route.route;
      });
    });
    if (!fallbackRoute) {
      throw new Error(`No fallback route found for route ${route.route}`);
    }
    return fallbackRoute.routeData;
  }
  function getCustom404Route(manifestData) {
    return manifestData.routes.find((r2) => isRoute404(r2.route));
  }
  function routeHasHtmlExtension(route) {
    return route.segments.some(
      (segment) => segment.some((part) => !part.dynamic && part.content.includes(".html"))
    );
  }
  function getProps(opts) {
    return __async(this, null, function* () {
      const {
        logger,
        mod,
        routeData: route,
        routeCache,
        pathname,
        serverLike,
        base,
        trailingSlash
      } = opts;
      if (!route || route.pathname) {
        return {};
      }
      if (routeIsRedirect(route) || routeIsFallback(route) || route.component === DEFAULT_404_COMPONENT) {
        return {};
      }
      const staticPaths = yield callGetStaticPaths({
        mod,
        route,
        routeCache,
        ssr: serverLike,
        base,
        trailingSlash
      });
      const params = getParams(route, pathname);
      const matchedStaticPath = findPathItemByKey(staticPaths, params, route, logger, trailingSlash);
      if (!matchedStaticPath && route.origin !== "internal" && (serverLike ? route.prerender : true)) {
        throw new AstroError(__spreadProps(__spreadValues({}, NoMatchingStaticPathFound), {
          message: NoMatchingStaticPathFound.message(pathname),
          hint: NoMatchingStaticPathFound.hint([route.component])
        }));
      }
      if (mod) {
        validatePrerenderEndpointCollision(route, mod, params);
      }
      const props = (matchedStaticPath == null ? void 0 : matchedStaticPath.props) ? __spreadValues({}, matchedStaticPath.props) : {};
      return props;
    });
  }
  function getParams(route, pathname) {
    if (!route.params.length) return {};
    const path = pathname.endsWith(".html") && route.type === "page" && !routeHasHtmlExtension(route) ? pathname.slice(0, -5) : pathname;
    const allPatterns = [route, ...route.fallbackRoutes].map((r2) => r2.pattern);
    const paramsMatch = allPatterns.map((pattern) => pattern.exec(path)).find((x) => x);
    if (!paramsMatch) return {};
    const params = {};
    route.params.forEach((key, i2) => {
      if (key.startsWith("...")) {
        params[key.slice(3)] = paramsMatch[i2 + 1] ? paramsMatch[i2 + 1] : void 0;
      } else {
        params[key] = paramsMatch[i2 + 1];
      }
    });
    return params;
  }
  function validatePrerenderEndpointCollision(route, mod, params) {
    if (route.type === "endpoint" && mod.getStaticPaths) {
      const lastSegment = route.segments[route.segments.length - 1];
      const paramValues = Object.values(params);
      const lastParam = paramValues[paramValues.length - 1];
      if (lastSegment.length === 1 && lastSegment[0].dynamic && lastParam === void 0) {
        throw new AstroError(__spreadProps(__spreadValues({}, PrerenderDynamicEndpointPathCollide), {
          message: PrerenderDynamicEndpointPathCollide.message(route.route),
          hint: PrerenderDynamicEndpointPathCollide.hint(route.component),
          location: {
            file: route.component
          }
        }));
      }
    }
  }
  function routeComparator(a2, b) {
    var _a29, _b13;
    const commonLength = Math.min(a2.segments.length, b.segments.length);
    for (let index = 0; index < commonLength; index++) {
      const aSegment = a2.segments[index];
      const bSegment = b.segments[index];
      const aIsStatic = aSegment.every((part) => !part.dynamic && !part.spread);
      const bIsStatic = bSegment.every((part) => !part.dynamic && !part.spread);
      if (aIsStatic && bIsStatic) {
        const aContent = aSegment.map((part) => part.content).join("");
        const bContent = bSegment.map((part) => part.content).join("");
        if (aContent !== bContent) {
          return aContent.localeCompare(bContent);
        }
      }
      if (aIsStatic !== bIsStatic) {
        return aIsStatic ? -1 : 1;
      }
      const aAllDynamic = aSegment.every((part) => part.dynamic);
      const bAllDynamic = bSegment.every((part) => part.dynamic);
      if (aAllDynamic !== bAllDynamic) {
        return aAllDynamic ? 1 : -1;
      }
      const aHasSpread = aSegment.some((part) => part.spread);
      const bHasSpread = bSegment.some((part) => part.spread);
      if (aHasSpread !== bHasSpread) {
        return aHasSpread ? 1 : -1;
      }
    }
    const aLength = a2.segments.length;
    const bLength = b.segments.length;
    if (aLength !== bLength) {
      const aEndsInRest = (_a29 = a2.segments.at(-1)) == null ? void 0 : _a29.some((part) => part.spread);
      const bEndsInRest = (_b13 = b.segments.at(-1)) == null ? void 0 : _b13.some((part) => part.spread);
      if (aEndsInRest !== bEndsInRest && Math.abs(aLength - bLength) === 1) {
        if (aLength > bLength && aEndsInRest) {
          return 1;
        }
        if (bLength > aLength && bEndsInRest) {
          return -1;
        }
      }
      return aLength > bLength ? -1 : 1;
    }
    if (a2.type === "endpoint" !== (b.type === "endpoint")) {
      return a2.type === "endpoint" ? -1 : 1;
    }
    return a2.route.localeCompare(b.route);
  }
  function normalizeBase(base) {
    if (!base) return "/";
    if (base === "/") return base;
    return prependForwardSlash(base);
  }
  function getRedirectForPathname(pathname) {
    let value = prependForwardSlash(pathname);
    if (value.startsWith("//")) {
      const collapsed = `/${value.replace(/^\/+/, "")}`;
      return { pathname: value, redirect: collapsed };
    }
    return { pathname: value };
  }
  function stripBase(pathname, base, baseWithoutTrailingSlash, trailingSlash) {
    if (base === "/") return pathname;
    const baseWithSlash = `${baseWithoutTrailingSlash}/`;
    if (pathname === baseWithoutTrailingSlash || pathname === base) {
      return trailingSlash === "always" ? null : "/";
    }
    if (pathname === baseWithSlash) {
      return trailingSlash === "never" ? null : "/";
    }
    if (pathname.startsWith(baseWithSlash)) {
      return pathname.slice(baseWithoutTrailingSlash.length);
    }
    return null;
  }
  function normalizeFileFormatPathname(pathname) {
    if (pathname.endsWith("/index.html")) {
      const trimmed = pathname.slice(0, -"/index.html".length);
      return trimmed === "" ? "/" : trimmed;
    }
    if (pathname.endsWith(".html")) {
      const trimmed = pathname.slice(0, -".html".length);
      return trimmed === "" ? "/" : trimmed;
    }
    return pathname;
  }
  function deserializeManifest(serializedManifest, routesList) {
    const routes = [];
    if (serializedManifest.routes) {
      for (const serializedRoute of serializedManifest.routes) {
        routes.push(__spreadProps(__spreadValues({}, serializedRoute), {
          routeData: deserializeRouteData(serializedRoute.routeData)
        }));
        const route = serializedRoute;
        route.routeData = deserializeRouteData(serializedRoute.routeData);
      }
    }
    const assets = new Set(serializedManifest.assets);
    const componentMetadata = new Map(serializedManifest.componentMetadata);
    const inlinedScripts = new Map(serializedManifest.inlinedScripts);
    const clientDirectives = new Map(serializedManifest.clientDirectives);
    const key = decodeKey(serializedManifest.key);
    return __spreadProps(__spreadValues({
      // in case user middleware exists, this no-op middleware will be reassigned (see plugin-ssr.ts)
      middleware() {
        return { onRequest: NOOP_MIDDLEWARE_FN };
      }
    }, serializedManifest), {
      rootDir: new URL(serializedManifest.rootDir),
      srcDir: new URL(serializedManifest.srcDir),
      publicDir: new URL(serializedManifest.publicDir),
      outDir: new URL(serializedManifest.outDir),
      cacheDir: new URL(serializedManifest.cacheDir),
      buildClientDir: new URL(serializedManifest.buildClientDir),
      buildServerDir: new URL(serializedManifest.buildServerDir),
      assets,
      componentMetadata,
      inlinedScripts,
      clientDirectives,
      routes,
      key
    });
  }
  function deserializeRouteData(rawRouteData) {
    return {
      route: rawRouteData.route,
      type: rawRouteData.type,
      // nosemgrep: javascript.lang.security.audit.detect-non-literal-regexp.detect-non-literal-regexp
      // This pattern is serialized from Astro's own route manifest.
      pattern: new RegExp(rawRouteData.pattern),
      params: rawRouteData.params,
      component: rawRouteData.component,
      pathname: rawRouteData.pathname || void 0,
      segments: rawRouteData.segments,
      prerender: rawRouteData.prerender,
      redirect: rawRouteData.redirect,
      redirectRoute: rawRouteData.redirectRoute ? deserializeRouteData(rawRouteData.redirectRoute) : void 0,
      fallbackRoutes: rawRouteData.fallbackRoutes.map((fallback) => {
        return deserializeRouteData(fallback);
      }),
      isIndex: rawRouteData.isIndex,
      origin: rawRouteData.origin,
      distURL: rawRouteData.distURL
    };
  }
  function deserializeRouteInfo(rawRouteInfo) {
    return {
      styles: rawRouteInfo.styles,
      file: rawRouteInfo.file,
      links: rawRouteInfo.links,
      scripts: rawRouteInfo.scripts,
      routeData: deserializeRouteData(rawRouteInfo.routeData)
    };
  }
  function log(opts, level, label, message, newLine = true) {
    const logLevel = opts.level;
    const dest = opts.destination;
    const event = {
      label,
      level,
      message,
      newLine
    };
    if (!isLogLevelEnabled(logLevel, level)) {
      return;
    }
    dest.write(event);
  }
  function isLogLevelEnabled(configuredLogLevel, level) {
    return levels[configuredLogLevel] <= levels[level];
  }
  function info(opts, label, message, newLine = true) {
    return log(opts, "info", label, message, newLine);
  }
  function warn(opts, label, message, newLine = true) {
    return log(opts, "warn", label, message, newLine);
  }
  function error(opts, label, message, newLine = true) {
    return log(opts, "error", label, message, newLine);
  }
  function debug(...args) {
    if ("_astroGlobalDebug" in globalThis) {
      globalThis._astroGlobalDebug(...args);
    }
  }
  function getEventPrefix({ level, label }) {
    const timestamp = `${dateTimeFormat.format(/* @__PURE__ */ new Date())}`;
    const prefix = [];
    if (level === "error" || level === "warn") {
      prefix.push(s.bold(timestamp));
      prefix.push(`[${level.toUpperCase()}]`);
    } else {
      prefix.push(timestamp);
    }
    if (label) {
      prefix.push(`[${label}]`);
    }
    if (level === "error") {
      return s.red(prefix.join(" "));
    }
    if (level === "warn") {
      return s.yellow(prefix.join(" "));
    }
    if (prefix.length === 1) {
      return s.dim(prefix[0]);
    }
    return s.dim(prefix[0]) + " " + s.blue(prefix.splice(1).join(" "));
  }
  function matchesLevel(messageLevel, configuredLevel) {
    return levels[messageLevel] >= levels[configuredLevel];
  }
  function nodeLogDestination(config2 = {}) {
    const { level = "info" } = config2;
    return {
      write(event) {
        let dest = process.stderr;
        if (levels[event.level] < levels["error"]) {
          dest = process.stdout;
        }
        if (!matchesLevel(event.level, level)) {
          return;
        }
        let trailingLine = event.newLine ? "\n" : "";
        if (event.label === "SKIP_FORMAT") {
          dest.write(event.message + trailingLine);
        } else {
          dest.write(getEventPrefix(event) + " " + event.message + trailingLine);
        }
      }
    };
  }
  function node_default(options) {
    return nodeLogDestination(options);
  }
  function consoleLogDestination(config2 = {}) {
    const { level = "info" } = config2;
    return {
      write(event) {
        let dest = console.error;
        if (levels[event.level] < levels["error"]) {
          dest = console.info;
        }
        if (!matchesLevel(event.level, level)) {
          return;
        }
        if (event.label === "SKIP_FORMAT") {
          dest(event.message);
        } else {
          dest(getEventPrefix(event) + " " + event.message);
        }
      }
    };
  }
  function createConsoleLogger({ level }) {
    return new AstroLogger({
      level,
      destination: consoleLogDestination()
    });
  }
  function console_default(options) {
    return consoleLogDestination(options);
  }
  function jsonLoggerDestination(config2 = {}) {
    const { pretty = false, level = "info" } = config2;
    return {
      write(event) {
        let dest = process.stderr;
        if (levels[event.level] < levels["error"]) {
          dest = process.stdout;
        }
        if (!matchesLevel(event.level, level)) {
          return;
        }
        let trailingLine = event.newLine ? "\n" : "";
        const message = event.message.replace(SGR_REGEX, "");
        if (pretty) {
          dest.write(
            JSON.stringify({ message, label: event.label, level: event.level }, null, 2) + trailingLine
          );
        } else {
          dest.write(
            JSON.stringify({ message, label: event.label, level: event.level }) + trailingLine
          );
        }
      }
    };
  }
  function compose(destinations) {
    return {
      write(chunk) {
        for (const logger of destinations) {
          logger.write(chunk);
        }
      },
      flush() {
        for (const logger of destinations) {
          if (logger.flush) {
            logger.flush();
          }
        }
      },
      close() {
        for (const logger of destinations) {
          if (logger.close) {
            logger.close();
          }
        }
      }
    };
  }
  function loadLogger(config2, level = "info") {
    return __async(this, null, function* () {
      var _a29, _b13;
      let cause = void 0;
      try {
        switch (config2.entrypoint) {
          case "astro/logger/node": {
            return new AstroLogger({
              destination: node_default(config2.config),
              level
            });
          }
          case "astro/logger/console": {
            return new AstroLogger({
              destination: console_default(config2.config),
              level
            });
          }
          case "astro/logger/json": {
            return new AstroLogger({
              destination: jsonLoggerDestination(config2.config),
              level
            });
          }
          case "astro/logger/compose": {
            let destinations = [];
            if ((_a29 = config2.config) == null ? void 0 : _a29.loggers) {
              const loggers = (_b13 = config2.config) == null ? void 0 : _b13.loggers;
              destinations = yield Promise.all(
                loggers.map((loggerConfig) => __async(null, null, function* () {
                  const logger = yield __orvalhoDynamicImport(
                    /* @vite-ignore */
                    loggerConfig.entrypoint
                  );
                  return logger.default(loggerConfig.config);
                }))
              );
            }
            return new AstroLogger({
              destination: compose(destinations),
              level
            });
          }
          default: {
            const nodeLogger = yield __orvalhoDynamicImport(
              /* @vite-ignore */
              config2.entrypoint
            );
            return new AstroLogger({
              destination: nodeLogger.default(config2.config),
              level
            });
          }
        }
      } catch (e2) {
        if (e2 instanceof Error) {
          cause = e2;
        }
      }
      const error2 = new AstroError(__spreadProps(__spreadValues({}, UnableToLoadLogger), {
        message: UnableToLoadLogger.message(config2.entrypoint)
      }));
      if (cause) {
        error2.cause = cause;
      }
      throw error2;
    });
  }
  function getFunctionExpression(slot) {
    var _a29;
    if (!slot) return;
    const expressions = (_a29 = slot == null ? void 0 : slot.expressions) == null ? void 0 : _a29.filter(
      (e2) => isRenderInstruction(e2) === false || isRenderTemplateResult(e2)
    );
    if ((expressions == null ? void 0 : expressions.length) !== 1) return;
    const expression = expressions[0];
    if (isRenderTemplateResult(expression)) {
      return getFunctionExpression(expression);
    }
    return expression;
  }
  function deduplicateDirectiveValues(existingDirective, newDirective) {
    const [directiveName, ...existingValues] = existingDirective.split(/\s+/).filter(Boolean);
    const [newDirectiveName, ...newValues] = newDirective.split(/\s+/).filter(Boolean);
    if (directiveName !== newDirectiveName) {
      return void 0;
    }
    const finalDirectives = Array.from(/* @__PURE__ */ new Set([...existingValues, ...newValues]));
    return `${directiveName} ${finalDirectives.join(" ")}`;
  }
  function pushDirective(directives, newDirective) {
    if (directives.length === 0) {
      return [newDirective];
    }
    const finalDirectives = [];
    let matched = false;
    for (const directive of directives) {
      if (matched) {
        finalDirectives.push(directive);
        continue;
      }
      const result = deduplicateDirectiveValues(directive, newDirective);
      if (result) {
        finalDirectives.push(result);
        matched = true;
      } else {
        finalDirectives.push(directive);
      }
    }
    if (!matched) {
      finalDirectives.push(newDirective);
    }
    return finalDirectives;
  }
  function computeFallbackRoute(options) {
    const {
      pathname,
      responseStatus,
      fallback,
      fallbackType,
      locales,
      defaultLocale,
      strategy,
      base
    } = options;
    if (responseStatus !== 404) {
      return { type: "none" };
    }
    if (!fallback || Object.keys(fallback).length === 0) {
      return { type: "none" };
    }
    const segments = pathname.split("/");
    const urlLocale = segments.find((segment) => {
      for (const locale of locales) {
        if (typeof locale === "string") {
          if (locale === segment) {
            return true;
          }
        } else if (locale.path === segment) {
          return true;
        }
      }
      return false;
    });
    if (!urlLocale) {
      return { type: "none" };
    }
    const fallbackKeys = Object.keys(fallback);
    if (!fallbackKeys.includes(urlLocale)) {
      return { type: "none" };
    }
    const fallbackLocale = fallback[urlLocale];
    const pathFallbackLocale = getPathByLocale(fallbackLocale, locales);
    let newPathname;
    if (pathFallbackLocale === defaultLocale && strategy === "pathname-prefix-other-locales") {
      if (pathname.includes(`${base}`)) {
        newPathname = pathname.replace(`/${urlLocale}`, ``);
      } else {
        newPathname = pathname.replace(`/${urlLocale}`, `/`);
      }
    } else {
      newPathname = pathname.replace(`/${urlLocale}`, `/${pathFallbackLocale}`);
    }
    return {
      type: fallbackType,
      pathname: newPathname
    };
  }
  function pathHasLocale(path, locales) {
    const segments = path.split("/").map(normalizeThePath);
    for (const segment of segments) {
      for (const locale of locales) {
        if (typeof locale === "string") {
          if (normalizeTheLocale(segment) === normalizeTheLocale(locale)) {
            return true;
          }
        } else if (segment === locale.path) {
          return true;
        }
      }
    }
    return false;
  }
  function getPathByLocale(locale, locales) {
    for (const loopLocale of locales) {
      if (typeof loopLocale === "string") {
        if (loopLocale === locale) {
          return loopLocale;
        }
      } else {
        for (const code of loopLocale.codes) {
          if (code === locale) {
            return loopLocale.path;
          }
        }
      }
    }
    throw new AstroError(i18nNoLocaleFoundInPath);
  }
  function normalizeTheLocale(locale) {
    return locale.replaceAll("_", "-").toLowerCase();
  }
  function normalizeThePath(path) {
    return path.endsWith(".html") ? path.slice(0, -5) : path;
  }
  function getAllCodes(locales) {
    const result = [];
    for (const loopLocale of locales) {
      if (typeof loopLocale === "string") {
        result.push(loopLocale);
      } else {
        result.push(...loopLocale.codes);
      }
    }
    return result;
  }
  function parseLocale(header) {
    if (header === "*") {
      return [{ locale: header, qualityValue: void 0 }];
    }
    const result = [];
    const localeValues = header.split(",").map((str) => str.trim());
    for (const localeValue of localeValues) {
      const split = localeValue.split(";").map((str) => str.trim());
      const localeName = split[0];
      const qualityValue = split[1];
      if (!split) {
        continue;
      }
      if (qualityValue && qualityValue.startsWith("q=")) {
        const qualityValueAsFloat = Number.parseFloat(qualityValue.slice("q=".length));
        if (Number.isNaN(qualityValueAsFloat) || qualityValueAsFloat > 1) {
          result.push({
            locale: localeName,
            qualityValue: void 0
          });
        } else {
          result.push({
            locale: localeName,
            qualityValue: qualityValueAsFloat
          });
        }
      } else {
        result.push({
          locale: localeName,
          qualityValue: void 0
        });
      }
    }
    return result;
  }
  function sortAndFilterLocales(browserLocaleList, locales) {
    const normalizedLocales = getAllCodes(locales).map(normalizeTheLocale);
    return browserLocaleList.filter((browserLocale) => {
      if (browserLocale.locale !== "*") {
        return normalizedLocales.includes(normalizeTheLocale(browserLocale.locale));
      }
      return true;
    }).sort((a2, b) => {
      if (a2.qualityValue && b.qualityValue) {
        return Math.sign(b.qualityValue - a2.qualityValue);
      }
      return 0;
    });
  }
  function computePreferredLocale(request, locales) {
    const acceptHeader = request.headers.get("Accept-Language");
    let result = void 0;
    if (acceptHeader) {
      const browserLocaleList = sortAndFilterLocales(parseLocale(acceptHeader), locales);
      const firstResult = browserLocaleList.at(0);
      if (firstResult && firstResult.locale !== "*") {
        outer: for (const currentLocale of locales) {
          if (typeof currentLocale === "string") {
            if (normalizeTheLocale(currentLocale) === normalizeTheLocale(firstResult.locale)) {
              result = currentLocale;
              break;
            }
          } else {
            for (const currentCode of currentLocale.codes) {
              if (normalizeTheLocale(currentCode) === normalizeTheLocale(firstResult.locale)) {
                result = currentCode;
                break outer;
              }
            }
          }
        }
      }
    }
    return result;
  }
  function computePreferredLocaleList(request, locales) {
    const acceptHeader = request.headers.get("Accept-Language");
    let result = [];
    if (acceptHeader) {
      const browserLocaleList = sortAndFilterLocales(parseLocale(acceptHeader), locales);
      if (browserLocaleList.length === 1 && browserLocaleList.at(0).locale === "*") {
        return getAllCodes(locales);
      } else if (browserLocaleList.length > 0) {
        for (const browserLocale of browserLocaleList) {
          for (const loopLocale of locales) {
            if (typeof loopLocale === "string") {
              if (normalizeTheLocale(loopLocale) === normalizeTheLocale(browserLocale.locale)) {
                result.push(loopLocale);
              }
            } else {
              for (const code of loopLocale.codes) {
                if (code === browserLocale.locale) {
                  result.push(code);
                }
              }
            }
          }
        }
      }
    }
    return result;
  }
  function computeCurrentLocale(pathname, locales, defaultLocale) {
    for (const segment of pathname.split("/").map(normalizeThePath)) {
      for (const locale of locales) {
        if (typeof locale === "string") {
          if (!segment.includes(locale)) continue;
          if (normalizeTheLocale(locale) === normalizeTheLocale(segment)) {
            return locale;
          }
        } else {
          if (locale.path === segment) {
            return locale.codes.at(0);
          } else {
            for (const code of locale.codes) {
              if (normalizeTheLocale(code) === normalizeTheLocale(segment)) {
                return code;
              }
            }
          }
        }
      }
    }
    for (const locale of locales) {
      if (typeof locale === "string") {
        if (locale === defaultLocale) {
          return locale;
        }
      } else {
        if (locale.path === defaultLocale) {
          return locale.codes.at(0);
        }
      }
    }
  }
  function computeCurrentLocaleFromParams(params, locales) {
    const byNormalizedCode = /* @__PURE__ */ new Map();
    const byPath = /* @__PURE__ */ new Map();
    for (const locale of locales) {
      if (typeof locale === "string") {
        byNormalizedCode.set(normalizeTheLocale(locale), locale);
      } else {
        byPath.set(locale.path, locale.codes[0]);
        for (const code of locale.codes) {
          byNormalizedCode.set(normalizeTheLocale(code), code);
        }
      }
    }
    for (const value of Object.values(params)) {
      if (!value) continue;
      const pathMatch = byPath.get(value);
      if (pathMatch) return pathMatch;
      const codeMatch = byNormalizedCode.get(normalizeTheLocale(value));
      if (codeMatch) return codeMatch;
    }
  }
  function callMiddleware(onRequest2, apiContext, responseFunction) {
    return __async(this, null, function* () {
      let nextCalled = false;
      let responseFunctionPromise = void 0;
      const next = (payload) => __async(null, null, function* () {
        nextCalled = true;
        responseFunctionPromise = responseFunction(apiContext, payload);
        return responseFunctionPromise;
      });
      const middlewarePromise = onRequest2(apiContext, next);
      return yield Promise.resolve(middlewarePromise).then((value) => __async(null, null, function* () {
        if (nextCalled) {
          if (typeof value !== "undefined") {
            if (value instanceof Response === false) {
              throw new AstroError(MiddlewareNotAResponse);
            }
            return value;
          } else {
            if (responseFunctionPromise) {
              return responseFunctionPromise;
            } else {
              throw new AstroError(MiddlewareNotAResponse);
            }
          }
        } else if (typeof value === "undefined") {
          throw new AstroError(MiddlewareNoDataOrNextCalled);
        } else if (value instanceof Response === false) {
          throw new AstroError(MiddlewareNotAResponse);
        } else {
          return value;
        }
      }));
    });
  }
  function createNormalizedUrl(requestUrl) {
    return normalizeUrl(new URL(requestUrl));
  }
  function normalizeUrl(url) {
    try {
      url.pathname = validateAndDecodePathname(url.pathname);
    } catch (e2) {
      try {
        url.pathname = decodeURI(url.pathname);
      } catch (e3) {
      }
    }
    url.pathname = collapseDuplicateSlashes(url.pathname);
    return url;
  }
  function applyRewriteToState(state, payload, { routeData, componentInstance, newUrl, pathname }, { mergeCookies = false } = {}) {
    const pipeline = state.pipeline;
    const oldPathname = state.pathname;
    const isI18nFallback = routeData.fallbackRoutes && routeData.fallbackRoutes.length > 0;
    if (pipeline.manifest.serverLike && !state.routeData.prerender && routeData.prerender && !isI18nFallback) {
      throw new AstroError(__spreadProps(__spreadValues({}, ForbiddenRewrite), {
        message: ForbiddenRewrite.message(state.pathname, pathname, routeData.component),
        hint: ForbiddenRewrite.hint(routeData.component)
      }));
    }
    state.routeData = routeData;
    state.componentInstance = componentInstance;
    if (payload instanceof Request) {
      state.request = payload;
    } else {
      state.request = copyRequest(
        newUrl,
        state.request,
        routeData.prerender,
        pipeline.logger,
        state.routeData.route
      );
    }
    state.url = createNormalizedUrl(state.request.url);
    if (mergeCookies) {
      const newCookies = new AstroCookies(state.request);
      if (state.cookies) {
        newCookies.merge(state.cookies);
      }
      state.cookies = newCookies;
    }
    state.params = getParams(routeData, pathname);
    state.pathname = pathname;
    state.isRewriting = true;
    state.status = 200;
    setOriginPathname(
      state.request,
      oldPathname,
      pipeline.manifest.trailingSlash,
      pipeline.manifest.buildFormat
    );
    state.invalidateContexts();
  }
  function matchRoute(pathname, manifest2) {
    if (isRoute404(pathname)) {
      const errorRoute = manifest2.routes.find((route) => isRoute404(route.route));
      if (errorRoute) return errorRoute;
    }
    if (isRoute500(pathname)) {
      const errorRoute = manifest2.routes.find((route) => isRoute500(route.route));
      if (errorRoute) return errorRoute;
    }
    return manifest2.routes.find((route) => {
      return route.pattern.test(pathname) || route.fallbackRoutes.some((fallbackRoute) => fallbackRoute.pattern.test(pathname));
    });
  }
  function isRoute404or500(route) {
    return isRoute404(route.route) || isRoute500(route.route);
  }
  function isRouteServerIsland(route) {
    return route.component === SERVER_ISLAND_COMPONENT;
  }
  function computePathnameFromDomain(request, url, i18n, base, trailingSlash, logger) {
    let pathname = void 0;
    if (i18n && (i18n.strategy === "domains-prefix-always" || i18n.strategy === "domains-prefix-other-locales" || i18n.strategy === "domains-prefix-always-no-redirect")) {
      let host = request.headers.get("X-Forwarded-Host");
      let protocol = request.headers.get("X-Forwarded-Proto");
      if (protocol) {
        protocol = protocol + ":";
      } else {
        protocol = url.protocol;
      }
      if (!host) {
        host = request.headers.get("Host");
      }
      if (host && protocol) {
        host = host.split(":")[0];
        try {
          let locale;
          const hostAsUrl = new URL(`${protocol}//${host}`);
          for (const [domainKey, localeValue] of Object.entries(i18n.domainLookupTable)) {
            const domainKeyAsUrl = new URL(domainKey);
            if (hostAsUrl.host === domainKeyAsUrl.host && hostAsUrl.protocol === domainKeyAsUrl.protocol) {
              locale = localeValue;
              break;
            }
          }
          if (locale) {
            pathname = prependForwardSlash(
              joinPaths(normalizeTheLocale(locale), removeBase(url.pathname, base))
            );
            if (trailingSlash === "always") {
              pathname = appendForwardSlash(pathname);
            } else if (trailingSlash === "never") {
              pathname = removeTrailingForwardSlash(pathname);
            } else if (url.pathname.endsWith("/")) {
              pathname = appendForwardSlash(pathname);
            }
          }
        } catch (e2) {
          logger.error(
            "router",
            `Astro tried to parse ${protocol}//${host} as an URL, but it threw a parsing error. Check the X-Forwarded-Host and X-Forwarded-Proto headers.`
          );
          logger.error("router", `Error: ${e2}`);
        }
      }
    }
    return pathname;
  }
  function removeBase(pathname, base) {
    pathname = collapseDuplicateLeadingSlashes(pathname);
    if (pathname.startsWith(base)) {
      return pathname.slice(removeTrailingForwardSlash(base).length + 1);
    }
    return pathname;
  }
  function getRenderOptions(request) {
    return Reflect.get(request, renderOptionsSymbol);
  }
  function setRenderOptions(request, options) {
    Reflect.set(request, renderOptionsSymbol, options);
  }
  function matchPattern(url, remotePattern) {
    return matchProtocol(url, remotePattern.protocol) && matchHostname(url, remotePattern.hostname, true) && matchPort(url, remotePattern.port) && matchPathname(url, remotePattern.pathname, true);
  }
  function matchPort(url, port) {
    return !port || port === url.port;
  }
  function matchProtocol(url, protocol) {
    return !protocol || protocol === url.protocol.slice(0, -1);
  }
  function matchHostname(url, hostname, allowWildcard = false) {
    if (!hostname) {
      return true;
    } else if (!allowWildcard || !hostname.startsWith("*")) {
      return hostname === url.hostname;
    } else if (hostname.startsWith("**.")) {
      const slicedHostname = hostname.slice(2);
      return slicedHostname !== url.hostname && url.hostname.endsWith(slicedHostname);
    } else if (hostname.startsWith("*.")) {
      const slicedHostname = hostname.slice(1);
      if (!url.hostname.endsWith(slicedHostname)) {
        return false;
      }
      const subdomainWithDot = url.hostname.slice(0, -(slicedHostname.length - 1));
      return subdomainWithDot.endsWith(".") && !subdomainWithDot.slice(0, -1).includes(".");
    }
    return false;
  }
  function matchPathname(url, pathname, allowWildcard = false) {
    if (!pathname) {
      return true;
    } else if (!allowWildcard || !pathname.endsWith("*")) {
      return pathname === url.pathname;
    } else if (pathname.endsWith("/**")) {
      const slicedPathname = pathname.slice(0, -2);
      return slicedPathname !== url.pathname && url.pathname.startsWith(slicedPathname);
    } else if (pathname.endsWith("/*")) {
      const slicedPathname = pathname.slice(0, -1);
      if (!url.pathname.startsWith(slicedPathname)) {
        return false;
      }
      const additionalPathChunks = url.pathname.slice(slicedPathname.length).split("/").filter(Boolean);
      return additionalPathChunks.length === 1;
    }
    return false;
  }
  function isRemoteAllowed(src, {
    domains,
    remotePatterns
  }) {
    if (!URL.canParse(src)) {
      return false;
    }
    const url = new URL(src);
    if (!["http:", "https:", "data:"].includes(url.protocol)) {
      return false;
    }
    return domains.some((domain) => matchHostname(url, domain)) || remotePatterns.some((remotePattern) => matchPattern(url, remotePattern));
  }
  function getFirstForwardedValue$1(multiValueHeader) {
    return multiValueHeader == null ? void 0 : multiValueHeader.toString().split(",").map((e2) => e2.trim())[0];
  }
  function sanitizeHost(hostname) {
    if (!hostname) return void 0;
    if (/[/\\]/.test(hostname)) return void 0;
    return hostname;
  }
  function parseHost(host) {
    const parts = host.split(":");
    return {
      hostname: parts[0],
      port: parts[1]
    };
  }
  function matchesAllowedDomains(hostname, protocol, port, allowedDomains) {
    const hostWithPort = port ? `${hostname}:${port}` : hostname;
    const urlString = `${protocol}://${hostWithPort}`;
    if (!URL.canParse(urlString)) {
      return false;
    }
    const testUrl = new URL(urlString);
    return allowedDomains.some((pattern) => matchPattern(testUrl, pattern));
  }
  function validateHost(host, protocol, allowedDomains) {
    if (!host || host.length === 0) return void 0;
    if (!allowedDomains || allowedDomains.length === 0) return void 0;
    const sanitized = sanitizeHost(host);
    if (!sanitized) return void 0;
    const { hostname, port } = parseHost(sanitized);
    if (matchesAllowedDomains(hostname, protocol, port, allowedDomains)) {
      return sanitized;
    }
    return void 0;
  }
  function validateForwardedHeaders(forwardedProtocol, forwardedHost, forwardedPort, allowedDomains) {
    const result = {};
    if (forwardedProtocol) {
      if (allowedDomains && allowedDomains.length > 0) {
        const hasProtocolPatterns = allowedDomains.some((pattern) => pattern.protocol !== void 0);
        if (hasProtocolPatterns) {
          try {
            const testUrl = new URL(`${forwardedProtocol}://example.com`);
            const isAllowed = allowedDomains.some(
              (pattern) => matchPattern(testUrl, { protocol: pattern.protocol })
            );
            if (isAllowed) {
              result.protocol = forwardedProtocol;
            }
          } catch (e2) {
          }
        } else if (/^https?$/.test(forwardedProtocol)) {
          result.protocol = forwardedProtocol;
        }
      }
    }
    if (forwardedPort && allowedDomains && allowedDomains.length > 0) {
      const hasPortPatterns = allowedDomains.some((pattern) => pattern.port !== void 0);
      if (hasPortPatterns) {
        const isAllowed = allowedDomains.some((pattern) => pattern.port === forwardedPort);
        if (isAllowed) {
          result.port = forwardedPort;
        }
      }
    }
    if (forwardedHost && forwardedHost.length > 0 && allowedDomains && allowedDomains.length > 0) {
      const protoForValidation = result.protocol || "https";
      const sanitized = sanitizeHost(forwardedHost);
      if (sanitized) {
        const { hostname, port: portFromHost } = parseHost(sanitized);
        const portForValidation = result.port || portFromHost;
        if (matchesAllowedDomains(hostname, protoForValidation, portForValidation, allowedDomains)) {
          result.host = sanitized;
        }
      }
    }
    return result;
  }
  function prepareResponse(response, { addCookieHeader }) {
    for (const headerName of INTERNAL_RESPONSE_HEADERS) {
      if (response.headers.has(headerName)) {
        response.headers.delete(headerName);
      }
    }
    if (addCookieHeader) {
      for (const setCookieHeaderValue of getSetCookiesFromResponse(response)) {
        response.headers.append("set-cookie", setCookieHeaderValue);
      }
    }
    Reflect.set(response, responseSentSymbol$1, true);
  }
  function redirectTemplate({
    status,
    absoluteLocation,
    relativeLocation,
    from
  }) {
    const delay = status === 302 ? 2 : 0;
    const rel = escape(String(relativeLocation));
    const abs = escape(String(absoluteLocation));
    const fromHtml = from ? `from <code>${escape(from)}</code> ` : "";
    return `<!doctype html>
<title>Redirecting to: ${rel}</title>
<meta http-equiv="refresh" content="${delay};url=${rel}">
<meta name="robots" content="noindex">
<link rel="canonical" href="${abs}">
<body>
	<a href="${rel}">Redirecting ${fromHtml}to <code>${rel}</code></a>
</body>`;
  }
  function defaultSetHeaders(options) {
    const headers = new Headers();
    const directives = [];
    if (options.maxAge !== void 0) {
      directives.push(`max-age=${options.maxAge}`);
    }
    if (options.swr !== void 0) {
      directives.push(`stale-while-revalidate=${options.swr}`);
    }
    if (directives.length > 0) {
      headers.set("CDN-Cache-Control", directives.join(", "));
    }
    if (options.tags && options.tags.length > 0) {
      headers.set("Cache-Tag", options.tags.join(", "));
    }
    if (options.lastModified) {
      headers.set("Last-Modified", options.lastModified.toUTCString());
    }
    if (options.etag) {
      headers.set("ETag", options.etag);
    }
    return headers;
  }
  function isLiveDataEntry(value) {
    return value != null && typeof value === "object" && "id" in value && "data" in value && "cacheHint" in value;
  }
  function applyCacheHeaders(cache, response) {
    if (APPLY_HEADERS in cache) {
      cache[APPLY_HEADERS](response);
    }
  }
  function getParts(part, file) {
    const result = [];
    part.split(ROUTE_DYNAMIC_SPLIT).map((str, i2) => {
      if (!str) return;
      const dynamic = i2 % 2 === 1;
      const [, content] = dynamic ? /([^(]+)$/.exec(str) || [null, null] : [null, str];
      if (!content || dynamic && !/^(?:\.\.\.)?[\w$]+$/.test(content)) {
        throw new Error(`Invalid route ${file} \u2014 parameter name must match /^[a-zA-Z0-9_$]+$/`);
      }
      result.push({
        content,
        dynamic,
        spread: dynamic && ROUTE_SPREAD.test(content)
      });
    });
    return result;
  }
  function compileCacheRoutes(routes, base, trailingSlash) {
    const compiled = Object.entries(routes).map(([path, options]) => {
      const segments = removeLeadingForwardSlash(path).split("/").filter(Boolean).map((s2) => getParts(s2, path));
      const pattern = getPattern(segments, base, trailingSlash);
      return { pattern, options, segments, route: path };
    });
    compiled.sort(
      (a2, b) => routeComparator(
        { segments: a2.segments, route: a2.route, type: "page" },
        { segments: b.segments, route: b.route, type: "page" }
      )
    );
    return compiled;
  }
  function matchCacheRoute(pathname, compiledRoutes) {
    for (const route of compiledRoutes) {
      if (route.pattern.test(pathname)) return route.options;
    }
    return null;
  }
  function provideCache(state) {
    const pipeline = state.pipeline;
    if (!pipeline.cacheConfig) {
      state.provide(CACHE_KEY, {
        create: () => new DisabledAstroCache(pipeline.logger)
      });
      return;
    }
    if (pipeline.runtimeMode === "development") {
      state.provide(CACHE_KEY, {
        create: () => new NoopAstroCache()
      });
      return;
    }
    return provideCacheAsync(state, pipeline);
  }
  function provideCacheAsync(state, pipeline) {
    return __async(this, null, function* () {
      const cacheProvider = yield pipeline.getCacheProvider();
      state.provide(CACHE_KEY, {
        create() {
          var _a29;
          const cache = new AstroCache(cacheProvider);
          if ((_a29 = pipeline.cacheConfig) == null ? void 0 : _a29.routes) {
            if (!pipeline.compiledCacheRoutes) {
              pipeline.compiledCacheRoutes = compileCacheRoutes(
                pipeline.cacheConfig.routes,
                pipeline.manifest.base,
                pipeline.manifest.trailingSlash
              );
            }
            const matched = matchCacheRoute(state.pathname, pipeline.compiledCacheRoutes);
            if (matched) {
              cache.set(matched);
            }
          }
          return cache;
        }
      });
    });
  }
  function isExternalURL(url) {
    return url.startsWith("http://") || url.startsWith("https://") || url.startsWith("//");
  }
  function redirectIsExternal(redirect) {
    if (typeof redirect === "string") {
      return isExternalURL(redirect);
    } else {
      return isExternalURL(redirect.destination);
    }
  }
  function computeRedirectStatus(method, redirect, redirectRoute) {
    return redirectRoute && typeof redirect === "object" ? redirect.status : method === "GET" ? 301 : 308;
  }
  function resolveRedirectTarget(params, redirect, redirectRoute, trailingSlash) {
    if (typeof redirectRoute !== "undefined") {
      const generate = getRouteGenerator(redirectRoute.segments, trailingSlash);
      return generate(params);
    } else if (typeof redirect === "string") {
      if (redirectIsExternal(redirect)) {
        return redirect;
      } else {
        let target = redirect;
        for (const param of Object.keys(params)) {
          const paramValue = params[param];
          target = target.replace(`[${param}]`, paramValue).replace(`[...${param}]`, paramValue);
        }
        return target;
      }
    } else if (typeof redirect === "undefined") {
      return "/";
    }
    return redirect.destination;
  }
  function renderRedirect(state) {
    return __async(this, null, function* () {
      state.pipeline.usedFeatures |= PipelineFeatures.redirects;
      const routeData = state.routeData;
      const { redirect, redirectRoute } = routeData;
      const status = computeRedirectStatus(state.request.method, redirect, redirectRoute);
      const headers = {
        location: encodeURI(
          resolveRedirectTarget(
            state.params,
            redirect,
            redirectRoute,
            state.pipeline.manifest.trailingSlash
          )
        )
      };
      if (redirect && redirectIsExternal(redirect)) {
        if (typeof redirect === "string") {
          return Response.redirect(redirect, status);
        } else {
          return Response.redirect(redirect.destination, status);
        }
      }
      return new Response(null, { status, headers });
    });
  }
  function jsonParseTransform(key, value) {
    if (key === "__proto__" || key === "constructor" && value && typeof value === "object" && "prototype" in value) {
      warnKeyDropped(key);
      return;
    }
    return value;
  }
  function warnKeyDropped(key) {
    console.warn(`[destr] Dropping "${key}" key to prevent prototype pollution.`);
  }
  function destr(value, options = {}) {
    if (typeof value !== "string") {
      return value;
    }
    if (value[0] === '"' && value[value.length - 1] === '"' && value.indexOf("\\") === -1) {
      return value.slice(1, -1);
    }
    const _value = value.trim();
    if (_value.length <= 9) {
      switch (_value.toLowerCase()) {
        case "true": {
          return true;
        }
        case "false": {
          return false;
        }
        case "undefined": {
          return void 0;
        }
        case "null": {
          return null;
        }
        case "nan": {
          return Number.NaN;
        }
        case "infinity": {
          return Number.POSITIVE_INFINITY;
        }
        case "-infinity": {
          return Number.NEGATIVE_INFINITY;
        }
      }
    }
    if (!JsonSigRx.test(value)) {
      if (options.strict) {
        throw new SyntaxError("[destr] Invalid JSON");
      }
      return value;
    }
    try {
      if (suspectProtoRx.test(value) || suspectConstructorRx.test(value)) {
        if (options.strict) {
          throw new Error("[destr] Possible prototype pollution");
        }
        return JSON.parse(value, jsonParseTransform);
      }
      return JSON.parse(value);
    } catch (error2) {
      if (options.strict) {
        throw error2;
      }
      return value;
    }
  }
  function wrapToPromise(value) {
    if (!value || typeof value.then !== "function") {
      return Promise.resolve(value);
    }
    return value;
  }
  function asyncCall(function_, ...arguments_) {
    try {
      return wrapToPromise(function_(...arguments_));
    } catch (error2) {
      return Promise.reject(error2);
    }
  }
  function isPrimitive(value) {
    const type = typeof value;
    return value === null || type !== "object" && type !== "function";
  }
  function isPureObject(value) {
    const proto = Object.getPrototypeOf(value);
    return !proto || proto.isPrototypeOf(Object);
  }
  function stringify$1(value) {
    if (isPrimitive(value)) {
      return String(value);
    }
    if (isPureObject(value) || Array.isArray(value)) {
      return JSON.stringify(value);
    }
    if (typeof value.toJSON === "function") {
      return stringify$1(value.toJSON());
    }
    throw new Error("[unstorage] Cannot stringify value!");
  }
  function serializeRaw(value) {
    if (typeof value === "string") {
      return value;
    }
    return BASE64_PREFIX + base64Encode(value);
  }
  function deserializeRaw(value) {
    if (typeof value !== "string") {
      return value;
    }
    if (!value.startsWith(BASE64_PREFIX)) {
      return value;
    }
    return base64Decode(value.slice(BASE64_PREFIX.length));
  }
  function base64Decode(input) {
    if (globalThis.Buffer) {
      return Buffer.from(input, "base64");
    }
    return Uint8Array.from(
      globalThis.atob(input),
      (c) => c.codePointAt(0)
    );
  }
  function base64Encode(input) {
    if (globalThis.Buffer) {
      return Buffer.from(input).toString("base64");
    }
    return globalThis.btoa(String.fromCodePoint(...input));
  }
  function normalizeKey2(key) {
    var _a29;
    if (!key) {
      return "";
    }
    return ((_a29 = key.split("?")[0]) == null ? void 0 : _a29.replace(/[/\\]/g, ":").replace(/:+/g, ":").replace(/^:|:$/g, "")) || "";
  }
  function joinKeys2(...keys) {
    return normalizeKey2(keys.join(":"));
  }
  function normalizeBaseKey(base) {
    base = normalizeKey2(base);
    return base ? base + ":" : "";
  }
  function filterKeyByDepth(key, depth) {
    if (depth === void 0) {
      return true;
    }
    let substrCount = 0;
    let index = key.indexOf(":");
    while (index > -1) {
      substrCount++;
      index = key.indexOf(":", index + 1);
    }
    return substrCount <= depth;
  }
  function filterKeyByBase(key, base) {
    if (base) {
      return key.startsWith(base) && key[key.length - 1] !== "$";
    }
    return key[key.length - 1] !== "$";
  }
  function defineDriver2(factory) {
    return factory;
  }
  function createStorage(options = {}) {
    const context = {
      mounts: { "": options.driver || memory() },
      mountpoints: [""],
      watching: false,
      watchListeners: [],
      unwatch: {}
    };
    const getMount = (key) => {
      for (const base of context.mountpoints) {
        if (key.startsWith(base)) {
          return {
            base,
            relativeKey: key.slice(base.length),
            driver: context.mounts[base]
          };
        }
      }
      return {
        base: "",
        relativeKey: key,
        driver: context.mounts[""]
      };
    };
    const getMounts = (base, includeParent) => {
      return context.mountpoints.filter(
        (mountpoint) => mountpoint.startsWith(base) || includeParent && base.startsWith(mountpoint)
      ).map((mountpoint) => ({
        relativeBase: base.length > mountpoint.length ? base.slice(mountpoint.length) : void 0,
        mountpoint,
        driver: context.mounts[mountpoint]
      }));
    };
    const onChange = (event, key) => {
      if (!context.watching) {
        return;
      }
      key = normalizeKey2(key);
      for (const listener of context.watchListeners) {
        listener(event, key);
      }
    };
    const startWatch = () => __async(null, null, function* () {
      if (context.watching) {
        return;
      }
      context.watching = true;
      for (const mountpoint in context.mounts) {
        context.unwatch[mountpoint] = yield watch(
          context.mounts[mountpoint],
          onChange,
          mountpoint
        );
      }
    });
    const stopWatch = () => __async(null, null, function* () {
      if (!context.watching) {
        return;
      }
      for (const mountpoint in context.unwatch) {
        yield context.unwatch[mountpoint]();
      }
      context.unwatch = {};
      context.watching = false;
    });
    const runBatch = (items, commonOptions, cb) => {
      const batches = /* @__PURE__ */ new Map();
      const getBatch = (mount) => {
        let batch = batches.get(mount.base);
        if (!batch) {
          batch = {
            driver: mount.driver,
            base: mount.base,
            items: []
          };
          batches.set(mount.base, batch);
        }
        return batch;
      };
      for (const item of items) {
        const isStringItem = typeof item === "string";
        const key = normalizeKey2(isStringItem ? item : item.key);
        const value = isStringItem ? void 0 : item.value;
        const options2 = isStringItem || !item.options ? commonOptions : __spreadValues(__spreadValues({}, commonOptions), item.options);
        const mount = getMount(key);
        getBatch(mount).items.push({
          key,
          value,
          relativeKey: mount.relativeKey,
          options: options2
        });
      }
      return Promise.all([...batches.values()].map((batch) => cb(batch))).then(
        (r2) => r2.flat()
      );
    };
    const storage = {
      // Item
      hasItem(key, opts = {}) {
        key = normalizeKey2(key);
        const { relativeKey, driver } = getMount(key);
        return asyncCall(driver.hasItem, relativeKey, opts);
      },
      getItem(key, opts = {}) {
        key = normalizeKey2(key);
        const { relativeKey, driver } = getMount(key);
        return asyncCall(driver.getItem, relativeKey, opts).then(
          (value) => destr(value)
        );
      },
      getItems(items, commonOptions = {}) {
        return runBatch(items, commonOptions, (batch) => {
          if (batch.driver.getItems) {
            return asyncCall(
              batch.driver.getItems,
              batch.items.map((item) => ({
                key: item.relativeKey,
                options: item.options
              })),
              commonOptions
            ).then(
              (r2) => r2.map((item) => ({
                key: joinKeys2(batch.base, item.key),
                value: destr(item.value)
              }))
            );
          }
          return Promise.all(
            batch.items.map((item) => {
              return asyncCall(
                batch.driver.getItem,
                item.relativeKey,
                item.options
              ).then((value) => ({
                key: item.key,
                value: destr(value)
              }));
            })
          );
        });
      },
      getItemRaw(key, opts = {}) {
        key = normalizeKey2(key);
        const { relativeKey, driver } = getMount(key);
        if (driver.getItemRaw) {
          return asyncCall(driver.getItemRaw, relativeKey, opts);
        }
        return asyncCall(driver.getItem, relativeKey, opts).then(
          (value) => deserializeRaw(value)
        );
      },
      setItem(_0, _1) {
        return __async(this, arguments, function* (key, value, opts = {}) {
          if (value === void 0) {
            return storage.removeItem(key);
          }
          key = normalizeKey2(key);
          const { relativeKey, driver } = getMount(key);
          if (!driver.setItem) {
            return;
          }
          yield asyncCall(driver.setItem, relativeKey, stringify$1(value), opts);
          if (!driver.watch) {
            onChange("update", key);
          }
        });
      },
      setItems(items, commonOptions) {
        return __async(this, null, function* () {
          yield runBatch(items, commonOptions, (batch) => __async(null, null, function* () {
            if (batch.driver.setItems) {
              return asyncCall(
                batch.driver.setItems,
                batch.items.map((item) => ({
                  key: item.relativeKey,
                  value: stringify$1(item.value),
                  options: item.options
                })),
                commonOptions
              );
            }
            if (!batch.driver.setItem) {
              return;
            }
            yield Promise.all(
              batch.items.map((item) => {
                return asyncCall(
                  batch.driver.setItem,
                  item.relativeKey,
                  stringify$1(item.value),
                  item.options
                );
              })
            );
          }));
        });
      },
      setItemRaw(_0, _1) {
        return __async(this, arguments, function* (key, value, opts = {}) {
          if (value === void 0) {
            return storage.removeItem(key, opts);
          }
          key = normalizeKey2(key);
          const { relativeKey, driver } = getMount(key);
          if (driver.setItemRaw) {
            yield asyncCall(driver.setItemRaw, relativeKey, value, opts);
          } else if (driver.setItem) {
            yield asyncCall(driver.setItem, relativeKey, serializeRaw(value), opts);
          } else {
            return;
          }
          if (!driver.watch) {
            onChange("update", key);
          }
        });
      },
      removeItem(_0) {
        return __async(this, arguments, function* (key, opts = {}) {
          if (typeof opts === "boolean") {
            opts = { removeMeta: opts };
          }
          key = normalizeKey2(key);
          const { relativeKey, driver } = getMount(key);
          if (!driver.removeItem) {
            return;
          }
          yield asyncCall(driver.removeItem, relativeKey, opts);
          if (opts.removeMeta || opts.removeMata) {
            yield asyncCall(driver.removeItem, relativeKey + "$", opts);
          }
          if (!driver.watch) {
            onChange("remove", key);
          }
        });
      },
      // Meta
      getMeta(_0) {
        return __async(this, arguments, function* (key, opts = {}) {
          if (typeof opts === "boolean") {
            opts = { nativeOnly: opts };
          }
          key = normalizeKey2(key);
          const { relativeKey, driver } = getMount(key);
          const meta = /* @__PURE__ */ Object.create(null);
          if (driver.getMeta) {
            Object.assign(meta, yield asyncCall(driver.getMeta, relativeKey, opts));
          }
          if (!opts.nativeOnly) {
            const value = yield asyncCall(
              driver.getItem,
              relativeKey + "$",
              opts
            ).then((value_) => destr(value_));
            if (value && typeof value === "object") {
              if (typeof value.atime === "string") {
                value.atime = new Date(value.atime);
              }
              if (typeof value.mtime === "string") {
                value.mtime = new Date(value.mtime);
              }
              Object.assign(meta, value);
            }
          }
          return meta;
        });
      },
      setMeta(key, value, opts = {}) {
        return this.setItem(key + "$", value, opts);
      },
      removeMeta(key, opts = {}) {
        return this.removeItem(key + "$", opts);
      },
      // Keys
      getKeys(_0) {
        return __async(this, arguments, function* (base, opts = {}) {
          var _a30;
          base = normalizeBaseKey(base);
          const mounts = getMounts(base, true);
          let maskedMounts = [];
          const allKeys = [];
          let allMountsSupportMaxDepth = true;
          for (const mount of mounts) {
            if (!((_a30 = mount.driver.flags) == null ? void 0 : _a30.maxDepth)) {
              allMountsSupportMaxDepth = false;
            }
            const rawKeys = yield asyncCall(
              mount.driver.getKeys,
              mount.relativeBase,
              opts
            );
            for (const key of rawKeys) {
              const fullKey = mount.mountpoint + normalizeKey2(key);
              if (!maskedMounts.some((p) => fullKey.startsWith(p))) {
                allKeys.push(fullKey);
              }
            }
            maskedMounts = [
              mount.mountpoint,
              ...maskedMounts.filter((p) => !p.startsWith(mount.mountpoint))
            ];
          }
          const shouldFilterByDepth = opts.maxDepth !== void 0 && !allMountsSupportMaxDepth;
          return allKeys.filter(
            (key) => (!shouldFilterByDepth || filterKeyByDepth(key, opts.maxDepth)) && filterKeyByBase(key, base)
          );
        });
      },
      // Utils
      clear(_0) {
        return __async(this, arguments, function* (base, opts = {}) {
          base = normalizeBaseKey(base);
          yield Promise.all(
            getMounts(base, false).map((m) => __async(null, null, function* () {
              if (m.driver.clear) {
                return asyncCall(m.driver.clear, m.relativeBase, opts);
              }
              if (m.driver.removeItem) {
                const keys = yield m.driver.getKeys(m.relativeBase || "", opts);
                return Promise.all(
                  keys.map((key) => m.driver.removeItem(key, opts))
                );
              }
            }))
          );
        });
      },
      dispose() {
        return __async(this, null, function* () {
          yield Promise.all(
            Object.values(context.mounts).map((driver) => dispose(driver))
          );
        });
      },
      watch(callback) {
        return __async(this, null, function* () {
          yield startWatch();
          context.watchListeners.push(callback);
          return () => __async(null, null, function* () {
            context.watchListeners = context.watchListeners.filter(
              (listener) => listener !== callback
            );
            if (context.watchListeners.length === 0) {
              yield stopWatch();
            }
          });
        });
      },
      unwatch() {
        return __async(this, null, function* () {
          context.watchListeners = [];
          yield stopWatch();
        });
      },
      // Mount
      mount(base, driver) {
        base = normalizeBaseKey(base);
        if (base && context.mounts[base]) {
          throw new Error(`already mounted at ${base}`);
        }
        if (base) {
          context.mountpoints.push(base);
          context.mountpoints.sort((a2, b) => b.length - a2.length);
        }
        context.mounts[base] = driver;
        if (context.watching) {
          Promise.resolve(watch(driver, onChange, base)).then((unwatcher) => {
            context.unwatch[base] = unwatcher;
          }).catch(console.error);
        }
        return storage;
      },
      unmount(base, _dispose = true) {
        return __async(this, null, function* () {
          var _a30, _b13;
          base = normalizeBaseKey(base);
          if (!base || !context.mounts[base]) {
            return;
          }
          if (context.watching && base in context.unwatch) {
            (_b13 = (_a30 = context.unwatch)[base]) == null ? void 0 : _b13.call(_a30);
            delete context.unwatch[base];
          }
          if (_dispose) {
            yield dispose(context.mounts[base]);
          }
          context.mountpoints = context.mountpoints.filter((key) => key !== base);
          delete context.mounts[base];
        });
      },
      getMount(key = "") {
        key = normalizeKey2(key) + ":";
        const m = getMount(key);
        return {
          driver: m.driver,
          base: m.base
        };
      },
      getMounts(base = "", opts = {}) {
        base = normalizeKey2(base);
        const mounts = getMounts(base, opts.parents);
        return mounts.map((m) => ({
          driver: m.driver,
          base: m.mountpoint
        }));
      },
      // Aliases
      keys: (base, opts = {}) => storage.getKeys(base, opts),
      get: (key, opts = {}) => storage.getItem(key, opts),
      set: (key, value, opts = {}) => storage.setItem(key, value, opts),
      has: (key, opts = {}) => storage.hasItem(key, opts),
      del: (key, opts = {}) => storage.removeItem(key, opts),
      remove: (key, opts = {}) => storage.removeItem(key, opts)
    };
    return storage;
  }
  function watch(driver, onChange, base) {
    return driver.watch ? driver.watch((event, key) => onChange(event, base + key)) : () => {
    };
  }
  function dispose(driver) {
    return __async(this, null, function* () {
      if (typeof driver.dispose === "function") {
        yield asyncCall(driver.dispose);
      }
    });
  }
  function provideSession(state) {
    state.pipeline.usedFeatures |= PipelineFeatures.sessions;
    const pipeline = state.pipeline;
    const config2 = pipeline.manifest.sessionConfig;
    if (!config2) return;
    return provideSessionAsync(state, config2);
  }
  function provideSessionAsync(state, config2) {
    return __async(this, null, function* () {
      const pipeline = state.pipeline;
      const driverFactory = yield pipeline.getSessionDriver();
      if (!driverFactory) return;
      state.provide(SESSION_KEY, {
        create() {
          const cookies = state.cookies;
          return new AstroSession({
            cookies,
            config: config2,
            runtimeMode: pipeline.runtimeMode,
            driverFactory,
            mockStorage: null
          });
        },
        finalize(session) {
          return session[PERSIST_SYMBOL]();
        }
      });
    });
  }
  function mergeResponses(newResponse, originalResponse, override) {
    let newResponseHeaders = newResponse.headers;
    if (override == null ? void 0 : override.removeContentEncodingHeaders) {
      newResponseHeaders = new Headers(newResponseHeaders);
      newResponseHeaders.delete("Content-Encoding");
      newResponseHeaders.delete("Content-Length");
    }
    if (!originalResponse) {
      if (override !== void 0) {
        return new Response(newResponse.body, {
          status: override.status,
          statusText: newResponse.statusText,
          headers: newResponseHeaders
        });
      }
      return newResponse;
    }
    const status = (override == null ? void 0 : override.status) ? override.status : originalResponse.status === 200 ? newResponse.status : originalResponse.status;
    try {
      originalResponse.headers.delete("Content-type");
      originalResponse.headers.delete("Content-Length");
      originalResponse.headers.delete("Transfer-Encoding");
    } catch (e2) {
    }
    const newHeaders = new Headers();
    const seen = /* @__PURE__ */ new Set();
    for (const [name, value] of originalResponse.headers) {
      newHeaders.append(name, value);
      seen.add(name.toLowerCase());
    }
    for (const [name, value] of newResponseHeaders) {
      if (!seen.has(name.toLowerCase())) {
        newHeaders.append(name, value);
      }
    }
    const mergedResponse = new Response(newResponse.body, {
      status,
      statusText: status === 200 ? newResponse.statusText : originalResponse.statusText,
      // If you're looking at here for possible bugs, it means that it's not a bug.
      // With the middleware, users can meddle with headers, and we should pass to the 404/500.
      // If users see something weird, it's because they are setting some headers they should not.
      //
      // Although, we don't want it to replace the content-type, because the error page must return `text/html`
      headers: newHeaders
    });
    const originalCookies = getCookiesFromResponse(originalResponse);
    const newCookies = getCookiesFromResponse(newResponse);
    if (originalCookies) {
      if (newCookies) {
        for (const cookieValue of newCookies.consume()) {
          originalResponse.headers.append("set-cookie", cookieValue);
        }
      }
      attachCookiesToResponse(mergedResponse, originalCookies);
    } else if (newCookies) {
      attachCookiesToResponse(mergedResponse, newCookies);
    }
    return mergedResponse;
  }
  function getAssetsPrefix(fileExtension2, assetsPrefix) {
    let prefix = "";
    if (!assetsPrefix) {
      prefix = "";
    } else if (typeof assetsPrefix === "string") {
      prefix = assetsPrefix;
    } else {
      const dotLessFileExtension = fileExtension2.slice(1);
      prefix = assetsPrefix[dotLessFileExtension] || assetsPrefix.fallback;
    }
    return prefix;
  }
  function splitAssetPath(path) {
    const parsed = new URL(path, URL_PARSE_BASE);
    const isAbsolute = URL.canParse(path);
    const pathname = !isAbsolute && !path.startsWith("/") ? parsed.pathname.slice(1) : parsed.pathname;
    return {
      pathname,
      suffix: `${parsed.search}${parsed.hash}`
    };
  }
  function createAssetLink(href, base, assetsPrefix, queryParams) {
    const { pathname, suffix } = splitAssetPath(href);
    let url = "";
    if (assetsPrefix) {
      const pf = getAssetsPrefix(fileExtension(pathname), assetsPrefix);
      url = joinPaths(pf, slash(pathname)) + suffix;
    } else if (base) {
      url = prependForwardSlash(joinPaths(base, slash(pathname))) + suffix;
    } else {
      url = href;
    }
    return url;
  }
  function createStylesheetElement(stylesheet, base, assetsPrefix, queryParams) {
    if (stylesheet.type === "inline") {
      return {
        props: {},
        children: stylesheet.content
      };
    } else {
      return {
        props: {
          rel: "stylesheet",
          href: createAssetLink(stylesheet.src, base, assetsPrefix)
        },
        children: ""
      };
    }
  }
  function createStylesheetElementSet(stylesheets, base, assetsPrefix, queryParams) {
    return new Set(
      stylesheets.map((s2) => createStylesheetElement(s2, base, assetsPrefix))
    );
  }
  function createModuleScriptElement(script, base, assetsPrefix, queryParams) {
    if (script.type === "external") {
      return createModuleScriptElementWithSrc(script.value, base, assetsPrefix);
    } else {
      return {
        props: {
          type: "module"
        },
        children: script.value
      };
    }
  }
  function createModuleScriptElementWithSrc(src, base, assetsPrefix, queryParams) {
    return {
      props: {
        type: "module",
        src: createAssetLink(src, base, assetsPrefix)
      },
      children: ""
    };
  }
  function getFirstForwardedValue(multiValueHeader) {
    var _a29, _b13;
    return (_b13 = (_a29 = multiValueHeader == null ? void 0 : multiValueHeader.toString()) == null ? void 0 : _a29.split(",").map((e2) => e2.trim())) == null ? void 0 : _b13[0];
  }
  function isValidIpAddress(value) {
    return IP_RE.test(value);
  }
  function getValidatedIpFromHeader(headerValue) {
    const raw = getFirstForwardedValue(headerValue);
    if (raw && isValidIpAddress(raw)) {
      return raw;
    }
    return void 0;
  }
  function matchStaticAsset(manifest2, requestUrl, env2) {
    const { pathname } = new URL(requestUrl);
    if (manifest2.assets.has(pathname)) {
      return env2.ASSETS.fetch(requestUrl.replace(/\.html$/, ""));
    }
    return void 0;
  }
  function fallbackToAssets(requestUrl, env2) {
    return __async(this, null, function* () {
      const asset = yield env2.ASSETS.fetch(
        requestUrl.replace(/index.html$/, "").replace(/\.html$/, "")
      );
      if (asset.status !== 404) {
        return asset;
      }
      return void 0;
    });
  }
  function createErrorPageFetch(env2) {
    return (url) => __async(null, null, function* () {
      return env2.ASSETS.fetch(url.replace(/\.html$/, ""));
    });
  }
  function createLocals(ctx) {
    const locals = {
      cfContext: ctx
    };
    Object.defineProperty(locals, "runtime", {
      enumerable: false,
      value: {
        get env() {
          throw new Error(
            `Astro.locals.runtime.env has been removed in Astro v6. Use 'import { env } from "cloudflare:workers"' instead.`
          );
        },
        get cf() {
          throw new Error(
            `Astro.locals.runtime.cf has been removed in Astro v6. Use 'Astro.request.cf' instead.`
          );
        },
        get caches() {
          throw new Error(
            `Astro.locals.runtime.caches has been removed in Astro v6. Use the global 'caches' object instead.`
          );
        },
        get ctx() {
          throw new Error(
            `Astro.locals.runtime.ctx has been removed in Astro v6. Use 'Astro.locals.cfContext' instead.`
          );
        }
      }
    });
    return locals;
  }
  function getClientAddress(request) {
    return getValidatedIpFromHeader(request.headers.get("cf-connecting-ip"));
  }
  function injectSessionBinding(manifest2, env2) {
    var _a29, _b13;
    if (env2[sessionKVBindingName]) {
      const sessionConfigOptions = (_b13 = (_a29 = manifest2.sessionConfig) == null ? void 0 : _a29.options) != null ? _b13 : {};
      Object.assign(sessionConfigOptions, {
        binding: env2[sessionKVBindingName]
      });
    }
  }
  function handle(request, env2, context) {
    return __async(this, null, function* () {
      injectSessionBinding(app.manifest, env2);
      const staticAsset = matchStaticAsset(app.manifest, request.url, env2);
      if (staticAsset) return staticAsset;
      let routeData = void 0;
      if (app.isDev()) {
        const result = yield app.devMatch(app.getPathnameFromRequest(request));
        if (result) {
          routeData = result.routeData;
        }
      } else {
        routeData = app.match(request);
      }
      if (!routeData) {
        const asset = yield fallbackToAssets(request.url, env2);
        if (asset) return asset;
      }
      const locals = createLocals(context);
      const waitUntil = context.waitUntil.bind(context);
      const response = yield app.render(request, {
        routeData,
        locals,
        waitUntil,
        prerenderedErrorPageFetch: createErrorPageFetch(env2),
        clientAddress: getClientAddress(request)
      });
      if (app.setCookieHeaders) {
        for (const setCookieHeader of app.setCookieHeaders(response)) {
          response.headers.append("Set-Cookie", setCookieHeader);
        }
      }
      return response;
    });
  }
  var _a17, _a18, _b9, sessionKVBindingName, e, t, n, r$1, i, a, o, s, MANY_LEADING_SLASHES, MANY_SLASHES, MANY_TRAILING_SLASHES, INTERNAL_PREFIXES, JUST_SLASHES, WITH_FILE_EXT, UNDEFINED, HOLE, NAN, POSITIVE_INFINITY, NEGATIVE_INFINITY, NEGATIVE_ZERO, SPARSE, MAX_ARRAY_LEN, MAX_ARRAY_INDEX, DevalueError, object_proto_names, is_identifier, native, _a19, buffer, encode64, decode64, ACTION_QUERY_PARAMS, ACTION_RPC_ROUTE_PATTERN, __vite_import_meta_env__$1, codeToStatusMap, statusToCodeMap, ActionError, ActionInputError, actionResultErrorStack, C, _a$1, $ZodAsyncError, $ZodEncodeError, _a20, globalConfig, EVALUATING, captureStackTrace, propertyKeyTypes, initializer$1, $ZodError, $ZodRealError, _parse, _parseAsync, _safeParse, safeParse$1, _safeParseAsync, safeParseAsync$1, _encode, _decode, _encodeAsync, _decodeAsync, _safeEncode, _safeDecode, _safeEncodeAsync, _safeDecodeAsync, $ZodCheck, $ZodCheckMaxLength, $ZodCheckMinLength, $ZodCheckLengthEquals, $ZodCheckOverwrite, version, $ZodType, $ZodArray, $ZodUnion, $ZodIntersection, $ZodEnum, $ZodTransform, $ZodOptional, $ZodExactOptional, $ZodNullable, $ZodDefault, $ZodPrefault, $ZodNonOptional, $ZodCatch, $ZodPipe, $ZodReadonly, $ZodCustom, _a21, $ZodRegistry, _a22, globalRegistry, createToJSONSchemaMethod, createStandardJSONSchemaMethod, enumProcessor, customProcessor, transformProcessor, arrayProcessor, unionProcessor, intersectionProcessor, nullableProcessor, nonoptionalProcessor, defaultProcessor, prefaultProcessor, catchProcessor, pipeProcessor, readonlyProcessor, optionalProcessor, ASTRO_VERSION, ASTRO_GENERATOR, REROUTE_DIRECTIVE_HEADER, REWRITE_DIRECTIVE_HEADER_KEY, REWRITE_DIRECTIVE_HEADER_VALUE, NOOP_MIDDLEWARE_HEADER, ROUTE_TYPE_HEADER, INTERNAL_RESPONSE_HEADERS, ASTRO_ERROR_HEADER, DEFAULT_404_COMPONENT, REDIRECT_STATUS_CODES, REROUTABLE_STATUS_CODES, clientAddressSymbol, originPathnameSymbol, pipelineSymbol, fetchStateSymbol, appSymbol, responseSentSymbol$1, ClientAddressNotAvailable, PrerenderClientAddressNotAvailable, StaticClientAddressNotAvailable, NoMatchingStaticPathFound, OnlyResponseCanBeReturned, MissingMediaQueryDirective, NoMatchingRenderer, NoClientOnlyHint, InvalidGetStaticPathsEntry, InvalidGetStaticPathsReturn, GetStaticPathsExpectedParams, GetStaticPathsInvalidRouteParam, GetStaticPathsRequired, ReservedSlotName, NoMatchingImport, InvalidComponentArgs, PageNumberParamNotFound, ImageMissingAlt, InvalidImageService, MissingImageDimension, FailedToFetchRemoteImageDimensions, RemoteImageNotAllowed, UnsupportedImageFormat, UnsupportedImageConversion, PrerenderDynamicEndpointPathCollide, ExpectedImage, ExpectedImageOptions, ExpectedNotESMImage, IncompatibleDescriptorOptions, NoImageMetadata, ResponseSentError, MiddlewareNoDataOrNextCalled, MiddlewareNotAResponse, EndpointDidNotReturnAResponse, LocalsNotAnObject, LocalsReassigned, AstroResponseHeadersReassigned, LocalImageUsedWrongly, i18nNoLocaleFoundInPath, RewriteWithBodyUsed, ForbiddenRewrite, FontFamilyNotFound, MissingGetFontFileRequestUrl, UnableToLoadLogger, ActionsReturnedInvalidDataError, ActionNotFoundError, SessionStorageInitError, SessionStorageSaveError, CacheNotEnabled, AstroError, BodySizeLimitError, __vite_import_meta_env__, ACTION_API_CONTEXT_SYMBOL, formContentTypes, dist, hasRequiredDist, distExports, DELETED_EXPIRATION, DELETED_VALUE, responseSentSymbol, identity, AstroCookie, _request, _requestValues, _outgoing, _consumed, _AstroCookies_instances, ensureParsed_fn, ensureOutgoingMap_fn, parse_fn, AstroCookies, astroCookiesSymbol, NOOP_ACTIONS_MOD, FORM_CONTENT_TYPES, SAFE_METHODS, NOOP_MIDDLEWARE_FN, MultiLevelEncodingError, MAX_DECODE_ITERATIONS, replace, ca, esca, pe, escape, DEFAULT_404_ROUTE, default404Instance, ROUTE404_RE, ROUTE500_RE, RedirectComponentInstance, RedirectSinglePageBuiltModule, VALID_PARAM_TYPES, RouteCache, escapeHTML, HTMLBytes, htmlStringSymbol, _a23, _b10, HTMLString, markHTMLString, AstroJSX, PROP_TYPE, transitionDirectivesToCopyOnIsland, dictionary, binary, headAndContentSym, astro_island_prebuilt_default, ISLAND_STYLES, RenderInstructionSymbol, voidElementNames, htmlBooleanAttributes, AMPERSAND_REGEX, DOUBLE_QUOTE_REGEX, STATIC_DIRECTIVES, INVALID_ATTR_NAME_CHAR, toIdent, toAttributeString, kebab, toStyleString, noop, BufferedRenderer, isNode, isDeno, alphabetUpperCase, decodeMap, EncodingPadding$1, DecodingPadding$1, base64Alphabet, EncodingPadding, DecodingPadding, base64DecodeMap, initializer, ZodRealError, parse, parseAsync, safeParse, safeParseAsync, encode, decode, encodeAsync, decodeAsync, safeEncode, safeDecode, safeEncodeAsync, safeDecodeAsync, _installedGroups, ZodType, ZodArray, ZodUnion, ZodIntersection, ZodEnum, ZodTransform, ZodOptional, ZodExactOptional, ZodNullable, ZodDefault, ZodPrefault, ZodNonOptional, ZodCatch, ZodPipe, ZodReadonly, ZodCustom, ZodIssueCode, ALGORITHMS, ALLOWED_DIRECTIVES, ALGORITHM, encoder$1, decoder$1, IV_LENGTH, renderTemplateResultSym, _a24, RenderTemplateResult, slotString, _a25, _b11, SlotString, internalProps, ServerIslandComponent, renderServerIslandRuntime, SERVER_ISLAND_REPLACER, Fragment, Renderer, encoder, decoder2, astroComponentInstanceSym, _a26, AstroComponentInstance, DOCTYPE_EXP, needsHeadRenderingSymbol, rendererAliases, clientOnlyValues, ASTRO_SLOT_EXP, ASTRO_STATIC_SLOT_EXP, ClientOnlyPlaceholder$1, hasTriedRenderComponentSymbol, ClientOnlyPlaceholder, SERVER_ISLAND_ROUTE, SERVER_ISLAND_COMPONENT, DEFAULT_BODY_SIZE_LIMIT, _routes, _base, _baseWithoutTrailingSlash, _buildFormat, _trailingSlash, Router, NodePool, HTMLStringCache, COMMON_HTML_PATTERNS, FORBIDDEN_PATH_KEYS, dateTimeFormat, levels, AstroLogger, AstroIntegrationLogger, SGR_REGEX, PipelineFeatures, ALL_PIPELINE_FEATURES, _router, Pipeline, _result, _slots, _logger, Slots, _strategy, _defaultLocale, _locales, _base2, _domains, I18nRouter, _i18n, _base3, _trailingSlash2, _format, _router2, I18n, EMPTY_OPTIONS, NoopAstroCache, hasWarned, _logger2, _DisabledAstroCache_instances, warn_fn, DisabledAstroCache, _pipeline, _AstroMiddleware_instances, finalize_fn, AstroMiddleware, EMPTY_SLOTS, _pipeline2, PagesHandler, Rewrites, renderOptionsSymbol, _params, _providers, _providersResolvedValues, _componentInstancePromise, _rewrites, _astroPagePartial, _domainPathname, _currentLocale, _preferredLocale, _preferredLocaleList, _FetchState_instances, stripHtmlExtension_fn, resolveRouteData_fn, computePathname_fn, applyForwardedHeaders_fn, FetchState, _ActionHandler_instances, executeAction_fn, ActionHandler, _app, _TrailingSlashHandler_instances, redirectTrailingSlash_fn, TrailingSlashHandler, APPLY_HEADERS, IS_ACTIVE, _options, _tags, _disabled, _provider, AstroCache, ROUTE_DYNAMIC_SPLIT, ROUTE_SPREAD, CACHE_KEY, _app2, CacheHandler, suspectProtoRx, suspectConstructorRx, JsonSigRx, BASE64_PREFIX, DRIVER_NAME2, memory, PERSIST_SYMBOL, DEFAULT_COOKIE_NAME, VALID_COOKIE_REGEX, unflatten, stringify, _cookies, _config, _cookieConfig, _cookieName, _storage, _data, _sessionID, _toDestroy, _toDelete, _dirty, _cookieSet, _sessionIDFromCookie, _partial, _driverFactory, _sharedStorage, _AstroSession_instances, setCookie_fn, ensureData_fn, ensureSessionID_fn, ensureStorage_fn, _AstroSession, AstroSession, SESSION_KEY, _app3, _trailingSlashHandler, _actionHandler, _astroMiddleware, _pagesHandler, _cacheHandler, _renderRouteCallback, _i18n2, _hasSession, _AstroHandler_instances, actionsAndPages_fn, AstroHandler, _app4, _handler, DefaultFetchHandler, fetchable, _app5, _astroMiddleware2, _pagesHandler2, DefaultErrorHandler, _adapterLogger, _fetchHandler, _errorHandler, _hasCustomFetchHandler, _featureCheckDone, _BaseApp_instances, warnMissingFeatures_fn, _BaseApp, BaseApp, URL_PARSE_BASE, AppPipeline, App, renderers, serializedData, _page0, _page1, pageMap, _manifest, manifestRoutes, manifest, createApp$1, createApp, IP_RE, app, server_default, workerEntry;
  var init_worker_entry_D8ZVWWpA = __esm({
    "chunks/worker-entry_D8ZVWWpA.mjs"() {
      "use strict";
      init_cloudflare_workers();
      (_a17 = globalThis.process) != null ? _a17 : globalThis.process = {};
      (_b9 = (_a18 = globalThis.process).env) != null ? _b9 : _a18.env = {};
      sessionKVBindingName = "SESSION";
      e = globalThis.process || {};
      t = e.argv || [];
      n = e.env || {};
      r$1 = !(n.NO_COLOR || t.includes(`--no-color`)) && (!!n.FORCE_COLOR || t.includes(`--color`) || e.platform === `win32` || (e.stdout || {}).isTTY && n.TERM !== `dumb` || !!n.CI);
      i = (e2, t2, n2 = e2) => (r2) => {
        let i2 = `` + r2, o2 = i2.indexOf(t2, e2.length);
        return ~o2 ? e2 + a(i2, t2, n2, o2) + t2 : e2 + i2 + t2;
      };
      a = (e2, t2, n2, r2) => {
        let i2 = ``, a2 = 0;
        do
          i2 += e2.substring(a2, r2) + n2, a2 = r2 + t2.length, r2 = e2.indexOf(t2, a2);
        while (~r2);
        return i2 + e2.substring(a2);
      };
      o = (e2 = r$1) => {
        let t2 = e2 ? i : () => String;
        return { isColorSupported: e2, reset: t2(`\x1B[0m`, `\x1B[0m`), bold: t2(`\x1B[1m`, `\x1B[22m`, `\x1B[22m\x1B[1m`), dim: t2(`\x1B[2m`, `\x1B[22m`, `\x1B[22m\x1B[2m`), italic: t2(`\x1B[3m`, `\x1B[23m`), underline: t2(`\x1B[4m`, `\x1B[24m`), inverse: t2(`\x1B[7m`, `\x1B[27m`), hidden: t2(`\x1B[8m`, `\x1B[28m`), strikethrough: t2(`\x1B[9m`, `\x1B[29m`), black: t2(`\x1B[30m`, `\x1B[39m`), red: t2(`\x1B[31m`, `\x1B[39m`), green: t2(`\x1B[32m`, `\x1B[39m`), yellow: t2(`\x1B[33m`, `\x1B[39m`), blue: t2(`\x1B[34m`, `\x1B[39m`), magenta: t2(`\x1B[35m`, `\x1B[39m`), cyan: t2(`\x1B[36m`, `\x1B[39m`), white: t2(`\x1B[37m`, `\x1B[39m`), gray: t2(`\x1B[90m`, `\x1B[39m`), bgBlack: t2(`\x1B[40m`, `\x1B[49m`), bgRed: t2(`\x1B[41m`, `\x1B[49m`), bgGreen: t2(`\x1B[42m`, `\x1B[49m`), bgYellow: t2(`\x1B[43m`, `\x1B[49m`), bgBlue: t2(`\x1B[44m`, `\x1B[49m`), bgMagenta: t2(`\x1B[45m`, `\x1B[49m`), bgCyan: t2(`\x1B[46m`, `\x1B[49m`), bgWhite: t2(`\x1B[47m`, `\x1B[49m`), blackBright: t2(`\x1B[90m`, `\x1B[39m`), redBright: t2(`\x1B[91m`, `\x1B[39m`), greenBright: t2(`\x1B[92m`, `\x1B[39m`), yellowBright: t2(`\x1B[93m`, `\x1B[39m`), blueBright: t2(`\x1B[94m`, `\x1B[39m`), magentaBright: t2(`\x1B[95m`, `\x1B[39m`), cyanBright: t2(`\x1B[96m`, `\x1B[39m`), whiteBright: t2(`\x1B[97m`, `\x1B[39m`), bgBlackBright: t2(`\x1B[100m`, `\x1B[49m`), bgRedBright: t2(`\x1B[101m`, `\x1B[49m`), bgGreenBright: t2(`\x1B[102m`, `\x1B[49m`), bgYellowBright: t2(`\x1B[103m`, `\x1B[49m`), bgBlueBright: t2(`\x1B[104m`, `\x1B[49m`), bgMagentaBright: t2(`\x1B[105m`, `\x1B[49m`), bgCyanBright: t2(`\x1B[106m`, `\x1B[49m`), bgWhiteBright: t2(`\x1B[107m`, `\x1B[49m`) };
      };
      s = o();
      MANY_LEADING_SLASHES = /^\/{2,}/;
      MANY_SLASHES = /\/{2,}/g;
      MANY_TRAILING_SLASHES = /\/{2,}$/g;
      INTERNAL_PREFIXES = /* @__PURE__ */ new Set(["/_", "/@", "/.", "//"]);
      JUST_SLASHES = /^\/{2,}$/;
      WITH_FILE_EXT = /\/[^/]+\.\w+$/;
      UNDEFINED = -1;
      HOLE = -2;
      NAN = -3;
      POSITIVE_INFINITY = -4;
      NEGATIVE_INFINITY = -5;
      NEGATIVE_ZERO = -6;
      SPARSE = -7;
      MAX_ARRAY_LEN = __pow(2, 32) - 1;
      MAX_ARRAY_INDEX = MAX_ARRAY_LEN - 1;
      DevalueError = class extends Error {
        /**
         * @param {string} message
         * @param {string[]} keys
         * @param {any} [value] - The value that failed to be serialized
         * @param {any} [root] - The root value being serialized
         */
        constructor(message, keys, value, root) {
          super(message);
          this.name = "DevalueError";
          this.path = keys.join("");
          this.value = value;
          this.root = root;
        }
      };
      object_proto_names = /* @__PURE__ */ Object.getOwnPropertyNames(Object.prototype).sort().join("\0");
      is_identifier = /^[a-zA-Z_$][a-zA-Z_$0-9]*$/;
      native = typeof Uint8Array.fromBase64 === "function";
      buffer = typeof process === "object" && ((_a19 = process.versions) == null ? void 0 : _a19.node) !== void 0;
      encode64 = native ? encode_native : buffer ? encode_buffer : encode_legacy;
      decode64 = native ? decode_native : buffer ? decode_buffer : decode_legacy;
      ACTION_QUERY_PARAMS = {
        actionName: "_action"
      };
      ACTION_RPC_ROUTE_PATTERN = "/_actions/[...path]";
      __vite_import_meta_env__$1 = { "ASSETS_PREFIX": void 0, "BASE_URL": "/", "DEV": false, "MODE": "production", "PROD": true, "SITE": void 0, "SSR": true };
      codeToStatusMap = {
        // Implemented from IANA HTTP Status Code Registry
        // https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
        BAD_REQUEST: 400,
        UNAUTHORIZED: 401,
        PAYMENT_REQUIRED: 402,
        FORBIDDEN: 403,
        NOT_FOUND: 404,
        METHOD_NOT_ALLOWED: 405,
        NOT_ACCEPTABLE: 406,
        PROXY_AUTHENTICATION_REQUIRED: 407,
        REQUEST_TIMEOUT: 408,
        CONFLICT: 409,
        GONE: 410,
        LENGTH_REQUIRED: 411,
        PRECONDITION_FAILED: 412,
        CONTENT_TOO_LARGE: 413,
        URI_TOO_LONG: 414,
        UNSUPPORTED_MEDIA_TYPE: 415,
        RANGE_NOT_SATISFIABLE: 416,
        EXPECTATION_FAILED: 417,
        MISDIRECTED_REQUEST: 421,
        UNPROCESSABLE_CONTENT: 422,
        LOCKED: 423,
        FAILED_DEPENDENCY: 424,
        TOO_EARLY: 425,
        UPGRADE_REQUIRED: 426,
        PRECONDITION_REQUIRED: 428,
        TOO_MANY_REQUESTS: 429,
        REQUEST_HEADER_FIELDS_TOO_LARGE: 431,
        UNAVAILABLE_FOR_LEGAL_REASONS: 451,
        INTERNAL_SERVER_ERROR: 500,
        NOT_IMPLEMENTED: 501,
        BAD_GATEWAY: 502,
        SERVICE_UNAVAILABLE: 503,
        GATEWAY_TIMEOUT: 504,
        HTTP_VERSION_NOT_SUPPORTED: 505,
        VARIANT_ALSO_NEGOTIATES: 506,
        INSUFFICIENT_STORAGE: 507,
        LOOP_DETECTED: 508,
        NETWORK_AUTHENTICATION_REQUIRED: 511
      };
      statusToCodeMap = Object.fromEntries(
        Object.entries(codeToStatusMap).map(([key, value]) => [value, key])
      );
      ActionError = class _ActionError extends Error {
        constructor(params) {
          super(params.message);
          __publicField(this, "type", "AstroActionError");
          __publicField(this, "code", "INTERNAL_SERVER_ERROR");
          __publicField(this, "status", 500);
          this.code = params.code;
          this.status = _ActionError.codeToStatus(params.code);
          if (params.stack) {
            this.stack = params.stack;
          }
        }
        static codeToStatus(code) {
          return codeToStatusMap[code];
        }
        static statusToCode(status) {
          var _a29;
          return (_a29 = statusToCodeMap[status]) != null ? _a29 : "INTERNAL_SERVER_ERROR";
        }
        static fromJson(body) {
          if (isInputError(body)) {
            return new ActionInputError(body.issues);
          }
          if (isActionError(body)) {
            return new _ActionError(body);
          }
          return new _ActionError({
            code: "INTERNAL_SERVER_ERROR"
          });
        }
      };
      ActionInputError = class extends ActionError {
        constructor(issues) {
          var _a29, _b13, _c;
          super({
            message: `Failed to validate: ${JSON.stringify(issues, null, 2)}`,
            code: "BAD_REQUEST"
          });
          __publicField(this, "type", "AstroActionInputError");
          // We don't expose all ZodError properties.
          // Not all properties will serialize from server to client,
          // and we don't want to import the full ZodError object into the client.
          __publicField(this, "issues");
          __publicField(this, "fields");
          this.issues = issues;
          this.fields = {};
          for (const issue2 of issues) {
            if (issue2.path.length > 0) {
              const key = issue2.path[0].toString();
              (_b13 = (_a29 = this.fields)[key]) != null ? _b13 : _a29[key] = [];
              (_c = this.fields[key]) == null ? void 0 : _c.push(issue2.message);
            }
          }
        }
      };
      actionResultErrorStack = /* @__PURE__ */ (function actionResultErrorStackFn() {
        let errorStack;
        return {
          set(stack) {
            errorStack = stack;
          },
          get() {
            return errorStack;
          }
        };
      })();
      (function(A) {
        return A[A.Static = 1] = "Static", A[A.Dynamic = 2] = "Dynamic", A[A.ImportMeta = 3] = "ImportMeta", A[A.StaticSourcePhase = 4] = "StaticSourcePhase", A[A.DynamicSourcePhase = 5] = "DynamicSourcePhase", A[A.StaticDeferPhase = 6] = "StaticDeferPhase", A[A.DynamicDeferPhase = 7] = "DynamicDeferPhase", A;
      })({});
      1 === new Uint8Array(new Uint16Array([1]).buffer)[0];
      C = () => {
        return A = "AGFzbQEAAAABKwhgAAF/YAF/AX9gAABgAn9/AX9gBH9/f38AYAN/f38Bf2ABfwBgA39/fwADPj0CAgEEBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgABBQICAgYBAQYBAQEFAQEBAQECAgIBAQEDAQEHAQMDBAUBcAECAgUHAQGCAoCAAgYPAn8BQaCLBAt/AEGgiwQLB80BHgZtZW1vcnkCAAJzYQACAWUABQJpcwAGAmllAAcCc3MACAJzZQAJAml0AAoCYWkACwJpZAAMAmlwAA0CZXMADgJlZQAPA2VscwAQA2VsZQARA2VzcwASAnJpABMCcmUAFAFmABUCbXMAFgJyYQAXA2FrcwAYA2FrZQAZA2F2cwAaA2F2ZQAbA3JzYQAcBXBhcnNlAB0LX19oZWFwX2Jhc2UDAQtfaW5pdGlhbGl6ZQABGV9faW5kaXJlY3RfZnVuY3Rpb25fdGFibGUBAAkHAQBBAQsBAAwBAQrcVD0oAEHYCkGAwAA2AgBB0ApBoIsENgIAQbgKQSo2AgBB1ApBgIAENgIACwQAEAALWQBB6AkgADYCACAAQQF0IgBBADsBoIsEQewJIABBoosEajYCAEHECUEANgIAQdQJQQA2AgBBzAlBADYCAEHICUEANgIAQdwJQQA2AgBB0AlBADYCAEGgiwQLuAEBAn9B7AlB7AkoAgAiBEEoajYCAAJAQdQJKAIAIgVFBEBBxAkgBDYCAAwBCyAFIAQ2AiQLQdQJIAQ2AgBB2AkgBTYCACAEIAA2AgggBEIANwIgIAQgA0EBRiIAOgAYIAQgAzYCFCAEQQA2AhAgBCACNgIEIAQgATYCACAEQQNBAUECIAAbIANBAkYiARs2AhwgBCACIAJBAmpBACAAGyABGzYCDCADQQFrQQFNBEBB8AlBAToAAAsLdwECf0HsCUHsCSgCACIEQRhqNgIAAkBB3AkoAgAiBUUEQEHICSAENgIADAELIAUgBDYCFAtB3AkgBDYCACAEIAM2AgwgBCACNgIIIAQgATYCBCAEIAA2AgBB4AkoAgAhACAEQQA2AhQgBCAANgIQQfAJQQE6AAALCABB9AkoAgALEwBBzAkoAgAoAgBBoIsEa0EBdQscAQF/QcwJKAIAKAIEIgBBoIsEa0EBdUF/IAAbCxMAQcwJKAIAKAIIQaCLBGtBAXULHAEBf0HMCSgCACgCDCIAQaCLBGtBAXVBfyAAGwsLAEHMCSgCACgCHAscAQF/QcwJKAIAKAIQIgBBoIsEa0EBdUF/IAAbCzUBAn9BfyEAAkACQAJAQcwJKAIAKAIUIgFBAWsOAgIBAAsgAUGgiwRrQQF1DwtBfiEACyAACwsAQcwJKAIALQAYCxMAQdAJKAIAKAIAQaCLBGtBAXULEwBB0AkoAgAoAgRBoIsEa0EBdQscAQF/QdAJKAIAKAIIIgBBoIsEa0EBdUF/IAAbCxwBAX9B0AkoAgAoAgwiAEGgiwRrQQF1QX8gABsLEwBB0AkoAgAoAhBBoIsEa0EBdQslAQF/QcwJQcwJKAIAIgBBJGpBxAkgABsoAgAiADYCACAAQQBHCyUBAX9B0AlB0AkoAgAiAEEUakHICSAAGygCACIANgIAIABBAEcLCABB+AktAAALCABB8AktAAALKwEBf0H8CUH8CSgCACIAQRBqQcwJKAIAQSBqIAAbKAIAIgA2AgAgAEEARwsTAEH8CSgCACgCAEGgiwRrQQF1CxMAQfwJKAIAKAIEQaCLBGtBAXULEwBB/AkoAgAoAghBoIsEa0EBdQsTAEH8CSgCACgCDEGgiwRrQQF1CwoAQfwJQQA2AgALow4BBn8jAEGA0ABrIgQkAEH4CUEBOgAAQYAIIQBBhApBgAg2AgBBnApBnosEIgFB6AkoAgBBAXRqIgU2AgBB8AlBADoAAEGACkEAOwEAQYIKQQA7AQBBiApBADoAAEH0CUEANgIAQeQJQQA6AABBjAogBEGAEGo2AgBBkAogBDYCAEGUCkEAOgAAA0AgACECQZgKIAFBAmoiADYCAAJAAkACfwJAAkAgASAFSQRAIAAvAQAiA0EJa0EFSQ0EAkACQAJAAkACQCADQeUAaw4FAQYGBgIACyADQSBGDQggA0EvRg0DIANBO0YNAgwFC0GCCi8BAA0BIAAQHkUNASABQQRqQYIIQQoQHw0BECBBmAooAgAhAEH4CS0AAA0BQYQKIAA2AgAgACICIQEMBQsgAS8BBEHtAEcNACAAEB5FDQAgASkABkLwgLyDoI6AOlINABAhQZgKKAIAIQALQYQKIAA2AgAMBgsgAS8BBCIAQSpHBEAgAEEvRw0CECIMBQtBARAjDAQLIAAhAUEAQeQJLQAADQIaDAELQfgJQQA6AAALA0ACQEGYCiABQQJqIgA2AgACQAJAAkAgASAFSQRAAkACQAJAAkACQAJAAkACQAJAAkACQAJAIAAvAQAiA0Egaw4QDw4IDg4ODggBBQ4OBA4OCQALAkACQAJAAkAgA0HbAGsODwURBhERDRERAxEBERERAgALIANBCWtBBUkNESADQfsAaw4DCBAJEAtBggovAQANDyAAEB5FDQ8gAUEEakGCCEEKEB8NDxAgDA8LIAEvAQRB7QBHDQ4gABAeRQ0OIAEpAAZC8IC8g6COgDpSDQ4QIQwOCyABLwEEQewARw0NIAAQHkUNDSABQQZqQbIIQQYQHw0NIAEvAQwQJEUNDUGUCkEBOgAADA0LQYIKQYIKLwEAIgBBAWo7AQAgBEGAEGogAEEDdGoiAEEBNgIAIAAgAjYCBAwMC0GCCkGCCi8BACIAQQFqOwEAIARBgBBqIABBA3RqIgBBCDYCACAAIAI2AgQMCwtBggovAQAiAEUNDEGCCiAAQQFrOwEADAoLQYAKLwEAIgBFDQlBggovAQAiA0UNCSADQQN0IARqQfgPaigCAEEFRw0JIAQgAEECdGpBBGsoAgAiACgCBA0JIAAgAkECajYCBEGYCiABQQRqNgIAQQEQJRogAEGYCigCACIANgIQQZgKIABBAms2AgAMCQtBggovAQAiAEUNCkGCCiAAQQFrIgM7AQBBgAovAQAiAEUNCCAEQYAQaiADQf//A3FBA3RqKAIAQQVHDQggBCAAQQJ0akEEaygCACIDKAIERQRAIAMgAkECajYCBAsgAyABQQRqNgIMQYAKIABBAWs7AQAMCAsCQCACLwEAQSlHDQBB1AkoAgAiAEUNACAAKAIMIAJBAmpHDQBB1AlB2AkoAgAiADYCACAABEAgAEEANgIkDAELQcQJQQA2AgALQYIKQYIKLwEAIgBBAWo7AQAgBEGAEGogAEEDdGoiAEEGQQJBlAotAAAbNgIAIAAgAjYCBEGUCkEAOgAADAcLQYIKLwEAIgBFDQhBggogAEEBayIAOwEAIARBgBBqIABB//8DcUEDdGooAgBBBEYNAwwGCyADECYMBQsCQCABLwEEIgBBKkcEQCAAQS9HDQEQIgwHC0EBECMMBgsCQCACLwEAIgEQJwRAAkACQAJAIAFBK2sOBAEIAgAICyACQQJrLwEAQTBrQf//A3FBCkkNAwwHCyACQQJrLwEAQStGDQIMBgsgAkECay8BAEEtRg0BDAULIAFBKUcNACAEQYAQakGCCi8BAEEDdGooAgQQKA0ECwJAAkBBggovAQAiAEUgAUHmAEdyDQAgBEGAEGogAEEDdGoiA0EIaygCAEEBRw0AIAJBAmsvAQBB7wBHDQEgAkEEaxApRQ0BIANBBGsoAgBBnglBAxAqRQ0BDAULIAFB/QBHDQAgBEGAEGogAEEDdGoiACgCBBArDQQgACgCAEEGRg0ECyACECwNA0GICi0AACABQS9GcSABQQBHc0UNAwJAQdwJKAIAIgBFDQAgAiAAKAIASQ0AIAIgACgCBE0NBAsDQCACQaCLBEsEQEGECiACQQJrIgI2AgAgAi8BACIBEC1FDQELCyABQf//A3EQLgRAA0AgAkGgiwRLBEBBhAogAkECayICNgIAIAIvAQAQLg0BCwsgAhAvDQQLQYgKQQE6AAAMBAtBggpBggovAQAiAEEBajsBACAEQYAQaiAAQQN0aiIAIAI2AgQgAEEDNgIACxAwDAILQYAKLwEARUGCCi8BAEVB5AktAABBf3NxcQwFCxAxQYgKQQA6AAALQYQKQZgKKAIAIgI2AgALQZgKKAIAIQEMAQsLEDJBAAshAyAEQYDQAGokACADDwsgAiEAC0GYCigCACEBDAALAAsWACAAQaCLBEYEQEEBDwsgAEECaxAzC0MBA38CQCACRQ0AA0AgAC0AACIEIAEtAAAiBUYEQCABQQFqIQEgAEEBaiEAIAJBAWsiAg0BDAILCyAEIAVrIQMLIAML0ggBBX9BmApBmAooAgAiBEEMaiIBNgIAQdwJKAIAIQNBARAlIQICQAJAAkACQAJAAkAgAUGYCigCACIARgRAIAIQNEUNAQtB4AkgBDYCAAJAAkACQCACQSpHBEAgAkH7AEcNAUGYCiAAQQJqNgIAQQEQJSECQZwKKAIAIQFBmAooAgAhAANAAkAgAkH//wNxIgJBIkYgAkEnRnJFBEAgAhA1GkGYCigCACECDAELIAIQJkGYCkGYCigCAEECaiICNgIAC0EBECUaIAAgAhA2IgJBLEYEQEGYCkGYCigCAEECajYCAEEBECUhAgsgAkH9AEYNAyAAQZgKKAIAIgBGDQggACABTQ0ACwwHC0GYCiAAQQJqNgIAQQEQJRpBmAooAgAiACAAEDYaDAILQfgJQQA6AAACQAJAAkACQAJAAkAgAkHhAGsODAIIBAEIAwgICAgIBQALIAJB9gBGDQQMBwtBmAogAEEOaiIENgIAAkACQAJAAkBBARAlQeEAaw4GAAwCDAwBDAtBmAooAgAiASkAAkLzgOSD4I3AMVINCyABLwEKEC5FDQtBmAogAUEKajYCAEEAECUaC0GYCigCACIDQQJqQaIIQQ4QHw0KAkAgAy8BECIBECQNACABQShrDgMACwALC0GYCiADQRBqNgIAQQEQJSIBQSpGBEBBmApBmAooAgBBAmo2AgBBARAlIQELIAFBKEcNAQwKC0GYCigCACIDKQACQuyAhIOwjsA5Ug0JIAMvAQoiARAkRSABQfsAR3ENCUGYCiADQQpqNgIAQQEQJSIBQfsARg0JC0GYCigCACEDIAEQNRpBmAooAgAiASADTQ0IIAAgBCADIAEQBAwKC0GYCiAAQQpqNgIAQQAQJRpBmAooAgAhAAtBmAogAEEQajYCAEEBECUiAEEqRgRAQZgKQZgKKAIAQQJqNgIAQQEQJSEACwwJCwJAIAApAAJC7ICEg7COwDlSDQAgAC8BChAtRQ0AQZgKIABBCmo2AgBBARAlIQAMCQsgAEEEaiEAC0GYCiAAQQZqNgIAQZwKKAIAIQMDQEEBECUhAEGYCigCACIBIANLDQcgABA3IQJBmAooAgAiACABRg0EIAJBPUYEQEEBEDghAkGYCigCACEACyACQSxHDQRBmAogAEECajYCAAwACwALQfAJQQE6AABBmApBmAooAgBBAmo2AgALQQEQJSEAQZgKKAIAIQECQCAAQeYARw0AIAFBAmpBnAhBBhAfDQBBmAogAUEIajYCACAEQQEQJUEAEDkgA0EUakHICSADGyECA0AgAigCACIARQ0CIABCADcCCCAAQRRqIQIMAAsAC0GYCiABQQJrNgIACw8LIAAhAQwCCyAAIARBAEEAEARBmAogAEEMajYCAA8LEDIPC0GYCiABQQJrNgIADwtBmAooAgAhASAAEDUaIAFBmAooAgAiACABIAAQBEGYCiAAQQJrNgIAC4oLAQp/QZgKQZgKKAIAIgZBDGoiCTYCAEEBECUhAEGYCigCACECAkACQAJAAkACQAJAAn8gAEEuRgRAQZgKIAJBAmo2AgBBARAlIgBB5ABHBEAgAEHzAEcEQCAAQe0ARw0HQZgKKAIAIgBBAmpBjAhBBhAfDQdBhAooAgAiARA6RQRAIAEvAQBBLkYNCAsgBiAGIABBCGpBAhADDwtBmAooAgAiAEECakGSCEEKEB8NBkGECigCACIBEDpFBEAgAS8BAEEuRg0HC0GYCiAAQQxqNgIAQQEhCEEFIQRBARAlIQBBAQwCC0GYCigCACIAKQACQuWAmIPQjIA5Ug0FQYQKKAIAIgEQOkUEQCABLwEAQS5GDQYLQZgKIABBCmo2AgBBByEEQQEhBUEBECUhAEEBIQhBAgwBCwJAAkAgAEHzAEcgAiAJTXJFBEBB8wAhACACQQJqQZIIQQoQHw0BIAIvAQwQJEUNAUGYCiACQQxqIgA2AgBBASEIQQEQJSEBIABBmAooAgAiBEcEQEHmACEAIAFB5gBHBEBBBSEEIAEhAEEBDAULQQEhAyAEQQJqQZwIQQYQHw0FIAQvAQgQLUUNBQtBmAogAjYCAEEHIQRBASEHQQAhCCABIQBBAAwDC0EHIQRBASEHIABB5ABHIAIgBkEKak1yDQFB5AAhACACKQACQuWAmIPQjIA5Ug0AIAIvAQoQJEUNAEGYCiACQQpqNgIAQSohAEEBIQVBAiEDQQEQJSIBQSpGDQRBmAogAjYCAEEAIQUgASEAQQAMAgsgAiEEDAILQQALIQMgAEEoRgRAQYwKKAIAQYIKLwEAIgVBA3RqIgBBBTYCAEGCCiAFQQFqOwEAIABBmAooAgAiAjYCBEGECigCAC8BAEEuRg0EQZgKIAJBAmo2AgBBARAlIQAgBkGYCigCACIBQQAgAhADQdQJKAIAIQMgCARAIAMgBDYCHAtBgApBgAovAQAiBEEBajsBAEGQCigCACAEQQJ0aiADNgIAAkAgAEEiRiAAQSdGckUEQAJAIABB4ABHDQBBnAooAgAhBiABIQADQCAAIgIgBk8NAQJAAkAgAEECaiIALwEAIgdB3ABrDgUAAgICBQELIAJBBGohAAwBCyAHQSRHDQAgAi8BBEH7AEcNAAsLQZgKIAFBAms2AgAPCyAAECZBmAooAgAhAAtBmAogAEECaiIANgIAAkACQAJAQQEQJUEpaw4EAQICAAILQZgKQZgKKAIAQQJqNgIAQQEQJRogAyAANgIEQZgKKAIAIQAgA0EBOgAYIAMgADYCEAwIC0GCCiAFOwEAIAMgADYCBEGYCigCACEAIANBAToAGCADIABBAmo2AgxBgAogBDsBAA8LQZgKQZgKKAIAQQJrNgIADwsgB0UgAEH7AEdyRQRAQZgKKAIAIQBBggovAQANBkGcCigCACEBA0ACQAJAIAAgAUkEQEEBECUiAEEiRiAAQSdGcg0BIABB/QBHDQJBmApBmAooAgBBAmo2AgALQQEQJSEBQZgKKAIAIQAgAUHmAEYEQCAAQQJqQZwIQQYQHw0HC0GYCiAAQQhqNgIAQQEQJSIAQSJHIABBJ0dxDQYgBiAAQQAQOQ8LIAAQJgtBmApBmAooAgBBAmoiADYCAAwACwALAkACQCAAQSdrDgQDAQEDAAsgAEEiRg0CC0GYCigCACEECyAEIAlHDQBBmAogBEECazYCAA8LIABBKkcgBXENAkGCCi8BAA0CQZgKKAIAIQBBnAooAgAhAgNAIAAgAk8NASAALwEAIgFBJ0cgAUEiR3EEQEGYCiAAQQJqIgA2AgAMAQUgBiABIAMQOQ8LAAsACxAyCw8LQZgKQZgKKAIAQQJrNgIADwtBmAogAEECazYCAAtDAQN/QZgKKAIAIQBBnAooAgAhAgNAAkAgAEECaiEBIAAgAk8NACABIQAgAS8BAEEKaw4EAAEBAAELC0GYCiABNgIAC3ABBH9BmAooAgBBAmohAUGcCigCACEEAkADQCABIgJBAmohASACIARPDQEgAS8BACEDAkAgAEUEQCADQSpGDQEgA0EKaw4EAwICAwILIANBKkcNAQsgAi8BBEEvRw0ACyACQQRqIQELQZgKIAE2AgALCwAgAEGfgIAEEDwLfQEEf0GcCigCACEDQZgKKAIAIQEDQAJAAkACQCABLwEAIgJBL0YEQCABLwECIgFBKkcEQCABQS9GDQJBLw8LIAAQIwwCCyAABEAgAhAkDQIMAwsgAhAuDQEMAgsQIgtBmApBmAooAgAiBEECaiIBNgIAIAMgBEsNAQsLIAILhgEBBH9BmAooAgAhAUGcCigCACEEAkADQAJAIAEiAkECaiEBIAIgBE8NACABLwEAIgMgAEYNAiADQdwARwRAIANBCmsOBAECAgECCyACQQRqIQEgAi8BBEENRw0BIAJBBmogASACLwEGQQpGGyEBDAELC0GYCiABNgIAEDIPC0GYCiABNgIAC24BAX8CQCAAQSlHIABBKGtB//8DcUEHSXEgAEEhayIBQQVNQQBBASABdEExcRtyRQRAIABBOmsiAUH//wNxQSVPQr+AgICgAiABrYinQQFxRXINAQtBAQ8LIABB/QBHIABB+wBrQf//A3FBBElxCy4BAX9BASEBAkAgAEGUCUEFECoNACAAQZ4JQQMQKg0AIABBpAlBAhAqIQELIAELbwEBfwJ/IAAvAQAiARAkIAFBKUZyIAFB/QBGckUEQEEAIAFB3QBHDQEaCwNAAkAgAEGgiwRNDQAgARAkRQ0AIABBAmsiAC8BACEBDAELC0EBIAFBKUYgAUHdAEZyIAFB/QBGcg0AGiABEDRBAXMLCz4BAn8CQCAAIAJBAXQiAmsiBEECaiIAQaCLBEkNACAAIAEgAhAfDQAgAEGgiwRGBEBBAQ8LIAQQMyEDCyADC4MBAQJ/QQEhAgJAAkACQAJAAkACQCAALwEAIgFBO2sOBAUEBAEACwJAIAFB5QBrDgQDBAQCAAsgAUEpRg0EIAFB+QBHDQMgAEECa0GwCUEGECoPCyAAQQJrLwEAQT1GDwsgAEECa0GoCUEEECoPCyAAQQJrQbwJQQMQKg8LQQAhAgsgAguqAwECfwJAAkACQAJAAkACQAJAAkACQAJAIAAvAQBB5ABrDhQAAQIJCQkJAwkJBAUJCQYJBwkJCAkLAkACQCAAQQJrLwEAQekAaw4EAAoKAQoLIABBBGtBuAhBAhAqDwsgAEEEa0G8CEEDECoPCwJAAkACQCAAQQJrLwEAQfMAaw4DAAECCgsgAEEEay8BACIBQeEARwRAIAFB7ABHDQogAEEGa0HlABA7DwsgAEEGa0HjABA7DwsgAEEEa0HCCEEEECoPCyAAQQRrQcoIQQYQKg8LIABBAmsvAQBB7wBHDQYgAEEEay8BAEHlAEcNBiAAQQZrLwEAIgFB8ABHBEAgAUHjAEcNByAAQQhrQdYIQQYQKg8LIABBCGtB4ghBAhAqDwsgAEECa0HmCEEEECoPC0EBIQIgAEECayIAQekAEDsNBCAAQe4IQQUQKg8LIABBAmtB5AAQOw8LIABBAmtB+AhBBxAqDwsgAEECa0GGCUEEECoPCyAAQQJrLwEAIgFB7wBHBEAgAUHlAEcNASAAQQRrQe4AEDsPCyAAQQRrQY4JQQMQKiECCyACCzQBAX8gAEGgAUYgAEEJayIBQRdNQQBBASABdEGfgIAEcRtyRQRAIAAQNCAAQS5HcQ8LQQELCwAgAEGNgIAEEDwLSAECfwJAIAAvAQAiAkHlAEcEQCACQesARw0BIABBAmtB5ghBBBAqDwsgAEECay8BAEH1AEcNACAAQQRrQcoIQQYQKiEBCyABC94BAQR/QZgKKAIAIQBBnAooAgAhAwJAAkADQAJAIAAiAUECaiEAIAEgA08NAAJAAkACQCAALwEAIgJB3ABrDgUCBAQEAQALIAJBJEcNAyABLwEEQfsARw0DQZgKIAFBBGoiAjYCAEGMCigCAEGCCi8BACIAQQN0aiIBQQQ2AgBBggogAEEBajsBACABIAI2AgQPC0GYCiAANgIAQYIKQYIKLwEAQQFrIgE7AQBBjAooAgAgAUH//wNxQQN0aigCAEEDRw0DDAQLIAFBBGohAAwBCwtBmAogADYCAAsQMgsL2wEBBH9BmAooAgAhAEGcCigCACEDA0AgAEECaiEBAkACQCAAIANPDQACQAJAAkAgAS8BACICQdsAaw4CAQIACyABIQAgAkEKaw4EAgQEAgMLAkADQAJAIAFBAmohACABIANPDQACQAJAIAAvAQAiAkHcAGsOAgAEAQsgAUEEaiEBDAILIAAhASACQQprDgQAAQEAAQsLQZgKIAA2AgAQMkGYCigCACEADAQLQZgKIAA2AgAMAwsgAEEEaiEADAILQZgKIAE2AgAQMg8LIAJBL0cNAAtBmAogADYCAAszAQF/QeQJQQE6AABBmAooAgAhAEGYCkGcCigCAEECajYCAEH0CSAAQaCLBGtBAXU2AgALPQEBfwJ/QQEgAC8BACIBQQlrQf//A3FBBUkgAUGAAXJBoAFGcg0AGkEAIAEQNEUNABogABA6IAFBLkdyCwteAQF/AkAgAEH4/wNxQShGIABBIWsiAUEFTUEAQQEgAXRBMXEbckUEQCAAQTprIgFB//8DcUElT0K/gICAoAMgAa2Ip0EBcUVyDQELQQEPCyAAQfsAa0H//wNxQQRJC1cBA39BmAooAgAhAQNAAkAgAEH//wNxIgIQJARAIAAhAwwBCyAAIQMgAhA0DQBBACEDQZgKIAFBAmoiAjYCACABLwECIQAgAiEBIAANAQsLIANB//8DcQulAQEEfwJAQZgKKAIAIgMvAQAiBUHhAEcEQCABIQIgACEEDAELQZgKIANBBGo2AgBBARAlIQJBmAooAgAhBAJAIAJBIkYgAkEnRnJFBEAgAhA1GkGYCigCACECDAELIAIQJkGYCkGYCigCAEECaiICNgIAC0EBECUhBUGYCigCACEDCyADIARHBEAgBCACQQAgACAAIAFGIgAbQQAgASAAGxAECyAFC9MEAQd/QZgKKAIAIQECQCAAQd//A3FB2wBGBEAgAS8BACEFQZgKIAFBAmo2AgBB/QBB3QAgBUH7AEYbIQZBARAlIQNBnAooAgAhBwNAAkAgBiADQf//A3EiAkZBmAooAgAiASAHS3INAAJAIAJBLkcNACABLwECQS5HDQAgAS8BBEEuRw0AQZgKIAFBBmo2AgBBARAlEDchAwwCCwJAAn8CQCAFQfsARgRAAkAgAkEiRiACQSdGckUEQCACQdsARw0BQQAQOBpBmApBmAooAgBBAmo2AgAgAQwECyACECZBmApBmAooAgBBAmo2AgAgAQwDCyABIQAgA0Ewa0H//wNxQQlLDQEDQCAAIgJBAmohACACLwECIgNBMGtB//8DcUEKSQ0AIANBwQBrIgRBHk1BAEEBIAR0Qb+AgYQEcRsNACADQeEAayIEQRdNQQBBASAEdEG/wIEEcRsNAAJAAkAgA0Eraw4EAAEAAgELIAIvAQBBIHJB5QBGDQELC0GYCiAANgIAIAEMAgsgAkEsRgRAQZgKIAFBAmo2AgBBARAlIQMMBQsgAhA3IQIMAgsgAhA1GkGYCigCAAshAEEBECUiAkE6RgRAQZgKQZgKKAIAQQJqNgIAQQEQJRA3IQIMAQsgACABTQ0AIAEgACABIAAQBAsgAkE9RgRAQQAQOCECC0GYCigCACEBIAJBLEcNAEGYCiABQQJqNgIAQQEQJSEDDAELC0GYCiABQQJqNgIADAELIAAQNRpBmAooAgAiACABTQ0AIAEgACABIAAQBAtBARAlC54NAQx/QYQKQZgKKAIAIgE2AgBBkAooAgAhCkGMCigCACEHQZwKKAIAIQxBggovAQAhCyABIgQhAgJAA0BBmAogAkECaiIJNgIAIAIgDE8EQEEAIQYMAgsCQAJAIAkvAQAiAxAuDQACQAJAAkACQAJAAkACQAJAAkACQAJAAkACQAJAAkACQAJAAkACQAJAAkBBggovAQAiBSALRw0AIAMhBgJAAkAgA0Epaw4EGgEBGgALIANBO0YgA0HdAEZyIANB/QBGcg0XCyAAIAhxQQFHDQACQCADQSJrDg4MExMTEwwFCRMTCBMTDQALAkAgA0HbAGsODwYTBxMTDhMTBBMCExMTAwALAkAgA0H7AGsOAwoTCwALIANBCmsOBBgSEhgSCwJAIANBImsODgsSEhISCwQIEhIHEhIMAAsCQCADQdsAaw4PBRIGEhINEhIDEgESEhICAAsCQCADQQprDgQVEhIVAAsgA0H7AGsOAwgRCRELIAUNECAJEB5FDRAgAkEEakGCCEEKEB8NEBAgDBALIAIvAQRB7QBHDQ8gCRAeRQ0PIAIpAAZC8IC8g6COgDpSDQ8QIQwPCyACLwEEQewARw0OIAkQHkUNDiACQQZqQbIIQQYQHw0OIAIvAQwQJEUNDkGUCkEBOgAADA4LIAcgBUEDdGoiAUEBNgIAQYIKIAVBAWo7AQAgASAENgIEDA0LIAcgBUEDdGoiAUEINgIAQYIKIAVBAWo7AQAgASAENgIEDAwLIAVFDQxBggogBUEBazsBAAwLCyAFRQ0KQYAKLwEAIgFFDQogByAFQQN0akEIaygCAEEFRw0KIAogAUECdGpBBGsoAgAiASgCBA0KIAEgBEECajYCBEGYCiACQQRqNgIAQQEQJRogAUGYCigCACIBNgIQQZgKIAFBAms2AgAMCgsgBUUNCkGCCiAFQQFrIgY7AQBBgAovAQAiAUUNCSAHIAZB//8DcUEDdGooAgBBBUcNCSAKIAFBAnRqQQRrKAIAIgYoAgRFBEAgBiAEQQJqNgIECyAGIAJBBGo2AgxBgAogAUEBazsBAAwJCwJAIAQvAQBBKUcNAEHUCSgCACIBRQ0AIAEoAgwgBEECakcNAEHUCUHYCSgCACIBNgIAIAEEQCABQQA2AiQMAQtBxAlBADYCAAsgByAFQQN0aiIBQQZBAkGUCi0AABs2AgBBggogBUEBajsBACABIAQ2AgRBlApBADoAAAwICyAFRQ0IQYIKIAVBAWsiATsBACAHIAFB//8DcUEDdGooAgBBBEYNAwwHCyADECYMBgsCQCACLwEEIgJBKkcEQCACQS9HDQEQIgwJC0EBECMMCAsCQCAELwEAIgEQJwRAAkACQAJAIAFBK2sOBAEJAgAJCyAEQQJrLwEAQTBrQf//A3FBCkkNAwwICyAEQQJrLwEAQStGDQIMBwsgBEECay8BAEEtRg0BDAYLIAFBKUcNACAHIAVBA3RqKAIEECgNBQsCQAJAIAVFIAFB5gBHcg0AIAcgBUEDdGoiAkEIaygCAEEBRw0AIARBAmsvAQBB7wBHDQEgBEEEaxApRQ0BIAJBBGsoAgBBnglBAxAqRQ0BDAYLIAFB/QBHDQAgByAFQQN0aiICKAIEECsNBSACKAIAQQZGDQULIAQQLA0EQYgKLQAAIAFBL0ZxIAFBAEdzRQ0EQdwJKAIAIgZFDQIgBCAGKAIASQ0CIAQiAiAGKAIETQ0EDAMLIAcgBUEDdGoiASAENgIEQYIKIAVBAWo7AQAgAUEDNgIACxAwDAMLIAQhAgsDQCACQaCLBEsEQCACQQJrIgIvAQAiARAtRQ0BCwsgARAuBEADQCACQaCLBEsEQEGECiACQQJrIgI2AgAgAi8BABAuDQELCyACEC8NAQtBiApBAToAAAwBCxAxQYgKQQA6AAALQYQKQZgKKAIAIgE2AgAMAQsQMgtBACEGQeQJLQAADQMCQCABIARGBEAgAEUNAUGCCi8BACALRiAIcUUNAUEBIQggASEEQZgKKAIALwEAIgZBCmsOBAUCAgUCCyADQS9GBEBBiAotAABBAXMhCAwBC0EBIQggA0Ewa0H//wNxQQpJIANB3/8DcUHBAGtB//8DcUEaSXIgA0EkRiADQd8ARnJyIANB/wBLcg0AIAEhBAJAAkAgA0Enaw4DAwEDAAsCQCADQd0Aaw4EAwEBAwALIANBIkYgA0H9AEZyDQELQQAhCAsgASEEC0GYCigCACECDAELCyADDwsgBguvBAEHfyABQSJGIAFBJ0ZyRQRAEDIPC0GYCigCACEDIAEQJiAAIANBAmpBmAooAgBBARADIAIEQEHUCSgCAEEEQQYgAkEBRhs2AhwLQZgKQZgKKAIAQQJqNgIAQQAQJSEAQZgKKAIAIQQCQAJAIABB9wBHDQAgBC8BAkHpAEcNACAELwEEQfQARw0AIAQvAQZB6ABGDQELQZgKIARBAms2AgAPC0GYCiAEQQhqNgIAAkBBARAlQfsARwRADAELQewJKAIAIQNB1AkoAgAhBUGYCigCACIGIQBBACECA0AgAyEBQZgKIABBAmo2AgBBARAlIQBBmAooAgAhBwJAAkACQCAAQSJHBEAgAEEnRw0BQScQJgwCC0EiECYMAQsgABA1IQNBmAooAgAhAAwBC0GYCigCAEECaiEAQZgKIAA2AgBBARAlIQMLIANBOkcEQAwCC0GYCkGYCigCAEECajYCAEEBECUiA0EiRiADQSdGckUEQAwCC0GYCigCACEIIAMQJkHsCSABQRRqIgM2AgBBmAooAgAhCSABQQA2AhAgASAINgIIIAEgADYCBCABIAc2AgAgASAJQQJqIgA2AgwCQCACRQRAIAUgATYCIAwBCyACIAE2AhALQZgKIAA2AgACQEEBECUiAEEsRwRAIABB/QBGDQEMAwtBmApBmAooAgBBAmoiADYCACABIQIMAQsLIAUgBjYCECAFQZgKKAIAQQJqNgIMDwtBmAogBDYCAAstAQF/AkAgAC8BAEEuRw0AIABBAmsvAQBBLkcNACAAQQRrLwEAQS5GIQELIAELNQEBfwJAIABBoIsESQ0AIAAvAQAgAUcNACAAQaCLBEYEQEEBDwsgAEECay8BABAtIQILIAILKQEBfyAAQaABRiAAQQlrIgJBF01BAEEBIAJ0IAFxG3JFBEBBAA8LQQELC8cBAQBBgggLvwF4AHAAbwByAHQAZQB0AGEAbwB1AHIAYwBlAHIAbwBtAHUAbgBjAHQAaQBvAG4AbABhAHMAcwB2AG8AeQBpAGUAZABlAGwAZQBjAG8AbgB0AGkAbgBpAG4AcwB0AGEAbgB0AHkAYgByAGUAYQByAGUAdAB1AHIAZABlAGIAdQBnAGcAZQBhAHcAYQBpAHQAaAByAHcAaABpAGwAZQBmAG8AcgBpAGYAYwBhAHQAYwBmAGkAbgBhAGwAbABlAGwAcw==", "undefined" != typeof Buffer ? Buffer.from(A, "base64") : Uint8Array.from(atob(A), (A2) => A2.charCodeAt(0));
        var A;
      };
      /*orvalho: stripped WebAssembly.init*/void 0;
      $ZodAsyncError = class extends Error {
        constructor() {
          super(`Encountered Promise during synchronous parse. Use .parseAsync() instead.`);
        }
      };
      $ZodEncodeError = class extends Error {
        constructor(name) {
          super(`Encountered unidirectional transform during encode: ${name}`);
          this.name = "ZodEncodeError";
        }
      };
      (_a20 = (_a$1 = globalThis).__zod_globalConfig) != null ? _a20 : _a$1.__zod_globalConfig = {};
      globalConfig = globalThis.__zod_globalConfig;
      EVALUATING = /* @__PURE__ */ Symbol("evaluating");
      captureStackTrace = "captureStackTrace" in Error ? Error.captureStackTrace : (..._args) => {
      };
      propertyKeyTypes = /* @__PURE__ */ new Set(["string", "number", "symbol"]);
      initializer$1 = (inst, def) => {
        inst.name = "$ZodError";
        Object.defineProperty(inst, "_zod", {
          value: inst._zod,
          enumerable: false
        });
        Object.defineProperty(inst, "issues", {
          value: def,
          enumerable: false
        });
        inst.message = JSON.stringify(def, jsonStringifyReplacer, 2);
        Object.defineProperty(inst, "toString", {
          value: () => inst.message,
          enumerable: false
        });
      };
      $ZodError = $constructor("$ZodError", initializer$1);
      $ZodRealError = $constructor("$ZodError", initializer$1, { Parent: Error });
      _parse = (_Err) => (schema, value, _ctx, _params2) => {
        var _a29;
        const ctx = _ctx ? __spreadProps(__spreadValues({}, _ctx), { async: false }) : { async: false };
        const result = schema._zod.run({ value, issues: [] }, ctx);
        if (result instanceof Promise) {
          throw new $ZodAsyncError();
        }
        if (result.issues.length) {
          const e2 = new ((_a29 = _params2 == null ? void 0 : _params2.Err) != null ? _a29 : _Err)(result.issues.map((iss) => finalizeIssue(iss, ctx, config())));
          captureStackTrace(e2, _params2 == null ? void 0 : _params2.callee);
          throw e2;
        }
        return result.value;
      };
      _parseAsync = (_Err) => (schema, value, _ctx, params) => __async(null, null, function* () {
        var _a29;
        const ctx = _ctx ? __spreadProps(__spreadValues({}, _ctx), { async: true }) : { async: true };
        let result = schema._zod.run({ value, issues: [] }, ctx);
        if (result instanceof Promise)
          result = yield result;
        if (result.issues.length) {
          const e2 = new ((_a29 = params == null ? void 0 : params.Err) != null ? _a29 : _Err)(result.issues.map((iss) => finalizeIssue(iss, ctx, config())));
          captureStackTrace(e2, params == null ? void 0 : params.callee);
          throw e2;
        }
        return result.value;
      });
      _safeParse = (_Err) => (schema, value, _ctx) => {
        const ctx = _ctx ? __spreadProps(__spreadValues({}, _ctx), { async: false }) : { async: false };
        const result = schema._zod.run({ value, issues: [] }, ctx);
        if (result instanceof Promise) {
          throw new $ZodAsyncError();
        }
        return result.issues.length ? {
          success: false,
          error: new (_Err != null ? _Err : $ZodError)(result.issues.map((iss) => finalizeIssue(iss, ctx, config())))
        } : { success: true, data: result.value };
      };
      safeParse$1 = /* @__PURE__ */ _safeParse($ZodRealError);
      _safeParseAsync = (_Err) => (schema, value, _ctx) => __async(null, null, function* () {
        const ctx = _ctx ? __spreadProps(__spreadValues({}, _ctx), { async: true }) : { async: true };
        let result = schema._zod.run({ value, issues: [] }, ctx);
        if (result instanceof Promise)
          result = yield result;
        return result.issues.length ? {
          success: false,
          error: new _Err(result.issues.map((iss) => finalizeIssue(iss, ctx, config())))
        } : { success: true, data: result.value };
      });
      safeParseAsync$1 = /* @__PURE__ */ _safeParseAsync($ZodRealError);
      _encode = (_Err) => (schema, value, _ctx) => {
        const ctx = _ctx ? __spreadProps(__spreadValues({}, _ctx), { direction: "backward" }) : { direction: "backward" };
        return _parse(_Err)(schema, value, ctx);
      };
      _decode = (_Err) => (schema, value, _ctx) => {
        return _parse(_Err)(schema, value, _ctx);
      };
      _encodeAsync = (_Err) => (schema, value, _ctx) => __async(null, null, function* () {
        const ctx = _ctx ? __spreadProps(__spreadValues({}, _ctx), { direction: "backward" }) : { direction: "backward" };
        return _parseAsync(_Err)(schema, value, ctx);
      });
      _decodeAsync = (_Err) => (schema, value, _ctx) => __async(null, null, function* () {
        return _parseAsync(_Err)(schema, value, _ctx);
      });
      _safeEncode = (_Err) => (schema, value, _ctx) => {
        const ctx = _ctx ? __spreadProps(__spreadValues({}, _ctx), { direction: "backward" }) : { direction: "backward" };
        return _safeParse(_Err)(schema, value, ctx);
      };
      _safeDecode = (_Err) => (schema, value, _ctx) => {
        return _safeParse(_Err)(schema, value, _ctx);
      };
      _safeEncodeAsync = (_Err) => (schema, value, _ctx) => __async(null, null, function* () {
        const ctx = _ctx ? __spreadProps(__spreadValues({}, _ctx), { direction: "backward" }) : { direction: "backward" };
        return _safeParseAsync(_Err)(schema, value, ctx);
      });
      _safeDecodeAsync = (_Err) => (schema, value, _ctx) => __async(null, null, function* () {
        return _safeParseAsync(_Err)(schema, value, _ctx);
      });
      $ZodCheck = /* @__PURE__ */ $constructor("$ZodCheck", (inst, def) => {
        var _a30, _b13;
        var _a29;
        (_a30 = inst._zod) != null ? _a30 : inst._zod = {};
        inst._zod.def = def;
        (_b13 = (_a29 = inst._zod).onattach) != null ? _b13 : _a29.onattach = [];
      });
      $ZodCheckMaxLength = /* @__PURE__ */ $constructor("$ZodCheckMaxLength", (inst, def) => {
        var _a30;
        var _a29;
        $ZodCheck.init(inst, def);
        (_a30 = (_a29 = inst._zod.def).when) != null ? _a30 : _a29.when = (payload) => {
          const val = payload.value;
          return !nullish(val) && val.length !== void 0;
        };
        inst._zod.onattach.push((inst2) => {
          var _a31;
          const curr = (_a31 = inst2._zod.bag.maximum) != null ? _a31 : Number.POSITIVE_INFINITY;
          if (def.maximum < curr)
            inst2._zod.bag.maximum = def.maximum;
        });
        inst._zod.check = (payload) => {
          const input = payload.value;
          const length = input.length;
          if (length <= def.maximum)
            return;
          const origin = getLengthableOrigin(input);
          payload.issues.push({
            origin,
            code: "too_big",
            maximum: def.maximum,
            inclusive: true,
            input,
            inst,
            continue: !def.abort
          });
        };
      });
      $ZodCheckMinLength = /* @__PURE__ */ $constructor("$ZodCheckMinLength", (inst, def) => {
        var _a30;
        var _a29;
        $ZodCheck.init(inst, def);
        (_a30 = (_a29 = inst._zod.def).when) != null ? _a30 : _a29.when = (payload) => {
          const val = payload.value;
          return !nullish(val) && val.length !== void 0;
        };
        inst._zod.onattach.push((inst2) => {
          var _a31;
          const curr = (_a31 = inst2._zod.bag.minimum) != null ? _a31 : Number.NEGATIVE_INFINITY;
          if (def.minimum > curr)
            inst2._zod.bag.minimum = def.minimum;
        });
        inst._zod.check = (payload) => {
          const input = payload.value;
          const length = input.length;
          if (length >= def.minimum)
            return;
          const origin = getLengthableOrigin(input);
          payload.issues.push({
            origin,
            code: "too_small",
            minimum: def.minimum,
            inclusive: true,
            input,
            inst,
            continue: !def.abort
          });
        };
      });
      $ZodCheckLengthEquals = /* @__PURE__ */ $constructor("$ZodCheckLengthEquals", (inst, def) => {
        var _a30;
        var _a29;
        $ZodCheck.init(inst, def);
        (_a30 = (_a29 = inst._zod.def).when) != null ? _a30 : _a29.when = (payload) => {
          const val = payload.value;
          return !nullish(val) && val.length !== void 0;
        };
        inst._zod.onattach.push((inst2) => {
          const bag = inst2._zod.bag;
          bag.minimum = def.length;
          bag.maximum = def.length;
          bag.length = def.length;
        });
        inst._zod.check = (payload) => {
          const input = payload.value;
          const length = input.length;
          if (length === def.length)
            return;
          const origin = getLengthableOrigin(input);
          const tooBig = length > def.length;
          payload.issues.push(__spreadProps(__spreadValues({
            origin
          }, tooBig ? { code: "too_big", maximum: def.length } : { code: "too_small", minimum: def.length }), {
            inclusive: true,
            exact: true,
            input: payload.value,
            inst,
            continue: !def.abort
          }));
        };
      });
      $ZodCheckOverwrite = /* @__PURE__ */ $constructor("$ZodCheckOverwrite", (inst, def) => {
        $ZodCheck.init(inst, def);
        inst._zod.check = (payload) => {
          payload.value = def.tx(payload.value);
        };
      });
      version = {
        major: 4,
        minor: 4,
        patch: 3
      };
      $ZodType = /* @__PURE__ */ $constructor("$ZodType", (inst, def) => {
        var _a30, _b13, _c;
        var _a29;
        inst != null ? inst : inst = {};
        inst._zod.def = def;
        inst._zod.bag = inst._zod.bag || {};
        inst._zod.version = version;
        const checks = [...(_a30 = inst._zod.def.checks) != null ? _a30 : []];
        if (inst._zod.traits.has("$ZodCheck")) {
          checks.unshift(inst);
        }
        for (const ch of checks) {
          for (const fn of ch._zod.onattach) {
            fn(inst);
          }
        }
        if (checks.length === 0) {
          (_b13 = (_a29 = inst._zod).deferred) != null ? _b13 : _a29.deferred = [];
          (_c = inst._zod.deferred) == null ? void 0 : _c.push(() => {
            inst._zod.run = inst._zod.parse;
          });
        } else {
          const runChecks = (payload, checks2, ctx) => {
            let isAborted = aborted(payload);
            let asyncResult;
            for (const ch of checks2) {
              if (ch._zod.def.when) {
                if (explicitlyAborted(payload))
                  continue;
                const shouldRun = ch._zod.def.when(payload);
                if (!shouldRun)
                  continue;
              } else if (isAborted) {
                continue;
              }
              const currLen = payload.issues.length;
              const _ = ch._zod.check(payload);
              if (_ instanceof Promise && (ctx == null ? void 0 : ctx.async) === false) {
                throw new $ZodAsyncError();
              }
              if (asyncResult || _ instanceof Promise) {
                asyncResult = (asyncResult != null ? asyncResult : Promise.resolve()).then(() => __async(null, null, function* () {
                  yield _;
                  const nextLen = payload.issues.length;
                  if (nextLen === currLen)
                    return;
                  if (!isAborted)
                    isAborted = aborted(payload, currLen);
                }));
              } else {
                const nextLen = payload.issues.length;
                if (nextLen === currLen)
                  continue;
                if (!isAborted)
                  isAborted = aborted(payload, currLen);
              }
            }
            if (asyncResult) {
              return asyncResult.then(() => {
                return payload;
              });
            }
            return payload;
          };
          const handleCanaryResult = (canary, payload, ctx) => {
            if (aborted(canary)) {
              canary.aborted = true;
              return canary;
            }
            const checkResult = runChecks(payload, checks, ctx);
            if (checkResult instanceof Promise) {
              if (ctx.async === false)
                throw new $ZodAsyncError();
              return checkResult.then((checkResult2) => inst._zod.parse(checkResult2, ctx));
            }
            return inst._zod.parse(checkResult, ctx);
          };
          inst._zod.run = (payload, ctx) => {
            if (ctx.skipChecks) {
              return inst._zod.parse(payload, ctx);
            }
            if (ctx.direction === "backward") {
              const canary = inst._zod.parse({ value: payload.value, issues: [] }, __spreadProps(__spreadValues({}, ctx), { skipChecks: true }));
              if (canary instanceof Promise) {
                return canary.then((canary2) => {
                  return handleCanaryResult(canary2, payload, ctx);
                });
              }
              return handleCanaryResult(canary, payload, ctx);
            }
            const result = inst._zod.parse(payload, ctx);
            if (result instanceof Promise) {
              if (ctx.async === false)
                throw new $ZodAsyncError();
              return result.then((result2) => runChecks(result2, checks, ctx));
            }
            return runChecks(result, checks, ctx);
          };
        }
        defineLazy(inst, "~standard", () => ({
          validate: (value) => {
            var _a31;
            try {
              const r2 = safeParse$1(inst, value);
              return r2.success ? { value: r2.data } : { issues: (_a31 = r2.error) == null ? void 0 : _a31.issues };
            } catch (_) {
              return safeParseAsync$1(inst, value).then((r2) => {
                var _a32;
                return r2.success ? { value: r2.data } : { issues: (_a32 = r2.error) == null ? void 0 : _a32.issues };
              });
            }
          },
          vendor: "zod",
          version: 1
        }));
      });
      $ZodArray = /* @__PURE__ */ $constructor("$ZodArray", (inst, def) => {
        $ZodType.init(inst, def);
        inst._zod.parse = (payload, ctx) => {
          const input = payload.value;
          if (!Array.isArray(input)) {
            payload.issues.push({
              expected: "array",
              code: "invalid_type",
              input,
              inst
            });
            return payload;
          }
          payload.value = Array(input.length);
          const proms = [];
          for (let i2 = 0; i2 < input.length; i2++) {
            const item = input[i2];
            const result = def.element._zod.run({
              value: item,
              issues: []
            }, ctx);
            if (result instanceof Promise) {
              proms.push(result.then((result2) => handleArrayResult(result2, payload, i2)));
            } else {
              handleArrayResult(result, payload, i2);
            }
          }
          if (proms.length) {
            return Promise.all(proms).then(() => payload);
          }
          return payload;
        };
      });
      $ZodUnion = /* @__PURE__ */ $constructor("$ZodUnion", (inst, def) => {
        $ZodType.init(inst, def);
        defineLazy(inst._zod, "optin", () => def.options.some((o2) => o2._zod.optin === "optional") ? "optional" : void 0);
        defineLazy(inst._zod, "optout", () => def.options.some((o2) => o2._zod.optout === "optional") ? "optional" : void 0);
        defineLazy(inst._zod, "values", () => {
          if (def.options.every((o2) => o2._zod.values)) {
            return new Set(def.options.flatMap((option) => Array.from(option._zod.values)));
          }
          return void 0;
        });
        defineLazy(inst._zod, "pattern", () => {
          if (def.options.every((o2) => o2._zod.pattern)) {
            const patterns = def.options.map((o2) => o2._zod.pattern);
            return new RegExp(`^(${patterns.map((p) => cleanRegex(p.source)).join("|")})$`);
          }
          return void 0;
        });
        const first = def.options.length === 1 ? def.options[0]._zod.run : null;
        inst._zod.parse = (payload, ctx) => {
          if (first) {
            return first(payload, ctx);
          }
          let async = false;
          const results = [];
          for (const option of def.options) {
            const result = option._zod.run({
              value: payload.value,
              issues: []
            }, ctx);
            if (result instanceof Promise) {
              results.push(result);
              async = true;
            } else {
              if (result.issues.length === 0)
                return result;
              results.push(result);
            }
          }
          if (!async)
            return handleUnionResults(results, payload, inst, ctx);
          return Promise.all(results).then((results2) => {
            return handleUnionResults(results2, payload, inst, ctx);
          });
        };
      });
      $ZodIntersection = /* @__PURE__ */ $constructor("$ZodIntersection", (inst, def) => {
        $ZodType.init(inst, def);
        inst._zod.parse = (payload, ctx) => {
          const input = payload.value;
          const left = def.left._zod.run({ value: input, issues: [] }, ctx);
          const right = def.right._zod.run({ value: input, issues: [] }, ctx);
          const async = left instanceof Promise || right instanceof Promise;
          if (async) {
            return Promise.all([left, right]).then(([left2, right2]) => {
              return handleIntersectionResults(payload, left2, right2);
            });
          }
          return handleIntersectionResults(payload, left, right);
        };
      });
      $ZodEnum = /* @__PURE__ */ $constructor("$ZodEnum", (inst, def) => {
        $ZodType.init(inst, def);
        const values = getEnumValues(def.entries);
        const valuesSet = new Set(values);
        inst._zod.values = valuesSet;
        inst._zod.pattern = new RegExp(`^(${values.filter((k) => propertyKeyTypes.has(typeof k)).map((o2) => typeof o2 === "string" ? escapeRegex(o2) : o2.toString()).join("|")})$`);
        inst._zod.parse = (payload, _ctx) => {
          const input = payload.value;
          if (valuesSet.has(input)) {
            return payload;
          }
          payload.issues.push({
            code: "invalid_value",
            values,
            input,
            inst
          });
          return payload;
        };
      });
      $ZodTransform = /* @__PURE__ */ $constructor("$ZodTransform", (inst, def) => {
        $ZodType.init(inst, def);
        inst._zod.optin = "optional";
        inst._zod.parse = (payload, ctx) => {
          if (ctx.direction === "backward") {
            throw new $ZodEncodeError(inst.constructor.name);
          }
          const _out = def.transform(payload.value, payload);
          if (ctx.async) {
            const output = _out instanceof Promise ? _out : Promise.resolve(_out);
            return output.then((output2) => {
              payload.value = output2;
              payload.fallback = true;
              return payload;
            });
          }
          if (_out instanceof Promise) {
            throw new $ZodAsyncError();
          }
          payload.value = _out;
          payload.fallback = true;
          return payload;
        };
      });
      $ZodOptional = /* @__PURE__ */ $constructor("$ZodOptional", (inst, def) => {
        $ZodType.init(inst, def);
        inst._zod.optin = "optional";
        inst._zod.optout = "optional";
        defineLazy(inst._zod, "values", () => {
          return def.innerType._zod.values ? /* @__PURE__ */ new Set([...def.innerType._zod.values, void 0]) : void 0;
        });
        defineLazy(inst._zod, "pattern", () => {
          const pattern = def.innerType._zod.pattern;
          return pattern ? new RegExp(`^(${cleanRegex(pattern.source)})?$`) : void 0;
        });
        inst._zod.parse = (payload, ctx) => {
          if (def.innerType._zod.optin === "optional") {
            const input = payload.value;
            const result = def.innerType._zod.run(payload, ctx);
            if (result instanceof Promise)
              return result.then((r2) => handleOptionalResult(r2, input));
            return handleOptionalResult(result, input);
          }
          if (payload.value === void 0) {
            return payload;
          }
          return def.innerType._zod.run(payload, ctx);
        };
      });
      $ZodExactOptional = /* @__PURE__ */ $constructor("$ZodExactOptional", (inst, def) => {
        $ZodOptional.init(inst, def);
        defineLazy(inst._zod, "values", () => def.innerType._zod.values);
        defineLazy(inst._zod, "pattern", () => def.innerType._zod.pattern);
        inst._zod.parse = (payload, ctx) => {
          return def.innerType._zod.run(payload, ctx);
        };
      });
      $ZodNullable = /* @__PURE__ */ $constructor("$ZodNullable", (inst, def) => {
        $ZodType.init(inst, def);
        defineLazy(inst._zod, "optin", () => def.innerType._zod.optin);
        defineLazy(inst._zod, "optout", () => def.innerType._zod.optout);
        defineLazy(inst._zod, "pattern", () => {
          const pattern = def.innerType._zod.pattern;
          return pattern ? new RegExp(`^(${cleanRegex(pattern.source)}|null)$`) : void 0;
        });
        defineLazy(inst._zod, "values", () => {
          return def.innerType._zod.values ? /* @__PURE__ */ new Set([...def.innerType._zod.values, null]) : void 0;
        });
        inst._zod.parse = (payload, ctx) => {
          if (payload.value === null)
            return payload;
          return def.innerType._zod.run(payload, ctx);
        };
      });
      $ZodDefault = /* @__PURE__ */ $constructor("$ZodDefault", (inst, def) => {
        $ZodType.init(inst, def);
        inst._zod.optin = "optional";
        defineLazy(inst._zod, "values", () => def.innerType._zod.values);
        inst._zod.parse = (payload, ctx) => {
          if (ctx.direction === "backward") {
            return def.innerType._zod.run(payload, ctx);
          }
          if (payload.value === void 0) {
            payload.value = def.defaultValue;
            return payload;
          }
          const result = def.innerType._zod.run(payload, ctx);
          if (result instanceof Promise) {
            return result.then((result2) => handleDefaultResult(result2, def));
          }
          return handleDefaultResult(result, def);
        };
      });
      $ZodPrefault = /* @__PURE__ */ $constructor("$ZodPrefault", (inst, def) => {
        $ZodType.init(inst, def);
        inst._zod.optin = "optional";
        defineLazy(inst._zod, "values", () => def.innerType._zod.values);
        inst._zod.parse = (payload, ctx) => {
          if (ctx.direction === "backward") {
            return def.innerType._zod.run(payload, ctx);
          }
          if (payload.value === void 0) {
            payload.value = def.defaultValue;
          }
          return def.innerType._zod.run(payload, ctx);
        };
      });
      $ZodNonOptional = /* @__PURE__ */ $constructor("$ZodNonOptional", (inst, def) => {
        $ZodType.init(inst, def);
        defineLazy(inst._zod, "values", () => {
          const v = def.innerType._zod.values;
          return v ? new Set([...v].filter((x) => x !== void 0)) : void 0;
        });
        inst._zod.parse = (payload, ctx) => {
          const result = def.innerType._zod.run(payload, ctx);
          if (result instanceof Promise) {
            return result.then((result2) => handleNonOptionalResult(result2, inst));
          }
          return handleNonOptionalResult(result, inst);
        };
      });
      $ZodCatch = /* @__PURE__ */ $constructor("$ZodCatch", (inst, def) => {
        $ZodType.init(inst, def);
        inst._zod.optin = "optional";
        defineLazy(inst._zod, "optout", () => def.innerType._zod.optout);
        defineLazy(inst._zod, "values", () => def.innerType._zod.values);
        inst._zod.parse = (payload, ctx) => {
          if (ctx.direction === "backward") {
            return def.innerType._zod.run(payload, ctx);
          }
          const result = def.innerType._zod.run(payload, ctx);
          if (result instanceof Promise) {
            return result.then((result2) => {
              payload.value = result2.value;
              if (result2.issues.length) {
                payload.value = def.catchValue(__spreadProps(__spreadValues({}, payload), {
                  error: {
                    issues: result2.issues.map((iss) => finalizeIssue(iss, ctx, config()))
                  },
                  input: payload.value
                }));
                payload.issues = [];
                payload.fallback = true;
              }
              return payload;
            });
          }
          payload.value = result.value;
          if (result.issues.length) {
            payload.value = def.catchValue(__spreadProps(__spreadValues({}, payload), {
              error: {
                issues: result.issues.map((iss) => finalizeIssue(iss, ctx, config()))
              },
              input: payload.value
            }));
            payload.issues = [];
            payload.fallback = true;
          }
          return payload;
        };
      });
      $ZodPipe = /* @__PURE__ */ $constructor("$ZodPipe", (inst, def) => {
        $ZodType.init(inst, def);
        defineLazy(inst._zod, "values", () => def.in._zod.values);
        defineLazy(inst._zod, "optin", () => def.in._zod.optin);
        defineLazy(inst._zod, "optout", () => def.out._zod.optout);
        defineLazy(inst._zod, "propValues", () => def.in._zod.propValues);
        inst._zod.parse = (payload, ctx) => {
          if (ctx.direction === "backward") {
            const right = def.out._zod.run(payload, ctx);
            if (right instanceof Promise) {
              return right.then((right2) => handlePipeResult(right2, def.in, ctx));
            }
            return handlePipeResult(right, def.in, ctx);
          }
          const left = def.in._zod.run(payload, ctx);
          if (left instanceof Promise) {
            return left.then((left2) => handlePipeResult(left2, def.out, ctx));
          }
          return handlePipeResult(left, def.out, ctx);
        };
      });
      $ZodReadonly = /* @__PURE__ */ $constructor("$ZodReadonly", (inst, def) => {
        $ZodType.init(inst, def);
        defineLazy(inst._zod, "propValues", () => def.innerType._zod.propValues);
        defineLazy(inst._zod, "values", () => def.innerType._zod.values);
        defineLazy(inst._zod, "optin", () => {
          var _a29, _b13;
          return (_b13 = (_a29 = def.innerType) == null ? void 0 : _a29._zod) == null ? void 0 : _b13.optin;
        });
        defineLazy(inst._zod, "optout", () => {
          var _a29, _b13;
          return (_b13 = (_a29 = def.innerType) == null ? void 0 : _a29._zod) == null ? void 0 : _b13.optout;
        });
        inst._zod.parse = (payload, ctx) => {
          if (ctx.direction === "backward") {
            return def.innerType._zod.run(payload, ctx);
          }
          const result = def.innerType._zod.run(payload, ctx);
          if (result instanceof Promise) {
            return result.then(handleReadonlyResult);
          }
          return handleReadonlyResult(result);
        };
      });
      $ZodCustom = /* @__PURE__ */ $constructor("$ZodCustom", (inst, def) => {
        $ZodCheck.init(inst, def);
        $ZodType.init(inst, def);
        inst._zod.parse = (payload, _) => {
          return payload;
        };
        inst._zod.check = (payload) => {
          const input = payload.value;
          const r2 = def.fn(input);
          if (r2 instanceof Promise) {
            return r2.then((r3) => handleRefineResult(r3, payload, input, inst));
          }
          handleRefineResult(r2, payload, input, inst);
          return;
        };
      });
      $ZodRegistry = class {
        constructor() {
          this._map = /* @__PURE__ */ new WeakMap();
          this._idmap = /* @__PURE__ */ new Map();
        }
        add(schema, ..._meta) {
          const meta = _meta[0];
          this._map.set(schema, meta);
          if (meta && typeof meta === "object" && "id" in meta) {
            this._idmap.set(meta.id, schema);
          }
          return this;
        }
        clear() {
          this._map = /* @__PURE__ */ new WeakMap();
          this._idmap = /* @__PURE__ */ new Map();
          return this;
        }
        remove(schema) {
          const meta = this._map.get(schema);
          if (meta && typeof meta === "object" && "id" in meta) {
            this._idmap.delete(meta.id);
          }
          this._map.delete(schema);
          return this;
        }
        get(schema) {
          var _a29;
          const p = schema._zod.parent;
          if (p) {
            const pm = __spreadValues({}, (_a29 = this.get(p)) != null ? _a29 : {});
            delete pm.id;
            const f = __spreadValues(__spreadValues({}, pm), this._map.get(schema));
            return Object.keys(f).length ? f : void 0;
          }
          return this._map.get(schema);
        }
        has(schema) {
          return this._map.has(schema);
        }
      };
      (_a22 = (_a21 = globalThis).__zod_globalRegistry) != null ? _a22 : _a21.__zod_globalRegistry = registry();
      globalRegistry = globalThis.__zod_globalRegistry;
      createToJSONSchemaMethod = (schema, processors = {}) => (params) => {
        const ctx = initializeContext(__spreadProps(__spreadValues({}, params), { processors }));
        process$1(schema, ctx);
        extractDefs(ctx, schema);
        return finalize(ctx, schema);
      };
      createStandardJSONSchemaMethod = (schema, io, processors = {}) => (params) => {
        const { libraryOptions, target } = params != null ? params : {};
        const ctx = initializeContext(__spreadProps(__spreadValues({}, libraryOptions != null ? libraryOptions : {}), { target, io, processors }));
        process$1(schema, ctx);
        extractDefs(ctx, schema);
        return finalize(ctx, schema);
      };
      enumProcessor = (schema, _ctx, json, _params2) => {
        const def = schema._zod.def;
        const values = getEnumValues(def.entries);
        if (values.every((v) => typeof v === "number"))
          json.type = "number";
        if (values.every((v) => typeof v === "string"))
          json.type = "string";
        json.enum = values;
      };
      customProcessor = (_schema, ctx, _json, _params2) => {
        if (ctx.unrepresentable === "throw") {
          throw new Error("Custom types cannot be represented in JSON Schema");
        }
      };
      transformProcessor = (_schema, ctx, _json, _params2) => {
        if (ctx.unrepresentable === "throw") {
          throw new Error("Transforms cannot be represented in JSON Schema");
        }
      };
      arrayProcessor = (schema, ctx, _json, params) => {
        const json = _json;
        const def = schema._zod.def;
        const { minimum, maximum } = schema._zod.bag;
        if (typeof minimum === "number")
          json.minItems = minimum;
        if (typeof maximum === "number")
          json.maxItems = maximum;
        json.type = "array";
        json.items = process$1(def.element, ctx, __spreadProps(__spreadValues({}, params), {
          path: [...params.path, "items"]
        }));
      };
      unionProcessor = (schema, ctx, json, params) => {
        const def = schema._zod.def;
        const isExclusive = def.inclusive === false;
        const options = def.options.map((x, i2) => process$1(x, ctx, __spreadProps(__spreadValues({}, params), {
          path: [...params.path, isExclusive ? "oneOf" : "anyOf", i2]
        })));
        if (isExclusive) {
          json.oneOf = options;
        } else {
          json.anyOf = options;
        }
      };
      intersectionProcessor = (schema, ctx, json, params) => {
        const def = schema._zod.def;
        const a2 = process$1(def.left, ctx, __spreadProps(__spreadValues({}, params), {
          path: [...params.path, "allOf", 0]
        }));
        const b = process$1(def.right, ctx, __spreadProps(__spreadValues({}, params), {
          path: [...params.path, "allOf", 1]
        }));
        const isSimpleIntersection = (val) => "allOf" in val && Object.keys(val).length === 1;
        const allOf = [
          ...isSimpleIntersection(a2) ? a2.allOf : [a2],
          ...isSimpleIntersection(b) ? b.allOf : [b]
        ];
        json.allOf = allOf;
      };
      nullableProcessor = (schema, ctx, json, params) => {
        const def = schema._zod.def;
        const inner = process$1(def.innerType, ctx, params);
        const seen = ctx.seen.get(schema);
        if (ctx.target === "openapi-3.0") {
          seen.ref = def.innerType;
          json.nullable = true;
        } else {
          json.anyOf = [inner, { type: "null" }];
        }
      };
      nonoptionalProcessor = (schema, ctx, _json, params) => {
        const def = schema._zod.def;
        process$1(def.innerType, ctx, params);
        const seen = ctx.seen.get(schema);
        seen.ref = def.innerType;
      };
      defaultProcessor = (schema, ctx, json, params) => {
        const def = schema._zod.def;
        process$1(def.innerType, ctx, params);
        const seen = ctx.seen.get(schema);
        seen.ref = def.innerType;
        json.default = JSON.parse(JSON.stringify(def.defaultValue));
      };
      prefaultProcessor = (schema, ctx, json, params) => {
        const def = schema._zod.def;
        process$1(def.innerType, ctx, params);
        const seen = ctx.seen.get(schema);
        seen.ref = def.innerType;
        if (ctx.io === "input")
          json._prefault = JSON.parse(JSON.stringify(def.defaultValue));
      };
      catchProcessor = (schema, ctx, json, params) => {
        const def = schema._zod.def;
        process$1(def.innerType, ctx, params);
        const seen = ctx.seen.get(schema);
        seen.ref = def.innerType;
        let catchValue;
        try {
          catchValue = def.catchValue(void 0);
        } catch (e2) {
          throw new Error("Dynamic catch values are not supported in JSON Schema");
        }
        json.default = catchValue;
      };
      pipeProcessor = (schema, ctx, _json, params) => {
        const def = schema._zod.def;
        const inIsTransform = def.in._zod.traits.has("$ZodTransform");
        const innerType = ctx.io === "input" ? inIsTransform ? def.out : def.in : def.out;
        process$1(innerType, ctx, params);
        const seen = ctx.seen.get(schema);
        seen.ref = innerType;
      };
      readonlyProcessor = (schema, ctx, json, params) => {
        const def = schema._zod.def;
        process$1(def.innerType, ctx, params);
        const seen = ctx.seen.get(schema);
        seen.ref = def.innerType;
        json.readOnly = true;
      };
      optionalProcessor = (schema, ctx, _json, params) => {
        const def = schema._zod.def;
        process$1(def.innerType, ctx, params);
        const seen = ctx.seen.get(schema);
        seen.ref = def.innerType;
      };
      ASTRO_VERSION = "6.4.8";
      ASTRO_GENERATOR = `Astro v${ASTRO_VERSION}`;
      REROUTE_DIRECTIVE_HEADER = "X-Astro-Reroute";
      REWRITE_DIRECTIVE_HEADER_KEY = "X-Astro-Rewrite";
      REWRITE_DIRECTIVE_HEADER_VALUE = "yes";
      NOOP_MIDDLEWARE_HEADER = "X-Astro-Noop";
      ROUTE_TYPE_HEADER = "X-Astro-Route-Type";
      INTERNAL_RESPONSE_HEADERS = [
        REROUTE_DIRECTIVE_HEADER,
        REWRITE_DIRECTIVE_HEADER_KEY,
        NOOP_MIDDLEWARE_HEADER,
        ROUTE_TYPE_HEADER
      ];
      ASTRO_ERROR_HEADER = "X-Astro-Error";
      DEFAULT_404_COMPONENT = "astro-default-404.astro";
      REDIRECT_STATUS_CODES = [301, 302, 303, 307, 308, 300, 304];
      REROUTABLE_STATUS_CODES = [404, 500];
      clientAddressSymbol = /* @__PURE__ */ Symbol.for("astro.clientAddress");
      originPathnameSymbol = /* @__PURE__ */ Symbol.for("astro.originPathname");
      pipelineSymbol = /* @__PURE__ */ Symbol.for("astro.pipeline");
      fetchStateSymbol = /* @__PURE__ */ Symbol.for("astro.fetchState");
      appSymbol = /* @__PURE__ */ Symbol.for("astro.app");
      responseSentSymbol$1 = /* @__PURE__ */ Symbol.for("astro.responseSent");
      ClientAddressNotAvailable = {
        name: "ClientAddressNotAvailable",
        title: "`Astro.clientAddress` is not available in current adapter.",
        message: (adapterName) => `\`Astro.clientAddress\` is not available in the \`${adapterName}\` adapter. File an issue with the adapter to add support.`
      };
      PrerenderClientAddressNotAvailable = {
        name: "PrerenderClientAddressNotAvailable",
        title: "`Astro.clientAddress` cannot be used inside prerendered routes.",
        message: (name) => `\`Astro.clientAddress\` cannot be used inside prerendered route ${name}.`
      };
      StaticClientAddressNotAvailable = {
        name: "StaticClientAddressNotAvailable",
        title: "`Astro.clientAddress` is not available in prerendered pages.",
        message: "`Astro.clientAddress` is only available on pages that are server-rendered.",
        hint: "See https://docs.astro.build/en/guides/on-demand-rendering/ for more information on how to enable SSR."
      };
      NoMatchingStaticPathFound = {
        name: "NoMatchingStaticPathFound",
        title: "No static path found for requested path.",
        message: (pathName) => `A \`getStaticPaths()\` route pattern was matched, but no matching static path was found for requested path \`${pathName}\`.`,
        hint: (possibleRoutes) => `Possible dynamic routes being matched: ${possibleRoutes.join(", ")}.`
      };
      OnlyResponseCanBeReturned = {
        name: "OnlyResponseCanBeReturned",
        title: "Invalid type returned by Astro page.",
        message: (route, returnedValue) => `Route \`${route ? route : ""}\` returned a \`${returnedValue}\`. Only a [Response](https://developer.mozilla.org/en-US/docs/Web/API/Response) can be returned from Astro files.`,
        hint: "See https://docs.astro.build/en/guides/on-demand-rendering/#response for more information."
      };
      MissingMediaQueryDirective = {
        name: "MissingMediaQueryDirective",
        title: "Missing value for `client:media` directive.",
        message: 'Media query not provided for `client:media` directive. A media query similar to `client:media="(max-width: 600px)"` must be provided.'
      };
      NoMatchingRenderer = {
        name: "NoMatchingRenderer",
        title: "No matching renderer found.",
        message: (componentName, componentExtension, plural, validRenderersCount) => `Unable to render \`${componentName}\`.

${validRenderersCount > 0 ? `There ${plural ? "are" : "is"} ${validRenderersCount} renderer${plural ? "s" : ""} configured in your \`astro.config.mjs\` file,
but ${plural ? "none were" : "it was not"} able to server-side render \`${componentName}\`.` : `No valid renderer was found ${componentExtension ? `for the \`.${componentExtension}\` file extension.` : `for this file extension.`}`}`,
        hint: (probableRenderers) => `Did you mean to enable the ${probableRenderers} integration?

See https://docs.astro.build/en/guides/framework-components/ for more information on how to install and configure integrations.`
      };
      NoClientOnlyHint = {
        name: "NoClientOnlyHint",
        title: "Missing hint on client:only directive.",
        message: (componentName) => `Unable to render \`${componentName}\`. When using the \`client:only\` hydration strategy, Astro needs a hint to use the correct renderer.`,
        hint: (probableRenderers) => `Did you mean to pass \`client:only="${probableRenderers}"\`? See https://docs.astro.build/en/reference/directives-reference/#clientonly for more information on \`client:only\`.`
      };
      InvalidGetStaticPathsEntry = {
        name: "InvalidGetStaticPathsEntry",
        title: "Invalid entry inside `getStaticPaths()`'s return value.",
        message: (entryType) => `Invalid entry returned by \`getStaticPaths()\`. Expected an object, got \`${entryType}\`.`,
        hint: "If you're using a `.map` call, you might be looking for `.flatMap()` instead. See https://docs.astro.build/en/reference/routing-reference/#getstaticpaths for more information on `getStaticPaths()`."
      };
      InvalidGetStaticPathsReturn = {
        name: "InvalidGetStaticPathsReturn",
        title: "Invalid value returned by `getStaticPaths()`.",
        message: (returnType) => `Invalid type returned by \`getStaticPaths()\`. Expected an \`array\`, got \`${returnType}\`.`,
        hint: "See https://docs.astro.build/en/reference/routing-reference/#getstaticpaths for more information on `getStaticPaths()`."
      };
      GetStaticPathsExpectedParams = {
        name: "GetStaticPathsExpectedParams",
        title: "Missing params property on `getStaticPaths()` route.",
        message: "Missing or empty required `params` property on `getStaticPaths()` route.",
        hint: "See https://docs.astro.build/en/reference/routing-reference/#getstaticpaths for more information on `getStaticPaths()`."
      };
      GetStaticPathsInvalidRouteParam = {
        name: "GetStaticPathsInvalidRouteParam",
        title: "Invalid route parameter returned by `getStaticPaths()`.",
        message: (key, value, valueType) => `Invalid \`getStaticPaths()\` route parameter for \`${key}\`. Expected a string or undefined, received \`${valueType}\` (\`${value}\`).`,
        hint: "See https://docs.astro.build/en/reference/routing-reference/#getstaticpaths for more information on `getStaticPaths()`."
      };
      GetStaticPathsRequired = {
        name: "GetStaticPathsRequired",
        title: "`getStaticPaths()` function required for dynamic routes.",
        message: "`getStaticPaths()` function is required for dynamic routes. Make sure that you `export` a `getStaticPaths()` function from your dynamic route.",
        hint: `See https://docs.astro.build/en/guides/routing/#dynamic-routes for more information on dynamic routes.

	If you meant for this route to be server-rendered, set \`export const prerender = false;\` in the page.`
      };
      ReservedSlotName = {
        name: "ReservedSlotName",
        title: "Invalid slot name.",
        message: (slotName) => `Unable to create a slot named \`${slotName}\`. \`${slotName}\` is a reserved slot name. Please update the name of this slot.`
      };
      NoMatchingImport = {
        name: "NoMatchingImport",
        title: "No import found for component.",
        message: (componentName) => `Could not render \`${componentName}\`. No matching import has been found for \`${componentName}\`.`,
        hint: "Please make sure the component is properly imported."
      };
      InvalidComponentArgs = {
        name: "InvalidComponentArgs",
        title: "Invalid component arguments.",
        message: (name) => `Invalid arguments passed to${name ? ` <${name}>` : ""} component.`,
        hint: "Astro components cannot be rendered directly via function call, such as `Component()` or `{items.map(Component)}`."
      };
      PageNumberParamNotFound = {
        name: "PageNumberParamNotFound",
        title: "Page number param not found.",
        message: (paramName) => `[paginate()] page number param \`${paramName}\` not found in your filepath.`,
        hint: "Rename your file to `[page].astro` or `[...page].astro`."
      };
      ImageMissingAlt = {
        name: "ImageMissingAlt",
        title: 'Image missing required "alt" property.',
        message: 'Image missing "alt" property. "alt" text is required to describe important images on the page.',
        hint: 'Use an empty string ("") for decorative images.'
      };
      InvalidImageService = {
        name: "InvalidImageService",
        title: "Error while loading image service.",
        message: "There was an error loading the configured image service. Please see the stack trace for more information."
      };
      MissingImageDimension = {
        name: "MissingImageDimension",
        title: "Missing image dimensions.",
        message: (missingDimension, imageURL) => `Missing ${missingDimension === "both" ? "width and height attributes" : `${missingDimension} attribute`} for ${imageURL}. When using remote images, both dimensions are required in order to avoid CLS.`,
        hint: "If your image is inside your `src` folder, you probably meant to import it instead. See [the Imports guide for more information](https://docs.astro.build/en/guides/imports/#other-assets). You can also use `inferSize={true}` for remote images to get the original dimensions."
      };
      FailedToFetchRemoteImageDimensions = {
        name: "FailedToFetchRemoteImageDimensions",
        title: "Failed to retrieve remote image dimensions.",
        message: (imageURL) => `Failed to get the dimensions for ${imageURL}.`,
        hint: "Verify your remote image URL is accurate, and that you are not using `inferSize` with a file located in your `public/` folder."
      };
      RemoteImageNotAllowed = {
        name: "RemoteImageNotAllowed",
        title: "Remote image is not allowed.",
        message: (imageURL) => `Remote image ${imageURL} is not allowed by your image configuration.`,
        hint: "Update `image.domains` or `image.remotePatterns`, or remove `inferSize` for this image."
      };
      UnsupportedImageFormat = {
        name: "UnsupportedImageFormat",
        title: "Unsupported image format.",
        message: (format, imagePath, supportedFormats) => `Received unsupported format \`${format}\` from \`${imagePath}\`. Currently only ${supportedFormats.join(
          ", "
        )} are supported by our image services.`,
        hint: "Using an `img` tag directly instead of the `Image` component might be what you're looking for."
      };
      UnsupportedImageConversion = {
        name: "UnsupportedImageConversion",
        title: "Unsupported image conversion.",
        message: "Converting between vector (such as SVGs) and raster (such as PNGs and JPEGs) images is not currently supported."
      };
      PrerenderDynamicEndpointPathCollide = {
        name: "PrerenderDynamicEndpointPathCollide",
        title: "Prerendered dynamic endpoint has path collision.",
        message: (pathname) => `Could not render \`${pathname}\` with an \`undefined\` param as the generated path will collide during prerendering. Prevent passing \`undefined\` as \`params\` for the endpoint's \`getStaticPaths()\` function, or add an additional extension to the endpoint's filename.`,
        hint: (filename) => `Rename \`${filename}\` to \`${filename.replace(/\.(?:js|ts)/, (m) => `.json` + m)}\``
      };
      ExpectedImage = {
        name: "ExpectedImage",
        title: "Expected src to be an image.",
        message: (src, typeofOptions, fullOptions) => `Expected \`src\` property for \`getImage\` or \`<Image />\` to be either an ESM imported image or a string with the path of a remote image. Received \`${src}\` (type: \`${typeofOptions}\`).

Full serialized options received: \`${fullOptions}\`.`,
        hint: "This error can often happen because of a wrong path. Make sure the path to your image is correct. If you're passing an async function, make sure to call and await it."
      };
      ExpectedImageOptions = {
        name: "ExpectedImageOptions",
        title: "Expected image options.",
        message: (options) => `Expected \`getImage()\` parameter to be an object. Received \`${options}\`.`
      };
      ExpectedNotESMImage = {
        name: "ExpectedNotESMImage",
        title: "Expected image options, not an ESM-imported image.",
        message: "An ESM-imported image cannot be passed directly to `getImage()`. Instead, pass an object with the image in the `src` property.",
        hint: "Try changing `getImage(myImage)` to `getImage({ src: myImage })`"
      };
      IncompatibleDescriptorOptions = {
        name: "IncompatibleDescriptorOptions",
        title: "Cannot set both `densities` and `widths`.",
        message: "Only one of `densities` or `widths` can be specified. In most cases, you'll probably want to use only `widths` if you require specific widths.",
        hint: "Those attributes are used to construct a `srcset` attribute, which cannot have both `x` and `w` descriptors."
      };
      NoImageMetadata = {
        name: "NoImageMetadata",
        title: "Could not process image metadata.",
        message: (imagePath) => `Could not process image metadata${imagePath ? ` for \`${imagePath}\`` : ""}.`,
        hint: "This is often caused by a corrupted or malformed image. Re-exporting the image from your image editor may fix this issue."
      };
      ResponseSentError = {
        name: "ResponseSentError",
        title: "Unable to set response.",
        message: "The response has already been sent to the browser and cannot be altered."
      };
      MiddlewareNoDataOrNextCalled = {
        name: "MiddlewareNoDataOrNextCalled",
        title: "The middleware didn't return a `Response`.",
        message: "Make sure your middleware returns a `Response` object, either directly or by returning the `Response` from calling the `next` function."
      };
      MiddlewareNotAResponse = {
        name: "MiddlewareNotAResponse",
        title: "The middleware returned something that is not a `Response` object.",
        message: "Any data returned from middleware must be a valid `Response` object."
      };
      EndpointDidNotReturnAResponse = {
        name: "EndpointDidNotReturnAResponse",
        title: "The endpoint did not return a `Response`.",
        message: "An endpoint must return either a `Response`, or a `Promise` that resolves with a `Response`."
      };
      LocalsNotAnObject = {
        name: "LocalsNotAnObject",
        title: "Value assigned to `locals` is not accepted.",
        message: "`locals` can only be assigned to an object. Other values like numbers, strings, etc. are not accepted.",
        hint: "If you tried to remove some information from the `locals` object, try to use `delete` or set the property to `undefined`."
      };
      LocalsReassigned = {
        name: "LocalsReassigned",
        title: "`locals` must not be reassigned.",
        message: "`locals` cannot be assigned directly.",
        hint: "Set a `locals` property instead."
      };
      AstroResponseHeadersReassigned = {
        name: "AstroResponseHeadersReassigned",
        title: "`Astro.response.headers` must not be reassigned.",
        message: "Individual headers can be added to and removed from `Astro.response.headers`, but it must not be replaced with another instance of `Headers` altogether.",
        hint: "Consider using `Astro.response.headers.add()`, and `Astro.response.headers.delete()`."
      };
      LocalImageUsedWrongly = {
        name: "LocalImageUsedWrongly",
        title: "Local images must be imported.",
        message: (imageFilePath) => `\`Image\`'s and \`getImage\`'s \`src\` parameter must be an imported image or a URL, it cannot be a string filepath. Received \`${imageFilePath}\`.`,
        hint: "If you want to use an image from your `src` folder, you need to either import it or if the image is coming from a content collection, use the [image() schema helper](https://docs.astro.build/en/guides/images/#images-in-content-collections). See https://docs.astro.build/en/reference/modules/astro-assets/#src-required for more information on the `src` property."
      };
      i18nNoLocaleFoundInPath = {
        name: "i18nNoLocaleFoundInPath",
        title: "The path doesn't contain any locale.",
        message: "You tried to use an i18n utility on a path that doesn't contain any locale. You can use `pathHasLocale` first to determine if the path has a locale."
      };
      RewriteWithBodyUsed = {
        name: "RewriteWithBodyUsed",
        title: "Cannot use `Astro.rewrite()` after the request body has been read.",
        message: "`Astro.rewrite()` cannot be used if the request body has already been read. If you need to read the body, first clone the request."
      };
      ForbiddenRewrite = {
        name: "ForbiddenRewrite",
        title: "Forbidden rewrite to a static route.",
        message: (from, to, component) => `You tried to rewrite the on-demand route '${from}' with the static route '${to}', when using the 'server' output. 

The static route '${to}' is rendered by the component
'${component}', which is marked as prerendered. This is a forbidden operation because during the build, the component '${component}' is compiled to an
HTML file, which can't be retrieved at runtime by Astro.`,
        hint: (component) => `Add \`export const prerender = false\` to the component '${component}', or use \`Astro.redirect()\`.`
      };
      FontFamilyNotFound = {
        name: "FontFamilyNotFound",
        title: "Font family not found.",
        message: (family) => `No data was found for the \`"${family}"\` family passed to the \`<Font>\` component.`,
        hint: "This is often caused by a typo. Check that the `<Font />` component is using a `cssVariable` specified in your config."
      };
      MissingGetFontFileRequestUrl = {
        name: "MissingGetFontFileRequestUrl",
        title: "`experimental_getFontFileURL()` requires the request URL with on-demand rendering.",
        hint: "Pass the request URL as the 2nd argument, for example `Astro.url`."
      };
      UnableToLoadLogger = {
        name: "UnableToLoadLogger",
        title: "Unable to load the logger.",
        message: (path) => `Couldn't load the logger at given path "${path}".`
      };
      ActionsReturnedInvalidDataError = {
        name: "ActionsReturnedInvalidDataError",
        title: "Action handler returned invalid data.",
        message: (error2) => `Action handler returned invalid data. Handlers should return serializable data types like objects, arrays, strings, and numbers. Parse error: ${error2}`,
        hint: "See the devalue library for all supported types: https://github.com/rich-harris/devalue"
      };
      ActionNotFoundError = {
        name: "ActionNotFoundError",
        title: "Action not found.",
        message: (actionName) => `The server received a request for an action named \`${actionName}\` but could not find a match. If you renamed an action, check that you've updated your \`actions/index\` file and your calling code to match.`,
        hint: "You can run `astro check` to detect type errors caused by mismatched action names."
      };
      SessionStorageInitError = {
        name: "SessionStorageInitError",
        title: "Session storage could not be initialized.",
        message: (error2, driver) => `Error when initializing session storage${driver ? ` with driver \`${driver}\`` : ""}. \`${error2 != null ? error2 : ""}\``,
        hint: "For more information, see https://docs.astro.build/en/guides/sessions/"
      };
      SessionStorageSaveError = {
        name: "SessionStorageSaveError",
        title: "Session data could not be saved.",
        message: (error2, driver) => `Error when saving session data${driver ? ` with driver \`${driver}\`` : ""}. \`${error2 != null ? error2 : ""}\``,
        hint: "For more information, see https://docs.astro.build/en/guides/sessions/"
      };
      CacheNotEnabled = {
        name: "CacheNotEnabled",
        title: "Cache is not enabled.",
        message: "`Astro.cache` is not available because the cache feature is not enabled. To use caching, configure a cache provider in your Astro config under `experimental.cache`.",
        hint: 'Use an adapter that provides a default cache provider, or set one explicitly: `experimental: { cache: { provider: "..." } }`. See https://docs.astro.build/en/reference/experimental-flags/route-caching/.'
      };
      AstroError = class extends Error {
        constructor(props, options) {
          const { name, title, message, stack, location, hint, frame } = props;
          super(message, options);
          __publicField(this, "loc");
          __publicField(this, "title");
          __publicField(this, "hint");
          __publicField(this, "frame");
          __publicField(this, "type", "AstroError");
          this.title = title;
          this.name = name;
          if (message) this.message = message;
          this.stack = stack ? stack : this.stack;
          this.loc = location;
          this.hint = hint;
          this.frame = frame;
        }
        setLocation(location) {
          this.loc = location;
        }
        setName(name) {
          this.name = name;
        }
        setMessage(message) {
          this.message = message;
        }
        setHint(hint) {
          this.hint = hint;
        }
        setFrame(source, location) {
          this.frame = codeFrame(source, location);
        }
        static is(err) {
          return (err == null ? void 0 : err.type) === "AstroError";
        }
      };
      BodySizeLimitError = class extends Error {
        constructor(limit) {
          super(`Request body exceeds the configured limit of ${limit} bytes`);
          __publicField(this, "limit");
          this.name = "BodySizeLimitError";
          this.limit = limit;
        }
      };
      __vite_import_meta_env__ = { "ASSETS_PREFIX": void 0, "BASE_URL": "/", "DEV": false, "MODE": "production", "PROD": true, "SITE": void 0, "SSR": true };
      ACTION_API_CONTEXT_SYMBOL = /* @__PURE__ */ Symbol.for("astro.actionAPIContext");
      formContentTypes = ["application/x-www-form-urlencoded", "multipart/form-data"];
      dist = {};
      distExports = /* @__PURE__ */ requireDist();
      DELETED_EXPIRATION = /* @__PURE__ */ new Date(0);
      DELETED_VALUE = "deleted";
      responseSentSymbol = /* @__PURE__ */ Symbol.for("astro.responseSent");
      identity = (value) => value;
      AstroCookie = class {
        constructor(value) {
          __publicField(this, "value");
          this.value = value;
        }
        json() {
          if (this.value === void 0) {
            throw new Error(`Cannot convert undefined to an object.`);
          }
          return JSON.parse(this.value);
        }
        number() {
          return Number(this.value);
        }
        boolean() {
          if (this.value === "false") return false;
          if (this.value === "0") return false;
          return Boolean(this.value);
        }
      };
      AstroCookies = class {
        constructor(request) {
          __privateAdd(this, _AstroCookies_instances);
          __privateAdd(this, _request);
          __privateAdd(this, _requestValues);
          __privateAdd(this, _outgoing);
          __privateAdd(this, _consumed);
          __privateSet(this, _request, request);
          __privateSet(this, _requestValues, null);
          __privateSet(this, _outgoing, null);
          __privateSet(this, _consumed, false);
        }
        /**
         * Astro.cookies.delete(key) is used to delete a cookie. Using this method will result
         * in a Set-Cookie header added to the response.
         * @param key The cookie to delete
         * @param options Options related to this deletion, such as the path of the cookie.
         */
        delete(key, options) {
          const _a29 = options || {}, {
            maxAge: _ignoredMaxAge,
            expires: _ignoredExpires
          } = _a29, sanitizedOptions = __objRest(_a29, [
            // @ts-expect-error
            "maxAge",
            // @ts-expect-error
            "expires"
          ]);
          const serializeOptions = __spreadValues({
            expires: DELETED_EXPIRATION
          }, sanitizedOptions);
          __privateMethod(this, _AstroCookies_instances, ensureOutgoingMap_fn).call(this).set(key, [
            DELETED_VALUE,
            distExports.serialize(key, DELETED_VALUE, serializeOptions),
            false
          ]);
        }
        /**
         * Astro.cookies.get(key) is used to get a cookie value. The cookie value is read from the
         * request. If you have set a cookie via Astro.cookies.set(key, value), the value will be taken
         * from that set call, overriding any values already part of the request.
         * @param key The cookie to get.
         * @returns An object containing the cookie value as well as convenience methods for converting its value.
         */
        get(key, options = void 0) {
          var _a29, _b13;
          if ((_a29 = __privateGet(this, _outgoing)) == null ? void 0 : _a29.has(key)) {
            let [serializedValue, , isSetValue] = __privateGet(this, _outgoing).get(key);
            if (isSetValue) {
              return new AstroCookie(serializedValue);
            } else {
              return void 0;
            }
          }
          const decode2 = (_b13 = options == null ? void 0 : options.decode) != null ? _b13 : decodeURIComponent;
          const values = __privateMethod(this, _AstroCookies_instances, ensureParsed_fn).call(this);
          if (key in values) {
            const value = values[key];
            if (value) {
              let decodedValue;
              try {
                decodedValue = decode2(value);
              } catch (_error) {
                decodedValue = value;
              }
              return new AstroCookie(decodedValue);
            }
          }
        }
        /**
         * Astro.cookies.has(key) returns a boolean indicating whether this cookie is either
         * part of the initial request or set via Astro.cookies.set(key)
         * @param key The cookie to check for.
         * @param _options This parameter is no longer used.
         * @returns
         */
        has(key, _options2) {
          var _a29;
          if ((_a29 = __privateGet(this, _outgoing)) == null ? void 0 : _a29.has(key)) {
            let [, , isSetValue] = __privateGet(this, _outgoing).get(key);
            return isSetValue;
          }
          const values = __privateMethod(this, _AstroCookies_instances, ensureParsed_fn).call(this);
          return values[key] !== void 0;
        }
        /**
         * Astro.cookies.set(key, value) is used to set a cookie's value. If provided
         * an object it will be stringified via JSON.stringify(value). Additionally you
         * can provide options customizing how this cookie will be set, such as setting httpOnly
         * in order to prevent the cookie from being read in client-side JavaScript.
         * @param key The name of the cookie to set.
         * @param value A value, either a string or other primitive or an object.
         * @param options Options for the cookie, such as the path and security settings.
         */
        set(key, value, options) {
          if (__privateGet(this, _consumed)) {
            const warning = new Error(
              "Astro.cookies.set() was called after the cookies had already been sent to the browser.\nThis may have happened if this method was called in an imported component.\nPlease make sure that Astro.cookies.set() is only called in the frontmatter of the main page."
            );
            warning.name = "Warning";
            console.warn(warning);
          }
          let serializedValue;
          if (typeof value === "string") {
            serializedValue = value;
          } else {
            let toStringValue = value.toString();
            if (toStringValue === Object.prototype.toString.call(value)) {
              serializedValue = JSON.stringify(value);
            } else {
              serializedValue = toStringValue;
            }
          }
          const serializeOptions = {};
          if (options) {
            Object.assign(serializeOptions, options);
          }
          __privateMethod(this, _AstroCookies_instances, ensureOutgoingMap_fn).call(this).set(key, [
            serializedValue,
            distExports.serialize(key, serializedValue, serializeOptions),
            true
          ]);
          if (__privateGet(this, _request)[responseSentSymbol]) {
            throw new AstroError(__spreadValues({}, ResponseSentError));
          }
        }
        /**
         * Merges a new AstroCookies instance into the current instance. Any new cookies
         * will be added to the current instance, overwriting any existing cookies with the same name.
         */
        merge(cookies) {
          const outgoing = __privateGet(cookies, _outgoing);
          if (outgoing) {
            for (const [key, value] of outgoing) {
              __privateMethod(this, _AstroCookies_instances, ensureOutgoingMap_fn).call(this).set(key, value);
            }
          }
        }
        /**
         * Astro.cookies.header() returns an iterator for the cookies that have previously
         * been set by either Astro.cookies.set() or Astro.cookies.delete().
         * This method is primarily used by adapters to set the header on outgoing responses.
         * @returns
         */
        *headers() {
          if (__privateGet(this, _outgoing) == null) return;
          for (const [, value] of __privateGet(this, _outgoing)) {
            yield value[1];
          }
        }
        /**
         * Marks the cookies as consumed and returns the header values.
         * After consumption, any subsequent `set()` calls will warn.
         */
        consume() {
          __privateSet(this, _consumed, true);
          return this.headers();
        }
        /**
         * @deprecated Use the instance method `cookies.consume()` instead.
         * Kept for backward compatibility with adapters.
         */
        static consume(cookies) {
          return cookies.consume();
        }
      };
      _request = new WeakMap();
      _requestValues = new WeakMap();
      _outgoing = new WeakMap();
      _consumed = new WeakMap();
      _AstroCookies_instances = new WeakSet();
      ensureParsed_fn = function() {
        if (!__privateGet(this, _requestValues)) {
          __privateMethod(this, _AstroCookies_instances, parse_fn).call(this);
        }
        if (!__privateGet(this, _requestValues)) {
          __privateSet(this, _requestValues, /* @__PURE__ */ Object.create(null));
        }
        return __privateGet(this, _requestValues);
      };
      ensureOutgoingMap_fn = function() {
        if (!__privateGet(this, _outgoing)) {
          __privateSet(this, _outgoing, /* @__PURE__ */ new Map());
        }
        return __privateGet(this, _outgoing);
      };
      parse_fn = function() {
        const raw = __privateGet(this, _request).headers.get("cookie");
        if (!raw) {
          return;
        }
        __privateSet(this, _requestValues, distExports.parse(raw, { decode: identity }));
      };
      astroCookiesSymbol = /* @__PURE__ */ Symbol.for("astro.cookies");
      NOOP_ACTIONS_MOD = {
        server: {}
      };
      FORM_CONTENT_TYPES = [
        "application/x-www-form-urlencoded",
        "multipart/form-data",
        "text/plain"
      ];
      SAFE_METHODS = ["GET", "HEAD", "OPTIONS"];
      NOOP_MIDDLEWARE_FN = (_ctx, next) => __async(null, null, function* () {
        const response = yield next();
        response.headers.set(NOOP_MIDDLEWARE_HEADER, "true");
        return response;
      });
      MultiLevelEncodingError = class extends Error {
        constructor() {
          super("URL encoding depth exceeded the maximum number of decode iterations");
          this.name = "MultiLevelEncodingError";
        }
      };
      MAX_DECODE_ITERATIONS = 10;
      ({ replace } = "");
      ca = /[&<>'"]/g;
      esca = {
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "'": "&#39;",
        '"': "&quot;"
      };
      pe = (m) => esca[m];
      escape = (es) => replace.call(es, ca, pe);
      DEFAULT_404_ROUTE = {
        component: DEFAULT_404_COMPONENT,
        params: [],
        pattern: /^\/404\/?$/,
        prerender: false,
        pathname: "/404",
        segments: [[{ content: "404", dynamic: false, spread: false }]],
        type: "page",
        route: "/404",
        fallbackRoutes: [],
        isIndex: false,
        origin: "internal",
        distURL: []
      };
      default404Page.isAstroComponentFactory = true;
      default404Instance = {
        default: default404Page
      };
      ROUTE404_RE = /^\/404\/?$/;
      ROUTE500_RE = /^\/500\/?$/;
      RedirectComponentInstance = {
        default() {
          return new Response(null, {
            status: 301
          });
        }
      };
      RedirectSinglePageBuiltModule = {
        page: () => Promise.resolve(RedirectComponentInstance),
        onRequest: (_, next) => next()
      };
      VALID_PARAM_TYPES = ["string", "undefined"];
      RouteCache = class {
        constructor(logger, runtimeMode = "production") {
          __publicField(this, "logger");
          __publicField(this, "cache", {});
          __publicField(this, "runtimeMode");
          this.logger = logger;
          this.runtimeMode = runtimeMode;
        }
        /** Clear the cache. */
        clearAll() {
          this.cache = {};
        }
        set(route, entry) {
          var _a29;
          const key = this.key(route);
          if (this.runtimeMode === "production" && ((_a29 = this.cache[key]) == null ? void 0 : _a29.staticPaths)) {
            this.logger.warn(null, `Internal Warning: route cache overwritten. (${key})`);
          }
          this.cache[key] = entry;
        }
        get(route) {
          return this.cache[this.key(route)];
        }
        key(route) {
          return `${route.route}_${route.component}`;
        }
      };
      escapeHTML = escape;
      HTMLBytes = class extends Uint8Array {
      };
      Object.defineProperty(HTMLBytes.prototype, Symbol.toStringTag, {
        get() {
          return "HTMLBytes";
        }
      });
      htmlStringSymbol = /* @__PURE__ */ Symbol.for("astro:html-string");
      HTMLString = class extends (_b10 = String, _a23 = htmlStringSymbol, _b10) {
        constructor() {
          super(...arguments);
          __publicField(this, _a23, true);
        }
      };
      markHTMLString = (value) => {
        if (isHTMLString(value)) {
          return value;
        }
        if (typeof value === "string") {
          return new HTMLString(value);
        }
        return value;
      };
      AstroJSX = "astro:jsx";
      PROP_TYPE = {
        Value: 0,
        JSON: 1,
        // Actually means Array
        RegExp: 2,
        Date: 3,
        Map: 4,
        Set: 5,
        BigInt: 6,
        URL: 7,
        Uint8Array: 8,
        Uint16Array: 9,
        Uint32Array: 10,
        Infinity: 11
      };
      transitionDirectivesToCopyOnIsland = Object.freeze([
        "data-astro-transition-scope",
        "data-astro-transition-persist",
        "data-astro-transition-persist-props"
      ]);
      dictionary = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXY";
      binary = dictionary.length;
      headAndContentSym = /* @__PURE__ */ Symbol.for("astro.headAndContent");
      astro_island_prebuilt_default = `(()=>{var g=Object.defineProperty;var w=(c,s,d)=>s in c?g(c,s,{enumerable:!0,configurable:!0,writable:!0,value:d}):c[s]=d;var l=(c,s,d)=>w(c,typeof s!="symbol"?s+"":s,d);var E=new Set(["__proto__","constructor","prototype"]);{let c={0:t=>y(t),1:t=>d(t),2:t=>new RegExp(t),3:t=>new Date(t),4:t=>new Map(d(t)),5:t=>new Set(d(t)),6:t=>BigInt(t),7:t=>new URL(t),8:t=>new Uint8Array(t),9:t=>new Uint16Array(t),10:t=>new Uint32Array(t),11:t=>Number.POSITIVE_INFINITY*t},s=t=>{let[p,e]=t;return p in c?c[p](e):void 0},d=t=>t.map(s),y=t=>typeof t!="object"||t===null?t:Object.fromEntries(Object.entries(t).map(([p,e])=>[p,s(e)]));class f extends HTMLElement{constructor(){super(...arguments);l(this,"Component");l(this,"hydrator");l(this,"hydrate",async()=>{var b;if(!this.hydrator||!this.isConnected)return;let e=(b=this.parentElement)==null?void 0:b.closest("astro-island[ssr]");if(e){e.addEventListener("astro:hydrate",this.hydrate,{once:!0});return}let n=this.querySelectorAll("astro-slot"),r={},i=this.querySelectorAll("template[data-astro-template]");for(let o of i){let a=o.closest(this.tagName);a!=null&&a.isSameNode(this)&&(r[o.getAttribute("data-astro-template")||"default"]=o.innerHTML,o.remove())}for(let o of n){let a=o.closest(this.tagName);a!=null&&a.isSameNode(this)&&(r[o.getAttribute("name")||"default"]=o.innerHTML)}let u;try{u=this.hasAttribute("props")?y(JSON.parse(this.getAttribute("props"))):{}}catch(o){let a=this.getAttribute("component-url")||"<unknown>",v=this.getAttribute("component-export");throw v&&(a+=\` (export \${v})\`),console.error(\`[hydrate] Error parsing props for component \${a}\`,this.getAttribute("props"),o),o}let h;await this.hydrator(this)(this.Component,u,r,{client:this.getAttribute("client")}),this.removeAttribute("ssr"),this.dispatchEvent(new CustomEvent("astro:hydrate"))});l(this,"unmount",()=>{this.isConnected||this.dispatchEvent(new CustomEvent("astro:unmount"))})}disconnectedCallback(){document.removeEventListener("astro:after-swap",this.unmount),document.addEventListener("astro:after-swap",this.unmount,{once:!0})}connectedCallback(){if(!this.hasAttribute("await-children")||document.readyState==="interactive"||document.readyState==="complete")this.childrenConnectedCallback();else{let e=()=>{document.removeEventListener("DOMContentLoaded",e),n.disconnect(),this.childrenConnectedCallback()},n=new MutationObserver(()=>{var r;((r=this.lastChild)==null?void 0:r.nodeType)===Node.COMMENT_NODE&&this.lastChild.nodeValue==="astro:end"&&(this.lastChild.remove(),e())});n.observe(this,{childList:!0}),document.addEventListener("DOMContentLoaded",e)}}async childrenConnectedCallback(){let e=this.getAttribute("before-hydration-url");e&&await import(e),this.start()}getRetryImportUrl(e){let n=new URL(e,document.baseURI),r=\`astro-retry=\${Date.now()}\`,i=n.hash.replace(/^#/,"");return n.hash=i?\`\${i}&\${r}\`:r,n.toString()}async importWithRetry(e){try{return await import(e)}catch(n){return await new Promise(r=>setTimeout(r,1e3)),import(this.getRetryImportUrl(e))}}handleHydrationError(e){let n=this.getAttribute("component-url"),r=new CustomEvent("astro:hydration-error",{cancelable:!0,bubbles:!0,composed:!0,detail:{error:e,componentUrl:n}});this.dispatchEvent(r)&&console.error(\`[astro-island] Error hydrating \${n}\`,e)}async start(){let e=JSON.parse(this.getAttribute("opts")),n=this.getAttribute("client");if(Astro[n]===void 0){window.addEventListener(\`astro:\${n}\`,()=>this.start(),{once:!0});return}try{await Astro[n](async()=>{let r=this.getAttribute("renderer-url");try{let[i,{default:u}]=await Promise.all([this.importWithRetry(this.getAttribute("component-url")),r?this.importWithRetry(r):Promise.resolve({default:()=>()=>{}})]),h=this.getAttribute("component-export")||"default";if(h.includes(".")){this.Component=i;for(let m of h.split(".")){if(E.has(m)||!this.Component||typeof this.Component!="object"&&typeof this.Component!="function"||!Object.hasOwn(this.Component,m))throw new Error(\`Invalid component export path: \${h}\`);this.Component=this.Component[m]}}else{if(E.has(h))throw new Error(\`Invalid component export path: \${h}\`);this.Component=i[h]}return this.hydrator=u,this.hydrate}catch(i){return this.handleHydrationError(i),()=>{}}},e,this)}catch(r){this.handleHydrationError(r)}}attributeChangedCallback(){this.hydrate()}}l(f,"observedAttributes",["props"]),customElements.get("astro-island")||customElements.define("astro-island",f)}})();`;
      ISLAND_STYLES = "astro-island,astro-slot,astro-static-slot{display:contents}";
      RenderInstructionSymbol = /* @__PURE__ */ Symbol.for("astro:render");
      voidElementNames = /^(area|base|br|col|command|embed|hr|img|input|keygen|link|meta|param|source|track|wbr)$/i;
      htmlBooleanAttributes = /^(?:allowfullscreen|async|autofocus|autoplay|checked|controls|default|defer|disabled|disablepictureinpicture|disableremoteplayback|formnovalidate|inert|loop|muted|nomodule|novalidate|open|playsinline|readonly|required|reversed|scoped|seamless|selected|itemscope)$/i;
      AMPERSAND_REGEX = /&/g;
      DOUBLE_QUOTE_REGEX = /"/g;
      STATIC_DIRECTIVES = /* @__PURE__ */ new Set(["set:html", "set:text"]);
      INVALID_ATTR_NAME_CHAR = /[\s"'>/=]/;
      toIdent = (k) => k.trim().replace(/(?!^)\b\w|\s+|\W+/g, (match, index) => {
        if (/\W/.test(match)) return "";
        return index === 0 ? match : match.toUpperCase();
      });
      toAttributeString = (value, shouldEscape = true) => shouldEscape ? String(value).replace(AMPERSAND_REGEX, "&amp;").replace(DOUBLE_QUOTE_REGEX, "&quot;") : value;
      kebab = (k) => k.toLowerCase() === k ? k : k.replace(/[A-Z]/g, (match) => `-${match.toLowerCase()}`);
      toStyleString = (obj) => Object.entries(obj).filter(([_, v]) => typeof v === "string" && v.trim() || typeof v === "number").map(([k, v]) => {
        if (k[0] !== "-" && k[1] !== "-") return `${kebab(k)}:${v}`;
        return `${k}:${v}`;
      }).join(";");
      noop = () => {
      };
      BufferedRenderer = class {
        constructor(destination, renderFunction) {
          __publicField(this, "chunks", []);
          __publicField(this, "renderPromise");
          __publicField(this, "destination");
          /**
           * Determines whether buffer has been flushed
           * to the final destination.
           */
          __publicField(this, "flushed", false);
          this.destination = destination;
          this.renderPromise = renderFunction(this);
          if (isPromise(this.renderPromise)) {
            Promise.resolve(this.renderPromise).catch(noop);
          }
        }
        write(chunk) {
          if (this.flushed) {
            this.destination.write(chunk);
          } else {
            this.chunks.push(chunk);
          }
        }
        flush() {
          if (this.flushed) {
            throw new Error("The render buffer has already been flushed.");
          }
          this.flushed = true;
          for (const chunk of this.chunks) {
            this.destination.write(chunk);
          }
          return this.renderPromise;
        }
      };
      isNode = typeof process !== "undefined" && Object.prototype.toString.call(process) === "[object process]";
      isDeno = typeof Deno !== "undefined";
      alphabetUpperCase = "0123456789ABCDEF";
      decodeMap = {
        "0": 0,
        "1": 1,
        "2": 2,
        "3": 3,
        "4": 4,
        "5": 5,
        "6": 6,
        "7": 7,
        "8": 8,
        "9": 9,
        a: 10,
        A: 10,
        b: 11,
        B: 11,
        c: 12,
        C: 12,
        d: 13,
        D: 13,
        e: 14,
        E: 14,
        f: 15,
        F: 15
      };
      (function(EncodingPadding2) {
        EncodingPadding2[EncodingPadding2["Include"] = 0] = "Include";
        EncodingPadding2[EncodingPadding2["None"] = 1] = "None";
      })(EncodingPadding$1 || (EncodingPadding$1 = {}));
      (function(DecodingPadding2) {
        DecodingPadding2[DecodingPadding2["Required"] = 0] = "Required";
        DecodingPadding2[DecodingPadding2["Ignore"] = 1] = "Ignore";
      })(DecodingPadding$1 || (DecodingPadding$1 = {}));
      base64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
      (function(EncodingPadding2) {
        EncodingPadding2[EncodingPadding2["Include"] = 0] = "Include";
        EncodingPadding2[EncodingPadding2["None"] = 1] = "None";
      })(EncodingPadding || (EncodingPadding = {}));
      (function(DecodingPadding2) {
        DecodingPadding2[DecodingPadding2["Required"] = 0] = "Required";
        DecodingPadding2[DecodingPadding2["Ignore"] = 1] = "Ignore";
      })(DecodingPadding || (DecodingPadding = {}));
      base64DecodeMap = {
        "0": 52,
        "1": 53,
        "2": 54,
        "3": 55,
        "4": 56,
        "5": 57,
        "6": 58,
        "7": 59,
        "8": 60,
        "9": 61,
        A: 0,
        B: 1,
        C: 2,
        D: 3,
        E: 4,
        F: 5,
        G: 6,
        H: 7,
        I: 8,
        J: 9,
        K: 10,
        L: 11,
        M: 12,
        N: 13,
        O: 14,
        P: 15,
        Q: 16,
        R: 17,
        S: 18,
        T: 19,
        U: 20,
        V: 21,
        W: 22,
        X: 23,
        Y: 24,
        Z: 25,
        a: 26,
        b: 27,
        c: 28,
        d: 29,
        e: 30,
        f: 31,
        g: 32,
        h: 33,
        i: 34,
        j: 35,
        k: 36,
        l: 37,
        m: 38,
        n: 39,
        o: 40,
        p: 41,
        q: 42,
        r: 43,
        s: 44,
        t: 45,
        u: 46,
        v: 47,
        w: 48,
        x: 49,
        y: 50,
        z: 51,
        "+": 62,
        "/": 63
      };
      initializer = (inst, issues) => {
        $ZodError.init(inst, issues);
        inst.name = "ZodError";
        Object.defineProperties(inst, {
          format: {
            value: (mapper) => formatError(inst, mapper)
            // enumerable: false,
          },
          flatten: {
            value: (mapper) => flattenError(inst, mapper)
            // enumerable: false,
          },
          addIssue: {
            value: (issue2) => {
              inst.issues.push(issue2);
              inst.message = JSON.stringify(inst.issues, jsonStringifyReplacer, 2);
            }
            // enumerable: false,
          },
          addIssues: {
            value: (issues2) => {
              inst.issues.push(...issues2);
              inst.message = JSON.stringify(inst.issues, jsonStringifyReplacer, 2);
            }
            // enumerable: false,
          },
          isEmpty: {
            get() {
              return inst.issues.length === 0;
            }
            // enumerable: false,
          }
        });
      };
      ZodRealError = /* @__PURE__ */ $constructor("ZodError", initializer, {
        Parent: Error
      });
      parse = /* @__PURE__ */ _parse(ZodRealError);
      parseAsync = /* @__PURE__ */ _parseAsync(ZodRealError);
      safeParse = /* @__PURE__ */ _safeParse(ZodRealError);
      safeParseAsync = /* @__PURE__ */ _safeParseAsync(ZodRealError);
      encode = /* @__PURE__ */ _encode(ZodRealError);
      decode = /* @__PURE__ */ _decode(ZodRealError);
      encodeAsync = /* @__PURE__ */ _encodeAsync(ZodRealError);
      decodeAsync = /* @__PURE__ */ _decodeAsync(ZodRealError);
      safeEncode = /* @__PURE__ */ _safeEncode(ZodRealError);
      safeDecode = /* @__PURE__ */ _safeDecode(ZodRealError);
      safeEncodeAsync = /* @__PURE__ */ _safeEncodeAsync(ZodRealError);
      safeDecodeAsync = /* @__PURE__ */ _safeDecodeAsync(ZodRealError);
      _installedGroups = /* @__PURE__ */ new WeakMap();
      ZodType = /* @__PURE__ */ $constructor("ZodType", (inst, def) => {
        $ZodType.init(inst, def);
        Object.assign(inst["~standard"], {
          jsonSchema: {
            input: createStandardJSONSchemaMethod(inst, "input"),
            output: createStandardJSONSchemaMethod(inst, "output")
          }
        });
        inst.toJSONSchema = createToJSONSchemaMethod(inst, {});
        inst.def = def;
        inst.type = def.type;
        Object.defineProperty(inst, "_def", { value: def });
        inst.parse = (data, params) => parse(inst, data, params, { callee: inst.parse });
        inst.safeParse = (data, params) => safeParse(inst, data, params);
        inst.parseAsync = (data, params) => __async(null, null, function* () {
          return parseAsync(inst, data, params, { callee: inst.parseAsync });
        });
        inst.safeParseAsync = (data, params) => __async(null, null, function* () {
          return safeParseAsync(inst, data, params);
        });
        inst.spa = inst.safeParseAsync;
        inst.encode = (data, params) => encode(inst, data, params);
        inst.decode = (data, params) => decode(inst, data, params);
        inst.encodeAsync = (data, params) => __async(null, null, function* () {
          return encodeAsync(inst, data, params);
        });
        inst.decodeAsync = (data, params) => __async(null, null, function* () {
          return decodeAsync(inst, data, params);
        });
        inst.safeEncode = (data, params) => safeEncode(inst, data, params);
        inst.safeDecode = (data, params) => safeDecode(inst, data, params);
        inst.safeEncodeAsync = (data, params) => __async(null, null, function* () {
          return safeEncodeAsync(inst, data, params);
        });
        inst.safeDecodeAsync = (data, params) => __async(null, null, function* () {
          return safeDecodeAsync(inst, data, params);
        });
        _installLazyMethods(inst, "ZodType", {
          check(...chks) {
            var _a29;
            const def2 = this.def;
            return this.clone(mergeDefs(def2, {
              checks: [
                ...(_a29 = def2.checks) != null ? _a29 : [],
                ...chks.map((ch) => typeof ch === "function" ? { _zod: { check: ch, def: { check: "custom" }, onattach: [] } } : ch)
              ]
            }), { parent: true });
          },
          with(...chks) {
            return this.check(...chks);
          },
          clone(def2, params) {
            return clone(this, def2, params);
          },
          brand() {
            return this;
          },
          register(reg, meta) {
            reg.add(this, meta);
            return this;
          },
          refine(check, params) {
            return this.check(refine(check, params));
          },
          superRefine(refinement, params) {
            return this.check(superRefine(refinement, params));
          },
          overwrite(fn) {
            return this.check(/* @__PURE__ */ _overwrite(fn));
          },
          optional() {
            return optional(this);
          },
          exactOptional() {
            return exactOptional(this);
          },
          nullable() {
            return nullable(this);
          },
          nullish() {
            return optional(nullable(this));
          },
          nonoptional(params) {
            return nonoptional(this, params);
          },
          array() {
            return array(this);
          },
          or(arg) {
            return union([this, arg]);
          },
          and(arg) {
            return intersection(this, arg);
          },
          transform(tx) {
            return pipe(this, transform2(tx));
          },
          default(d) {
            return _default2(this, d);
          },
          prefault(d) {
            return prefault(this, d);
          },
          catch(params) {
            return _catch(this, params);
          },
          pipe(target) {
            return pipe(this, target);
          },
          readonly() {
            return readonly(this);
          },
          describe(description) {
            const cl = this.clone();
            globalRegistry.add(cl, { description });
            return cl;
          },
          meta(...args) {
            if (args.length === 0)
              return globalRegistry.get(this);
            const cl = this.clone();
            globalRegistry.add(cl, args[0]);
            return cl;
          },
          isOptional() {
            return this.safeParse(void 0).success;
          },
          isNullable() {
            return this.safeParse(null).success;
          },
          apply(fn) {
            return fn(this);
          }
        });
        Object.defineProperty(inst, "description", {
          get() {
            var _a29;
            return (_a29 = globalRegistry.get(inst)) == null ? void 0 : _a29.description;
          },
          configurable: true
        });
        return inst;
      });
      ZodArray = /* @__PURE__ */ $constructor("ZodArray", (inst, def) => {
        $ZodArray.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => arrayProcessor(inst, ctx, json, params);
        inst.element = def.element;
        _installLazyMethods(inst, "ZodArray", {
          min(n2, params) {
            return this.check(/* @__PURE__ */ _minLength(n2, params));
          },
          nonempty(params) {
            return this.check(/* @__PURE__ */ _minLength(1, params));
          },
          max(n2, params) {
            return this.check(/* @__PURE__ */ _maxLength(n2, params));
          },
          length(n2, params) {
            return this.check(/* @__PURE__ */ _length(n2, params));
          },
          unwrap() {
            return this.element;
          }
        });
      });
      ZodUnion = /* @__PURE__ */ $constructor("ZodUnion", (inst, def) => {
        $ZodUnion.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => unionProcessor(inst, ctx, json, params);
        inst.options = def.options;
      });
      ZodIntersection = /* @__PURE__ */ $constructor("ZodIntersection", (inst, def) => {
        $ZodIntersection.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => intersectionProcessor(inst, ctx, json, params);
      });
      ZodEnum = /* @__PURE__ */ $constructor("ZodEnum", (inst, def) => {
        $ZodEnum.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => enumProcessor(inst, ctx, json);
        inst.enum = def.entries;
        inst.options = Object.values(def.entries);
        const keys = new Set(Object.keys(def.entries));
        inst.extract = (values, params) => {
          const newEntries = {};
          for (const value of values) {
            if (keys.has(value)) {
              newEntries[value] = def.entries[value];
            } else
              throw new Error(`Key ${value} not found in enum`);
          }
          return new ZodEnum(__spreadProps(__spreadValues(__spreadProps(__spreadValues({}, def), {
            checks: []
          }), normalizeParams(params)), {
            entries: newEntries
          }));
        };
        inst.exclude = (values, params) => {
          const newEntries = __spreadValues({}, def.entries);
          for (const value of values) {
            if (keys.has(value)) {
              delete newEntries[value];
            } else
              throw new Error(`Key ${value} not found in enum`);
          }
          return new ZodEnum(__spreadProps(__spreadValues(__spreadProps(__spreadValues({}, def), {
            checks: []
          }), normalizeParams(params)), {
            entries: newEntries
          }));
        };
      });
      ZodTransform = /* @__PURE__ */ $constructor("ZodTransform", (inst, def) => {
        $ZodTransform.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => transformProcessor(inst, ctx);
        inst._zod.parse = (payload, _ctx) => {
          if (_ctx.direction === "backward") {
            throw new $ZodEncodeError(inst.constructor.name);
          }
          payload.addIssue = (issue$1) => {
            var _a29, _b13, _c;
            if (typeof issue$1 === "string") {
              payload.issues.push(issue(issue$1, payload.value, def));
            } else {
              const _issue = issue$1;
              if (_issue.fatal)
                _issue.continue = false;
              (_a29 = _issue.code) != null ? _a29 : _issue.code = "custom";
              (_b13 = _issue.input) != null ? _b13 : _issue.input = payload.value;
              (_c = _issue.inst) != null ? _c : _issue.inst = inst;
              payload.issues.push(issue(_issue));
            }
          };
          const output = def.transform(payload.value, payload);
          if (output instanceof Promise) {
            return output.then((output2) => {
              payload.value = output2;
              payload.fallback = true;
              return payload;
            });
          }
          payload.value = output;
          payload.fallback = true;
          return payload;
        };
      });
      ZodOptional = /* @__PURE__ */ $constructor("ZodOptional", (inst, def) => {
        $ZodOptional.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => optionalProcessor(inst, ctx, json, params);
        inst.unwrap = () => inst._zod.def.innerType;
      });
      ZodExactOptional = /* @__PURE__ */ $constructor("ZodExactOptional", (inst, def) => {
        $ZodExactOptional.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => optionalProcessor(inst, ctx, json, params);
        inst.unwrap = () => inst._zod.def.innerType;
      });
      ZodNullable = /* @__PURE__ */ $constructor("ZodNullable", (inst, def) => {
        $ZodNullable.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => nullableProcessor(inst, ctx, json, params);
        inst.unwrap = () => inst._zod.def.innerType;
      });
      ZodDefault = /* @__PURE__ */ $constructor("ZodDefault", (inst, def) => {
        $ZodDefault.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => defaultProcessor(inst, ctx, json, params);
        inst.unwrap = () => inst._zod.def.innerType;
        inst.removeDefault = inst.unwrap;
      });
      ZodPrefault = /* @__PURE__ */ $constructor("ZodPrefault", (inst, def) => {
        $ZodPrefault.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => prefaultProcessor(inst, ctx, json, params);
        inst.unwrap = () => inst._zod.def.innerType;
      });
      ZodNonOptional = /* @__PURE__ */ $constructor("ZodNonOptional", (inst, def) => {
        $ZodNonOptional.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => nonoptionalProcessor(inst, ctx, json, params);
        inst.unwrap = () => inst._zod.def.innerType;
      });
      ZodCatch = /* @__PURE__ */ $constructor("ZodCatch", (inst, def) => {
        $ZodCatch.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => catchProcessor(inst, ctx, json, params);
        inst.unwrap = () => inst._zod.def.innerType;
        inst.removeCatch = inst.unwrap;
      });
      ZodPipe = /* @__PURE__ */ $constructor("ZodPipe", (inst, def) => {
        $ZodPipe.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => pipeProcessor(inst, ctx, json, params);
        inst.in = def.in;
        inst.out = def.out;
      });
      ZodReadonly = /* @__PURE__ */ $constructor("ZodReadonly", (inst, def) => {
        $ZodReadonly.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => readonlyProcessor(inst, ctx, json, params);
        inst.unwrap = () => inst._zod.def.innerType;
      });
      ZodCustom = /* @__PURE__ */ $constructor("ZodCustom", (inst, def) => {
        $ZodCustom.init(inst, def);
        ZodType.init(inst, def);
        inst._zod.processJSONSchema = (ctx, json, params) => customProcessor(inst, ctx);
      });
      ZodIssueCode = {
        custom: "custom"
      };
      ALGORITHMS = {
        "SHA-256": "sha256-",
        "SHA-384": "sha384-",
        "SHA-512": "sha512-"
      };
      _enum(Object.keys(ALGORITHMS)).optional().default("SHA-256");
      ALLOWED_DIRECTIVES = [
        "base-uri",
        "child-src",
        "connect-src",
        "default-src",
        "fenced-frame-src",
        "font-src",
        "form-action",
        "frame-ancestors",
        "frame-src",
        "img-src",
        "manifest-src",
        "media-src",
        "object-src",
        "referrer",
        "report-to",
        "report-uri",
        "require-trusted-types-for",
        "sandbox",
        "trusted-types",
        "upgrade-insecure-requests",
        "worker-src"
      ];
      custom((v) => typeof v === "string").superRefine((value, ctx) => {
        const isAllowed = ALLOWED_DIRECTIVES.some((allowedValue) => {
          return value.startsWith(allowedValue);
        });
        if (!isAllowed) {
          if (value.startsWith("script-src") || value.startsWith("style-src")) {
            ctx.addIssue({
              code: ZodIssueCode.custom,
              message: `Directives \`script-src\` and \`style-src\` are not allowed in \`security.csp.directives\`. Please use \`security.csp.scriptDirective\` and \`security.csp.styleDirective\` instead.`,
              fatal: true
            });
          } else {
            ctx.addIssue({
              code: ZodIssueCode.custom,
              message: `Invalid directive: "${value}". Allowed directives are: ${ALLOWED_DIRECTIVES.join(", ")}`,
              fatal: true
            });
          }
        }
      });
      ALGORITHM = "AES-GCM";
      encoder$1 = new TextEncoder();
      decoder$1 = new TextDecoder();
      IV_LENGTH = 24;
      renderTemplateResultSym = /* @__PURE__ */ Symbol.for("astro.renderTemplateResult");
      _a24 = renderTemplateResultSym;
      RenderTemplateResult = class {
        constructor(htmlParts, expressions) {
          __publicField(this, _a24, true);
          __publicField(this, "htmlParts");
          __publicField(this, "expressions");
          __publicField(this, "error");
          this.htmlParts = htmlParts;
          this.error = void 0;
          this.expressions = expressions.map((expression) => {
            if (isPromise(expression)) {
              return Promise.resolve(expression).catch((err) => {
                if (!this.error) {
                  this.error = err;
                  throw err;
                }
              });
            }
            return expression;
          });
        }
        render(destination) {
          const { htmlParts, expressions } = this;
          for (let i2 = 0; i2 < htmlParts.length; i2++) {
            const html = htmlParts[i2];
            if (html) {
              destination.write(markHTMLString(html));
            }
            if (i2 >= expressions.length) break;
            const exp = expressions[i2];
            if (!(exp || exp === 0)) continue;
            const result = renderChild(destination, exp);
            if (isPromise(result)) {
              const startIdx = i2 + 1;
              const remaining = expressions.length - startIdx;
              const flushers = new Array(remaining);
              for (let j = 0; j < remaining; j++) {
                const rExp = expressions[startIdx + j];
                flushers[j] = createBufferedRenderer(destination, (bufferDestination) => {
                  if (rExp || rExp === 0) {
                    return renderChild(bufferDestination, rExp);
                  }
                });
              }
              return result.then(() => {
                let k = 0;
                const iterate = () => {
                  while (k < flushers.length) {
                    const rHtml = htmlParts[startIdx + k];
                    if (rHtml) {
                      destination.write(markHTMLString(rHtml));
                    }
                    const flushResult = flushers[k++].flush();
                    if (isPromise(flushResult)) {
                      return flushResult.then(iterate);
                    }
                  }
                  const lastHtml = htmlParts[htmlParts.length - 1];
                  if (lastHtml) {
                    destination.write(markHTMLString(lastHtml));
                  }
                };
                return iterate();
              });
            }
          }
        }
      };
      slotString = /* @__PURE__ */ Symbol.for("astro:slot-string");
      SlotString = class extends (_b11 = HTMLString, _a25 = slotString, _b11) {
        constructor(content, instructions) {
          super(content);
          __publicField(this, "instructions");
          __publicField(this, _a25);
          this.instructions = instructions;
          this[slotString] = true;
        }
      };
      internalProps = /* @__PURE__ */ new Set([
        "server:component-path",
        "server:component-export",
        "server:component-directive",
        "server:defer"
      ]);
      ServerIslandComponent = class {
        constructor(result, props, slots, displayName) {
          __publicField(this, "result");
          __publicField(this, "props");
          __publicField(this, "slots");
          __publicField(this, "displayName");
          __publicField(this, "hostId");
          __publicField(this, "islandContent");
          __publicField(this, "componentPath");
          __publicField(this, "componentExport");
          __publicField(this, "componentId");
          this.result = result;
          this.props = props;
          this.slots = slots;
          this.displayName = displayName;
        }
        init() {
          return __async(this, null, function* () {
            const content = yield this.getIslandContent();
            if (this.result.cspDestination) {
              this.result._metadata.extraScriptHashes.push(
                yield generateCspDigest(SERVER_ISLAND_REPLACER, this.result.cspAlgorithm)
              );
              const contentDigest = yield generateCspDigest(content, this.result.cspAlgorithm);
              this.result._metadata.extraScriptHashes.push(contentDigest);
            }
            return createThinHead();
          });
        }
        render(destination) {
          return __async(this, null, function* () {
            const hostId = yield this.getHostId();
            const islandContent = yield this.getIslandContent();
            destination.write(createRenderInstruction({ type: "server-island-runtime" }));
            destination.write("<!--[if astro]>server-island-start<![endif]-->");
            for (const name in this.slots) {
              if (name === "fallback") {
                yield renderChild(destination, this.slots.fallback(this.result));
              }
            }
            destination.write(
              `<script type="module" data-astro-rerun data-island-id="${hostId}">${islandContent}</script>`
            );
          });
        }
        getComponentPath() {
          if (this.componentPath) {
            return this.componentPath;
          }
          const componentPath = this.props["server:component-path"];
          if (!componentPath) {
            throw new Error(`Could not find server component path`);
          }
          this.componentPath = componentPath;
          return componentPath;
        }
        getComponentExport() {
          if (this.componentExport) {
            return this.componentExport;
          }
          const componentExport = this.props["server:component-export"];
          if (!componentExport) {
            throw new Error(`Could not find server component export`);
          }
          this.componentExport = componentExport;
          return componentExport;
        }
        getHostId() {
          return __async(this, null, function* () {
            if (!this.hostId) {
              this.hostId = yield crypto.randomUUID();
            }
            return this.hostId;
          });
        }
        getIslandContent() {
          return __async(this, null, function* () {
            if (this.islandContent) {
              return this.islandContent;
            }
            const componentPath = this.getComponentPath();
            const componentExport = this.getComponentExport();
            const serverIslandNameMap2 = yield this.result.getServerIslandNameMap();
            let componentId = serverIslandNameMap2.get(componentPath);
            if (!componentId) {
              throw new Error(`Could not find server component name ${componentPath}`);
            }
            for (const key2 of Object.keys(this.props)) {
              if (internalProps.has(key2)) {
                delete this.props[key2];
              }
            }
            const renderedSlots = {};
            for (const name in this.slots) {
              if (name !== "fallback") {
                const content = yield renderSlotToString(this.result, this.slots[name]);
                let slotHtml = content.toString();
                const slotContent = content;
                if (Array.isArray(slotContent.instructions)) {
                  for (const instruction of slotContent.instructions) {
                    if (instruction.type === "script") {
                      slotHtml += instruction.content;
                    }
                  }
                }
                renderedSlots[name] = slotHtml;
              }
            }
            const key = yield this.result.key;
            const componentExportEncrypted = yield encryptString(
              key,
              componentExport,
              `export:${componentId}`
            );
            const propsEncrypted = Object.keys(this.props).length === 0 ? "" : yield encryptString(key, JSON.stringify(this.props), `props:${componentId}`);
            const slotsEncrypted = Object.keys(renderedSlots).length === 0 ? "" : yield encryptString(key, JSON.stringify(renderedSlots), `slots:${componentId}`);
            const hostId = yield this.getHostId();
            const slash2 = this.result.base.endsWith("/") ? "" : "/";
            let serverIslandUrl = `${this.result.base}${slash2}_server-islands/${componentId}${this.result.trailingSlash === "always" ? "/" : ""}`;
            const potentialSearchParams = createSearchParams(
              componentExportEncrypted,
              propsEncrypted,
              slotsEncrypted
            );
            const useGETRequest = isWithinURLLimit(serverIslandUrl, potentialSearchParams);
            if (useGETRequest) {
              serverIslandUrl += "?" + potentialSearchParams.toString();
              this.result._metadata.extraHead.push(
                markHTMLString(
                  `<link rel="preload" as="fetch" href="${serverIslandUrl}" crossorigin="anonymous">`
                )
              );
            }
            const adapterHeaders = this.result.internalFetchHeaders || {};
            const headersJson = stringifyForScript(adapterHeaders);
            const method = useGETRequest ? (
              // GET request
              `const headers = new Headers(${headersJson});
let response = await fetch('${serverIslandUrl}', { headers });`
            ) : (
              // POST request
              `let data = {
	encryptedComponentExport: ${stringifyForScript(componentExportEncrypted)},
	encryptedProps: ${stringifyForScript(propsEncrypted)},
	encryptedSlots: ${stringifyForScript(slotsEncrypted)},
};
const headers = new Headers({ 'Content-Type': 'application/json', ...${headersJson} });
let response = await fetch('${serverIslandUrl}', {
	method: 'POST',
	body: JSON.stringify(data),
	headers,
});`
            );
            this.islandContent = `${method}replaceServerIsland('${hostId}', response);`;
            return this.islandContent;
          });
        }
      };
      renderServerIslandRuntime = () => {
        return `<script>${SERVER_ISLAND_REPLACER}</script>`;
      };
      SERVER_ISLAND_REPLACER = markHTMLString(
        `async function replaceServerIsland(id, r) {
	let s = document.querySelector(\`script[data-island-id="\${id}"]\`);
	// If there's no matching script, or the request fails then return
	if (!s || r.status !== 200 || r.headers.get('content-type')?.split(';')[0].trim() !== 'text/html') return;
	// Load the HTML before modifying the DOM in case of errors
	let html = await r.text();
	// Remove any placeholder content before the island script
	while (s.previousSibling && s.previousSibling.nodeType !== 8 && s.previousSibling.data !== '[if astro]>server-island-start<![endif]')
		s.previousSibling.remove();
	s.previousSibling?.remove();
	// Insert the new HTML
	s.before(document.createRange().createContextualFragment(html));
	// Remove the script. Prior to v5.4.2, this was the trick to force rerun of scripts.  Keeping it to minimize change to the existing behavior.
	s.remove();
}`.split("\n").map((line) => line.trim()).filter((line) => line && !line.startsWith("//")).join(" ")
      );
      Fragment = /* @__PURE__ */ Symbol.for("astro:fragment");
      Renderer = /* @__PURE__ */ Symbol.for("astro:renderer");
      encoder = new TextEncoder();
      decoder2 = new TextDecoder();
      astroComponentInstanceSym = /* @__PURE__ */ Symbol.for("astro.componentInstance");
      _a26 = astroComponentInstanceSym;
      AstroComponentInstance = class {
        constructor(result, props, slots, factory) {
          __publicField(this, _a26, true);
          __publicField(this, "result");
          __publicField(this, "props");
          __publicField(this, "slotValues");
          __publicField(this, "factory");
          __publicField(this, "returnValue");
          this.result = result;
          this.props = props;
          this.factory = factory;
          this.slotValues = {};
          for (const name in slots) {
            let didRender = false;
            let value = slots[name](result);
            this.slotValues[name] = () => {
              if (!didRender) {
                didRender = true;
                return value;
              }
              return slots[name](result);
            };
          }
        }
        init(result) {
          if (this.returnValue !== void 0) {
            return this.returnValue;
          }
          this.returnValue = this.factory(result, this.props, this.slotValues);
          if (isPromise(this.returnValue)) {
            this.returnValue.then((resolved) => {
              this.returnValue = resolved;
            }).catch(() => {
            });
          }
          return this.returnValue;
        }
        render(destination) {
          const returnValue = this.init(this.result);
          if (isPromise(returnValue)) {
            return returnValue.then((x) => this.renderImpl(destination, x));
          }
          return this.renderImpl(destination, returnValue);
        }
        renderImpl(destination, returnValue) {
          if (isHeadAndContent(returnValue)) {
            return returnValue.content.render(destination);
          } else {
            return renderChild(destination, returnValue);
          }
        }
      };
      DOCTYPE_EXP = /<!doctype html/i;
      needsHeadRenderingSymbol = /* @__PURE__ */ Symbol.for("astro.needsHeadRendering");
      rendererAliases = /* @__PURE__ */ new Map([["solid", "solid-js"]]);
      clientOnlyValues = /* @__PURE__ */ new Set(["solid-js", "react", "preact", "vue", "svelte"]);
      ASTRO_SLOT_EXP = /<\/?astro-slot\b[^>]*>/g;
      ASTRO_STATIC_SLOT_EXP = /<\/?astro-static-slot\b[^>]*>/g;
      ClientOnlyPlaceholder$1 = "astro-client-only";
      hasTriedRenderComponentSymbol = /* @__PURE__ */ Symbol("hasTriedRenderComponent");
      ClientOnlyPlaceholder = "astro-client-only";
      "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_".split("").reduce((v, c) => (v[c.charCodeAt(0)] = c, v), []);
      "-0123456789_".split("").reduce((v, c) => (v[c.charCodeAt(0)] = c, v), []);
      SERVER_ISLAND_ROUTE = "/_server-islands/[name]";
      SERVER_ISLAND_COMPONENT = "_server-islands.astro";
      DEFAULT_BODY_SIZE_LIMIT = 1024 * 1024;
      Router = class {
        constructor(routes, options) {
          __privateAdd(this, _routes);
          __privateAdd(this, _base);
          __privateAdd(this, _baseWithoutTrailingSlash);
          __privateAdd(this, _buildFormat);
          __privateAdd(this, _trailingSlash);
          __privateSet(this, _routes, [...routes].sort(routeComparator));
          __privateSet(this, _base, normalizeBase(options.base));
          __privateSet(this, _baseWithoutTrailingSlash, removeTrailingForwardSlash(__privateGet(this, _base)));
          __privateSet(this, _buildFormat, options.buildFormat);
          __privateSet(this, _trailingSlash, options.trailingSlash);
        }
        /**
         * Match an input pathname against the route list.
         * If allowWithoutBase is true, a non-base-prefixed path is still considered.
         */
        match(inputPathname, { allowWithoutBase = false } = {}) {
          const normalized = getRedirectForPathname(inputPathname);
          if (normalized.redirect) {
            return { type: "redirect", location: normalized.redirect, status: 301 };
          }
          if (__privateGet(this, _base) !== "/") {
            const baseWithSlash = `${__privateGet(this, _baseWithoutTrailingSlash)}/`;
            if (__privateGet(this, _trailingSlash) === "always" && (normalized.pathname === __privateGet(this, _baseWithoutTrailingSlash) || normalized.pathname === __privateGet(this, _base))) {
              return { type: "redirect", location: baseWithSlash, status: 301 };
            }
            if (__privateGet(this, _trailingSlash) === "never" && normalized.pathname === baseWithSlash) {
              return { type: "redirect", location: __privateGet(this, _baseWithoutTrailingSlash), status: 301 };
            }
          }
          const baseResult = stripBase(
            normalized.pathname,
            __privateGet(this, _base),
            __privateGet(this, _baseWithoutTrailingSlash),
            __privateGet(this, _trailingSlash)
          );
          if (!baseResult) {
            if (!allowWithoutBase) {
              return { type: "none", reason: "outside-base" };
            }
          }
          let pathname = baseResult != null ? baseResult : normalized.pathname;
          if (__privateGet(this, _buildFormat) === "file") {
            pathname = normalizeFileFormatPathname(pathname);
          }
          const route = __privateGet(this, _routes).find((candidate) => {
            if (candidate.pattern.test(pathname)) return true;
            return candidate.fallbackRoutes.some((fallbackRoute) => fallbackRoute.pattern.test(pathname));
          });
          if (!route) {
            return { type: "none", reason: "no-match" };
          }
          const params = getParams(route, pathname);
          return { type: "match", route, params, pathname };
        }
        /**
         * Returns all routes that match the given pathname, in priority order.
         * Used when the first match (e.g. a prerendered route) cannot serve
         * the request and subsequent matches need to be tried.
         */
        matchAll(inputPathname, { allowWithoutBase = false } = {}) {
          const normalized = getRedirectForPathname(inputPathname);
          if (normalized.redirect) {
            return [];
          }
          const baseResult = stripBase(
            normalized.pathname,
            __privateGet(this, _base),
            __privateGet(this, _baseWithoutTrailingSlash),
            __privateGet(this, _trailingSlash)
          );
          if (!baseResult && !allowWithoutBase) {
            return [];
          }
          let pathname = baseResult != null ? baseResult : normalized.pathname;
          if (__privateGet(this, _buildFormat) === "file") {
            pathname = normalizeFileFormatPathname(pathname);
          }
          return __privateGet(this, _routes).filter((candidate) => {
            if (candidate.pattern.test(pathname)) return true;
            return candidate.fallbackRoutes.some((fallbackRoute) => fallbackRoute.pattern.test(pathname));
          });
        }
      };
      _routes = new WeakMap();
      _base = new WeakMap();
      _baseWithoutTrailingSlash = new WeakMap();
      _buildFormat = new WeakMap();
      _trailingSlash = new WeakMap();
      NodePool = class {
        /**
         * Creates a new object pool for queue nodes.
         *
         * @param maxSize - Maximum number of nodes to keep in the pool (default: 1000).
         *   The cap is shared across all typed sub-pools.
         * @param enableStats - Enable statistics tracking (default: false for performance)
         */
        constructor(maxSize = 1e3, enableStats = false) {
          __publicField(this, "textPool", []);
          __publicField(this, "htmlStringPool", []);
          __publicField(this, "componentPool", []);
          __publicField(this, "instructionPool", []);
          __publicField(this, "maxSize");
          __publicField(this, "enableStats");
          __publicField(this, "stats", {
            acquireFromPool: 0,
            acquireNew: 0,
            released: 0,
            releasedDropped: 0
          });
          this.maxSize = maxSize;
          this.enableStats = enableStats;
        }
        /**
         * Acquires a queue node from the pool or creates a new one if the pool is empty.
         * Pops from the type-specific sub-pool to reuse an existing object when available.
         *
         * @param type - The type of queue node to acquire
         * @param content - Optional content to set on the node (for text or html-string types)
         * @returns A queue node ready to be populated with data
         */
        acquire(type, content) {
          const pooledNode = this.popFromTypedPool(type);
          if (pooledNode) {
            if (this.enableStats) {
              this.stats.acquireFromPool = this.stats.acquireFromPool + 1;
            }
            this.resetNodeContent(pooledNode, type, content);
            return pooledNode;
          }
          if (this.enableStats) {
            this.stats.acquireNew = this.stats.acquireNew + 1;
          }
          return this.createNode(type, content);
        }
        /**
         * Creates a new node of the specified type with the given content.
         * Helper method to reduce branching in acquire().
         */
        createNode(type, content = "") {
          switch (type) {
            case "text":
              return { type: "text", content };
            case "html-string":
              return { type: "html-string", html: content };
            case "component":
              return { type: "component", instance: void 0 };
            case "instruction":
              return { type: "instruction", instruction: void 0 };
          }
        }
        /**
         * Pops a node from the type-specific sub-pool.
         * Returns undefined if the sub-pool for the requested type is empty.
         */
        popFromTypedPool(type) {
          switch (type) {
            case "text":
              return this.textPool.pop();
            case "html-string":
              return this.htmlStringPool.pop();
            case "component":
              return this.componentPool.pop();
            case "instruction":
              return this.instructionPool.pop();
          }
        }
        /**
         * Resets the content/value field on a reused pooled node.
         * The type discriminant is already correct since we pop from the matching sub-pool.
         */
        resetNodeContent(node, type, content) {
          switch (type) {
            case "text":
              node.content = content != null ? content : "";
              break;
            case "html-string":
              node.html = content != null ? content : "";
              break;
            case "component":
              node.instance = void 0;
              break;
            case "instruction":
              node.instruction = void 0;
              break;
          }
        }
        /**
         * Returns the total number of nodes across all typed sub-pools.
         */
        totalPoolSize() {
          return this.textPool.length + this.htmlStringPool.length + this.componentPool.length + this.instructionPool.length;
        }
        /**
         * Releases a queue node back to the pool for reuse.
         * If the pool is at max capacity, the node is discarded (will be GC'd).
         *
         * @param node - The node to release back to the pool
         */
        release(node) {
          if (this.totalPoolSize() >= this.maxSize) {
            if (this.enableStats) {
              this.stats.releasedDropped = this.stats.releasedDropped + 1;
            }
            return;
          }
          switch (node.type) {
            case "text":
              node.content = "";
              this.textPool.push(node);
              break;
            case "html-string":
              node.html = "";
              this.htmlStringPool.push(node);
              break;
            case "component":
              node.instance = void 0;
              this.componentPool.push(node);
              break;
            case "instruction":
              node.instruction = void 0;
              this.instructionPool.push(node);
              break;
          }
          if (this.enableStats) {
            this.stats.released = this.stats.released + 1;
          }
        }
        /**
         * Releases all nodes in an array back to the pool.
         * This is a convenience method for releasing multiple nodes at once.
         *
         * @param nodes - Array of nodes to release
         */
        releaseAll(nodes) {
          for (const node of nodes) {
            this.release(node);
          }
        }
        /**
         * Clears all typed sub-pools, discarding all cached nodes.
         * This can be useful if you want to free memory after a large render.
         */
        clear() {
          this.textPool.length = 0;
          this.htmlStringPool.length = 0;
          this.componentPool.length = 0;
          this.instructionPool.length = 0;
        }
        /**
         * Gets the current total number of nodes across all typed sub-pools.
         * Useful for monitoring pool usage and tuning maxSize.
         *
         * @returns Number of nodes currently available in the pool
         */
        size() {
          return this.totalPoolSize();
        }
        /**
         * Gets pool statistics for debugging.
         *
         * @returns Pool usage statistics including computed metrics
         */
        getStats() {
          return __spreadProps(__spreadValues({}, this.stats), {
            poolSize: this.totalPoolSize(),
            maxSize: this.maxSize,
            hitRate: this.stats.acquireFromPool + this.stats.acquireNew > 0 ? this.stats.acquireFromPool / (this.stats.acquireFromPool + this.stats.acquireNew) * 100 : 0
          });
        }
        /**
         * Resets pool statistics.
         */
        resetStats() {
          this.stats = {
            acquireFromPool: 0,
            acquireNew: 0,
            released: 0,
            releasedDropped: 0
          };
        }
      };
      HTMLStringCache = class {
        constructor(maxSize = 1e3) {
          __publicField(this, "cache", /* @__PURE__ */ new Map());
          __publicField(this, "maxSize");
          this.maxSize = maxSize;
          this.warm(COMMON_HTML_PATTERNS);
        }
        /**
         * Get or create an HTMLString for the given content.
         * If cached, the existing object is returned and moved to end (most recently used).
         * If not cached, a new HTMLString is created, cached, and returned.
         *
         * @param content - The HTML string content
         * @returns HTMLString object (cached or newly created)
         */
        getOrCreate(content) {
          const cached = this.cache.get(content);
          if (cached) {
            this.cache.delete(content);
            this.cache.set(content, cached);
            return cached;
          }
          const htmlString = new HTMLString(content);
          this.cache.set(content, htmlString);
          if (this.cache.size > this.maxSize) {
            const firstKey = this.cache.keys().next().value;
            if (firstKey !== void 0) {
              this.cache.delete(firstKey);
            }
          }
          return htmlString;
        }
        /**
         * Get current cache size
         */
        size() {
          return this.cache.size;
        }
        /**
         * Pre-warms the cache with common HTML patterns.
         * This ensures first-render cache hits for frequently used tags.
         *
         * @param patterns - Array of HTML strings to pre-cache
         */
        warm(patterns) {
          for (const pattern of patterns) {
            if (!this.cache.has(pattern)) {
              this.cache.set(pattern, new HTMLString(pattern));
            }
          }
        }
        /**
         * Clear the entire cache
         */
        clear() {
          this.cache.clear();
        }
      };
      COMMON_HTML_PATTERNS = [
        // Structural elements
        "<div>",
        "</div>",
        "<span>",
        "</span>",
        "<p>",
        "</p>",
        "<section>",
        "</section>",
        "<article>",
        "</article>",
        "<header>",
        "</header>",
        "<footer>",
        "</footer>",
        "<nav>",
        "</nav>",
        "<main>",
        "</main>",
        "<aside>",
        "</aside>",
        // List elements
        "<ul>",
        "</ul>",
        "<ol>",
        "</ol>",
        "<li>",
        "</li>",
        // Void/self-closing elements
        "<br>",
        "<hr>",
        "<br/>",
        "<hr/>",
        // Heading elements
        "<h1>",
        "</h1>",
        "<h2>",
        "</h2>",
        "<h3>",
        "</h3>",
        "<h4>",
        "</h4>",
        // Inline elements
        "<a>",
        "</a>",
        "<strong>",
        "</strong>",
        "<em>",
        "</em>",
        "<code>",
        "</code>",
        // Common whitespace
        " ",
        "\n"
      ];
      FORBIDDEN_PATH_KEYS = /* @__PURE__ */ new Set(["__proto__", "constructor", "prototype"]);
      dateTimeFormat = new Intl.DateTimeFormat([], {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false
      });
      levels = {
        debug: 20,
        info: 30,
        warn: 40,
        error: 50,
        silent: 90
      };
      AstroLogger = class {
        constructor(options) {
          __publicField(this, "options");
          this.options = options;
        }
        info(label, message, newLine = true) {
          info(this.options, label, message, newLine);
        }
        warn(label, message, newLine = true) {
          warn(this.options, label, message, newLine);
        }
        error(label, message, newLine = true) {
          error(this.options, label, message, newLine);
        }
        debug(label, ...messages) {
          debug(label, ...messages);
        }
        level() {
          return this.options.level;
        }
        forkIntegrationLogger(label) {
          return new AstroIntegrationLogger(this.options, label);
        }
        setDestination(destination) {
          this.options.destination = destination;
        }
        /**
         * It calls the `close` function of the provided destination, if it exists.
         */
        close() {
          if (this.options.destination.close) {
            this.options.destination.close();
          }
        }
        /**
         * It calls the `flush` function of the provided destination, if it exists.
         */
        flush() {
          if (this.options.destination.flush) {
            this.options.destination.flush();
          }
        }
      };
      AstroIntegrationLogger = class _AstroIntegrationLogger {
        constructor(logging, label) {
          __publicField(this, "options");
          __publicField(this, "label");
          this.options = logging;
          this.label = label;
        }
        /**
         * Creates a new logger instance with a new label, but the same log options.
         */
        fork(label) {
          return new _AstroIntegrationLogger(this.options, label);
        }
        info(message) {
          info(this.options, this.label, message);
        }
        warn(message) {
          warn(this.options, this.label, message);
        }
        error(message) {
          error(this.options, this.label, message);
        }
        debug(message) {
          debug(this.label, message);
        }
        /**
         * It calls the `flush` function of the provided destination, if it exists.
         */
        flush() {
          if (this.options.destination.flush) {
            this.options.destination.flush();
          }
        }
        /**
         * It calls the `close` function of the provided destination, if it exists.
         */
        close() {
          if (this.options.destination.close) {
            this.options.destination.close();
          }
        }
      };
      SGR_REGEX = new RegExp(`${String.fromCharCode(27)}\\[[0-9;]*m`, "g");
      PipelineFeatures = {
        redirects: 1 << 0,
        sessions: 1 << 1,
        actions: 1 << 2,
        middleware: 1 << 3,
        i18n: 1 << 4,
        cache: 1 << 5
      };
      ALL_PIPELINE_FEATURES = PipelineFeatures.redirects | PipelineFeatures.sessions | PipelineFeatures.actions | PipelineFeatures.middleware | PipelineFeatures.i18n | PipelineFeatures.cache;
      Pipeline = class {
        constructor(logger, manifest2, runtimeMode, renderers2, resolve, streaming, adapterName = manifest2.adapterName, clientDirectives = manifest2.clientDirectives, inlinedScripts = manifest2.inlinedScripts, compressHTML = manifest2.compressHTML, i18n = manifest2.i18n, middleware = manifest2.middleware, routeCache = new RouteCache(logger, runtimeMode), site = manifest2.site ? new URL(manifest2.site) : void 0, defaultRoutes = createDefaultRoutes(manifest2), actions = manifest2.actions, sessionDriver = manifest2.sessionDriver, cacheProvider = manifest2.cacheProvider, cacheConfig = manifest2.cacheConfig, serverIslands = manifest2.serverIslandMappings) {
          __publicField(this, "internalMiddleware");
          __publicField(this, "resolvedMiddleware");
          __publicField(this, "resolvedLogger", false);
          __publicField(this, "resolvedActions");
          __publicField(this, "resolvedSessionDriver");
          __publicField(this, "resolvedCacheProvider");
          __publicField(this, "compiledCacheRoutes");
          __publicField(this, "nodePool");
          __publicField(this, "htmlStringCache");
          /**
           * Bit mask of pipeline features activated by handler classes.
           * Each handler sets its bit via `|=`. Only meaningful when a
           * custom `src/app.ts` fetch handler is in use.
           */
          __publicField(this, "usedFeatures", 0);
          __publicField(this, "logger");
          __publicField(this, "manifest");
          /**
           * "development" or "production" only
           */
          __publicField(this, "runtimeMode");
          __publicField(this, "renderers");
          __publicField(this, "resolve");
          __publicField(this, "streaming");
          /**
           * Used to provide better error messages for `Astro.clientAddress`
           */
          __publicField(this, "adapterName");
          __publicField(this, "clientDirectives");
          __publicField(this, "inlinedScripts");
          __publicField(this, "compressHTML");
          __publicField(this, "i18n");
          __publicField(this, "middleware");
          __publicField(this, "routeCache");
          /**
           * Used for `Astro.site`.
           */
          __publicField(this, "site");
          /**
           * Array of built-in, internal, routes.
           * Used to find the route module
           */
          __publicField(this, "defaultRoutes");
          __publicField(this, "actions");
          __publicField(this, "sessionDriver");
          __publicField(this, "cacheProvider");
          __publicField(this, "cacheConfig");
          __publicField(this, "serverIslands");
          /** Route data derived from the manifest, used for route matching. */
          __publicField(this, "manifestData");
          /** Pattern-matching router built from manifestData. */
          __privateAdd(this, _router);
          var _a29, _b13;
          this.logger = logger;
          this.manifest = manifest2;
          this.runtimeMode = runtimeMode;
          this.renderers = renderers2;
          this.resolve = resolve;
          this.streaming = streaming;
          this.adapterName = adapterName;
          this.clientDirectives = clientDirectives;
          this.inlinedScripts = inlinedScripts;
          this.compressHTML = compressHTML;
          this.i18n = i18n;
          this.middleware = middleware;
          this.routeCache = routeCache;
          this.site = site;
          this.defaultRoutes = defaultRoutes;
          this.actions = actions;
          this.sessionDriver = sessionDriver;
          this.cacheProvider = cacheProvider;
          this.cacheConfig = cacheConfig;
          this.serverIslands = serverIslands;
          this.manifestData = { routes: ((_a29 = manifest2.routes) != null ? _a29 : []).map((route) => route.routeData) };
          ensure404Route(this.manifestData);
          __privateSet(this, _router, new Router(this.manifestData.routes, {
            base: manifest2.base,
            trailingSlash: manifest2.trailingSlash,
            buildFormat: manifest2.buildFormat
          }));
          this.internalMiddleware = [];
          if (manifest2.experimentalQueuedRendering.enabled) {
            this.nodePool = this.createNodePool(
              (_b13 = manifest2.experimentalQueuedRendering.poolSize) != null ? _b13 : 1e3,
              false
            );
            if (manifest2.experimentalQueuedRendering.contentCache) {
              this.htmlStringCache = this.createStringCache();
            }
          }
        }
        /**
         * Low-level route matching against the manifest routes. Returns the
         * matched `RouteData` or `undefined`. Does not filter prerendered
         * routes or check public assets — use `BaseApp.match()` for that.
         */
        matchRoute(pathname) {
          const match = __privateGet(this, _router).match(pathname, { allowWithoutBase: true });
          if (match.type !== "match") return void 0;
          return match.route;
        }
        /**
         * Returns all routes matching the given pathname, in priority order.
         * Used when the first match cannot serve the request (e.g. a
         * prerendered dynamic route that doesn't cover this specific path)
         * and the caller needs to try subsequent matches.
         */
        matchAllRoutes(pathname) {
          return __privateGet(this, _router).matchAll(pathname, { allowWithoutBase: true });
        }
        /**
         * Rebuilds the internal router after routes have been added or
         * removed (e.g. by the dev server on HMR).
         */
        rebuildRouter() {
          __privateSet(this, _router, new Router(this.manifestData.routes, {
            base: this.manifest.base,
            trailingSlash: this.manifest.trailingSlash,
            buildFormat: this.manifest.buildFormat
          }));
        }
        /**
         * Resolves the middleware from the manifest, and returns the `onRequest` function. If `onRequest` isn't there,
         * it returns a no-op function
         */
        getMiddleware() {
          return __async(this, null, function* () {
            var _a29;
            if (this.resolvedMiddleware) {
              return this.resolvedMiddleware;
            }
            if (this.middleware) {
              const middlewareInstance = yield this.middleware();
              const onRequest2 = (_a29 = middlewareInstance.onRequest) != null ? _a29 : NOOP_MIDDLEWARE_FN;
              const internalMiddlewares = [onRequest2];
              if (this.manifest.checkOrigin) {
                internalMiddlewares.unshift(createOriginCheckMiddleware());
              }
              this.resolvedMiddleware = sequence(...internalMiddlewares);
              return this.resolvedMiddleware;
            } else {
              this.resolvedMiddleware = NOOP_MIDDLEWARE_FN;
              return this.resolvedMiddleware;
            }
          });
        }
        /**
         * Clears the cached middleware so it is re-resolved on the next request.
         * Called via HMR when middleware files change during development.
         */
        clearMiddleware() {
          this.resolvedMiddleware = void 0;
        }
        /**
         * Resolves the logger destination from the manifest and updates the pipeline logger.
         * If the user configured `experimental.logger`, the bundled logger factory is loaded
         * and replaces the default console destination. This is lazy and only resolves once.
         */
        getLogger() {
          return __async(this, null, function* () {
            if (this.resolvedLogger) {
              return this.logger;
            }
            this.resolvedLogger = true;
            if (this.manifest.experimentalLogger) {
              this.logger = yield loadLogger(this.manifest.experimentalLogger);
            }
            return this.logger;
          });
        }
        getActions() {
          return __async(this, null, function* () {
            if (this.resolvedActions) {
              return this.resolvedActions;
            } else if (this.actions) {
              this.resolvedActions = yield this.actions();
              return this.resolvedActions;
            }
            return NOOP_ACTIONS_MOD;
          });
        }
        getSessionDriver() {
          return __async(this, null, function* () {
            if (this.resolvedSessionDriver !== void 0) {
              return this.resolvedSessionDriver;
            }
            if (this.sessionDriver) {
              const driverModule = yield this.sessionDriver();
              this.resolvedSessionDriver = (driverModule == null ? void 0 : driverModule.default) || null;
              return this.resolvedSessionDriver;
            }
            this.resolvedSessionDriver = null;
            return null;
          });
        }
        getCacheProvider() {
          return __async(this, null, function* () {
            var _a29;
            if (this.resolvedCacheProvider !== void 0) {
              return this.resolvedCacheProvider;
            }
            if (this.cacheProvider) {
              const mod = yield this.cacheProvider();
              const factory = (mod == null ? void 0 : mod.default) || null;
              this.resolvedCacheProvider = factory ? factory((_a29 = this.cacheConfig) == null ? void 0 : _a29.options) : null;
              return this.resolvedCacheProvider;
            }
            this.resolvedCacheProvider = null;
            return null;
          });
        }
        getServerIslands() {
          return __async(this, null, function* () {
            if (this.serverIslands) {
              return this.serverIslands();
            }
            return {
              serverIslandMap: /* @__PURE__ */ new Map(),
              serverIslandNameMap: /* @__PURE__ */ new Map()
            };
          });
        }
        getAction(path) {
          return __async(this, null, function* () {
            const pathKeys = path.split(".").map((key) => decodeURIComponent(key));
            let { server: server2 } = yield this.getActions();
            if (!server2 || !(typeof server2 === "object")) {
              throw new TypeError(
                `Expected \`server\` export in actions file to be an object. Received ${typeof server2}.`
              );
            }
            for (const key of pathKeys) {
              if (FORBIDDEN_PATH_KEYS.has(key)) {
                throw new AstroError(__spreadProps(__spreadValues({}, ActionNotFoundError), {
                  message: ActionNotFoundError.message(pathKeys.join("."))
                }));
              }
              if (!Object.hasOwn(server2, key)) {
                throw new AstroError(__spreadProps(__spreadValues({}, ActionNotFoundError), {
                  message: ActionNotFoundError.message(pathKeys.join("."))
                }));
              }
              server2 = server2[key];
            }
            if (typeof server2 !== "function") {
              throw new TypeError(
                `Expected handler for action ${pathKeys.join(".")} to be a function. Received ${typeof server2}.`
              );
            }
            return server2;
          });
        }
        getModuleForRoute(route) {
          return __async(this, null, function* () {
            for (const defaultRoute of this.defaultRoutes) {
              if (route.component === defaultRoute.component) {
                return {
                  page: () => Promise.resolve(defaultRoute.instance)
                };
              }
            }
            if (route.type === "redirect") {
              return RedirectSinglePageBuiltModule;
            } else {
              if (this.manifest.pageMap) {
                const importComponentInstance = this.manifest.pageMap.get(route.component);
                if (!importComponentInstance) {
                  throw new Error(
                    `Unexpectedly unable to find a component instance for route ${route.route}`
                  );
                }
                return yield importComponentInstance();
              } else if (this.manifest.pageModule) {
                return this.manifest.pageModule;
              }
              throw new Error(
                "Astro couldn't find the correct page to render, probably because it wasn't correctly mapped for SSR usage. This is an internal error, please file an issue."
              );
            }
          });
        }
        createNodePool(poolSize, stats) {
          return new NodePool(poolSize, stats);
        }
        createStringCache() {
          return new HTMLStringCache(1e3);
        }
      };
      _router = new WeakMap();
      Slots = class {
        constructor(result, slots, logger) {
          __privateAdd(this, _result);
          __privateAdd(this, _slots);
          __privateAdd(this, _logger);
          __privateSet(this, _result, result);
          __privateSet(this, _slots, slots);
          __privateSet(this, _logger, logger);
          if (slots) {
            for (const key of Object.keys(slots)) {
              if (this[key] !== void 0) {
                throw new AstroError(__spreadProps(__spreadValues({}, ReservedSlotName), {
                  message: ReservedSlotName.message(key)
                }));
              }
              Object.defineProperty(this, key, {
                get() {
                  return true;
                },
                enumerable: true
              });
            }
          }
        }
        has(name) {
          if (!__privateGet(this, _slots)) return false;
          return Boolean(__privateGet(this, _slots)[name]);
        }
        render(_0) {
          return __async(this, arguments, function* (name, args = []) {
            if (!__privateGet(this, _slots) || !this.has(name)) return;
            const result = __privateGet(this, _result);
            if (!Array.isArray(args)) {
              __privateGet(this, _logger).warn(
                null,
                `Expected second parameter to be an array, received a ${typeof args}. If you're trying to pass an array as a single argument and getting unexpected results, make sure you're passing your array as an item of an array. Ex: Astro.slots.render('default', [["Hello", "World"]])`
              );
            } else if (args.length > 0) {
              const slotValue = __privateGet(this, _slots)[name];
              const component = typeof slotValue === "function" ? yield slotValue(result) : yield slotValue;
              const expression = getFunctionExpression(component);
              if (expression) {
                const slot = () => __async(this, null, function* () {
                  return typeof expression === "function" ? expression(...args) : expression;
                });
                return yield renderSlotToString(result, slot).then((res) => {
                  return res;
                });
              }
              if (typeof component === "function") {
                return yield renderJSX(result, component(...args)).then(
                  (res) => res != null ? String(res) : res
                );
              }
            }
            const content = yield renderSlotToString(result, __privateGet(this, _slots)[name]);
            const outHTML = chunkToString(result, content);
            return outHTML;
          });
        }
      };
      _result = new WeakMap();
      _slots = new WeakMap();
      _logger = new WeakMap();
      I18nRouter = class {
        constructor(options) {
          __privateAdd(this, _strategy);
          __privateAdd(this, _defaultLocale);
          __privateAdd(this, _locales);
          __privateAdd(this, _base2);
          __privateAdd(this, _domains);
          __privateSet(this, _strategy, options.strategy);
          __privateSet(this, _defaultLocale, options.defaultLocale);
          __privateSet(this, _locales, options.locales);
          __privateSet(this, _base2, options.base === "/" ? "/" : removeTrailingForwardSlash(options.base || ""));
          __privateSet(this, _domains, options.domains);
        }
        /**
         * Evaluate routing strategy for a pathname.
         * Returns decision object (not HTTP Response).
         */
        match(pathname, context) {
          if (this.shouldSkipProcessing(pathname, context)) {
            return { type: "continue" };
          }
          switch (__privateGet(this, _strategy)) {
            case "manual":
              return { type: "continue" };
            case "pathname-prefix-always":
              return this.matchPrefixAlways(pathname, context);
            case "domains-prefix-always":
              if (this.localeHasntDomain(context.currentLocale, context.currentDomain)) {
                return { type: "continue" };
              }
              return this.matchPrefixAlways(pathname, context);
            case "pathname-prefix-other-locales":
              return this.matchPrefixOtherLocales(pathname, context);
            case "domains-prefix-other-locales":
              if (this.localeHasntDomain(context.currentLocale, context.currentDomain)) {
                return { type: "continue" };
              }
              return this.matchPrefixOtherLocales(pathname, context);
            case "pathname-prefix-always-no-redirect":
              return this.matchPrefixAlwaysNoRedirect(pathname, context);
            case "domains-prefix-always-no-redirect":
              if (this.localeHasntDomain(context.currentLocale, context.currentDomain)) {
                return { type: "continue" };
              }
              return this.matchPrefixAlwaysNoRedirect(pathname, context);
            default:
              return { type: "continue" };
          }
        }
        /**
         * Check if i18n processing should be skipped for this request
         */
        shouldSkipProcessing(pathname, context) {
          if (pathname.includes("/404") || pathname.includes("/500")) {
            return true;
          }
          if (pathname.includes("/_server-islands/")) {
            return true;
          }
          if (context.isReroute) {
            return true;
          }
          if (context.routeType && context.routeType !== "page" && context.routeType !== "fallback") {
            return true;
          }
          return false;
        }
        /**
         * Strategy: pathname-prefix-always
         * All locales must have a prefix, including the default locale.
         */
        matchPrefixAlways(pathname, _context) {
          const isRoot = pathname === __privateGet(this, _base2) + "/" || pathname === __privateGet(this, _base2);
          if (isRoot) {
            const basePrefix = __privateGet(this, _base2) === "/" ? "" : __privateGet(this, _base2);
            return {
              type: "redirect",
              location: `${basePrefix}/${__privateGet(this, _defaultLocale)}`
            };
          }
          if (!pathHasLocale(pathname, __privateGet(this, _locales))) {
            return { type: "notFound" };
          }
          return { type: "continue" };
        }
        /**
         * Strategy: pathname-prefix-other-locales
         * Default locale has no prefix, other locales must have a prefix.
         */
        matchPrefixOtherLocales(pathname, _context) {
          let pathnameContainsDefaultLocale = false;
          for (const segment of pathname.split("/")) {
            if (normalizeTheLocale(segment) === normalizeTheLocale(__privateGet(this, _defaultLocale))) {
              pathnameContainsDefaultLocale = true;
              break;
            }
          }
          if (pathnameContainsDefaultLocale) {
            const newLocation = pathname.replace(`/${__privateGet(this, _defaultLocale)}`, "");
            return {
              type: "notFound",
              location: newLocation
            };
          }
          return { type: "continue" };
        }
        /**
         * Strategy: pathname-prefix-always-no-redirect
         * Like prefix-always but allows root to serve instead of redirecting
         */
        matchPrefixAlwaysNoRedirect(pathname, _context) {
          const isRoot = pathname === __privateGet(this, _base2) + "/" || pathname === __privateGet(this, _base2);
          if (isRoot) {
            return { type: "continue" };
          }
          if (!pathHasLocale(pathname, __privateGet(this, _locales))) {
            return { type: "notFound" };
          }
          return { type: "continue" };
        }
        /**
         * Check if the current locale doesn't belong to the configured domain.
         * Used for domain-based routing strategies.
         */
        localeHasntDomain(currentLocale, currentDomain) {
          if (!__privateGet(this, _domains) || !currentDomain) {
            return false;
          }
          if (!currentLocale) {
            return false;
          }
          const localesForDomain = __privateGet(this, _domains)[currentDomain];
          if (!localesForDomain) {
            return true;
          }
          return !localesForDomain.includes(currentLocale);
        }
      };
      _strategy = new WeakMap();
      _defaultLocale = new WeakMap();
      _locales = new WeakMap();
      _base2 = new WeakMap();
      _domains = new WeakMap();
      I18n = class {
        constructor(i18n, base, trailingSlash, format) {
          __privateAdd(this, _i18n);
          __privateAdd(this, _base3);
          __privateAdd(this, _trailingSlash2);
          __privateAdd(this, _format);
          __privateAdd(this, _router2);
          __privateSet(this, _i18n, i18n);
          __privateSet(this, _base3, base);
          __privateSet(this, _trailingSlash2, trailingSlash);
          __privateSet(this, _format, format);
          __privateSet(this, _router2, new I18nRouter({
            strategy: i18n.strategy,
            defaultLocale: i18n.defaultLocale,
            locales: i18n.locales,
            base,
            domains: i18n.domainLookupTable ? Object.keys(i18n.domainLookupTable).reduce(
              (acc, domain) => {
                const locale = i18n.domainLookupTable[domain];
                if (!acc[domain]) {
                  acc[domain] = [];
                }
                acc[domain].push(locale);
                return acc;
              },
              {}
            ) : void 0
          }));
        }
        finalize(state, response) {
          return __async(this, null, function* () {
            var _a29;
            state.pipeline.usedFeatures |= PipelineFeatures.i18n;
            const i18n = __privateGet(this, _i18n);
            const typeHeader = response.headers.get(ROUTE_TYPE_HEADER);
            if (typeHeader) {
              response.headers.delete(ROUTE_TYPE_HEADER);
            }
            const isReroute = response.headers.get(REROUTE_DIRECTIVE_HEADER);
            if (isReroute === "no" && typeof i18n.fallback === "undefined") {
              return response;
            }
            if (typeHeader !== "page" && typeHeader !== "fallback") {
              return response;
            }
            const url = state.url;
            const currentLocale = state.computeCurrentLocale();
            const isPrerendered = state.routeData.prerender;
            const routerContext = {
              currentLocale,
              currentDomain: url.hostname,
              routeType: typeHeader,
              isReroute: isReroute === "yes"
            };
            const routeDecision = __privateGet(this, _router2).match(url.pathname, routerContext);
            switch (routeDecision.type) {
              case "redirect": {
                let location = routeDecision.location;
                if (shouldAppendForwardSlash(__privateGet(this, _trailingSlash2), __privateGet(this, _format))) {
                  location = appendForwardSlash(location);
                }
                return new Response(null, {
                  status: (_a29 = routeDecision.status) != null ? _a29 : 302,
                  headers: { Location: location }
                });
              }
              case "notFound": {
                if (isPrerendered) {
                  const prerenderedRes = new Response(response.body, {
                    status: 404,
                    headers: response.headers
                  });
                  prerenderedRes.headers.set(REROUTE_DIRECTIVE_HEADER, "no");
                  if (routeDecision.location) {
                    prerenderedRes.headers.set("Location", routeDecision.location);
                  }
                  return prerenderedRes;
                }
                const headers = new Headers();
                if (routeDecision.location) {
                  headers.set("Location", routeDecision.location);
                }
                return new Response(null, { status: 404, headers });
              }
            }
            if (i18n.fallback && i18n.fallbackType) {
              const effectiveStatus = typeHeader === "fallback" ? 404 : response.status;
              const fallbackDecision = computeFallbackRoute({
                pathname: url.pathname,
                responseStatus: effectiveStatus,
                fallback: i18n.fallback,
                fallbackType: i18n.fallbackType,
                locales: i18n.locales,
                defaultLocale: i18n.defaultLocale,
                strategy: i18n.strategy,
                base: __privateGet(this, _base3)
              });
              switch (fallbackDecision.type) {
                case "redirect":
                  return new Response(null, {
                    status: 302,
                    headers: { Location: fallbackDecision.pathname + url.search }
                  });
                case "rewrite":
                  return yield state.rewrite(fallbackDecision.pathname + url.search);
              }
            }
            return response;
          });
        }
      };
      _i18n = new WeakMap();
      _base3 = new WeakMap();
      _trailingSlash2 = new WeakMap();
      _format = new WeakMap();
      _router2 = new WeakMap();
      EMPTY_OPTIONS = Object.freeze({ tags: [] });
      NoopAstroCache = class {
        constructor() {
          __publicField(this, "enabled", false);
        }
        set() {
        }
        get tags() {
          return [];
        }
        get options() {
          return EMPTY_OPTIONS;
        }
        invalidate() {
          return __async(this, null, function* () {
          });
        }
      };
      hasWarned = false;
      DisabledAstroCache = class {
        constructor(logger) {
          __privateAdd(this, _DisabledAstroCache_instances);
          __publicField(this, "enabled", false);
          __privateAdd(this, _logger2);
          __privateSet(this, _logger2, logger);
        }
        set() {
          __privateMethod(this, _DisabledAstroCache_instances, warn_fn).call(this);
        }
        get tags() {
          return [];
        }
        get options() {
          return EMPTY_OPTIONS;
        }
        invalidate() {
          return __async(this, null, function* () {
            throw new AstroError(CacheNotEnabled);
          });
        }
      };
      _logger2 = new WeakMap();
      _DisabledAstroCache_instances = new WeakSet();
      warn_fn = function() {
        var _a29;
        if (!hasWarned) {
          hasWarned = true;
          (_a29 = __privateGet(this, _logger2)) == null ? void 0 : _a29.warn(
            "cache",
            "`cache.set()` was called but caching is not enabled. Configure a cache provider in your Astro config under `experimental.cache` to enable caching."
          );
        }
      };
      AstroMiddleware = class {
        constructor(pipeline) {
          __privateAdd(this, _AstroMiddleware_instances);
          __privateAdd(this, _pipeline);
          __privateSet(this, _pipeline, pipeline);
        }
        handle(state, renderRouteCallback) {
          return __async(this, null, function* () {
            state.pipeline.usedFeatures |= PipelineFeatures.middleware;
            const pipeline = __privateGet(this, _pipeline);
            yield state.getProps();
            const apiContext = state.getAPIContext();
            state.counter++;
            if (state.counter === 4) {
              return new Response("Loop Detected", {
                // https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/508
                status: 508,
                statusText: "Astro detected a loop where you tried to call the rewriting logic more than four times."
              });
            }
            const next = (ctx, payload) => __async(this, null, function* () {
              if (payload) {
                pipeline.logger.debug("router", "Called rewriting to:", payload);
                const result = yield pipeline.tryRewrite(payload, state.request);
                applyRewriteToState(state, payload, result);
              }
              return renderRouteCallback(state, ctx);
            });
            let response;
            if (state.skipMiddleware) {
              response = yield next(apiContext);
            } else {
              const pipelineMiddleware = yield pipeline.getMiddleware();
              const composed = sequence(...pipeline.internalMiddleware, pipelineMiddleware);
              response = yield callMiddleware(composed, apiContext, next);
            }
            response = __privateMethod(this, _AstroMiddleware_instances, finalize_fn).call(this, state, response);
            state.response = response;
            return response;
          });
        }
      };
      _pipeline = new WeakMap();
      _AstroMiddleware_instances = new WeakSet();
      finalize_fn = function(state, response) {
        attachCookiesToResponse(response, state.cookies);
        return response;
      };
      EMPTY_SLOTS = Object.freeze({});
      PagesHandler = class {
        constructor(pipeline) {
          __privateAdd(this, _pipeline2);
          __privateSet(this, _pipeline2, pipeline);
        }
        handle(state, ctx) {
          return __async(this, null, function* () {
            var _a29;
            const pipeline = __privateGet(this, _pipeline2);
            const { logger, streaming } = pipeline;
            let response;
            const componentInstance = yield state.loadComponentInstance();
            switch (state.routeData.type) {
              case "endpoint": {
                response = yield renderEndpoint(
                  componentInstance,
                  ctx,
                  state.routeData.prerender,
                  logger
                );
                break;
              }
              case "page": {
                const props = yield state.getProps();
                const actionApiContext = state.getActionAPIContext();
                const result = yield state.createResult(componentInstance, actionApiContext);
                try {
                  response = yield renderPage(
                    result,
                    componentInstance == null ? void 0 : componentInstance.default,
                    props,
                    (_a29 = state.slots) != null ? _a29 : EMPTY_SLOTS,
                    streaming,
                    state.routeData
                  );
                } catch (e2) {
                  result.cancelled = true;
                  throw e2;
                }
                response.headers.set(ROUTE_TYPE_HEADER, "page");
                if (state.routeData.route === "/404" || state.routeData.route === "/500") {
                  response.headers.set(REROUTE_DIRECTIVE_HEADER, "no");
                }
                if (state.isRewriting) {
                  response.headers.set(REWRITE_DIRECTIVE_HEADER_KEY, REWRITE_DIRECTIVE_HEADER_VALUE);
                }
                break;
              }
              case "redirect": {
                return new Response(null, { status: 404, headers: { [ASTRO_ERROR_HEADER]: "true" } });
              }
              case "fallback": {
                return new Response(null, { status: 500, headers: { [ROUTE_TYPE_HEADER]: "fallback" } });
              }
            }
            const responseCookies = getCookiesFromResponse(response);
            if (responseCookies) {
              state.cookies.merge(responseCookies);
            }
            state.response = response;
            return response;
          });
        }
      };
      _pipeline2 = new WeakMap();
      Rewrites = class {
        execute(state, payload) {
          return __async(this, null, function* () {
            const pipeline = state.pipeline;
            pipeline.logger.debug("router", "Calling rewrite: ", payload);
            const result = yield pipeline.tryRewrite(payload, state.request);
            applyRewriteToState(state, payload, result, { mergeCookies: true });
            const middleware = new AstroMiddleware(pipeline);
            const pagesHandler = new PagesHandler(pipeline);
            return middleware.handle(state, pagesHandler.handle.bind(pagesHandler));
          });
        }
      };
      renderOptionsSymbol = /* @__PURE__ */ Symbol.for("astro.renderOptions");
      FetchState = class {
        constructor(pipeline, request, options) {
          __privateAdd(this, _FetchState_instances);
          __publicField(this, "pipeline");
          /**
           * The request to render. Mutated during rewrites so subsequent renders
           * see the rewritten URL.
           */
          __publicField(this, "request");
          __publicField(this, "routeData");
          /**
           * The pathname to use for routing and rendering. Starts out as the raw,
           * base-stripped, decoded pathname from the request. May be further
           * normalized by `AstroHandler` after routeData is known (in dev, when
           * the matched route has no `.html` extension, `.html` / `/index.html`
           * suffixes are stripped).
           */
          __publicField(this, "pathname");
          /** Resolved render options (addCookieHeader, clientAddress, locals, etc.). */
          __publicField(this, "renderOptions");
          /** When the request started, used to log duration. */
          __publicField(this, "timeStart");
          /**
           * The route's loaded component module. Set before middleware runs; may
           * be swapped during in-flight rewrites from inside the middleware chain.
           */
          __publicField(this, "componentInstance");
          /**
           * Slot overrides supplied by the container API. `undefined` for HTTP
           * requests — `PagesHandler` coalesces to `{}` on read so we don't
           * allocate an empty object per request.
           */
          __publicField(this, "slots");
          /**
           * The `Response` produced by handlers, if any. Set after page
           * rendering or middleware completes.
           */
          __publicField(this, "response");
          /**
           * Default HTTP status for the rendered response. Callers override
           * before rendering runs (e.g. `AstroHandler` sets this from
           * `BaseApp.getDefaultStatusCode`; error handlers set `404` / `500`).
           */
          __publicField(this, "status", 200);
          /** Whether user middleware should be skipped for this request. */
          __publicField(this, "skipMiddleware", false);
          /**
           * Set to `true` when the request path was encoded too many times to fully
           * decode (see {@link validateAndDecodePathname}). These requests are
           * rejected with a `400` before middleware or routing run.
           */
          __publicField(this, "invalidEncoding", false);
          /** A flag that tells the render content if the rewriting was triggered. */
          __publicField(this, "isRewriting", false);
          /** A safety net in case of loops (rewrite counter). */
          __publicField(this, "counter", 0);
          /** Cookies for this request. Created lazily on first access. */
          __publicField(this, "cookies");
          /** Route params derived from routeData + pathname. Computed lazily. */
          __privateAdd(this, _params);
          /** Normalized URL for this request. */
          __publicField(this, "url");
          /** Client address for this request. */
          __publicField(this, "clientAddress");
          /** Whether this is a partial render (container API). */
          __publicField(this, "partial");
          /** Whether to inject CSP meta tags. */
          __publicField(this, "shouldInjectCspMetaTags");
          /** Request-scoped locals object, shared with user middleware. */
          __publicField(this, "locals", {});
          /**
           * Memoized `props` (see `getProps`). `null` means "not yet computed"
           * — using `null` (rather than `undefined`) keeps the hidden class
           * stable and distinct from a valid-but-empty result.
           */
          __publicField(this, "props", null);
          /** Memoized `ActionAPIContext` (see `getActionAPIContext`). */
          __publicField(this, "actionApiContext", null);
          /** Memoized `APIContext` (see `getAPIContext`). */
          __publicField(this, "apiContext", null);
          /** Registered context providers keyed by name. Lazy-initialized on first provide(). */
          __privateAdd(this, _providers);
          /** Cached values from resolved providers. Lazy-initialized on first resolve(). */
          __privateAdd(this, _providersResolvedValues);
          /** Cached promise for lazy component instance loading. */
          __privateAdd(this, _componentInstancePromise);
          /** SSR result for the current page render. */
          __publicField(this, "result");
          /** Initial props (from container/error handler). */
          __publicField(this, "initialProps", {});
          /** Rewrites handler instance. Lazy-initialized on first rewrite(). */
          __privateAdd(this, _rewrites);
          /** Memoized Astro page partial. */
          __privateAdd(this, _astroPagePartial);
          /**
           * Locale-prefixed pathname derived from the Host header for domain-based
           * i18n routing (e.g. `/en/boats/1/foo`), or `undefined` when the request
           * isn't served from a locale-mapped domain. When set, `this.pathname` is
           * derived from it so locale/param resolution match the route pattern.
           */
          __privateAdd(this, _domainPathname);
          /** Memoized current locale. */
          __privateAdd(this, _currentLocale);
          /** Memoized preferred locale. */
          __privateAdd(this, _preferredLocale);
          /** Memoized preferred locale list. */
          __privateAdd(this, _preferredLocaleList);
          var _a29;
          this.pipeline = pipeline;
          this.request = request;
          options != null ? options : options = getRenderOptions(request);
          this.routeData = options == null ? void 0 : options.routeData;
          this.renderOptions = options != null ? options : {
            addCookieHeader: false,
            clientAddress: void 0,
            locals: void 0,
            prerenderedErrorPageFetch: fetch,
            routeData: void 0,
            waitUntil: void 0
          };
          this.componentInstance = void 0;
          this.slots = void 0;
          const url = new URL(request.url);
          const domainPathname = computePathnameFromDomain(
            request,
            url,
            pipeline.manifest.i18n,
            pipeline.manifest.base,
            pipeline.manifest.trailingSlash,
            pipeline.logger
          );
          if (domainPathname) {
            __privateSet(this, _domainPathname, domainPathname);
            try {
              this.pathname = decodeURI(domainPathname);
            } catch (e2) {
              this.pathname = domainPathname;
            }
          } else {
            this.pathname = __privateMethod(this, _FetchState_instances, computePathname_fn).call(this, url);
          }
          this.timeStart = performance.now();
          this.clientAddress = options == null ? void 0 : options.clientAddress;
          this.locals = (_a29 = options == null ? void 0 : options.locals) != null ? _a29 : {};
          this.url = normalizeUrl(url);
          this.cookies = new AstroCookies(request);
          if (pipeline.manifest.allowedDomains && pipeline.manifest.allowedDomains.length > 0) {
            __privateMethod(this, _FetchState_instances, applyForwardedHeaders_fn).call(this);
          }
          if (!Reflect.get(this.request, originPathnameSymbol)) {
            setOriginPathname(
              this.request,
              this.pathname,
              pipeline.manifest.trailingSlash,
              pipeline.manifest.buildFormat
            );
          }
          __privateMethod(this, _FetchState_instances, resolveRouteData_fn).call(this);
        }
        get params() {
          if (!__privateGet(this, _params) && this.routeData) {
            __privateSet(this, _params, getParams(this.routeData, this.pathname));
          }
          return __privateGet(this, _params);
        }
        set params(value) {
          __privateSet(this, _params, value);
        }
        /**
         * Triggers a rewrite. Delegates to the Rewrites handler.
         */
        rewrite(payload) {
          var _a29;
          return ((_a29 = __privateGet(this, _rewrites)) != null ? _a29 : __privateSet(this, _rewrites, new Rewrites())).execute(this, payload);
        }
        /**
         * Creates the SSR result for the current page render.
         */
        createResult(mod, ctx) {
          return __async(this, null, function* () {
            var _a29, _b13, _c, _d, _f, _g, _h, _i, _j, _k, _l, _m, _n, _o, _p, _q;
            const pipeline = this.pipeline;
            const { clientDirectives, inlinedScripts, compressHTML, manifest: manifest2, renderers: renderers2, resolve } = pipeline;
            const routeData = this.routeData;
            const { links, scripts, styles } = yield pipeline.headElements(routeData);
            const extraStyleHashes = [];
            const extraScriptHashes = [];
            const shouldInjectCspMetaTags = (_a29 = this.shouldInjectCspMetaTags) != null ? _a29 : manifest2.shouldInjectCspMetaTags;
            const cspAlgorithm = (_c = (_b13 = manifest2.csp) == null ? void 0 : _b13.algorithm) != null ? _c : "SHA-256";
            if (shouldInjectCspMetaTags) {
              for (const style of styles) {
                extraStyleHashes.push(yield generateCspDigest(style.children, cspAlgorithm));
              }
              for (const script of scripts) {
                extraScriptHashes.push(yield generateCspDigest(script.children, cspAlgorithm));
              }
            }
            const componentMetadata = (_d = yield pipeline.componentMetadata(routeData)) != null ? _d : manifest2.componentMetadata;
            const headers = new Headers({ "Content-Type": "text/html" });
            const partial = typeof this.partial === "boolean" ? this.partial : Boolean(mod.partial);
            const actionResult = hasActionPayload(this.locals) ? deserializeActionResult(this.locals._actionPayload.actionResult) : void 0;
            const status = this.status;
            const response = {
              status: (actionResult == null ? void 0 : actionResult.error) ? actionResult == null ? void 0 : actionResult.error.status : status,
              statusText: (actionResult == null ? void 0 : actionResult.error) ? actionResult == null ? void 0 : actionResult.error.type : "OK",
              get headers() {
                return headers;
              },
              set headers(_) {
                throw new AstroError(AstroResponseHeadersReassigned);
              }
            };
            const state = this;
            const result = {
              base: manifest2.base,
              userAssetsBase: manifest2.userAssetsBase,
              cancelled: false,
              clientDirectives,
              inlinedScripts,
              componentMetadata,
              compressHTML,
              cookies: this.cookies,
              createAstro: (props, slots) => state.createAstro(result, props, slots, ctx),
              links,
              // SAFETY: createResult is only called after route resolution, so routeData
              // is always set and the params getter always returns a value.
              params: this.params,
              partial,
              pathname: this.pathname,
              renderers: renderers2,
              resolve,
              response,
              request: this.request,
              scripts,
              styles,
              actionResult,
              getServerIslandNameMap() {
                return __async(this, null, function* () {
                  var _a30;
                  const serverIslands = yield pipeline.getServerIslands();
                  return (_a30 = serverIslands.serverIslandNameMap) != null ? _a30 : /* @__PURE__ */ new Map();
                });
              },
              key: manifest2.key,
              trailingSlash: manifest2.trailingSlash,
              _experimentalQueuedRendering: {
                pool: pipeline.nodePool,
                htmlStringCache: pipeline.htmlStringCache,
                enabled: (_f = manifest2.experimentalQueuedRendering) == null ? void 0 : _f.enabled,
                poolSize: (_g = manifest2.experimentalQueuedRendering) == null ? void 0 : _g.poolSize,
                contentCache: (_h = manifest2.experimentalQueuedRendering) == null ? void 0 : _h.contentCache
              },
              _metadata: {
                hasHydrationScript: false,
                rendererSpecificHydrationScripts: /* @__PURE__ */ new Set(),
                hasRenderedHead: false,
                renderedScripts: /* @__PURE__ */ new Set(),
                hasDirectives: /* @__PURE__ */ new Set(),
                hasRenderedServerIslandRuntime: false,
                headInTree: false,
                extraHead: [],
                extraStyleHashes,
                extraScriptHashes,
                propagators: /* @__PURE__ */ new Set(),
                templateDepth: 0
              },
              cspDestination: (_j = (_i = manifest2.csp) == null ? void 0 : _i.cspDestination) != null ? _j : routeData.prerender ? "meta" : "header",
              shouldInjectCspMetaTags,
              cspAlgorithm,
              scriptHashes: ((_k = manifest2.csp) == null ? void 0 : _k.scriptHashes) ? [...manifest2.csp.scriptHashes] : [],
              scriptResources: ((_l = manifest2.csp) == null ? void 0 : _l.scriptResources) ? [...manifest2.csp.scriptResources] : [],
              styleHashes: ((_m = manifest2.csp) == null ? void 0 : _m.styleHashes) ? [...manifest2.csp.styleHashes] : [],
              styleResources: ((_n = manifest2.csp) == null ? void 0 : _n.styleResources) ? [...manifest2.csp.styleResources] : [],
              directives: ((_o = manifest2.csp) == null ? void 0 : _o.directives) ? [...manifest2.csp.directives] : [],
              isStrictDynamic: (_q = (_p = manifest2.csp) == null ? void 0 : _p.isStrictDynamic) != null ? _q : false,
              internalFetchHeaders: manifest2.internalFetchHeaders
            };
            this.result = result;
            return result;
          });
        }
        /**
         * Creates the Astro global object for a component render.
         */
        createAstro(result, props, slotValues, apiContext) {
          var _a29;
          let astroPagePartial;
          if (this.isRewriting) {
            __privateSet(this, _astroPagePartial, this.createAstroPagePartial(result, apiContext));
          }
          (_a29 = __privateGet(this, _astroPagePartial)) != null ? _a29 : __privateSet(this, _astroPagePartial, this.createAstroPagePartial(result, apiContext));
          astroPagePartial = __privateGet(this, _astroPagePartial);
          const astroComponentPartial = { props, self: null };
          const Astro = Object.assign(
            Object.create(astroPagePartial),
            astroComponentPartial
          );
          let _slots2;
          Object.defineProperty(Astro, "slots", {
            get: () => {
              if (!_slots2) {
                _slots2 = new Slots(
                  result,
                  slotValues,
                  this.pipeline.logger
                );
              }
              return _slots2;
            }
          });
          return Astro;
        }
        /**
         * Creates the Astro page-level partial (prototype for Astro global).
         */
        createAstroPagePartial(result, apiContext) {
          const state = this;
          const { cookies, locals, params, pipeline, url } = this;
          const { response } = result;
          const redirect = (path, status = 302) => {
            if (state.request[responseSentSymbol$1]) {
              throw new AstroError(__spreadValues({}, ResponseSentError));
            }
            return new Response(null, { status, headers: { Location: path } });
          };
          const rewrite = (reroutePayload) => __async(this, null, function* () {
            return yield state.rewrite(reroutePayload);
          });
          const callAction = createCallAction(apiContext);
          const partial = {
            generator: ASTRO_GENERATOR,
            routePattern: this.routeData.route,
            isPrerendered: this.routeData.prerender,
            cookies,
            get clientAddress() {
              return state.getClientAddress();
            },
            get currentLocale() {
              return state.computeCurrentLocale();
            },
            params,
            get preferredLocale() {
              return state.computePreferredLocale();
            },
            get preferredLocaleList() {
              return state.computePreferredLocaleList();
            },
            locals,
            redirect,
            rewrite,
            request: this.request,
            response,
            site: pipeline.site,
            getActionResult: createGetActionResult(locals),
            get callAction() {
              return callAction;
            },
            url,
            get originPathname() {
              return getOriginPathname(state.request);
            },
            get csp() {
              return state.getCsp();
            },
            get logger() {
              return {
                info(msg) {
                  pipeline.logger.info(null, msg);
                },
                warn(msg) {
                  pipeline.logger.warn(null, msg);
                },
                error(msg) {
                  pipeline.logger.error(null, msg);
                }
              };
            }
          };
          this.defineProviderGetters(partial);
          return partial;
        }
        getClientAddress() {
          const { pipeline, clientAddress } = this;
          const routeData = this.routeData;
          if (routeData.prerender) {
            throw new AstroError(__spreadProps(__spreadValues({}, PrerenderClientAddressNotAvailable), {
              message: PrerenderClientAddressNotAvailable.message(routeData.component)
            }));
          }
          if (clientAddress) {
            return clientAddress;
          }
          if (pipeline.adapterName) {
            throw new AstroError(__spreadProps(__spreadValues({}, ClientAddressNotAvailable), {
              message: ClientAddressNotAvailable.message(pipeline.adapterName)
            }));
          }
          throw new AstroError(StaticClientAddressNotAvailable);
        }
        getCookies() {
          return this.cookies;
        }
        getCsp() {
          const state = this;
          const { pipeline } = this;
          if (!pipeline.manifest.csp) {
            if (pipeline.runtimeMode === "production") {
              pipeline.logger.warn(
                "csp",
                `context.csp was used when rendering the route ${s.green(state.routeData.route)}, but CSP was not configured. For more information, see https://docs.astro.build/en/reference/configuration-reference/#securitycsp`
              );
            }
            return void 0;
          }
          return {
            insertDirective(payload) {
              if (state.result) {
                state.result.directives = pushDirective(state.result.directives, payload);
              }
            },
            insertScriptResource(resource) {
              var _a29;
              (_a29 = state.result) == null ? void 0 : _a29.scriptResources.push(resource);
            },
            insertStyleResource(resource) {
              var _a29;
              (_a29 = state.result) == null ? void 0 : _a29.styleResources.push(resource);
            },
            insertStyleHash(hash) {
              var _a29;
              (_a29 = state.result) == null ? void 0 : _a29.styleHashes.push(hash);
            },
            insertScriptHash(hash) {
              var _a29;
              (_a29 = state.result) == null ? void 0 : _a29.scriptHashes.push(hash);
            }
          };
        }
        computeCurrentLocale() {
          var _a29;
          const {
            url,
            pipeline: { i18n },
            routeData
          } = this;
          if (!i18n || !routeData) return;
          const { defaultLocale, locales, strategy } = i18n;
          const fallbackTo = strategy === "pathname-prefix-other-locales" || strategy === "domains-prefix-other-locales" ? defaultLocale : void 0;
          if (__privateGet(this, _currentLocale)) {
            return __privateGet(this, _currentLocale);
          }
          let computedLocale;
          if (isRouteServerIsland(routeData)) {
            let referer = this.request.headers.get("referer");
            if (referer) {
              if (URL.canParse(referer)) {
                referer = new URL(referer).pathname;
              }
              computedLocale = computeCurrentLocale(referer, locales, defaultLocale);
            }
          } else {
            let pathname = routeData.pathname;
            if (__privateGet(this, _domainPathname)) {
              pathname = this.pathname;
            } else if (url && !routeData.pattern.test(url.pathname)) {
              for (const fallbackRoute of routeData.fallbackRoutes) {
                if (fallbackRoute.pattern.test(url.pathname)) {
                  pathname = fallbackRoute.pathname;
                  break;
                }
              }
            }
            pathname = pathname && !isRoute404or500(routeData) ? pathname : (_a29 = url.pathname) != null ? _a29 : this.pathname;
            computedLocale = computeCurrentLocale(pathname, locales, defaultLocale);
            if (routeData.params.length > 0) {
              const localeFromParams = computeCurrentLocaleFromParams(this.params, locales);
              if (localeFromParams) {
                computedLocale = localeFromParams;
              }
            }
          }
          __privateSet(this, _currentLocale, computedLocale != null ? computedLocale : fallbackTo);
          return __privateGet(this, _currentLocale);
        }
        computePreferredLocale() {
          var _a29;
          const {
            pipeline: { i18n },
            request
          } = this;
          if (!i18n) return;
          return (_a29 = __privateGet(this, _preferredLocale)) != null ? _a29 : __privateSet(this, _preferredLocale, computePreferredLocale(request, i18n.locales));
        }
        computePreferredLocaleList() {
          var _a29;
          const {
            pipeline: { i18n },
            request
          } = this;
          if (!i18n) return;
          return (_a29 = __privateGet(this, _preferredLocaleList)) != null ? _a29 : __privateSet(this, _preferredLocaleList, computePreferredLocaleList(request, i18n.locales));
        }
        /**
         * Lazily loads the route's component module. Returns the cached
         * instance if already loaded. The promise is cached so concurrent
         * callers share the same load.
         */
        loadComponentInstance() {
          return __async(this, null, function* () {
            if (this.componentInstance) return this.componentInstance;
            if (__privateGet(this, _componentInstancePromise)) return __privateGet(this, _componentInstancePromise);
            __privateSet(this, _componentInstancePromise, this.pipeline.getComponentByRoute(this.routeData).then((mod) => {
              this.componentInstance = mod;
              return mod;
            }));
            return __privateGet(this, _componentInstancePromise);
          });
        }
        /**
         * Registers a context provider under the given key. Handlers call
         * this to contribute values to the request context (e.g. sessions).
         * The `create` factory is called lazily on the first `resolve(key)`.
         */
        provide(key, provider) {
          var _a29;
          ((_a29 = __privateGet(this, _providers)) != null ? _a29 : __privateSet(this, _providers, /* @__PURE__ */ new Map())).set(key, provider);
        }
        /**
         * Lazily resolves a provider registered under `key`. Calls
         * `provider.create()` on first access and caches the result.
         * Returns `undefined` if no provider was registered for the key.
         */
        resolve(key) {
          var _a29, _b13, _c;
          if ((_a29 = __privateGet(this, _providersResolvedValues)) == null ? void 0 : _a29.has(key)) {
            return __privateGet(this, _providersResolvedValues).get(key);
          }
          const provider = (_b13 = __privateGet(this, _providers)) == null ? void 0 : _b13.get(key);
          if (!provider) return void 0;
          const value = provider.create();
          ((_c = __privateGet(this, _providersResolvedValues)) != null ? _c : __privateSet(this, _providersResolvedValues, /* @__PURE__ */ new Map())).set(key, value);
          return value;
        }
        /**
         * Runs all registered `finalize` callbacks. Should be called after
         * the response is produced, typically in a `finally` block.
         *
         * Returns synchronously (no promise allocation) when nothing needs
         * finalizing — important for the hot path where sessions are not used.
         */
        finalizeAll() {
          if (!__privateGet(this, _providersResolvedValues) || __privateGet(this, _providersResolvedValues).size === 0) return;
          let chain;
          for (const [key, provider] of __privateGet(this, _providers)) {
            if (provider.finalize && __privateGet(this, _providersResolvedValues).has(key)) {
              const result = provider.finalize(__privateGet(this, _providersResolvedValues).get(key));
              if (result) {
                chain = chain ? chain.then(() => result) : result;
              }
            }
          }
          return chain;
        }
        /**
         * Adds lazy getters to `target` for each registered provider key.
         * Used by context creation (APIContext, Astro global) so that
         * provider values like `session` and `cache` appear as properties
         * without hard-coding the keys.
         */
        defineProviderGetters(target) {
          if (!__privateGet(this, _providers)) return;
          const state = this;
          for (const key of __privateGet(this, _providers).keys()) {
            Object.defineProperty(target, key, {
              get: () => state.resolve(key),
              enumerable: true,
              configurable: true
            });
          }
        }
        /**
         * Returns the resolved `props` for this render, computing them lazily
         * from the route + component module on first access. If the
         * `initialProps` already carries user-supplied props (e.g. the
         * container API) those are used verbatim.
         */
        getProps() {
          return __async(this, null, function* () {
            if (this.props !== null) return this.props;
            if (Object.keys(this.initialProps).length > 0) {
              this.props = this.initialProps;
              return this.props;
            }
            const pipeline = this.pipeline;
            const mod = yield this.loadComponentInstance();
            this.props = yield getProps({
              mod,
              routeData: this.routeData,
              routeCache: pipeline.routeCache,
              pathname: this.pathname,
              logger: pipeline.logger,
              serverLike: pipeline.manifest.serverLike,
              base: pipeline.manifest.base,
              trailingSlash: pipeline.manifest.trailingSlash
            });
            return this.props;
          });
        }
        /**
         * Returns the `ActionAPIContext` for this render, creating it lazily.
         * Used by middleware, actions, and page dispatch.
         */
        getActionAPIContext() {
          if (this.actionApiContext !== null) return this.actionApiContext;
          const state = this;
          const ctx = {
            get cookies() {
              return state.cookies;
            },
            routePattern: this.routeData.route,
            isPrerendered: this.routeData.prerender,
            get clientAddress() {
              return state.getClientAddress();
            },
            get currentLocale() {
              return state.computeCurrentLocale();
            },
            generator: ASTRO_GENERATOR,
            get locals() {
              return state.locals;
            },
            set locals(_) {
              throw new AstroError(LocalsReassigned);
            },
            // SAFETY: getActionAPIContext is only called after route resolution,
            // so routeData is always set and the params getter always returns a value.
            params: this.params,
            get preferredLocale() {
              return state.computePreferredLocale();
            },
            get preferredLocaleList() {
              return state.computePreferredLocaleList();
            },
            request: this.request,
            site: this.pipeline.site,
            url: this.url,
            get originPathname() {
              return getOriginPathname(state.request);
            },
            get csp() {
              return state.getCsp();
            },
            get logger() {
              if (!state.pipeline.manifest.experimentalLogger) {
                state.pipeline.logger.warn(
                  null,
                  "The Astro.logger is available only when experimental.logger is defined."
                );
                return void 0;
              }
              return {
                info(msg) {
                  state.pipeline.logger.info(null, msg);
                },
                warn(msg) {
                  state.pipeline.logger.warn(null, msg);
                },
                error(msg) {
                  state.pipeline.logger.error(null, msg);
                }
              };
            }
          };
          this.defineProviderGetters(ctx);
          this.actionApiContext = ctx;
          return this.actionApiContext;
        }
        /**
         * Returns the `APIContext` for this render, creating it lazily from
         * the memoized props + action context.
         *
         * Callers must ensure `getProps()` has resolved at least once before
         * calling this.
         */
        getAPIContext() {
          if (this.apiContext !== null) return this.apiContext;
          const actionApiContext = this.getActionAPIContext();
          const state = this;
          const redirect = (path, status = 302) => new Response(null, { status, headers: { Location: path } });
          const rewrite = (reroutePayload) => __async(this, null, function* () {
            return yield state.rewrite(reroutePayload);
          });
          Reflect.set(actionApiContext, pipelineSymbol, this.pipeline);
          actionApiContext[fetchStateSymbol] = this;
          this.apiContext = Object.assign(actionApiContext, {
            props: this.props,
            redirect,
            rewrite,
            getActionResult: createGetActionResult(actionApiContext.locals),
            callAction: createCallAction(actionApiContext)
          });
          return this.apiContext;
        }
        /**
         * Invalidates the cached `APIContext` so the next `getAPIContext()`
         * call re-derives it from the (possibly mutated) state. Used
         * after an in-flight rewrite swaps the route / request / params.
         */
        invalidateContexts() {
          this.props = null;
          this.actionApiContext = null;
          this.apiContext = null;
        }
      };
      _params = new WeakMap();
      _providers = new WeakMap();
      _providersResolvedValues = new WeakMap();
      _componentInstancePromise = new WeakMap();
      _rewrites = new WeakMap();
      _astroPagePartial = new WeakMap();
      _domainPathname = new WeakMap();
      _currentLocale = new WeakMap();
      _preferredLocale = new WeakMap();
      _preferredLocaleList = new WeakMap();
      _FetchState_instances = new WeakSet();
      /**
       * Resolves the route to use for this request and stores it on
       * `this.routeData`. If the adapter (or the dev server) provided a
       * `routeData` via render options it's already set and this is a
       * no-op. Otherwise we use the app's synchronous route matcher and
       * fall back to a `404.astro` route so middleware can still run.
       *
       * Called eagerly from the constructor so individual handlers
       * (actions, pages, middleware, etc.) always see a resolved route
       * without the caller needing an extra setup step.
       *
       * Once routeData is known, finalizes `this.pathname`: in dev, if the
       * matched route has no `.html` extension, strip `.html` / `/index.html`
       * suffixes so the rendering pipeline sees the canonical pathname.
       */
      /**
       * Strip `.html` / `/index.html` suffixes from the pathname so the
       * rendering pipeline sees the canonical route path. Only applies to
       * page routes where `.html` is framework-injected. Endpoint routes
       * preserve `.html` because any such suffix is user-provided (e.g.
       * from `getStaticPaths` params). Skipped when the matched route
       * itself has an `.html` extension in its definition.
       */
      stripHtmlExtension_fn = function() {
        if (this.routeData && this.routeData.type === "page" && !routeHasHtmlExtension(this.routeData)) {
          this.pathname = this.pathname.replace(/\/index\.html$/, "/").replace(/\.html$/, "");
        }
      };
      resolveRouteData_fn = function() {
        const pipeline = this.pipeline;
        if (this.routeData) {
          __privateMethod(this, _FetchState_instances, stripHtmlExtension_fn).call(this);
          return;
        }
        const matched = pipeline.matchRoute(this.pathname);
        if (matched && matched.prerender && pipeline.manifest.serverLike) {
          if (matched.params.length > 0) {
            const allMatches = pipeline.matchAllRoutes(this.pathname);
            this.routeData = allMatches.find((r2) => !r2.prerender);
          } else {
            this.routeData = void 0;
          }
        } else {
          this.routeData = matched;
        }
        pipeline.logger.debug("router", "Astro matched the following route for " + this.request.url);
        pipeline.logger.debug("router", "RouteData:\n" + this.routeData);
        if (!this.routeData) {
          const custom404 = getCustom404Route(pipeline.manifestData);
          if (custom404 && !custom404.prerender) {
            this.routeData = custom404;
          }
        }
        if (!this.routeData) {
          pipeline.logger.debug("router", "Astro hasn't found routes that match " + this.request.url);
          pipeline.logger.debug("router", "Here's the available routes:\n", pipeline.manifestData);
          return;
        }
        __privateMethod(this, _FetchState_instances, stripHtmlExtension_fn).call(this);
      };
      /**
       * Strips the pipeline's base from the request URL, prepends a forward
       * slash, and decodes the pathname. Falls back to the raw (not decoded)
       * pathname if `decodeURI` throws.
       *
       * Mirrors `BaseApp.removeBase`, including the
       * `collapseDuplicateLeadingSlashes` fix that prevents middleware
       * authorization bypass when the URL starts with `//`.
       */
      computePathname_fn = function(url) {
        let pathname = collapseDuplicateLeadingSlashes(url.pathname);
        const base = this.pipeline.manifest.base;
        if (pathname.startsWith(base)) {
          const baseWithoutTrailingSlash = removeTrailingForwardSlash(base);
          pathname = pathname.slice(baseWithoutTrailingSlash.length + 1);
        }
        pathname = prependForwardSlash(pathname);
        try {
          return validateAndDecodePathname(pathname);
        } catch (e2) {
          if (e2 instanceof MultiLevelEncodingError) {
            this.invalidEncoding = true;
            return pathname;
          }
          this.pipeline.logger.error(null, e2.toString());
          return pathname;
        }
      };
      /**
       * Reads X-Forwarded-Proto, X-Forwarded-Host, and X-Forwarded-Port
       * from the request headers, validates them against the manifest's
       * `allowedDomains`, and updates `this.url` accordingly. Also resolves
       * `clientAddress` from X-Forwarded-For when the host is trusted.
       *
       * Only called when `allowedDomains` is configured — without it,
       * forwarded headers are never trusted.
       */
      applyForwardedHeaders_fn = function() {
        var _a29, _b13, _c, _d;
        const headers = this.request.headers;
        const allowedDomains = this.pipeline.manifest.allowedDomains;
        const validated = validateForwardedHeaders(
          getFirstForwardedValue$1((_a29 = headers.get("x-forwarded-proto")) != null ? _a29 : void 0),
          getFirstForwardedValue$1((_b13 = headers.get("x-forwarded-host")) != null ? _b13 : void 0),
          getFirstForwardedValue$1((_c = headers.get("x-forwarded-port")) != null ? _c : void 0),
          allowedDomains
        );
        if (!validated.protocol && !validated.host && !validated.port) return;
        if (validated.protocol) {
          this.url.protocol = validated.protocol + ":";
        }
        if (validated.host) {
          const colonIdx = validated.host.indexOf(":");
          if (colonIdx !== -1) {
            this.url.hostname = validated.host.slice(0, colonIdx);
            this.url.port = validated.host.slice(colonIdx + 1);
          } else {
            this.url.hostname = validated.host;
            this.url.port = "";
          }
        }
        if (validated.port) {
          this.url.port = validated.port;
        }
        const hostTrusted = validated.host !== void 0;
        if (hostTrusted && !this.clientAddress) {
          const forwardedFor = getFirstForwardedValue$1(
            (_d = this.request.headers.get("x-forwarded-for")) != null ? _d : void 0
          );
          if (forwardedFor) {
            this.clientAddress = forwardedFor;
          }
        }
        const oldRequest = this.request;
        this.request = new Request(this.url, oldRequest);
        const app2 = Reflect.get(oldRequest, appSymbol);
        if (app2 !== void 0) {
          Reflect.set(this.request, appSymbol, app2);
        }
      };
      ActionHandler = class {
        constructor() {
          __privateAdd(this, _ActionHandler_instances);
        }
        /**
         * Run action handling for the current request. Expects the APIContext
         * that is already being used by the render pipeline.
         *
         * Returns a `Response` when the action fully handles the request (RPC),
         * or `undefined` when the caller should continue processing the
         * request (form actions or non-action requests).
         */
        handle(apiContext, state) {
          state.pipeline.usedFeatures |= PipelineFeatures.actions;
          if (apiContext.isPrerendered) {
            return void 0;
          }
          const { action, setActionResult } = getActionContext(apiContext);
          if (!action) {
            return void 0;
          }
          return __privateMethod(this, _ActionHandler_instances, executeAction_fn).call(this, action, setActionResult);
        }
      };
      _ActionHandler_instances = new WeakSet();
      executeAction_fn = function(action, setActionResult) {
        return __async(this, null, function* () {
          const actionResult = yield action.handler();
          const serialized = serializeActionResult(actionResult);
          if (action.calledFrom === "rpc") {
            if (serialized.type === "empty") {
              return new Response(null, {
                status: serialized.status
              });
            }
            return new Response(serialized.body, {
              status: serialized.status,
              headers: {
                "Content-Type": serialized.contentType
              }
            });
          }
          setActionResult(action.name, serialized);
          return void 0;
        });
      };
      TrailingSlashHandler = class {
        constructor(app2) {
          __privateAdd(this, _TrailingSlashHandler_instances);
          __privateAdd(this, _app);
          __privateSet(this, _app, app2);
        }
        /**
         * Returns a redirect `Response` if the request pathname needs
         * normalization, or `undefined` if no redirect is required.
         */
        handle(state) {
          const url = new URL(state.request.url);
          const redirect = __privateMethod(this, _TrailingSlashHandler_instances, redirectTrailingSlash_fn).call(this, url.pathname);
          if (redirect === url.pathname) {
            return void 0;
          }
          const addCookieHeader = state.renderOptions.addCookieHeader;
          const status = state.request.method === "GET" ? 301 : 308;
          const response = new Response(
            redirectTemplate({
              status,
              relativeLocation: url.pathname,
              absoluteLocation: redirect,
              from: state.request.url
            }),
            {
              status,
              headers: {
                location: redirect + url.search
              }
            }
          );
          prepareResponse(response, { addCookieHeader });
          return response;
        }
      };
      _app = new WeakMap();
      _TrailingSlashHandler_instances = new WeakSet();
      redirectTrailingSlash_fn = function(pathname) {
        const { trailingSlash } = __privateGet(this, _app).manifest;
        if (pathname === "/" || isInternalPath(pathname)) {
          return pathname;
        }
        const path = collapseDuplicateTrailingSlashes(pathname, trailingSlash !== "never");
        if (path !== pathname) {
          return path;
        }
        if (trailingSlash === "ignore") {
          return pathname;
        }
        if (trailingSlash === "always" && !hasFileExtension(pathname)) {
          return appendForwardSlash(pathname);
        }
        if (trailingSlash === "never") {
          return removeTrailingForwardSlash(pathname);
        }
        return pathname;
      };
      APPLY_HEADERS = /* @__PURE__ */ Symbol.for("astro:cache:apply");
      IS_ACTIVE = /* @__PURE__ */ Symbol.for("astro:cache:active");
      AstroCache = class {
        constructor(provider) {
          __privateAdd(this, _options, {});
          __privateAdd(this, _tags, /* @__PURE__ */ new Set());
          __privateAdd(this, _disabled, false);
          __privateAdd(this, _provider);
          __publicField(this, "enabled", true);
          __privateSet(this, _provider, provider);
        }
        set(input) {
          if (input === false) {
            __privateSet(this, _disabled, true);
            __privateGet(this, _tags).clear();
            __privateSet(this, _options, {});
            return;
          }
          __privateSet(this, _disabled, false);
          let options;
          if (isLiveDataEntry(input)) {
            if (!input.cacheHint) return;
            options = input.cacheHint;
          } else {
            options = input;
          }
          if ("maxAge" in options && options.maxAge !== void 0) __privateGet(this, _options).maxAge = options.maxAge;
          if ("swr" in options && options.swr !== void 0)
            __privateGet(this, _options).swr = options.swr;
          if ("etag" in options && options.etag !== void 0)
            __privateGet(this, _options).etag = options.etag;
          if (options.lastModified !== void 0) {
            if (!__privateGet(this, _options).lastModified || options.lastModified > __privateGet(this, _options).lastModified) {
              __privateGet(this, _options).lastModified = options.lastModified;
            }
          }
          if (options.tags) {
            for (const tag of options.tags) __privateGet(this, _tags).add(tag);
          }
        }
        get tags() {
          return [...__privateGet(this, _tags)];
        }
        /**
         * Get the current cache options (read-only snapshot).
         * Includes all accumulated options: maxAge, swr, tags, etag, lastModified.
         */
        get options() {
          return __spreadProps(__spreadValues({}, __privateGet(this, _options)), {
            tags: this.tags
          });
        }
        invalidate(input) {
          return __async(this, null, function* () {
            var _a29, _b13;
            if (!__privateGet(this, _provider)) {
              throw new AstroError(CacheNotEnabled);
            }
            let options;
            if (isLiveDataEntry(input)) {
              options = { tags: (_b13 = (_a29 = input.cacheHint) == null ? void 0 : _a29.tags) != null ? _b13 : [] };
            } else {
              options = input;
            }
            return __privateGet(this, _provider).invalidate(options);
          });
        }
        /** @internal */
        [APPLY_HEADERS](response) {
          var _a29, _b13, _c, _d;
          if (__privateGet(this, _disabled)) return;
          const finalOptions = __spreadProps(__spreadValues({}, __privateGet(this, _options)), { tags: this.tags });
          if (finalOptions.maxAge === void 0 && !((_a29 = finalOptions.tags) == null ? void 0 : _a29.length)) return;
          const headers = (_d = (_c = (_b13 = __privateGet(this, _provider)) == null ? void 0 : _b13.setHeaders) == null ? void 0 : _c.call(_b13, finalOptions)) != null ? _d : defaultSetHeaders(finalOptions);
          for (const [key, value] of headers) {
            response.headers.set(key, value);
          }
        }
        /** @internal */
        get [IS_ACTIVE]() {
          return !__privateGet(this, _disabled) && (__privateGet(this, _options).maxAge !== void 0 || __privateGet(this, _tags).size > 0);
        }
      };
      _options = new WeakMap();
      _tags = new WeakMap();
      _disabled = new WeakMap();
      _provider = new WeakMap();
      ROUTE_DYNAMIC_SPLIT = /\[(.+?\(.+?\)|.+?)\]/;
      ROUTE_SPREAD = /^\.{3}.+$/;
      CACHE_KEY = "cache";
      CacheHandler = class {
        constructor(app2) {
          __privateAdd(this, _app2);
          __privateSet(this, _app2, app2);
        }
        handle(state, next) {
          return __async(this, null, function* () {
            __privateGet(this, _app2).pipeline.usedFeatures |= PipelineFeatures.cache;
            if (!__privateGet(this, _app2).pipeline.cacheProvider) {
              return next();
            }
            const cache = state.resolve(CACHE_KEY);
            const cacheProvider = yield __privateGet(this, _app2).pipeline.getCacheProvider();
            if (cacheProvider == null ? void 0 : cacheProvider.onRequest) {
              const response2 = yield cacheProvider.onRequest(
                {
                  request: state.request,
                  url: new URL(state.request.url),
                  waitUntil: state.renderOptions.waitUntil
                },
                () => __async(this, null, function* () {
                  const res = yield next();
                  applyCacheHeaders(cache, res);
                  return res;
                })
              );
              response2.headers.delete("CDN-Cache-Control");
              response2.headers.delete("Cache-Tag");
              return response2;
            }
            const response = yield next();
            applyCacheHeaders(cache, response);
            return response;
          });
        }
      };
      _app2 = new WeakMap();
      suspectProtoRx = /"(?:_|\\u0{2}5[Ff]){2}(?:p|\\u0{2}70)(?:r|\\u0{2}72)(?:o|\\u0{2}6[Ff])(?:t|\\u0{2}74)(?:o|\\u0{2}6[Ff])(?:_|\\u0{2}5[Ff]){2}"\s*:/;
      suspectConstructorRx = /"(?:c|\\u0063)(?:o|\\u006[Ff])(?:n|\\u006[Ee])(?:s|\\u0073)(?:t|\\u0074)(?:r|\\u0072)(?:u|\\u0075)(?:c|\\u0063)(?:t|\\u0074)(?:o|\\u006[Ff])(?:r|\\u0072)"\s*:/;
      JsonSigRx = /^\s*["[{]|^\s*-?\d{1,16}(\.\d{1,17})?([Ee][+-]?\d+)?\s*$/;
      BASE64_PREFIX = "base64:";
      DRIVER_NAME2 = "memory";
      memory = defineDriver2(() => {
        const data = /* @__PURE__ */ new Map();
        return {
          name: DRIVER_NAME2,
          getInstance: () => data,
          hasItem(key) {
            return data.has(key);
          },
          getItem(key) {
            var _a29;
            return (_a29 = data.get(key)) != null ? _a29 : null;
          },
          getItemRaw(key) {
            var _a29;
            return (_a29 = data.get(key)) != null ? _a29 : null;
          },
          setItem(key, value) {
            data.set(key, value);
          },
          setItemRaw(key, value) {
            data.set(key, value);
          },
          removeItem(key) {
            data.delete(key);
          },
          getKeys() {
            return [...data.keys()];
          },
          clear() {
            data.clear();
          },
          dispose() {
            data.clear();
          }
        };
      });
      PERSIST_SYMBOL = /* @__PURE__ */ Symbol();
      DEFAULT_COOKIE_NAME = "astro-session";
      VALID_COOKIE_REGEX = /^[\w-]+$/;
      unflatten = (parsed, _) => {
        return unflatten$1(parsed, {
          URL: (href) => new URL(href)
        });
      };
      stringify = (data, _) => {
        return stringify$2(data, {
          // Support URL objects
          URL: (val) => val instanceof URL && val.href
        });
      };
      _AstroSession = class _AstroSession {
        constructor({
          cookies,
          config: config2,
          runtimeMode,
          driverFactory,
          mockStorage
        }) {
          __privateAdd(this, _AstroSession_instances);
          // The cookies object.
          __privateAdd(this, _cookies);
          // The session configuration.
          __privateAdd(this, _config);
          // The cookie config
          __privateAdd(this, _cookieConfig);
          // The cookie name
          __privateAdd(this, _cookieName);
          // The unstorage object for the session driver.
          __privateAdd(this, _storage);
          __privateAdd(this, _data);
          // The session ID. A v4 UUID.
          __privateAdd(this, _sessionID);
          // Sessions to destroy. Needed because we won't have the old session ID after it's destroyed locally.
          __privateAdd(this, _toDestroy, /* @__PURE__ */ new Set());
          // Session keys to delete. Used for partial data sets to avoid overwriting the deleted value.
          __privateAdd(this, _toDelete, /* @__PURE__ */ new Set());
          // Whether the session is dirty and needs to be saved.
          __privateAdd(this, _dirty, false);
          // Whether the session cookie has been set.
          __privateAdd(this, _cookieSet, false);
          // Whether the session ID was sourced from a client cookie rather than freshly generated.
          __privateAdd(this, _sessionIDFromCookie, false);
          // The local data is "partial" if it has not been loaded from storage yet and only
          // contains values that have been set or deleted in-memory locally.
          // We do this to avoid the need to block on loading data when it is only being set.
          // When we load the data from storage, we need to merge it with the local partial data,
          // preserving in-memory changes and deletions.
          __privateAdd(this, _partial, true);
          // The driver factory function provided by the pipeline
          __privateAdd(this, _driverFactory);
          if (!config2) {
            throw new AstroError(__spreadProps(__spreadValues({}, SessionStorageInitError), {
              message: SessionStorageInitError.message(
                "No driver was defined in the session configuration and the adapter did not provide a default driver."
              )
            }));
          }
          __privateSet(this, _cookies, cookies);
          __privateSet(this, _driverFactory, driverFactory);
          const _a29 = config2, { cookie: cookieConfig = DEFAULT_COOKIE_NAME } = _a29, configRest = __objRest(_a29, ["cookie"]);
          let cookieConfigObject;
          if (typeof cookieConfig === "object") {
            const _b13 = cookieConfig, { name = DEFAULT_COOKIE_NAME } = _b13, rest = __objRest(_b13, ["name"]);
            __privateSet(this, _cookieName, name);
            cookieConfigObject = rest;
          } else {
            __privateSet(this, _cookieName, cookieConfig || DEFAULT_COOKIE_NAME);
          }
          __privateSet(this, _cookieConfig, __spreadProps(__spreadValues({
            sameSite: "lax",
            secure: runtimeMode === "production",
            path: "/"
          }, cookieConfigObject), {
            httpOnly: true
          }));
          __privateSet(this, _config, configRest);
          if (mockStorage) {
            __privateSet(this, _storage, mockStorage);
          }
        }
        /**
         * Gets a session value. Returns `undefined` if the session or value does not exist.
         */
        get(key) {
          return __async(this, null, function* () {
            var _a29;
            return (_a29 = (yield __privateMethod(this, _AstroSession_instances, ensureData_fn).call(this)).get(key)) == null ? void 0 : _a29.data;
          });
        }
        /**
         * Checks if a session value exists.
         */
        has(key) {
          return __async(this, null, function* () {
            return (yield __privateMethod(this, _AstroSession_instances, ensureData_fn).call(this)).has(key);
          });
        }
        /**
         * Gets all session values.
         */
        keys() {
          return __async(this, null, function* () {
            return (yield __privateMethod(this, _AstroSession_instances, ensureData_fn).call(this)).keys();
          });
        }
        /**
         * Gets all session values.
         */
        values() {
          return __async(this, null, function* () {
            return [...(yield __privateMethod(this, _AstroSession_instances, ensureData_fn).call(this)).values()].map((entry) => entry.data);
          });
        }
        /**
         * Gets all session entries.
         */
        entries() {
          return __async(this, null, function* () {
            return [...(yield __privateMethod(this, _AstroSession_instances, ensureData_fn).call(this)).entries()].map(([key, entry]) => [key, entry.data]);
          });
        }
        /**
         * Deletes a session value.
         */
        delete(key) {
          var _a29;
          (_a29 = __privateGet(this, _data)) != null ? _a29 : __privateSet(this, _data, /* @__PURE__ */ new Map());
          __privateGet(this, _data).delete(key);
          if (__privateGet(this, _partial)) {
            __privateGet(this, _toDelete).add(key);
          }
          __privateSet(this, _dirty, true);
        }
        /**
         * Sets a session value. The session is created if it does not exist.
         */
        set(key, value, { ttl } = {}) {
          var _a29;
          if (!key) {
            throw new AstroError(__spreadProps(__spreadValues({}, SessionStorageSaveError), {
              message: "The session key was not provided."
            }));
          }
          let cloned;
          try {
            cloned = unflatten(JSON.parse(stringify(value)));
          } catch (err) {
            throw new AstroError(
              __spreadProps(__spreadValues({}, SessionStorageSaveError), {
                message: `The session data for ${key} could not be serialized.`,
                hint: "See the devalue library for all supported types: https://github.com/rich-harris/devalue"
              }),
              { cause: err }
            );
          }
          if (!__privateGet(this, _cookieSet)) {
            __privateMethod(this, _AstroSession_instances, setCookie_fn).call(this);
            __privateSet(this, _cookieSet, true);
          }
          (_a29 = __privateGet(this, _data)) != null ? _a29 : __privateSet(this, _data, /* @__PURE__ */ new Map());
          const lifetime = ttl != null ? ttl : __privateGet(this, _config).ttl;
          const expires = typeof lifetime === "number" ? Date.now() + lifetime * 1e3 : lifetime;
          __privateGet(this, _data).set(key, {
            data: cloned,
            expires
          });
          __privateSet(this, _dirty, true);
        }
        /**
         * Destroys the session, clearing the cookie and storage if it exists.
         */
        destroy() {
          var _a29, _b13;
          const sessionId = (_b13 = __privateGet(this, _sessionID)) != null ? _b13 : (_a29 = __privateGet(this, _cookies).get(__privateGet(this, _cookieName))) == null ? void 0 : _a29.value;
          if (sessionId) {
            __privateGet(this, _toDestroy).add(sessionId);
          }
          __privateGet(this, _cookies).delete(__privateGet(this, _cookieName), __privateGet(this, _cookieConfig));
          __privateSet(this, _sessionID, void 0);
          __privateSet(this, _data, void 0);
          __privateSet(this, _dirty, true);
        }
        /**
         * Regenerates the session, creating a new session ID. The existing session data is preserved.
         */
        regenerate() {
          return __async(this, null, function* () {
            let data = /* @__PURE__ */ new Map();
            try {
              data = yield __privateMethod(this, _AstroSession_instances, ensureData_fn).call(this);
            } catch (err) {
              console.error("Failed to load session data during regeneration:", err);
            }
            const oldSessionId = __privateGet(this, _sessionID);
            __privateSet(this, _sessionID, crypto.randomUUID());
            __privateSet(this, _sessionIDFromCookie, false);
            __privateSet(this, _data, data);
            __privateSet(this, _dirty, true);
            yield __privateMethod(this, _AstroSession_instances, setCookie_fn).call(this);
            if (oldSessionId && __privateGet(this, _storage)) {
              __privateGet(this, _storage).removeItem(oldSessionId).catch((err) => {
                console.error("Failed to remove old session data:", err);
              });
            }
          });
        }
        // Persists the session data to storage.
        // This is called automatically at the end of the request.
        // Uses a symbol to prevent users from calling it directly.
        [PERSIST_SYMBOL]() {
          return __async(this, null, function* () {
            if (!__privateGet(this, _dirty) && !__privateGet(this, _toDestroy).size) {
              return;
            }
            const storage = yield __privateMethod(this, _AstroSession_instances, ensureStorage_fn).call(this);
            if (__privateGet(this, _dirty) && __privateGet(this, _data)) {
              const data = yield __privateMethod(this, _AstroSession_instances, ensureData_fn).call(this);
              __privateGet(this, _toDelete).forEach((key2) => data.delete(key2));
              const key = __privateMethod(this, _AstroSession_instances, ensureSessionID_fn).call(this);
              let serialized;
              try {
                serialized = stringify(data);
              } catch (err) {
                throw new AstroError(
                  __spreadProps(__spreadValues({}, SessionStorageSaveError), {
                    message: SessionStorageSaveError.message(
                      "The session data could not be serialized.",
                      __privateGet(this, _config).driver
                    )
                  }),
                  { cause: err }
                );
              }
              yield storage.setItem(key, serialized);
              __privateSet(this, _dirty, false);
            }
            if (__privateGet(this, _toDestroy).size > 0) {
              const cleanupPromises = [...__privateGet(this, _toDestroy)].map(
                (sessionId) => storage.removeItem(sessionId).catch((err) => {
                  console.error("Failed to clean up session %s:", sessionId, err);
                })
              );
              yield Promise.all(cleanupPromises);
              __privateGet(this, _toDestroy).clear();
            }
          });
        }
        get sessionID() {
          return __privateGet(this, _sessionID);
        }
        /**
         * Loads a session from storage with the given ID, and replaces the current session.
         * Any changes made to the current session will be lost.
         * This is not normally needed, as the session is automatically loaded using the cookie.
         * However it can be used to restore a session where the ID has been recorded somewhere
         * else (e.g. in a database).
         */
        load(sessionID) {
          return __async(this, null, function* () {
            __privateSet(this, _sessionID, sessionID);
            __privateSet(this, _data, void 0);
            yield __privateMethod(this, _AstroSession_instances, setCookie_fn).call(this);
            yield __privateMethod(this, _AstroSession_instances, ensureData_fn).call(this);
          });
        }
      };
      _cookies = new WeakMap();
      _config = new WeakMap();
      _cookieConfig = new WeakMap();
      _cookieName = new WeakMap();
      _storage = new WeakMap();
      _data = new WeakMap();
      _sessionID = new WeakMap();
      _toDestroy = new WeakMap();
      _toDelete = new WeakMap();
      _dirty = new WeakMap();
      _cookieSet = new WeakMap();
      _sessionIDFromCookie = new WeakMap();
      _partial = new WeakMap();
      _driverFactory = new WeakMap();
      _sharedStorage = new WeakMap();
      _AstroSession_instances = new WeakSet();
      setCookie_fn = function() {
        return __async(this, null, function* () {
          if (!VALID_COOKIE_REGEX.test(__privateGet(this, _cookieName))) {
            throw new AstroError(__spreadProps(__spreadValues({}, SessionStorageSaveError), {
              message: "Invalid cookie name. Cookie names can only contain letters, numbers, and dashes."
            }));
          }
          const value = __privateMethod(this, _AstroSession_instances, ensureSessionID_fn).call(this);
          __privateGet(this, _cookies).set(__privateGet(this, _cookieName), value, __privateGet(this, _cookieConfig));
        });
      };
      ensureData_fn = function() {
        return __async(this, null, function* () {
          var _a29, _b13;
          if (__privateGet(this, _data) && !__privateGet(this, _partial)) {
            return __privateGet(this, _data);
          }
          (_a29 = __privateGet(this, _data)) != null ? _a29 : __privateSet(this, _data, /* @__PURE__ */ new Map());
          if (!__privateGet(this, _sessionID) && !((_b13 = __privateGet(this, _cookies).get(__privateGet(this, _cookieName))) == null ? void 0 : _b13.value)) {
            __privateSet(this, _partial, false);
            return __privateGet(this, _data);
          }
          const storage = yield __privateMethod(this, _AstroSession_instances, ensureStorage_fn).call(this);
          const raw = yield storage.get(__privateMethod(this, _AstroSession_instances, ensureSessionID_fn).call(this));
          if (!raw) {
            if (__privateGet(this, _sessionIDFromCookie)) {
              __privateSet(this, _sessionID, crypto.randomUUID());
              __privateSet(this, _sessionIDFromCookie, false);
              if (__privateGet(this, _cookieSet)) {
                yield __privateMethod(this, _AstroSession_instances, setCookie_fn).call(this);
              }
            }
            return __privateGet(this, _data);
          }
          try {
            const storedMap = unflatten(raw);
            if (!(storedMap instanceof Map)) {
              yield this.destroy();
              throw new AstroError(__spreadProps(__spreadValues({}, SessionStorageInitError), {
                message: SessionStorageInitError.message(
                  "The session data was an invalid type.",
                  __privateGet(this, _config).driver
                )
              }));
            }
            const now = Date.now();
            for (const [key, value] of storedMap) {
              const expired = typeof value.expires === "number" && value.expires < now;
              if (!__privateGet(this, _data).has(key) && !__privateGet(this, _toDelete).has(key) && !expired) {
                __privateGet(this, _data).set(key, value);
              }
            }
            __privateSet(this, _partial, false);
            return __privateGet(this, _data);
          } catch (err) {
            yield this.destroy();
            if (err instanceof AstroError) {
              throw err;
            }
            throw new AstroError(
              __spreadProps(__spreadValues({}, SessionStorageInitError), {
                message: SessionStorageInitError.message(
                  "The session data could not be parsed.",
                  __privateGet(this, _config).driver
                )
              }),
              { cause: err }
            );
          }
        });
      };
      /**
       * Returns the session ID, generating a new one if it does not exist.
       */
      ensureSessionID_fn = function() {
        var _a29;
        if (!__privateGet(this, _sessionID)) {
          const cookieValue = (_a29 = __privateGet(this, _cookies).get(__privateGet(this, _cookieName))) == null ? void 0 : _a29.value;
          if (cookieValue) {
            __privateSet(this, _sessionID, cookieValue);
            __privateSet(this, _sessionIDFromCookie, true);
          } else {
            __privateSet(this, _sessionID, crypto.randomUUID());
          }
        }
        return __privateGet(this, _sessionID);
      };
      ensureStorage_fn = function() {
        return __async(this, null, function* () {
          if (__privateGet(this, _storage)) {
            return __privateGet(this, _storage);
          }
          if (__privateGet(_AstroSession, _sharedStorage).has(__privateGet(this, _config).driver)) {
            __privateSet(this, _storage, __privateGet(_AstroSession, _sharedStorage).get(__privateGet(this, _config).driver));
            return __privateGet(this, _storage);
          }
          if (!__privateGet(this, _driverFactory)) {
            throw new AstroError(__spreadProps(__spreadValues({}, SessionStorageInitError), {
              message: SessionStorageInitError.message(
                "Astro could not load the driver correctly. Does it exist?",
                __privateGet(this, _config).driver
              )
            }));
          }
          const driver = __privateGet(this, _driverFactory);
          try {
            __privateSet(this, _storage, createStorage({
              driver: __spreadProps(__spreadValues({}, driver(__privateGet(this, _config).options)), {
                // Unused methods
                hasItem() {
                  return false;
                },
                getKeys() {
                  return [];
                }
              })
            }));
            __privateGet(_AstroSession, _sharedStorage).set(__privateGet(this, _config).driver, __privateGet(this, _storage));
            return __privateGet(this, _storage);
          } catch (err) {
            throw new AstroError(
              __spreadProps(__spreadValues({}, SessionStorageInitError), {
                message: SessionStorageInitError.message("Unknown error", __privateGet(this, _config).driver)
              }),
              { cause: err }
            );
          }
        });
      };
      __privateAdd(_AstroSession, _sharedStorage, /* @__PURE__ */ new Map());
      AstroSession = _AstroSession;
      SESSION_KEY = "session";
      AstroHandler = class {
        constructor(app2) {
          __privateAdd(this, _AstroHandler_instances);
          __privateAdd(this, _app3);
          __privateAdd(this, _trailingSlashHandler);
          __privateAdd(this, _actionHandler);
          __privateAdd(this, _astroMiddleware);
          __privateAdd(this, _pagesHandler);
          __privateAdd(this, _cacheHandler);
          /** Bound callback for the middleware chain — created once, reused per request. */
          __privateAdd(this, _renderRouteCallback);
          /**
           * i18n post-processor. Only set when the app has i18n configured and
           * the strategy is not `manual` — for the manual strategy users wire
           * `astro:i18n.middleware(...)` into their own `onRequest`.
           */
          __privateAdd(this, _i18n2);
          /** Whether sessions are configured on the manifest. */
          __privateAdd(this, _hasSession);
          __privateSet(this, _app3, app2);
          __privateSet(this, _trailingSlashHandler, new TrailingSlashHandler(app2));
          __privateSet(this, _actionHandler, new ActionHandler());
          __privateSet(this, _astroMiddleware, new AstroMiddleware(app2.pipeline));
          __privateSet(this, _pagesHandler, new PagesHandler(app2.pipeline));
          __privateSet(this, _cacheHandler, new CacheHandler(app2));
          __privateSet(this, _renderRouteCallback, __privateMethod(this, _AstroHandler_instances, actionsAndPages_fn).bind(this));
          __privateSet(this, _hasSession, !!app2.manifest.sessionConfig);
          const i18n = app2.manifest.i18n;
          if (i18n && i18n.strategy !== "manual") {
            __privateSet(this, _i18n2, new I18n(
              i18n,
              app2.manifest.base,
              app2.manifest.trailingSlash,
              app2.manifest.buildFormat
            ));
          }
        }
        handle(state) {
          return __async(this, null, function* () {
            state.pipeline.usedFeatures |= ALL_PIPELINE_FEATURES;
            if (state.invalidEncoding) {
              return new Response(null, { status: 400, statusText: "Bad Request" });
            }
            const trailingSlashRedirect = __privateGet(this, _trailingSlashHandler).handle(state);
            if (trailingSlashRedirect) {
              return trailingSlashRedirect;
            }
            if (!state.routeData) {
              return __privateGet(this, _app3).renderError(state.request, __spreadProps(__spreadValues({}, state.renderOptions), {
                status: 404,
                pathname: state.pathname
              }));
            }
            return this.render(state);
          });
        }
        /**
         * Renders a response for the given `FetchState`. Assumes
         * trailing-slash redirects and routeData resolution have already run.
         *
         * User-triggered rewrites (`Astro.rewrite` / `ctx.rewrite`) go through
         * `Rewrites.execute` on the current `FetchState` — they mutate the
         * existing state in place and re-run middleware + page dispatch.
         */
        render(state) {
          return __async(this, null, function* () {
            const routeData = state.routeData;
            const pathname = state.pathname;
            const request = state.request;
            const { addCookieHeader } = state.renderOptions;
            const defaultStatus = __privateGet(this, _app3).getDefaultStatusCode(routeData, pathname);
            state.status = defaultStatus;
            let response;
            try {
              const sessionP = __privateGet(this, _hasSession) ? provideSession(state) : void 0;
              const cacheP = provideCache(state);
              if (sessionP || cacheP) yield Promise.all([sessionP, cacheP]);
              state.pipeline.usedFeatures |= PipelineFeatures.sessions;
              if (routeData.type === "redirect") {
                const redirectResponse = yield renderRedirect(state);
                __privateGet(this, _app3).logThisRequest({
                  pathname,
                  method: request.method,
                  statusCode: redirectResponse.status,
                  isRewrite: false,
                  timeStart: state.timeStart
                });
                prepareResponse(redirectResponse, { addCookieHeader });
                __privateGet(this, _app3).pipeline.logger.flush();
                return redirectResponse;
              }
              if (!__privateGet(this, _app3).pipeline.cacheProvider) {
                __privateGet(this, _app3).pipeline.usedFeatures |= PipelineFeatures.cache;
                response = yield __privateGet(this, _astroMiddleware).handle(state, __privateGet(this, _renderRouteCallback));
                if (__privateGet(this, _i18n2)) {
                  response = yield __privateGet(this, _i18n2).finalize(state, response);
                }
              } else {
                const runPipeline = () => __async(this, null, function* () {
                  let res = yield __privateGet(this, _astroMiddleware).handle(state, __privateGet(this, _renderRouteCallback));
                  if (__privateGet(this, _i18n2)) {
                    res = yield __privateGet(this, _i18n2).finalize(state, res);
                  }
                  return res;
                });
                response = yield __privateGet(this, _cacheHandler).handle(state, runPipeline);
              }
              const isRewrite = response.headers.has(REWRITE_DIRECTIVE_HEADER_KEY);
              __privateGet(this, _app3).logThisRequest({
                pathname,
                method: request.method,
                statusCode: response.status,
                isRewrite,
                timeStart: state.timeStart
              });
            } catch (err) {
              __privateGet(this, _app3).logger.error(null, err.stack || err.message || String(err));
              return __privateGet(this, _app3).renderError(request, __spreadProps(__spreadValues({}, state.renderOptions), {
                status: 500,
                error: err,
                pathname: state.pathname
              }));
            } finally {
              const finalize2 = state.finalizeAll();
              if (finalize2) yield finalize2;
            }
            if (REROUTABLE_STATUS_CODES.includes(response.status) && // If the body isn't null, that means the user sets the 404 status
            // but uses the current route to handle the 404
            response.body === null && response.headers.get(REROUTE_DIRECTIVE_HEADER) !== "no") {
              return __privateGet(this, _app3).renderError(request, __spreadProps(__spreadValues({}, state.renderOptions), {
                response,
                status: response.status,
                // We don't have an error to report here. Passing null means we pass nothing intentionally
                // while undefined means there's no error
                error: response.status === 500 ? null : void 0,
                pathname: state.pathname
              }));
            }
            prepareResponse(response, { addCookieHeader });
            __privateGet(this, _app3).pipeline.logger.flush();
            return response;
          });
        }
      };
      _app3 = new WeakMap();
      _trailingSlashHandler = new WeakMap();
      _actionHandler = new WeakMap();
      _astroMiddleware = new WeakMap();
      _pagesHandler = new WeakMap();
      _cacheHandler = new WeakMap();
      _renderRouteCallback = new WeakMap();
      _i18n2 = new WeakMap();
      _hasSession = new WeakMap();
      _AstroHandler_instances = new WeakSet();
      /**
       * Runs actions then pages — the callback at the bottom of the
       * middleware chain. Bound once in the constructor to avoid
       * per-request closure allocation.
       */
      actionsAndPages_fn = function(state, ctx) {
        if (!state.skipMiddleware) {
          const actionResult = __privateGet(this, _actionHandler).handle(ctx, state);
          if (actionResult) {
            return actionResult.then((response) => response != null ? response : __privateGet(this, _pagesHandler).handle(state, ctx));
          }
        }
        return __privateGet(this, _pagesHandler).handle(state, ctx);
      };
      DefaultFetchHandler = class {
        constructor(app2) {
          __privateAdd(this, _app4);
          __privateAdd(this, _handler);
          __publicField(this, "fetch", (request) => {
            if (!__privateGet(this, _app4)) {
              const app2 = Reflect.get(request, appSymbol);
              if (!app2) {
                throw new Error("No fetch handler provided.");
              }
              __privateSet(this, _app4, app2);
              __privateSet(this, _handler, new AstroHandler(app2));
            }
            const state = new FetchState(__privateGet(this, _app4).pipeline, request);
            if (!__privateGet(this, _handler)) {
              throw new Error("No fetch handler provided.");
            }
            return __privateGet(this, _handler).handle(state);
          });
          __privateSet(this, _app4, app2 != null ? app2 : null);
          __privateSet(this, _handler, app2 ? new AstroHandler(app2) : null);
        }
        /**
         * Fast path: called directly by `BaseApp.render()` with pre-resolved
         * options, avoiding the `Reflect.set/get` round-trip through the request.
         */
        renderWithOptions(request, options) {
          if (!__privateGet(this, _app4)) {
            const app2 = Reflect.get(request, appSymbol);
            if (!app2) {
              throw new Error("No fetch handler provided.");
            }
            __privateSet(this, _app4, app2);
            __privateSet(this, _handler, new AstroHandler(app2));
          }
          const state = new FetchState(__privateGet(this, _app4).pipeline, request, options);
          return __privateGet(this, _handler).handle(state);
        }
      };
      _app4 = new WeakMap();
      _handler = new WeakMap();
      fetchable = new DefaultFetchHandler();
      DefaultErrorHandler = class {
        constructor(app2) {
          __privateAdd(this, _app5);
          __privateAdd(this, _astroMiddleware2);
          __privateAdd(this, _pagesHandler2);
          __privateSet(this, _app5, app2);
          __privateSet(this, _astroMiddleware2, new AstroMiddleware(app2.pipeline));
          __privateSet(this, _pagesHandler2, new PagesHandler(app2.pipeline));
        }
        renderError(request, _a29) {
          return __async(this, null, function* () {
            var _b13 = _a29, {
              status,
              response: originalResponse,
              skipMiddleware = false,
              error: error2,
              pathname
            } = _b13, resolvedRenderOptions = __objRest(_b13, [
              "status",
              "response",
              "skipMiddleware",
              "error",
              "pathname"
            ]);
            var _a30;
            const app2 = __privateGet(this, _app5);
            const resolvedPathname = pathname != null ? pathname : new FetchState(app2.pipeline, request).pathname;
            const errorRoutePath = `/${status}${app2.manifest.trailingSlash === "always" ? "/" : ""}`;
            const errorRouteData = matchRoute(errorRoutePath, app2.manifestData);
            const url = new URL(request.url);
            if (errorRouteData) {
              if (errorRouteData.prerender) {
                const maybeDotHtml = errorRouteData.route.endsWith(`/${status}`) ? ".html" : "";
                const allowedDomains = app2.manifest.allowedDomains;
                const validatedHost = validateHost(url.host, url.protocol.replace(":", ""), allowedDomains);
                const safeOrigin = validatedHost ? url.origin : `${url.protocol}//localhost`;
                const statusURL = new URL(
                  `${app2.baseWithoutTrailingSlash}/${status}${maybeDotHtml}`,
                  safeOrigin
                );
                if (statusURL.toString() !== request.url && resolvedRenderOptions.prerenderedErrorPageFetch) {
                  try {
                    const response2 = yield resolvedRenderOptions.prerenderedErrorPageFetch(
                      statusURL.toString()
                    );
                    const override = { status, removeContentEncodingHeaders: true };
                    const newResponse = mergeResponses(response2, originalResponse, override);
                    prepareResponse(newResponse, resolvedRenderOptions);
                    return newResponse;
                  } catch (e2) {
                    const response2 = mergeResponses(new Response(null, { status }), originalResponse);
                    prepareResponse(response2, resolvedRenderOptions);
                    return response2;
                  }
                }
              }
              const mod = yield app2.pipeline.getComponentByRoute(errorRouteData);
              const errorState = new FetchState(app2.pipeline, request);
              errorState.skipMiddleware = skipMiddleware;
              errorState.clientAddress = resolvedRenderOptions.clientAddress;
              errorState.routeData = errorRouteData;
              errorState.pathname = resolvedPathname;
              errorState.status = status;
              errorState.componentInstance = mod;
              errorState.locals = (_a30 = resolvedRenderOptions.locals) != null ? _a30 : {};
              errorState.initialProps = { error: error2 };
              try {
                yield provideSession(errorState);
                const response2 = yield __privateGet(this, _astroMiddleware2).handle(
                  errorState,
                  __privateGet(this, _pagesHandler2).handle.bind(__privateGet(this, _pagesHandler2))
                );
                const newResponse = mergeResponses(response2, originalResponse);
                prepareResponse(newResponse, resolvedRenderOptions);
                return newResponse;
              } catch (e2) {
                if (skipMiddleware === false) {
                  return this.renderError(request, __spreadProps(__spreadValues({}, resolvedRenderOptions), {
                    status,
                    response: originalResponse,
                    skipMiddleware: true,
                    pathname: resolvedPathname
                  }));
                }
              } finally {
                yield errorState.finalizeAll();
              }
            }
            const response = mergeResponses(new Response(null, { status }), originalResponse);
            prepareResponse(response, resolvedRenderOptions);
            return response;
          });
        }
      };
      _app5 = new WeakMap();
      _astroMiddleware2 = new WeakMap();
      _pagesHandler2 = new WeakMap();
      _BaseApp = class _BaseApp {
        constructor(manifest2, streaming = true, ...args) {
          __privateAdd(this, _BaseApp_instances);
          __publicField(this, "manifest");
          __publicField(this, "manifestData");
          __publicField(this, "pipeline");
          __privateAdd(this, _adapterLogger);
          __publicField(this, "baseWithoutTrailingSlash");
          /**
           * The handler that turns incoming `Request` objects into `Response`s.
           * Defaults to a `DefaultFetchHandler` pinned to this app and can be
           * overridden via `setFetchHandler` — typically by the bundled
           * entrypoint after importing `virtual:astro:fetchable`.
           */
          __privateAdd(this, _fetchHandler);
          __privateAdd(this, _errorHandler);
          /**
           * Whether a custom fetch handler (from `src/app.ts`) has been set
           * via `setFetchHandler`. When false, the `DefaultFetchHandler` is
           * in use and all features are implicitly active.
           */
          __privateAdd(this, _hasCustomFetchHandler, false);
          /**
           * Whether the missing-feature check has already run. We only want
           * to warn once — after the first request in dev, or at build end.
           */
          __privateAdd(this, _featureCheckDone, false);
          this.manifest = manifest2;
          this.baseWithoutTrailingSlash = removeTrailingForwardSlash(manifest2.base);
          this.pipeline = this.createPipeline(streaming, manifest2, ...args);
          this.manifestData = this.pipeline.manifestData;
          __privateSet(this, _fetchHandler, new DefaultFetchHandler(this));
          __privateSet(this, _errorHandler, this.createErrorHandler());
        }
        get logger() {
          return this.pipeline.logger;
        }
        get adapterLogger() {
          const currentOptions = this.logger.options;
          if (!__privateGet(this, _adapterLogger) || __privateGet(this, _adapterLogger).options !== currentOptions) {
            __privateSet(this, _adapterLogger, new AstroIntegrationLogger(currentOptions, this.manifest.adapterName));
          }
          return __privateGet(this, _adapterLogger);
        }
        /**
         * Override the fetch handler used to dispatch requests. Entrypoints
         * call this with the default export of `virtual:astro:fetchable` to
         * plug in a user-authored handler from `src/app.ts`.
         */
        setFetchHandler(handler) {
          __privateSet(this, _fetchHandler, handler);
          __privateSet(this, _hasCustomFetchHandler, !(handler instanceof DefaultFetchHandler));
        }
        /**
         * Returns the error handler strategy used by this app. Override to
         * provide environment-specific behavior (dev overlay, build-time throws, etc.).
         */
        createErrorHandler() {
          return new DefaultErrorHandler(this);
        }
        /**
         * Resets the cached adapter logger so it picks up a new logger instance.
         * Used by BuildApp when the logger is replaced via setOptions().
         */
        resetAdapterLogger() {
          __privateSet(this, _adapterLogger, void 0);
        }
        getAllowedDomains() {
          return this.manifest.allowedDomains;
        }
        matchesAllowedDomains(forwardedHost, protocol) {
          return _BaseApp.validateForwardedHost(forwardedHost, this.manifest.allowedDomains, protocol);
        }
        static validateForwardedHost(forwardedHost, allowedDomains, protocol) {
          if (!allowedDomains || allowedDomains.length === 0) {
            return false;
          }
          try {
            const testUrl = new URL(`${protocol || "https"}://${forwardedHost}`);
            return allowedDomains.some((pattern) => {
              return matchPattern(testUrl, pattern);
            });
          } catch (e2) {
            return false;
          }
        }
        set setManifestData(newManifestData) {
          this.manifestData = newManifestData;
          this.pipeline.manifestData = newManifestData;
          this.pipeline.rebuildRouter();
        }
        removeBase(pathname) {
          pathname = collapseDuplicateLeadingSlashes(pathname);
          if (pathname.startsWith(this.manifest.base)) {
            return pathname.slice(this.baseWithoutTrailingSlash.length + 1);
          }
          return pathname;
        }
        /**
         * Decodes a pathname with `decodeURI`, falling back to the raw pathname when it
         * contains an invalid percent-sequence (e.g. `%C0%AF`, an overlong-UTF-8 encoding of
         * `/` commonly sent by path-traversal scanners). A raw `decodeURI()` would throw
         * `URIError: URI malformed`, and because `match()` runs before `render()` that error
         * escapes the adapter's request handler as an uncaught exception (HTTP 500) that user
         * middleware can't catch.
         */
        safeDecodeURI(pathname) {
          try {
            return decodeURI(pathname);
          } catch (e2) {
            this.adapterLogger.debug(e2.toString());
            return pathname;
          }
        }
        /**
         * Extracts the base-stripped, decoded pathname from a request.
         * Used by adapters to compute the pathname for dev-mode route matching.
         */
        getPathnameFromRequest(request) {
          const url = new URL(request.url);
          const pathname = prependForwardSlash(this.removeBase(url.pathname));
          return this.safeDecodeURI(pathname);
        }
        /**
         * Given a `Request`, it returns the `RouteData` that matches its `pathname`. By default, prerendered
         * routes aren't returned, even if they are matched.
         *
         * When `allowPrerenderedRoutes` is `true`, the function returns matched prerendered routes too.
         * @param request
         * @param allowPrerenderedRoutes
         */
        match(request, allowPrerenderedRoutes = false) {
          const url = new URL(request.url);
          if (this.manifest.assets.has(url.pathname)) return void 0;
          let pathname = this.computePathnameFromDomain(request);
          if (!pathname) {
            pathname = prependForwardSlash(this.removeBase(url.pathname));
          }
          const routeData = this.pipeline.matchRoute(this.safeDecodeURI(pathname));
          if (!routeData) return void 0;
          if (allowPrerenderedRoutes) {
            return routeData;
          }
          if (routeData.prerender) {
            if (routeData.params.length > 0) {
              const allMatches = this.pipeline.matchAllRoutes(this.safeDecodeURI(pathname));
              return allMatches.find((r2) => !r2.prerender);
            }
            return void 0;
          }
          return routeData;
        }
        /**
         * A matching route function to use in the development server.
         * Contrary to the `.match` function, this function resolves props and params, returning the correct
         * route based on the priority, segments. It also returns the correct, resolved pathname.
         * @param pathname
         */
        devMatch(pathname) {
          return void 0;
        }
        computePathnameFromDomain(request) {
          return computePathnameFromDomain(
            request,
            new URL(request.url),
            this.manifest.i18n,
            this.manifest.base,
            this.manifest.trailingSlash,
            this.logger
          );
        }
        render(_0) {
          return __async(this, arguments, function* (request, {
            addCookieHeader = false,
            clientAddress = Reflect.get(request, clientAddressSymbol),
            locals,
            prerenderedErrorPageFetch = fetch,
            routeData,
            waitUntil
          } = {}) {
            yield this.pipeline.getLogger();
            if (routeData) {
              this.logger.debug(
                "router",
                "The adapter " + this.manifest.adapterName + " provided a custom RouteData for ",
                request.url
              );
              this.logger.debug("router", "RouteData");
              this.logger.debug("router", routeData);
            }
            if (locals) {
              if (typeof locals !== "object") {
                const error2 = new AstroError(LocalsNotAnObject);
                this.logger.error(null, error2.stack);
                return this.renderError(request, {
                  addCookieHeader,
                  clientAddress,
                  prerenderedErrorPageFetch,
                  // If locals are invalid, we don't want to include them when
                  // rendering the error page
                  locals: void 0,
                  routeData,
                  waitUntil,
                  status: 500,
                  error: error2
                });
              }
            }
            if (!routeData) {
              const domainPathname = this.computePathnameFromDomain(request);
              if (domainPathname) {
                routeData = this.pipeline.matchRoute(this.safeDecodeURI(domainPathname));
              }
            }
            const resolvedOptions = {
              addCookieHeader,
              clientAddress,
              prerenderedErrorPageFetch,
              locals,
              routeData,
              waitUntil
            };
            let response;
            if (__privateGet(this, _fetchHandler) instanceof DefaultFetchHandler) {
              Reflect.set(request, appSymbol, this);
              response = yield __privateGet(this, _fetchHandler).renderWithOptions(request, resolvedOptions);
            } else {
              setRenderOptions(request, resolvedOptions);
              Reflect.set(request, appSymbol, this);
              response = yield __privateGet(this, _fetchHandler).fetch(request);
            }
            __privateMethod(this, _BaseApp_instances, warnMissingFeatures_fn).call(this);
            if (response.headers.get(ASTRO_ERROR_HEADER)) {
              response.headers.delete(ASTRO_ERROR_HEADER);
              return this.renderError(request, {
                addCookieHeader,
                clientAddress,
                prerenderedErrorPageFetch,
                locals,
                routeData,
                waitUntil,
                response,
                status: response.status,
                error: response.status === 500 ? null : void 0
              });
            }
            return response;
          });
        }
        setCookieHeaders(response) {
          return getSetCookiesFromResponse(response);
        }
        /**
         * If it is a known error code, try sending the according page (e.g. 404.astro / 500.astro).
         * This also handles pre-rendered /404 or /500 routes.
         *
         * Delegates to the app's configured `ErrorHandler`. To customize behavior
         * for a specific environment, override `createErrorHandler()` rather than
         * this method.
         */
        renderError(request, options) {
          return __async(this, null, function* () {
            return __privateGet(this, _errorHandler).renderError(request, options);
          });
        }
        getDefaultStatusCode(routeData, pathname) {
          if (!routeData.pattern.test(pathname)) {
            for (const fallbackRoute of routeData.fallbackRoutes) {
              if (fallbackRoute.pattern.test(pathname)) {
                return 302;
              }
            }
          }
          const route = removeTrailingForwardSlash(routeData.route);
          if (route.endsWith("/404")) return 404;
          if (route.endsWith("/500")) return 500;
          return 200;
        }
        getManifest() {
          return this.pipeline.manifest;
        }
        logThisRequest({
          pathname,
          method,
          statusCode,
          isRewrite,
          timeStart
        }) {
          const timeEnd = performance.now();
          this.logRequest({
            pathname,
            method,
            statusCode,
            isRewrite,
            reqTime: timeEnd - timeStart
          });
        }
      };
      _adapterLogger = new WeakMap();
      _fetchHandler = new WeakMap();
      _errorHandler = new WeakMap();
      _hasCustomFetchHandler = new WeakMap();
      _featureCheckDone = new WeakMap();
      _BaseApp_instances = new WeakSet();
      /**
       * One-shot check: after the first request with a custom `src/app.ts`,
       * compare `usedFeatures` against the manifest and warn about any
       * configured features the user's pipeline doesn't call.
       */
      warnMissingFeatures_fn = function() {
        if (__privateGet(this, _featureCheckDone) || !__privateGet(this, _hasCustomFetchHandler)) return;
        __privateSet(this, _featureCheckDone, true);
        const manifest2 = this.manifest;
        const missing = [];
        const used = this.pipeline.usedFeatures;
        if (manifest2.routes.some((r2) => r2.routeData.type === "redirect") && !(used & PipelineFeatures.redirects)) {
          missing.push("redirects");
        }
        if (manifest2.sessionConfig && !(used & PipelineFeatures.sessions)) {
          missing.push("sessions");
        }
        if (manifest2.actions && !(used & PipelineFeatures.actions)) {
          missing.push("actions");
        }
        if (manifest2.middleware && !(used & PipelineFeatures.middleware)) {
          missing.push("middleware");
        }
        if (manifest2.i18n && manifest2.i18n.strategy !== "manual" && !(used & PipelineFeatures.i18n)) {
          missing.push("i18n");
        }
        if (manifest2.cacheConfig && !(used & PipelineFeatures.cache)) {
          missing.push("cache");
        }
        for (const feature of missing) {
          this.logger.warn(
            "router",
            `Your project uses ${feature}, but your custom src/app.ts does not call the ${feature}() handler. This feature will not work unless you add it to your app.ts pipeline.`
          );
        }
      };
      /**
       * Reads all the cookies written by `Astro.cookie.set()` onto the passed response.
       * For example,
       * ```ts
       * for (const cookie_ of App.getSetCookieFromResponse(response)) {
       *     const cookie: string = cookie_
       * }
       * ```
       * @param response The response to read cookies from.
       * @returns An iterator that yields key-value pairs as equal-sign-separated strings.
       */
      __publicField(_BaseApp, "getSetCookieFromResponse", getSetCookiesFromResponse);
      BaseApp = _BaseApp;
      URL_PARSE_BASE = "https://astro.build";
      AppPipeline = class _AppPipeline extends Pipeline {
        getName() {
          return "AppPipeline";
        }
        static create({ manifest: manifest2, streaming }) {
          const resolve = function resolve2(specifier) {
            return __async(this, null, function* () {
              if (!(specifier in manifest2.entryModules)) {
                throw new Error(`Unable to resolve [${specifier}]`);
              }
              const bundlePath = manifest2.entryModules[specifier];
              if (bundlePath.startsWith("data:") || bundlePath.length === 0) {
                return bundlePath;
              } else {
                return createAssetLink(bundlePath, manifest2.base, manifest2.assetsPrefix);
              }
            });
          };
          const logger = createConsoleLogger({ level: manifest2.logLevel });
          const pipeline = new _AppPipeline(
            logger,
            manifest2,
            "production",
            manifest2.renderers,
            resolve,
            streaming,
            void 0,
            void 0,
            void 0,
            void 0,
            void 0,
            void 0,
            void 0,
            void 0
          );
          return pipeline;
        }
        headElements(routeData) {
          return __async(this, null, function* () {
            var _a29, _b13;
            const { assetsPrefix, base } = this.manifest;
            const routeInfo = this.manifest.routes.find(
              (route) => route.routeData.route === routeData.route
            );
            const links = /* @__PURE__ */ new Set();
            const scripts = /* @__PURE__ */ new Set();
            const styles = createStylesheetElementSet((_a29 = routeInfo == null ? void 0 : routeInfo.styles) != null ? _a29 : [], base, assetsPrefix);
            for (const script of (_b13 = routeInfo == null ? void 0 : routeInfo.scripts) != null ? _b13 : []) {
              if ("stage" in script) {
                if (script.stage === "head-inline") {
                  scripts.add({
                    props: {},
                    children: script.children
                  });
                }
              } else {
                scripts.add(createModuleScriptElement(script, base, assetsPrefix));
              }
            }
            return { links, styles, scripts };
          });
        }
        componentMetadata() {
        }
        getComponentByRoute(routeData) {
          return __async(this, null, function* () {
            const module = yield this.getModuleForRoute(routeData);
            return module.page();
          });
        }
        getModuleForRoute(route) {
          return __async(this, null, function* () {
            for (const defaultRoute of this.defaultRoutes) {
              if (route.component === defaultRoute.component) {
                return {
                  page: () => Promise.resolve(defaultRoute.instance)
                };
              }
            }
            let routeToProcess = route;
            if (routeIsRedirect(route)) {
              if (route.redirectRoute) {
                routeToProcess = route.redirectRoute;
              } else {
                return RedirectSinglePageBuiltModule;
              }
            } else if (routeIsFallback(route)) {
              routeToProcess = getFallbackRoute(route, this.manifest.routes);
            }
            if (this.manifest.pageMap) {
              const importComponentInstance = this.manifest.pageMap.get(routeToProcess.component);
              if (!importComponentInstance) {
                throw new Error(
                  `Unexpectedly unable to find a component instance for route ${route.route}`
                );
              }
              return yield importComponentInstance();
            } else if (this.manifest.pageModule) {
              return this.manifest.pageModule;
            }
            throw new Error(
              "Astro couldn't find the correct page to render, probably because it wasn't correctly mapped for SSR usage. This is an internal error, please file an issue."
            );
          });
        }
        tryRewrite(payload, request) {
          return __async(this, null, function* () {
            var _a29, _b13;
            const { newUrl, pathname, routeData } = findRouteToRewrite({
              payload,
              request,
              routes: (_a29 = this.manifest) == null ? void 0 : _a29.routes.map((r2) => r2.routeData),
              trailingSlash: this.manifest.trailingSlash,
              buildFormat: this.manifest.buildFormat,
              base: this.manifest.base,
              outDir: ((_b13 = this.manifest) == null ? void 0 : _b13.serverLike) ? this.manifest.buildClientDir : this.manifest.outDir
            });
            const componentInstance = yield this.getComponentByRoute(routeData);
            return { newUrl, pathname, componentInstance, routeData };
          });
        }
      };
      App = class extends BaseApp {
        createPipeline(streaming) {
          return AppPipeline.create({
            manifest: this.manifest,
            streaming
          });
        }
        isDev() {
          return false;
        }
        // Should we log something for our users?
        logRequest(_options2) {
        }
      };
      renderers = [];
      serializedData = [{ "file": "", "links": [], "scripts": [], "styles": [], "routeData": { "type": "page", "component": "_server-islands.astro", "params": ["name"], "segments": [[{ "content": "_server-islands", "dynamic": false, "spread": false }], [{ "content": "name", "dynamic": true, "spread": false }]], "pattern": "^\\/_server-islands\\/([^/]+?)\\/?$", "prerender": false, "isIndex": false, "fallbackRoutes": [], "route": "/_server-islands/[name]", "origin": "internal", "distURL": [], "_meta": { "trailingSlash": "ignore" } } }, { "file": "", "links": [], "scripts": [], "styles": [], "routeData": { "route": "/_image", "component": "node_modules/@astrojs/cloudflare/dist/entrypoints/image-transform-endpoint.js", "params": [], "pathname": "/_image", "pattern": "^\\/_image\\/?$", "segments": [[{ "content": "_image", "dynamic": false, "spread": false }]], "type": "endpoint", "prerender": false, "fallbackRoutes": [], "distURL": [], "isIndex": false, "origin": "internal", "_meta": { "trailingSlash": "ignore" } } }, { "file": "", "links": [], "scripts": [], "styles": [], "routeData": { "route": "/", "isIndex": true, "type": "page", "pattern": "^\\/$", "segments": [], "params": [], "component": "src/pages/index.astro", "pathname": "/", "prerender": false, "fallbackRoutes": [], "distURL": [], "origin": "project", "_meta": { "trailingSlash": "ignore" } } }];
      serializedData.map(deserializeRouteInfo);
      _page0 = () => Promise.resolve().then(() => (init_image_transform_endpoint_BgUicGG2(), image_transform_endpoint_BgUicGG2_exports)).then((n2) => n2.i);
      _page1 = () => Promise.resolve().then(() => (init_index_DHcMpgJR(), index_DHcMpgJR_exports));
      pageMap = /* @__PURE__ */ new Map([
        ["node_modules/@astrojs/cloudflare/dist/entrypoints/image-transform-endpoint.js", _page0],
        ["src/pages/index.astro", _page1]
      ]);
      _manifest = deserializeManifest({ "rootDir": "file:///home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/", "cacheDir": "file:///home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/node_modules/.astro/", "outDir": "file:///home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/dist/", "srcDir": "file:///home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/src/", "publicDir": "file:///home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/public/", "buildClientDir": "file:///home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/dist/client/", "buildServerDir": "file:///home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/dist/server/", "adapterName": "@astrojs/cloudflare", "assetsDir": "_astro", "routes": [{ "file": "", "links": [], "scripts": [], "styles": [], "routeData": { "type": "page", "component": "_server-islands.astro", "params": ["name"], "segments": [[{ "content": "_server-islands", "dynamic": false, "spread": false }], [{ "content": "name", "dynamic": true, "spread": false }]], "pattern": "^\\/_server-islands\\/([^/]+?)\\/?$", "prerender": false, "isIndex": false, "fallbackRoutes": [], "route": "/_server-islands/[name]", "origin": "internal", "distURL": [], "_meta": { "trailingSlash": "ignore" } } }, { "file": "", "links": [], "scripts": [], "styles": [], "routeData": { "route": "/_image", "component": "node_modules/@astrojs/cloudflare/dist/entrypoints/image-transform-endpoint.js", "params": [], "pathname": "/_image", "pattern": "^\\/_image\\/?$", "segments": [[{ "content": "_image", "dynamic": false, "spread": false }]], "type": "endpoint", "prerender": false, "fallbackRoutes": [], "distURL": [], "isIndex": false, "origin": "internal", "_meta": { "trailingSlash": "ignore" } } }, { "file": "", "links": [], "scripts": [], "styles": [{ "type": "inline", "content": ":root{color-scheme:light dark;font-family:system-ui,sans-serif;--accent: #f4a261;--ok: #2a9d8f;--card: color-mix(in srgb, CanvasText 6%, Canvas);--border: color-mix(in srgb, CanvasText 18%, transparent)}body{max-width:36rem;margin:3rem auto;padding:0 1rem 3rem;line-height:1.5}h1[data-astro-cid-j7pv25f6]{font-size:1.35rem;font-weight:650;margin-bottom:.25rem}p[data-astro-cid-j7pv25f6].lead{opacity:.85;margin-top:0}.card[data-astro-cid-j7pv25f6]{margin-top:1.5rem;background:var(--card);border:1px solid var(--border);border-radius:.75rem;padding:1.1rem 1.15rem}blockquote[data-astro-cid-j7pv25f6]{margin:0;padding:.85rem 1rem;border-left:4px solid var(--accent);background:color-mix(in srgb,var(--accent) 10%,transparent);border-radius:0 .5rem .5rem 0;font-size:1.1rem;line-height:1.45}.badge[data-astro-cid-j7pv25f6]{display:inline-block;margin-top:1rem;padding:.2rem .55rem;border-radius:999px;font-size:.75rem;font-weight:600;letter-spacing:.04em;text-transform:uppercase;background:color-mix(in srgb,var(--ok) 22%,transparent);color:var(--ok)}p[data-astro-cid-j7pv25f6].meta{opacity:.65;font-size:.85rem;margin-top:2rem}code[data-astro-cid-j7pv25f6]{font-size:.9em}\n" }], "routeData": { "route": "/", "isIndex": true, "type": "page", "pattern": "^\\/$", "segments": [], "params": [], "component": "src/pages/index.astro", "pathname": "/", "prerender": false, "fallbackRoutes": [], "distURL": [], "origin": "project", "_meta": { "trailingSlash": "ignore" } } }], "serverLike": true, "middlewareMode": "classic", "base": "/", "trailingSlash": "ignore", "compressHTML": true, "experimentalQueuedRendering": { "enabled": false, "poolSize": 0, "contentCache": false }, "componentMetadata": [["/home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/src/pages/index.astro", { "propagation": "none", "containsHead": true }]], "renderers": [], "clientDirectives": [["idle", '(()=>{var l=(n,t)=>{let i=async()=>{await(await n())()},e=typeof t.value=="object"?t.value:void 0,s={timeout:e==null?void 0:e.timeout};"requestIdleCallback"in window?window.requestIdleCallback(i,s):setTimeout(i,s.timeout||200)};(self.Astro||(self.Astro={})).idle=l;window.dispatchEvent(new Event("astro:idle"));})();'], ["load", '(()=>{var e=async t=>{await(await t())()};(self.Astro||(self.Astro={})).load=e;window.dispatchEvent(new Event("astro:load"));})();'], ["media", '(()=>{var n=(a,t)=>{let i=async()=>{await(await a())()};if(t.value){let e=matchMedia(t.value);e.matches?i():e.addEventListener("change",i,{once:!0})}};(self.Astro||(self.Astro={})).media=n;window.dispatchEvent(new Event("astro:media"));})();'], ["only", '(()=>{var e=async t=>{await(await t())()};(self.Astro||(self.Astro={})).only=e;window.dispatchEvent(new Event("astro:only"));})();'], ["visible", '(()=>{var a=(s,i,o)=>{let r=async()=>{await(await s())()},t=typeof i.value=="object"?i.value:void 0,c={rootMargin:t==null?void 0:t.rootMargin},n=new IntersectionObserver(e=>{for(let l of e)if(l.isIntersecting){n.disconnect(),r();break}},c);for(let e of o.children)n.observe(e)};(self.Astro||(self.Astro={})).visible=a;window.dispatchEvent(new Event("astro:visible"));})();']], "entryModules": { "virtual:cloudflare/worker-entry": "entry.mjs", "\0virtual:astro:actions/noop-entrypoint": "chunks/noop-entrypoint_DsBX4kaI.mjs", "\0noop-middleware": "virtual_astro_middleware.mjs", "\0virtual:astro:session-driver": "chunks/_virtual_astro_session-driver_Csu6PJY6.mjs", "\0virtual:astro:server-island-manifest": "chunks/_virtual_astro_server-island-manifest_CFl7y3Qj.mjs", "\0virtual:astro:page:src/pages/index@_@astro": "chunks/index_DHcMpgJR.mjs", "/home/lucasew/WORKSPACE/LEWTEC/eletrocromo/examples/astro/node_modules/@astrojs/cloudflare/dist/entrypoints/image-service-workerd.js": "chunks/image-service-workerd_C84WONpr.mjs", "virtual:astro:noop": "_astro/_virtual_astro_noop.BNC9bj4-.js", "astro:scripts/before-hydration.js": "" }, "inlinedScripts": [], "assets": [], "buildFormat": "directory", "checkOrigin": true, "actionBodySizeLimit": 1048576, "serverIslandBodySizeLimit": 1048576, "allowedDomains": [], "key": "3UXXasgdgvT9aW4DR0JjajhYy4Q3dcbFI8IYTcOdD30=", "sessionConfig": { "driver": "unstorage/drivers/cloudflare-kv-binding", "options": { "binding": "SESSION" } }, "image": {}, "devToolbar": { "enabled": false, "debugInfoOutput": "" }, "logLevel": "info", "shouldInjectCspMetaTags": false });
      manifestRoutes = _manifest.routes;
      manifest = Object.assign(_manifest, {
        renderers,
        actions: () => Promise.resolve().then(() => (init_noop_entrypoint_DsBX4kaI(), noop_entrypoint_DsBX4kaI_exports)),
        middleware: () => Promise.resolve().then(() => (init_virtual_astro_middleware(), virtual_astro_middleware_exports)),
        sessionDriver: () => Promise.resolve().then(() => (init_virtual_astro_session_driver_Csu6PJY6(), virtual_astro_session_driver_Csu6PJY6_exports)),
        serverIslandMappings: () => Promise.resolve().then(() => (init_virtual_astro_server_island_manifest_CFl7y3Qj(), virtual_astro_server_island_manifest_CFl7y3Qj_exports)),
        routes: manifestRoutes,
        pageMap
      });
      createApp$1 = ({ streaming } = {}) => {
        const app2 = new App(manifest, streaming);
        app2.setFetchHandler(fetchable);
        return app2;
      };
      createApp = createApp$1;
      IP_RE = /^[0-9a-fA-F.:]{1,45}$/;
      app = createApp();
      server_default = {
        fetch: handle
      };
      workerEntry = server_default != null ? server_default : {};
    }
  });

  // entry.mjs
  var entry_exports = {};
  __export(entry_exports, {
    default: () => workerEntry
  });
  init_cloudflare_workers();
  init_worker_entry_D8ZVWWpA();
  var _a27;
  (_a27 = globalThis.process) != null ? _a27 : globalThis.process = {};
  var _a28, _b12;
  (_b12 = (_a28 = globalThis.process).env) != null ? _b12 : _a28.env = {};
  return __toCommonJS(entry_exports);
})();

;if (typeof __orvalhoWorker !== 'undefined' && __orvalhoWorker) {
  globalThis.default = __orvalhoWorker.default !== undefined ? __orvalhoWorker.default : __orvalhoWorker;
}
