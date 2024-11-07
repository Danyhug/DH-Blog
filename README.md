# DH-Blog

## 技术栈

### 后端部分

- JDK21 + SpringBoot3
- MySQL + MybatisPlus
- Redis
- Swagger

### 前端部分

- TS + Vue3 + ElementPlus + Echarts
- Axios
- [vue-echarts](https://github.com/ecomfe/vue-echarts)
- [md-editor-v3](https://github.com/imzbf/md-editor-v3) `作为 markdown 编辑器`

## 博客功能

### 前端部分实现

- [x] 首页设计实现
- [x] 文章页设计实现
- [x] 加密页面设计实现
- [x] 登录页面设计实现

### 后端部分实现

- [x] 通用功能
  - [x] 分页功能
  - [x] 文章访问密码
  - [x] 登录注册功能
  - [ ] 搜索功能
  - [ ] 文章类型功能 `博客 / 随笔，在不同页面展现`
- [ ] 数据总览页
  - [x] 在线人数统计
  - [x] 文章、标签、分类数据统计
  - [ ] 评论数据统计
  - [ ] 访问记录统计
  - [ ] 访问量统计
- [x] 博客发布功页
  - [x] emoji 文字添加
  - [x] 图片上传
  - [x] 增删改查
  - [x] 加密文章
  - [ ] 语音上传
- [x] 博客管理页
  - [x] 文章管理
  - [x] 分类管理
    - [x] 分类添加默认标签
  - [x] 标签功能
- [ ] 系统配置页
- [ ] 评论管理页

## 部署

> `DH-Blog` 使用前后端分离技术，因此前端可以部署到任意地方，直接在 `src/types/Constant.ts` 中更改服务器地址和端口即可

> 后端推荐使用 `docker` 进行部署

### 前端部分

1. 下载源代码
2. 进入根目录
3. 更改配置中的后端接口地址，执行 `npm build`
4. 将打包文件部署到 `nginx` 中

### 后端部分

1. 进入 `admin` 目录下
2. docker 容器开启端口 `8080:8080`
3. 博客使用 `Swagger` 书写 API 文档，可以访问 `http://localhost:8080/doc.html` 直接打开文档
4. 后台默认账号密码为 `admin` `admin`，博客为个人设计不提供注册接口，可以去 `admin/src/test/java/top/zzf4/blog/JwtTest.java` 调用 `addUser` 方法手动添加用户
