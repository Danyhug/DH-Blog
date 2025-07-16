import type { Component } from 'vue'
import type { FileInfo } from '@/api/file'

export interface FileItem {
  id?: string;
  name: string;
  type: 'file' | 'folder' | 'image' | 'video' | 'audio' | 'code' | 'pdf' | 'archive' | 'spreadsheet' | 'presentation' | 'text';
  size: string;
  modified?: string;
  icon?: Component;
  originalFile?: FileInfo;
} 