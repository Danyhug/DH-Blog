# 使用 openjdk 作为父镜像，来运行 Spring Boot
FROM eclipse-temurin:21-jdk-alpine

# 添加一个工作目录
WORKDIR /app

# 将构建好的文件放到容器中
COPY target/blog-0.0.1-SNAPSHOT.jar /app/dhBlog.jar

# 设置环境变量，指定使用prov配置文件
ENV SPRING_PROFILES_ACTIVE=prod

# 暴露 Spring Boot 和 Nginx 端口
EXPOSE 8080

# 启动脚本，同时启动 Nginx 和 Spring Boot
CMD java -jar /app/dhBlog.jar
