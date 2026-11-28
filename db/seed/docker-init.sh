#!/bin/sh
# MySQL 容器首次初始化（数据目录为空）时执行的演示数据导入。
#
# 由 docker-compose.yml 挂载为 /docker-entrypoint-initdb.d/99_seed.sh：
#   1. 先按文件名顺序执行 01..11 的迁移；
#   2. 再执行本脚本，把 /seed/dev_demo.sql 灌进 MYSQL_DATABASE。
#
# 开关：SEED_ON_INIT=0 时跳过（生产/共享实例用）。默认 1 = 导入，
# 因为 compose 栈是 APP_ENV=dev 的本地演示环境，没有种子就登录不了。
#
# 注意：不要在这里用 exit/return —— 官方 entrypoint 可能 source 本文件，
# 退出语句会把初始化流程一起带走。
if [ "${SEED_ON_INIT:-1}" = "0" ]; then
  echo "[seed] SEED_ON_INIT=0，跳过演示数据导入"
elif [ ! -f /seed/dev_demo.sql ]; then
  echo "[seed] 未挂载 /seed/dev_demo.sql，跳过演示数据导入"
else
  echo "[seed] 导入 /seed/dev_demo.sql -> ${MYSQL_DATABASE}"
  mysql --default-character-set=utf8mb4 -uroot -p"${MYSQL_ROOT_PASSWORD}" "${MYSQL_DATABASE}" < /seed/dev_demo.sql
  echo "[seed] 完成：市民 13800000001 / 管理员 13800000002 可直接登录"
fi
