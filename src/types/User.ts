export interface User {
  username: string;
  password: string;
}

export interface UserLogin extends User {
  valid: boolean;       // 账户验证是否通过
  remember: boolean;    // 是否记住密码
}