import * as wasm from "./filter_rust_bg.wasm";
import { __wbg_set_wasm } from "./filter_rust_bg.js";
__wbg_set_wasm(wasm);
export * from "./filter_rust_bg.js";
