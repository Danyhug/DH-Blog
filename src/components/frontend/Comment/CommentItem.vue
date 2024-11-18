<template>
  <li v-for="comment in commentList" :key="comment.id" class="comment-box">
    <div class="comment-container">
      <div class="comment-avatar">
        <img :alt="`${comment.author}'s avatar`" :src="`//cravatar.cn/avatar/${comment.email}?s=256&d=monsterid`"
          class="avatar">
      </div>
      <div class="comment-main">
        <div class="comment-main-top">
          <div class="comment-meta">
            <div class="comment-author"><a>{{ comment.author }}</a></div>
            <span v-if="comment.admin" class="admin-tag">博主</span>
            <time class="comment-time"> • {{ formatDate(comment.create_time) }}</time>
          </div>
          <div class="comment-content">
            <p>{{ comment.content }}</p>
          </div>
        </div>

        <div class="reply-button">
          <span :class="{ 'reply-enter': replay == comment.id }" @click="replyComment(comment.id)">回复</span>

          <Transition>
            <div class="reply-edit" v-if="replay == comment.id">
              <Publish />
            </div>
          </Transition>
        </div>
      </div>
    </div>
    <ol class="children" v-if="comment.children && comment.children.length > 0">
      <CommentItem :commentList="comment.children" />
    </ol>
  </li>
</template>

<style lang="less" scoped>
.v-enter-active {
  animation: bottom .5s ease;
}

.v-leave-active {
  animation: bottom .5s ease reverse;
}

@keyframes bottom {
  0% {
    opacity: 0;
    transform: translateY(20px);
  }

  100% {
    opacity: 1;
    -webkit-transform: translateY(0);
    transform: translateY(0);
  }
}

.admin-tag {
  margin: 0 3px 0 2px;
  color: #fff;
  background: #6b7280;
  padding: 1px 3px;
  font-size: 10px;
  line-height: 1.1;
  font-weight: 500;
  border-radius: 3px;
  display: inline-block;
  opacity: .9;
}

ul,
li,
ol {
  list-style: none;
}

.comment-time {
  color: #6b7280;
  font-size: 10px;
  margin-left: 1px;
}

.comment-list {
  width: 100%;
  margin: 30px auto;
  color: rgb(49, 49, 49);
}

ul {
  margin: 24px 0;
  width: 100%;

  .comment-box {
    .comment-container {
      display: flex;
      margin-bottom: 20px;

      .comment-avatar {
        cursor: pointer;

        img {
          width: 44px;
          height: 44px;
          border-radius: 5px;
          transition: all .5s;

          border: 1px solid #ddd;

          &:hover {
            border-radius: 30% 70% 70% 30% / 30% 30% 70% 70%;
          }
        }
      }

      .comment-main {
        margin-left: 15px;
        font-size: 14px;
        width: 100%;

        .comment-main-top {
          height: 44px;
          display: flex;
          flex-direction: column;
          justify-content: space-between;
        }

        .comment-meta {
          display: flex;
          align-items: baseline;
          color: #666;
        }

        .comment-content {
          color: rgb(74, 85, 104);
        }

        .reply-button {
          margin-top: 16px;
          font-size: 12px;

          span {
            display: inline-block;
            cursor: pointer;
          }
        }

        .reply-enter {
          color: rgb(31, 109, 218);
          font-weight: bold;
        }

        .reply-edit {
          margin-top: 16px;
        }
      }
    }

    .children {
      padding-left: 24px;
    }
  }
}
</style>

<script setup>
import { defineProps } from 'vue'
import { formatDate } from '@/utils/tool'
import CommentItem from '@/components/frontend/Comment/CommentItem.vue';
import Publish from '@/components/frontend/Comment/Publish.vue';

const replay = ref(-1)

const replyComment = (commentId) => {
  if (replay.value == commentId) {
    replay.value = -1
  } else {
    replay.value = commentId
  }
}

defineProps({
  commentList: {
    type: Array,
    required: true
  }
})
</script>