# 桌宠唱歌的音频

`let-me-go-loop.mp3` 由 `let-me-go-duet-16s.mp3`（B 站「大肥鱼」那首）剪出来，供
`composables/usePetSing.ts` 循环播放。剪辑参数：

| 参数 | 值 | 说明 |
|---|---|---|
| 起点 | 3.83s | "Let me go, let me go" 收尾处，正好是下一句的换气点 |
| 长度 | 6 小节 = 11.625s（123.87 BPM，1 小节 1.937571s） | 整小节，循环时鼓点相位不漂 |
| 接缝 | 0.5s 等功率交叉淡化 | 尾巴（"Let me go"）融进起点前那半秒，接缝处波形连续 |
| 响度 | -17.8 LUFS / 峰值 -1 dBFS | 当背景音，不抢正文 |
| 编码 | mono 128kbps MP3 | 原曲本身就是单声道（左右声道完全相同） |

这样循环一次的听感是「I'm making the calls → Let me write the JSON → I'll call the tool →
let me do it, okay, okay → let me go…」然后无缝回到第一句。

换音频时记得同步 `usePetSing.ts` 里的 `LOOP_SECONDS` 和 `LYRICS` 时间轴（相对循环起点）。
