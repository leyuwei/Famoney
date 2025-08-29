CREATE TABLE users (
  id INT AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(255) UNIQUE,
  password VARCHAR(255)
);

CREATE TABLE wallets (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255),
  color VARCHAR(7) DEFAULT '#b5651d'
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
