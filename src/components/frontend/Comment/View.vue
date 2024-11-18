<template>
  <div class="comment-list">
    <div class="count">{{ commentList.flat(Infinity).length }} 条评论</div>
    <ul>
      <CommentItem :commentList="commentList" />
    </ul>
  </div>
</template>

<style lang="less" scoped>
.comment-list {
  width: 100%;
  margin: 30px auto;
  color: rgb(49, 49, 49);
}

ul {
  margin: 24px 0;
  width: 100%;
}
</style>

<script setup>
import { reactive } from 'vue';
import CommentItem from '@/components/frontend/Comment/CommentItem.vue';
import { formatDate } from '@/utils/tool'
import { getCommentList } from '@/api/user'
import { useUserStore } from '@/store/index';
const store = useUserStore()

// const commentList = reactive([
//   {
//     "id": "1",
//     "author": "Alice",
//     "email": "alice@example.com",
//     "content": "This is a great post!",
//     "create_time": "2023-10-01T12:34:56Z",
//     "parentId": null,
//     "children": []
//   },
//   {
//     "id": "2",
//     "author": "Bob",
//     "email": "bob@example.com",
//     "content": "I agree with Alice.",
//     "create_time": "2023-10-01T12:35:56Z",
//     "parentId": "1",
//     "children": [
//       {
//         "id": "3",
//         "author": "Charlie",
//         "email": "charlie@example.com",
//         "content": "I disagree with both of you.",
//         "create_time": "2023-10-01T12:36:56Z",
//         "parentId": "2",
//         "children": [
//           {
//             "id": "32",
//             "author": "Charlie",
//             "email": "cha2rlie@example.com",
//             "content": "哦，我已失去了你！",
//             "create_time": "2023-10-01T12:36:56Z",
//             "admin": true,
//             "parentId": "2",
//             "children": []
//           }
//         ]
//       }
//     ]
//   },
//   {
//     "id": "4",
//     "author": "David",
//     "email": "david@example.com",
//     "content": "Interesting discussion.",
//     "create_time": "2023-10-01T12:37:56Z",
//     "parentId": null,
//     "children": [
//       {
//         "id": "5",
//         "author": "Eve",
//         "email": "eve@example.com",
//         "content": "I think David is right.",
//         "create_time": "2023-10-01T12:38:56Z",
//         "parentId": "4",
//         "children": []
//       },
//       {
//         "id": "6",
//         "author": "Frank",
//         "email": "frank@example.com",
//         "content": "I disagree with David.",
//         "create_time": "2023-10-01T12:39:56Z",
//         "parentId": "4",
//         "children": []
//       }
//     ]
//   },
//   {
//     "id": "7",
//     "author": "Grace",
//     "email": "grace@example.com",
//     "content": "This is a private comment.",
//     "create_time": "2023-10-01T12:40:56Z",
//     "parentId": null,
//     "children": []
//   },
//   {
//     "id": "8",
//     "author": "Henry",
//     "email": "henry@example.com",
//     "content": "Another comment here.",
//     "create_time": "2023-10-01T12:41:56Z",
//     "parentId": null,
//     "children": [
//       {
//         "id": "9",
//         "author": "Ivy",
//         "email": "ivy@example.com",
//         "content": "Reply to Henry.",
//         "create_time": "2023-10-01T12:42:56Z",
//         "parentId": "8",
//         "children": []
//       }
//     ]
//   }
// ]);
const commentList = ref([]);
onMounted(() => {
  setTimeout(async () => {
    const data = await getCommentList(store.homeHeaderInfo.id);
    commentList.value = data.list
  }, 1000)

})
</script>