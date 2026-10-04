import { defineComponent, h } from 'vue'

// 影院专用的媒体图标。播放控制类用实心（fill），界面类用描边（stroke），
// 与 Netflix / Apple Music 的图标语言一致。
function solid(...paths: string[]) {
  return defineComponent({
    render: () => h('svg', { viewBox: '0 0 24 24', fill: 'currentColor' }, paths.map(d => h('path', { d })))
  })
}

function line(...paths: string[]) {
  return defineComponent({
    render: () =>
      h(
        'svg',
        {
          viewBox: '0 0 24 24',
          fill: 'none',
          stroke: 'currentColor',
          'stroke-width': '2',
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round'
        },
        paths.map(d => h('path', { d }))
      )
  })
}

export const PlayIcon = solid('M7 4.5v15a1 1 0 0 0 1.52.85l12-7.5a1 1 0 0 0 0-1.7l-12-7.5A1 1 0 0 0 7 4.5z')
export const PauseIcon = solid('M6 4h4v16H6zM14 4h4v16h-4z')
export const NextIcon = solid('M4 5.5v13a1 1 0 0 0 1.5.86L15 13.6V18a1 1 0 0 0 2 0V6a1 1 0 0 0-2 0v4.4L5.5 4.64A1 1 0 0 0 4 5.5zM18 5h2v14h-2z')
export const PrevIcon = solid('M20 5.5v13a1 1 0 0 1-1.5.86L9 13.6V18a1 1 0 0 1-2 0V6a1 1 0 0 1 2 0v4.4l9.5-5.76A1 1 0 0 1 20 5.5zM4 5h2v14H4z')
export const ShuffleIcon = line('M16 3h5v5', 'M4 20 21 3', 'M21 16v5h-5', 'M15 15l6 6', 'M4 4l5 5')
export const RepeatIcon = line('M17 2l4 4-4 4', 'M3 11v-1a4 4 0 0 1 4-4h14', 'M7 22l-4-4 4-4', 'M21 13v1a4 4 0 0 1-4 4H3')
export const RepeatOneIcon = line('M17 2l4 4-4 4', 'M3 11v-1a4 4 0 0 1 4-4h14', 'M7 22l-4-4 4-4', 'M21 13v1a4 4 0 0 1-4 4H3', 'M11 10h1v4')
export const VolumeIcon = line('M11 5 6 9H2v6h4l5 4z', 'M15.54 8.46a5 5 0 0 1 0 7.07', 'M19.07 4.93a10 10 0 0 1 0 14.14')
export const MuteIcon = line('M11 5 6 9H2v6h4l5 4z', 'M23 9l-6 6', 'M17 9l6 6')
export const FullscreenIcon = line('M8 3H5a2 2 0 0 0-2 2v3', 'M21 8V5a2 2 0 0 0-2-2h-3', 'M3 16v3a2 2 0 0 0 2 2h3', 'M16 21h3a2 2 0 0 0 2-2v-3')
export const ExitFullscreenIcon = line('M8 3v3a2 2 0 0 1-2 2H3', 'M21 8h-3a2 2 0 0 1-2-2V3', 'M3 16h3a2 2 0 0 1 2 2v3', 'M16 21v-3a2 2 0 0 1 2-2h3')
export const PipIcon = line('M21 10V6a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h5', 'M14 14h7v6h-7z')
export const SubtitleIcon = line('M3 5h18v14H3z', 'M7 13h3', 'M13 13h4', 'M7 16h6', 'M15 16h2')
export const SpeedIcon = line('M12 14l4-4', 'M3.34 19a10 10 0 1 1 17.32 0')
export const BackIcon = line('M19 12H5', 'M12 19l-7-7 7-7')
export const Rewind10Icon = line('M3 12a9 9 0 1 0 3-6.7', 'M3 4v5h5', 'M10 15V9l-1.5 1', 'M13.5 9.5h2v5h-2z')
export const Forward10Icon = line('M21 12a9 9 0 1 1-3-6.7', 'M21 4v5h-5', 'M8.5 15V9L7 10', 'M12 9.5h2v5h-2z')
export const InfoIcon = line('M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z', 'M12 16v-4', 'M12 8h.01')
export const PlusIcon = line('M12 5v14', 'M5 12h14')
export const CloseIcon = line('M18 6 6 18', 'M6 6l12 12')
export const SearchIcon = line('M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16z', 'M21 21l-4.35-4.35')
export const SettingsIcon = line(
  'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z',
  'M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z'
)
export const QueueIcon = line('M3 6h13', 'M3 12h13', 'M3 18h9', 'M19 15v6', 'M16 18h6')
export const LyricsIcon = line('M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z', 'M8 9h8', 'M8 13h5')
export const MoreIcon = solid('M5 10.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zM12 10.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zM19 10.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z')
export const EpisodesIcon = line('M4 6h16v10H4z', 'M7 20h10', 'M8 3h8')
export const MusicNoteIcon = line('M9 18V5l12-2v13', 'M6 21a3 3 0 1 0 0-6 3 3 0 0 0 0 6z', 'M18 19a3 3 0 1 0 0-6 3 3 0 0 0 0 6z')
export const FilmIcon = line('M3 3h18v18H3z', 'M7 3v18', 'M17 3v18', 'M3 12h18', 'M3 7.5h4', 'M3 16.5h4', 'M17 7.5h4', 'M17 16.5h4')
export const FolderIcon = line('M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z')
export const AlbumIcon = line('M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z', 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z')
export const MicIcon = line('M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3z', 'M19 10v2a7 7 0 0 1-14 0v-2', 'M12 19v3')
export const ClockIcon = line('M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z', 'M12 6v6l4 2')
export const SparkIcon = line('M12 3l1.9 5.8L20 10.7l-6.1 1.9L12 18.5l-1.9-5.9L4 10.7l6.1-1.9z', 'M19 17v4', 'M17 19h4')
export const HomeIcon = line('m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z', 'M9 22V12h6v10')
export const ChevronLeftIcon = line('M15 18l-6-6 6-6')
export const ChevronRightIcon = line('M9 18l6-6-6-6')
export const ChevronDownIcon = line('M6 9l6 6 6-6')
export const CheckIcon = line('M20 6 9 17l-5-5')
// 电视：TV 模式开关
export const TvIcon = line('M3 6.5A1.5 1.5 0 0 1 4.5 5h15A1.5 1.5 0 0 1 21 6.5v10a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 16.5z', 'M8 21h8', 'M12 18v3')
