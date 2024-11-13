import { ToolbarNames } from "md-editor-v3/lib/types/MdEditor/type";

/**
 * 定义项目的所有常量
 */
export const SERVER_URL = import.meta.env.VITE_APP_SERVER_URL;
export const OSS_URL = import.meta.env.VITE_APP_OSS_URL;

/**
 * md编辑器
 */
export const toolbars: ToolbarNames[] = [
  'bold',
  'underline',
  'italic',
  'strikeThrough',
  '-',
  'sub',
  'sup',
  'quote',
  'unorderedList',
  'orderedList',
  'task',
  '-',
  'codeRow',
  'code',
  'link',
  'image',
  'table',
  // 'mermaid',
  // 'katex',
  0,
  1,
  2,
  3,
  '-',
  'revoke',
  'next',
  // 'save',
  '=',
  'prettier',
  'pageFullscreen',
  'fullscreen',
  'preview',
  'previewOnly',
  'htmlPreview',
  'catalog',
  'github'
];

export const emojis = [
  "😀", "🤡", "😄", "😁", "😆",
  "😅", "😂", "🤣", "😇", "😉",
  "😊", "😋", "😎", "😍", "🥰",
  "😘", "😗", "😙", "😚", "😜",
  "😝", "😛", "🤑", "🤗", "🤔",
  "🤐", "🥵", "🥶", "😭", "🤕",
];