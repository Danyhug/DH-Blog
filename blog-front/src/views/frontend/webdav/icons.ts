import type { Component } from 'vue'
import { defineComponent, h } from 'vue'

// 定义所有图标组件，使用 h 函数渲染 SVG 而不是模板字符串
export const HomeIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'm3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z' }),
      h('polyline', { points: '9,22 9,12 15,12 15,22' })
    ])
  }
})

export const ChevronRightIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'm9 18 6-6-6-6' })
    ])
  }
})

export const Grid3X3Icon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('rect', { width: '7', height: '7', x: '3', y: '3', rx: '1' }),
      h('rect', { width: '7', height: '7', x: '14', y: '3', rx: '1' }),
      h('rect', { width: '7', height: '7', x: '3', y: '14', rx: '1' }),
      h('rect', { width: '7', height: '7', x: '14', y: '14', rx: '1' })
    ])
  }
})

export const ListIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('line', { x1: '8', x2: '21', y1: '6', y2: '6' }),
      h('line', { x1: '8', x2: '21', y1: '12', y2: '12' }),
      h('line', { x1: '8', x2: '21', y1: '18', y2: '18' }),
      h('line', { x1: '3', x2: '3.01', y1: '6', y2: '6' }),
      h('line', { x1: '3', x2: '3.01', y1: '12', y2: '12' }),
      h('line', { x1: '3', x2: '3.01', y1: '18', y2: '18' })
    ])
  }
})

export const SettingsIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z' }),
      h('circle', { cx: '12', cy: '12', r: '3' })
    ])
  }
})

export const PlusIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M5 12h14' }),
      h('path', { d: 'm12 5v14' })
    ])
  }
})

export const UploadIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4' }),
      h('polyline', { points: '17,8 12,3 7,8' }),
      h('line', { x1: '12', x2: '12', y1: '3', y2: '15' })
    ])
  }
})

export const SearchIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('circle', { cx: '11', cy: '11', r: '8' }),
      h('path', { d: 'm21 21-4.35-4.35' })
    ])
  }
})

export const FolderIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z' })
    ])
  }
})

export const FileIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z' }),
      h('path', { d: 'M14 2v4a2 2 0 0 0 2 2h4' })
    ])
  }
})

export const FileTextIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z' }),
      h('path', { d: 'M14 2v4a2 2 0 0 0 2 2h4' }),
      h('line', { x1: '10', x2: '8', y1: '9', y2: '9' }),
      h('line', { x1: '16', x2: '8', y1: '13', y2: '13' }),
      h('line', { x1: '16', x2: '8', y1: '17', y2: '17' })
    ])
  }
})

export const ImageIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('rect', { width: '18', height: '18', x: '3', y: '3', rx: '2', ry: '2' }),
      h('circle', { cx: '9', cy: '9', r: '2' }),
      h('path', { d: 'm21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21' })
    ])
  }
})

export const VideoIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'm22 8-6 4 6 4V8Z' }),
      h('rect', { width: '14', height: '12', x: '2', y: '6', rx: '2', ry: '2' })
    ])
  }
})

export const MusicIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M9 18V5l12-2v13' }),
      h('circle', { cx: '6', cy: '18', r: '3' }),
      h('circle', { cx: '18', cy: '16', r: '3' })
    ])
  }
})

export const MonitorIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('rect', { width: '20', height: '14', x: '2', y: '3', rx: '2' }),
      h('line', { x1: '8', x2: '16', y1: '21', y2: '21' }),
      h('line', { x1: '12', x2: '12', y1: '17', y2: '21' })
    ])
  }
})

export const SmartphoneIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('rect', { width: '14', height: '20', x: '5', y: '2', rx: '2', ry: '2' }),
      h('line', { x1: '12', x2: '12.01', y1: '18', y2: '18' })
    ])
  }
})

export const ServerIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('rect', { width: '20', height: '8', x: '2', y: '2', rx: '2', ry: '2' }),
      h('rect', { width: '20', height: '8', x: '2', y: '14', rx: '2', ry: '2' }),
      h('line', { x1: '6', x2: '6.01', y1: '6', y2: '6' }),
      h('line', { x1: '6', x2: '6.01', y1: '18', y2: '18' })
    ])
  }
})

export const CloudIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z' })
    ])
  }
})

export const XIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M18 6 6 18' }),
      h('path', { d: 'm6 6 12 12' })
    ])
  }
})

export const CheckIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('path', { d: 'M20 6 9 17l-5-5' })
    ])
  }
})

export const MoreHorizontalIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('circle', { cx: '12', cy: '12', r: '1' }),
      h('circle', { cx: '19', cy: '12', r: '1' }),
      h('circle', { cx: '5', cy: '12', r: '1' })
    ])
  }
})

export const StarIcon = defineComponent({
  render() {
    return h('svg', {
      viewBox: '0 0 24 24',
      fill: 'none',
      stroke: 'currentColor',
      'stroke-width': '2'
    }, [
      h('polygon', { points: '12,2 15.09,8.26 22,9.27 17,14.14 18.18,21.02 12,17.77 5.82,21.02 7,14.14 2,9.27 8.91,8.26' })
    ])
  }
})

// 导出所有图标类型
export type IconComponent = typeof HomeIcon
