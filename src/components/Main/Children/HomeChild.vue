<template>
    <div>
        <!-- 这里是文章列表 -->
        <div class="tip">
            <p>文章列表</p>
        </div>
        <div class="posts">
            <Article v-for="item in articleList" :id="item.ID" :title="item.Title" :content="item.Content"
                :created="item.Created"></Article>
        </div>
        <Pagination :total="Math.ceil(this.pageSize / this.limit)" :nowPage="this.$route.params.id"></Pagination>
    </div>
</template>
<script>
import Article from '@/components/Article/Article.vue'
import Pagination from '@/components/Pagination/Pagination.vue'
export default {
    name: 'HomeChild',
    components: {
        Article,
        Pagination
    },

    watch: {
        $route(to, from) {
            console.log('路由变化了')
            this.getArticleList()
        }
    },
    data() {
        return {
            // 文章列表
            articleList: [],
            pageSize: 0,
            start: 0,
            limit: 6
        }
    },
    methods: {
        async getArticleList() {
            let page_id = this.$route.params.id
            // 获取起始页
            this.start = page_id * this.limit - this.limit

            let data = {
                start: this.start,
                limit: this.limit
            }
            let res = await this.$http.post('article/query', data)
            this.articleList = res.data.data
            console.log(this.articleList)
        },
        async getPageSize() {
            let res = await this.$http.post('article/pageSize')
            this.pageSize = res.data.size
        }
    },
    created() {
        this.getPageSize()
    },
    mounted() {
        this.getArticleList()
    }
}
</script>
<style lang="less" scoped>
.tip {
    display: block;
    width: 100%;
    text-align: center;
    font-size: 24px;
    font-weight: 700;
    border-top: 1px solid var(--grey-4);

    p {
        display: inline-block;
        transform: translateY(-50%);
        background-color: #fff;
        padding: 0 16px;
        color: var(--grey-4);
    }
}

/* 文章父元素 */
.posts {
    padding: 0 30px;
}

Article {
    margin-bottom: 40px;
}
</style>