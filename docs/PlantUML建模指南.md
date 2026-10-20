# PlantUML 建模指南（本项目的"怎么画图"手册）

> 面向对象：P1（建模负责人）及需要补图的全员。
> 配套源文件：`docs/uml/*.puml`（11 张图，开箱可渲染）。
> 先说一句：**"printUML" 应是 PlantUML 的口误/笔误**——业界没有 printUML 这个工具，而实验指导书 §二.1 明确点名接受 **PlantUML**。本指南按 PlantUML 写；若你指的是别的工具（StarUML / draw.io / ProcessOn），见 §7 的对照表。

---

## 1. 为什么这个项目该用 PlantUML

| 维度 | PlantUML | StarUML / Rose（图形化） | draw.io（图形化） |
|---|---|---|---|
| 图与代码一起进 Git | ✅ 是纯文本，`git diff` 能看出"改了哪个类名" | ❌ 二进制文件，无法 diff | ⚠️ 可存 XML，diff 不可读 |
| 改一处全图同步 | ✅ 改一行文本，重渲染 | ❌ 手动拖拽，容易漏改 | ❌ 同左 |
| 出图速度 | ✅ 命令行批量导出 11 张 | ❌ 一张张画 | ❌ 一张张画 |
| 图与文档同源 | ✅ Markdown 里直接嵌代码块 | ❌ 需截图粘贴 | ❌ 同左 |
| 一致性（类名=表名=代码） | ✅ 可脚本校验 | ❌ 靠人眼 | ❌ 靠人眼 |
| 答辩"能解释每处设计" | ✅ 图就是可读的声明式代码 | ⚠️ 图形里藏了很多"手画的"细节 | ⚠️ 同左 |

**对本项目的直接收益**：报告要求 8 类图（用例/活动/域模型/包图/部署图/鲁棒图/序列图/状态图），图形化工具画完改一轮就是灾难；PlantUML 把"画图"变成"写文本"，改需求时同步改图只需几分钟。

---

## 2. 环境准备（三条路，按你的情况选一条）

### 2.1 本机现状（已实测）

| 组件 | 状态 | 说明 |
|---|---|---|
| Java | ✅ 已装 `25.0.3` | PlantUML 是 Java 程序，有 JRE 就能跑 |
| Graphviz (`dot`) | ⚠️ 未做系统安装，但**便携版已就位** `docs/uml/.tools/graphviz/dot.exe`（2.44.1） | 由 PlantUML 自动下载后挪到固定位置，见 §2.2 第 5 步；`scripts/uml-export.ps1` 会自动探测它 |
| `plantuml.jar` | ✅ 已放 `docs/uml/.tools/plantuml.jar`（1.2024.8，22 MB） | 已被 `.gitignore` 忽略，换台机器要重下 |
| 11 张图 | ✅ **已实测**：语法校验全通过 + 渲染出 PNG | 一条命令：`scripts/uml-export.ps1`，见 §2.5 |

### 2.2 路线 A：VS Code 插件（推荐日常用）

**本仓库已配好**：`.vscode/settings.json`、`.vscode/extensions.json` 都已提交，
打开仓库时 VS Code 会提示安装 `jebbs.plantuml`，装完直接能预览。

1. 装扩展 **PlantUML**（发布者 `jebbs`）。
   推荐弹窗点安装；或命令行：`code --install-extension jebbs.plantuml`
2. 打开任意 `docs/uml/*.puml` → **`Alt + D`** 实时预览（改一行立刻重画）。
3. 导出：`Ctrl + Shift + P` → `PlantUML: Export Current Diagram`（或编辑器右上角图标），
   按本仓库配置会落到 `docs/uml/out/`。批量导出用 `scripts/uml-export.ps1`（§2.5）。
4. 关键设置（已写进 `.vscode/settings.json`，这里解释为什么这么配）：

   | 键 | 值 | 为什么 |
   |---|---|---|
   | `plantuml.render` | `Local` | 本地 jar 渲染，不依赖公网、设计稿不外传 |
   | `plantuml.commandArgs` | 见下 | 这几个是 **JVM 参数**，必须在 `-jar` 之前；该键正好插在 java 命令行上 |
   | `plantuml.diagramsRoot` | `docs/uml` | 批量导出的根目录 |
   | `plantuml.exportOutDir` | `out` | 相对根目录 → `docs/uml/out/` |
   | `plantuml.exportSubFolder` | `false` | 否则每张图会再套一层同名子文件夹 |

   ```jsonc
   "plantuml.commandArgs": [
     "-Dfile.encoding=UTF-8",        // 中文不乱码
     "-DPLANTUML_LIMIT_SIZE=20000",  // 默认画布上限 4096px，超了会被静默裁切
     "-DGRAPHVIZ_DOT=D:/coding/green-travel-points/docs/uml/.tools/graphviz/dot.exe"
   ]
   ```

   > 插件没有单独的 "graphvizDot" 设置项（2.18 版已移除），所以**只能**用 `-DGRAPHVIZ_DOT` 指定。

5. **Graphviz 怎么解决**（本机没装，且 `winget` 在此环境不可用）：
   - ✅ 已解决：便携版 `docs/uml/.tools/graphviz/dot.exe`（2.44.1，共 12 个文件）。
     它是 PlantUML 首次渲染时**自己下载**到 `<tmpdir>\_graphviz` 的，把整个文件夹挪到
     `docs/uml/.tools/graphviz/` 即可长期复用（`.tools/` 已被 `.gitignore` 忽略，不入库）。
   - 换机器想正常装：`winget install --id Graphviz.Graphviz -e`，`dot -V` 能出版本号后，
     把 `-DGRAPHVIZ_DOT` 那一项删掉，PlantUML 会自动走 PATH。
   - 换机器不想装：渲染任意一张**类图**（只有类图/用例图/包图强依赖 Graphviz），
     PlantUML 会自己下到 `<tmpdir>\_graphviz\`，复制过来即可。
   - 实在没有：加 `-Playout=smetana` 也能出图（纯 Java 布局），只是走线方正、不紧凑。

6. **验证与排错**：
   - 预览出图 = 通了。
   - 报 `Cannot find Graphviz` → 第 5 步的路径没对上；先在终端跑
     `docs\uml\.tools\graphviz\dot.exe -V`。
   - 中文变问号/方块 → `-Dfile.encoding=UTF-8` 没生效（用 `.vscode/settings.json` 就不会被拆坏）。
   - 图被切掉右侧/底部 → `-DPLANTUML_LIMIT_SIZE` 没加，或值仍偏小。

### 2.3 路线 B：在线渲染（零安装，赶报告时最快）

打开 <https://www.plantuml.com/plantuml/uml/>，把 `.puml` 内容粘进去即可预览、导出 PNG/SVG。
⚠️ 注意：**不要**把含真实手机号/内网地址的图粘到公网；本项目只有设计图，无此风险。

### 2.4 路线 C：命令行 jar（批量导出、进 CI）

```powershell
# 1) 下载 jar（放本地工具目录，不入库）
New-Item -ItemType Directory -Force docs\uml\.tools | Out-Null
Invoke-WebRequest "https://github.com/plantuml/plantuml/releases/latest/download/plantuml.jar" `
  -OutFile docs\uml\.tools\plantuml.jar

# 2) 单张验证（-charset UTF-8 必须加，否则中文乱码）
java -jar docs\uml\.tools\plantuml.jar -charset UTF-8 -tpng docs\uml\01-用例图-总览.puml
```

没有 Graphviz 时，用 PlantUML 内置的纯 Java 布局引擎兜底：

```powershell
java -jar docs\uml\.tools\plantuml.jar -charset UTF-8 -tpng -Playout=smetana docs\uml\*.puml
```

> `smetana` 布局略"方正"，但胜在不装依赖也能出图。装了 Graphviz 后去掉该参数即可。

### 2.5 批量导出脚本（已写好：`scripts/uml-export.ps1`）

脚本已经存在，直接用：

```powershell
.\scripts\uml-export.ps1                 # 导出 PNG 到 docs\uml\out\
.\scripts\uml-export.ps1 -Format svg     # 导出 SVG（Word 里更清晰）
.\scripts\uml-export.ps1 -CheckOnly      # 只校验语法，不出图——提交/答辩前跑这个
```

它做的事：找 jar（缺了自动下）→ 建临时目录 → 设 JVM 参数 → 逐个渲染 → 报告结果。
三条 JVM 参数不是可选项，缺一条就会踩下面的坑：

| 参数 | 不设的后果 |
|---|---|
| `-Dfile.encoding=UTF-8` | 中文变方块；且 PowerShell 会把它拆坏 → 必须走 `$env:JAVA_TOOL_OPTIONS` |
| `-Djava.io.tmpdir=<工作区>` | 导出报 `Can't create cache file!` 直接失败（系统临时目录不可写时） |
| `-DPLANTUML_LIMIT_SIZE=20000` | 默认画布上限 **4096px**，宽图右侧被**静默裁切**，不报错 |

> 手工敲命令时最容易漏的是第三条：图看着"少了一块"，其实是超出 4096px 被切掉了。

没有 Graphviz 时脚本自动加 `-Playout=smetana`（纯 Java 布局，样式略方正，但零依赖）。

**Git 策略**：`.puml` 永远入库；`docs/uml/out/` 与 `docs/uml/.tools/` 已加进 `.gitignore`，报告定稿时再把最终 PNG 一次性提交（避免每次改图都产生二进制 diff）。

---

## 3. 十分钟语法速通

### 3.1 骨架与"三件套"

```plantuml
@startuml 图名
' 单引号是注释，可以写设计说明，不会出现在图里
title 图 2-1 行程上报活动图
caption 数据来源：需求规格说明书 §2.3.2
skinparam defaultFontName "Microsoft YaHei"   ' ← 中文必加，否则方块字
skinparam dpi 150                             ' ← 报告插图清晰度

' ... 图的内容 ...

legend right
  图例说明写这里
endlegend
@enduml
```

### 3.2 五种最常用的关系

```plantuml
A --> B : 关联（实线箭头）
A -- B  : 关联（无箭头）
A ..> B : 依赖（虚线箭头，包图/类图用）
A <|-- B: 泛化/继承（空心三角）
A *-- B : 组合；A o-- B : 聚合
```

### 3.3 类图（域模型）

```plantuml
class "出行记录 TripRecord" as Trip <<实体类>> {
  + id: bigint
  - tripHash: varchar(64)
  + distanceKm: decimal(8,3)
  + status: TripStatus
  + carbonSavedGrams(): decimal
}
User "1" -- "0..*" Trip : 上报 >
Trip "1" -- "1" CarbonRecord : 产生 >
```

要点：
- `class "中文中文 English" as Alias`：中文标签 + 英文别名，后面引用只用别名，**改名只改一处**。
- `<<实体类>>` 是 stereotype，报告里用它能直接对应 CRC 卡片的分类。
- `-` 私有 / `+` 公有 / `#` protected / `~` 包内。
- `skinparam classAttributeIconSize 0` 关掉丑丑的小图标。

### 3.4 用例图

```plantuml
left to right direction
actor "市民" as Citizen
actor "定时任务" as Timer <<系统>>

rectangle "绿色出行积分平台" {
  usecase "上报行程" as UC02
  usecase "兑换商品" as UC08
}
Citizen --> UC02
UC02 ..> UC08 : <<include>>
```

- `left to right direction` 让人在左、系统在右，符合阅读习惯。
- 系统参与者用 `<<系统>>` 标注，答辩时能讲清"这不是人"。
- `<<include>>` 用 `..>`，`<<extend>>` 也用 `..>`，**箭头方向别搞反**：`基用例 ..> 被包含用例`（include）、`扩展用例 ..> 基用例`（extend）。

### 3.5 活动图（事件流）

```plantuml
start
:校验距离 ∈ [0.1, 100] km;
if (合法?) then (是)
  :计算 trip_hash;
else (否)
  :返回 400 E_DISTANCE_OUT_OF_RANGE;
  stop
endif
while (当日积分未达上限?) is (是)
  :发放积分;
endwhile (否)
stop
```

要点：`start` / `stop` / `if…then…else…endif` / `while…endwhile` / `fork…fork again…end fork`（并行）/ `|泳道名|`（泳道，写在动作前一行）。

### 3.6 序列图（用例实现）

```plantuml
autonumber
actor 市民 as U
participant "API 层" as API
participant "积分服务" as SVC
database "MySQL" as DB

U -> API : POST /trips
activate API
API -> SVC : EvaluateTrip(distance, mode)
SVC -> DB : SELECT ef
activate DB
DB --> SVC : ef_base, ef_actual
deactivate DB
alt 距离非法
  API --> U : 400 E_DISTANCE_OUT_OF_RANGE
else 合法
  SVC -> DB : 事务写入 trip/carbon/txn
  API --> U : 200 {points: 12}
end
deactivate API
```

- `autonumber` 自动编号，答辩时可以直接说"第 5 步加锁"。
- `activate/deactivate` 画生命线上的激活条，能一眼看出事务边界（报告里讲"事务边界"全靠它）。

### 3.7 状态图（生命周期）

```plantuml
[*] --> PENDING : 创建订单
PENDING --> PAID : 支付/确认
PENDING --> CANCELLED : 用户取消
PAID --> SHIPPED : 发货
SHIPPED --> COMPLETED : 核销完成
COMPLETED --> [*]
CANCELLED --> [*]
note right of CANCELLED : 触发原路回滚（BR-12）
```

### 3.8 组件图 / 部署图 / 包图

```plantuml
package "domain 层" as domain {
  [碳核算] as carbon
}
node "应用服务器 4C8G" {
  artifact "apps/api" as api
}
database "MySQL 8.0" as db
api ..> db : JDBC
```

PlantUML 里 `node` / `artifact` / `database` / `cloud` / `component` / `package` / `folder` 都可用，够画部署图。

### 3.9 鲁棒图（UML 扩展，需自己"约定"画法）

PlantUML 没有内置鲁棒图元素，用三个 stereotype 模拟（报告里要加一句"约定说明"，见 `docs/uml/07-鲁棒图-兑换.puml` 头注释）：

```plantuml
actor 市民 as U
boundary "兑换页面" as B      ' 边界类
control  "RedeemService" as C ' 控制类
entity   "RedeemOrder" as E   ' 实体类
U --> B
B --> C
C --> E
```

鲁棒图的**语法规则**（答辩会问）：边界类只连 actor 或控制类；控制类居中；实体类只连控制类。**禁止** actor 直连实体类。

---

## 4. 本项目要交付的 11 张图（与报告章节对照）

| # | 文件 | 图类型 | 报告位置 | 建模重点 |
|---|---|---|---|---|
| 01 | `01-用例图-总览.puml` | 用例图 | 一.2.3 | 6 个角色、19 个用例、3 个功能分区、include/extend 口径 |
| 02 | `02-活动图-行程上报.puml` | 活动图（泳道） | 一.2.3 | 距离校验、幂等、日上限三条分支必须画出来 |
| 03 | `03-活动图-兑换下单.puml` | 活动图（泳道） | 一.2.3 / 二.2.2.2 | 加锁→复核→扣减→回滚，体现"禁止先查后写" |
| 04 | `04-域模型.puml` | 类图（域模型） | 一.2.4 | 14 个实体、基数、CRC 卡片对应 |
| 05 | `05-包图-逻辑架构.puml` | 包图 | 二.2.1.2 | 分层 + 依赖方向（外层依赖内层，domain 不依赖框架） |
| 06 | `06-部署图.puml` | 部署图 | 二.2.1.3 | 浏览器 / API / MySQL / Worker / 反向代理 |
| 07 | `07-鲁棒图-兑换.puml` | 鲁棒图 | 二.2.2.1 | 边界-控制-实体三色分层 |
| 08 | `08-序列图-行程上报.puml` | 序列图 | 二.2.2.1 | 事务边界、幂等命中分支 |
| 09 | `09-序列图-兑换下单.puml` | 序列图 | 二.2.2.1 | 锁、Outbox、事务提交后异步 |
| 10 | `10-状态图-订单.puml` | 状态图 | 二.2.2.1 | 合法/非法跃迁，非法边要标红 |
| 11 | `11-状态图-积分.puml` | 状态图 | 二.2.2.1 | 可用/冻结/过期，先进先出扣减 |

> 实验一（需求）只需要 01–04；05–11 属于实验二（设计），但**现在就把骨架建起来**，Sprint 2 只填细节，比临时赶图省一半时间。

---

## 5. 建模方法论：从需求文字到图（真正要掌握的部分）

这一步比语法重要。按下面的顺序做，图必然与需求自洽：

### 第 1 步　抽角色（Actor）——问"谁按了按钮"

- 从需求文字里找**名词性的人/组织/外部系统**：市民、运营管理员、商户、风控专员、数据提供方、定时任务。
- 判定口径：**能主动发起用例**的才是 actor；只是被操作的对象（"商品"）不是。
- 定时任务也是 actor（时间触发者），别漏——UC-18 清算就靠它。

### 第 2 步　抽用例（Use Case）——动词 + 宾语

用 **"角色 + 动词 + 名词"** 造句，凡是"用户能用系统完成的一件有业务价值的事"就是候选用例：

> 市民**上报**行程 → UC-02；市民**兑换**商品 → UC-08；运营**配置**排放因子 → UC-14。

过滤三条：
1. **删内部步骤**："校验距离""计算哈希"是活动，不是用例。
2. **删纯查询的细枝**：能合并就合并（"看余额"和"看流水"合成 UC-06）。
3. **每个用例必须能被一个角色单独走完**，走不完就说明它该被 include 进别人。

### 第 3 步　画用例图——先分区，再连线

- 按**功能模块**分区（出行与核算 / 积分与商城 / 运营与治理），不要按角色分区（按角色分会让图变成"人-事"的放射状，读不出系统结构）。
- 先画 actor 与 usecase，**最后才加 include/extend**——先加关系会让你为了"图好看"硬凑依赖。

### 第 4 步　复杂用例画活动图——只画"有分支和并发"的

判定：**事件流里出现 if / 循环 / 事务回滚** 才画。本项目只有 `行程上报`（校验+幂等+上限）和 `兑换下单`（锁+回滚）够格，其余用"用例表"文字描述即可——**不要给每个用例都配活动图**，那是工作量陷阱。

### 第 5 步　域模型：名词短语法 → 类

1. 把需求段落里所有**名词短语**列成一张表（见需求文档 §2.4.1）。
2. 逐个判定：**实体类**（要持久化、有生命周期）/ **属性**（依附于实体）/ **不建**（同义词、背景词）。
3. 动词派生**控制类**（采集、核算、核发、兑换、清算）。
4. 用 **CRC 卡片**补职责与协作者：写不出"职责"的类，通常是属性而不是类。
5. 最后画类图，**先画实体和基数，再补方法**——方法在分析阶段不必全，设计阶段再补。

**本项目最关键的一次判定**：`积分` 拆成 `PointAccount`（余额快照）+ `PointTxn`（只增流水）。理由要能讲：单靠余额无法对账，单靠流水查询太慢；两者配合才同时满足 BR-07/BR-08。

### 第 6 步　一致性自检（提交前必做）

| 检查项 | 方法 |
|---|---|
| 用例 ↔ 接口 | 用例清单的每条 P0，都能在 `docs/api/openapi.yaml` 找到端点 |
| 类 ↔ 表 | 域模型的每个实体，都能在 `db/migrations` 找到同名表（`TripRecord` ↔ `trip_record`） |
| 规则 ↔ 代码 | BR-01~BR-15 每条都能在 `apps/api/internal/domain/` 找到实现或注释引用 |
| 图 ↔ 图 | 状态图里的状态名，与活动图/序列图里的状态名一字不差 |
| 术语统一 | "积分/碳积分/points" 全文只用一种；"兑换/核销/发货" 定义清楚不放混 |

建议把这些检查做成一页 `docs/uml/一致性自查表.md`，答辩时直接展示（能加分）。

---

## 6. 常见坑（踩过的，直接抄）

| 现象 | 原因 | 解决 |
|---|---|---|
| 中文显示成方块/乱码 | 未设字体或未加 `-charset UTF-8` | `skinparam defaultFontName "Microsoft YaHei"` + 命令行 `-charset UTF-8` |
| `Cannot find Graphviz` | 没装 dot | 装 Graphviz，或加 `-Playout=smetana` |
| 图被截断/太大 | 默认尺寸写死 | 去掉 `scale` 限制；报告里用 SVG 或 `-tpng` 加 `skinparam dpi 200` |
| 类图连线乱飞 | 类声明顺序乱 | 按"核心实体在前、字典表在后"排列；必要时 `A -[hidden]right- B` 强制布局 |
| 序列图看不出事务边界 | 没写 activate | `activate`/`deactivate` 包住事务区间 |
| 改了类名要改很多处 | 直接用了中文名做引用 | 中文名只写在 `as` 前，引用一律用别名 |
| 图片进 Word 后模糊 | PNG 分辨率低 | 导出 SVG（Word 2016+ 支持）或 `skinparam dpi 200` |
| Git 里图片天天变 | 每改图就提交 PNG | `.puml` 入库，PNG 只在定稿时提交一次 |
| **活动图**里 `if` 突然报 `Cannot find if` | 上一条活动语句尾部写了 `' 注释`（PlantUML 的活动图**不支持行尾注释**，会把后面的 `if` 吞掉） | 注释**单独占一行**（行首 `'`），或把说明写进语句文本里 |
| `note right of \|泳道\|` 报 `Syntax Error` | 活动图不支持给泳道挂 note | 用 `floating note right` 或干脆用 `legend right ... endlegend`（legend 画在图外，不会压住流程） |
| JVM 参数报 `找不到或无法加载主类 .encoding=UTF-8` | PowerShell 把 `-Dfile.encoding=UTF-8` 当成了两个参数 | 不要写在命令行里，改成 `$env:JAVA_TOOL_OPTIONS = "-Dfile.encoding=UTF-8"` |
| 导出报 `Can't create cache file!` | 系统临时目录不可写（企业机/沙箱常见） | 加 `-Djava.io.tmpdir=<工作区内目录>`，并预先创建该目录 |
| 图右上/右下内容凭空消失 | `PLANTUML_LIMIT_SIZE` 默认 4096px，超了**静默裁切** | 加 `-DPLANTUML_LIMIT_SIZE=20000` |
| 用例图 19 个用例排成 6500px 长条 | 默认从左到右堆 actor | `left to right direction`，并删掉压在系统框上的 note（细节移到活动图/序列图） |
| 想知道错在第几行 | `-checkonly` 只说"有错" | `Get-Content x.puml -Raw \| java -jar plantuml.jar -syntax` 会打印行号 + 原因 |

---

## 7. 如果老师要求"必须用图形化工具"

指导书原文：「建模工具：**StarUML、Rational Rose 或 PlantUML 等其他 UML 建模工具**」——PlantUML 是被点名的合规选项，可以直接用。

若遇到个别教师坚持图形化，用**降级方案**：

| 要求的图 | 做法 |
|---|---|
| 用例图 / 类图 / 状态图 | 用 PlantUML 出图 → 在 StarUML 里照着重绘（这两张最关键，值得重画） |
| 活动图 / 序列图 | 直接把 PlantUML 导出的 PNG 贴进 StarUML 的 Diagram 里当"背景图"，或截图粘贴——一般可接受 |
| 答辩 | 强调"图是代码化维护的，改需求能分钟级同步"，这是加分项而非减分项 |

---

## 8. 五分钟上手清单（今天就能做完）

1. （**可选**）`winget install Graphviz.Graphviz`，重开终端 `dot -V` 有版本号。不装也能出图（脚本走 `smetana`），只是布局没那么好看。
2. VS Code 装 PlantUML 扩展，打开 `docs/uml/01-用例图-总览.puml`，`Alt+D` 看预览。
3. 改一个字（比如把"市民"改成"碳积分用户"），看预览是否实时变——证明工具链通了。
4. 直接跑 `.\scripts\uml-export.ps1`（脚本已在仓库里，不用自己写），`docs\uml\out\` 出 11 张 PNG；提交前跑 `-CheckOnly`。
5. 挑 01（用例图）和 04（域模型）**逐条对着需求文档改一遍**——这两张是实验一的核心得分点。
