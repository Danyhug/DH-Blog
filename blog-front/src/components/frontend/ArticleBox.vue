<template>
  <article class="type-img-left">
    <router-link :to="'./article/' + article.id">
      <div class="cover">
        <div class="left">
          <img :src="getArticleBg(article.thumbnailUrl, article.id)" :alt="article.title" loading="lazy">
        </div>
        <div class="right">
          <div class="top">
            <span class="date">
              <Icon iconName="icon-calendar" iconSize="1.35"></Icon>
              {{ article.createTime?.slice(0, 10) }}
            </span>
            <span class="num-word">
              <Icon iconName="icon-image-text" iconSize="1.35"></Icon>
              {{ article.wordNum }} 字
            </span>
            <span class="time-consum">
              <Icon iconName="icon-browse" iconSize="1.35"></Icon>
              {{ article.views }} 次
            </span>
          </div>
          <p class="title">
            <span class="title-link">{{ article.title }}</span>
          </p>
          <div v-if="article.isLocked && !article.canAccess" class="private-summary" role="note" aria-label="私密文章提示">
            <span class="private-lock" aria-hidden="true">
              <el-icon><Lock /></el-icon>
            </span>
            <span class="private-copy">
              <span class="private-label">PRIVATE ENTRY</span>
              <strong>这篇文章已上锁</strong>
              <span class="private-hint">正文需使用密钥解锁后阅读</span>
            </span>
          </div>
          <p v-else class="text">{{ article.content }}</p>
          <div class="bottom">
            <span class="more">{{ article.isLocked && !article.canAccess ? '解锁...' : 'more...' }}</span>
          </div>
        </div>
      </div>
    </router-link>
  </article>
</template>

<script lang="ts" setup>
import { Article } from '@/types/Article.ts'
import { Tag } from '@/types/Tag'
import { getArticleBg } from '@/utils/tool'
const props = defineProps(['article'])
const article: Article<Tag> = props.article
</script>

<style lang="less" scoped>
.cover {
  display: flex;
  width: 100%;
  min-height: 244px;
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
    display: flex;
    flex-direction: column;
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
        margin-left: 1.45rem;

        .icon {
          transform: translateY(-1px);
        }
      }
    }

    .title {
      margin: 26px 0 16px;
      // text-align: center;
      font-weight: 700;

      .title-link {
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
      line-clamp: 3;
      text-overflow: ellipsis;
      overflow: hidden;
    }

    .private-summary {
      position: relative;
      display: flex;
      align-items: center;
      gap: .875rem;
      min-height: 82px;
      margin: .2rem 0 1rem;
      padding: .8rem 1rem;
      overflow: hidden;
      border: 1px dashed var(--color-pink-a3);
      border-radius: .75rem;
      background: linear-gradient(135deg, var(--color-red-a1), rgba(236, 140, 105, .08));

      &::after {
        position: absolute;
        right: -1.3rem;
        bottom: -2.2rem;
        width: 5.5rem;
        height: 5.5rem;
        content: '';
        border: 1px solid rgba(233, 84, 107, .12);
        border-radius: 50%;
      }

      .private-lock {
        display: inline-flex;
        flex: 0 0 2.65rem;
        align-items: center;
        justify-content: center;
        width: 2.65rem;
        height: 2.65rem;
        color: #fff;
        border-radius: 50%;
        background: linear-gradient(135deg, var(--color-pink), var(--color-orange));
        box-shadow: 0 6px 16px rgba(233, 84, 107, .22);

        .el-icon {
          font-size: 1.2rem;
        }
      }

      .private-copy {
        display: flex;
        flex-direction: column;
        min-width: 0;
        color: var(--grey-7);
        line-height: 1.35;

        .private-label {
          margin-bottom: .15rem;
          color: var(--color-red);
          font-size: .625rem;
          font-weight: 700;
          letter-spacing: .14em;
        }

        strong {
          font-size: .95rem;
        }

        .private-hint {
          margin-top: .2rem;
          color: var(--grey-6);
          font-size: .75rem;
        }
      }
    }

    .bottom {
      display: flex;
      align-items: flex-end;
      justify-content: flex-end;
      margin-top: auto;

      .more {
        flex: 0 0 6rem;
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

@media screen and (max-width: 768px) {
  .cover {
    height: 244px;
    min-height: 244px;

    .private-summary {
      min-height: auto;
      margin-bottom: 0;
      padding: .65rem;

      .private-hint {
        display: none;
      }
    }
  }
}
</style>
