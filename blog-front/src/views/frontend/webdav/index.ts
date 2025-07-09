// 主视图组件
import WebDriveView from './components/WebDriveView.vue'
import MobileView from './components/MobileView.vue'

// 模态框组件
import SettingsModal from './modals/SettingsModal.vue'
import ShareLinkPopup from './modals/ShareLinkPopup.vue'
import UploadModal from './modals/UploadModal.vue'

// 工具和类型
import * as icons from './utils/icons'
import type { FileItem } from './utils/types/file'

// 分组导出
export {
  // 主视图组件
  WebDriveView,
  MobileView,
  
  // 模态框组件
  SettingsModal,
  ShareLinkPopup,
  UploadModal,
  
  // 工具和类型
  icons,
  type FileItem
}

// 默认导出实用组件
export default WebDriveView 