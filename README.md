# Famoney

Family Finance Management Website built with Go and HTML templates.

## Features

- 用户账户系统（中英双语切换，默认中文）
- 支持创建个人或共享钱包并记录收支流水，可选择基准货币
- 钱包余额可按类别分类，并支持手动调整余额时自动生成记录
- 支持多币种及汇率换算，允许负余额以便家庭活动等场景
- HTML 前端使用可折叠表单以保持界面整洁

## Database setup (MySQL)

1. 安装并启动 MySQL，创建数据库与表：

   ```sql
   CREATE DATABASE famoney CHARACTER SET utf8mb4;
   USE famoney;
   CREATE TABLE users (
     id INT AUTO_INCREMENT PRIMARY KEY,
     username VARCHAR(255) UNIQUE,
     password VARCHAR(255)
   );
   CREATE TABLE wallets (
     id INT AUTO_INCREMENT PRIMARY KEY,
     name VARCHAR(255)
   );
   CREATE TABLE wallet_balances (
     wallet_id INT,
     currency VARCHAR(3),
     balance DOUBLE,
     PRIMARY KEY (wallet_id, currency),
     FOREIGN KEY (wallet_id) REFERENCES wallets(id)
   );
   CREATE TABLE wallet_owners (
     wallet_id INT,
     user_id INT,
     PRIMARY KEY (wallet_id, user_id),
     FOREIGN KEY (wallet_id) REFERENCES wallets(id),
     FOREIGN KEY (user_id) REFERENCES users(id)
   );
   CREATE TABLE categories (
     id INT AUTO_INCREMENT PRIMARY KEY,
     name VARCHAR(255) UNIQUE
   );
   CREATE TABLE flows (
     id INT AUTO_INCREMENT PRIMARY KEY,
     wallet_id INT,
     amount DOUBLE,
     currency VARCHAR(3),
     category_id INT,
     description TEXT,
     created_at DATETIME,
     FOREIGN KEY (wallet_id) REFERENCES wallets(id),
     FOREIGN KEY (category_id) REFERENCES categories(id)
   );
   ```

2. 运行环境变量写入 `/etc/default/famoney` 并设置权限：

   ```bash
   sudo tee /etc/default/famoney >/dev/null <<'EOF'
DB_USER=your_db_user_here
DB_PASSWORD=your_db_password_here
EXRATE_API=your_api_value_here
EOF
   sudo chmod 600 /etc/default/famoney
   ```

   systemd 服务应包含 `EnvironmentFile=/etc/default/famoney`。

## Running locally

```bash
go build
./famoney
```

访问 <http://localhost:8295/famoney/> 查看页面。

## 部署指南 (Ubuntu + Nginx)

1. **获取代码并编译**

   ```bash
   git clone https://your.repo/Famoney.git
   cd Famoney
   go build
   ```

2. **创建 systemd 服务** `/etc/systemd/system/famoney.service`

   ```ini
   [Unit]
   Description=Famoney service
   After=network.target

   [Service]
   ExecStart=/usr/local/bin/famoney
   WorkingDirectory=/var/www/Famoney
   EnvironmentFile=/etc/default/famoney
   Restart=always

   [Install]
   WantedBy=multi-user.target
   ```

   将编译后的 `famoney` 二进制复制到 `/usr/local/bin/`，代码放到 `/var/www/Famoney`。

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now famoney
   ```

3. **配置 Nginx** `/etc/nginx/sites-available/famoney.conf`

   ```nginx
   server {
       listen 80;
       server_name example.com;

       location /famoney/ {
           proxy_pass http://127.0.0.1:8295;
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

