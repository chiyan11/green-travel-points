import fs from "node:fs";
import path from "node:path";
import { isMainThread } from "node:worker_threads";

// Vite 8 的 windowsSafeRealPathSync() 会在 Windows 上先调用
// fs.realpathSync.native(cwd)。如果这次调用成功，它就继续用
// child_process.exec("net use") 探测网络驱动器；受限沙箱禁止一切子进程，
// 这个探测会在加载 vitest/vite 配置之前抛 spawn EPERM。
//
// 这里只让“对当前工作目录的第一次 native realpath”抛出 Vite 已识别的
// EISDIR，让 Vite 走它自己的 fs.realpathSync 回退分支。回退后不再探测
// net use，也不影响之后任何 realpath 调用；没有修改 Vite 源码或依赖。
// 仅在 Windows 的 vite / vitest 主进程启用；Linux 上 Vite 直接使用
// fs.realpathSync.native，拦截会变成未捕获异常，因此必须跳过。
const isVite = process.argv.some((arg) =>
  /(?:^|[\\/])(?:vite|vitest)(?:\.mjs|\.js)?$/i.test(arg),
);
if (isMainThread && process.platform === "win32" && isVite) {
  const target = path.resolve(".");
  const original = fs.realpathSync.native;
  let armed = true;

  fs.realpathSync.native = function realpathSyncNative(p) {
    if (armed && typeof p === "string" && path.resolve(p) === target) {
      armed = false;
      const error = new Error("EISDIR: illegal operation on a directory, realpath");
      error.code = "EISDIR";
      throw error;
    }
    return original.apply(this, arguments);
  };
}
