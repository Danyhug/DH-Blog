// highlight.js 的完整包注册了近 200 种语言（压缩后约 1MB），而网盘预览只认 fileType.ts 里列出的扩展名。
// 这里只取官方的常用语言集合，再补上 CODE_EXTS / LANG_MAP 用到、但常用集合里没有的几种。
// 在 fileType.ts 增加扩展名时，若对应语言不在 highlight.js/lib/common 里，要在这里一并注册，
// 否则会退回自动识别（不报错，只是高亮可能不准）。
import hljs from 'highlight.js/lib/common'
import cmake from 'highlight.js/lib/languages/cmake'
import dart from 'highlight.js/lib/languages/dart'
import dockerfile from 'highlight.js/lib/languages/dockerfile'
import dos from 'highlight.js/lib/languages/dos'
import elixir from 'highlight.js/lib/languages/elixir'
import gradle from 'highlight.js/lib/languages/gradle'
import groovy from 'highlight.js/lib/languages/groovy'
import julia from 'highlight.js/lib/languages/julia'
import powershell from 'highlight.js/lib/languages/powershell'
import protobuf from 'highlight.js/lib/languages/protobuf'
import scala from 'highlight.js/lib/languages/scala'

const extraLanguages = { cmake, dart, dockerfile, dos, elixir, gradle, groovy, julia, powershell, protobuf, scala }
for (const [name, language] of Object.entries(extraLanguages)) hljs.registerLanguage(name, language)

export default hljs
