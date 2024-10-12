# DH-Blog

## 技术栈

### 后端部分

- JDK21 + SpringBoot3
- MySQL + MybatisPlus
- Redis

### 前端部分

- TS + Vue3 + ElementPlus
- Axios
- [md-editor-v3](https://github.com/imzbf/md-editor-v3) `作为 markdown 编辑器`

## 博客功能
- [ ] 系统信息展示
- [ ] 全局系统配置
- [ ] 文章增删改查
  - [x] emoji 文字添加
  - [ ] 发表语音文章
- [x] 分类功能
- [x] 标签功能
- [x] 分页功能
- [ ] 文章访问密码
- [ ] 搜索功能
- [ ] 文章类型功能 `博客 / 随笔，在不同页面展现`

## 部署
> DH-Blog使用前后端分离技术，因此前端可以部署到任意地方，直接在 `src/types/Constant.ts` 中更改服务器地址和端口即可

> 后端推荐使用 `docker` 进行部署

### 前端部分

1. 下载源代码
2. 进入根目录
3. 更改配置中的后端接口地址，执行 `npm build`
4. 将打包文件部署到 `nginx` 中

### 后端部分
1. 进入 `admin` 目录下
2. docker 容器开启端口 `8080:8080`