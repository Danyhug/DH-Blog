export interface IpStat {
  ipAddress: string; // IP 地址
  city: string; // 城市
  accessCount: number; // 访问次数
  bannedCount: number; // 封禁次数
  banStatus: number;
}