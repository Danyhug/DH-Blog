import type { Component } from 'vue'
import {
  FolderIcon,
  ImageIcon,
  VideoIcon,
  MusicIcon,
  FilePdfIcon,
  FileZipIcon,
  FileSpreadsheetIcon,
  FilePresentationIcon,
  FileCodeIcon,
  FileTextIcon
} from './icons'
import type { FileItem } from './types/file'

export type FileType = FileItem['type']

// 浏览器可直接解码的图片格式；heic/tiff 之类交给下载，避免出现空白预览
const IMAGE_EXTS = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg', 'ico', 'avif', 'apng', 'jfif']
const VIDEO_EXTS = ['mp4', 'webm', 'ogv', 'mov', 'm4v', 'avi', 'wmv', 'flv', 'mkv', 'mpeg', 'mpg', '3gp']
const AUDIO_EXTS = ['mp3', 'wav', 'ogg', 'oga', 'opus', 'weba', 'flac', 'aac', 'm4a', 'm4b', 'wma', 'aiff']
const ARCHIVE_EXTS = ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz', 'zst', 'tgz']
const SPREADSHEET_EXTS = ['xls', 'xlsx', 'ods', 'numbers']
const PRESENTATION_EXTS = ['ppt', 'pptx', 'odp', 'key']
// 表格类文本，单独成类以便渲染成表格
const CSV_EXTS = ['csv', 'tsv']

// 代码类文本（用等宽 + 高亮渲染）
const CODE_EXTS = [
  'js', 'mjs', 'cjs', 'jsx', 'ts', 'mts', 'cts', 'tsx', 'vue', 'svelte',
  'html', 'htm', 'xhtml', 'css', 'scss', 'sass', 'less', 'styl',
  'xml', 'json', 'json5', 'jsonc', 'yaml', 'yml', 'toml',
  'py', 'java', 'kt', 'kts', 'scala', 'groovy', 'c', 'h', 'cpp', 'cc', 'cxx', 'hpp', 'hxx',
  'cs', 'go', 'rs', 'swift', 'dart', 'php', 'rb', 'lua', 'pl', 'pm', 'r', 'jl', 'ex', 'exs',
  'sh', 'bash', 'zsh', 'fish', 'bat', 'cmd', 'ps1', 'sql', 'graphql', 'gql', 'proto',
  'tf', 'tfvars', 'gradle', 'cmake', 'mk', 'dockerfile', 'patch', 'diff'
]

// 纯文本（无高亮，直接按 pre 展示）
const PLAIN_TEXT_EXTS = [
  'txt', 'text', 'log', 'rtf', 'md', 'markdown', 'ini', 'conf', 'config', 'cfg', 'env',
  'properties', 'lock', 'srt', 'vtt', 'ass', 'lrc', 'csr', 'pem', 'gitignore', 'gitattributes',
  'editorconfig', 'npmrc', 'nvmrc', 'babelrc', 'eslintrc', 'prettierrc'
]

// 没有扩展名但内容是文本的常见文件名
const TEXT_FILENAMES = [
  'dockerfile', 'makefile', 'jenkinsfile', 'procfile', 'license', 'licence',
  'readme', 'changelog', 'authors', 'contributing', 'notice', 'codeowners', 'go.sum'
]

const ICON_BY_TYPE: Record<FileType, Component> = {
  folder: FolderIcon,
  image: ImageIcon,
  video: VideoIcon,
  audio: MusicIcon,
  pdf: FilePdfIcon,
  archive: FileZipIcon,
  spreadsheet: FileSpreadsheetIcon,
  presentation: FilePresentationIcon,
  csv: FileSpreadsheetIcon,
  code: FileCodeIcon,
  text: FileTextIcon,
  file: FileTextIcon
}

/** 前端能直接渲染的类型；其余一律走「无法预览 + 下载」分支 */
const PREVIEWABLE_TYPES: FileType[] = ['image', 'video', 'audio', 'pdf', 'text', 'csv']

export function isPreviewable(type: string): boolean {
  return PREVIEWABLE_TYPES.includes(type as FileType)
}

export function getExtension(fileName: string): string {
  const name = fileName.toLowerCase()
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(dot + 1) : ''
}

/**
 * 判断文件类型。mimeType 只在扩展名无法判定时兜底 —— 服务端对未知后缀常返回
 * application/octet-stream，扩展名反而更可靠。
 */
export function detectFileType(fileName: string, mimeType?: string, isFolder = false): FileType {
  if (isFolder) return 'folder'

  const ext = getExtension(fileName)
  if (ext) {
    if (IMAGE_EXTS.includes(ext)) return 'image'
    if (VIDEO_EXTS.includes(ext)) return 'video'
    if (AUDIO_EXTS.includes(ext)) return 'audio'
    if (ext === 'pdf') return 'pdf'
    if (CSV_EXTS.includes(ext)) return 'csv'
    if (ARCHIVE_EXTS.includes(ext)) return 'archive'
    if (SPREADSHEET_EXTS.includes(ext)) return 'spreadsheet'
    if (PRESENTATION_EXTS.includes(ext)) return 'presentation'
    if (CODE_EXTS.includes(ext) || PLAIN_TEXT_EXTS.includes(ext)) return 'text'
  } else if (TEXT_FILENAMES.includes(fileName.toLowerCase())) {
    return 'text'
  }

  if (mimeType) {
    if (mimeType.startsWith('image/')) return 'image'
    if (mimeType.startsWith('video/')) return 'video'
    if (mimeType.startsWith('audio/')) return 'audio'
    if (mimeType.startsWith('application/pdf')) return 'pdf'
    if (mimeType.includes('csv')) return 'csv'
    if (
      mimeType.includes('zip') || mimeType.includes('compressed') ||
      mimeType.includes('archive') || mimeType.includes('x-tar') || mimeType.includes('x-rar')
    ) return 'archive'
    if (mimeType.includes('excel') || mimeType.includes('spreadsheet')) return 'spreadsheet'
    if (mimeType.includes('powerpoint') || mimeType.includes('presentation')) return 'presentation'
    if (
      mimeType.startsWith('text/') || mimeType.includes('javascript') ||
      mimeType.includes('json') || mimeType.includes('xml') || mimeType.includes('yaml') ||
      mimeType.includes('x-sh') || mimeType.includes('x-httpd-php')
    ) return 'text'
  }

  return 'file'
}

/** 列表里展示用的图标；text 类型再细分代码/纯文本两种图标 */
export function getFileIcon(fileName: string, type: FileType): Component {
  if (type === 'text' && CODE_EXTS.includes(getExtension(fileName))) return FileCodeIcon
  return ICON_BY_TYPE[type] || FileTextIcon
}

// 扩展名 → highlight.js 语言标识
const LANG_MAP: Record<string, string> = {
  js: 'javascript', mjs: 'javascript', cjs: 'javascript', jsx: 'javascript',
  ts: 'typescript', mts: 'typescript', cts: 'typescript', tsx: 'typescript',
  vue: 'xml', svelte: 'xml', htm: 'html', xhtml: 'xml',
  scss: 'scss', sass: 'scss', less: 'less', styl: 'less',
  yml: 'yaml', json5: 'json', jsonc: 'json',
  py: 'python', rb: 'ruby', pl: 'perl', pm: 'perl', kt: 'kotlin', kts: 'kotlin',
  cc: 'cpp', cxx: 'cpp', hpp: 'cpp', hxx: 'cpp', h: 'c', cs: 'csharp',
  rs: 'rust', ex: 'elixir', exs: 'elixir', jl: 'julia',
  sh: 'bash', bash: 'bash', zsh: 'bash', fish: 'bash', cmd: 'dos', bat: 'dos', ps1: 'powershell',
  gql: 'graphql', proto: 'protobuf', tf: 'hcl', tfvars: 'hcl', gradle: 'groovy',
  mk: 'makefile', dockerfile: 'dockerfile', patch: 'diff', diff: 'diff',
  md: 'markdown', markdown: 'markdown', conf: 'ini', config: 'ini', cfg: 'ini',
  properties: 'ini', env: 'ini'
}

export function getLanguage(fileName: string): string {
  const ext = getExtension(fileName)
  if (!ext) {
    const base = fileName.toLowerCase()
    if (base === 'dockerfile') return 'dockerfile'
    if (base === 'makefile') return 'makefile'
    return 'plaintext'
  }
  if (LANG_MAP[ext]) return LANG_MAP[ext]
  return hljsKnown(ext) ? ext : 'plaintext'
}

// CODE_EXTS 里未做映射的扩展名，其字面量本身就是 hljs 的语言名
function hljsKnown(ext: string): boolean {
  return CODE_EXTS.includes(ext)
}

/** 该文本是否走代码高亮渲染 */
export function isCodeFile(fileName: string): boolean {
  return CODE_EXTS.includes(getExtension(fileName))
}

/**
 * 解析 CSV/TSV。支持双引号包裹、引号内换行与 "" 转义。
 * 仅用于预览，超大文件由调用方截断行数。
 */
export function parseDelimitedText(content: string, delimiter: string): string[][] {
  const rows: string[][] = []
  let row: string[] = []
  let field = ''
  let inQuotes = false

  for (let i = 0; i < content.length; i++) {
    const ch = content[i]

    if (inQuotes) {
      if (ch === '"') {
        if (content[i + 1] === '"') {
          field += '"'
          i++
        } else {
          inQuotes = false
        }
      } else {
        field += ch
      }
      continue
    }

    if (ch === '"') {
      inQuotes = true
    } else if (ch === delimiter) {
      row.push(field)
      field = ''
    } else if (ch === '\n' || ch === '\r') {
      if (ch === '\r' && content[i + 1] === '\n') i++
      row.push(field)
      rows.push(row)
      row = []
      field = ''
    } else {
      field += ch
    }
  }

  if (field !== '' || row.length > 0) {
    row.push(field)
    rows.push(row)
  }

  return rows
}
