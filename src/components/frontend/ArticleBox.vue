<template>
  <article class="type-img-left">
    <router-link :to="'./article/' + article.id">
      <div class="cover">
        <div class="left">
          <img :src="getArticleBg(article.thumbnailUrl)" :alt="article.title">
        </div>
        <div class="right">
          <div class="top">
            <span class="date">
              <Icon iconName="icon-calendar" iconSize="1.3"></Icon>
              {{ article.publishDate?.slice(0, 10) }}
            </span>
            <span class="num-word">
              <Icon iconName="icon-image-text" iconSize="1.3"></Icon>
              {{ article.wordNum }} 字
            </span>
            <span class="time-consum">
              <Icon iconName="icon-browse" iconSize="1.3"></Icon>
              <!-- {{ article.wordNum ? (article.wordNum / 200 + 0.5).toFixed(0) : 0 }} 分钟 -->
                {{ article.views }}
            </span>
          </div>
          <p class="title">
            <a href="">{{ article.title }}</a>
          </p>
          <p class="text">{{ article.content }}</p>
          <div class="bottom">
            <div class="bottom-tags">
              {{ article.tags?.map(item => item.name).join('&nbsp;&nbsp;|&nbsp;&nbsp;') }}
            </div>
            <a href="" class="more">more...</a>
          </div>
        </div>
      </div>
    </router-link>
  </article>
</template>

<script lang="ts" setup>
import { defineProps } from 'vue'
import { Article } from '@/types/Article.ts'
import { Tag } from '@/types/Tag'
import { getArticleBg } from '@/utils/tool'
const props = defineProps(['article'])
const article: Article<Tag> = props.article
console.log(article)
</script>

<style lang="less" scoped>
.cover {
  display: flex;
  width: 100%;
  height: 244px;
  border-radius: 1rem;
  background: var(--grey-0);
  overflow: hidden;
  border: 1px solid #eee;

  div {
    width: 50%;
  }

  .left {
    position: relative;
    overflow: hidden;
    clip-path: polygon(0 0, 92% 0%, 100% 100%, 0% 100%);

    img {
      display: block;
      width: 100%;
      height: 100%;
      object-fit: cover;
      transition: all .5s;
    }
  }

  .right {
    position: relative;
    padding: 1rem;
    padding-top: .3rem;

    div {
      width: 100%;
    }

    .top {
      display: flex;
      justify-content: flex-end;
      font-size: .75rem;
      color: #606266;

      span {
        margin-left: 1.25rem;
      }
    }

    .title {
      margin: 26px 0 16px;
      // text-align: center;
      font-weight: 700;

      a {
        font-size: 1.5rem;
        color: rgb(233, 84, 107);
      }
    }

    // 正文描述
    .text {
      font-size: .875rem;
      line-height: 2;
      max-height: 128px;
      display: -webkit-box;
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 3;
      text-overflow: ellipsis;
      overflow: hidden;
    }

    .bottom {
      position: absolute;
      left: 0;
      bottom: 0;
      display: flex;
      justify-content: space-between;
      line-height: 2.625rem;

      .bottom-tags {
        padding-left: 16px;
        padding-right: 2rem;
        font-size: .875rem;
        color: var(--grey-5);
      }

      .more {
        width: 6rem;
        height: 2.625rem;
        line-height: 2.625rem;
        text-align: center;
        color: rgba(255, 255, 255, .6);
        border-radius: 1rem 0;
        background-image: linear-gradient(to right, var(--color-pink) 0, var(--color-orange) 100%);
        transition: all .5s;
      }
    }
  }

  &:hover .more {
    color: #fff !important;
    // transform: translateX(-7px);
    // border-radius: 5px!important;
  }

  &:hover .left img {
    transform: scale(1.05);
  }
}
</style>