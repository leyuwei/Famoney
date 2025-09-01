# Famoney - 简洁的家庭财务管理平台

<img width="1911" height="1392" alt="image" src="https://github.com/user-attachments/assets/48222ca5-4868-4240-9120-4c842537e9e2" />

使用Go后端的家庭财务管理平台

Family Finance Management Platform built with Go.

## 平台用途与特点

- 用户账户系统（中英双语切换，默认中文）
- 支持创建个人或共享钱包，记录家庭每个人/集体收支流水，可选择基准货币
- 每一笔收支均可挂靠特定类别，同一钱包里的钱可划归多类归属，收支更有序
- 钱包余额可按类别分类，并支持手动调整余额时自动生成记录
- 支持多币种及汇率换算，允许负余额以便家庭活动等场景

## 数据库准备 (MySQL)

1. 安装并启动 MySQL，创建数据库与表：直接导入`init_db.sql`

2. 运行环境变量（3条）写入 `/etc/default/famoney` 并设置权限：

   ```bash
   sudo tee /etc/default/famoney >/dev/null <<'EOF'
   DB_USER=your_db_user_here
   DB_PASSWORD=your_db_password_here
   EXRATE_API=your_api_value_here
   EOF
   sudo chmod 600 /etc/default/famoney
   ```

   `DB_USER` 与 `DB_PASSWORD` 为MySQL数据库的用户名及密码。

   `EXRATE_API` 为汇率公开数据平台的API，请访问 https://www.exchangerate-api.com/ 申请获取，每年有1500次免费访问次数，本平台的汇率需依赖此API获取。

   systemd 服务应包含 `EnvironmentFile=/etc/default/famoney`。

## 服务器运行

```bash
go mod tidy
go build
```

访问 <http://localhost:8295/famoney/> 查看页面。
此处Go后端本地监听端口，可在`main.go`中修改，可搜索并全局替换为您的偏好端口。

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

