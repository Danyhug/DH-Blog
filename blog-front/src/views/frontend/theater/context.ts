import { inject, provide, type InjectionKey, type Ref } from 'vue'

/** 影院外壳提供给各页面的公共能力：打开媒体库设置、感知设置变更后重新加载 */
interface TheaterContext {
  openSettings: () => void
  /** 每次保存媒体库设置后自增，页面 watch 它来重新拉取曲库/片库 */
  libraryVersion: Ref<number>
  /** 背景氛围图（会被大幅模糊），由当前页面设置：影视页用推荐作品，音乐页用正在播放的封面 */
  ambient: Ref<string>
}

const key: InjectionKey<TheaterContext> = Symbol('theater')

export function provideTheater(context: TheaterContext) {
  provide(key, context)
}

export function useTheater(): TheaterContext {
  const context = inject(key)
  if (!context) throw new Error('useTheater() 只能在影院页面内使用')
  return context
}
