<template>
  <div class="comment-form relative w-full mx-auto py-4 px-2 bg-white dark:bg-[#222] dark:text-[#ccc] rounded-xl text-[12px]">
    <div class="right-top absolute right-3 top-2">
      <label class="ui-bookmark">
        <input type="checkbox" v-model="comment.isPublic" />
        <div class="bookmark">
          <svg viewBox="0 0 32 32">
            <g>
              <path
                d="M27 4v27a1 1 0 0 1-1.625.781L16 24.281l-9.375 7.5A1 1 0 0 1 5 31V4a4 4 0 0 1 4-4h14a4 4 0 0 1 4 4z">
              </path>
            </g>
          </svg>
        </div>
      </label>
    </div>
    <div class="author-info w-full grid grid-cols-3 items-start leading-[24px] dark:border-b-[#444]!">
      <div class="min-w-0">
        <input ref="authorInput" class="input w-full border-none text-[12px] py-2 px-3 focus:[outline:none]" type="text"
          placeholder="* 昵称" maxlength="15" v-model="comment.author" @input="authorTouched = true"
          :aria-invalid="!!authorHint" />
        <!-- 恒定占位：提示出现/消失不改动卡片高度，滚动条就不会跟着跳 -->
        <p class="hint-line" aria-live="polite">{{ authorHint }}</p>
      </div>
      <div class="min-w-0">
        <input ref="emailInput" class="input w-full border-none text-[12px] py-2 px-3 focus:[outline:none]" type="email"
          placeholder="* 邮箱" maxlength="20" v-model="comment.email" @input="emailTouched = true"
          @blur="emailTouched = true" :aria-invalid="!!emailHint" />
        <p class="hint-line" aria-live="polite">{{ emailHint }}</p>
      </div>
      <div class="text-[#666] dark:text-[#aaa] mr-[30px]">
        {{ comment.isPublic ? '评论已公开，任何人均可阅读' : '评论已私密，仅博主可见' }}
      </div>
    </div>
    <div class="comment-content mt-4">
      <textarea @focus="viewState.showEmoji = true" @blur="handleTextareaBlur"
        class="input border-none w-full h-[70px] py-2 px-[13px] resize-y min-h-[70px] focus:[outline:none]" type="textarea"
        placeholder="想要说些什么呢" ref="textarea" v-model="comment.content" @input="contentTouched = true"
        :aria-invalid="!!contentHint"></textarea>
      <p class="hint-line" aria-live="polite">{{ contentHint }}</p>
    </div>

    <div class="comment-action mt-2 w-full flex justify-between">
      <div class="action-left flex-1">
        <div class="emoji">
          <Transition name="emoji-panel">
            <!-- mousedown 挂在整个面板上：输入框不失焦，面板才不会在点中表情之前先收起 -->
            <ul v-show="viewState.showEmoji" ref="emojiPanel"
              class="list-none text-[18px] grid grid-cols-[repeat(10,1fr)] gap-[6px]"
              @mousedown="keepTextareaFocus">
              <li v-for="(ji, index) in emojis" :key="ji" class="animate-emoji-pop mr-[10px] cursor-pointer"
                :style="{ animationDelay: emojiDelay(index) }" @click="insertEmoji(ji)">{{ ji }}</li>
            </ul>
          </Transition>
        </div>
      </div>
      <div class="action-right ml-[30px] relative left-[10px]">
        <button @click="submitComment"
          :disabled="submitting"
          class="[transform:scale(0.87)] flex items-center justify-center gap-[10px] py-0 px-[10px] text-white [text-shadow:2px_2px_rgb(116,116,116)] uppercase cursor-pointer border-solid border-2 border-black tracking-[1px] font-semibold text-[17px] bg-[hsl(49deg_98%_60%)] rounded-[50px] relative overflow-hidden [transition:all_0.5s_ease] active:[transform:scale(0.77)] active:[transition:all_100ms_ease] disabled:pointer-events-none disabled:opacity-60">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36" width="36px" height="36px"
            class="[transition:all_0.5s_ease] z-[2]">
            <rect width="36" height="36" x="0" y="0" fill="#fdd835"></rect>
            <path fill="#e53935"
              d="M38.67,42H11.52C11.27,40.62,11,38.57,11,36c0-5,0-11,0-11s1.44-7.39,3.22-9.59 c1.67-2.06,2.76-3.48,6.78-4.41c3-0.7,7.13-0.23,9,1c2.15,1.42,3.37,6.67,3.81,11.29c1.49-0.3,5.21,0.2,5.5,1.28 C40.89,30.29,39.48,38.31,38.67,42z">
            </path>
            <path fill="#b71c1c"
              d="M39.02,42H11.99c-0.22-2.67-0.48-7.05-0.49-12.72c0.83,4.18,1.63,9.59,6.98,9.79 c3.48,0.12,8.27,0.55,9.83-2.45c1.57-3,3.72-8.95,3.51-15.62c-0.19-5.84-1.75-8.2-2.13-8.7c0.59,0.66,3.74,4.49,4.01,11.7 c0.03,0.83,0.06,1.72,0.08,2.66c4.21-0.15,5.93,1.5,6.07,2.35C40.68,33.85,39.8,38.9,39.02,42z">
            </path>
            <path fill="#212121"
              d="M35,27.17c0,3.67-0.28,11.2-0.42,14.83h-2C32.72,38.42,33,30.83,33,27.17 c0-5.54-1.46-12.65-3.55-14.02c-1.65-1.08-5.49-1.48-8.23-0.85c-3.62,0.83-4.57,1.99-6.14,3.92L15,16.32 c-1.31,1.6-2.59,6.92-3,8.96v10.8c0,2.58,0.28,4.61,0.54,5.92H10.5c-0.25-1.41-0.5-3.42-0.5-5.92l0.02-11.09 c0.15-0.77,1.55-7.63,3.43-9.94l0.08-0.09c1.65-2.03,2.96-3.63,7.25-4.61c3.28-0.76,7.67-0.25,9.77,1.13 C33.79,13.6,35,22.23,35,27.17z">
            </path>
            <path fill="#01579b"
              d="M17.165,17.283c5.217-0.055,9.391,0.283,9,6.011c-0.391,5.728-8.478,5.533-9.391,5.337 c-0.913-0.196-7.826-0.043-7.696-5.337C9.209,18,13.645,17.32,17.165,17.283z">
            </path>
            <path fill="#212121"
              d="M40.739,37.38c-0.28,1.99-0.69,3.53-1.22,4.62h-2.43c0.25-0.19,1.13-1.11,1.67-4.9 c0.57-4-0.23-11.79-0.93-12.78c-0.4-0.4-2.63-0.8-4.37-0.89l0.1-1.99c1.04,0.05,4.53,0.31,5.71,1.49 C40.689,24.36,41.289,33.53,40.739,37.38z">
            </path>
            <path fill="#81d4fa"
              d="M10.154,20.201c0.261,2.059-0.196,3.351,2.543,3.546s8.076,1.022,9.402-0.554 c1.326-1.576,1.75-4.365-0.891-5.267C19.336,17.287,12.959,16.251,10.154,20.201z">
            </path>
            <path fill="#212121"
              d="M17.615,29.677c-0.502,0-0.873-0.03-1.052-0.069c-0.086-0.019-0.236-0.035-0.434-0.06 c-5.344-0.679-8.053-2.784-8.052-6.255c0.001-2.698,1.17-7.238,8.986-7.32l0.181-0.002c3.444-0.038,6.414-0.068,8.272,1.818 c1.173,1.191,1.712,3,1.647,5.53c-0.044,1.688-0.785,3.147-2.144,4.217C22.785,29.296,19.388,29.677,17.615,29.677z M17.086,17.973 c-7.006,0.074-7.008,4.023-7.008,5.321c-0.001,3.109,3.598,3.926,6.305,4.27c0.273,0.035,0.48,0.063,0.601,0.089 c0.563,0.101,4.68,0.035,6.855-1.732c0.865-0.702,1.299-1.57,1.326-2.653c0.051-1.958-0.301-3.291-1.073-4.075 c-1.262-1.281-3.834-1.255-6.825-1.222L17.086,17.973z">
            </path>
            <path fill="#e1f5fe"
              d="M15.078,19.043c1.957-0.326,5.122-0.529,4.435,1.304c-0.489,1.304-7.185,2.185-7.185,0.652 C12.328,19.467,15.078,19.043,15.078,19.043z">
            </path>
          </svg>
          <span class="now absolute left-0 [transform:translateX(-100%)] [transition:all_0.5s_ease] z-[2]">评论!</span>
          <span class="play [transition:all_0.5s_ease_300ms]">发表</span>
        </button>
      </div>
    </div>
  </div>
</template>
<style scoped>
/* 祖先 :hover + 后代，工具类无对应写法 */
button:hover svg {
  transform: scale(3) translate(50%);
}

button:hover .now {
  transform: translateX(10px);
  transition-delay: 300ms;
}

button:hover .play {
  transform: translateX(200%);
  transition-delay: 300ms;
}

/*
  固定一行高度：提示文字出现/消失不改变卡片高度，页面滚动条也就不会突然长出一截。
  用 min-height 而不是 height，长提示换行时不会被裁掉。
*/
.hint-line {
  margin: 1px 0 0;
  min-height: 16px;
  color: var(--color-red);
  font-size: 11px;
  line-height: 16px;
  /* 占位行本身不响应点击，免得它挡住下面一点点的空白区域 */
  pointer-events: none;
}

/* 面板从上方落下再收回。用 transition 而不是 @keyframes：进、出两个方向共用一套，
   也避开「scoped 里的 keyframes 被 Vue 改名」那个坑。 */
.emoji-panel-enter-active,
.emoji-panel-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.emoji-panel-enter-from,
.emoji-panel-leave-to {
  opacity: 0;
  transform: translateY(-0.375rem);
}
</style>

<style scoped>

/* Uiverse 书签动画控件：兄弟组合器 + 伪元素 + 3 组 keyframes，整块保留 */
.ui-bookmark {
  --icon-size: 24px;
  --icon-secondary-color: rgb(77, 77, 77);
  --icon-hover-color: rgb(97, 97, 97);
  --icon-primary-color: gold;
  --icon-circle-border: 1px solid var(--icon-primary-color);
  --icon-circle-size: 35px;
  --icon-anmt-duration: 0.3s;
}

.ui-bookmark input {
  -webkit-appearance: none;
  -moz-appearance: none;
  appearance: none;
  display: none;
}

.ui-bookmark .bookmark {
  width: var(--icon-size);
  height: auto;
  fill: var(--icon-secondary-color);
  cursor: pointer;
  -webkit-transition: 0.2s;
  -o-transition: 0.2s;
  transition: 0.2s;
  display: -webkit-box;
  display: -ms-flexbox;
  display: flex;
  -webkit-box-pack: center;
  -ms-flex-pack: center;
  justify-content: center;
  -webkit-box-align: center;
  -ms-flex-align: center;
  align-items: center;
  position: relative;
  -webkit-transform-origin: top;
  -ms-transform-origin: top;
  transform-origin: top;
}

.bookmark::after {
  content: "";
  position: absolute;
  width: 10px;
  height: 10px;
  -webkit-box-shadow: 0 30px 0 -4px var(--icon-primary-color),
    30px 0 0 -4px var(--icon-primary-color),
    0 -30px 0 -4px var(--icon-primary-color),
    -30px 0 0 -4px var(--icon-primary-color),
    -22px 22px 0 -4px var(--icon-primary-color),
    -22px -22px 0 -4px var(--icon-primary-color),
    22px -22px 0 -4px var(--icon-primary-color),
    22px 22px 0 -4px var(--icon-primary-color);
  box-shadow: 0 30px 0 -4px var(--icon-primary-color),
    30px 0 0 -4px var(--icon-primary-color),
    0 -30px 0 -4px var(--icon-primary-color),
    -30px 0 0 -4px var(--icon-primary-color),
    -22px 22px 0 -4px var(--icon-primary-color),
    -22px -22px 0 -4px var(--icon-primary-color),
    22px -22px 0 -4px var(--icon-primary-color),
    22px 22px 0 -4px var(--icon-primary-color);
  border-radius: 50%;
  -webkit-transform: scale(0);
  -ms-transform: scale(0);
  transform: scale(0);
}

.bookmark::before {
  content: "";
  position: absolute;
  border-radius: 50%;
  border: var(--icon-circle-border);
  opacity: 0;
}

/* actions */

.ui-bookmark:hover .bookmark {
  fill: var(--icon-hover-color);
}

.ui-bookmark input:checked+.bookmark::after {
  -webkit-animation: circles var(--icon-anmt-duration) cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards;
  animation: circles var(--icon-anmt-duration) cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards;
  -webkit-animation-delay: var(--icon-anmt-duration);
  animation-delay: var(--icon-anmt-duration);
}

.ui-bookmark input:checked+.bookmark {
  fill: var(--icon-primary-color);
  -webkit-animation: bookmark var(--icon-anmt-duration) forwards;
  animation: bookmark var(--icon-anmt-duration) forwards;
  -webkit-transition-delay: 0.3s;
  -o-transition-delay: 0.3s;
  transition-delay: 0.3s;
}

.ui-bookmark input:checked+.bookmark::before {
  -webkit-animation: circle var(--icon-anmt-duration) cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards;
  animation: circle var(--icon-anmt-duration) cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards;
  -webkit-animation-delay: var(--icon-anmt-duration);
  animation-delay: var(--icon-anmt-duration);
}

@-webkit-keyframes bookmark {
  50% {
    -webkit-transform: scaleY(0.6);
    transform: scaleY(0.6);
  }

  100% {
    -webkit-transform: scaleY(1);
    transform: scaleY(1);
  }
}

@keyframes bookmark {
  50% {
    -webkit-transform: scaleY(0.6);
    transform: scaleY(0.6);
  }

  100% {
    -webkit-transform: scaleY(1);
    transform: scaleY(1);
  }
}

@-webkit-keyframes circle {
  from {
    width: 0;
    height: 0;
    opacity: 0;
  }

  90% {
    width: var(--icon-circle-size);
    height: var(--icon-circle-size);
    opacity: 1;
  }

  to {
    opacity: 0;
  }
}

@keyframes circle {
  from {
    width: 0;
    height: 0;
    opacity: 0;
  }

  90% {
    width: var(--icon-circle-size);
    height: var(--icon-circle-size);
    opacity: 1;
  }

  to {
    opacity: 0;
  }
}

@-webkit-keyframes circles {
  from {
    -webkit-transform: scale(0);
    transform: scale(0);
  }

  40% {
    opacity: 1;
  }

  to {
    -webkit-transform: scale(0.8);
    transform: scale(0.8);
    opacity: 0;
  }
}

@keyframes circles {
  from {
    -webkit-transform: scale(0);
    transform: scale(0);
  }

  40% {
    opacity: 1;
  }

  to {
    -webkit-transform: scale(0.8);
    transform: scale(0.8);
    opacity: 0;
  }
}
</style>


<script setup>
import { emojis } from '@/types/Constant';
import { useUserStore } from '@/store/index'
import { addComment } from '@/api/user.ts'

const store = useUserStore()

const emit = defineEmits(['comment-submitted']);
const props = defineProps({
  parentId: {
    type: String,
    default: null
  }
})

const viewState = reactive({ showEmoji: false })
const emojiPanel = ref(null)

// 焦点还在面板里（Tab 到某个表情上）就先别收起，否则面板会在点中它之前先消失。
const handleTextareaBlur = (event) => {
  const nextFocus = event.relatedTarget
  if (nextFocus instanceof Node && emojiPanel.value?.contains(nextFocus)) return
  viewState.showEmoji = false
}

// 逐个错峰浮入。封顶是为了表情多起来以后，最后一排不用等太久。
const EMOJI_STAGGER_MS = 12
const EMOJI_STAGGER_LIMIT = 18
const emojiDelay = (index) => `${Math.min(index, EMOJI_STAGGER_LIMIT) * EMOJI_STAGGER_MS}ms`

const AUTHOR_MINIMUM = 2
const CONTENT_MINIMUM = 3
const EMAIL_PATTERN = /^([a-zA-Z0-9_-])+@([a-zA-Z0-9_-])+((\.[a-zA-Z0-9_-]{2,3}){1,2})$/

const commenterStorageKey = 'dh-blog:commenter'

function readSavedCommenter() {
  try {
    const saved = JSON.parse(localStorage.getItem(commenterStorageKey) || 'null')
    if (!saved || typeof saved !== 'object') return emptyCommenter()
    return {
      author: typeof saved.author === 'string' ? saved.author.slice(0, 15) : '',
      email: typeof saved.email === 'string' ? saved.email.slice(0, 20) : '',
      isPublic: saved.isPublic !== false,
    }
  } catch {
    return emptyCommenter()
  }
}

function emptyCommenter() {
  return { author: '', email: '', isPublic: true }
}

// 只记身份，不记正文：成功后的评论内容不该再填回输入框。
function saveCommenter(commenter) {
  try {
    localStorage.setItem(commenterStorageKey, JSON.stringify({
      author: commenter.author,
      email: commenter.email,
      isPublic: commenter.isPublic,
    }))
  } catch {
    // 无痕模式或配额满时评论已经发出，存档失败不打断发表。
  }
}

const savedCommenter = readSavedCommenter()
const comment = reactive({
  articleId: null,
  author: savedCommenter.author,
  content: '',
  email: savedCommenter.email,
  parentId: props.parentId,
  isPublic: savedCommenter.isPublic,
})

const textarea = ref(null)
const authorInput = ref(null)
const emailInput = ref(null)
const submitting = ref(false)

// 没碰过就不报错，否则空表单一进来就满屏红字。
const authorTouched = ref(false)
const emailTouched = ref(false)
const contentTouched = ref(false)

const authorHint = computed(() => {
  if (!authorTouched.value || comment.author.length >= AUTHOR_MINIMUM) return ''
  return `昵称至少 ${AUTHOR_MINIMUM} 个字`
})

const emailHint = computed(() => {
  if (EMAIL_PATTERN.test(comment.email)) return ''
  // 只在写完了 @ 之后才挑格式，免得刚打第一个字母就被说「格式不正确」；失焦或提交后一律提示。
  if (!emailTouched.value && !comment.email.includes('@')) return ''
  return comment.email ? '邮箱格式不正确' : '请填写邮箱'
})

const contentHint = computed(() => {
  if (!contentTouched.value || comment.content.length >= CONTENT_MINIMUM) return ''
  const remaining = CONTENT_MINIMUM - comment.content.length
  return comment.content ? `还差 ${remaining} 个字` : `评论至少 ${CONTENT_MINIMUM} 个字`
})

// 按下表情时阻止默认行为：输入框不失焦，面板就不会在插入之前先收起，光标也留在原处。
const keepTextareaFocus = (event) => event.preventDefault()

// 插到光标处而不是末尾；插完光标停在表情后面，可以接着打字。
const insertEmoji = (emoji) => {
  const field = textarea.value
  const caretStart = field ? field.selectionStart ?? comment.content.length : comment.content.length
  const caretEnd = field ? field.selectionEnd ?? caretStart : caretStart

  comment.content = comment.content.slice(0, caretStart) + emoji + comment.content.slice(caretEnd)
  contentTouched.value = true
  if (!field) return

  // v-model 要等这次渲染才把新值写回 DOM，提前摆光标会被覆盖掉。
  nextTick(() => {
    const caretAfterEmoji = caretStart + emoji.length
    field.focus()
    field.setSelectionRange(caretAfterEmoji, caretAfterEmoji)
  })
}

const submitComment = async () => {
  if (submitting.value) return

  // 提示就在各自的输入框下面，这里只补一次「都摸过了」并把光标送过去，不再叠一层 toast。
  authorTouched.value = true
  emailTouched.value = true
  contentTouched.value = true
  const firstInvalidInput = [
    [authorHint.value, authorInput],
    [emailHint.value, emailInput],
    [contentHint.value, textarea],
  ].find(([hint]) => hint)?.[1]
  if (firstInvalidInput) {
    firstInvalidInput.value?.focus()
    return
  }

  submitting.value = true
  try {
    await addComment({
      articleId: store.homeHeaderInfo.id,
      author: comment.author,
      email: comment.email,
      content: comment.content,
      isPublic: comment.isPublic,
      parentId: props.parentId,
    })
    saveCommenter(comment)
    comment.content = ''
    // 发完刚清空，别立刻在下面顶出一行「评论至少 3 个字」。
    contentTouched.value = false
    emit('comment-submitted')
  } catch {
    // 拦截器已经提示；失败不存档，输入留着让人改完再发。
  } finally {
    submitting.value = false
  }
}
</script>
