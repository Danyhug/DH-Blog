# Tailwind 渐进式迁移方案

> **状态：已完成（46/46 个文件）。** `.vue` 的 `<style>` 总量 6,288 → 3,379 行（−46%），
> 45 → 32 个文件仍带样式块，13 个文件的样式块彻底消失。收尾总结见 §8。
>
> 目标：把 `blog-front` 从「自定义 CSS/LESS 为主、Tailwind 为辅」逐步收敛为「一律 Tailwind + Element Plus」，
> 使样式与结构同处一地，降低人和 AI 的维护成本。
>
> **总原则：不搞一次性大迁移。一次只做一个文件（或一批小文件），每步可独立验证、可回退。**
> 本文档同时是**决策记录**：§4 的每一条都是实测踩出来的，遇到反直觉的情况先翻这里。

---

## 1. 现状盘点（2026-09-27 基线）

| 维度 | 数据 |
|---|---|
| Vue 文件总数 | 58 |
| 带 `<style>` 块的文件 | 45（20 个 LESS、25 个原生 CSS） |
| 自定义 CSS 总量 | ≈ 6,288 行 |
| 模板已用 Tailwind 工具类的文件 | 20 |
| 全局样式 | `src/assets/css/style.less` 318 行，含 53 个主题 CSS 变量 |
| Tailwind 配置 | `src/assets/css/tailwind.css`：`@import "tailwindcss"` + **P1 已落的 `@theme inline` 主题映射**（v4，CSS-first）✅ `5b81e6e` |

**项目已有 Tailwind 代码的风格约定**（迁移必须保持一致，见 `WebDriveView.vue`、`CodeBlock.vue` 等）：
内联任意值直接用十六进制，如 `bg-[#f0f5ff]`、`text-[#666]`、`rounded-[10px]`。
即：**不要**把 `#666` 近似成 `text-gray-600`，零视觉 diff 优先于"用起来更标准"。

自定义 CSS 的构成（决定迁移策略）：

| 构成 | 文件数 | 迁移方式 |
|---|---|---|
| `:deep()` 覆盖 Element Plus | 12 | **保留**，不迁（见 §4.3） |
| `@keyframes` 动画 | 9 | 视情况：装饰组件豁免；其余挪 `@theme` 变 `animate-*` |
| `var(--xx)` 主题变量 | 14 | P1 之前用 `bg-[var(--grey-0)]`；P1 之后可用语义类 |
| `@media` 响应式 | 8 | `sm:` / `max-md:` / `lg:` 前缀替代 |
| 伪类/伪元素 | 151 处 | 简单装饰可迁；复杂结构（CSS 艺术）豁免 |
| 文件内重复的组件类（`.btn-primary`、`.toolbar` 等） | 若干 | **保留为精简 scoped 块**（见 §4.5） |
| 硬编码颜色/渐变（不在主题变量内） | 若干 | 内联 arbitrary value（见 §4.6） |

### 永久豁免清单（不迁，任何人/AI 都不要动其样式）

| 文件 | CSS 行数 | 豁免理由 |
|---|---|---|
| `src/components/frontend/Pet.vue` | 950 | 纯 CSS 艺术（胖丁）：伪元素 + em 比例 + keyframes，Tailwind 重写是可读性灾难 |
| `src/components/frontend/Loading.vue` | 92 | 纯动画组件 |
| `src/assets/css/style.less` | 318 | 全局基底：主题变量（P1 会改造变量部分）、滚动条、通知、编辑器覆盖、全局媒体查询 |

### 进度总览（已完成）

- 计划迁移 42 个文件（实际动手 46 个）；豁免清单 3 项未动（`Pet.vue` 950 行、`Loading.vue` 92 行、`style.less` 318 行）
- 阶段：P0 后台 19 → P1 变量层 1 → P2 webdav 7 → P3 前台 19
- 结果：`.vue` 的 `<style>` 6,288 → **3,379** 行；带样式块的文件 45 → **32**；**13 个**文件的样式块彻底消失
- 完成标准（已达到）：除豁免清单与 §4 允许保留的部分外，所有 `<style>` 块已移除或仅剩无法消除的规则

---

## 2. 迁移规则（每个文件都照此执行）

### 2.1 六步流程

1. **基准**：迁移前该文件的 `<style>` 块就是对照基准（git 中可查，无需额外截图）。
2. **对照翻译**：把 `<style>` 里的规则逐条翻译到 template 的 class 上（对照表见 §3）。
3. **删样式**：template 翻译完成后删除 `<style>` 块（或仅剩 §4 允许保留的部分）。
4. **验证**：`bun run build:type-check` 必须通过（仅保证类型/构建，不代表视觉正确）。
5. **人工确认**：向用户汇报，由用户对运行中的页面做人工检查（响应式断点也要看）。
6. **提交**：**用户确认后**才 commit。一个文件（或一批小文件）一个 commit，
   信息格式 `refactor(style): migrate <组件名> to tailwind`。

> 汇报格式：`已完成 <文件>：删除 CSS N 行，构建通过，请人工看 <页面/状态>`。

### 2.2 翻译准则

- **忠于原样**：迁移不是重设计。尺寸/颜色/间距必须与原 CSS 一致，不改视觉。
- **风格与存量 Tailwind 一致**：内联任意 hex（`text-[#666]`），不近似成调色板刻度（见 §1）。
- **任意值兜底**：没有对应刻度的值用 arbitrary value，如 `min-height: 244px` → `min-h-[244px]`、
  `clip-path: polygon(...)` → `[clip-path:polygon(0_0,92%_0,100%_100%,0_100%)]`（**空格用 `_`**）。
- **响应式**：`@media (max-width: 1024px)` → `max-lg:`、`(max-width: 768px)` → `max-md:`，逐条核对 min/max 语义。
- **伪类**：`:hover` → `hover:`、`::before/::after` → `before:` / `after:`（配合 `content-['']`）。
- **嵌套 LESS**：展开后逐选择器翻译；子元素共用的样式直接写在子元素 class 上。
- **动态 class**：`:class` 绑定里的条件样式照常保留，只把静态部分翻译成工具类。
- **nth-child 处理**：模板里是**静态写死的兄弟元素**（如 TotalItem 的 4 个 `<li>`）→ 把差异样式
  直接写成各自的 class，不再用 nth-child；**`v-for` 动态列表** → 保留少量 CSS 或改用 `:class` 按 index 绑定。
- **先检查 `<style>` 是否带 `scoped`**：非 scoped 的块（如 `LoginView.vue`）样式全局泄漏，
  其选择器可能影响其他组件——翻译前确认每条选择器的作用范围，迁移后泄漏自然消除。
- 拿不准的地方**停下来问用户**，不要自由发挥。

### 2.3 完成定义（单文件）

- [ ] `<style>` 块已删除（或仅剩 §4 允许保留的规则，并注明原因）
- [ ] `bun run build:type-check` 通过
- [ ] 用户人工确认页面无视觉差异
- [ ] 用户确认后已单独 commit

---

## 3. 常用对照表

### 3.1 CSS → Tailwind 速查

| CSS | Tailwind |
|---|---|
| `display: flex / grid / block / none` | `flex` / `grid` / `block` / `hidden` |
| `flex-direction: column` | `flex-col` |
| `justify-content / align-items` | `justify-*` / `items-*` |
| `width/height: 100%` | `w-full` / `h-full` |
| `padding/margin: Npx` | `p-[Npx]` / `m-[Npx]`（有刻度用刻度，如 `p-4`） |
| `border-radius: 1rem` | `rounded-2xl`（16px）；其他值 `rounded-[Npx]` |
| `background: var(--grey-0)` | P1 前：`bg-[var(--grey-0)]`；P1 后：`bg-grey-0` |
| `color: var(--text-color)` | P1 前：`text-[var(--text-color)]`；P1 后：`text-grey-7` |
| `box-shadow: var(--dh-admin-border-shadow-normal)` | P1 前：`shadow-[var(--dh-admin-border-shadow-normal)]`；P1 后：`shadow-dh-normal` |
| `border-radius: var(--dh-admin-border-radius-big)` | `rounded-[var(--dh-admin-border-radius-big)]` |
| `background-image: linear-gradient(310deg, rgb(80,208,255), rgb(80,163,255))` | `bg-[linear-gradient(310deg,rgb(80,208,255),rgb(80,163,255))]` |
| `position: absolute/relative/fixed` | `absolute` / `relative` / `fixed` |
| `overflow: hidden` | `overflow-hidden` |
| `object-fit: cover` | `object-cover` |
| `transition: 0.5s ease-in-out` | `transition-all duration-500 ease-in-out` |
| `cursor: pointer` | `cursor-pointer` |
| `line-height: 42px` | `leading-[42px]` |
| `text-overflow: ellipsis` 三件套 | `truncate`（单行）/ `line-clamp-N`（多行） |
| `@media (max-width: 768px)` | `max-md:` 前缀 |
| `:hover { ... }` | `hover:` 前缀 |

### 3.2 主题变量 → 语义类（✅ P1 已落地，可直接使用）

在 `tailwind.css` 的 `@theme` 中声明，生成对应工具类：

| 原变量 | 生成的类示例 |
|---|---|
| `--grey-0 … --grey-9` | `bg-grey-0` / `text-grey-7` / `border-grey-4` … |
| `--grey-1-a7` 等透明色 | `bg-grey-1-a7` |
| `--color-red / --color-pink / --color-blue …` | `text-color-red` / `bg-color-blue` … |
| `--text-color`（别名） | 直接用 `text-grey-7`（别名不进 theme，避免重复） |
| `--primary-color` | `bg-primary` / `text-primary` |
| `--nav-bg`（渐变） | `bg-nav` |
| `--dh-admin-bg` | `bg-dh-admin` |
| `--dh-admin-border-radius-small/normal/big` | `rounded-dh-sm` / `rounded-dh` / `rounded-dh-lg` |
| `--dh-admin-border-shadow-shallow/normal` | `shadow-dh-shallow` / `shadow-dh-normal` |
| `--dh-admin-color-success/error/info` | `text-dh-success` / `bg-dh-error` / `text-dh-info` |

> **P1 已落地**（`5b81e6e`）：`tailwind.css` 用 `@theme inline` 映射，工具类直接内联 `var(--grey-0)`，
> 不产生平行的 `--color-*` 变量；`--nav-bg` 因为 v4 没有背景图命名空间，用 `@utility bg-nav` 实现。
> 因此 P2/P3 直接写 `bg-grey-0` / `text-color-red` / `rounded-dh-lg` / `shadow-dh-normal` / `bg-nav`，
> **不要再写 `bg-[var(--grey-0)]` 这类任意值**。
> **注意**：主题变量只覆盖全局色板；各组件里大量**硬编码颜色/渐变**（网关卡片渐变、仪表盘渐变、
> 登录页黑板等）不在此列，按 §4.6 用内联任意值处理，不进 `@theme`。

---

## 4. 特殊场景处理

### 4.1 `@keyframes` 动画

- 装饰组件（Pet、Loading）：**豁免**，不迁。
- 其余文件：把 keyframes 挪进 `tailwind.css` 的 `@theme`（`--animate-*` + `@keyframes`），
  组件里改用 `animate-<name>`；只在首次遇到时建一次，后续复用。
  - 实测（2d）：`@theme` 里声明的 keyframes 以**原名**输出，且只在对应的 `animate-*` 被用到时才生成，
    所以既不会与 scoped 改名冲突，也不会留下没人用的 keyframes。`animate-spin` 是内置的，直接用。
  - 判据：凡是「类里写了 `animation: xxx`、而 `@keyframes xxx` 也在这个 `<style>` 块里」的组合，
    都**不能**只把类内联掉——要么整组留在 scoped 块，要么把 keyframes 整组搬到 `tailwind.css`。

### 4.2 响应式

- 全局 `style.less` 里的布局媒体查询（`.inner`、`#banner`、`#nav` 等）**留在全局**，
  等 P3 迁到对应组件时再处理。
- 组件内的 `@media` 一律翻成 `sm:` / `max-md:` / `lg:` 等前缀，逐条核对 max/min 语义。

### 4.3 Element Plus 覆盖（`:deep()`）

**保留，不强行迁。** 理由：el 组件内部 DOM 由库控制，工具类加不上去，`:deep()` 是官方认可的穿透方式。

- 可做的最小收敛：scoped 块里**只留** `:deep()` 规则，组件自身的样式全部迁走；
  `:deep()` 块顶部加一行注释说明覆盖目标。
- 若某 `:deep()` 只是给 el 组件根节点加间距/宽度，优先改为在模板上直接给组件加 class
  （Element Plus 会把 class 透传到根节点），从而消掉这条 `:deep()`。

### 4.3.1 运行时生成的 HTML（`v-html`）

与 `:deep()` 同类：`marked` 渲染的 Markdown、`highlight.js` 的代码高亮等**运行时生成**的 DOM，
加不上 class，样式只能留在 CSS 里（`FilePreview.vue` 的 `.markdown-content` / `.code-content`）。
另外 `Element Plus` 的 Upload 根节点不带任何类名，`el-upload` 位于其内部 `upload-content` 的根节点，
因此 `PublishView.vue` 那段非 scoped 的 `.avatar-uploader .el-upload` 也必须保留。

### 4.4 全局 `style.less`

- P1 只动 `:root` 变量部分（见 §5 P1）。
- 其余（滚动条、`.dh-notification`、编辑器覆盖、`.inner`、媒体查询）属全局基底，**不在本方案迁移范围**，
  保持 LESS 原样。

### 4.5 文件内重复的组件类（`.btn-primary`、`.toolbar`、`.card` 等）

webdav、gateway 模块里存在**同一文件内被多个元素复用的语义类**（如 `.btn-primary` 出现在 5+ 个按钮上、
`.search-input`、`.toolbar`、`.card`、`.link`、`.hint`）。这类样式若逐元素翻译成工具类，
会在模板里产生大段重复的长 class 串，反而更难维护。

规则：

- **同一文件内复用 ≥ 3 次的组件类 → 保留**在精简后的 `<style scoped>` 块里，
  块顶部加注释：`/* 组件类：模板中重复出现的同款样式，集中维护 */`；
- 只出现一两次的样式照常迁成工具类；
- **断点密集的簇优先保留**（2d 新增判据）：即使模板里只出现一次，若该元素带 2–3 档 `@media`
  的一整套声明，内联会产出 300+ 字符、含 6+ 个 `[@media(max-width:768px)]:` 前缀的 class 串，
  比原 CSS 更难读。实测例子：`WebDriveView` 的 `.toolbar-left button`（7 个按钮共用一套 3 档断点）、
  `.file-icon-container`（8 种配色 × 2 元素 × 3 档尺寸）。判据是**字符密度**而不是复用次数。
- **跨文件重复**的模式（如多个 webdav 文件都有类似按钮）：**第一次遇到时停下来问用户**，
  由用户决定抽共享 CSS 还是各文件各自保留；
- 最终目标仍是 utility-first，不是 utility-only：保留的组件类是 sanctioned exception，不算遗留债。

### 4.6 硬编码颜色与渐变（不在主题变量内）

大量组件使用与全局变量无关的一次性颜色/渐变（网关卡片渐变、TotalItem 的 4 个图标渐变、
登录页黑色斜板等）：

- 一律内联 arbitrary value：`bg-[linear-gradient(310deg,rgb(80,208,255),rgb(80,163,255))]`、
  `text-[#78829d]`、`bg-[rgba(0,0,0,0.8)]`（空格用 `_`）；
- **不进** `@theme`：一次性值不值得命名；
- 静态兄弟元素的 nth-child 差异样式 → 逐项写 class（见 §2.2）。

### 4.7 非 scoped 的 `<style>` 块

`LoginView.vue` 等少数文件的 `<style lang="less">` **没有 scoped**，样式全局生效。
翻译前逐条确认选择器是否只命中本组件的 DOM；若有命中其他组件的规则，停下来问用户。
迁移为工具类后全局泄漏自然消除。

### 4.8 工具类压不过「无层级 CSS」（0b 实测发现，务必逐条验证）

Tailwind v4 的工具类输出在 `@layer utilities` 里，而 **Element Plus 的样式、`style.less`、
组件自己的 scoped 块都是无层级的**。按 CSS 层叠规则，无层级声明**永远压过**分层声明——
与特异性、加载顺序都无关。后果：当同一元素上已存在无层级声明且**属性冲突**时，
新加的工具类会**静默失效**（`bun run build:type-check` 也照样通过）。

实测（构建产物 + headless Chrome 读 `getComputedStyle`）：

| 场景 | 结果 |
|---|---|
| 0b `AdminView.vue`：`.el-aside` 自带 `overflow: auto`，加 `overflow-hidden` | ❌ 仍是 `auto`；改回 scoped `.el-aside { overflow: hidden }` 才生效 |
| 0c `SectionPanel.vue`：`<h3>` 加 `font-semibold` | ❌ 被全局 `h1, h2, h3 { font-weight: 400 }` 压成 `400`；改回 scoped `.panel-title { font-weight: 600 }` 才是 `600` |
| 0b 回归 `GatewayKeys.vue`：`.masked`（在 `<code>` 上）改成工具类 | ❌ 被父级 `GatewayView` 的 `:deep(code)` 压掉，文字色从 `#667085` 变 `#476582`；已改回 scoped `.masked` |

**最容易漏的一类：父组件的 `:deep(元素名)`。** 被 `:deep()` 命中的是**标签选择器**
（`code`、`h3`、`table` 等），特异性虽低但无层级，照样压过工具类。
迁移子组件时，凡是目标标签与父级 `:deep(元素名)` 重合的元素（本项目已知：`GatewayView` 的
`:deep(code)`，作用到 `GatewayKeys` / `GatewayMcp` / `GatewayProviders` 里所有 `<code>`），
其样式**只能用 scoped 类名表达**，全部内联成工具类会静默变色。
排查办法：迁移前先 `rg ":deep\(" 父组件` 看有没有元素名选择器。

**重点排查项**：`style.less` 里的元素级规则会命中组件内的同名标签——
`h1, h2, h3 { font-weight: 400 }`（任何标题上的字重工具类都失效）、
`a { color: currentColor }`（链接上的 `text-*` 可能失效）、`* { box-sizing: border-box }`、
`.icon { display: inline-block; vertical-align: middle; overflow: hidden }`。

**Element Plus 组件自身的类同样会挡工具类。** 命中列表（实测）：

| 元素 | 被压掉的属性 |
|---|---|
| `el-button`（含 `is-link`） | `display` / `height` / `padding` / `color` / `font-size` / `white-space` / `background-color` / `border` |
| `el-table` | `background-color` / `overflow` / `width` / `font-size` / `color` |
| `el-aside` | `width` / `overflow` / `flex-shrink` |
| `el-menu` | `background-color` / `border` / `padding-left` |
| `el-menu-item` | `margin` / `padding` / `height` / `color` / `border-bottom` |
| `el-icon` | `color` / `font-size` / `margin-right` |
| `el-select` | `width` |
| `el-radio-button__inner` | `padding`（还叠着 `.el-radio-button--small` 的高特异性） |

给这些元素的根节点加 `bg-*` / `shadow-*` / `transition-*` / `[--el-xxx:value]` 等多种情况下仍安全，
但**一旦与上表属性同名就必须保留成 scoped 类**。判断办法：先 `rg` 一遍 `vendor-*.css` 里该组件类的声明。

处理办法：

- **属性冲突**（同一元素上库/全局样式声明了同名属性）→ 该条**保留在精简 scoped 块里**，
  顶部注释说明是哪个库规则压住了它；
- **不冲突** → 工具类照常生效，放心用。典型不冲突场景：给 `el-*` 组件根节点加
  `bg-*` / `shadow-*` / `transition-*` / 任意属性 `[--el-xxx:value]`（这些属性库本身没声明）。
- **注意 `el-icon`**：Element Plus 的 `.el-icon` 自带 `color: var(--color)` 与 `font-size: inherit`，
  所以 `text-[#xxx]` / `text-[17px]` 加在 `<el-icon>` 上**无效**。改用组件自己的 props：
  `<el-icon :size="17" color="#3f8cff">`（写 inline style，压得住一切）。

### 4.9 同属性工具类之间也不能靠 class 顺序覆盖

同一元素上写两个声明了相同属性的工具类（如基础态 `border-[#e6e9ee]` + 选中态 `border-[#3f8cff]`、
`bg-white` + `bg-[#f5f9ff]`）时，谁生效取决于 **Tailwind 生成 CSS 的排序**，与模板里的
class 先后无关。实测（0c）：`.p-0` 排在 `.px-*` / `.pt-*` / `.pb-*` **之前**，
所以 `class="px-[20px] pt-[18px] pb-[20px] p-0"` 的计算值是 `18px 20px 20px`——`p-0` 被吃掉。

规则：

- **多状态二选一 → 整体切换**，写成两套完整 class 串，而不是「基础 + 覆盖」。
  例：`SectionPanel` 的 flush 用 `:class="flush ? 'p-0' : 'px-[20px] pt-[18px] pb-[20px]'"`；
  `GatewayRouting` 的三种按钮状态用 `optionClass(option)` 返回三份互斥的样式串。
- `:hover` / `:not(:disabled)` 这类**伪类**没问题：`hover:` 变体会生成 `:hover`，特异性更高，稳定赢过基础态。
- 拿不准时**编译一遍看生成顺序**，不要靠猜。
- **`border-[<color>]` / `border-x-*` 之类的「组」写法会一次写多条边**：
  实测把 `border-bottom: 1px solid var(--border-color)` 写成
  `border-b border-[var(--border-color)]` 后，元素的 `border-top-color` 从
  `currentColor`（继承自 `color`）变成了边框色 —— 因为 `border-<color>` 写的是
  **四条边**的颜色。宽度仍是 0 所以看不出，但语义已经错了。
  只有单边有边框时用 `border-b-[var(--border-color)]`（`border-<side>-<color>`）。

- **简写属性会重置同族长写**，这比「两个同属性工具类」更隐蔽：
  实测 `.play { transition: all .5s ease; transition-delay: 300ms }` 拆成
  `[transition:all_0.5s_ease] [transition-delay:300ms]` 后，计算出的 `transition-delay` 是
  **0s** —— `transition` 简写把 delay 重置了，谁生效取决于生成顺序。
  同类简写：`transition` / `animation` / `background` / `border` / `font` / `flex` / `grid`。
  遇到「简写 + 该简写覆盖的长写」的组合，**合并进一条**任意属性：
  `[transition:all_0.5s_ease_300ms]`。

### 4.9.1 `shadow-[...]` 的颜色歧义（实测）

`shadow-[<值>]` 在 v4 里会判断该值「像不像颜色」：像颜色就当成 **shadow color**，只写
`--tw-shadow-color`，`box-shadow` 本身不变（实测某处写成 `shadow-[rgba(151,65,252,0.2)_0_15px_30px_-5px]`
后计算值是 `none`）。

- 值以偏移量开头（`0_25px_50px_-12px_rgba(...)`）→ 正常当阴影值，可用 `shadow-[...]`；
- 值以颜色开头（`rgba(...)_0_15px_30px_-5px`）→ **必须**写成任意属性 `[box-shadow:...]`。

### 4.9.2 顺手加 `-webkit-line-clamp` 会改变 `display`（实测）

`display: -webkit-box` + `line-clamp: N` 是原项目的多行截断写法。若「顺手补上」
`[-webkit-line-clamp:N]`，Chromium 会把计算后的 `display` 从 `-webkit-box` **块化成 `flow-root`**
（实测），元素高度随之改变。原样保留 `[display:-webkit-box] [-webkit-box-orient:vertical] [line-clamp:N]`
即可，不要画蛇添足。

### 4.9.3 反向陷阱：内联会「唤醒」原本失效的移动端声明（3d 实测）

前面几节讲的都是「工具类压不过无层级 CSS」。反过来也会出事：

原来「移动端覆盖」与「基础规则」**特异性相同或更低**、只是靠源码顺序输掉时，
基础值一旦内联成 `@layer utilities` 的工具类，就不再参与那种比较 ——
那条无层级的移动端声明会**突然生效**，把观感改掉。

3d 实测两处（都已确认为**原本就失效的死声明**并删除）：
`LockView` 移动端 `.right { justify-content: center }`（基础同特异性但更靠后）、
`ArticleBox` 移动端 `.private-summary { min-height/margin-bottom/padding }`
（基础 `.cover .right .private-summary` 特异性更高）。

**检查办法**：迁移带媒体查询的组件时，先在每个断点实测一遍**
「移动端声明里有没有哪条，原本就输给了同属性、同/更高特异性的基础规则」。

### 4.11 两条会「悄悄改变视觉」的翻译陷阱（0d 实测）

**① transform 组合必须用 arbitrary property，不能用 `rotate-*` / `translate-*` 组合。**
v4 的 `rotate-45` 写的是独立属性 `rotate: 45deg`，`translate-x-1/2` 写的是 `translate: …`；
按 CSS 规范它们的合成顺序（translate → rotate → scale）与单个 `transform: rotate() translateX()`
**不同**。CSS 艺术类写法一律用 `[transform:rotate(-38deg)_translateX(50%)]`。

**② `max-*` 断点在 v4 不含端点。** `max-[960px]:` 生成 `@media (width < 960px)`，
而原 `@media (max-width: 960px)` 是**含** 960px 的。需要精确语义时用 arbitrary variant：
`[@media(max-width:960px)]:hidden`（实测在 960px 视口下与原文一致）。
迁移带断点的组件时，建议按端点值实测一次（headless Chrome 的 `--window-size` 就能验）。

**③ 语义字号 `text-sm` / `text-xs` / `text-base` 会**连带**设置 line-height。** 原 CSS 里
`font-size: 0.875rem` 不带行高，元素继承 `html` 的 `line-height: 1.5`（14px 字号 → 21px 行高）；
写成 `text-sm` 会额外写 `line-height: 1.25rem`（20px），**实测差 1px**，并连带影响父容器高度。
凡原文只写 `font-size` 的地方，一律用任意值 `text-[0.875rem]` / `text-[0.75rem]` / `text-[1rem]`——
任意字号只输出 `font-size`，不碰行高。同理 `rounded-full` 是 `calc(infinity * 1px)`，
需要精确 `50%` 时用 `rounded-[50%]`。

### 4.11 ④ `-webkit-` 与 unlayered 选择器造成的、只有实测才看得见的差异

§4.11 ①②③ 之外，P3 又实测出三条「写法看起来等价、计算值不等价」的情况，已分别记在
§4.9.1（shadow 颜色歧义）、§4.9.2（`-webkit-line-clamp` 块化 `display`）、
§4.9.3（内联唤醒失效的移动端声明）。共同教训是：**这些都不会报错、构建也通过**，
只有逐属性比对 `getComputedStyle` 才能发现。

### 4.12 死 CSS 的成本比想象中高（3e 实测）

`Knowledge.vue` 的 CSS 里有 13 条规则挂在 `.section-categories` / `.section-tags` 上，
但模板实际用的是裸 `.content-section` 与 `.content-section.tag-section` —— 这两个类名
**在模板里从未出现**。

因为 scoped 规则必须带 `[data-v-x]` 才能命中本组件元素，这些规则**永不可能匹配**。
它们还会误导后来的读者（以为存在「分类用 40% 宽、标签用 60%」的布局），
并且其中的 `@media (min-width: 901px)` 块整块都是死的。

**判定方法（三步，缺一不可）**：
1. `rg '<类名>' --glob '*.vue'` —— 确认它是否只出现在某个 `<style>` 里、任何模板都没有；
2. 从构建产物里确认这些选择器**带 `[data-v-<该文件 scope>]`**（不带 scope 的才是真全局规则，
   可能被别处依赖）；
3. 若是非 scoped 文件，还要看它有没有可能命中别组件的 DOM（§4.7）。

顺带一个副产品：`.grid-container { display: grid; gap: 20px }` 被两个栅格共用，
`.tag-section .grid-container` 再叠列定义；内联时按**实际命中对象**写即可
（分类那份没有列定义 → 单列；标签那份取 `repeat(auto-fill, minmax(130px, 1fr))`）。

### 4.10 视觉零 diff 的验证手段（0a–0d 一直在用）

`bun run build:type-check` 只保证能编译，**不代表视觉正确**，必须另做验证：

1. **计算样式比对**（推荐，覆盖面广）：先构建改动前的基线产物，用 headless Chrome 加载
   `dist/assets` 里的 `index-*.css` + `vendor-*.css` + 该组件 chunk，按真实 DOM 结构（含
   `data-v-*` scope 属性、Element Plus 内部类名）复刻一份最小 HTML，读 `getComputedStyle`
   存成 JSON；改完再构建一次、同一份 DOM 结构（只换 class 串）再测一次，逐属性 diff。
   ⚠️ 两个坑：**基线必须用改动前的源码构建**（否则是「旧 DOM + 新 CSS」的假结果）；
   复刻 DOM 必须与真实模板一致，包括作为 `:deep()` 锚点的类名。
2. **像素级比对**（用于可疑的等价性判断）：同一页面结构下只改一个声明，用
   `chrome --headless --screenshot` 各截一张，逐字节比 PNG。
   0a 起一直沿用的 `shadow-[...]` 就靠这个证实了与原 `box-shadow` **渲染逐像素一致**
   （v4 会改写成 `--tw-shadow` 链，链上其余槽位 `@property` 初始值是全透明 0 偏移，属空操作）。

**两条流程教训（都是真踩过的）：**

- **删整个 `<style>` 块前，逐条确认每个类是否真的不再需要。** 0b 把 `GatewayKeys.vue` 的
  `<style>` 块整块删掉时，连带删了 `.masked`——但那个类因为父级 `:deep(code)` 的缘故**不能**内联。
  这类失误不会报错、构建也通过，只会让颜色悄悄变掉。
- **改完后直接读文件确认，不要只看编辑工具的返回。** 一次对 `GatewayKeys.vue` 的补丁
  声称成功，但内容并未落进文件（提交时只记录了 1 行改动），直到下一个会话才发现。
  检查方式：`rg <关键字> <文件>` 配合 `git diff --stat`。

---

## 5. 分批迁移计划

> 顺序：**先后台、再变量层、后前台**。后台 CSS 多为 flex/间距/颜色，最好迁、风险最低；
> 前台主题嵌套深、伪元素多、依赖全局变量与全局媒体查询，放最后且等 P1 变量类可用后再动。

### P0 · 后台（19 个文件，1,205 → 564 行）— 最先做

| 批次 | 文件 | 行数 | 备注 |
|---|---|---|---|
| 0a 样板 | `components/backend/DashBoard/TotalItem.vue` | 69 → 0 | **首块样板**；4 个静态 `<li>` 的 nth-child 渐变 → 逐项 class（§2.2）✅ `d3580d4` |
| 0b 小组件 | `views/backend/ManagerView.vue` | 5 → 0 | 凑批；`.box-card` 为死样式（模板未引用），整块删除 ✅ `39936b2` |
| | `components/backend/AdminFooter.vue` | 14 → 0 | ✅ `39936b2` |
| | `components/backend/gateway/GatewayKeys.vue` | 15 → 15 | ⚠️ 0b 误删了 `.masked` 规则（该文件 `<style>` 块被整块移除），导致徽标被父级 `:deep(code)` 兜底变色；`e74e50b` 已补回 ✅ `39936b2` + `e74e50b` |
| | `views/backend/AdminView.vue` | 16 → 9 | 保留 `.el-aside { overflow: hidden }`（§4.8）✅ `39936b2` |
| | `components/backend/DashBoard/VisitTable.vue` | 19 → 9 | 保留 `.ban-row`（库内部 `<tr>`，工具类够不着）✅ `39936b2` |
| 0c 中组件 | `components/backend/gateway/GatewayOverview.vue` | 40 → 0 | 4 个 tile 类各用 1 次，全部内联，CSS 清零 ✅ `78cb419` |
| | `components/backend/gateway/GatewayMcp.vue` | 43 → 29 | 保留 `.scope-group+.scope-group`（相邻兄弟选择器）、`.step`（§4.5，3 次）+ `:deep()` ✅ `78cb419` |
| | `views/backend/SystemView.vue` | 52 → 52 | **无需改动**：整块样式都是 `:deep()`，已符合 §4.3 ✅ `78cb419` |
| | `views/backend/DashBoardView.vue` | 52 → 45 | 保留 `:deep(>div)`（跨组件覆盖 VisitTable/VisitChart），`.chart-item` 留作锚点 ✅ `78cb419` |
| | `views/backend/GatewayView.vue` | 54 → 55 | 保留 `.tab-label`（§4.5，6 次）+ 全部 `:deep()`。（+1 是加了一行说明注释，全项目唯一「迁移后比迁移前多」的文件）✅ `78cb419` |
| | `components/backend/gateway/SectionPanel.vue` | 55 → 17 | 保留 `.panel+.panel`（相邻兄弟）与 `.panel-title` 字重（§4.8，被全局 h3 规则压住） ✅ `78cb419` |
| 0d 大组件 | `components/backend/gateway/GatewayRouting.vue` | 64 → 0 | 按钮三态改 `optionClass()` 整体切换（§4.9），`el-icon` 尺寸/颜色改用 props（§4.8）✅ `e2e0acb` |
| | `components/backend/DashBoard/VisitChart.vue` | 71 → 20 | 原为**非 scoped**（§4.7）→ 改 scoped；保留 `.chart-select` 定宽与 `:deep(.el-radio-button__inner)`；`item-top/item-title/item-sub` 三个类名是父组件 `:deep(>div)` 的锚点，必须留 ✅ `b9e1180` |
| | `components/backend/AdminSide.vue` | 86 → 78 | 只迁 `.container` 布局与 `.tool`；保留 `h1`（§4.8）、`:deep(.el-menu)`、`.el-menu-item`/`.is-active`（EP 内部类 + 运行期状态类）、`.fold-container`（运行期切换）✅ `721cab0` |
| | `components/backend/gateway/GatewayProviders.vue` | 124 → 70 | 卡片三态用「静态 class 不含该属性 + `:class` 只给一份值」；保留 `.hint`/`.group-title`/`.group-sub`（各 4 处）、`.link`（全局 `a` 规则）、`.masked`（父级 `:deep(code)`）、`.key-row`（相邻兄弟锚点）、`.card-off :deep(...)` ✅ `994095b` |
| | `views/backend/CommentView.vue` | 226 → 85 | 后台最大；正文截断用 `line-clamp-2`；保留四个 CSS 变量、`.title-block h2`（§4.8）、`.comment-table`/`.article-title`（EP 同名属性，§4.8）、全部 `:deep()` + reduced-motion 媒体查询 ✅ `45aba5d` |
| | `views/backend/PublishView.vue` | 97 → 59 | 保留 `.title-input`（EP `.el-input` 的 width/font-size 压工具类）与第二个**非 scoped** 块——EP 的 Upload 根节点不带类名、`el-upload` 在其内部，`.emojis` 属第三方 `@vavt/v3-extension` ✅ `32238c7` |
| | `views/backend/LoginView.vue` | 103 → 21 | 非 scoped 整块迁完（泄漏面为零）；保留 `.login-title` 字重与 `.login-btn-main` 的 display/height/background-color；logo 用 arbitrary transform、断点用 arbitrary media（§4.11）✅ `7133aea` |

**人工测试点**：管理后台每个菜单页（仪表盘/文章/评论/分类标签/系统/网关）正常态 + 弹窗 + 移动端。

### P1 · 主题变量层改造（1 个点，全局收益）— ✅ 已完成（`5b81e6e`）

把 `style.less` 的 `:root` 变量搬进 `tailwind.css` 的 `@theme`（映射表见 §3.2），
使 `bg-grey-0`、`text-color-red`、`shadow-dh-normal` 等语义类可用。

- 是一次性小改造，**不改变任何视觉**，只新增工具类、保留原变量（旧代码继续可用）；
- 落地后，后续 P2/P3 迁移用语义类替代 `bg-[var(--xx)]`，后台已迁文件**不回头改**（除非顺手）；
- 需验证：构建产物中变量被正确内联/保留、Element Plus 样式不受影响。

**人工测试点**：全站随便点几个页面，确认配色/阴影无变化。

### P2 · webdav 模块（7 个文件）— 完成，CSS 2,193 → 894 行

| 批次 | 文件 | 行数 | 备注 |
|---|---|---|---|
| 2a | `views/frontend/webdav/components/DriveHeader.vue` | 0 → 0 | 文档此处笔误：该文件在基线里就没有 `<style>` 块（早已是纯 Tailwind）✅ 无需迁移 |
| | `views/frontend/webdav/components/FilePreview.vue` | 136 → 129 | 图标色改由 `getIconClass` 返回完整工具类（§4.9 的坑）；`.markdown-content` / `.code-content` 是运行时 HTML，保留（§4.3.1）；根节点 `animate-[fade-in…]` 本就失效（Vue 改名了 keyframes），未擅自修复 ✅ `5e15f36` |
| 2b | `views/frontend/webdav/components/MobileView.vue` | 201 → 47 | 保留 `.tab`/`.option-item`/`.icon-sm`（§4.5）与 h3 字重（§4.8）；字号用任意值避免行高差（§4.11 ③）✅ `eb242aa` |
| | `views/frontend/webdav/modals/ShareManagerModal.vue` | 222 → 60 | 保留 `.hint`/`.badge`(+danger,warn)/`.link-btn`(+danger,disabled,hover) 三组组件类；`last:border-b-0` 替代 `:last-child`；多态颜色整组二选一（§4.9）✅ `6fa0a79` |
| 2c | `views/frontend/webdav/modals/ShareLinkPopup.vue` | 271 → 128 | 保留开关的纯 CSS 状态机（`input:checked + .toggle-slider`，兄弟选择器无工具类写法）、带 :focus/:hover/:disabled 且重复的表单控件类、`.popup-title` 字重（§4.8）✅ `49ec57b` |
| | `views/frontend/webdav/modals/UploadModal.vue` | 489 → 132 | 拖放区三态 / 上传项四态走 §4.9 整体切换；`.loading-spinner` 与 `.progress-fill` 连同 `@keyframes spin` / `progress-animation` 保留（animation 必须与声明同块）；`.progress-fill` 原有两条重复规则合并（实测等价）✅ `c12a366` |
| 2d | `views/frontend/webdav/components/WebDriveView.vue` | 874 → 398 | 全项目第二大，且原为**非 scoped** 的 `lang="less"`（全局泄漏）；收紧为 scoped 前逐条核对过泄漏面。3 处 keyframes 按 §4.1 挪进 `tailwind.css`。断点用 arbitrary variant（§4.11②）✅ `a512430` |

**人工测试点**：云盘列表/预览/上传/分享弹窗，桌面 + 手机两种宽度。

### P3 · 前台主题（19 个文件）— 最后做、最谨慎；CSS 1,848 → 879 行

依赖全局变量与全局媒体查询最多，视觉敏感（博客门面）。**一批一个页面，逐页人工验收。**

| 批次 | 文件 | 行数 | 备注 |
|---|---|---|---|
| 3a 小组件 | `components/Child/Icon.vue` / `App.vue` | 0 / 1 → 0 | Icon 无样式；App 删掉空的 `<style></style>` ✅ `f691617` |
| | `components/frontend/Footer.vue` | 9 → 0 | ✅ `f691617` |
| | `components/frontend/Side/HomeSide.vue` | 49 → 0 | 首次用上 P1 的 `text-grey-7` / `border-grey-4` ✅ `f691617` |
| | `components/frontend/Banner.vue` | 48 → 34 | `#banner`/`.info` 内联；`.fade-in-article` 由 JS 运行时 add/remove，连同 `@keyframes fadeIn` 整组保留（§4.1/§4.3.1）。实测 `bg-[url(@/assets/images/banner.png)]` 能被 Vite alias 解析并 emit 资源 ✅ `f691617` |
| 3b 中组件 | `components/frontend/Comment/View.vue` | 13 → 0 | ✅ `1ab55d3` |
| | `views/frontend/ErrorView.vue` | 33 → 0 | 原为**非 scoped**；内联后 CSS chunk 整个消失 ✅ `1ab55d3` |
| | `views/frontend/MainView.vue` | 17 → 0 | 标签选择器 `Article` → 给 `<ArticleBox>` 加 `mb-[46px]`；先确认 ArticleBox 没在自己的根元素上写 margin（§4.8）✅ `1ab55d3` |
| | `components/frontend/Comment.vue` | 15 → 0 | ✅ `1ab55d3` |
| 3c 页面级 | `views/frontend/HomeView.vue` | 79 → 28 | 保留 `.v-*` transition 类与 `h3 { font-weight: bold }`（§4.8）✅ `9260b26` |
| | `components/frontend/Side/ArticleInfoSide.vue` | 84 → 21 | 保留 `.box{background:#fff}`（`.el-card` 冲突）、`:deep(.el-card__body)`、裸 `::-webkit-scrollbar` ✅ `9260b26` |
| | `views/frontend/ArticleView.vue` | 85 → 59 | 普通/全屏两态的 padding 改为**互斥 class 串**（§4.9）；其余几乎全是 `:deep()`（md-editor 运行时 HTML）✅ `9260b26` |
| | `components/frontend/Pagination.vue` | 86 → 79 | 只内联 `.pagination-shell`；`:deep()` 整块保留（实测 `.el-pagination` 自带 `--el-pagination-hover-color` 等，工具类会输，§4.8）✅ `9260b26` |
| | `components/frontend/Header.vue` | 37 → 0 | CSS chunk 消失；`#nav` / `.menu` 类名保留（style.less 在 1024/768 用 `!important` 覆盖）✅ `9260b26` |
| | `components/frontend/Comment/CommentItem.vue` | 127 → 43 | 保留 `.v-*` + `@keyframes bottom`、运行时切换的 `.reply-enter`。关键发现：原 CSS 大量嵌在 `ul {…}` 下而本组件模板里没有 `ul` —— Vue 只给**最后一个**复合选择器加 scope，那个 `ul` 由父组件 Comment/View 提供，所以规则一直是生效的（差点误判成死代码）✅ `9260b26` |
| 3d 复杂 | `views/frontend/LockView.vue` | 166 → 48 | 只留移动端覆盖块与 `.right button:hover span`。删掉一条**原本就失效**的移动端 `justify-content: center` ✅ `b473e6b` |
| | `components/frontend/ArticleBox.vue` | 193 → 63 | `.cover div{width:50%}` 等后代选择器按**实际命中对象**逐个内联；删掉三行**原本就失效**的移动端 `.private-summary` 声明。实测两坑：`shadow-[rgba(...)_…]` 被当作 shadow color（须写 `[box-shadow:…]`）；额外的 `-webkit-line-clamp` 会把 `display` 块化成 `flow-root` ✅ `b473e6b` |
| | `components/frontend/Comment/Publish.vue` | 339 → 244 | 三个 style 块：①79 行全内联；②61 行只留 3 条祖先 hover；③199 行是 Uiverse 书签动画控件，按 §4.1 **装饰组件豁免**整块保留。实测 `transition` 简写会重置 `transition-delay`，须合并成 `[transition:all_0.5s_ease_300ms]` ✅ `a4633b3` |
| | `views/frontend/Knowledge.vue` | 467 → 260 | 单块最大。**顺带删掉 13 条已证死规则**（`.section-categories`/`.section-tags` 两族 + 只含它们的 `@media(min-width:901px)` 块——这两个类名在模板里从未出现，见 §4.12）；`fadeInUp` 挪进 `tailwind.css`（§4.1）；其余按「非 v-for、非伪元素、非断点」逐块内联 ✅ `a2b9e7c` |

**人工测试点**：首页 / 文章详情 / 评论区 / 知识页 / 解锁页，桌面 + 平板 + 手机三档宽度，重点看首屏与封面。

---

## 6. 进度跟踪

> **行数口径**：均指 `<style>` 块的总行数，**含 `<style>` / `</style>` 两行标签**，
> 与 §1 基线（6,288 行）同口径。可用 `git show <rev>:<文件>` 逐文件复核。
> 个别文件的「迁移后」含后续补写的说明性注释（如 `FilePreview`、`Knowledge`、`GatewayView`）。
>
> 每完成一个文件就在对应行尾打 `✅` 并注明 commit；批次完成后更新下方计数。

| 阶段 | 文件数 | 状态 |
|---|---|---|
| P0 后台 | **19 / 19** ✅ | 完成；后台 CSS 1,205 → 564 行（净 -641），5 个文件清零 |
| P1 变量层 | **1 / 1** ✅ | 完成；`@theme inline` 映射，产物与改动前逐字节相同 |
| P2 webdav | **7 / 7** ✅ | 完成；webdav CSS 2,193 → 894 行（净 -1,299） |
| P3 前台 | **19 / 19** ✅ | 完成；前台 CSS 1,848 → 879 行（净 -969） |
| **合计** | **46 / 46** ✅ | 全部完成 |

**自洽校验**（可随时重跑）：§5 逐文件求和 = 5,246 → 2,337 行；加上豁免的
`Pet.vue` 950 + `Loading.vue` 92 = 1,042，得 6,288 → **3,379**，与 §1 基线及 §8.1 一致。

---

## 6.1 迁移过程中发现、已单独修复的既有问题

按「每个改动只含样式迁移、不夹带功能改动」（§7.3），这几个问题当时只登记、不混在迁移里，
后来各自单独开 commit 修复（因为都是**行为变更**：原本失效的效果开始生效）。

| 位置 | 现象 | 说明 |
|---|---|---|
| `webdav/components/FilePreview.vue` | 根节点的 `animate-[fade-in_0.3s_ease]` 从未播放 | Vue 把 scoped 块里的 `@keyframes` 改名成 `fade-in-<scope>`，工具类引用的仍是 `fade-in` → 失配。✅ 已修 `d03c59c`：keyframes 挪进 `tailwind.css` 的 `@theme`，根元素改 `animate-fade-in`。实测 `getAnimations()` 由 0 → 1 个动画，逐帧采样 `opacity` 为 0 → 0.802 → 1 |
| `views/frontend/Knowledge.vue` | 弹窗遮罩的 `backdrop-filter: blur(5px)` 在 Chrome 下不生效 | 原文同时写了无前缀与 `-webkit-` 两行，lightningcss 按目标浏览器归并后**只剩 `-webkit-`**，而当前 Chrome 不认它。✅ 已修 `9866bea`：只留无前缀那一条，让 lightningcss 自行补前缀。实测 `backdropFilter` 由 `none` → `blur(5px)`，遮罩区边缘强度 5.75 → 0.45 |

---

## 7. 给执行者（含 AI）的硬性要求

1. **一次只拿一批里的一个文件**（或一批小组件），做完汇报再拿下一个；不要跨阶段并行。
2. **不许 git commit**：改动留在工作区，等用户人工确认后由主会话提交。
3. 每个改动只含样式迁移，**不夹带功能改动**。
4. 遇到 §4 的特殊场景按规则处理；规则没覆盖到的，**停下来问用户**。
5. 不许动「永久豁免清单」里的文件。
6. 汇报格式：`已完成 <文件>：删除 CSS N 行，构建通过，请人工看 <页面/状态>`。

---

## 8. 收尾总结（交接必读）

### 8.1 结果

| 指标 | 迁移前 | 迁移后 |
|---|---|---|
| `.vue` 的 `<style>` 总行数 | 6,288 | **3,379**（−2,909，−46.3%） |
| 带 `<style>` 块的文件 | 45 | **32** |
| 样式块彻底消失的文件 | — | **13 个** |

13 个「样式块归零」的文件：`TotalItem` / `GatewayOverview` / `GatewayRouting` / `ManagerView` /
`AdminFooter` / `Comment/View` / `ErrorView` / `Header` / `MainView` / `Comment.vue` /
`Footer` / `HomeSide` / `App.vue`。

提交：22 个 `refactor(style)` + 2 个 `fix(style)`（另有 2 个更早的 `fix(style)` 属迁移前的修正）。

### 8.2 现在还剩什么（32 个带样式块的文件）

按性质分五类，**都不是「没做完」**，而是工具类表达不了或已获批的例外：

1. **运行时才能确定的类名**
   - Vue `<transition>` 在运行期生成的 `.v-*` / `.modal-fade-*` / `.simple-fade-*`（写不进模板）
   - 组件 `:class` 动态切换、且切换的是「整组属性」的（如 `CommentItem` 的 `.reply-enter`）
   - 由 JS `classList.add/remove` 加的（如 `Banner` 的 `.fade-in-article`）
   - 非 scoped 文件里**可能命中别组件 DOM** 的全局规则（迁移时逐条判定过，见 §4.7）
2. **伪元素与伪类**：`::-webkit-scrollbar` 系列、`::before` / `::after`、`:nth-of-type` /
   `:first-of-type` / `:last-child`、祖先 `:hover` + 后代（`a:hover b`）
3. **子组件 / 第三方 DOM**：`:deep()`（Element Plus 内部、md-editor 运行时 HTML）、
   子组件根元素（如 `Icon` 渲染出的 `<svg class="icon">`）
4. **`@keyframes` 与它引用的 `animation`**：按 §4.1 处理 —— 要么整组留在同一个 scoped 块里，
   要么整组搬进 `tailwind.css` 的 `@theme`（推荐，见 §8.4 现状）
5. **§4.5 的组件类与断点密集簇**：`v-for` 里重复渲染的 `.card` / `.badge` / `.count`，
   以及 2–3 档媒体查询叠一整套声明的元素

### 8.3 六类「不得内联」的判据（一句话版）

| # | 情况 | 处理 |
|---|---|---|
| 1 | 类名在**运行期**才出现 | 留在 CSS |
| 2 | 选择器是**伪元素/伪类/祖先 hover** | 留在 CSS |
| 3 | 目标 DOM 属于**子组件或第三方库** | 留在 CSS 或 `:deep()` |
| 4 | 类里写了 `animation`、且 `@keyframes` 同块 | 整组搬进 `tailwind.css` 的 `@theme` |
| 5 | 类在 `v-for` 里重复 ≥3 次，或自带 2–3 档断点 | 留作组件类（§4.5） |
| 6 | 元素上已有**无层级的同属性声明**（库/全局 scoped） | 留 scoped（§4.8） |

### 8.4 现在的动画清单（都在 `tailwind.css` 的 `@theme`）

| 工具类 | 原位置 | 备注 |
|---|---|---|
| `animate-spin` | — | Tailwind 内置 |
| `animate-dialog-appear` | `WebDriveView` | |
| `animate-highlight-pulse` | `WebDriveView` | |
| `animate-fade-in-up` | `Knowledge` | 错峰延迟仍由 scoped 的 `:first-of-type` / `:nth-of-type(2)` 与内联 style 提供 |
| `animate-fade-in` | `FilePreview` | 本次修复才真正生效（§6.1） |

仍留在各组件 scoped 块里的 keyframes：`UploadModal`（`spin` / `progress-animation`）、
`CommentItem`（`bottom`）、`Banner`（`fadeIn`）、`Publish`（书签控件的 3 个）、`Pet`（豁免）。
它们与各自的 `animation` 同块，是**有意保留**的。

### 8.5 验证方法论（下一轮改动请照用）

1. **计算样式比对**（主力）：改动前构建基线产物 → 用 headless Chrome 加载 `dist/assets` 里的 CSS +
   按真实 DOM 结构（含 `data-v-*` scope 属性）复刻最小 HTML → 读 `getComputedStyle` 存 JSON →
   改完再测同一份 DOM（只换 class 串）→ 逐属性 diff。
   **基线必须用改动前的源码构建**；复刻 DOM 必须与真实模板一致，包括 `:deep()` 的锚点类名与伪元素。
2. **规则文本比对**：把两个产物的「含本文件 scope 的规则」抽出来、把 scope 哈希归一成 `@S@` 后
   做集合 diff。期望「仅 after」为 0 或逐条可解释。
3. **像素级比对**：`chrome --headless --screenshot` 各截一张、逐字节比 PNG。
   用于判断等价性成疑的单条声明（如 `shadow-[...]`、`backdrop-filter`）。
4. **动画是否真的在跑**：`getComputedStyle` **区分不了**「声明了动画」和「动画真的在播」——
   必须用 `el.getAnimations().length`，必要时 `currentTime` 逐帧采样。
5. **多档视口**：断点相关的改动至少测 1440 / 768 / 480，并**在断点整数值上各测一次**
   （`@media(max-width:768px)` 含端点，而 Tailwind 的 `max-md:` 不含，见 §4.11②）。
6. **`bun run build:type-check`** —— 注意它只保证能编译，**不代表视觉正确**；
   但它多次抓到过真错误（如遗漏 `<script>` 块、`CommentItem` 的 `replay` 未定义）。

### 8.6 仍需人工确认的点

自动化验证覆盖了计算样式、规则文本、关键像素与动画，但以下只能靠肉眼：

- 首屏观感与封面图（`ArticleBox` / `Banner` / `HomeSide`）
- `Knowledge` 页（粒子背景 + 弹窗动画；该页在 headless 下会挂住，无法自动截图）
- Element Plus 组件的 hover / active / focus 等交互态（`:deep()` 部分）
- 各页面在真机上的滚动、触摸与安全区表现

发现观感偏差时，先用 §8.5 的第 1、3 条定位是哪条声明变了，再回 §4 找对应判据。
