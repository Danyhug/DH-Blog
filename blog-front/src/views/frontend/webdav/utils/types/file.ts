import type { Component } from 'vue'
import type { FileInfo } from '@/api/file'

export interface FileItem {
  id?: string;
  name: string;
  type: 'file' | 'folder' | 'image' | 'video' | 'audio' | 'code' | 'pdf' | 'archive' | 'spreadsheet' | 'presentation' | 'text' | 'csv';
  size: string;
  modified?: string;
  icon?: Component;
  originalFile?: FileInfo;
  /** 全盘搜索命中时的所在目录展示路径；普通目录列表下为 undefined */
  parentPath?: string;
  /** 全盘搜索命中时从根到所在目录的链路，用于跳转过去 */
  parentSegments?: PathSegment[];
} 

/** 网盘路径面包屑的一段（列表页与预览页共用） */
export interface PathSegment {
  id: string;
  name: string;
}
