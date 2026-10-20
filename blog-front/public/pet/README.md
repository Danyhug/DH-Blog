# 桌宠素材包（运行期加载，不经过打包）

目录结构照搬上游，`public/pet/` 会原样出现在 `dist/pet/` 里，由后端 `internal/frontend/embed.go`
的 `/pet/*` 路由直接给出。`Pet.vue` 用
`new URL('pet/whale/', document.baseURI)` 定位，所以**换素材只替换这里的文件，不用改任何代码**。

```
pet/
├── kit/            body.js（身体、动作、表情、粒子）、rig.js（WebGL 网格渲染）
└── whale/          figure.js（大肥鱼本体）、model.json（部件盒与锚点）、figure.json（清单）
    ├── tex/ feat/  默认配色（DeepSeek 原版）的分层贴图与表情部件（WebP）
    ├── schemes/    另外 7 套配色，同一套几何、只换贴图
    ├── outfits/    cape 冬装的整套部件
    └── thumbs/     各配色的 144×144 预览图（当前 UI 未使用，留给以后的选择器）
```

## 与上游的差异

上游：<https://github.com/Pal-AI-Lab/Coopanion/tree/main/packages/cortico-world-desktop-pet/web>

`kit/` 是**原样**复制。`whale/figure.js` 打了 5 处补丁（都在这个文件里，替换上游文件后需要重新打一遍）：

1. **腿的摆幅按腿长换算**。`kit` 给的是 Coo 的 29 单位髋→脚线段，她的腿是 53；原来的角度直接套用会让步幅
   放大 1.8 倍、鞋被踢翻。改成 `asin(sin(a) * 29 / legLen)`，只让脚走同样的距离，手臂仍用原始角度。
2. **蹲/落地不再让鞋穿地**。`kit` 的 `low` 同时包含坐下、蹲、落地和行走起伏，整段加到 `body.ty` 上会把
   连鞋一起压进地板；现在只有真正的坐姿（`sitK → 1`）移动整体，其余换算成绕鞋底的压缩。
3. **脚离地高度改成读脚本身**。原来用 `29 - 髋脚距离` 判断抬脚，髋一下沉就被误判成「抬脚」，落地时
   先飘再陷；改用 `241 - 脚y`（`kit` 站立时脚心 y=241），并且只在被拎起时允许负值。
4. **蹲姿的脚不再前滑**。`kit` 用「半坐」实现蹲，会把脚前移 26 单位；算站立脚角时先减掉这部分位移。
5. **跳跃的二次动作与落地挤压**。`figure` 拿不到竖直速度，离地/落地时给头发、裙摆、尾巴、呆毛各加一次冲量；
   落地的压扁在触地那一帧由 `sp.land` 起，绕鞋底缩放。

另外在 `draw()` 里留了一个调试出口：页面设置 `globalThis.__figureDebug = true` 时，每帧把内部状态写到
`globalThis.__figureSt`，量落地/走路用得上，正常使用不会触发。

## 贴图格式：WebP（不是上游的 PNG）

430 张贴图全部换成了**无损** WebP（`cwebp -lossless`，像素与原 PNG 完全一致，alpha 通道原样保留），
整包从 11.1 MB 降到 7.8 MB，直接体现在打包后的可执行文件体积上。同时改了三处文件名引用：
`figure.js` 里拼 `${dir}tex/...` 与 `${dir}feat/...` 的两行模板字符串、`model.json` 的 20 个
`outfits` src、`figure.json` 的 11 个 `thumbs`。

**换上游素材后要重跑一次**（脚本幂等，已经转过的文件不会再动）：

```bash
cd blog-deploy && ./webp-pet.sh
```

依赖 `cwebp`：macOS 用 `brew install webp`，Debian/Ubuntu 用 `apt install webp`。
