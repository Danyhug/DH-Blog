<template>
  <TransitionGroup>
    <li v-for="comment in commentList" :key="comment.id" class="comment-box list-none">
      <div class="comment-container flex mb-5">
        <div class="comment-avatar cursor-pointer">
          <img :alt="`${comment.author}'s avatar`"
            :src="`//cravatar.cn/avatar/${comment.email == 'danyhug@qq.com' ? '4831d466a6ba28c45b70edefc025fa8f' : comment.email}?s=256&d=monsterid`"
            class="avatar w-[44px] h-[44px] rounded-[5px] [transition:all_0.5s] border border-[#ddd] hover:rounded-[30%_70%_70%_30%/30%_30%_70%_70%]">
        </div>
        <div class="comment-main ml-[15px] text-[14px] w-full">
          <div class="comment-main-top h-[44px] flex flex-col justify-between">
            <div class="comment-meta flex items-baseline text-[#666]">
              <div class="comment-author"><a>{{ comment.author }}</a></div>
              <span v-if="comment.isAdmin" class="admin-tag text-white bg-[#6b7280] py-px px-[3px] text-[10px] leading-[1.1] font-medium rounded-[3px] inline-block opacity-90 mr-[3px] ml-[2px]">博主</span>
              <time class="comment-time text-[#6b7280] text-[10px] ml-px"> • {{ formatDate(comment.createTime) }}</time>
            </div>
            <div class="comment-content text-[rgb(74,85,104)]">
              <p>{{ comment.content }}</p>
            </div>
          </div>

          <div class="reply-button mt-4 text-[12px]">
            <span class="inline-block cursor-pointer" :class="{ 'reply-enter': replay == comment.id }" @click="replyComment(comment.id)">回复</span>

            <Transition>
              <div class="reply-edit mt-4" v-if="replay == comment.id">
                <Publish @comment-submitted="send" :parentId="comment.id" />
              </div>
            </Transition>
          </div>
        </div>
      </div>
      <ol class="children list-none pl-6" v-if="comment.children && comment.children.length > 0">
        <CommentItem :commentList="comment.children" />
      </ol>
    </li>
  </TransitionGroup>
</template>

<style lang="less" scoped>
/*
  只保留三类工具类表达不了的：

  1. Vue <transition> 运行时类 + 它引用的 keyframes（Vue 会把两者一起改名，
     拆开就会静默失效，§4.1）。
  2. `.reply-enter` —— 由 `:class="{ 'reply-enter': replay == comment.id }"` 在运行时切换，
     模板里写不出固定的 class 串（§4.3.1 同类）。
  3. 无（其余全部内联）。

  注意一个容易误判的点：原 CSS 里绝大多数规则嵌在 `ul { ... }` 之下，而本组件模板里
  **没有** `ul`。但 Vue 只把 scope 属性加在**最后一个**复合选择器上，编译结果是
  `ul .comment-box .comment-container[data-v-x]` —— 那个 `ul` 是**父组件**（Comment/View）
  提供的，所以这些规则一直是生效的。内联后反而不再依赖祖先 `ul`，更稳。

  另：原 `.comment-list` 规则在本组件模板里没有任何对应元素（与 Comment/View 的类名重名
  但实现不同），是死样式，已删除。
*/
.v-enter-active {
  animation: bottom .6s ease;
}

.v-leave-active {
  animation: bottom .5s ease reverse;
}

@keyframes bottom {
  0% {
    opacity: 0;
    transform: translateY(30px);
  }

  100% {
    opacity: 1;
    transform: translateY(0);
  }
}

.reply-enter {
  color: rgb(31, 109, 218);
  font-weight: bold;
}
</style>

<script setup>
import { defineProps } from 'vue'
import { formatDate } from '@/utils/tool'
import { addComment } from '@/api/user.ts'
import CommentItem from '@/components/frontend/Comment/CommentItem.vue';
import Publish from '@/components/frontend/Comment/Publish.vue';
import { useUserStore } from '@/store';
const store = useUserStore()
const replay = ref(-1)

const replyComment = (commentId) => {
  if (replay.value == commentId) {
    replay.value = -1
  } else {
    replay.value = commentId
  }
}

const send = (comment) => {
  addComment(comment)
  store.commentKey = !store.commentKey
}

defineProps({
  commentList: {
    type: Array,
    required: true
  }
})
</script>
