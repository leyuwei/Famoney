# Famoney

Family Finance Management Website built with Go and HTML templates.

## Features

- 用户账户系统（中英双语切换）
- 支持创建个人或共享钱包并记录收支流水
- 钱包余额可按类别分类，并支持手动调整余额时自动生成记录
- 支持多币种及简单汇率换算
- HTML 前端使用简洁卡片式布局

## Database setup (MySQL)

1. 安装并启动 MySQL，创建数据库与表：

   ```sql
   CREATE DATABASE famoney CHARACTER SET utf8mb4;
   USE famoney;
   CREATE TABLE users (
       id INT AUTO_INCREMENT PRIMARY KEY,
       username VARCHAR(50) UNIQUE,
       password VARCHAR(100)
   );
   CREATE TABLE wallets (
       id INT AUTO_INCREMENT PRIMARY KEY,
       name VARCHAR(100),
       currency VARCHAR(10),
       balance DOUBLE
   );
   CREATE TABLE wallet_owners (
       wallet_id INT,
       user_id INT,
       PRIMARY KEY(wallet_id, user_id)
   );
   CREATE TABLE categories (
       id INT AUTO_INCREMENT PRIMARY KEY,
       name VARCHAR(50) UNIQUE
   );
   CREATE TABLE flows (
       id INT AUTO_INCREMENT PRIMARY KEY,
       wallet_id INT,
       amount DOUBLE,
       currency VARCHAR(10),
       category_id INT,
       description TEXT,
       created_at DATETIME
   );
   ```

2. 设置连接字符串环境变量（示例）：

   ```bash
   export DB_DSN="user:password@tcp(127.0.0.1:3306)/famoney?parseTime=true"
   ```

## Running locally

```bash
go build
./famoney
```

访问 <http://localhost:8080/famoney/> 查看页面。

## 部署指南 (Ubuntu + Nginx)

1. **获取代码并编译**

   ```bash
   git clone https://your.repo/Famoney.git
   cd Famoney
   go build
   ```

2. **准备 MySQL 数据库**，参见上文 `Database setup`，并在运行环境中设置 `DB_DSN`。

3. **创建 systemd 服务** `/etc/systemd/system/famoney.service`

   ```ini
   [Unit]
   Description=Famoney service
   After=network.target

   [Service]
   ExecStart=/usr/local/bin/famoney
   WorkingDirectory=/var/www/Famoney
   Restart=always

   [Install]
   WantedBy=multi-user.target
   ```

   将编译后的 `famoney` 二进制复制到 `/usr/local/bin/`，代码放到 `/var/www/Famoney`。

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now famoney
   ```

4. **配置 Nginx** `/etc/nginx/sites-available/famoney.conf`

   ```nginx
   server {
       listen 80;
       server_name example.com;

       location /famoney/ {
           proxy_pass http://127.0.0.1:8080;
           proxy_set_header Host $host;
       }
   }
   ```

   启用配置并重载：

   ```bash
   sudo ln -s /etc/nginx/sites-available/famoney.conf /etc/nginx/sites-enabled/
   sudo nginx -t
   sudo systemctl reload nginx
   ```

完成以上步骤后，通过 `http://example.com/famoney/` 访问网站。

