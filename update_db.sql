CREATE TABLE wallet_balances (
  wallet_id INT,
  currency VARCHAR(3),
  balance DOUBLE,
  PRIMARY KEY (wallet_id, currency),
  FOREIGN KEY (wallet_id) REFERENCES wallets(id)
);

INSERT INTO wallet_balances (wallet_id, currency, balance)
SELECT id, currency, balance FROM wallets;

ALTER TABLE wallets DROP COLUMN currency, DROP COLUMN balance;
