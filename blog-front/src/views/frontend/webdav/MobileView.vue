<template>
  <div class="mobile-container">
    <div class="mobile-screen">
      <!-- Mobile Header -->
      <div class="mobile-header">
        <div class="header-content">
          <h1 class="mobile-title">My Drive</h1>
          <div class="header-actions">
            <button class="mobile-icon-btn">
              <SearchIcon class="icon-sm" />
            </button>
            <button class="mobile-icon-btn">
              <MoreHorizontalIcon class="icon-sm" />
            </button>
          </div>
        </div>
      </div>

      <!-- Mobile Content -->
      <div class="mobile-content">
        <div
          v-for="(file, index) in mobileFiles"
          :key="index"
          class="mobile-file-item"
        >
          <div class="mobile-file-icon">
            <FolderIcon v-if="file.type === 'folder'" class="folder-icon-mobile" />
            <FileIcon v-else class="file-icon-mobile" />
          </div>
          <div class="mobile-file-info">
            <p class="mobile-file-name">{{ file.name }}</p>
            <p class="mobile-file-size">{{ file.size }}</p>
          </div>
          <button class="mobile-more-btn">
            <MoreHorizontalIcon class="icon-sm" />
          </button>
        </div>
      </div>

      <!-- Mobile Bottom Navigation -->
      <div class="mobile-nav">
        <div class="nav-content">
          <button class="nav-btn active">
            <HomeIcon class="nav-icon" />
            <span class="nav-label">Home</span>
          </button>
          <button class="nav-btn">
            <StarIcon class="nav-icon" />
            <span class="nav-label">Starred</span>
          </button>
          <button class="nav-btn">
            <CloudIcon class="nav-icon" />
            <span class="nav-label">Sync</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FileItem } from './types/file'
import {
  SearchIcon,
  MoreHorizontalIcon,
  FolderIcon,
  FileIcon,
  HomeIcon,
  StarIcon,
  CloudIcon
} from './icons'

interface Props {
  mobileFiles: FileItem[]
}

defineProps<Props>()
</script>

<style scoped>
.mobile-container {
  position: absolute;
  bottom: 4rem;
  right: 4rem;
  width: 280px;
  height: 500px;
  background: #1f2937;
  border-radius: 2.5rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  z-index: 20;
  padding: 0.5rem;
}

.mobile-screen {
  width: 100%;
  height: 100%;
  background: white;
  border-radius: 2rem;
  overflow: hidden;
  position: relative;
}

.mobile-header {
  background: white;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #f3f4f6;
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.mobile-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.mobile-icon-btn {
  background: none;
  border: none;
  padding: 0.5rem;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.mobile-icon-btn:hover {
  background: rgba(156, 163, 175, 0.1);
}

.mobile-content {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  height: calc(100% - 120px);
  overflow-y: auto;
}

.mobile-file-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  border-radius: 0.5rem;
  transition: background-color 0.2s;
  cursor: pointer;
}

.mobile-file-item:hover {
  background: #f9fafb;
}

.mobile-file-icon {
  width: 2.5rem;
  height: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.folder-icon-mobile {
  width: 2rem;
  height: 2rem;
  color: #2563eb;
  fill: currentColor;
}

.file-icon-mobile {
  width: 2rem;
  height: 2rem;
  color: #6b7280;
}

.mobile-file-info {
  flex: 1;
}

.mobile-file-name {
  font-size: 0.875rem;
  font-weight: 500;
  color: #111827;
  margin: 0;
}

.mobile-file-size {
  font-size: 0.75rem;
  color: #6b7280;
  margin: 0.25rem 0 0 0;
}

.mobile-more-btn {
  background: none;
  border: none;
  padding: 0.5rem;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.mobile-more-btn:hover {
  background: rgba(156, 163, 175, 0.1);
}

.mobile-nav {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: white;
  border-top: 1px solid #f3f4f6;
  padding: 0.5rem 1rem;
}

.nav-content {
  display: flex;
  align-items: center;
  justify-content: space-around;
}

.nav-btn {
  background: none;
  border: none;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  padding: 0.5rem;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.nav-btn:hover {
  background: rgba(156, 163, 175, 0.1);
}

.nav-icon {
  width: 1rem;
  height: 1rem;
  color: #9ca3af;
}

.nav-btn.active .nav-icon {
  color: #2563eb;
}

.nav-label {
  font-size: 0.75rem;
  color: #9ca3af;
}

.nav-btn.active .nav-label {
  color: #2563eb;
}

.icon-sm {
  width: 1rem;
  height: 1rem;
}
</style>
