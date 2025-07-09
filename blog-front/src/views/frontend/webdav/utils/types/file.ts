import type { Component } from 'vue'

export interface FileItem {
  name: string
  type: 'file' | 'folder' | 'image' | 'video' | 'audio'
  size: string
  modified?: string
  icon?: Component
} 